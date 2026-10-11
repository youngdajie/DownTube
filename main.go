package main

import (
	"archive/zip"
	"bufio"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bodgit/sevenzip"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

type TaskStatus string

const (
	StatusQueued   TaskStatus = "等待中"
	StatusRunning  TaskStatus = "准备中"
	StatusPaused   TaskStatus = "已暂停"
	StatusComplete TaskStatus = "已完成"
	StatusFailed   TaskStatus = "失败"
)

type Task struct {
	ID        int        `json:"id"`
	URL       string     `json:"url"`
	Title     string     `json:"title"`
	Status    TaskStatus `json:"status"`
	Progress  int        `json:"progress"`
	Quality   string     `json:"quality"`
	CreatedAt time.Time  `json:"createdAt"`
	Error     string     `json:"error,omitempty"`
}

type Settings struct {
	Quality          string `json:"quality"`
	DownloadType     string `json:"downloadType"`
	DownloadDir      string `json:"downloadDir"`
	UseProxy         bool   `json:"useProxy"`
	ProxyURL         string `json:"proxyURL"`
	AutoDetectProxy  bool   `json:"autoDetectProxy"`
	UseBrowserCookie bool   `json:"useBrowserCookie"`
	CookieSource     string `json:"cookieSource"`
	CookieBrowser    string `json:"cookieBrowser"`
	CookieFile       string `json:"cookieFile"`
	FFmpegPath       string `json:"ffmpegPath"`
	HighFrameRate    bool   `json:"highFrameRate"`
	OpenOnComplete   bool   `json:"openOnComplete"`
}

type Snapshot struct {
	Tasks    []Task   `json:"tasks"`
	Settings Settings `json:"settings"`
}

type Store struct {
	Version  int      `json:"version"`
	NextID   int      `json:"nextId"`
	Tasks    []Task   `json:"tasks"`
	Settings Settings `json:"settings"`
}

type App struct {
	ctx              context.Context
	mu               sync.Mutex
	store            Store
	storePath        string
	workers          map[int]context.CancelFunc
	autoRun          bool
	ffmpegInstalling bool
	titleJobs        chan struct{}
}

type titleJob struct {
	id  int
	url string
}

type appLogger struct {
	mu   sync.Mutex
	file *os.File
}

func (l *appLogger) write(level, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = fmt.Fprintf(l.file, "%s %s %s\n", time.Now().Format(time.RFC3339), level, message)
}
func (l *appLogger) Print(s string)   { l.write("PRINT", s) }
func (l *appLogger) Trace(s string)   { l.write("TRACE", s) }
func (l *appLogger) Debug(s string)   { l.write("DEBUG", s) }
func (l *appLogger) Info(s string)    { l.write("INFO", s) }
func (l *appLogger) Warning(s string) { l.write("WARN", s) }
func (l *appLogger) Error(s string)   { l.write("ERROR", s) }
func (l *appLogger) Fatal(s string)   { l.write("FATAL", s) }

func defaultSettings() Settings {
	dir, err := os.UserHomeDir()
	if err != nil || dir == "" {
		dir, _ = os.Getwd()
	}
	return Settings{
		Quality: "1080p", DownloadType: "视频 + 音频",
		DownloadDir: filepath.Join(dir, "Downloads", "DownTube"),
		ProxyURL:    "http://127.0.0.1:10809", AutoDetectProxy: true,
		CookieBrowser: "Chrome",
	}
}

func NewApp() *App {
	app := &App{store: Store{Version: 1, NextID: 1, Tasks: []Task{}, Settings: defaultSettings()}, workers: make(map[int]context.CancelFunc), titleJobs: make(chan struct{}, 2)}
	if base, err := os.UserConfigDir(); err == nil {
		app.storePath = filepath.Join(base, "DownTube", "state.json")
		app.load()
	}
	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Print("Wails startup hook entered")
	_ = os.MkdirAll(a.store.Settings.DownloadDir, 0o755)
	// 对尚未解析出标题的任务补一次异步解析。
	a.mu.Lock()
	jobs := []titleJob{}
	for _, t := range a.store.Tasks {
		if t.Title == "" || t.Title == "待获取视频信息" || t.Title == "正在获取视频信息…" {
			jobs = append(jobs, titleJob{id: t.ID, url: t.URL})
		}
	}
	a.mu.Unlock()
	for _, j := range jobs {
		a.resolveTitleAsync(j.id, j.url)
	}
}

func (a *App) ReportFrontendError(message string) { log.Printf("frontend error: %s", message) }

func (a *App) OpenEngineDownloads() {
	runtime.BrowserOpenURL(a.ctx, "https://github.com/yt-dlp/yt-dlp/releases/latest")
}

func (a *App) OpenFFmpegDownloads() {
	runtime.BrowserOpenURL(a.ctx, "https://ffmpeg.org/download.html")
}

// OpenExternalURL 打开外部链接(仅供界面上的下载镜像等使用,仅放行 http/https)。
func (a *App) OpenExternalURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return false
	}
	runtime.BrowserOpenURL(a.ctx, u.String())
	return true
}

func (a *App) OpenHomepage() {
	runtime.BrowserOpenURL(a.ctx, "https://www.yangjie.site")
}

