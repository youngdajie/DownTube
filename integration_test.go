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
	s.DownloadType = "视频 + 音频"
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
	if !seenPhases["下载视频中"] || !seenPhases["下载音频中"] {
		t.Errorf("缺少流阶段文案: %v", seenPhases)
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
	s.DownloadType = "视频 + 音频"
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
	s.DownloadType = "视频 + 音频"
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
