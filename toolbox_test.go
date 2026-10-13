package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestToolboxRunnerSingleFlight 保证同一时刻只有一个合并/分离任务。
func TestToolboxRunnerSingleFlight(t *testing.T) {
	var r toolboxRunner
	ctx, ok := r.begin()
	if !ok || ctx == nil {
		t.Fatal("第一次 begin 应该成功")
	}
	if _, ok2 := r.begin(); ok2 {
		t.Error("任务进行中时第二次 begin 应该失败")
	}
	if !r.busy() {
		t.Error("busy 应为 true")
	}
	r.end()
	if r.busy() {
		t.Error("end 之后 busy 应为 false")
	}
	if _, ok3 := r.begin(); !ok3 {
		t.Error("end 之后应能再次 begin")
	}
}

// TestToolboxRunnerCancel 只有真正在跑的任务才能取消。
func TestToolboxRunnerCancel(t *testing.T) {
	var r toolboxRunner
	if r.cancelTask() {
		t.Error("没有任务时 cancelTask 应返回 false")
	}
	if _, ok := r.begin(); !ok {
		t.Fatal("begin 失败")
	}
	if !r.cancelTask() {
		t.Error("任务进行中 cancelTask 应返回 true")
	}
	r.end()
}

// TestToolboxOutputContainers 输出容器必须与编码器匹配,否则 ffmpeg 直接报错。
func TestToolboxOutputContainers(t *testing.T) {
	cases := []struct {
		codec    string
		reencode bool
		wantExt  string
	}{
		{"h264", false, ".mp4"},
		{"hevc", false, ".mp4"},
		{"av1", false, ".mp4"},
		{"vp9", false, ".mp4"},
		{"mpeg4", false, ".mkv"},
		{"wmv3", false, ".mkv"},
		// 重编码成 H.265 时一律 mp4:H.265 在 mkv 里能用但 mp4 兼容性最好,
		// 而 webm 根本不支持 H.265
		{"vp9", true, ".mp4"},
		{"h264", true, ".mp4"},
	}
	for _, c := range cases {
		if got := mergeOutputExt(c.codec, c.reencode); got != c.wantExt {
			t.Errorf("mergeOutputExt(%q, %v) = %q, 期望 %q", c.codec, c.reencode, got, c.wantExt)
		}
		if got := splitVideoExt(c.codec, c.reencode); got != c.wantExt && !(c.codec == "mpeg4" && got == ".mp4") {
			t.Errorf("splitVideoExt(%q, %v) = %q, 期望 %q", c.codec, c.reencode, got, c.wantExt)
		}
	}
}

// TestToolboxAudioExt 音频扩展名必须与编码器对应。
func TestToolboxAudioExt(t *testing.T) {
	cases := map[string]string{
		"aac": ".m4a", "mp3": ".m4a", "opus": ".ogg", "vorbis": ".ogg",
		"flac": ".flac", "alac": ".mka", "pcm_s16le": ".mka", "": ".mka", "奇怪的编码": ".mka",
	}
	for codec, want := range cases {
		if got := splitAudioExt(codec); got != want {
			t.Errorf("splitAudioExt(%q) = %q, 期望 %q", codec, got, want)
		}
	}
}

