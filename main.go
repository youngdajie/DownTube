package main

import (
	"archive/zip"
	"bufio"
	"bytes"
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
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/bodgit/sevenzip"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/text/encoding/simplifiedchinese"
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
	Phase     string     `json:"phase,omitempty"`
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
	MaxConcurrent    int    `json:"maxConcurrent"`
	AutoRetry        int    `json:"autoRetry"`
	VideoCodec       string `json:"videoCodec"`
	HevcEncoder      string `json:"hevcEncoder"`
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
	retryAttempts    map[int]int
	bestProgress     map[int]int
	autoRun          bool
	ffmpegInstalling bool
	disableEvents    bool
	titleJobs        chan struct{}
	hevcMu           sync.Mutex
	hevcProbed       bool
	hevcProbedFor    string
	hevcGPU          *hevcEncoderSpec
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
		Quality: "1080p", DownloadType: "视频（带音频）",
		DownloadDir: filepath.Join(dir, "Downloads", "DownTube"),
		ProxyURL:    "http://127.0.0.1:10809", AutoDetectProxy: true,
		CookieBrowser: "Chrome",
		MaxConcurrent: 3, AutoRetry: 2,
		VideoCodec: "auto", HevcEncoder: "auto",
	}
}

func NewApp() *App {
	app := &App{store: Store{Version: 2, NextID: 1, Tasks: []Task{}, Settings: defaultSettings()}, workers: make(map[int]context.CancelFunc), retryAttempts: make(map[int]int), bestProgress: make(map[int]int), titleJobs: make(chan struct{}, 2)}
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
		if t.Title == "" || t.Title == "待获取视频信息" || t.Title == "正在获取视频信息…" || strings.HasPrefix(t.Title, "未获取到标题") {
			jobs = append(jobs, titleJob{id: t.ID, url: t.URL})
		}
	}
	a.mu.Unlock()
	for _, j := range jobs {
		a.resolveTitleAsync(j.id, j.url, false)
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

func (a *App) snapshotLocked() Snapshot {
	return Snapshot{Tasks: clone(a.store.Tasks), Settings: a.store.Settings}
}

// emit 向界面广播状态快照;无界面上下文(如自动化测试)时跳过。
func (a *App) emit(snapshot Snapshot) {
	if a.ctx == nil || a.disableEvents {
		return
	}
	runtime.EventsEmit(a.ctx, "state:updated", snapshot)
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
	return "", errors.New("未找到 yt-dlp。请将 yt-dlp.exe 放在程序目录或加入 PATH")
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
		if settings.CookieSource == "file" {
			// 明确选择"从文件导入"时只用文件;没选文件就不带 cookies,
			// 不悄悄回退到浏览器方式,避免用错身份导致下载异常。
			if settings.CookieFile != "" {
				args = append(args, "--cookies", settings.CookieFile)
			}
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

// fetchTitle 用 yt-dlp 仅解析标题(不下载),失败自动重试一次,仍失败返回空字符串。
func (a *App) fetchTitle(url string) string {
	engine, err := findYTDLP()
	if err != nil {
		return ""
	}
	settings := a.GetSettings()
	args := []string{"--skip-download", "--no-warnings", "--no-playlist", "--socket-timeout", "15", "--print", "%(title)j"}
	args = appendNetworkArgs(args, settings)
	args = append(args, url)
	for attempt := 0; attempt < 2; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		cmd := exec.CommandContext(ctx, engine, args...)
		hideConsoleWindow(cmd)
		cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
		out, err := cmd.Output()
		cancel()
		if err == nil {
			var title string
			if json.Unmarshal([]byte(strings.TrimSpace(string(out))), &title) == nil && strings.TrimSpace(title) != "" {
				return strings.TrimSpace(title)
			}
		}
		if attempt == 0 {
			time.Sleep(1200 * time.Millisecond)
		}
	}
	return ""
}

// resolveTitleAsync 在后台解析任务标题（并发上限 2）。
// autoStart 为 true 时(新添加的链接),解析完成或失败后自动接入下载。
func (a *App) resolveTitleAsync(id int, url string, autoStart bool) {
	go func() {
		a.titleJobs <- struct{}{}
		defer func() { <-a.titleJobs }()
		if title := a.fetchTitle(url); title != "" {
			a.updateTitle(id, title)
		} else {
			a.markTitleUnresolved(id)
		}
		if autoStart {
			a.autoStartAfterResolve(id)
		}
	}()
}

// autoStartAfterResolve 解析结束后自动开始该任务:
// 有空闲并发位时立即下载,否则留在队列,由调度器在有空位时接上。
func (a *App) autoStartAfterResolve(id int) {
	a.mu.Lock()
	engine, engineErr := findYTDLP()
	var launches []launchSpec
	started := false
	for i := range a.store.Tasks {
		t := &a.store.Tasks[i]
		if t.ID != id {
			continue
		}
		// 用户已暂停/移除,或任务已进入其他流程时不做干预
		if t.Status != StatusQueued || t.Phase != "" {
			break
		}
		if engineErr != nil || a.ctx == nil {
			break
		}
		a.autoRun = true
		if len(a.workers) < a.maxConcurrentLocked() {
			ctx, cancel := context.WithCancel(a.ctx)
			a.workers[id] = cancel
			t.Status = StatusRunning
			t.Error = ""
			t.Phase = ""
			launches = append(launches, launchSpec{id: id, task: *t, ctx: ctx, engine: engine})
			started = true
		}
		break
	}
	_ = a.saveLocked()
	snapshot := a.snapshotLocked()
	a.mu.Unlock()
	a.dispatch(launches)
	if started {
		a.emit(snapshot)
	}
}

// markTitleUnresolved 把尚未解析出标题的占位文案替换掉,避免一直停留在"正在获取"。
func (a *App) markTitleUnresolved(id int) {
	a.mu.Lock()
	changed := false
	for i := range a.store.Tasks {
		t := &a.store.Tasks[i]
		if t.ID != id {
			continue
		}
		if t.Title == "正在获取视频信息…" || t.Title == "待获取视频信息" || t.Title == "" {
			t.Title = "未获取到标题，下载时自动解析"
			changed = true
		}
		break
	}
	snapshot := a.snapshotLocked()
	a.mu.Unlock()
	if changed {
		a.emit(snapshot)
	}
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
		a.resolveTitleAsync(j.id, j.url, true)
	}
	return snapshot
}

func (a *App) RemoveTask(id int) Snapshot {
	a.mu.Lock()
	for i := range a.store.Tasks {
		if a.store.Tasks[i].ID == id {
			if cancel := a.workers[id]; cancel != nil {
				cancel()
				delete(a.workers, id)
			}
			delete(a.retryAttempts, id)
			delete(a.bestProgress, id)
			a.store.Tasks = append(a.store.Tasks[:i], a.store.Tasks[i+1:]...)
			break
		}
	}
	// 移除占用的并发位后,让排队任务继续
	engine, engineErr := findYTDLP()
	launches := a.fillSlotsLocked(engine, engineErr)
	_ = a.saveLocked()
	snapshot := a.snapshotLocked()
	a.mu.Unlock()
	a.dispatch(launches)
	a.emit(snapshot)
	return snapshot
}

func (a *App) ClearTasks() Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.autoRun = false
	for _, cancel := range a.workers {
		cancel()
	}
	a.workers = make(map[int]context.CancelFunc)
	a.retryAttempts = make(map[int]int)
	a.bestProgress = make(map[int]int)
	a.store.Tasks = []Task{}
	_ = a.saveLocked()
	return a.snapshotLocked()
}

