package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// 百宝箱里的 FFmpeg 工具箱:视频音频合并、视频分离。
//
// 设计约束(来自需求,别改):**输出目录必须由用户每次弹窗指定**。
// 所以选择目录的逻辑放在后端,前端连「记住上次路径」的字段都没有 ——
// 从结构上保证它不可能被自动化掉。

// ToolboxStartResult 合并/分离任务的启动结果。
type ToolboxStartResult struct {
	OK       bool   `json:"ok"`
	Message  string `json:"message,omitempty"`
	Canceled bool   `json:"canceled,omitempty"`
}

// ToolboxProgress 处理进度事件(对应前端 toolbox:progress)。
type ToolboxProgress struct {
	Phase   string  `json:"phase"`   // merge / split
	Percent float64 `json:"percent"` // 0~100
	Message string  `json:"message"` // 当前阶段文案
}

// ToolboxFinished 处理完成事件(对应前端 toolbox:finished)。
type ToolboxFinished struct {
	OK       bool   `json:"ok"`
	Canceled bool   `json:"canceled,omitempty"` // 用户主动取消,不是失败
	Message  string `json:"message"`
	Output   string `json:"output,omitempty"` // 成功时的输出目录
}

// toolboxRunner 持有「当前是否有任务在跑」与取消钩子。
// 合并与分离都是长任务,必须可取消;同一时刻只允许一个,避免两个 ffmpeg 抢同一份输出。
type toolboxRunner struct {
	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
}

// begin 尝试占用执行权;已被占用时返回 ok=false(调用方应提示「已有任务在进行」)。
func (t *toolboxRunner) begin() (context.Context, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.running {
		return nil, false
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.running, t.cancel = true, cancel
	return ctx, true
}

// end 释放执行权并取消尚未结束的进程。
func (t *toolboxRunner) end() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}
	t.running = false
}

func (t *toolboxRunner) cancelTask() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.running || t.cancel == nil {
		return false
	}
	t.cancel()
	return true
}

func (t *toolboxRunner) busy() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

// PickToolboxVideo 选择要处理的视频文件。
func (a *App) PickToolboxVideo() FilePickResult {
	chosen, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择视频文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "视频文件 (*.mp4;*.mkv;*.mov;*.avi;*.flv;*.ts;*.webm;*.wmv)", Pattern: "*.mp4;*.mkv;*.mov;*.avi;*.flv;*.ts;*.webm;*.wmv"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil || chosen == "" {
		return FilePickResult{Canceled: true}
	}
	chosen = filepath.Clean(chosen)
	codec, derr := probeVideoCodec(a.ffmpegPathForToolbox(), chosen)
	if derr != nil {
		return FilePickResult{OK: false, Message: "无法读取该视频:" + derr.Error()}
	}
	// ffprobe 对「有文件但没有视频流」不报错,只会返回空编码器名。
	// 这类文件(比如误选了一个纯音频)要在这里就拦下来,
	// 否则会一路跑到 ffmpeg 才失败,报错还看不懂。
	if codec == "" {
		return FilePickResult{OK: false, Message: "这个文件里没有视频画面,请换一个"}
	}
	return FilePickResult{OK: true, Path: chosen, Version: codec}
}

// PickToolboxAudio 选择用于合并的音频文件。
func (a *App) PickToolboxAudio() FilePickResult {
	chosen, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择音频文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "音频文件 (*.m4a;*.mp3;*.aac;*.wav;*.flac;*.ogg;*.opus;*.wma)", Pattern: "*.m4a;*.mp3;*.aac;*.wav;*.flac;*.ogg;*.opus;*.wma"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil || chosen == "" {
		return FilePickResult{Canceled: true}
	}
	chosen = filepath.Clean(chosen)
	codec, derr := probeAudioCodec(a.ffmpegPathForToolbox(), chosen)
	if derr != nil {
		return FilePickResult{OK: false, Message: "无法读取该音频:" + derr.Error()}
	}
	if codec == "" {
		return FilePickResult{OK: false, Message: "这个文件里没有声音,请换一个"}
	}
	return FilePickResult{OK: true, Path: chosen, Version: codec}
}

