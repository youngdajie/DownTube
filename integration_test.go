package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
		t.Logf("未轮询到「下载视频中」(分片下载窗口过短属正常),合并阶段已确认: %v", seenPhases)
	}
	// 音频流同理:可能出现不到一秒,轮询错过;「合并中」出现即说明两路流都被跟踪。
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

	// 记录编码器探测与转码阶段(显卡可用时走 NVENC/QSV/AMF,不可用回退 CPU)
	spec, gpu := app.resolveHEVCVideoEncoder(ffmpegPath)
	t.Logf("H.265 编码器探测: gpu=%v id=%s label=%s", gpu, spec.id, spec.label)

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
	var lastPhase string
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
					lastPhase = snap.Tasks[0].Phase
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
	t.Logf("输出编码: %s,观察到阶段提示: %v(最近阶段 %q),最后进度: %d%%", codec, seenPhase, lastPhase, lastPercent)
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

// TestProgressPhaseLabels 验证下载阶段文案的推导(视频 / 音频 / 合并 / 转换)。
func TestProgressPhaseLabels(t *testing.T) {
	cases := []struct {
		percent, seen, expected int
		vcodec, acodec          string
		downloadType            string
		want                    string
	}{
		{50, 1, 2, "avc1", "none", "视频（带音频）", "下载视频中"},
		{100, 1, 2, "avc1", "none", "视频（带音频）", "下载视频中"},
		{35, 2, 2, "none", "mp4a", "视频（带音频）", "下载音频中"},
		{100, 2, 2, "none", "mp4a", "视频（带音频）", "合并中"},
		{100, 1, 1, "vc", "ac", "视频（带音频）", "下载中"},
		{100, 1, 1, "none", "mp3a", "仅音频 · MP3", "转换音频中"},
		{100, 1, 1, "avc1", "none", "仅视频", "下载视频中"},
	}
	for _, c := range cases {
		got := progressPhase(c.percent, c.seen, c.expected, c.vcodec, c.acodec, c.downloadType)
		if got != c.want {
			t.Errorf("progressPhase(percent=%d, seen=%d, expected=%d, v=%s, a=%s, type=%s) = %q, 期望 %q",
				c.percent, c.seen, c.expected, c.vcodec, c.acodec, c.downloadType, got, c.want)
		}
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

// TestTranscodeResume 验证转码中断后续跑:已完成的分片被复用,最终成品完整(时长与音轨正常)。
func TestTranscodeResume(t *testing.T) {
	if os.Getenv("DOWNTUBE_IT") == "" {
		t.Skip("设置 DOWNTUBE_IT=1 运行该集成测试")
	}
	baseDir := filepath.Join(os.TempDir(), "downtube-it")
	_ = os.MkdirAll(baseDir, 0o755)
	ffmpegPath := ensureTestFFmpeg(t, baseDir)

	// 生成 75 秒测试视频(约 3 个分片),含音频轨
	src := filepath.Join(baseDir, "resume-src.mp4")
	_ = os.Remove(src)
	_ = os.RemoveAll(hevcPartsDir(src))
	gen := exec.Command(ffmpegPath, "-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc2=size=1280x720:rate=30:duration=75",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=75",
		"-c:v", "libx264", "-preset", "ultrafast", "-g", "60", "-pix_fmt", "yuv420p", "-c:a", "aac", "-shortest", src)
	hideConsoleWindow(gen)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("生成测试视频失败: %v %s", err, string(out))
	}

	app := &App{
		store:         Store{Version: 2, NextID: 2, Tasks: []Task{{ID: 1, Status: StatusRunning}}, Settings: defaultSettings()},
		workers:       make(map[int]context.CancelFunc),
		retryAttempts: make(map[int]int),
		bestProgress:  make(map[int]int),
		disableEvents: true,
		titleJobs:     make(chan struct{}, 2),
	}
	app.ctx = context.Background()

	// 第一轮:编码进行中中断(模拟暂停或关闭程序)
	ctx1, cancel1 := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- app.transcodeToHEVC(ctx1, 1, src, 75, ffmpegPath) }()
	time.Sleep(3 * time.Second)
	cancel1()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatalf("中断后转码未退出")
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("中断后原文件应保留: %v", err)
	}
	partial := 0
	if entries, rerr := os.ReadDir(filepath.Join(hevcPartsDir(src), "enc")); rerr == nil {
		partial = len(entries)
	}
	t.Logf("中断时已完成分片: %d", partial)

	// 第二轮:重新执行应复用已完成分片并跑完
	if err := app.transcodeToHEVC(context.Background(), 1, src, 75, ffmpegPath); err != nil {
		t.Fatalf("续跑转码失败: %v", err)
	}
	if _, err := os.Stat(hevcPartsDir(src)); !os.IsNotExist(err) {
		t.Errorf("续跑完成后应清理转码工作目录")
	}

	probeDir := filepath.Dir(ffmpegPath)
	probeV := exec.Command(filepath.Join(probeDir, "ffprobe.exe"),
		"-v", "error", "-select_streams", "v:0", "-show_entries", "stream=codec_name", "-of", "csv=p=0", src)
	hideConsoleWindow(probeV)
	outV, err := probeV.Output()
	if err != nil {
		t.Fatalf("ffprobe 视频校验失败: %v", err)
	}
	probeA := exec.Command(filepath.Join(probeDir, "ffprobe.exe"),
		"-v", "error", "-select_streams", "a", "-show_entries", "stream=index", "-of", "csv=p=0", src)
	hideConsoleWindow(probeA)
	outA, _ := probeA.Output()
	probeD := exec.Command(filepath.Join(probeDir, "ffprobe.exe"),
		"-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", src)
	hideConsoleWindow(probeD)
	outD, _ := probeD.Output()
	dur, _ := strconv.ParseFloat(strings.TrimSpace(string(outD)), 64)
	t.Logf("续跑成品: codec=%s audio=%q duration=%.2fs", strings.TrimSpace(string(outV)), strings.TrimSpace(string(outA)), dur)
	if strings.TrimSpace(string(outV)) != "hevc" {
		t.Errorf("期望 hevc,实际 %s", strings.TrimSpace(string(outV)))
	}
	if strings.TrimSpace(string(outA)) == "" {
		t.Errorf("成品缺少音轨")
	}
	if dur < 73.5 || dur > 76.5 {
		t.Errorf("成品时长异常: %.2fs", dur)
	}
}