// ============================================================
// 下载调度:并发执行 + 失败自动重试
// ============================================================

// maxConcurrentLocked 返回归一化后的并发下载上限(1-6,默认 3)。调用方须持有锁。
func (a *App) maxConcurrentLocked() int {
	n := a.store.Settings.MaxConcurrent
	if n <= 0 {
		n = 3
	}
	if n > 6 {
		n = 6
	}
	return n
}

// launchSpec 描述一个待启动的下载任务(锁内收集,锁外启动)。
type launchSpec struct {
	id     int
	task   Task
	ctx    context.Context
	engine string
}

// fillSlotsLocked 在自动调度模式下把排队任务填入并发空位,返回待启动清单。
// 调用方须持有锁;"等待重试"中的任务(Phase 非空)由定时器单独拉起。
func (a *App) fillSlotsLocked(engine string, engineErr error) []launchSpec {
	if !a.autoRun || a.ctx == nil {
		return nil
	}
	if engineErr != nil {
		for i := range a.store.Tasks {
			t := &a.store.Tasks[i]
			if t.Status == StatusQueued {
				t.Status = StatusFailed
				t.Phase = ""
				t.Error = engineErr.Error()
				delete(a.retryAttempts, t.ID)
				delete(a.bestProgress, t.ID)
			}
		}
		a.autoRun = false
		return nil
	}
	var out []launchSpec
	for i := range a.store.Tasks {
		if len(a.workers) >= a.maxConcurrentLocked() {
			break
		}
		t := &a.store.Tasks[i]
		if t.Status != StatusQueued || t.Phase != "" {
			continue
		}
		ctx, cancel := context.WithCancel(a.ctx)
		a.workers[t.ID] = cancel
		t.Status = StatusRunning
		t.Error = ""
		out = append(out, launchSpec{id: t.ID, task: *t, ctx: ctx, engine: engine})
	}
	return out
}

// dispatch 在锁外启动收集到的任务。
func (a *App) dispatch(launches []launchSpec) {
	for _, l := range launches {
		go a.runTask(l.ctx, l.id, l.engine, l.task)
	}
}

// StartTask 启动单个任务:有空闲并发位时立即开始,否则进入等待队列。
func (a *App) StartTask(id int) Snapshot {
	a.mu.Lock()
	engine, engineErr := findYTDLP()
	var launches []launchSpec
	for i := range a.store.Tasks {
		t := &a.store.Tasks[i]
		if t.ID != id {
			continue
		}
		if t.Status != StatusQueued && t.Status != StatusPaused && t.Status != StatusFailed {
			break
		}
		// 手动开始视为重新出发,清掉历史重试计数与阶段提示
		delete(a.retryAttempts, id)
		delete(a.bestProgress, id)
		t.Error = ""
		t.Phase = ""
		if engineErr != nil {
			t.Status = StatusFailed
			t.Error = engineErr.Error()
			break
		}
		if len(a.workers) < a.maxConcurrentLocked() {
			ctx, cancel := context.WithCancel(a.ctx)
			a.workers[id] = cancel
			t.Status = StatusRunning
			launches = append(launches, launchSpec{id: id, task: *t, ctx: ctx, engine: engine})
		} else {
			// 并发已满:进入等待队列,由自动调度在有空位时接上
			a.autoRun = true
			t.Status = StatusQueued
		}
		break
	}
	_ = a.saveLocked()
	snapshot := a.snapshotLocked()
	a.mu.Unlock()
	a.dispatch(launches)
	a.emit(snapshot)
	return snapshot
}

// StartAllTasks 把所有待下载任务排队,并按并发上限依次开始。
func (a *App) StartAllTasks() Snapshot {
	a.mu.Lock()
	a.autoRun = true
	for i := range a.store.Tasks {
		t := &a.store.Tasks[i]
		if t.Status == StatusPaused || t.Status == StatusFailed {
			t.Status = StatusQueued
			t.Error = ""
			t.Phase = ""
			delete(a.retryAttempts, t.ID)
			delete(a.bestProgress, t.ID)
		}
	}
	engine, engineErr := findYTDLP()
	launches := a.fillSlotsLocked(engine, engineErr)
	_ = a.saveLocked()
	snapshot := a.snapshotLocked()
	a.mu.Unlock()
	a.dispatch(launches)
	a.emit(snapshot)
	return snapshot
}

// PauseTask 暂停单个任务;若有排队任务,空出的并发位会自动补上。
func (a *App) PauseTask(id int) Snapshot {
	a.mu.Lock()
	for i := range a.store.Tasks {
		t := &a.store.Tasks[i]
		if t.ID == id && t.Status == StatusRunning {
			t.Status = StatusPaused
			t.Phase = ""
			if cancel := a.workers[id]; cancel != nil {
				cancel()
				delete(a.workers, id)
			}
			break
		}
	}
	engine, engineErr := findYTDLP()
	launches := a.fillSlotsLocked(engine, engineErr)
	_ = a.saveLocked()
	snapshot := a.snapshotLocked()
	a.mu.Unlock()
	a.dispatch(launches)
	a.emit(snapshot)
	return snapshot
}

// PauseAllTasks 暂停所有正在下载的任务并停止自动调度。
func (a *App) PauseAllTasks() Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.autoRun = false
	for i := range a.store.Tasks {
		t := &a.store.Tasks[i]
		if t.Status == StatusRunning {
			t.Status = StatusPaused
			t.Phase = ""
			if cancel := a.workers[t.ID]; cancel != nil {
				cancel()
				delete(a.workers, t.ID)
			}
		}
	}
	_ = a.saveLocked()
	return a.snapshotLocked()
}

// scheduleRetry 在延迟后重新拉起一个"等待重试"的任务。
func (a *App) scheduleRetry(id int, delay time.Duration) {
	time.AfterFunc(delay, func() {
		a.mu.Lock()
		var launches []launchSpec
		engine, engineErr := findYTDLP()
		if a.ctx != nil {
			for i := range a.store.Tasks {
				t := &a.store.Tasks[i]
				if t.ID != id {
					continue
				}
				if t.Status != StatusQueued || !strings.HasPrefix(t.Phase, "重试中") {
					break
				}
				if engineErr != nil {
					t.Status = StatusFailed
					t.Phase = ""
					t.Error = engineErr.Error()
					break
				}
				if len(a.workers) < a.maxConcurrentLocked() {
					ctx, cancel := context.WithCancel(a.ctx)
					a.workers[id] = cancel
					t.Status = StatusRunning
					t.Phase = ""
					// 重试已经开始:清掉上一次失败的错误文案,避免运行/转码期间还挂着旧报错
					t.Error = ""
					launches = append(launches, launchSpec{id: id, task: *t, ctx: ctx, engine: engine})
				} else {
					// 还没有空位:转回普通排队,由调度器接上
					t.Phase = ""
					t.Error = ""
					a.autoRun = true
				}
				break
			}
		}
		launches = append(launches, a.fillSlotsLocked(engine, engineErr)...)
		_ = a.saveLocked()
		snapshot := a.snapshotLocked()
		a.mu.Unlock()
		a.dispatch(launches)
			a.emit(snapshot)
	})
}