// MergeVideoAudio 把外部音频合并进视频,替换/补上原视频的音轨。
//
// **只做流复制,不重新编码** —— 合并的职责是把两路拼起来,不是压缩。
// 想让文件变小请用「视频压缩」,那里有格式和压缩率的选项。
// 流复制的好处:瞬间完成,画质零损失。想压体积就别指望这一步顺便做掉。
func (a *App) MergeVideoAudio(videoPath, audioPath string) ToolboxStartResult {
	ffmpegPath := a.ffmpegPathForToolbox()
	if ffmpegPath == "" {
		return ToolboxStartResult{OK: false, Message: "未检测到 FFmpeg,请先在「设置 → 视频设置」安装"}
	}
	if videoPath == "" || audioPath == "" {
		return ToolboxStartResult{OK: false, Message: "请先选择视频文件与音频文件"}
	}
	if _, err := os.Stat(videoPath); err != nil {
		return ToolboxStartResult{OK: false, Message: "视频文件不存在或无法读取"}
	}
	if _, err := os.Stat(audioPath); err != nil {
		return ToolboxStartResult{OK: false, Message: "音频文件不存在或无法读取"}
	}
	// 输出目录:每次弹窗询问,不记、不默认
	outDir, canceled := a.askToolboxOutputDir()
	if canceled {
		return ToolboxStartResult{Canceled: true}
	}

	videoCodec, _ := probeVideoCodec(ffmpegPath, videoPath)
	ext := mergeOutputExt(videoCodec)
	outPath := uniqueFilePath(outDir, mergeOutputName(videoPath), ext)

	ctx, ok := a.toolbox.begin()
	if !ok {
		return ToolboxStartResult{OK: false, Message: "已有一个任务正在进行,请先等它完成或取消"}
	}
	go a.runToolboxFFmpeg(ctx, "merge", "正在合并视频与音频", []string{videoPath, audioPath}, outPath,
		buildMergeArgs(videoPath, audioPath, outPath))
	return ToolboxStartResult{OK: true}
}

// SplitVideoStream 把视频拆成独立的视频流文件与音频流文件。
func (a *App) SplitVideoStream(videoPath string) ToolboxStartResult {
	ffmpegPath := a.ffmpegPathForToolbox()
	if ffmpegPath == "" {
		return ToolboxStartResult{OK: false, Message: "未检测到 FFmpeg,请先在「设置 → 视频设置」安装"}
	}
	if videoPath == "" {
		return ToolboxStartResult{OK: false, Message: "请先选择视频文件"}
	}
	if _, err := os.Stat(videoPath); err != nil {
		return ToolboxStartResult{OK: false, Message: "视频文件不存在或无法读取"}
	}
	videoCodec, _ := probeVideoCodec(ffmpegPath, videoPath)
	audioCodec, _ := probeAudioCodec(ffmpegPath, videoPath)
	audioCodec = firstAudioCodec(audioCodec)

	// 没有音轨就没必要拆音频,直接告诉用户,别产出一个空文件
	if audioCodec == "" {
		return ToolboxStartResult{OK: false, Message: "这个视频没有音轨,无需分离"}
	}

	outDir, canceled := a.askToolboxOutputDir()
	if canceled {
		return ToolboxStartResult{Canceled: true}
	}
	videoExt := splitVideoExt(videoCodec)
	audioExt := splitAudioExt(audioCodec)
	videoOut := uniqueFilePath(outDir, splitOutputName(videoPath, "视频"), videoExt)
	audioOut := uniqueFilePath(outDir, splitOutputName(videoPath, "音频"), audioExt)

	ctx, ok := a.toolbox.begin()
	if !ok {
		return ToolboxStartResult{OK: false, Message: "已有一个任务正在进行,请先等它完成或取消"}
	}
	go a.runSplitFFmpeg(ctx, ffmpegPath, videoPath, videoOut, audioOut, outDir)
	return ToolboxStartResult{OK: true}
}