func (a *App) OpenBilibili() {
	runtime.BrowserOpenURL(a.ctx, "https://space.bilibili.com/646325099")
}

func (a *App) OpenProjectRepo() {
	runtime.BrowserOpenURL(a.ctx, "https://github.com/youngdajie/DownTube")
}

func (a *App) OpenCNBRepo() {
	runtime.BrowserOpenURL(a.ctx, "https://cnb.cool/youngdajie/downtube")
}

func (a *App) shutdown(context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, cancel := range a.workers {
		cancel()
	}
	_ = a.saveLocked()
}

func clone[T any](items []T) []T { return append([]T{}, items...) }

// nextRunnableLocked 在 autoRun 模式下寻找下一个待运行任务；调用方须持有锁。
func (a *App) nextRunnableLocked() int {
	if !a.autoRun {
		return 0
	}
	for _, task := range a.store.Tasks {
		if task.Status == StatusQueued || task.Status == StatusPaused {
			return task.ID
		}
	}
	a.autoRun = false
	return 0
}

func (a *App) snapshotLocked() Snapshot {
	return Snapshot{Tasks: clone(a.store.Tasks), Settings: a.store.Settings}
}

func (a *App) Snapshot() Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.snapshotLocked()
}

func (a *App) GetSystemStatus() map[string]any {
	engine, err := findYTDLP()
	ffmpeg, ffmpegErr := a.findFFmpeg()
	ffmpegLabel := ""
	if ffmpegErr == nil {
		if v, verr := ffmpegVersion(ffmpeg); verr == nil {
			ffmpegLabel = v
		}
	}
	a.mu.Lock()
	proxyEnabled, proxyURL, autoDetect := a.store.Settings.UseProxy, a.store.Settings.ProxyURL, a.store.Settings.AutoDetectProxy
	dataDir := filepath.Dir(a.storePath)
	a.mu.Unlock()
	proxyOK, proxyMessage := false, "代理检测已关闭"
	if proxyEnabled && autoDetect {
		result := a.TestProxy(proxyURL)
		proxyOK, _ = result["ok"].(bool)
		proxyMessage, _ = result["message"].(string)
	}
	return map[string]any{
		"engineReady": err == nil, "enginePath": engine, "engineError": errorText(err),
		"ffmpegReady": ffmpegErr == nil, "ffmpegPath": ffmpeg, "ffmpegError": errorText(ffmpegErr), "ffmpegVersion": ffmpegLabel,
		"proxyChecked": proxyEnabled && autoDetect, "proxyReady": proxyOK, "proxyMessage": proxyMessage,
		"dataDir": dataDir,
	}
}

// ffmpegInstallDir 返回用户数据目录下自动安装 FFmpeg 的备用目录。
func (a *App) ffmpegInstallDir() string {
	if base, err := os.UserConfigDir(); err == nil {
		return filepath.Join(base, "DownTube", "ffmpeg")
	}
	return ""
}

// ffmpegPortableDir 返回程序安装目录下的 FFmpeg 目录(优先安装位置)。
func (a *App) ffmpegPortableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), "ffmpeg")
}

// ensureWritable 确认目录存在且可写(通过一次探针写入判断)。
func ensureWritable(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	probe := filepath.Join(dir, ".downtube-write-test")
	f, err := os.Create(probe)
	if err != nil {
		return err
	}
	_ = f.Close()
	return os.Remove(probe)
}

// ffmpegTargetBinDir 返回导入安装包时 bin 目录的目标位置:
// 优先程序安装目录(便携/用户级安装通常可写),不可写时回退用户数据目录。
func (a *App) ffmpegTargetBinDir() (string, error) {
	if dir := a.ffmpegPortableDir(); dir != "" {
		if err := ensureWritable(dir); err == nil {
			return filepath.Join(dir, "bin"), nil
		}
	}
	if dir := a.ffmpegInstallDir(); dir != "" {
		if err := ensureWritable(dir); err == nil {
			return filepath.Join(dir, "bin"), nil
		}
	}
	return "", errors.New("无法创建 FFmpeg 安装目录(程序目录不可写)")
}

// findFFmpeg 依次查找:用户指定路径 → 程序目录内 FFmpeg → 用户数据目录自动安装 → 系统 PATH。
func (a *App) findFFmpeg() (string, error) {
	a.mu.Lock()
	custom := a.store.Settings.FFmpegPath
	a.mu.Unlock()
	if custom != "" {
		if info, err := os.Stat(custom); err == nil && !info.IsDir() {
			return custom, nil
		}
	}
	if dir := a.ffmpegPortableDir(); dir != "" {
		auto := filepath.Join(dir, "bin", "ffmpeg.exe")
		if info, err := os.Stat(auto); err == nil && !info.IsDir() {
			return auto, nil
		}
	}
	if dir := a.ffmpegInstallDir(); dir != "" {
		auto := filepath.Join(dir, "bin", "ffmpeg.exe")
		if info, err := os.Stat(auto); err == nil && !info.IsDir() {
			return auto, nil
		}
	}
	if path, err := exec.LookPath("ffmpeg.exe"); err == nil {
		return path, nil
	}
	return exec.LookPath("ffmpeg")
}

