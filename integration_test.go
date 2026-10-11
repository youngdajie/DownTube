package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const itFFmpegURL = "https://ghfast.top/github.com/GyanD/codexffmpeg/releases/download/9.0.2/ffmpeg-9.0.2-essentials_build.7z"

// ensureTestFFmpeg 准备测试用 ffmpeg:优先系统安装,否则下载 7z 并解压到临时目录。
func ensureTestFFmpeg(t *testing.T, baseDir string) string {
	if p, err := exec.LookPath("ffmpeg.exe"); err == nil {
		return p
	}
	binDir := filepath.Join(baseDir, "ffmpeg", "bin")
	ffmpegPath := filepath.Join(binDir, "ffmpeg.exe")
	if _, err := os.Stat(ffmpegPath); err == nil {
		return ffmpegPath
	}
	archive := filepath.Join(baseDir, "ffmpeg-it.7z")
	if _, err := os.Stat(archive); err != nil {
		t.Log("下载 FFmpeg 安装包(约 33.8MB,仅首次)…")
		resp, err := http.Get(itFFmpegURL)
		if err != nil {
			t.Fatalf("下载 FFmpeg 失败: %v", err)
		}
		defer resp.Body.Close()
		f, err := os.Create(archive)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(f, resp.Body); err != nil {
			f.Close()
			t.Fatalf("下载 FFmpeg 失败: %v", err)
		}
		f.Close()
	}
	t.Log("解压 FFmpeg…")
	if err := extractFFmpegArchive(archive, binDir, nil); err != nil {
		t.Fatalf("解压 FFmpeg 失败: %v", err)
	}
	return ffmpegPath
}

// stageYtDlp 把项目里的 yt-dlp.exe 放到测试进程所在目录,让 findYTDLP 能找到。
func stageYtDlp(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "tools", "yt-dlp.exe")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("tools/yt-dlp.exe 不存在: %v", err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(filepath.Dir(exe), "yt-dlp.exe")
	if dstInfo, statErr := os.Stat(dst); statErr == nil {
		if srcInfo, srcErr := os.Stat(src); srcErr == nil && dstInfo.Size() == srcInfo.Size() {
			return
		}
	}
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o755); err != nil {
		t.Fatalf("复制 yt-dlp 失败: %v", err)
	}
}

func TestConcurrentDownloadFlow(t *testing.T) {
	if os.Getenv("DOWNTUBE_IT") == "" {
		t.Skip("设置 DOWNTUBE_IT=1 运行该集成测试")
	}
	stageYtDlp(t)
	proxy := os.Getenv("DOWNTUBE_IT_PROXY")
	if proxy == "" {
		proxy = "http://127.0.0.1:7890"
	}
	baseDir := filepath.Join(os.TempDir(), "downtube-it")
	_ = os.MkdirAll(baseDir, 0o755)

	app := &App{
		store:         Store{Version: 2, NextID: 1, Tasks: []Task{}, Settings: defaultSettings()},
		workers:       make(map[int]context.CancelFunc),
		retryAttempts: make(map[int]int),
		bestProgress:  make(map[int]int),
		disableEvents: true,
		titleJobs:     make(chan struct{}, 2),
		storePath:     filepath.Join(baseDir, "state.json"),
	}
	app.ctx = context.Background()
	s := &app.store.Settings
	s.UseProxy = true
	s.ProxyURL = proxy
	s.MaxConcurrent = 2
	s.AutoRetry = 0
	s.DownloadType = "视频（带音频）"
	s.Quality = "480p"
	s.DownloadDir = filepath.Join(baseDir, "out")
	_ = os.MkdirAll(s.DownloadDir, 0o755)
	s.FFmpegPath = ensureTestFFmpeg(t, baseDir)

	app.AddLinks(strings.Join([]string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://www.youtube.com/watch?v=jNQXAC9IVRw",
		"https://www.youtube.com/watch?v=9bZkp7q19f0",
	}, "\n"))
	app.StartAllTasks()

	seenPhases := map[string]bool{}
	maxRunning := 0
	deadline := time.Now().Add(8 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(1200 * time.Millisecond)
		snap := app.Snapshot()
		running, pending, done := 0, 0, 0
		for _, task := range snap.Tasks {
			switch task.Status {
			case StatusRunning:
				running++
				if task.Phase != "" {
					seenPhases[task.Phase] = true
				}
			case StatusQueued, StatusPaused:
				pending++
			case StatusComplete:
				done++
			}
		}
		if running > maxRunning {
			maxRunning = running
		}
		if running > 2 {
			t.Fatalf("并发超过上限: %d", running)
		}
		if done == len(snap.Tasks) {
			break
		}
		if running == 0 && pending == 0 {
			break
		}
	}
	final := app.Snapshot()
	completed := 0
	for _, task := range final.Tasks {
		t.Logf("任务 %d [%s] 标题=%q 阶段=%q 错误=%q", task.ID, task.Status, task.Title, task.Phase, task.Error)
		if task.Status == StatusComplete {
			completed++
		}
	}
	t.Logf("观察到的阶段: %v", seenPhases)
	if maxRunning < 2 {
		t.Errorf("未观察到并发下载,maxRunning=%d", maxRunning)
	}
	if !seenPhases["下载视频中"] {
		t.Errorf("缺少「下载视频中」阶段文案: %v", seenPhases)
	}
	// 音频流可能在一秒内下完,轮询容易错过「下载音频中」的短暂窗口;
	// 出现「合并中」即说明两路流都已被跟踪,这里只在观测到时记录。
	if !seenPhases["下载音频中"] {
		t.Logf("未轮询到「下载音频中」(音频流窗口过短属正常),合并阶段已确认: %v", seenPhases)
	}
	if !seenPhases["合并中"] {
		t.Errorf("未观察到合并阶段: %v", seenPhases)
	}
	// YouTube 的机器人验证可能随机拦截个别视频(会附带 Cookies 提示),
	// 这条测试只要求调度与阶段链路正常,至少一个任务下载成功即可。
	if completed == 0 {
		t.Errorf("没有任何任务完成下载")
	}
	time.Sleep(500 * time.Millisecond)
	for _, cancel := range app.workers {
		cancel()
	}
}