// CancelToolboxTask 取消正在进行的合并/分离。
func (a *App) CancelToolboxTask() bool {
	return a.toolbox.cancelTask()
}

// ToolboxBusy 供前端查询当前是否有任务在跑。
func (a *App) ToolboxBusy() bool {
	return a.toolbox.busy()
}

// askToolboxOutputDir 每次弹窗询问输出目录。**刻意不读设置、不记上次路径。**
func (a *App) askToolboxOutputDir() (string, bool) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "选择输出文件夹(每次都会询问)"})
	if err != nil || dir == "" {
		return "", true
	}
	return dir, false
}

func (a *App) ffmpegPathForToolbox() string {
	path, err := a.findFFmpeg()
	if err != nil {
		return ""
	}
	return path
}

// runToolboxFFmpeg 跑一次两输入(视频 + 音频)的 ffmpeg,上报进度并在结束时发事件。
func (a *App) runToolboxFFmpeg(ctx context.Context, phase, phaseText string, inputs []string, outPath string, args []string) {
	defer a.toolbox.end()
	a.emitToolboxProgress(phase, 0, phaseText+"…")
	ffmpegPath := a.ffmpegPathForToolbox()
	if ffmpegPath == "" {
		a.emitToolboxFinished(false, "FFmpeg 已不可用", "")
		return
	}
	total := a.toolboxTotalDuration(ffmpegPath, inputs[0], inputs[1])
	msg, err := a.execToolboxFFmpeg(ctx, ffmpegPath, args, total, func(percent float64) {
		a.emitToolboxProgress(phase, percent, phaseText+fmt.Sprintf(" %d%%", int(percent)))
	})
	if err != nil {
		if ctx.Err() != nil {
			a.emitToolboxFinishedCanceled()
			return
		}
		a.emitToolboxFinished(false, "处理失败:"+err.Error(), "")
		return
	}
	a.emitToolboxProgress(phase, 100, "完成")
	a.emitToolboxFinished(true, msg, filepath.Dir(outPath))
}

// runSplitFFmpeg 跑两次 ffmpeg:先抽视频流,再抽音频流,各自上报进度。
// 两路都走流复制,不重新编码 —— 分离是拆包,不是压缩。
func (a *App) runSplitFFmpeg(ctx context.Context, ffmpegPath, videoPath, videoOut, audioOut, outDir string) {
	defer a.toolbox.end()
	total, _ := probeDuration(ffmpegPath, videoPath)

	// 第一遍:视频流
	a.emitToolboxProgress("split", 0, "正在分离视频流…")
	videoArgs := []string{
		"-y", "-hide_banner", "-loglevel", "error", "-progress", "pipe:1", "-nostats",
		"-i", videoPath, "-map", "0:v:0", "-an", "-c:v", "copy", videoOut,
	}
	if _, err := a.execToolboxFFmpeg(ctx, ffmpegPath, videoArgs, total, func(percent float64) {
		// 视频流占总进度的一半
		a.emitToolboxProgress("split", percent/2, fmt.Sprintf("正在分离视频流 %d%%", int(percent/2)))
	}); err != nil {
		a.finishSplitError(ctx, err)
		return
	}

	// 第二遍:音频流(音频占比小,固定按剩余 50% 里的一小段走)
	a.emitToolboxProgress("split", 55, "正在分离音频流…")
	audioArgs := []string{
		"-y", "-hide_banner", "-loglevel", "error", "-progress", "pipe:1", "-nostats",
		"-i", videoPath, "-map", "0:a:0", "-vn", "-c:a", "copy", audioOut,
	}
	if _, err := a.execToolboxFFmpeg(ctx, ffmpegPath, audioArgs, total, func(percent float64) {
		a.emitToolboxProgress("split", 55+percent*0.45, fmt.Sprintf("正在分离音频流 %d%%", int(55+percent*0.45)))
	}); err != nil {
		a.finishSplitError(ctx, err)
		return
	}

	a.emitToolboxProgress("split", 100, "完成")
	a.emitToolboxFinished(true, "已分离为视频流与音频流", outDir)
}

