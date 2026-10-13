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
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Output  string `json:"output,omitempty"` // 成功时的输出目录
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
	return FilePickResult{OK: true, Path: chosen, Version: codec}
}

// MergeVideoAudio 把外部音频合并进视频,替换/补上原视频的音轨。
//
// reencode=false 走流复制:不重编码,瞬间完成且无画质损失(推荐)。
// reencode=true 用显卡编码器把视频重压成 H.265,体积更小但耗时且不可完全无损。
func (a *App) MergeVideoAudio(videoPath, audioPath string, reencode bool) ToolboxStartResult {
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
	spec, gpu := a.toolboxEncoder(ffmpegPath, reencode)
	if reencode && !gpu {
		return ToolboxStartResult{OK: false, Message: "未检测到可用的显卡编码器,请取消勾选「显卡加速」后重试"}
	}

	// 输出目录:每次弹窗询问,不记、不默认
	outDir, canceled := a.askToolboxOutputDir()
	if canceled {
		return ToolboxStartResult{Canceled: true}
	}

	videoCodec, _ := probeVideoCodec(ffmpegPath, videoPath)
	ext := mergeOutputExt(videoCodec, reencode)
	outPath := uniqueFilePath(outDir, mergeOutputName(videoPath), ext)

	ctx, ok := a.toolbox.begin()
	if !ok {
		return ToolboxStartResult{OK: false, Message: "已有一个任务正在进行,请先等它完成或取消"}
	}
	go a.runToolboxFFmpeg(ctx, "merge", "正在合并视频与音频", []string{videoPath, audioPath}, outPath, func(args []string) []string {
		a := []string{"-y", "-hide_banner", "-loglevel", "error", "-progress", "pipe:1", "-nostats", "-i", args[0], "-i", args[1]}
		a = append(a, "-map", "0:v:0", "-map", "1:a:0")
		a = append(a, "-c:a", "copy")
		a = append(a, spec.videoArgs...)
		return append(a, "-movflags", "+faststart", args[2])
	})
	return ToolboxStartResult{OK: true}
}