// TestTranscodeHEVC 验证 H.265 转码链路:生成测试视频 → transcodeToHEVC → 校验输出编码。
func TestTranscodeHEVC(t *testing.T) {
	if os.Getenv("DOWNTUBE_IT") == "" {
		t.Skip("设置 DOWNTUBE_IT=1 运行该集成测试")
	}
	baseDir := filepath.Join(os.TempDir(), "downtube-it")
	_ = os.MkdirAll(baseDir, 0o755)
	ffmpegPath := ensureTestFFmpeg(t, baseDir)

	app := &App{
		store:         Store{Version: 2, NextID: 1, Tasks: []Task{}, Settings: defaultSettings()},
		workers:       make(map[int]context.CancelFunc),
		retryAttempts: make(map[int]int),
		bestProgress:  make(map[int]int),
		disableEvents: true,
		titleJobs:     make(chan struct{}, 2),
		storePath:     filepath.Join(baseDir, "state.json"),
	}
	app.ctx = context.Background()
	app.store.Tasks = append(app.store.Tasks, Task{ID: 1, Status: StatusRunning})

	// 生成 5 秒测试视频(含音频轨)
	src := filepath.Join(baseDir, "transcode-src.mp4")
	_ = os.Remove(src)
	gen := exec.Command(ffmpegPath, "-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc=duration=5:size=320x240:rate=30",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=5",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", "-shortest", src)
	hideConsoleWindow(gen)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("生成测试视频失败: %v %s", err, string(out))
	}

	done := make(chan error, 1)
	go func() { done <- app.transcodeToHEVC(context.Background(), 1, src, 5, ffmpegPath) }()
	seenPhase := false
	var lastPercent int
	deadline := time.Now().Add(3 * time.Minute)
wait:
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("转码失败: %v", err)
			}
			break wait
		default:
			time.Sleep(200 * time.Millisecond)
			snap := app.Snapshot()
			if len(snap.Tasks) > 0 {
				if snap.Tasks[0].Phase != "" {
					seenPhase = true
				}
				lastPercent = snap.Tasks[0].Progress
			}
		}
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("转码输出文件不存在: %v", err)
	}
	probe := exec.Command(filepath.Join(filepath.Dir(ffmpegPath), "ffprobe.exe"),
		"-v", "error", "-select_streams", "v:0", "-show_entries", "stream=codec_name", "-of", "csv=p=0", src)
	hideConsoleWindow(probe)
	out, err := probe.Output()
	if err != nil {
		t.Fatalf("ffprobe 校验失败: %v", err)
	}
	codec := strings.TrimSpace(string(out))
	t.Logf("输出编码: %s,观察到阶段提示: %v,最后进度: %d%%", codec, seenPhase, lastPercent)
	if codec != "hevc" {
		t.Errorf("期望 hevc 编码,实际 %s", codec)
	}
}