func (a *App) runTask(ctx context.Context, id int, engine string, task Task) {
	settings := a.GetSettings()
	ffmpegPath, ffmpegErr := a.findFFmpeg()
	args := []string{
		"--newline", "--progress",
		"--progress-template", "download:__DTPROGRESS__%(progress._percent_str)s|%(progress._speed_str)s|%(progress._eta_str)s|%(info.vcodec)s|%(info.acodec)s",
		"--no-warnings",
		// 断点续传与网络层重试:文件保留 .part,中断后可续下;分片与文件访问错误也自动重试
		"--continue",
		"--retries", "10",
		"--fragment-retries", "10",
		"--file-access-retries", "5",
		"-o", filepath.Join(settings.DownloadDir, "%(title)s.%(ext)s"),
	}
	maxHeight := 0
	if settings.Quality != "最佳可用" && settings.Quality != "" {
		_, _ = fmt.Sscanf(strings.TrimSuffix(settings.Quality, "p"), "%d", &maxHeight)
	}
	if settings.HighFrameRate {
		// 分辨率优先,其次帧率,再偏好兼容性最好的 H.264 + AAC(mp4)组合
		args = append(args, "-S", "res,fps,codec:avc:m4a")
	} else {
		// 分辨率优先,再偏好 H.264 + AAC(mp4),播放兼容性最好;
		// 高画质档位没有对应编码时(如 4K 的 AV1/VP9)自动回退,不影响清晰度选择
		args = append(args, "-S", "res,codec:avc:m4a")
	}
	audioOnly := strings.HasPrefix(settings.DownloadType, "仅音频")
	// H.265 需要在下载完成后重新编码;音频模式无需转码
	hevcRecode := settings.VideoCodec == "hevc" && !audioOnly
	if hevcRecode && ffmpegErr != nil {
		a.finishTask(id, errors.New("H.265 转码需要 FFmpeg，请先安装 FFmpeg 或导入安装包"))
		return
	}
	if settings.DownloadType == "仅视频" {
		format := "bv*/b"
		if maxHeight > 0 {
			format = fmt.Sprintf("bv*[height<=%d]/b[height<=%d]", maxHeight, maxHeight)
		}
		args = append(args, "-f", format)
	} else if audioOnly {
		if ffmpegErr != nil {
			a.finishTask(id, errors.New("音频格式转换需要 FFmpeg，请先安装 FFmpeg 或导入安装包"))
			return
		}
		audioFormat := "mp3"
		if strings.Contains(settings.DownloadType, "FLAC") {
			audioFormat = "flac"
		}
		args = append(args, "-x", "--audio-format", audioFormat, "--audio-quality", "0")
	} else if ffmpegErr != nil {
		format := "b"
		if maxHeight > 0 {
			format = fmt.Sprintf("b[height<=%d]", maxHeight)
		}
		args = append(args, "-f", format)
	} else if maxHeight > 0 {
		args = append(args, "-f", fmt.Sprintf("bv*[height<=%d]+ba/b[height<=%d]", maxHeight, maxHeight))
	}
	// "视频（带音频）"会下载两路流并合并为单文件:仅在此时校验同名旧文件是否带音轨
	wantsAudioMerge := !audioOnly && settings.DownloadType != "仅视频" && ffmpegErr == nil
	args = appendNetworkArgs(args, settings)
	if ffmpegErr == nil {
		args = append(args, "--ffmpeg-location", ffmpegPath)
	}
	args = append(args, "--print", "before_dl:__DOWNTUBE_TITLE__%(title)j")
	args = append(args, "--print", "before_dl:__DTFMT__%(format_id)s")
	args = append(args, "--print", "after_move:__DTPATH__%(filepath)j")
	args = append(args, "--print", "after_move:__DTDUR__%(duration)s")
	args = append(args, task.URL)
	if err := os.MkdirAll(settings.DownloadDir, 0o755); err != nil {
		a.finishTask(id, fmt.Errorf("创建保存目录失败：%w", err))
		return
	}
	res, err := a.execDownload(ctx, id, engine, args, settings)
	if ctx.Err() != nil {
		a.mu.Lock()
		delete(a.workers, id)
		eng, engErr := findYTDLP()
		launches := a.fillSlotsLocked(eng, engErr)
		_ = a.saveLocked()
		snapshot := a.snapshotLocked()
		a.mu.Unlock()
		a.dispatch(launches)
		a.emit(snapshot)
		return
	}
	// 合并模式下整轮没有任何下载进度,说明命中了「同名文件已存在」的跳过:
	// 校验已有文件是否真的包含音轨,不包含时把旧文件改名保留并重新下载一次,
	// 避免「视频（带音频）」最终留下的是无声的旧文件
	if err == nil && wantsAudioMerge && !res.sawProgress && res.finalPath != "" {
		if !fileHasAudio(ffmpegPath, res.finalPath) {
			if moveAsideExisting(res.finalPath) != "" {
				res, err = a.execDownload(ctx, id, engine, args, settings)
			}
		}
	}
	if err == nil {
		// 清理与成品同名的残留中间文件(旧流文件、断点文件、转码临时文件)
		cleanupStreamRemnants(res.finalPath)
	}
	if err == nil && hevcRecode {
		if terr := a.transcodeToHEVC(ctx, id, res.finalPath, res.duration, ffmpegPath); terr != nil {
			if ctx.Err() != nil {
				// 用户暂停或退出导致的中断:原文件与已完成的转码片段都保留,任务维持原状态,恢复时续跑
				a.mu.Lock()
				delete(a.workers, id)
				eng, engErr := findYTDLP()
				launches := a.fillSlotsLocked(eng, engErr)
				_ = a.saveLocked()
				snapshot := a.snapshotLocked()
				a.mu.Unlock()
				a.dispatch(launches)
				a.emit(snapshot)
				return
			}
			// 转码失败:原视频已保留,不再进入自动重试(重试会重复下载)
			a.finishTaskEx(id, fmt.Errorf("下载已完成，但 H.265 转码失败：%v（原视频已保留）", terr), false)
			return
		}
	}
	detail := res.stderrText
	if err != nil && detail != "" {
		detail = extractErrorLines(detail)
		if hint := transcriptHints(detail, settings, ffmpegErr != nil); hint != "" {
			detail += hint
		}
		err = fmt.Errorf("%w: %s", err, detail)
	}
	a.finishTask(id, err)
}

// dlResult 记录一轮 yt-dlp 下载进程的输出结果。
type dlResult struct {
	stderrText  string
	finalPath   string
	duration    float64
	sawProgress bool
}