// TestPauseDuringTranscodeAndResume 验证转码中暂停不会把任务置为失败,
// 恢复后跳过下载直接续跑转码,最终完成并产出带音轨的 HEVC 文件。
func TestPauseDuringTranscodeAndResume(t *testing.T) {
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
	outDir := filepath.Join(baseDir, "out-pause")
	_ = os.RemoveAll(outDir)
	_ = os.MkdirAll(outDir, 0o755)
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
	s := &app.store.Settings
	s.UseProxy = true
	s.ProxyURL = proxy
	s.MaxConcurrent = 1
	s.AutoRetry = 1
	s.DownloadType = "视频（带音频）"
	s.Quality = "最佳可用"
	s.VideoCodec = "hevc"
	s.HevcEncoder = "cpu" // 强制 CPU:转码窗口足够长,便于在转码中暂停
	s.DownloadDir = outDir
	s.FFmpegPath = ffmpegPath

	app.AddLinks("https://www.youtube.com/watch?v=jNQXAC9IVRw")
	app.StartAllTasks()

	// 等进入"转码"阶段
	deadline := time.Now().Add(5 * time.Minute)
	sawTranscode := false
	for time.Now().Before(deadline) {
		time.Sleep(250 * time.Millisecond)
		task := app.Snapshot().Tasks[0]
		if strings.HasPrefix(task.Phase, "转码") {
			sawTranscode = true
			break
		}
		if task.Status == StatusComplete {
			t.Skipf("转码窗口太短,未赶上暂停时机")
		}
		if task.Status == StatusFailed {
			t.Skipf("下载未成功(可能是风控拦截),跳过: %s", task.Error)
		}
	}
	if !sawTranscode {
		t.Skipf("未在时限内观察到转码阶段")
	}

	// 转码中暂停:任务应保持"已暂停"而不是变为失败
	app.PauseTask(1)
	time.Sleep(1500 * time.Millisecond)
	task := app.Snapshot().Tasks[0]
	if task.Status != StatusPaused {
		t.Fatalf("转码中暂停后任务应保持已暂停,实际 %s(错误=%q)", task.Status, task.Error)
	}

	// 恢复:跳过下载、续跑转码
	app.StartTask(1)
	deadline = time.Now().Add(6 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		task = app.Snapshot().Tasks[0]
		if task.Status == StatusComplete {
			break
		}
		if task.Status == StatusFailed {
			t.Fatalf("恢复后续跑失败: %s", task.Error)
		}
	}
	if task.Status != StatusComplete {
		t.Fatalf("恢复后未完成: %s | %s", task.Status, task.Error)
	}

	// 成品校验:hevc + 带音轨
	mp4Path := ""
	entries, _ := os.ReadDir(outDir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".mp4") && !strings.HasSuffix(e.Name(), ".hevc-tmp.mp4") {
			mp4Path = filepath.Join(outDir, e.Name())
		}
	}
	if mp4Path == "" {
		t.Fatalf("输出目录没有 mp4 文件")
	}
	probeV := exec.Command(filepath.Join(filepath.Dir(ffmpegPath), "ffprobe.exe"),
		"-v", "error", "-select_streams", "v:0", "-show_entries", "stream=codec_name", "-of", "csv=p=0", mp4Path)
	hideConsoleWindow(probeV)
	outV, _ := probeV.Output()
	probeA := exec.Command(filepath.Join(filepath.Dir(ffmpegPath), "ffprobe.exe"),
		"-v", "error", "-select_streams", "a", "-show_entries", "stream=index", "-of", "csv=p=0", mp4Path)
	hideConsoleWindow(probeA)
	outA, _ := probeA.Output()
	t.Logf("暂停恢复后的成品: %s | codec=%s | audio=%q", filepath.Base(mp4Path), strings.TrimSpace(string(outV)), strings.TrimSpace(string(outA)))
	if strings.TrimSpace(string(outV)) != "hevc" {
		t.Errorf("期望 hevc,实际 %s", strings.TrimSpace(string(outV)))
	}
	if strings.TrimSpace(string(outA)) == "" {
		t.Errorf("成品缺少音轨")
	}
}