func (a *App) finishSplitError(ctx context.Context, err error) {
	if ctx.Err() != nil {
		a.emitToolboxFinishedCanceled()
		return
	}
	a.emitToolboxFinished(false, "处理失败:"+err.Error(), "")
}

// execToolboxFFmpeg 执行一次 ffmpeg,按 onPercent 回调上报进度。
func (a *App) execToolboxFFmpeg(ctx context.Context, ffmpegPath string, args []string, total float64, onPercent func(float64)) (string, error) {
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	hideConsoleWindow(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf
	if err := cmd.Start(); err != nil {
		return "", err
	}
	// 取消时要终结整棵进程树:ffmpeg 可能再拉子进程,只杀父进程会留下孤儿
	stopWatch := watchCancel(ctx, cmd)
	defer stopWatch()

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
		if total <= 0 || us <= 0 {
			continue
		}
		percent := float64(us) / 1e6 / total * 100
		if percent > 99 {
			percent = 99
		}
		onPercent(percent)
	}
	if err := cmd.Wait(); err != nil {
		return "", errors.New(ffmpegErrText(err, stderrBuf.Bytes()))
	}
	return "完成", nil
}

// outputTailArgs 命令行末尾:mp4 补上 +faststart,然后**必须跟着输出文件路径**。
//
// ⚠️ 这个 helper 曾经拆成两半(一个只返回 "-movflags +faststart",输出路径在调用点拼),
// 结果改的时候把输出路径漏掉了 —— ffmpeg 收不到输出文件,直接报
//
//	At least one output file must be specified
//
// mp4 是压缩的默认格式,所以合并与压缩两个功能同时全挂。
// **把输出路径和它前面的参数绑在一起返回,让调用点没法漏。**
//
// +faststart 是 mp4 封装器的选项,把索引挪到文件头,便于边下边播;
// mkv / webm 没有这个选项,加了虽然不报错(实测 ffmpeg 静默忽略),但属于无意义参数。
func outputTailArgs(outPath string) []string {
	if strings.EqualFold(filepath.Ext(outPath), ".mp4") {
		return []string{"-movflags", "+faststart", outPath}
	}
	return []string{outPath}
}

// buildMergeArgs 拼合并的完整命令行(末尾含输出文件)。
func buildMergeArgs(videoPath, audioPath, outPath string) []string {
	a := []string{"-y", "-hide_banner", "-loglevel", "error", "-progress", "pipe:1", "-nostats",
		"-i", videoPath, "-i", audioPath, "-map", "0:v:0", "-map", "1:a:0", "-c", "copy"}
	return append(a, outputTailArgs(outPath)...)
}

// buildCompressArgs 拼压缩的完整命令行(末尾含输出文件)。
func buildCompressArgs(videoPath, outPath string, videoArgs, audioArgs []string) []string {
	a := []string{"-y", "-hide_banner", "-loglevel", "error", "-progress", "pipe:1", "-nostats", "-i", videoPath}
	a = append(a, videoArgs...)
	a = append(a, audioArgs...)
	return append(a, outputTailArgs(outPath)...)
}

// toolboxTotalDuration 合并任务的总时长按视频与音频里较长的那个算。
func (a *App) toolboxTotalDuration(ffmpegPath, videoPath, audioPath string) float64 {
	// 压缩只有一个输入(第二路传空串),空路径要跳过 ——
	// 否则每次压缩都会白跑一次 ffprobe("") 拿到 Invalid argument
	var v, a2 float64
	if videoPath != "" {
		v, _ = probeDuration(ffmpegPath, videoPath)
	}
	if audioPath != "" {
		a2, _ = probeDuration(ffmpegPath, audioPath)
	}
	return math.Max(v, a2)
}

func (a *App) emitToolboxProgress(phase string, percent float64, message string) {
	if a.ctx == nil || a.disableEvents {
		return
	}
	runtime.EventsEmit(a.ctx, "toolbox:progress", ToolboxProgress{Phase: phase, Percent: percent, Message: message})
}