// TestUniqueFilePathNeverOverwrites 输出文件已存在时必须换名,不能覆盖。
func TestUniqueFilePathNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	first := uniqueFilePath(dir, "影片", ".mp4")
	if want := filepath.Join(dir, "影片.mp4"); first != want {
		t.Fatalf("首次应为 %q, 得到 %q", want, first)
	}
	if err := os.WriteFile(first, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	second := uniqueFilePath(dir, "影片", ".mp4")
	if want := filepath.Join(dir, "影片(1).mp4"); second != want {
		t.Errorf("重名时应为 %q, 得到 %q", want, second)
	}
	if err := os.WriteFile(second, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	third := uniqueFilePath(dir, "影片", ".mp4")
	if want := filepath.Join(dir, "影片(2).mp4"); third != want {
		t.Errorf("再次重名时应为 %q, 得到 %q", want, third)
	}
}

// TestToolboxOutputNames 输出文件名要能看出是合并还是分离的结果。
func TestToolboxOutputNames(t *testing.T) {
	if got := mergeOutputName("D:/a/b/我的视频.mp4"); got != "我的视频_合并" {
		t.Errorf("mergeOutputName = %q", got)
	}
	if got := splitOutputName("D:/a/b/我的视频.mkv", "视频"); got != "我的视频_视频" {
		t.Errorf("splitOutputName = %q", got)
	}
	if got := splitOutputName("D:/a/b/我的视频.mkv", "音频"); got != "我的视频_音频" {
		t.Errorf("splitOutputName = %q", got)
	}
}

// TestToolboxEncoderCopy 不勾选显卡加速时必须走流复制 —— 合并/分离本身不需要重编码,
// 强行用显卡编码反而会让「瞬间完成」变成「几分钟且不可无损」。
func TestToolboxEncoderCopy(t *testing.T) {
	app := &App{}
	spec, gpu := app.toolboxEncoder("ffmpeg.exe", false)
	if !gpu {
		t.Error("流复制路径应当始终可用")
	}
	if spec.id != "copy" {
		t.Errorf("流复制 spec.id 应为 copy, 得到 %q", spec.id)
	}
	if len(spec.videoArgs) != 2 || spec.videoArgs[0] != "-c:v" || spec.videoArgs[1] != "copy" {
		t.Errorf("流复制参数应为 -c:v copy, 得到 %v", spec.videoArgs)
	}
}

// TestToolboxContractKeys 新接口的 JSON 键集。
func TestToolboxContractKeys(t *testing.T) {
	startOld := map[string]any{"ok": true}
	startNew := ToolboxStartResult{OK: true}
	assertSameKeys(t, "ToolboxStartResult", startOld, startNew)

	busyOld := map[string]any{"ok": false, "message": "m"}
	busyNew := ToolboxStartResult{Message: "m"}
	assertSameKeys(t, "ToolboxStartResult(失败)", busyOld, busyNew)

	// 取消分支:与 SelectCookieFile 同理,结构化后会多出 ok:false(前端先判 canceled,行为不变)
	canceledOld := map[string]any{"ok": false, "canceled": true}
	canceledNew := ToolboxStartResult{Canceled: true}
	assertSameKeys(t, "ToolboxStartResult(取消)", canceledOld, canceledNew)

	progressNew := ToolboxProgress{Phase: "merge", Percent: 50, Message: "m"}
	progressOld := map[string]any{"phase": "merge", "percent": float64(50), "message": "m"}
	assertSameKeys(t, "ToolboxProgress", progressOld, progressNew)

	finishedOld := map[string]any{"ok": true, "message": "m", "output": "D:/out"}
	finishedNew := ToolboxFinished{OK: true, Message: "m", Output: "D:/out"}
	assertSameKeys(t, "ToolboxFinished", finishedOld, finishedNew)

	// PickToolboxVideo 复用 FilePickResult:ok/path/version/canceled/message
	videoOld := map[string]any{"ok": true, "path": "p", "version": "h264"}
	videoNew := FilePickResult{OK: true, Path: "p", Version: "h264"}
	assertSameKeys(t, "FilePickResult(视频)", videoOld, videoNew)
}

// TestFirstAudioCodec 只取第一行:ffprobe 偶尔会带回车或多行。
func TestFirstAudioCodec(t *testing.T) {
	cases := map[string]string{
		"aac":          "aac",
		"aac\n":        "aac",
		"opus\nflac":   "opus",
		"  aac  \n  x": "aac",
		"":             "",
	}
	for in, want := range cases {
		if got := firstAudioCodec(in); got != want {
			t.Errorf("firstAudioCodec(%q) = %q, 期望 %q", in, got, want)
		}
	}
}

// TestToolboxRunnerConcurrent 并发抢占只能有一个赢家。
func TestToolboxRunnerConcurrent(t *testing.T) {
	var r toolboxRunner
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins := 0
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, ok := r.begin(); ok {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Errorf("20 个并发 begin 只应有 1 个成功, 实际 %d", wins)
	}
}

// ---------- 视频压缩 ----------

// TestToolboxCompressOptions 选项必须由后端下发,且三种常用格式都在。
func TestToolboxCompressOptions(t *testing.T) {
	app := &App{toolboxGPU: make(map[string]*hevcEncoderSpec)}
	opts := app.ToolboxCompressOptionsList()
	if len(opts.Formats) != 3 {
		t.Fatalf("应有 3 种格式, 得到 %d", len(opts.Formats))
	}
	ids := map[string]bool{}
	for _, f := range opts.Formats {
		ids[f.ID] = true
		if f.Ext == "" || f.Label == "" {
			t.Errorf("格式 %q 的 ext/label 不能为空", f.ID)
		}
	}
	for _, want := range []string{"mp4-h264", "mp4-h265", "webm-vp9"} {
		if !ids[want] {
			t.Errorf("缺少格式 %s", want)
		}
	}
	if len(opts.Quality) != 3 {
		t.Fatalf("应有 3 档压缩率, 得到 %d", len(opts.Quality))
	}
}

// TestToolboxCompressCRF 压缩率档位必须单调:越高画质 CRF 越小。
func TestToolboxCompressCRF(t *testing.T) {
	high, balanced, small := toolboxCompressCRF("high"), toolboxCompressCRF("balanced"), toolboxCompressCRF("small")
	if !(high < balanced && balanced < small) {
		t.Errorf("CRF 应随压缩率递增: 高画质=%d 均衡=%d 小体积=%d", high, balanced, small)
	}
	if got := toolboxCompressCRF("不存在的档位"); got != balanced {
		t.Errorf("未知档位应回退到均衡(%d), 得到 %d", balanced, got)
	}
}

// TestToolboxCompressFormatFallback 未知格式必须回退,不能让 ffmpeg 拿到空扩展名。
func TestToolboxCompressFormatFallback(t *testing.T) {
	if got := toolboxCompressFormat("胡说八道"); got.ID != "mp4-h264" {
		t.Errorf("未知格式应回退到 mp4-h264, 得到 %s", got.ID)
	}
	if got := toolboxCompressQuality("胡说八道"); got.ID != "balanced" {
		t.Errorf("未知档位应回退到 balanced, 得到 %s", got.ID)
	}
}

// TestToolboxCompressArgs 编码参数:CPU 回退、VP9 的 -b:v 0、音频容器匹配。
func TestToolboxCompressArgs(t *testing.T) {
	app := &App{toolboxGPU: make(map[string]*hevcEncoderSpec)}
	// 不探测显卡(useGPU=false),必须落到 CPU 编码器
	for _, tc := range []struct {
		format string
		want   string
		audio  string
	}{
		{"mp4-h264", "libx264", "aac"},
		{"mp4-h265", "libx265", "aac"},
		{"webm-vp9", "libvpx-vp9", "libopus"},
	} {
		args, gpu := app.toolboxCompressVideoArgs("ffmpeg.exe", tc.format, 26, false)
		if gpu {
			t.Errorf("%s 未勾选显卡却走了显卡", tc.format)
		}
		if !containsArg(args, "-c:v", tc.want) {
			t.Errorf("%s 应使用 %s, 实际参数 %v", tc.format, tc.want, args)
		}
		if !containsArg(args, "-crf", "26") {
			t.Errorf("%s 应带上 -crf 26, 实际参数 %v", tc.format, args)
		}
		a := toolboxCompressAudioArgs(tc.format)
		if !containsArg(a, "-c:a", tc.audio) {
			t.Errorf("%s 的音频应为 %s, 实际 %v", tc.format, tc.audio, a)
		}
	}
	// VP9 的 CRF 模式必须配 -b:v 0,否则按 VBR 跑,画质档位失效
	vp9, _ := app.toolboxCompressVideoArgs("ffmpeg.exe", "webm-vp9", 28, false)
	if !containsArg(vp9, "-b:v", "0") {
		t.Errorf("VP9 必须配 -b:v 0, 实际 %v", vp9)
	}
	// webm 里不能放 aac,mp4 里不能放 opus
	if got := toolboxCompressAudioArgs("webm-vp9"); containsArg(got, "-c:a", "aac") {
		t.Error("WebM 容器不支持 AAC")
	}
	if got := toolboxCompressAudioArgs("mp4-h264"); containsArg(got, "-c:a", "libopus") {
		t.Error("MP4 容器不支持 Opus")
	}
}

// TestToolboxCompressGPUProbeCache 显卡探测结果要按「编码器 + ffmpeg 路径」缓存,
// 且探测不到时缓存 false,不会每次压缩都重新探测一遍。
func TestToolboxCompressGPUProbeCache(t *testing.T) {
	app := &App{toolboxGPU: make(map[string]*hevcEncoderSpec)}
	// 用一个不可能存在的 ffmpeg 路径:探测必然失败,但结果要被缓存
	if _, ok := app.resolveToolboxGPU("Z:/不存在的/ffmpeg.exe", "h264"); ok {
		t.Error("不存在的 ffmpeg 不该探测到显卡编码器")
	}
	if _, cached := app.toolboxGPU["h264|Z:/不存在的/ffmpeg.exe"]; !cached {
		t.Error("探测失败的结果也应缓存,避免重复探测")
	}
	// hevc 与 h264 分开缓存
	if _, cached := app.toolboxGPU["hevc|Z:/不存在的/ffmpeg.exe"]; cached {
		t.Error("hevc 的缓存键不应被 h264 污染")
	}
	// 未知编码器名:候选列表为空,直接返回 false 且不 panic
	if _, ok := app.resolveToolboxGPU("ffmpeg.exe", "av1"); ok {
		t.Error("未登记的编码器不该返回可用")
	}
}

// TestCompressOutputName 输出名要能看出是压缩产物,且不带重复扩展名。
func TestCompressOutputName(t *testing.T) {
	if got := compressOutputName("D:/a/b/我的视频.mp4", ".mp4"); got != "我的视频_压缩.mp4" {
		t.Errorf("compressOutputName = %q", got)
	}
	if got := compressOutputName("D:/a/b/我的视频.mkv", ".webm"); got != "我的视频_压缩.webm" {
		t.Errorf("compressOutputName = %q", got)
	}
	if got := compressOutputName("D:/a/b/无扩展名", ".mp4"); got != "无扩展名_压缩.mp4" {
		t.Errorf("compressOutputName = %q", got)
	}
}

// TestToolboxCompressContractKeys 压缩接口的 JSON 键集。
func TestToolboxCompressContractKeys(t *testing.T) {
	opts := (&App{toolboxGPU: make(map[string]*hevcEncoderSpec)}).ToolboxCompressOptionsList()
	assertSameKeys(t, "ToolboxCompressOptions",
		map[string]any{"formats": []any{}, "quality": []any{}}, opts)
	assertSameKeys(t, "ToolboxCompressFormat",
		map[string]any{"id": "mp4-h264", "label": "L", "ext": ".mp4"}, opts.Formats[0])
	assertSameKeys(t, "ToolboxCompressQuality",
		map[string]any{"id": "high", "label": "L", "note": "N"}, opts.Quality[0])
}

func containsArg(args []string, flag, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}