// ffmpegVersion 运行 ffmpeg -version 并返回首行（用于安装校验与界面展示）。
func ffmpegVersion(path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "-version")
	hideConsoleWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	line := string(out)
	if i := strings.IndexByte(line, '\n'); i > 0 {
		line = line[:i]
	}
	return strings.TrimSpace(line), nil
}

func (a *App) TestProxy(raw string) map[string]any {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" {
		return map[string]any{"ok": false, "message": "代理地址无效"}
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(u.Hostname(), port), 2*time.Second)
	if err != nil {
		return map[string]any{"ok": false, "message": "连接失败：" + err.Error()}
	}
	_ = conn.Close()
	return map[string]any{"ok": true, "message": fmt.Sprintf("代理端口可连接 · %d ms", time.Since(start).Milliseconds())}
}

func findYTDLP() (string, error) {
	if exe, err := os.Executable(); err == nil {
		for _, name := range []string{filepath.Join(filepath.Dir(exe), "yt-dlp.exe"), filepath.Join(filepath.Dir(exe), "tools", "yt-dlp.exe")} {
			if info, statErr := os.Stat(name); statErr == nil && !info.IsDir() {
				return name, nil
			}
		}
	}
	if path, err := exec.LookPath("yt-dlp"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("yt-dlp.exe"); err == nil {
		return path, nil
	}
	return "", errors.New("未找到 yt-dlp。请将 yt-dlp.exe 放在程序目录或加入 PATH。")
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// appendNetworkArgs 追加代理与 Cookies 参数（下载与标题解析共用）。
func appendNetworkArgs(args []string, settings Settings) []string {
	if settings.UseProxy && settings.ProxyURL != "" {
		args = append(args, "--proxy", settings.ProxyURL)
	}
	if settings.UseBrowserCookie {
		if settings.CookieSource == "file" && settings.CookieFile != "" {
			args = append(args, "--cookies", settings.CookieFile)
		} else {
			browserName := strings.ToLower(settings.CookieBrowser)
			if browserName == "microsoft edge" {
				browserName = "edge"
			}
			args = append(args, "--cookies-from-browser", browserName)
		}
	}
	return args
}

// fetchTitle 用 yt-dlp 仅解析标题（不下载），失败时返回空字符串。
func (a *App) fetchTitle(url string) string {
	engine, err := findYTDLP()
	if err != nil {
		return ""
	}
	settings := a.GetSettings()
	args := []string{"--skip-download", "--no-warnings", "--print", "%(title)j"}
	args = appendNetworkArgs(args, settings)
	args = append(args, url)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, engine, args...)
	hideConsoleWindow(cmd)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	var title string
	if json.Unmarshal([]byte(strings.TrimSpace(string(out))), &title) != nil {
		return ""
	}
	return strings.TrimSpace(title)
}

// resolveTitleAsync 在后台解析任务标题（并发上限 2）。
func (a *App) resolveTitleAsync(id int, url string) {
	go func() {
		a.titleJobs <- struct{}{}
		defer func() { <-a.titleJobs }()
		if title := a.fetchTitle(url); title != "" {
			a.updateTitle(id, title)
		}
	}()
}

func (a *App) GetTasks() []Task { return a.Snapshot().Tasks }

func (a *App) GetSettings() Settings { return a.Snapshot().Settings }

func normalizeYouTubeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.ParseRequestURI(raw)
	if err != nil || u == nil || (u.Scheme != "https" && u.Scheme != "http") {
		return "", fmt.Errorf("不是有效的 HTTP/HTTPS 链接")
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if host != "youtube.com" && host != "www.youtube.com" && host != "m.youtube.com" && host != "youtu.be" && host != "www.youtu.be" {
		return "", fmt.Errorf("只支持 youtube.com 和 youtu.be 链接")
	}
	if host == "youtu.be" || host == "www.youtu.be" {
		if strings.Trim(u.Path, "/") == "" {
			return "", fmt.Errorf("链接中没有视频 ID")
		}
	} else if u.Path != "/watch" && !strings.HasPrefix(u.Path, "/shorts/") && !strings.HasPrefix(u.Path, "/live/") && !strings.HasPrefix(u.Path, "/playlist") && !strings.HasPrefix(u.Path, "/embed/") {
		return "", fmt.Errorf("链接不是视频、直播或播放列表地址")
	}
	return u.String(), nil
}

func (a *App) AddLinks(input string) Snapshot {
	a.mu.Lock()
	parts := strings.FieldsFunc(input, func(r rune) bool { return r == '\n' || r == '\r' || r == '\t' || r == ' ' || r == ',' || r == ';' })
	jobs := []titleJob{}
	for _, raw := range parts {
		link, err := normalizeYouTubeURL(raw)
		if err != nil {
			continue
		}
		a.store.Tasks = append(a.store.Tasks, Task{ID: a.store.NextID, URL: link, Title: "正在获取视频信息…", Status: StatusQueued, Quality: a.store.Settings.Quality, CreatedAt: time.Now()})
		jobs = append(jobs, titleJob{id: a.store.NextID, url: link})
		a.store.NextID++
	}
	_ = a.saveLocked()
	snapshot := a.snapshotLocked()
	a.mu.Unlock()
	for _, j := range jobs {
		a.resolveTitleAsync(j.id, j.url)
	}
	return snapshot
}

func (a *App) RemoveTask(id int) Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.store.Tasks {
		if a.store.Tasks[i].ID == id {
			if cancel := a.workers[id]; cancel != nil {
				a.autoRun = false
				cancel()
			}
			a.store.Tasks = append(a.store.Tasks[:i], a.store.Tasks[i+1:]...)
			break
		}
	}
	_ = a.saveLocked()
	return a.snapshotLocked()
}