// emitToolboxFinishedCanceled 取消不是失败:单独一个标记,
// 前端据此把进度区收掉,而不是停在中途显示一个「已取消」的红条。
func (a *App) emitToolboxFinishedCanceled() {
	if a.ctx == nil || a.disableEvents {
		return
	}
	runtime.EventsEmit(a.ctx, "toolbox:finished", ToolboxFinished{OK: false, Canceled: true, Message: "已取消"})
}

func (a *App) emitToolboxFinished(ok bool, message, output string) {
	if a.ctx == nil || a.disableEvents {
		return
	}
	runtime.EventsEmit(a.ctx, "toolbox:finished", ToolboxFinished{OK: ok, Message: message, Output: output})
}

// probeVideoCodec / probeAudioCodec 用 ffprobe 读编码器名,用来决定输出容器。

func probeVideoCodec(ffmpegPath, mediaPath string) (string, error) {
	return probeCodec(ffmpegPath, mediaPath, "v:0")
}

func probeAudioCodec(ffmpegPath, mediaPath string) (string, error) {
	return probeCodec(ffmpegPath, mediaPath, "a:0")
}

func probeCodec(ffmpegPath, mediaPath, stream string) (string, error) {
	if ffmpegPath == "" {
		return "", errors.New("FFmpeg 不可用")
	}
	ffprobe := strings.TrimSuffix(ffmpegPath, "ffmpeg.exe") + "ffprobe.exe"
	if _, err := os.Stat(ffprobe); err != nil {
		// ffprobe 与 ffmpeg 同目录;找不到时退回 PATH
		if p, lerr := exec.LookPath("ffprobe"); lerr == nil {
			ffprobe = p
		} else {
			return "", errors.New("找不到 ffprobe")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, ffprobe,
		"-v", "error", "-select_streams", stream, "-show_entries", "stream=codec_name", "-of", "csv=p=0", mediaPath).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func firstAudioCodec(s string) string {
	// probeCodec 只选 a:0,单行即可;防御性地取第一行
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// mergeOutputExt / splitVideoExt / splitAudioExt 按编码器挑输出容器。
// 走流复制时必须让容器装得下原编码器,否则 ffmpeg 直接报错。
func mergeOutputExt(videoCodec string) string {
	switch videoCodec {
	case "h264", "hevc", "av1", "vp9", "vp8":
		return ".mp4"
	default:
		return ".mkv"
	}
}

func splitVideoExt(videoCodec string) string {
	switch videoCodec {
	case "h264", "hevc", "av1", "vp9", "vp8", "mpeg4":
		return ".mp4"
	default:
		return ".mkv"
	}
}

// splitAudioExt 按音频编码器挑容器。**走流复制,容器装不下就必然失败。**
//
// mp3 单独一档:`.m4a` 会强制用 ipod 封装器,它不认 MP3,实测直接失败
// (Tag mp4a incompatible with output codec id '86017',退出码 183),
// 而且报错是封装器内部信息,用户完全看不懂。MP3 就用 .mp3。
func splitAudioExt(codec string) string {
	switch codec {
	case "aac":
		return ".m4a"
	case "mp3":
		return ".mp3"
	case "opus", "vorbis":
		return ".ogg"
	case "flac":
		return ".flac"
	case "alac", "pcm_s16le", "pcm_s24le", "pcm_s32le", "pcm_f32le", "pcm_u8":
		return ".mka"
	default:
		// 不认识的编码器一律丢进 mkv 系的容器 —— 它最能装
		return ".mka"
	}
}

// mergeOutputName 用「原名_合并」当输出文件名主体。
func mergeOutputName(videoPath string) string {
	return strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath)) + "_合并"
}

func splitOutputName(videoPath, tag string) string {
	return strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath)) + "_" + tag
}