// execDownload 运行一次 yt-dlp 下载并解析进度、标题与输出路径标记行。
func (a *App) execDownload(ctx context.Context, id int, engine string, args []string, settings Settings) (dlResult, error) {
	res := dlResult{}
	cmd := exec.CommandContext(ctx, engine, args...)
	// 隐藏控制台窗口，并禁用 Python 输出缓冲，保证进度行实时到达。
	hideConsoleWindow(cmd)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return res, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return res, err
	}
	if err := cmd.Start(); err != nil {
		return res, err
	}
	stderrText := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(io.LimitReader(stderr, 1<<20))
		stderrText <- strings.TrimSpace(decodeOutput(b))
	}()
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	// 下载阶段跟踪:预期流数来自格式选择结果("396+251" 表示视频+音频两条流),
	// 逐流更新"下载视频中 / 下载音频中 / 合并中"等提示。
	expectedStreams := 1
	streamsSeen := 0
	lastFingerprint := ""
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "__DOWNTUBE_TITLE__") {
			raw := strings.TrimPrefix(line, "__DOWNTUBE_TITLE__")
			var title string
			if json.Unmarshal([]byte(raw), &title) == nil {
				a.updateTitle(id, title)
			}
		} else if strings.HasPrefix(line, "__DTFMT__") {
			spec := strings.TrimSpace(strings.TrimPrefix(line, "__DTFMT__"))
			expectedStreams = strings.Count(spec, "+") + 1
		} else if strings.HasPrefix(line, "__DTPATH__") {
			raw := strings.TrimPrefix(line, "__DTPATH__")
			var p string
			if json.Unmarshal([]byte(raw), &p) == nil {
				res.finalPath = p
			}
		} else if strings.HasPrefix(line, "__DTDUR__") {
			raw := strings.TrimSpace(strings.TrimPrefix(line, "__DTDUR__"))
			var d float64
			if _, scanErr := fmt.Sscanf(raw, "%f", &d); scanErr == nil {
				res.duration = d
			}
		} else if strings.HasPrefix(line, "__DTPROGRESS__") {
			res.sawProgress = true
			payload := strings.TrimPrefix(line, "__DTPROGRESS__")
			parts := strings.Split(payload, "|")
			if len(parts) == 0 {
				continue
			}
			value := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(parts[0]), "%"))
			var percent int
			_, _ = fmt.Sscanf(value, "%d", &percent)
			if percent < 0 {
				percent = 0
			}
			if percent > 100 {
				percent = 100
			}
			vcodec, acodec := "", ""
			if len(parts) >= 5 {
				vcodec = strings.TrimSpace(parts[3])
				acodec = strings.TrimSpace(parts[4])
			}
			if streamKind(vcodec, acodec) != "" {
				if fp := vcodec + "|" + acodec; fp != lastFingerprint {
					lastFingerprint = fp
					streamsSeen++
				}
			}
			phase := progressPhase(percent, streamsSeen, expectedStreams, vcodec, acodec, settings.DownloadType)
			// 多流下载时把整体进度折算到 0-100,避免"视频 100% 后又回到 0%"的观感
			overall := percent
			if expectedStreams > 1 {
				completed := streamsSeen - 1
				if completed > expectedStreams-1 {
					completed = expectedStreams - 1
				}
				if completed < 0 {
					completed = 0
				}
				overall = (completed*100 + percent) / expectedStreams
			}
			a.updateProgress(id, overall, phase)
		}
	}
	err = cmd.Wait()
	res.stderrText = <-stderrText
	return res, err
}

// fileHasAudio 检查媒体文件是否包含音频流;ffprobe 不可用时返回 true(不做干预)。
func fileHasAudio(ffmpegPath, mediaPath string) bool {
	probe := filepath.Join(filepath.Dir(ffmpegPath), "ffprobe.exe")
	if info, err := os.Stat(probe); err != nil || info.IsDir() {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, probe, "-v", "error", "-select_streams", "a", "-show_entries", "stream=index", "-of", "csv=p=0", mediaPath)
	hideConsoleWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

// moveAsideExisting 把已存在的同名文件改名保留(追加「 (旧文件)」后缀),返回新路径;失败返回空串。
func moveAsideExisting(path string) string {
	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(path, ext)
	for i := 1; i <= 20; i++ {
		suffix := " (旧文件)"
		if i > 1 {
			suffix = fmt.Sprintf(" (旧文件 %d)", i)
		}
		candidate := stem + suffix + ext
		if _, err := os.Stat(candidate); err == nil {
			continue
		}
		if os.Rename(path, candidate) == nil {
			return candidate
		}
		return ""
	}
	return ""
}

// cleanupStreamRemnants 清理与成品同名的残留中间文件
// (yt-dlp 的 f<id> 流文件、.part 断点文件、转码临时文件),避免下载目录堆积零散文件。
// 只处理 3 分钟以前修改过的文件,不会影响其他任务正在写入的内容。
func cleanupStreamRemnants(finalPath string) {
	if finalPath == "" {
		return
	}
	dir := filepath.Dir(finalPath)
	base := strings.TrimSuffix(filepath.Base(finalPath), filepath.Ext(finalPath))
	if base == "" {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-3 * time.Minute)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, base+".") {
			continue
		}
		if !isRemnantName(name[len(base)+1:]) {
			continue
		}
		info, ierr := e.Info()
		if ierr != nil || info.ModTime().After(cutoff) {
			continue
		}
		_ = os.Remove(filepath.Join(dir, name))
	}
}

// isRemnantName 判断去掉成品文件名前缀后的剩余部分是否为下载/转码中间产物。
func isRemnantName(rest string) bool {
	if strings.HasPrefix(rest, "hevc-tmp.") || strings.HasSuffix(rest, ".hevc-bak") {
		return true
	}
	if body := strings.TrimSuffix(rest, ".part"); body != rest {
		return isFormatTagged(body) || isPlainExt(body)
	}
	return isFormatTagged(rest)
}