// TestTranscodeSpecialDirChars 验证源文件名含 #、中文、单引号等特殊字符时
// 分段转码仍能拼接成功(历史上相对路径会被 ffmpeg 当 URL 锚点截断,永远失败)。
func TestTranscodeSpecialDirChars(t *testing.T) {
	if os.Getenv("DOWNTUBE_IT") == "" {
		t.Skip("设置 DOWNTUBE_IT=1 运行该集成测试")
	}
	baseDir := filepath.Join(os.TempDir(), "downtube-it")
	_ = os.MkdirAll(baseDir, 0o755)
	ffmpegPath := ensureTestFFmpeg(t, baseDir)

	// 目录与文件名都带 # ⧸ 单引号,覆盖会触发路径截断的组合
	specialDir := filepath.Join(baseDir, "特殊 目录 #1")
	_ = os.MkdirAll(specialDir, 0o755)
	src := filepath.Join(specialDir, "标题 含#号 ⧸ it's 测试.mp4")
	_ = os.Remove(src)
	_ = os.RemoveAll(hevcPartsDir(src))
	gen := exec.Command(ffmpegPath, "-y", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc2=size=1280x720:rate=30:duration=40",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=40",
		"-c:v", "libx264", "-preset", "ultrafast", "-g", "60", "-pix_fmt", "yuv420p", "-c:a", "aac", "-shortest", src)
	hideConsoleWindow(gen)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("生成测试视频失败: %v %s", err, string(out))
	}

	app := &App{
		store:         Store{Version: 2, NextID: 2, Tasks: []Task{{ID: 1, Status: StatusRunning}}, Settings: defaultSettings()},
		workers:       make(map[int]context.CancelFunc),
		retryAttempts: make(map[int]int),
		bestProgress:  make(map[int]int),
		disableEvents: true,
		titleJobs:     make(chan struct{}, 2),
	}
	app.ctx = context.Background()

	if err := app.transcodeToHEVC(context.Background(), 1, src, 40, ffmpegPath); err != nil {
		t.Fatalf("特殊字符目录下转码失败: %v", err)
	}
	// 转码成功后源文件被替换为 HEVC 成品
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("成品不存在: %v", err)
	}
	if _, err := os.Stat(hevcPartsDir(src)); !os.IsNotExist(err) {
		t.Errorf("工作目录未清理")
	}
	probe := exec.Command(filepath.Join(filepath.Dir(ffmpegPath), "ffprobe.exe"), "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=codec_name", "-of", "csv=p=0", src)
	hideConsoleWindow(probe)
	out, err := probe.CombinedOutput()
	if err != nil {
		t.Fatalf("校验成品失败: %v %s", err, string(out))
	}
	codec := strings.TrimSpace(string(out))
	t.Logf("特殊字符目录下的成品编码: %s", codec)
	if codec != "hevc" {
		t.Errorf("期望 hevc,实际 %s", codec)
	}
	got, derr := probeDuration(ffmpegPath, src)
	if derr != nil {
		t.Fatalf("读取成品时长失败: %v", derr)
	}
	t.Logf("成品时长: %.2f 秒(源 40 秒)", got)
	if got < 38 || got > 42 {
		t.Errorf("成品时长异常: %.2f 秒", got)
	}
	_ = os.Remove(src)
}