// uniqueFilePath 目标已存在时自动加 (1)(2),绝不静默覆盖用户文件。
func uniqueFilePath(dir, name, ext string) string {
	candidate := filepath.Join(dir, name+ext)
	if _, err := os.Stat(candidate); err != nil {
		return candidate
	}
	for i := 1; i < 1000; i++ {
		p := filepath.Join(dir, fmt.Sprintf("%s(%d)%s", name, i, ext))
		if _, err := os.Stat(p); err != nil {
			return p
		}
	}
	return filepath.Join(dir, name+ext)
}

// ============================================================
// 视频压缩
// ============================================================

// ToolboxCompressFormat 一个压缩目标:容器 + 编码器的组合。
type ToolboxCompressFormat struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Ext   string `json:"ext"`
}

// ToolboxCompressQuality 一档压缩率。
type ToolboxCompressQuality struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Note  string `json:"note"`
}

// ToolboxCompressOptions 压缩可选项,前端据此渲染下拉框。
// 由后端下发而不是前端硬编码,避免两边各写一份以后对不上。
type ToolboxCompressOptions struct {
	Formats []ToolboxCompressFormat  `json:"formats"`
	Quality []ToolboxCompressQuality `json:"quality"`
}

// 常用格式三种:MP4+H.264(兼容性最好)、MP4+H.265(体积最小)、WebM+VP9(网页播放)。
var toolboxCompressFormats = []ToolboxCompressFormat{
	{ID: "mp4-h264", Label: "MP4 · H.264", Ext: ".mp4"},
	{ID: "mp4-h265", Label: "MP4 · H.265", Ext: ".mp4"},
	{ID: "webm-vp9", Label: "WebM · VP9", Ext: ".webm"},
}

// 压缩率三档,CRF 越小画质越好、文件越大。均衡是默认档。
var toolboxCompressQualities = []ToolboxCompressQuality{
	{ID: "high", Label: "高画质", Note: "几乎看不出差别"},
	{ID: "balanced", Label: "均衡"},
	{ID: "small", Label: "小体积", Note: "画质有可见损失"},
}

// ToolboxCompressOptionsList 给前端下拉框用。
func (a *App) ToolboxCompressOptionsList() ToolboxCompressOptions {
	return ToolboxCompressOptions{Formats: toolboxCompressFormats, Quality: toolboxCompressQualities}
}

// toolboxCompressCRF 压缩率档位 → CRF 值。档位与数值的对应集中在这里,
// 前后端都只认档位 ID,不认数值。
func toolboxCompressCRF(quality string) int {
	switch quality {
	case "high":
		return 20
	case "small":
		return 32
	default:
		return 26
	}
}

// toolboxCompressFormat 按 ID 找目标;找不到时回退到默认的 MP4 · H.264。
func toolboxCompressFormat(id string) ToolboxCompressFormat {
	for _, f := range toolboxCompressFormats {
		if f.ID == id {
			return f
		}
	}
	return toolboxCompressFormats[0]
}

// toolboxCompressQuality 按 ID 找档位;找不到时回退到均衡档。
func toolboxCompressQuality(id string) ToolboxCompressQuality {
	for _, q := range toolboxCompressQualities {
		if q.ID == id {
			return q
		}
	}
	return toolboxCompressQualities[1]
}