// isFormatTagged 匹配 yt-dlp 的流文件命名(如 f137.mp4、f251.m4a)。
func isFormatTagged(s string) bool {
	if len(s) < 3 || s[0] != 'f' {
		return false
	}
	dot := strings.IndexByte(s, '.')
	if dot < 2 || dot == len(s)-1 {
		return false
	}
	for _, r := range s[1:dot] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// isPlainExt 匹配单层扩展名(如 mp4、webm,用于 *.part 断点文件)。
func isPlainExt(s string) bool {
	if s == "" || strings.Contains(s, ".") {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

// transcodeLocks 保证同一源文件的转码串行执行(重复添加同一视频时避免共用工作目录互相干扰)。
var transcodeLocks sync.Map

func lockTranscode(srcPath string) func() {
	v, _ := transcodeLocks.LoadOrStore(srcPath, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// transcodeToHEVC 用 ffmpeg 把下载好的视频重新编码为 H.265(HEVC),
// 默认优先显卡加速(自动探测 NVENC / QSV / AMF,不可用或编码失败时回退 CPU);
// 转码按片段执行,暂停/中断后重新开始会复用已完成片段续跑,成功后替换原文件。
func (a *App) transcodeToHEVC(ctx context.Context, id int, srcPath string, duration float64, ffmpegPath string) error {
	if srcPath == "" {
		return errors.New("未获取到下载文件路径")
	}
	unlock := lockTranscode(srcPath)
	defer unlock()
	if _, err := os.Stat(srcPath); err != nil {
		return fmt.Errorf("找不到下载文件：%v", err)
	}
	outPath := strings.TrimSuffix(srcPath, filepath.Ext(srcPath)) + ".mp4"
	tmpPath := strings.TrimSuffix(outPath, filepath.Ext(outPath)) + ".hevc-tmp.mp4"
	defer os.Remove(tmpPath)

	// 编码器:默认优先显卡加速(NVENC / QSV / AMF,自动探测),失败时回退 CPU
	settings := a.GetSettings()
	spec := hevcCPUSpec()
	gpuOK := false
	if settings.HevcEncoder != "cpu" {
		if detected, ok := a.resolveHEVCVideoEncoder(ffmpegPath); ok {
			spec, gpuOK = detected, true
		}
	}
	encodeErr := a.transcodeSegmented(ctx, id, ffmpegPath, srcPath, tmpPath, duration, spec)
	if encodeErr != nil && gpuOK && ctx.Err() == nil {
		// 显卡编码失败(驱动/位深/分辨率限制等):清理临时文件后回退 CPU 再试一次
		_ = os.Remove(tmpPath)
		encodeErr = a.transcodeSegmented(ctx, id, ffmpegPath, srcPath, tmpPath, duration, hevcCPUSpec())
	}
	if encodeErr != nil {
		return encodeErr
	}
	// 用"挪走-替换-清理"三步完成替换,中途失败时可回滚,不会丢失原文件
	backup := outPath + ".hevc-bak"
	_ = os.Remove(backup)
	if _, statErr := os.Stat(outPath); statErr == nil {
		if err := os.Rename(outPath, backup); err != nil {
			return err
		}
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		_ = os.Rename(backup, outPath) // 回滚
		return err
	}
	_ = os.Remove(backup)
	if srcPath != outPath {
		_ = os.Remove(srcPath)
	}
	return nil
}

// hevcSegmentSeconds 转码分片的目标时长(秒)。分片保证中断后可复用已完成片段续跑。
const hevcSegmentSeconds = 30

// hevcSegment 描述一个待编码的分片及它在源视频中的时间范围。
type hevcSegment struct {
	path       string
	start, end float64
}

func (s hevcSegment) dur() float64 {
	d := s.end - s.start
	if d <= 0 {
		d = hevcSegmentSeconds
	}
	return d
}

// hevcPartsDir 返回某个源文件对应的转码工作目录(与源文件同目录,便于续跑时定位)。
func hevcPartsDir(srcPath string) string {
	return strings.TrimSuffix(srcPath, filepath.Ext(srcPath)) + ".hevc-parts"
}

// transcodeSegmented 把源视频切分为若干片段分别编码,全部完成后拼接为 tmpPath。
// 已完成的片段会被直接复用,因此暂停、关闭程序或编码失败后再次执行时会从断点继续。
func (a *App) transcodeSegmented(ctx context.Context, id int, ffmpegPath, srcPath, tmpPath string, duration float64, spec hevcEncoderSpec) error {
	partsDir := hevcPartsDir(srcPath)
	segDir := filepath.Join(partsDir, "seg")
	encDir := filepath.Join(partsDir, "enc")
	csvPath := filepath.Join(partsDir, "segments.csv")
	markerPath := filepath.Join(partsDir, "source.info")

	// 源文件发生变化(重新下载等)时废弃旧的片段缓存
	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	marker := fmt.Sprintf("%d|%d", info.Size(), info.ModTime().UnixNano())
	if data, rerr := os.ReadFile(markerPath); rerr == nil && strings.TrimSpace(string(data)) != marker {
		_ = os.RemoveAll(partsDir)
	}
	if err := os.MkdirAll(encDir, 0o755); err != nil {
		return err
	}
	_ = os.WriteFile(markerPath, []byte(marker), 0o644)

	// 1) 切分:缺失或不完整时执行(切分失败时回退整段转码)
	segments, err := readSegmentList(csvPath, segDir)
	if err != nil {
		_ = os.RemoveAll(segDir)
		_ = os.MkdirAll(segDir, 0o755)
		_ = os.Remove(csvPath)
		a.updateProgress(id, 0, "转码 H.265 中")
		if serr := splitSource(ctx, ffmpegPath, srcPath, segDir, csvPath); serr != nil {
			return a.runTranscode(ctx, id, ffmpegPath, srcPath, tmpPath, duration, spec)
		}
		segments, err = readSegmentList(csvPath, segDir)
		if err != nil {
			return a.runTranscode(ctx, id, ffmpegPath, srcPath, tmpPath, duration, spec)
		}
	}

	total := duration
	if total <= 0 {
		total = segments[len(segments)-1].end
	}

	// 2) 逐片编码:已存在的片段文件直接复用(断点续跑)
	done := 0.0
	for i, seg := range segments {
		if cerr := ctx.Err(); cerr != nil {
			return cerr
		}
		outPath := filepath.Join(encDir, fmt.Sprintf("out-%04d.mp4", i))
		if fi, ferr := os.Stat(outPath); ferr == nil && fi.Size() > 0 {
			done += seg.dur()
			continue
		}
		if eerr := a.encodeSegment(ctx, id, ffmpegPath, seg.path, outPath, done, total, spec); eerr != nil {
			return eerr
		}
		done += seg.dur()
	}

	// 3) 拼接为成品
	a.updateProgress(id, 99, "转码 H.265 中")
	listPath := filepath.Join(partsDir, "concat.txt")
	var list strings.Builder
	for i := range segments {
		list.WriteString(fmt.Sprintf("file 'enc/out-%04d.mp4'\n", i))
	}
	if werr := os.WriteFile(listPath, []byte(list.String()), 0o644); werr != nil {
		return werr
	}
	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-y", "-hide_banner", "-loglevel", "error",
		"-f", "concat", "-safe", "0", "-i", listPath,
		"-c", "copy", "-tag:v", "hvc1", "-movflags", "+faststart",
		tmpPath)
	hideConsoleWindow(cmd)
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("拼接转码片段失败：%s", ffmpegErrText(err, stderrBuf.Bytes()))
	}
	// 拼接结果自检:时长与预期明显不符时视为失败并重置缓存,避免产出被截断的成品
	if total > 0 {
		if got, derr := probeDuration(ffmpegPath, tmpPath); derr == nil && (got < total-2 || got > total+2) {
			_ = os.RemoveAll(partsDir)
			return fmt.Errorf("拼接结果时长异常（%.1f 秒，预期 %.1f 秒），已重置转码缓存", got, total)
		}
	}
	_ = os.RemoveAll(partsDir)
	return nil
}

// splitSource 用流拷贝把源视频按关键帧切分为 mkv 片段,并写出 CSV(名称,起点,终点)。
func splitSource(ctx context.Context, ffmpegPath, srcPath, segDir, csvPath string) error {
	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-y", "-hide_banner", "-loglevel", "error",
		"-i", srcPath,
		"-c", "copy",
		"-f", "segment",
		"-segment_time", fmt.Sprint(hevcSegmentSeconds),
		"-reset_timestamps", "1",
		"-segment_format", "matroska",
		"-segment_list", csvPath,
		"-segment_list_type", "csv",
		filepath.Join(segDir, "seg-%04d.mkv"))
	hideConsoleWindow(cmd)
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("切分源视频失败：%s", ffmpegErrText(err, stderrBuf.Bytes()))
	}
	return nil
}

// readSegmentList 读取切分产出的 CSV,并校验片段文件存在且非空。
func readSegmentList(csvPath, segDir string) ([]hevcSegment, error) {
	data, err := os.ReadFile(csvPath)
	if err != nil {
		return nil, err
	}
	var out []hevcSegment
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		fields := strings.Split(strings.TrimSpace(line), ",")
		if len(fields) < 3 {
			continue
		}
		name := filepath.Base(strings.TrimSpace(fields[0]))
		start, e1 := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64)
		end, e2 := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64)
		if e1 != nil || e2 != nil {
			continue
		}
		p := filepath.Join(segDir, name)
		fi, ferr := os.Stat(p)
		if ferr != nil || fi.Size() == 0 {
			return nil, errors.New("分片文件缺失")
		}
		out = append(out, hevcSegment{path: p, start: start, end: end})
	}
	if len(out) == 0 {
		return nil, errors.New("分片列表为空")
	}
	return out, nil
}

