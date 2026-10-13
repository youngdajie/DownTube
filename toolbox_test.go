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