// CompressVideo 重新编码压缩视频。
//
// 不提供「用不用显卡」的开关:压缩本来就是重编码,有显卡就用、没有就回退 CPU
// 是显然的选择。让用户再勾一次只会和「输出格式」里的 H.264/H.265 选项重叠。
// VP9 在 Windows 上没有可用的显卡编码器,固定走 CPU。
func (a *App) CompressVideo(videoPath, format, quality string) ToolboxStartResult {
	ffmpegPath := a.ffmpegPathForToolbox()
	if ffmpegPath == "" {
		return ToolboxStartResult{OK: false, Message: "未检测到 FFmpeg,请先在「设置 → 视频设置」安装"}
	}
	if videoPath == "" {
		return ToolboxStartResult{OK: false, Message: "请先选择要压缩的视频"}
	}
	if _, err := os.Stat(videoPath); err != nil {
		return ToolboxStartResult{OK: false, Message: "视频文件不存在或无法读取"}
	}
	target := toolboxCompressFormat(format)
	preset := toolboxCompressQuality(quality)
	crf := toolboxCompressCRF(preset.ID)

	videoArgs := a.toolboxCompressVideoArgs(ffmpegPath, target.ID, crf)
	audioArgs := toolboxCompressAudioArgs(target.ID)

	outDir, canceled := a.askToolboxOutputDir()
	if canceled {
		return ToolboxStartResult{Canceled: true}
	}
	outPath := uniqueFilePath(outDir, compressOutputName(videoPath, target.Ext), target.Ext)

	ctx, ok := a.toolbox.begin()
	if !ok {
		return ToolboxStartResult{OK: false, Message: "已有一个任务正在进行,请先等它完成或取消"}
	}
	go a.runToolboxFFmpeg(ctx, "compress", "正在压缩视频", []string{videoPath, ""}, outPath,
		buildCompressArgs(videoPath, outPath, videoArgs, audioArgs))
	return ToolboxStartResult{OK: true}
}

// toolboxCompressVideoArgs 拼视频编码参数。
//
// 有可用的显卡编码器就用它,并把**用户选的画质值替换进去** ——
// 参数表里的画质是给下载转码用的固定值,压缩必须尊重用户选的档位,
// 否则下拉里选什么都不会生效。
func (a *App) toolboxCompressVideoArgs(ffmpegPath, formatID string, crf int) []string {
	q := strconv.Itoa(crf)
	switch formatID {
	case "mp4-h265":
		if spec, ok := a.resolveToolboxGPU(ffmpegPath, "hevc"); ok {
			return withQuality(spec, crf)
		}
		return []string{"-c:v", "libx265", "-preset", "medium", "-crf", q}
	case "webm-vp9":
		// Windows 上没有可用的 VP9 显卡编码器,固定走 CPU。
		// CRF 模式必须配 -b:v 0,否则 ffmpeg 会按普通 VBR 跑,画质档位失效
		return []string{"-c:v", "libvpx-vp9", "-crf", q, "-b:v", "0", "-row-mt", "1"}
	default: // mp4-h264
		if spec, ok := a.resolveToolboxGPU(ffmpegPath, "h264"); ok {
			return withQuality(spec, crf)
		}
		return []string{"-c:v", "libx264", "-preset", "medium", "-crf", q}
	}
}

// toolboxCompressAudioArgs 音频统一重编码成目标容器能放的格式。
// 视频压缩本来就在重编码,音频跟着转一遍比「探测能不能直接复制」更可靠。
func toolboxCompressAudioArgs(formatID string) []string {
	if formatID == "webm-vp9" {
		return []string{"-c:a", "libopus", "-b:a", "128k"}
	}
	return []string{"-c:a", "aac", "-b:a", "160k"}
}

// resolveToolboxGPU 探测指定编码器的显卡实现。
// 与下载器的 HEVC 缓存分开:同一张卡上 h264 与 hevc 的可用性不一定相同,别互相污染。
func (a *App) resolveToolboxGPU(ffmpegPath, codec string) (hevcEncoderSpec, bool) {
	a.toolboxGPUMu.Lock()
	defer a.toolboxGPUMu.Unlock()
	key := codec + "|" + ffmpegPath
	if cached, ok := a.toolboxGPU[key]; ok {
		if cached == nil {
			return hevcEncoderSpec{}, false
		}
		return *cached, true
	}
	var found *hevcEncoderSpec
	for i := range gpuEncoderCandidates[codec] {
		spec := gpuEncoderCandidates[codec][i]
		if probeHEVCEncoder(ffmpegPath, spec) {
			found = &spec
			break
		}
	}
	a.toolboxGPU[key] = found
	if found == nil {
		return hevcEncoderSpec{}, false
	}
	return *found, true
}

// compressOutputName 输出名带上目标格式,方便区分「这是压缩出来的」。
func compressOutputName(videoPath, ext string) string {
	base := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	return base + "_压缩" + strings.TrimPrefix(ext, ".")
}