func (a *App) ClearTasks() Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.autoRun = false
	for _, cancel := range a.workers {
		cancel()
	}
	a.workers = make(map[int]context.CancelFunc)
	a.store.Tasks = []Task{}
	_ = a.saveLocked()
	return a.snapshotLocked()
}

func (a *App) StartTask(id int) Snapshot {
	a.mu.Lock()
	engine, engineErr := findYTDLP()
	var task Task
	for i := range a.store.Tasks {
		t := &a.store.Tasks[i]
		if t.ID == id && (t.Status == StatusQueued || t.Status == StatusPaused || t.Status == StatusFailed) {
			if len(a.workers) > 0 {
				a.autoRun = true
				t.Status = StatusQueued
				t.Error = ""
			} else if engineErr != nil {
				t.Status = StatusFailed
				t.Error = engineErr.Error()
			} else {
				ctx, cancel := context.WithCancel(a.ctx)
				a.workers[id] = cancel
				t.Status = StatusRunning
				t.Error = ""
				task = *t
				go a.runTask(ctx, id, engine, task)
			}
			break
		}
	}
	_ = a.saveLocked()
	snapshot := a.snapshotLocked()
	a.mu.Unlock()
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "state:updated", snapshot)
	}
	return snapshot
}

func (a *App) StartAllTasks() Snapshot {
	a.mu.Lock()
	a.autoRun = true
	nextID := 0
	for _, task := range a.store.Tasks {
		if task.Status == StatusQueued || task.Status == StatusPaused || task.Status == StatusFailed {
			nextID = task.ID
			break
		}
	}
	a.mu.Unlock()
	if nextID != 0 {
		a.StartTask(nextID)
	}
	return a.Snapshot()
}

func (a *App) PauseTask(id int) Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.store.Tasks {
		t := &a.store.Tasks[i]
		if t.ID == id && t.Status == StatusRunning {
			a.autoRun = false
			t.Status = StatusPaused
			if cancel := a.workers[id]; cancel != nil {
				cancel()
			}
			break
		}
	}
	_ = a.saveLocked()
	return a.snapshotLocked()
}

func (a *App) PauseAllTasks() Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.autoRun = false
	for i := range a.store.Tasks {
		t := &a.store.Tasks[i]
		if t.Status == StatusRunning {
			t.Status = StatusPaused
			if cancel := a.workers[t.ID]; cancel != nil {
				cancel()
			}
		}
	}
	_ = a.saveLocked()
	return a.snapshotLocked()
}

