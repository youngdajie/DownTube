package main

import (
	"math"
	"strings"
	"testing"
)

// TestNormalizeVideoURLAcceptsAnySite 下载内核是 yt-dlp,不该只放行 YouTube。
func TestNormalizeVideoURLAcceptsAnySite(t *testing.T) {
	accepted := []string{
		"https://www.bilibili.com/video/BV1tepM6aECP/",
		"https://b23.tv/abcdefg",
		"https://www.douyin.com/video/123456",
		"https://vimeo.com/12345",
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ",
		"https://x.com/user/status/1",
		"http://example.com/video.mp4",
		"https://www.bilibili.com/video/BV1x?p=2&t=10",
	}
	for _, u := range accepted {
		got, err := normalizeVideoURL(u)
		if err != nil {
			t.Errorf("应接受 %q,却报错: %v", u, err)
			continue
		}
		if got == "" {
			t.Errorf("应接受 %q,却返回空链接", u)
		}
	}
}

// TestNormalizeVideoURLRejectsGarbage 覆盖必须挡掉的输入。
// 其中命令行形态的字符串尤其重要:链接会被原样拼进 yt-dlp 参数,
// 若放行"--xxx"这类内容就会被当成选项执行。
func TestNormalizeVideoURLRejectsGarbage(t *testing.T) {
	rejected := []string{
		"",
		"hello",
		"www",
		"just some text",
		"https://nodot",
		"http://",
		"javascript:alert(1)",
		"file:///C:/Windows/System32/calc.exe",
		"ftp://example.com/x",
		"--exec=calc",
		"--output=/tmp/x",
	}
	for _, u := range rejected {
		if got, err := normalizeVideoURL(u); err == nil {
			t.Errorf("应拒绝 %q,却返回 %q", u, got)
		}
	}
}

// TestPlanTranscodeSegments 分片边界必须首尾衔接、无重叠无缝隙,
// 它是"拼接成品时长与源一致"的前提。
func TestPlanTranscodeSegments(t *testing.T) {
	// 用户报障那个视频的真实时长
	segs := planTranscodeSegments(1198.105556)
	if len(segs) != 40 {
		t.Fatalf("1198.105556 秒应切 40 段,得到 %d 段", len(segs))
	}
	if math.Abs(segs[0].start-0) > 1e-9 || math.Abs(segs[0].end-30) > 1e-9 {
		t.Fatalf("首段应为 [0,30],得到 [%v,%v]", segs[0].start, segs[0].end)
	}
	last := segs[len(segs)-1]
	if math.Abs(last.start-1170) > 1e-6 || math.Abs(last.end-1198.105556) > 1e-6 {
		t.Fatalf("末段应为 [1170,1198.105556],得到 [%v,%v]", last.start, last.end)
	}

	// 边界必须严格衔接:每段终点等于下一段起点
	for i, s := range segs {
		if s.end <= s.start {
			t.Fatalf("第 %d 段时长非正: [%v,%v]", i, s.start, s.end)
		}
		if i > 0 && math.Abs(segs[i-1].end-s.start) > 1e-9 {
			t.Fatalf("第 %d 段起点 %v 与上一段终点 %v 不衔接", i, s.start, segs[i-1].end)
		}
	}
	// 总时长必须守恒
	sum := 0.0
	for _, s := range segs {
		sum += s.dur()
	}
	if math.Abs(sum-1198.105556) > 1e-6 {
		t.Fatalf("分段总时长 %v 与源时长不符", sum)
	}
}

func TestPlanTranscodeSegmentsEdgeCases(t *testing.T) {
	cases := []struct {
		total float64
		want  int
	}{
		{0, 0},
		{-1, 0},
		{1, 1},
		{29.999, 1},
		{30, 1}, // 整除时不应多切出一段
		// 刚过整秒一点点时由末段吸收(留 1 毫秒容差),
		// 否则会切出一个几毫秒的碎片段,得不偿失
		{30.001, 1},
		{31, 2},
		{60, 2},
		{60.5, 3},
		{90, 3},
	}
	for _, c := range cases {
		if got := len(planTranscodeSegments(c.total)); got != c.want {
			t.Errorf("planTranscodeSegments(%v) 得到 %d 段,期望 %d 段", c.total, got, c.want)
		}
	}
}

// TestEngineErrorZh 用 yt-dlp / ffmpeg 的真实报错样本验证中文化,
// 确保界面不会直接漏出英文。
func TestEngineErrorZh(t *testing.T) {
	cases := []struct{ raw, want string }{
		{"ERROR: [BiliBili] BV1xx: Video unavailable", "视频不可用，可能已被删除或设为私享"},
		{"ERROR: [youtube] abc: This video has been removed by the uploader", "视频已被上传者删除"},
		{"ERROR: [youtube] abc: Sign in to confirm you're not a bot", "站点要求登录验证，请在设置 → Cookies 中提供 Cookies"},
		{"ERROR: [bilibili] BV1: Unsupported URL: https://www.bilibili.com/", "下载内核不支持这个链接"},
		{"ERROR: [youtube] abc: This video is available to this channel's members", "会员专属内容，需要对应账号的 Cookies"},
		{"ERROR: Unable to download webpage: HTTP Error 403: Forbidden", "请求被拒绝（HTTP 403），通常是 Cookies 失效或当前代理 IP 被限制"},
		{"ERROR: [youtube] abc: HTTP Error 404: Not Found", "资源不存在（HTTP 404），视频可能已被删除或地址有误"},
		{"ERROR: [youtube] abc: HTTP Error 429: Too Many Requests", "请求过于频繁（HTTP 429），请稍后重试或减少并发下载数"},
		{"ERROR: Unable to download API page: HTTP Error 503", "对方服务器出错（HTTP 503），请稍后重试"},
		{"ERROR: [youtube] abc: Requested format is not available for this video", "没有符合当前画质设置的格式，试试「最佳可用」或安装 FFmpeg"},
		{"ERROR: unable to open cookie file: No such file or directory", "Cookies 文件无法读取，请在设置中重新选择"},
		{"Conversion failed!", "转码失败"},
		{"No space left on device", "磁盘空间不足"},
		{"Connection refused", "连接被拒绝，请检查网络或代理设置"},
	}
	for _, c := range cases {
		if got := engineErrorZh(c.raw); got != c.want {
			t.Errorf("engineErrorZh(%q)\n  得到 %q\n  期望 %q", c.raw, got, c.want)
		}
	}
}

// TestEngineErrorTextFallback 没有对应规则时保留原文,但要标注来源,
// 避免用户以为是软件自身的报错。
func TestEngineErrorTextFallback(t *testing.T) {
	if got := engineErrorText("ERROR: SomeBrandNewExtractorBrand new error message"); got != "下载内核报错：ERROR: SomeBrandNewExtractorBrand new error message" {
		t.Fatalf("未命中规则时应保留原文并标注来源,得到 %q", got)
	}
	if got := engineErrorText("ERROR: [youtube] abc: Video unavailable"); strings.Contains(got, "ERROR") || !strings.Contains(got, "视频不可用") {
		t.Fatalf("命中规则时不应出现英文原文,得到 %q", got)
	}
	if got := engineErrorText(""); got != "" {
		t.Fatalf("空输入应返回空串,得到 %q", got)
	}
	// 多行报错要去重后用分号连接
	multi := "ERROR: [youtube] a: Video unavailable\nERROR: [youtube] b: Video unavailable"
	if got := engineErrorZh(multi); got != "视频不可用，可能已被删除或设为私享" {
		t.Fatalf("重复报错应去重,得到 %q", got)
	}
}