// encodeSegment 编码单个片段,并把片段内的编码进度折算为整体进度上报。
func (a *App) encodeSegment(ctx context.Context, id int, ffmpegPath, segPath, outPath string, doneDur, totalDur float64, spec hevcEncoderSpec) error {
	phase := "转码 H.265 中"
	if spec.phase != "" {
		phase = "转码 H.265 中(" + spec.phase + ")"
	}
	if totalDur > 0 {
		percent := int(doneDur / totalDur * 100)
		if percent > 99 {
			percent = 99
		}
		a.updateProgress(id, percent, phase)
	}
	args := []string{
		"-y", "-hide_banner", "-loglevel", "error",
		"-progress", "pipe:1", "-nostats",
		"-i", segPath,
	}
	args = append(args, spec.videoArgs...)
	// 先写临时文件,编码成功后再原子改名:中断产生的半成品不会被续跑误用
	tmpOut := strings.TrimSuffix(outPath, ".mp4") + ".tmp.mp4"
	defer os.Remove(tmpOut)
	args = append(args, "-tag:v", "hvc1", "-c:a", "copy", tmpOut)
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	hideConsoleWindow(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	if err := cmd.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "out_time_us=") {
			continue
		}
		var us int64
		if _, scanErr := fmt.Sscanf(strings.TrimPrefix(line, "out_time_us="), "%d", &us); scanErr != nil {
			continue
		}
		if totalDur <= 0 {
			continue
		}
		percent := int((doneDur + float64(us)/1e6) / totalDur * 100)
		if percent > 99 {
			percent = 99
		}
		if percent < 0 {
			percent = 0
		}
		a.updateProgress(id, percent, phase)
	}
	if err := cmd.Wait(); err != nil {
		return errors.New(ffmpegErrText(err, stderrBuf.Bytes()))
	}
	if err := os.Rename(tmpOut, outPath); err != nil {
		return err
	}
	return nil
}

// probeDuration 用 ffprobe 读取媒体文件时长(秒)。
func probeDuration(ffmpegPath, mediaPath string) (float64, error) {
	probe := filepath.Join(filepath.Dir(ffmpegPath), "ffprobe.exe")
	if _, err := os.Stat(probe); err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, probe, "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", mediaPath)
	hideConsoleWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
}

// ffmpegErrText 把 ffmpeg 的失败归类为可读的错误文本。
func ffmpegErrText(err error, stderr []byte) string {
	msg := strings.TrimSpace(decodeOutput(stderr))
	if len(msg) > 300 {
		msg = msg[:300] + "…"
	}
	if msg == "" {
		msg = err.Error()
	}
	return msg
}

// runTranscode 执行一次 H.265 编码并在过程中上报进度。
func (a *App) runTranscode(ctx context.Context, id int, ffmpegPath, srcPath, tmpPath string, duration float64, spec hevcEncoderSpec) error {
	phase := "转码 H.265 中"
	if spec.phase != "" {
		phase = "转码 H.265 中(" + spec.phase + ")"
	}
	a.updateProgress(id, 0, phase)
	args := []string{
		"-y", "-hide_banner", "-loglevel", "error",
		"-progress", "pipe:1", "-nostats",
		"-i", srcPath,
	}
	args = append(args, spec.videoArgs...)
	args = append(args,
		"-tag:v", "hvc1", "-movflags", "+faststart",
		"-c:a", "copy",
		tmpPath,
	)
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	hideConsoleWindow(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	if err := cmd.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "out_time_us=") {
			continue
		}
		var us int64
		if _, scanErr := fmt.Sscanf(strings.TrimPrefix(line, "out_time_us="), "%d", &us); scanErr != nil {
			continue
		}
		if duration <= 0 {
			continue
		}
		percent := int(float64(us) / 1e6 / duration * 100)
		if percent > 99 {
			percent = 99
		}
		if percent < 0 {
			percent = 0
		}
		a.updateProgress(id, percent, phase)
	}
	if err := cmd.Wait(); err != nil {
		msg := strings.TrimSpace(decodeOutput(stderrBuf.Bytes()))
		if len(msg) > 300 {
			msg = msg[:300] + "…"
		}
		if msg == "" {
			msg = err.Error()
		}
		return errors.New(msg)
	}
	return nil
}

// hevcEncoderSpec 描述一套 H.265 编码参数。
type hevcEncoderSpec struct {
	id        string   // nvenc / qsv / amf / cpu
	label     string   // 完整名称(设置页展示)
	phase     string   // 任务阶段后缀
	videoArgs []string // ffmpeg 视频编码参数
}

func hevcCPUSpec() hevcEncoderSpec {
	return hevcEncoderSpec{
		id: "cpu", label: "CPU(libx265)", phase: "CPU",
		videoArgs: []string{"-c:v", "libx265", "-preset", "medium", "-crf", "26"},
	}
}

// hevcGPUCandidates 显卡编码器候选(按优先级探测);参数只有在该机器上探测通过才会使用。
var hevcGPUCandidates = []hevcEncoderSpec{
	{id: "nvenc", label: "NVIDIA 显卡加速(NVENC)", phase: "显卡", videoArgs: []string{"-c:v", "hevc_nvenc", "-preset", "p5", "-tune", "hq", "-rc", "vbr", "-cq", "26", "-b:v", "0"}},
	{id: "qsv", label: "Intel 核显加速(QSV)", phase: "显卡", videoArgs: []string{"-c:v", "hevc_qsv", "-preset", "medium", "-global_quality", "26"}},
	{id: "amf", label: "AMD 显卡加速(AMF)", phase: "显卡", videoArgs: []string{"-c:v", "hevc_amf", "-quality", "quality", "-rc", "cqp", "-qp_i", "24", "-qp_p", "26"}},
}

// probeHEVCEncoder 用一小段合成画面实测编码器在当前机器上是否可用
// (驱动缺失、显卡不支持等情况会失败),避免选到名义存在但实际不可用的编码器。
func probeHEVCEncoder(ffmpegPath string, spec hevcEncoderSpec) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	args := []string{
		"-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "color=c=black:s=640x360:r=30:d=0.5",
	}
	args = append(args, spec.videoArgs...)
	args = append(args, "-an", "-f", "null", "-")
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	hideConsoleWindow(cmd)
	return cmd.Run() == nil
}

// resolveHEVCVideoEncoder 探测可用的显卡编码器并缓存结果(FFmpeg 路径变化时重新探测)。
// 返回是否找到显卡编码器;未找到时返回 CPU 方案。
func (a *App) resolveHEVCVideoEncoder(ffmpegPath string) (hevcEncoderSpec, bool) {
	a.hevcMu.Lock()
	defer a.hevcMu.Unlock()
	if a.hevcProbed && a.hevcProbedFor == ffmpegPath {
		if a.hevcGPU != nil {
			return *a.hevcGPU, true
		}
		return hevcCPUSpec(), false
	}
	var found *hevcEncoderSpec
	for i := range hevcGPUCandidates {
		if probeHEVCEncoder(ffmpegPath, hevcGPUCandidates[i]) {
			spec := hevcGPUCandidates[i]
			found = &spec
			break
		}
	}
	a.hevcProbed = true
	a.hevcProbedFor = ffmpegPath
	a.hevcGPU = found
	if found != nil {
		return *found, true
	}
	return hevcCPUSpec(), false
}