func (a *App) runTask(ctx context.Context, id int, engine string, task Task) {
	settings := a.GetSettings()
	ffmpegPath, ffmpegErr := a.findFFmpeg()
	args := []string{"--newline", "--progress", "--progress-template", "download:__DTPROGRESS__%(progress._percent_str)s|%(progress._speed_str)s|%(progress._eta_str)s", "--no-warnings", "-o", filepath.Join(settings.DownloadDir, "%(title)s.%(ext)s")}
	maxHeight := 0
	if settings.Quality != "最佳可用" && settings.Quality != "" {
		_, _ = fmt.Sscanf(strings.TrimSuffix(settings.Quality, "p"), "%d", &maxHeight)
	}
	if settings.HighFrameRate {
		args = append(args, "-S", "fps")
	}
	if settings.DownloadType == "仅视频" {
		format := "bv*/b"
		if maxHeight > 0 {
			format = fmt.Sprintf("bv*[height<=%d]/b[height<=%d]", maxHeight, maxHeight)
		}
		args = append(args, "-f", format)
	} else if settings.DownloadType == "仅音频 · MP3" {
		if ffmpegErr != nil {
			a.finishTask(id, errors.New("MP3 转换需要 FFmpeg。请安装 FFmpeg 并将其加入系统 PATH。"))
			return
		}
		args = append(args, "-x", "--audio-format", "mp3")
	} else if ffmpegErr != nil {
		format := "b"
		if maxHeight > 0 {
			format = fmt.Sprintf("b[height<=%d]", maxHeight)
		}
		args = append(args, "-f", format)
	} else if maxHeight > 0 {
		args = append(args, "-f", fmt.Sprintf("bv*[height<=%d]+ba/b[height<=%d]", maxHeight, maxHeight))
	}
	args = appendNetworkArgs(args, settings)
	if ffmpegErr == nil {
		args = append(args, "--ffmpeg-location", ffmpegPath)
	}
	args = append(args, "--print", "before_dl:__DOWNTUBE_TITLE__%(title)j")
	args = append(args, task.URL)
	if err := os.MkdirAll(settings.DownloadDir, 0o755); err != nil {
		a.finishTask(id, fmt.Errorf("创建保存目录失败：%w", err))
		return
	}
	cmd := exec.CommandContext(ctx, engine, args...)
	// 隐藏控制台窗口，并禁用 Python 输出缓冲，保证进度行实时到达。
	hideConsoleWindow(cmd)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		a.finishTask(id, err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		a.finishTask(id, err)
		return
	}
	if err := cmd.Start(); err != nil {
		a.finishTask(id, err)
		return
	}
	stderrText := make(chan string, 1)
	go func() { b, _ := io.ReadAll(io.LimitReader(stderr, 1<<20)); stderrText <- strings.TrimSpace(string(b)) }()
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "__DOWNTUBE_TITLE__") {
			raw := strings.TrimPrefix(line, "__DOWNTUBE_TITLE__")
			var title string
			if json.Unmarshal([]byte(raw), &title) == nil {
				a.updateTitle(id, title)
			}
		} else if strings.HasPrefix(line, "__DTPROGRESS__") {
			a.updateProgress(id, strings.TrimPrefix(line, "__DTPROGRESS__"))
		}
	}
	err = cmd.Wait()
	detail := <-stderrText
	if ctx.Err() != nil {
		a.mu.Lock()
		delete(a.workers, id)
		nextID := a.nextRunnableLocked()
		_ = a.saveLocked()
		a.mu.Unlock()
		if nextID != 0 {
			a.StartTask(nextID)
		}
		return
	}
	if err != nil && detail != "" {
		if ffmpegErr != nil && strings.Contains(detail, "Requested format is not available") {
			detail += "（该视频只提供分离的音视频流，需要安装 FFmpeg 才能下载；当前系统未检测到 FFmpeg）"
		}
		if strings.Contains(detail, "Could not copy Chrome cookie database") {
			detail += "（浏览器运行时会锁定 Cookies 数据库，请先完全退出 Edge / Chrome 再重试，或关闭「使用浏览器 Cookies」）"
		}
		err = fmt.Errorf("%w: %s", err, detail)
	}
	a.finishTask(id, err)
}

func (a *App) updateTitle(id int, title string) {
	title = strings.TrimSpace(title)
	if title == "" {
		return
	}
	a.mu.Lock()
	for i := range a.store.Tasks {
		if a.store.Tasks[i].ID == id {
			a.store.Tasks[i].Title = title
			break
		}
	}
	snapshot := a.snapshotLocked()
	a.mu.Unlock()
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "state:updated", snapshot)
	}
}

func (a *App) updateProgress(id int, line string) {
	parts := strings.Split(line, "|")
	if len(parts) == 0 {
		return
	}
	value := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(parts[0]), "%"))
	var progress int
	_, _ = fmt.Sscanf(value, "%d", &progress)
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	a.mu.Lock()
	changed := false
	for i := range a.store.Tasks {
		if a.store.Tasks[i].ID == id {
			if a.store.Tasks[i].Progress != progress {
				a.store.Tasks[i].Progress = progress
				changed = true
			}
			break
		}
	}
	snapshot := a.snapshotLocked()
	a.mu.Unlock()
	if changed && a.ctx != nil {
		runtime.EventsEmit(a.ctx, "state:updated", snapshot)
	}
}

func (a *App) finishTask(id int, err error) {
	a.mu.Lock()
	delete(a.workers, id)
	openFolder := false
	downloadDir := a.store.Settings.DownloadDir
	for i := range a.store.Tasks {
		t := &a.store.Tasks[i]
		if t.ID != id {
			continue
		}
		if err == nil {
			t.Status = StatusComplete
			t.Progress = 100
			t.Error = ""
			openFolder = a.store.Settings.OpenOnComplete
		} else {
			t.Status = StatusFailed
			t.Error = err.Error()
		}
		break
	}
	nextID := a.nextRunnableLocked()
	_ = a.saveLocked()
	snapshot := a.snapshotLocked()
	a.mu.Unlock()
	if openFolder {
		_ = exec.Command("explorer.exe", downloadDir).Start()
	}
	if nextID != 0 {
		a.StartTask(nextID)
	}
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "state:updated", snapshot)
	}
}

func (a *App) SaveSettings(settings Settings) Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	if settings.Quality == "" {
		settings.Quality = "1080p"
	}
	if settings.DownloadType == "" {
		settings.DownloadType = "视频 + 音频"
	}
	if settings.DownloadDir == "" {
		settings.DownloadDir = defaultSettings().DownloadDir
	}
	if settings.CookieBrowser == "" {
		settings.CookieBrowser = "Chrome"
	}
	if settings.CookieSource == "" {
		settings.CookieSource = "browser"
	}
	a.store.Settings = settings
	_ = a.saveLocked()
	return a.snapshotLocked()
}

func (a *App) SelectFolder() string {
	settings := a.GetSettings()
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "选择视频保存位置", DefaultDirectory: settings.DownloadDir})
	if err != nil || dir == "" {
		return ""
	}
	return filepath.Clean(dir)
}