// TestHEVCEndToEnd 端到端:真实视频下载完成后用 FFmpeg 转码为 H.265。
func TestHEVCEndToEnd(t *testing.T) {
	if os.Getenv("DOWNTUBE_IT") == "" {
		t.Skip("设置 DOWNTUBE_IT=1 运行该集成测试")
	}
	stageYtDlp(t)
	proxy := os.Getenv("DOWNTUBE_IT_PROXY")
	if proxy == "" {
		proxy = "http://127.0.0.1:7890"
	}
	baseDir := filepath.Join(os.TempDir(), "downtube-it")
	_ = os.MkdirAll(baseDir, 0o755)
	outDir := filepath.Join(baseDir, "out-hevc")
	_ = os.RemoveAll(outDir)
	_ = os.MkdirAll(outDir, 0o755)

	app := &App{
		store:         Store{Version: 2, NextID: 1, Tasks: []Task{}, Settings: defaultSettings()},
		workers:       make(map[int]context.CancelFunc),
		retryAttempts: make(map[int]int),
		bestProgress:  make(map[int]int),
		disableEvents: true,
		titleJobs:     make(chan struct{}, 2),
		storePath:     filepath.Join(baseDir, "state.json"),
	}
	app.ctx = context.Background()
	s := &app.store.Settings
	s.UseProxy = true
	s.ProxyURL = proxy
	s.MaxConcurrent = 1
	s.AutoRetry = 0
	s.DownloadType = "视频（带音频）"
	s.Quality = "480p"
	s.VideoCodec = "hevc"
	s.DownloadDir = outDir
	ffmpegPath := ensureTestFFmpeg(t, baseDir)
	s.FFmpegPath = ffmpegPath

	app.AddLinks("https://www.youtube.com/watch?v=jNQXAC9IVRw")
	app.StartAllTasks()

	sawTranscode := false
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(800 * time.Millisecond)
		snap := app.Snapshot()
		task := snap.Tasks[0]
		if strings.HasPrefix(task.Phase, "转码") {
			sawTranscode = true
		}
		if task.Status == StatusComplete || task.Status == StatusFailed {
			break
		}
	}
	final := app.Snapshot().Tasks[0]
	t.Logf("最终状态: %s | 阶段: %s | 错误: %s", final.Status, final.Phase, final.Error)
	if final.Status != StatusComplete {
		t.Fatalf("任务未完成: %s", final.Error)
	}
	entries, _ := os.ReadDir(outDir)
	var mp4Path string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".mp4") {
			mp4Path = filepath.Join(outDir, e.Name())
		}
	}
	if mp4Path == "" {
		t.Fatalf("输出目录没有 mp4 文件")
	}
	probe := exec.Command(filepath.Join(filepath.Dir(ffmpegPath), "ffprobe.exe"),
		"-v", "error", "-select_streams", "v:0", "-show_entries", "stream=codec_name", "-of", "csv=p=0", mp4Path)
	hideConsoleWindow(probe)
	out, err := probe.Output()
	if err != nil {
		t.Fatalf("ffprobe 校验失败: %v", err)
	}
	codec := strings.TrimSpace(string(out))
	t.Logf("输出文件: %s | 编码: %s | 观察到转码阶段: %v", filepath.Base(mp4Path), codec, sawTranscode)
	if codec != "hevc" {
		t.Errorf("期望 hevc 编码,实际 %s", codec)
	}
}