// GetHEVCEncoderStatus 返回当前设备可用的 H.265 显卡编码器(探测结果缓存,供设置页展示)。
func (a *App) GetHEVCEncoderStatus() map[string]any {
	ffmpegPath, err := a.findFFmpeg()
	if err != nil {
		return map[string]any{"ok": false, "gpu": false, "message": "未检测到 FFmpeg，无法使用 H.265 转码"}
	}
	spec, ok := a.resolveHEVCVideoEncoder(ffmpegPath)
	if ok {
		return map[string]any{"ok": true, "gpu": true, "encoder": spec.id, "label": spec.label}
	}
	return map[string]any{"ok": true, "gpu": false, "encoder": "cpu", "label": "CPU(libx265)", "message": "未检测到可用的显卡编码器，将使用 CPU 转码"}
}

// decodeOutput 把 yt-dlp 的输出解码为 UTF-8:Windows 下 Python 的错误信息常为 GBK 编码。
func decodeOutput(raw []byte) string {
	if utf8.Valid(raw) {
		return string(raw)
	}
	if decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(raw); err == nil {
		return string(decoded)
	}
	return string(raw)
}

// streamKind 依据编解码信息判断当前进度属于哪一类流。
func streamKind(vcodec, acodec string) string {
	hasV := vcodec != "" && vcodec != "none" && vcodec != "NA"
	hasA := acodec != "" && acodec != "none" && acodec != "NA"
	switch {
	case hasV && hasA:
		return "av"
	case hasV:
		return "video"
	case hasA:
		return "audio"
	default:
		return ""
	}
}

// progressPhase 依据流进度推导正在进行的阶段文案。
func progressPhase(percent, streamsSeen, expectedStreams int, vcodec, acodec, downloadType string) string {
	if percent >= 100 && expectedStreams > 1 && streamsSeen >= expectedStreams {
		return "合并中"
	}
	if percent >= 100 && strings.HasPrefix(downloadType, "仅音频") {
		return "转换音频中"
	}
	switch streamKind(vcodec, acodec) {
	case "video":
		return "下载视频中"
	case "audio":
		return "下载音频中"
	case "av":
		return "下载中"
	}
	return ""
}

// extractErrorLines 从 yt-dlp 的 stderr 里挑出 ERROR 行,避免把整段输出塞进界面。
func extractErrorLines(raw string) string {
	lines := strings.Split(raw, "\n")
	var picked []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "ERROR:") {
			if len(l) > 600 {
				l = l[:600] + "…"
			}
			picked = append(picked, l)
			if len(picked) >= 3 {
				break
			}
		}
	}
	if len(picked) == 0 {
		if len(raw) > 800 {
			raw = raw[:800] + "…"
		}
		return strings.TrimSpace(raw)
	}
	return strings.Join(picked, "\n")
}

// transcriptHints 针对常见失败原因追加可操作的排查建议.
func transcriptHints(detail string, settings Settings, ffmpegMissing bool) string {
	var hints []string
	add := func(s string) {
		for _, h := range hints {
			if h == s {
				return
			}
		}
		hints = append(hints, s)
	}
	has := func(sub string) bool { return strings.Contains(detail, sub) }

	if ffmpegMissing && has("Requested format is not available") {
		add("该视频只提供分离的音视频流，需要安装 FFmpeg 才能下载")
	}
	if has("Could not copy Chrome cookie database") {
		add("浏览器运行时会锁定 Cookies 数据库，请先完全退出浏览器再重试，或改用文件导入 Cookies")
	}
	if has("Sign in to confirm") || has("not a bot") || has("Sign in required") {
		add("YouTube 要求登录验证，请在设置 → Cookies 中提供有效 Cookies")
	}
	if has("HTTP Error 403") {
		add("请求被拒绝(403)，通常是 Cookies 失效或当前代理 IP 被限制，尝试更新 Cookies 或更换代理节点")
	}
	if has("HTTP Error 429") {
		add("请求过于频繁(429)，请稍后重试或降低并发下载数")
	}
	if settings.UseBrowserCookie && settings.CookieSource == "file" && (has("HTTP Error 403") || has("Sign in to confirm") || has("not a bot")) {
		add("当前使用的是本地 Cookies 文件，若持续失败可重新导出，或改用「从浏览器读取」方式")
	}
	if has("does not look like a Netscape format") || has("cookies file uses an unsupported format") || has("unsupported file format") {
		add("Cookies 文件格式不正确，需要 Netscape 格式的 cookies.txt(用浏览器扩展导出，而不是 JSON)")
	}
	if has("Unable to open cookie file") || has("unable to read cookie file") || (has("cookie") && has("No such file")) {
		add("Cookies 文件不存在或无法读取，请在设置中重新选择")
	}
	if has("Video unavailable") {
		add("视频不可用，可能已被删除或设为私享")
	}
	if has("members-only") || has("available to this channel's members") {
		add("会员专属内容，需要对应账号的 Cookies")
	}
	if has("Private video") {
		add("这是私享视频，需要拥有访问权限的账号 Cookies")
	}
	if has("unable to download video data") || has("Unable to download webpage") {
		add("与 YouTube 的连接中断，网络或代理不稳定；任务已开启自动重试")
	}
	if has("nsig") || has("Signature extraction failed") || has("Some formats may be missing") {
		add("YouTube 客户端校验异常，请更新 yt-dlp 或提供 Cookies")
	}
	if has("Connection reset") || has("timed out") || has("Temporary failure") || has("EOF occurred") {
		add("网络连接被中断，请检查代理稳定性；任务已开启自动重试")
	}
	if has("This video is not available in your country") {
		add("当前地区不可观看，请更换代理节点")
	}
	if len(hints) == 0 {
		return ""
	}
	return "(" + strings.Join(hints, ";") + ")"
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
	a.emit(snapshot)
}

func (a *App) updateProgress(id int, progress int, phase string) {
	a.mu.Lock()
	changed := false
	for i := range a.store.Tasks {
		t := &a.store.Tasks[i]
		if t.ID != id {
			continue
		}
		if t.Progress != progress {
			t.Progress = progress
			changed = true
		}
		if phase != "" && t.Phase != phase {
			t.Phase = phase
			changed = true
		}
		break
	}
	// 进度超过此前的最高水平,说明重试后下载确实恢复推进:
	// 连续失败计数归零,之后再失败会从「重试中(1/N)」重新开始
	if a.bestProgress == nil {
		a.bestProgress = make(map[int]int)
	}
	if progress > a.bestProgress[id] {
		a.bestProgress[id] = progress
		if a.retryAttempts[id] > 0 {
			delete(a.retryAttempts, id)
		}
	}
	snapshot := a.snapshotLocked()
	a.mu.Unlock()
	if changed {
		a.emit(snapshot)
	}
}

func (a *App) finishTask(id int, err error) {
	a.finishTaskEx(id, err, true)
}

