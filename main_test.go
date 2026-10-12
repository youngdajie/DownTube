package main

import "testing"

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