func (a *App) SelectCookieFile() string {
	current := a.GetSettings().CookieFile
	defaultDir := ""
	if current != "" {
		defaultDir = filepath.Dir(current)
	}
	chosen, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "选择 cookies.txt 文件",
		DefaultDirectory: defaultDir,
		Filters: []runtime.FileFilter{
			{DisplayName: "Cookies 文件 (*.txt)", Pattern: "*.txt"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil || chosen == "" {
		return ""
	}
	return filepath.Clean(chosen)
}

// SelectFFmpegFile 让用户指定已下载的 ffmpeg.exe，校验后保存。
func (a *App) SelectFFmpegFile() map[string]any {
	chosen, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 ffmpeg.exe",
		Filters: []runtime.FileFilter{
			{DisplayName: "FFmpeg 程序 (ffmpeg.exe)", Pattern: "ffmpeg.exe"},
			{DisplayName: "可执行文件 (*.exe)", Pattern: "*.exe"},
		},
	})
	if err != nil || chosen == "" {
		return map[string]any{"ok": false, "canceled": true}
	}
	chosen = filepath.Clean(chosen)
	version, verr := ffmpegVersion(chosen)
	if verr != nil {
		return map[string]any{"ok": false, "message": "所选文件无法作为 FFmpeg 运行：" + verr.Error()}
	}
	a.mu.Lock()
	a.store.Settings.FFmpegPath = chosen
	_ = a.saveLocked()
	a.mu.Unlock()
	return map[string]any{"ok": true, "path": chosen, "version": version}
}

const (
	ffmpegDownloadURL1 = "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"
	ffmpegDownloadURL2 = "https://github.com/BtbN/FFmpeg-Builds/releases/latest/download/ffmpeg-master-latest-win64-gpl.zip"
)

func (a *App) emitFFmpegProgress(phase string, received, total int64, message string) {
	percent := 0
	if total > 0 {
		percent = int(received * 100 / total)
		if percent > 100 {
			percent = 100
		}
	}
	// 解压阶段没有可靠的总量信息,保持 0 由界面侧显示为不确定进度。
	if phase == "done" {
		percent = 100
	}
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "ffmpeg:progress", map[string]any{
			"phase": phase, "percent": percent, "message": message,
			"received": received, "total": total,
		})
	}
}

// InstallFFmpeg 自动下载并安装 FFmpeg(优先走用户配置的代理)。
// 解压目标优先程序安装目录,不可写时回退用户数据目录。
func (a *App) InstallFFmpeg() map[string]any {
	a.mu.Lock()
	if a.ffmpegInstalling {
		a.mu.Unlock()
		return map[string]any{"ok": false, "message": "FFmpeg 正在安装中,请稍候"}
	}
	a.ffmpegInstalling = true
	useProxy, proxyURL := a.store.Settings.UseProxy, a.store.Settings.ProxyURL
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.ffmpegInstalling = false
		a.mu.Unlock()
	}()

	binDir, err := a.ffmpegTargetBinDir()
	if err != nil {
		return map[string]any{"ok": false, "message": err.Error()}
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return map[string]any{"ok": false, "message": "创建安装目录失败:" + err.Error()}
	}

	tmpZip := filepath.Join(os.TempDir(), "downtube-ffmpeg.zip")
	defer os.Remove(tmpZip)

	proxy := ""
	if useProxy {
		proxy = proxyURL
	}

	var downloadErr error
	for i, src := range []string{ffmpegDownloadURL1, ffmpegDownloadURL2} {
		a.emitFFmpegProgress("download", 0, 0, fmt.Sprintf("正在连接下载源 %d/2 …", i+1))
		downloadErr = a.downloadFile(src, tmpZip, proxy)
		if downloadErr == nil {
			break
		}
	}
	if downloadErr != nil {
		return map[string]any{"ok": false, "message": "下载失败:" + downloadErr.Error() + "。如果网络受限,请在设置中开启代理后重试。"}
	}

	a.emitFFmpegProgress("extract", 0, 0, "正在解压安装(约需十余秒)…")
	if err := extractFFmpegArchive(tmpZip, binDir, nil); err != nil {
		return map[string]any{"ok": false, "message": "解压失败:" + err.Error()}
	}

	ffmpegPath := filepath.Join(binDir, "ffmpeg.exe")
	version, verr := ffmpegVersion(ffmpegPath)
	if verr != nil {
		return map[string]any{"ok": false, "message": "安装校验未通过:" + verr.Error()}
	}

	a.mu.Lock()
	a.store.Settings.FFmpegPath = ffmpegPath
	_ = a.saveLocked()
	a.mu.Unlock()

	a.emitFFmpegProgress("done", 0, 0, "安装完成")
	return map[string]any{"ok": true, "path": ffmpegPath, "version": version}
}