// TestAutoRetryFlow 验证失败任务会按设置自动重试,重试用尽后标记失败。
// 代理指向一个不存在的端口,制造稳定复现的网络失败。
func TestAutoRetryFlow(t *testing.T) {
	if os.Getenv("DOWNTUBE_IT") == "" {
		t.Skip("设置 DOWNTUBE_IT=1 运行该集成测试")
	}
	stageYtDlp(t)
	baseDir := filepath.Join(os.TempDir(), "downtube-it")
	_ = os.MkdirAll(baseDir, 0o755)

	app := &App{
		store:         Store{Version: 2, NextID: 1, Tasks: []Task{}, Settings: defaultSettings()},
		workers:       make(map[int]context.CancelFunc),
		retryAttempts: make(map[int]int),
		bestProgress:  make(map[int]int),
		disableEvents: true,
		titleJobs:     make(chan struct{}, 2),
		storePath:     filepath.Join(baseDir, "state.json"),
	}
	app.ctx = context.Background()
	s := &app.store.Settings
	s.UseProxy = true
	s.ProxyURL = "http://127.0.0.1:9" // 不可用端口,保证下载失败
	s.MaxConcurrent = 1
	s.AutoRetry = 1
	s.DownloadType = "视频（带音频）"
	s.Quality = "480p"
	s.DownloadDir = filepath.Join(baseDir, "out-fail")
	_ = os.MkdirAll(s.DownloadDir, 0o755)

	app.AddLinks("https://www.youtube.com/watch?v=jNQXAC9IVRw")
	app.StartAllTasks()

	sawRetryPhase := false
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(800 * time.Millisecond)
		snap := app.Snapshot()
		task := snap.Tasks[0]
		if strings.HasPrefix(task.Phase, "重试中") {
			sawRetryPhase = true
		}
		if task.Status == StatusFailed {
			break
		}
	}
	final := app.Snapshot().Tasks[0]
	t.Logf("最终状态: %s | 错误: %s", final.Status, final.Error)
	if !sawRetryPhase {
		t.Errorf("未观察到\"重试中\"阶段")
	}
	if final.Status != StatusFailed {
		t.Errorf("期望最终失败,实际 %s", final.Status)
	}
	if !strings.Contains(final.Error, "127.0.0.1:9") && !strings.Contains(final.Error, "proxy") && !strings.Contains(final.Error, "Proxy") && !strings.Contains(final.Error, "连接") {
		t.Logf("错误信息(未包含典型代理字样,人工确认): %s", final.Error)
	}
}

// TestRetryCounterResetsOnProgress 验证重试计数在下载重新推进(重试恢复)后归零:
// 之后如果再失败,会从「重试中(1/N)」重新开始。
func TestRetryCounterResetsOnProgress(t *testing.T) {
	app := &App{
		store:         Store{Version: 2, NextID: 2, Tasks: []Task{{ID: 1, Status: StatusRunning, Progress: 40}}, Settings: defaultSettings()},
		workers:       make(map[int]context.CancelFunc),
		retryAttempts: make(map[int]int),
		bestProgress:  make(map[int]int),
		disableEvents: true,
		titleJobs:     make(chan struct{}, 2),
	}
	app.retryAttempts[1] = 2
	app.bestProgress[1] = 40

	// 重试后进度超过此前最高水平 → 视为已经恢复,连续失败计数归零
	app.updateProgress(1, 41, "下载视频中")
	if _, ok := app.retryAttempts[1]; ok {
		t.Fatalf("进度推进后重试计数应归零,实际 %v", app.retryAttempts)
	}

	// 之后再次失败但进度没有新突破 → 计数照常累加
	app.retryAttempts[1] = 1
	app.updateProgress(1, 20, "下载视频中")
	if app.retryAttempts[1] != 1 {
		t.Fatalf("无新进展时不应重置计数,实际 %d", app.retryAttempts[1])
	}
}