// finishTaskEx 结束任务;allowRetry 为 false 时不进入自动重试
// (如转码失败:原文件已保留,重试会重复下载)。
func (a *App) finishTaskEx(id int, err error, allowRetry bool) {
	a.mu.Lock()
	delete(a.workers, id)
	openFolder := false
	downloadDir := a.store.Settings.DownloadDir
	retryID := 0
	var retryDelay time.Duration
	for i := range a.store.Tasks {
		t := &a.store.Tasks[i]
		if t.ID != id {
			continue
		}
		if err == nil {
			t.Status = StatusComplete
			t.Progress = 100
			t.Phase = ""
			t.Error = ""
			delete(a.retryAttempts, id)
			delete(a.bestProgress, id)
			openFolder = a.store.Settings.OpenOnComplete
		} else {
			maxRetry := a.store.Settings.AutoRetry
			if !allowRetry {
				maxRetry = 0
			}
			if maxRetry < 0 {
				maxRetry = 0
			}
			attempts := a.retryAttempts[id]
			if attempts < maxRetry {
				// 保留在队列里,延迟后自动重试(退避:8s、18s、28s…)
				a.retryAttempts[id] = attempts + 1
				t.Status = StatusQueued
				t.Progress = 0
				t.Phase = fmt.Sprintf("重试中(%d/%d)", attempts+1, maxRetry)
				t.Error = err.Error()
				retryID = id
				retryDelay = time.Duration(8+attempts*10) * time.Second
			} else {
				t.Status = StatusFailed
				t.Phase = ""
				t.Error = err.Error()
				delete(a.retryAttempts, id)
				delete(a.bestProgress, id)
			}
		}
		break
	}
	// 没有正在运行或等待中的任务时,自动调度收尾
	if a.autoRun && len(a.workers) == 0 && !a.hasRunnableLocked() {
		a.autoRun = false
	}
	engine, engineErr := findYTDLP()
	launches := a.fillSlotsLocked(engine, engineErr)
	_ = a.saveLocked()
	snapshot := a.snapshotLocked()
	a.mu.Unlock()
	if openFolder {
		_ = exec.Command("explorer.exe", downloadDir).Start()
	}
	if retryID != 0 {
		a.scheduleRetry(retryID, retryDelay)
	}
	a.dispatch(launches)
	a.emit(snapshot)
}

// hasRunnableLocked 判断是否还有等待调度的任务(含"等待重试")。调用方须持有锁。
func (a *App) hasRunnableLocked() bool {
	for _, t := range a.store.Tasks {
		if t.Status == StatusQueued {
			return true
		}
	}
	return false
}

func (a *App) SaveSettings(settings Settings) Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	if settings.Quality == "" {
		settings.Quality = "1080p"
	}
	if settings.DownloadType == "" {
		settings.DownloadType = "视频（带音频）"
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
	if settings.MaxConcurrent <= 0 {
		settings.MaxConcurrent = 3
	}
	if settings.MaxConcurrent > 6 {
		settings.MaxConcurrent = 6
	}
	if settings.AutoRetry < 0 {
		settings.AutoRetry = 0
	}
	if settings.AutoRetry > 5 {
		settings.AutoRetry = 5
	}
	if settings.VideoCodec != "hevc" {
		settings.VideoCodec = "auto"
	}
	if settings.HevcEncoder != "cpu" {
		settings.HevcEncoder = "auto"
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

// SelectCookieFile 让用户选择 cookies.txt,并做格式校验与自动修复。
func (a *App) SelectCookieFile() map[string]any {
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
		return map[string]any{"canceled": true}
	}
	chosen = filepath.Clean(chosen)
	note, verifyErr := repairCookieFile(chosen)
	if verifyErr != nil {
		return map[string]any{"ok": false, "message": verifyErr.Error()}
	}
	return map[string]any{"ok": true, "path": chosen, "note": note}
}

// repairCookieFile 校验 cookies 文件,并自动修复两个高频问题:
// UTF-8 BOM 和缺失的 Netscape 文件头(两者都会让 yt-dlp 直接报格式错误)。
func repairCookieFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败：%v", err)
	}
	if len(data) == 0 {
		return "", errors.New("文件是空的，请重新导出 cookies")
	}
	if len(data) > 8<<20 {
		return "", errors.New("文件过大，不像是 cookies.txt")
	}
	var notes []string
	changed := false
	if bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
		data = data[3:]
		changed = true
		notes = append(notes, "已移除文件开头的 BOM")
	}
	text := string(data)
	firstLine := text
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		firstLine = text[:i]
	}
	firstLine = strings.TrimSpace(firstLine)
	hasHeader := strings.HasPrefix(firstLine, "# Netscape HTTP Cookie File") || strings.HasPrefix(firstLine, "# HTTP Cookie File")
	if !hasHeader {
		if !looksLikeCookieLines(text) {
			return "", errors.New("文件不像 Netscape 格式的 cookies.txt，请用浏览器扩展重新导出(不要选 JSON 格式)")
		}
		data = append([]byte("# Netscape HTTP Cookie File\n"), data...)
		changed = true
		notes = append(notes, "已补全 Netscape 文件头")
	}
	if !strings.Contains(string(data), "youtube.com") {
		notes = append(notes, "里面没有 youtube.com 的 cookies，可能无法用于登录验证")
	}
	if changed {
		if err := os.WriteFile(filePath, data, 0o600); err != nil {
			return "", fmt.Errorf("修复后写回文件失败：%v", err)
		}
	}
	return strings.Join(notes, ";"), nil
}

// looksLikeCookieLines 粗判文本是否包含 Netscape cookie 数据行。
func looksLikeCookieLines(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "#HttpOnly_") {
			continue
		}
		if strings.Count(line, "\t") >= 5 {
			return true
		}
	}
	return false
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
		return map[string]any{"ok": false, "message": "FFmpeg 正在安装中，请稍候"}
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
		return map[string]any{"ok": false, "message": "创建安装目录失败：" + err.Error()}
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
		return map[string]any{"ok": false, "message": "下载失败：" + downloadErr.Error() + "。如果网络受限，请在设置中开启代理后重试"}
	}

	a.emitFFmpegProgress("extract", 0, 0, "正在解压安装(约需十余秒)…")
	if err := extractFFmpegArchive(tmpZip, binDir, nil); err != nil {
		return map[string]any{"ok": false, "message": "解压失败：" + err.Error()}
	}

	ffmpegPath := filepath.Join(binDir, "ffmpeg.exe")
	version, verr := ffmpegVersion(ffmpegPath)
	if verr != nil {
		return map[string]any{"ok": false, "message": "安装校验未通过：" + verr.Error()}
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
		return map[string]any{"ok": false, "message": "FFmpeg 正在安装中，请稍候"}
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
		return map[string]any{"ok": false, "message": "解压失败：" + err.Error()}
	}

	ffmpegPath := filepath.Join(binDir, "ffmpeg.exe")
	version, verr := ffmpegVersion(ffmpegPath)
	if verr != nil {
		return map[string]any{"ok": false, "message": "安装校验未通过：" + verr.Error()}
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
		return errors.New("不支持的压缩格式，请选择 .7z 或 .zip 安装包")
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
		return fmt.Errorf("无法读取 7z 压缩包：%w", err)
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
	if saved.Version < 2 {
		// v1 → v2:补充并发下载与失败重试设置
		if saved.Settings.MaxConcurrent <= 0 {
			saved.Settings.MaxConcurrent = 3
		}
		if saved.Settings.AutoRetry <= 0 {
			saved.Settings.AutoRetry = 2
		}
		saved.Version = 2
	}
	// 下载内容的文案调整:"视频 + 音频" 更名为 "视频（带音频）"
	if saved.Settings.DownloadType == "视频 + 音频" {
		saved.Settings.DownloadType = "视频（带音频）"
	}
	if saved.Tasks == nil {
		saved.Tasks = []Task{}
	}
	for i := range saved.Tasks {
		if saved.Tasks[i].Status == StatusRunning {
			saved.Tasks[i].Status = StatusQueued
		}
		// 阶段与重试提示是运行期的瞬时状态,重启后不再展示
		saved.Tasks[i].Phase = ""
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