// ImportFFmpegArchive 让用户选择已下载的 FFmpeg 安装包(.7z / .zip),
// 自动解压到程序目录并配置 bin 下的可执行文件。
func (a *App) ImportFFmpegArchive() map[string]any {
	a.mu.Lock()
	if a.ffmpegInstalling {
		a.mu.Unlock()
		return map[string]any{"ok": false, "message": "FFmpeg 正在安装中,请稍候"}
	}
	a.ffmpegInstalling = true
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.ffmpegInstalling = false
		a.mu.Unlock()
	}()

	chosen, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 FFmpeg 安装包(.7z / .zip)",
		Filters: []runtime.FileFilter{
			{DisplayName: "FFmpeg 安装包 (*.7z;*.zip)", Pattern: "*.7z;*.zip"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil || chosen == "" {
		return map[string]any{"ok": false, "canceled": true}
	}
	chosen = filepath.Clean(chosen)
	switch strings.ToLower(filepath.Ext(chosen)) {
	case ".7z", ".zip":
	default:
		return map[string]any{"ok": false, "message": "仅支持 .7z 或 .zip 格式的安装包"}
	}

	binDir, err := a.ffmpegTargetBinDir()
	if err != nil {
		return map[string]any{"ok": false, "message": err.Error()}
	}

	a.emitFFmpegProgress("extract", 0, 0, "正在解压安装包…")
	progress := func(received int64) {
		a.emitFFmpegProgress("extract", 0, 0, fmt.Sprintf("正在解压(已写出 %.0f MB)…", float64(received)/1048576))
	}
	if err := extractFFmpegArchive(chosen, binDir, progress); err != nil {
		return map[string]any{"ok": false, "message": "解压失败:" + err.Error()}
	}

	ffmpegPath := filepath.Join(binDir, "ffmpeg.exe")
	version, verr := ffmpegVersion(ffmpegPath)
	if verr != nil {
		return map[string]any{"ok": false, "message": "安装校验未通过:" + verr.Error()}
	}

	a.mu.Lock()
	a.store.Settings.FFmpegPath = ffmpegPath
	_ = a.saveLocked()
	a.mu.Unlock()

	a.emitFFmpegProgress("done", 0, 0, "安装完成")
	return map[string]any{"ok": true, "path": ffmpegPath, "version": version}
}

// downloadFile 下载 src 到 dest，期间通过事件广播进度；proxyURL 为空时直连。
func (a *App) downloadFile(src, dest, proxyURL string) error {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if proxyURL != "" {
		if u, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}
	client := &http.Client{Transport: transport}
	resp, err := client.Get(src)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	total := resp.ContentLength
	buf := make([]byte, 1<<20)
	var received int64
	last := time.Now()
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return werr
			}
			received += int64(n)
			if time.Since(last) >= 250*time.Millisecond {
				msg := fmt.Sprintf("已下载 %.0f MB", float64(received)/1048576)
				if total > 0 {
					msg = fmt.Sprintf("已下载 %.0f / %.0f MB", float64(received)/1048576, float64(total)/1048576)
				}
				a.emitFFmpegProgress("download", received, total, msg)
				last = time.Now()
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	a.emitFFmpegProgress("download", received, total, "下载完成")
	return nil
}

// extractFFmpegArchive 从 .7z 或 .zip 安装包中仅提取 bin 目录下的 ffmpeg.exe 与 ffprobe.exe。
// onProgress 可为 nil,收到的是已写出的字节数(压缩包内通常还包含文档等文件,进度仅供提示)。
func extractFFmpegArchive(archivePath, destBinDir string, onProgress func(int64)) error {
	if err := os.MkdirAll(destBinDir, 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(archivePath); err != nil {
		return errors.New("安装包文件不存在或不可读")
	}
	switch strings.ToLower(filepath.Ext(archivePath)) {
	case ".7z":
		return extractSevenZip(archivePath, destBinDir, onProgress)
	case ".zip":
		return extractZip(archivePath, destBinDir, onProgress)
	default:
		return errors.New("不支持的压缩格式,请选择 .7z 或 .zip 安装包")
	}
}

// ffmpegBinEntry 判断压缩包条目是否为 bin 目录下的目标可执行文件,返回规范化文件名。
func ffmpegBinEntry(name string, isDir bool) (string, bool) {
	if isDir {
		return "", false
	}
	slashed := strings.ReplaceAll(filepath.ToSlash(name), "\\", "/")
	base := strings.ToLower(path.Base(slashed))
	if base != "ffmpeg.exe" && base != "ffprobe.exe" {
		return "", false
	}
	if strings.ToLower(path.Base(path.Dir(slashed))) != "bin" {
		return "", false
	}
	return base, true
}

// progressReader 包装解压流,按节流节奏回调已读字节数。
type progressReader struct {
	r          io.Reader
	onProgress func(int64)
	received   int64
	last       time.Time
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.received += int64(n)
		now := time.Now()
		if now.Sub(p.last) >= 400*time.Millisecond {
			p.last = now
			p.onProgress(p.received)
		}
	}
	return n, err
}

// writeEntry 将压缩包内一个条目写出到目标 bin 目录。
func writeEntry(name, destBinDir string, open func() (io.ReadCloser, error), onProgress func(int64)) error {
	rc, err := open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.Create(filepath.Join(destBinDir, name))
	if err != nil {
		return err
	}
	defer out.Close()
	var src io.Reader = rc
	if onProgress != nil {
		src = &progressReader{r: rc, onProgress: onProgress}
	}
	_, err = io.Copy(out, src)
	return err
}

func extractZip(zipPath, destBinDir string, onProgress func(int64)) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	extracted := 0
	for _, f := range r.File {
		name, ok := ffmpegBinEntry(f.Name, f.FileInfo().IsDir())
		if !ok {
			continue
		}
		if err := writeEntry(name, destBinDir, f.Open, onProgress); err != nil {
			return err
		}
		extracted++
	}
	if extracted < 2 {
		return errors.New("压缩包结构异常(未找到 bin 目录下的 ffmpeg.exe / ffprobe.exe)")
	}
	return nil
}