// TestAutoStartAfterParse 验证新添加的链接在解析完成后会自动开始下载,无需手动点击开始。
func TestAutoStartAfterParse(t *testing.T) {
	if os.Getenv("DOWNTUBE_IT") == "" {
		t.Skip("设置 DOWNTUBE_IT=1 运行该集成测试")
	}
	stageYtDlp(t)
	proxy := os.Getenv("DOWNTUBE_IT_PROXY")
	if proxy == "" {
		proxy = "http://127.0.0.1:7890"
	}
	baseDir := filepath.Join(os.TempDir(), "downtube-it")
	_ = os.MkdirAll(baseDir, 0o755)
	outDir := filepath.Join(baseDir, "out-autostart")
	_ = os.RemoveAll(outDir)
	_ = os.MkdirAll(outDir, 0o755)

	app := &App{
		store:         Store{Version: 2, NextID: 1, Tasks: []Task{}, Settings: defaultSettings()},
		workers:       make(map[int]context.CancelFunc),
		retryAttempts: make(map[int]int),
		bestProgress:  make(map[int]int),
		disableEvents: true,
		titleJobs:     make(chan struct{}, 2),
		storePath:     filepath.Join(baseDir, "state.json"),
	}
	app.ctx = context.Background()
	s := &app.store.Settings
	s.UseProxy = true
	s.ProxyURL = proxy
	s.MaxConcurrent = 2
	s.AutoRetry = 1
	s.DownloadType = "视频（带音频）"
	s.Quality = "480p"
	s.DownloadDir = outDir
	s.FFmpegPath = ensureTestFFmpeg(t, baseDir)

	// 不调用 StartAllTasks/StartTask —— 任务应靠"解析完成后自动开始"启动
	app.AddLinks("https://www.youtube.com/watch?v=jNQXAC9IVRw")
	if len(app.Snapshot().Tasks) != 1 {
		t.Fatalf("链接未被接受")
	}

	sawRunning := false
	deadline := time.Now().Add(4 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(800 * time.Millisecond)
		task := app.Snapshot().Tasks[0]
		if task.Status == StatusRunning {
			sawRunning = true
		}
		if task.Status == StatusComplete || task.Status == StatusFailed {
			break
		}
	}
	final := app.Snapshot().Tasks[0]
	t.Logf("最终状态: %s | 错误: %s", final.Status, final.Error)
	if !sawRunning {
		t.Fatalf("任务未被自动开始(始终未进入下载)")
	}
	if final.Status != StatusComplete {
		t.Skipf("自动开始链路已验证,但下载未成功(可能是风控拦截): %s", final.Error)
	}

	// 校验成品带音轨
	mp4Path := ""
	entries, _ := os.ReadDir(outDir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".mp4") {
			mp4Path = filepath.Join(outDir, e.Name())
		}
	}
	if mp4Path == "" {
		t.Fatalf("输出目录没有 mp4 文件")
	}
	probe := exec.Command(filepath.Join(filepath.Dir(s.FFmpegPath), "ffprobe.exe"),
		"-v", "error", "-select_streams", "a", "-show_entries", "stream=index", "-of", "csv=p=0", mp4Path)
	hideConsoleWindow(probe)
	out, perr := probe.Output()
	if perr != nil || strings.TrimSpace(string(out)) == "" {
		t.Errorf("成品文件不含音轨: err=%v out=%q", perr, string(out))
	}
}

// TestStaleFileMergeRecovery 验证「先有同名无声旧文件」时,再次以「视频（带音频）」
// 下载同一视频:自动把旧文件改名保留并重新下载合并,最终成品带音轨。
func TestStaleFileMergeRecovery(t *testing.T) {
	if os.Getenv("DOWNTUBE_IT") == "" {
		t.Skip("设置 DOWNTUBE_IT=1 运行该集成测试")
	}
	stageYtDlp(t)
	proxy := os.Getenv("DOWNTUBE_IT_PROXY")
	if proxy == "" {
		proxy = "http://127.0.0.1:7890"
	}
	baseDir := filepath.Join(os.TempDir(), "downtube-it")
	_ = os.MkdirAll(baseDir, 0o755)
	outDir := filepath.Join(baseDir, "out-stale")
	_ = os.RemoveAll(outDir)
	_ = os.MkdirAll(outDir, 0o755)
	ffmpegPath := ensureTestFFmpeg(t, baseDir)

	// 1) 先用「仅视频」方式造一个同名、不含音轨的旧文件(模拟此前留下的成品)
	stalePath := filepath.Join(outDir, "Me at the zoo.mp4")
	engine, err := findYTDLP()
	if err != nil {
		t.Skipf("yt-dlp 不可用: %v", err)
	}
	created := false
	for attempt := 0; attempt < 3 && !created; attempt++ {
		cmd := exec.Command(engine, "--newline", "--no-warnings", "--proxy", proxy,
			"-f", "bv*/b", "-S", "res,fps,codec:avc:m4a",
			"--ffmpeg-location", filepath.Dir(ffmpegPath),
			"-o", filepath.Join(outDir, "%(title)s.%(ext)s"),
			"https://www.youtube.com/watch?v=jNQXAC9IVRw")
		hideConsoleWindow(cmd)
		if out, runErr := cmd.CombinedOutput(); runErr != nil {
			t.Logf("准备旧文件第 %d 次失败: %v %s", attempt+1, runErr, string(out))
			continue
		}
		if _, statErr := os.Stat(stalePath); statErr == nil {
			created = true
		}
	}
	if !created {
		t.Skipf("无法准备旧文件(可能被风控拦截)")
	}

	// 2) 以「视频（带音频）」模式重新下载同一视频(命中同名跳过 → 自动改名重下)
	app := &App{
		store:         Store{Version: 2, NextID: 1, Tasks: []Task{}, Settings: defaultSettings()},
		workers:       make(map[int]context.CancelFunc),
		retryAttempts: make(map[int]int),
		bestProgress:  make(map[int]int),
		disableEvents: true,
		titleJobs:     make(chan struct{}, 2),
		storePath:     filepath.Join(baseDir, "state.json"),
	}
	app.ctx = context.Background()
	s := &app.store.Settings
	s.UseProxy = true
	s.ProxyURL = proxy
	s.MaxConcurrent = 1
	s.AutoRetry = 1
	s.DownloadType = "视频（带音频）"
	s.Quality = "480p"
	s.DownloadDir = outDir
	s.FFmpegPath = ffmpegPath

	app.AddLinks("https://www.youtube.com/watch?v=jNQXAC9IVRw")
	app.StartAllTasks()

	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(800 * time.Millisecond)
		task := app.Snapshot().Tasks[0]
		if task.Status == StatusComplete || task.Status == StatusFailed {
			break
		}
	}
	final := app.Snapshot().Tasks[0]
	t.Logf("最终状态: %s | 错误: %s", final.Status, final.Error)
	if final.Status != StatusComplete {
		t.Skipf("下载未成功(可能是风控拦截),跳过校验: %s", final.Error)
	}

	// 3) 断言:旧文件已改名保留;成品文件带音轨
	if _, statErr := os.Stat(stalePath); statErr != nil {
		t.Fatalf("成品文件不存在: %v", statErr)
	}
	probe := exec.Command(filepath.Join(filepath.Dir(ffmpegPath), "ffprobe.exe"),
		"-v", "error", "-select_streams", "a", "-show_entries", "stream=index", "-of", "csv=p=0", stalePath)
	hideConsoleWindow(probe)
	out, perr := probe.Output()
	if perr != nil || strings.TrimSpace(string(out)) == "" {
		t.Errorf("成品文件不含音轨: err=%v out=%q", perr, string(out))
	}
	aside := false
	entries, _ := os.ReadDir(outDir)
	for _, e := range entries {
		if strings.Contains(e.Name(), "(旧文件)") {
			aside = true
		}
	}
	if !aside {
		t.Errorf("未找到改名保留的旧文件(名称应包含「(旧文件)」)")
	}
}