// TestUpdateCheckLive 真实网络验证检查更新:CNB 或 GitHub 至少一路可用,
// 能从低版本检测到当前最新版,且不高于最新版时正确报告「已是最新」。
func TestUpdateCheckLive(t *testing.T) {
	if os.Getenv("DOWNTUBE_IT") == "" {
		t.Skip("设置 DOWNTUBE_IT=1 运行该集成测试")
	}
	app := &App{
		store:         Store{Version: 2, NextID: 1, Tasks: []Task{}, Settings: defaultSettings()},
		workers:       make(map[int]context.CancelFunc),
		retryAttempts: make(map[int]int),
		bestProgress:  make(map[int]int),
		disableEvents: true,
		titleJobs:     make(chan struct{}, 2),
	}

	// 低版本:应检测到更新,并给出可下载地址。
	low := app.CheckUpdate("0.0.1")
	t.Logf("0.0.1 → ok=%v hasUpdate=%v latest=%q source=%q url=%q",
		low.OK, low.HasUpdate, low.Latest, low.Source, low.URL)
	if low.OK != true {
		t.Fatalf("检查更新失败: %v", low.Error)
	}
	if low.HasUpdate != true {
		t.Fatalf("低版本未检出更新: %v", low)
	}
	latest := low.Latest
	if latest == "" {
		t.Fatal("latest 为空")
	}
	url := low.URL
	if !allowedUpdateHost(url) {
		t.Fatalf("下载地址不在白名单: %q", url)
	}

	// 等于最新版:应报告已是最新。
	same := app.CheckUpdate(latest)
	t.Logf("当前最新版 %s → ok=%v hasUpdate=%v", latest, same.OK, same.HasUpdate)
	if same.OK != true {
		t.Fatalf("最新版检查失败: %v", same.Error)
	}
	if same.HasUpdate != false {
		t.Fatalf("最新版仍报告有更新: %v", same)
	}

	// 更高版本:不应报告更新。
	high := app.CheckUpdate("99.0.0")
	if high.OK != true || high.HasUpdate != false {
		t.Fatalf("高版本判断异常: %+v", high)
	}

	// 已发布过的版本:必须都能检出当前版本。
	// 这条对应真实场景 —— 老用户打开软件时应看到更新提示。
	// 列表里保留最近几个版本即可;某天某个版本被淘汰(不再小于最新版)时删掉它。
	for _, prevVersion := range []string{"1.2.4", "1.2.5", "1.2.6", "1.2.7", "1.2.8", "1.3.0", "1.3.1", "1.3.2", "1.3.3"} {
		prev := app.CheckUpdate(prevVersion)
		t.Logf("%s → ok=%v hasUpdate=%v latest=%q source=%q", prevVersion, prev.OK, prev.HasUpdate, prev.Latest, prev.Source)
		if prev.OK != true {
			t.Fatalf("%s 检查失败: %v", prevVersion, prev.Error)
		}
		if prev.HasUpdate != true {
			t.Fatalf("%s 未检出更新,老用户将收不到提示: %+v", prevVersion, prev)
		}
	}
}