// SplitVideoStream 把视频拆成独立的视频流文件与音频流文件。
func (a *App) SplitVideoStream(videoPath string, reencode bool) ToolboxStartResult {
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
	spec, gpu := a.toolboxEncoder(ffmpegPath, reencode)
	if reencode && !gpu {
		return ToolboxStartResult{OK: false, Message: "未检测到可用的显卡编码器,请取消勾选「显卡加速」后重试"}
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
	videoExt := splitVideoExt(videoCodec, reencode)
	audioExt := splitAudioExt(audioCodec)
	videoOut := uniqueFilePath(outDir, splitOutputName(videoPath, "视频"), videoExt)
	audioOut := uniqueFilePath(outDir, splitOutputName(videoPath, "音频"), audioExt)

	ctx, ok := a.toolbox.begin()
	if !ok {
		return ToolboxStartResult{OK: false, Message: "已有一个任务正在进行,请先等它完成或取消"}
	}
	go a.runSplitFFmpeg(ctx, ffmpegPath, videoPath, videoOut, audioOut, spec, reencode, outDir)
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

// toolboxEncoder 选择编码参数:不重编码时用流复制;重编码时优先显卡,失败则报错。
func (a *App) toolboxEncoder(ffmpegPath string, reencode bool) (hevcEncoderSpec, bool) {
	if !reencode {
		// 流复制:不编码,自然也不需要显卡
		return hevcEncoderSpec{id: "copy", label: "无损复制", phase: "复制", videoArgs: []string{"-c:v", "copy"}}, true
	}
	spec, gpu := a.resolveHEVCVideoEncoder(ffmpegPath)
	return spec, gpu
}

// runToolboxFFmpeg 跑一次两输入(视频 + 音频)的 ffmpeg,上报进度并在结束时发事件。
func (a *App) runToolboxFFmpeg(ctx context.Context, phase, phaseText string, inputs []string, outPath string, buildArgs func([]string) []string) {
	defer a.toolbox.end()
	a.emitToolboxProgress(phase, 0, phaseText+"…")
	ffmpegPath := a.ffmpegPathForToolbox()
	if ffmpegPath == "" {
		a.emitToolboxFinished(false, "FFmpeg 已不可用", "")
		return
	}
	args := buildArgs(append(append([]string{}, inputs...), outPath))
	total := a.toolboxTotalDuration(ffmpegPath, inputs[0], inputs[1])
	msg, err := a.execToolboxFFmpeg(ctx, ffmpegPath, args, total, func(percent float64) {
		a.emitToolboxProgress(phase, percent, phaseText+fmt.Sprintf(" %d%%", int(percent)))
	})
	if err != nil {
		if ctx.Err() != nil {
			a.emitToolboxFinished(false, "已取消", "")
			return
		}
		a.emitToolboxFinished(false, "处理失败:"+err.Error(), "")
		return
	}
	a.emitToolboxProgress(phase, 100, "完成")
	a.emitToolboxFinished(true, msg, filepath.Dir(outPath))
}

// runSplitFFmpeg 跑两次 ffmpeg:先抽视频流,再抽音频流,各自上报进度。
func (a *App) runSplitFFmpeg(ctx context.Context, ffmpegPath, videoPath, videoOut, audioOut string, spec hevcEncoderSpec, reencode bool, outDir string) {
	defer a.toolbox.end()
	total, _ := probeDuration(ffmpegPath, videoPath)

	// 第一遍:视频流
	a.emitToolboxProgress("split", 0, "正在分离视频流…")
	videoArgs := []string{
		"-y", "-hide_banner", "-loglevel", "error", "-progress", "pipe:1", "-nostats",
		"-i", videoPath, "-map", "0:v:0", "-an",
	}
	videoArgs = append(videoArgs, spec.videoArgs...)
	videoArgs = append(videoArgs, videoOut)
	if _, err := a.execToolboxFFmpeg(ctx, ffmpegPath, videoArgs, total, func(percent float64) {
		// 视频流占总进度的一半
		a.emitToolboxProgress("split", percent/2, fmt.Sprintf("正在分离视频流 %d%%", int(percent)))
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
		a.emitToolboxProgress("split", 55+percent*0.45, fmt.Sprintf("正在分离音频流 %d%%", int(percent)))
	}); err != nil {
		a.finishSplitError(ctx, err)
		return
	}

	a.emitToolboxProgress("split", 100, "完成")
	a.emitToolboxFinished(true, "已分离为视频流与音频流", outDir)
}

func (a *App) finishSplitError(ctx context.Context, err error) {
	if ctx.Err() != nil {
		a.emitToolboxFinished(false, "已取消", "")
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

// toolboxTotalDuration 合并任务的总时长按视频与音频里较长的那个算。
func (a *App) toolboxTotalDuration(ffmpegPath, videoPath, audioPath string) float64 {
	v, _ := probeDuration(ffmpegPath, videoPath)
	a2, _ := probeDuration(ffmpegPath, audioPath)
	return math.Max(v, a2)
}

func (a *App) emitToolboxProgress(phase string, percent float64, message string) {
	if a.ctx == nil || a.disableEvents {
		return
	}
	runtime.EventsEmit(a.ctx, "toolbox:progress", ToolboxProgress{Phase: phase, Percent: percent, Message: message})
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
// H.264/H.265 用 mp4(兼容性最好);重编码成 H.265 时也一律 mp4。
func mergeOutputExt(videoCodec string, reencode bool) string {
	if reencode {
		return ".mp4"
	}
	switch videoCodec {
	case "h264", "hevc", "av1", "vp9", "vp8":
		return ".mp4"
	default:
		return ".mkv"
	}
}

func splitVideoExt(videoCodec string, reencode bool) string {
	if reencode {
		return ".mp4"
	}
	switch videoCodec {
	case "h264", "hevc", "av1", "vp9", "vp8", "mpeg4":
		return ".mp4"
	default:
		return ".mkv"
	}
}

func splitAudioExt(codec string) string {
	switch codec {
	case "aac", "mp3":
		return ".m4a"
	case "opus", "vorbis":
		return ".ogg"
	case "flac":
		return ".flac"
	case "alac", "pcm_s16le", "pcm_s24le":
		return ".mka"
	default:
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