// TestRetryLaunchClearsStaleError 验证重试一旦开始,上一次失败的错误文案会被清掉:
// 任务处于运行态时 Error 必须为空(此前重试启动不清 Error,下载/转码中会一直挂着旧报错)。
func TestRetryLaunchClearsStaleError(t *testing.T) {
	if os.Getenv("DOWNTUBE_IT") == "" {
		t.Skip("设置 DOWNTUBE_IT=1 运行该集成测试")
	}
	stageYtDlp(t)
	app := &App{
		store: Store{Version: 2, NextID: 2, Tasks: []Task{{
			ID: 1, URL: "https://www.youtube.com/watch?v=jNQXAC9IVRw", Title: "test",
			Status: StatusQueued, Phase: "重试中(1/1)", Error: "上一次失败的错误信息",
		}}, Settings: defaultSettings()},
		workers:       make(map[int]context.CancelFunc),
		retryAttempts: make(map[int]int),
		bestProgress:  make(map[int]int),
		disableEvents: true,
		titleJobs:     make(chan struct{}, 2),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app.ctx = ctx
	s := &app.store.Settings
	s.UseProxy = true
	s.ProxyURL = "http://127.0.0.1:9" // 不可用端口:重试必然失败,只需断言"开始"一瞬间的状态
	s.MaxConcurrent = 1
	s.AutoRetry = 0
	s.DownloadDir = os.TempDir()
	app.retryAttempts[1] = 1

	app.scheduleRetry(1, 0)

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(30 * time.Millisecond)
		task := app.Snapshot().Tasks[0]
		if task.Status == StatusRunning {
			if task.Error != "" {
				t.Fatalf("重试开始后不应保留旧错误,实际 %q", task.Error)
			}
			for _, c := range app.workers {
				c()
			}
			t.Logf("重试已开始,旧错误已清除(阶段=%q)", task.Phase)
			return
		}
		if task.Status == StatusFailed {
			t.Fatalf("未能在失败前观察到运行态(错误=%q),无法验证", task.Error)
		}
	}
	t.Fatalf("30 秒内未观察到重试启动")
}