// TestUpdateDownloadLive 端到端下载安装包(真实网络),校验文件为合法 PE 安装包。
// 用 CNB 直连(国内稳定)验证下载链路;若配置了 DOWNTUBE_IT_PROXY 则同时覆盖代理分支。
func TestUpdateDownloadLive(t *testing.T) {
	if os.Getenv("DOWNTUBE_IT") == "" {
		t.Skip("设置 DOWNTUBE_IT=1 运行该集成测试")
	}
	settings := defaultSettings()
	if proxy := os.Getenv("DOWNTUBE_IT_PROXY"); proxy != "" {
		settings.UseProxy = true
		settings.ProxyURL = proxy
	}
	app := &App{
		store:         Store{Version: 2, NextID: 1, Tasks: []Task{}, Settings: settings},
		workers:       make(map[int]context.CancelFunc),
		retryAttempts: make(map[int]int),
		bestProgress:  make(map[int]int),
		disableEvents: true,
		titleJobs:     make(chan struct{}, 2),
	}

	// 取当前已发布版本的安装包地址:
	// 优先用 check 结果(新命名);探测失败时回退到 CNB 当前 Release 的已知地址,
	// 保证下载机制在「命名切换过渡期」也能被真实验证。
	url := ""
	if info := app.CheckUpdate("0.0.1"); info.OK == true && info.HasUpdate == true {
		url = info.URL
	}
	version := "1.2.3"
	if url == "" || !app.remoteFileExists(url) {
		url = "https://cnb.cool/youngdajie/downtube/-/releases/download/V1.2.3/DownTube-Setup-1.2.3-cnb-i9q-1k4ldedms.exe"
	}
	if !app.remoteFileExists(url) {
		t.Skipf("找不到可下载的安装包地址: %s", url)
	}

	result := app.DownloadUpdate(url, version)
	t.Logf("下载 %s → ok=%v size=%v error=%v", url, result.OK, result.Size, result.Error)
	if result.OK != true {
		t.Fatalf("下载安装包失败: %v", result.Error)
	}
	path := result.Path
	defer func() {
		_ = os.Remove(path)
		_ = os.RemoveAll(filepath.Join(os.TempDir(), updateTempDir))
	}()
	if err := validateInstallerFile(path); err != nil {
		t.Fatalf("安装包校验失败: %v", err)
	}
	info2, statErr := os.Stat(path)
	if statErr != nil || info2.Size() < 4<<20 {
		t.Fatalf("安装包体积异常: %v %v", statErr, info2)
	}
	t.Logf("安装包就绪: %s (%.1f MB)", path, float64(info2.Size())/1048576)
}