func extractSevenZip(archivePath, destBinDir string, onProgress func(int64)) error {
	r, err := sevenzip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("无法读取 7z 压缩包:%w", err)
	}
	defer r.Close()
	extracted := 0
	for _, f := range r.File {
		name, ok := ffmpegBinEntry(f.Name, f.FileInfo().IsDir())
		if !ok {
			continue
		}
		if err := writeEntry(name, destBinDir, f.Open, onProgress); err != nil {
			return err
		}
		extracted++
	}
	if extracted < 2 {
		return errors.New("压缩包结构异常(未找到 bin 目录下的 ffmpeg.exe / ffprobe.exe)")
	}
	return nil
}

var browserProcessNames = map[string][]string{
	"chrome":         {"chrome.exe"},
	"microsoft edge": {"msedge.exe"},
	"firefox":        {"firefox.exe"},
	"brave":          {"brave.exe"},
}

func (a *App) CheckBrowserRunning(browser string) bool {
	names := browserProcessNames[strings.ToLower(strings.TrimSpace(browser))]
	if len(names) == 0 {
		return false
	}
	for _, name := range names {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		cmd := exec.CommandContext(ctx, "tasklist", "/FI", "IMAGENAME eq "+name, "/NH")
		hideConsoleWindow(cmd)
		out, err := cmd.Output()
		cancel()
		if err == nil && strings.Contains(string(out), name) {
			return true
		}
	}
	return false
}

func (a *App) OpenDownloadFolder() error {
	dir := a.GetSettings().DownloadDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return exec.Command("explorer.exe", dir).Start()
}

// OpenDataFolder 打开程序数据目录(存放 state.json 与回退安装的组件)。
func (a *App) OpenDataFolder() error {
	if base, err := os.UserConfigDir(); err == nil {
		dir := filepath.Join(base, "DownTube")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		return exec.Command("explorer.exe", dir).Start()
	}
	return errors.New("无法确定程序数据目录")
}

func (a *App) load() {
	data, err := os.ReadFile(a.storePath)
	if err != nil {
		return
	}
	var saved Store
	if jerr := json.Unmarshal(data, &saved); jerr != nil {
		// 配置损坏时先留一份现场，避免后续保存覆盖掉可能的人工恢复素材。
		backup := a.storePath + ".corrupt-" + time.Now().Format("20060102-150405")
		if werr := os.WriteFile(backup, data, 0o600); werr == nil {
			log.Printf("state.json 解析失败（%v），已备份到 %s", jerr, backup)
		} else {
			log.Printf("state.json 解析失败（%v），备份也失败：%v", jerr, werr)
		}
		return
	}
	if saved.NextID < 1 {
		saved.NextID = 1
	}
	if saved.Settings.Quality == "" {
		saved.Settings = defaultSettings()
	}
	if saved.Tasks == nil {
		saved.Tasks = []Task{}
	}
	for i := range saved.Tasks {
		if saved.Tasks[i].Status == StatusRunning {
			saved.Tasks[i].Status = StatusQueued
		}
	}
	a.store = saved
}

func (a *App) saveLocked() (err error) {
	defer func() {
		if err != nil {
			log.Printf("保存本地状态失败：%v", err)
		}
	}()
	if a.storePath == "" {
		return errors.New("无法确定本地配置目录")
	}
	if err := os.MkdirAll(filepath.Dir(a.storePath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(a.store, "", "  ")
	if err != nil {
		return err
	}
	tmp := a.storePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, a.storePath)
}

func main() {
	releaseInstance, fresh := acquireSingleInstance()
	if !fresh {
		showAlreadyRunningDialog()
		return
	}
	defer releaseInstance()
	logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "downtube-dev.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Print("main entered")
	app := NewApp()
	frontendAssets, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		log.Printf("prepare frontend assets: %v", err)
		panic(err)
	}
	err = wails.Run(&options.App{
		Title: "DownTube",
		Width: 1180, Height: 780, MinWidth: 980, MinHeight: 650,
		Frameless:          true,
		BackgroundColour:   options.NewRGB(16, 18, 22),
		AssetServer:        &assetserver.Options{Assets: frontendAssets},
		OnStartup:          app.startup,
		OnDomReady:         func(context.Context) { log.Print("WebView DOM ready") },
		OnShutdown:         app.shutdown,
		Logger:             &appLogger{file: logFile},
		LogLevel:           logger.DEBUG,
		LogLevelProduction: logger.DEBUG,
		Bind:               []interface{}{app},
		Windows:            &windows.Options{WebviewIsTransparent: false, WindowIsTranslucent: false},
	})
	if err != nil {
		log.Printf("wails run: %v", err)
		panic(err)
	}
}
