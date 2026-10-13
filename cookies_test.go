package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const youtubeCookieLine = ".youtube.com\tTRUE\t/\tTRUE\t1800000000\tSID\tabc123"
const googleCookieLine = ".google.com\tTRUE\t/\tTRUE\t1800000000\tSID\tdef456"
const bilibiliCookieLine = ".bilibili.com\tTRUE\t/\tTRUE\t1800000000\tSESSDATA\txyz789"
const otherCookieLine = ".example.com\tTRUE\t/\tTRUE\t1800000000\tfoo\tbar"

func TestDomainMatches(t *testing.T) {
	patterns := []string{"youtube.com", "youtu.be"}
	cases := []struct {
		domain string
		want   bool
	}{
		{"youtube.com", true},
		{".youtube.com", true},
		{"www.youtube.com", true},
		{"m.youtube.com", true},
		{"YOUTUBE.COM", true},
		{" youtu.be ", true},
		// 后缀必须落在点边界上,否则会把别人的域名当成自己人
		{"notyoutube.com", false},
		{"fakeyoutube.com", false},
		{"youtube.com.evil.com", false},
		{"bilibili.com", false},
		{"", false},
	}
	for _, c := range cases {
		if got := domainMatches(c.domain, patterns); got != c.want {
			t.Errorf("domainMatches(%q) = %v, want %v", c.domain, got, c.want)
		}
	}
}

func TestCookieLineDomain(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{youtubeCookieLine, ".youtube.com"},
		{"#HttpOnly_.youtube.com\tTRUE\t/\tTRUE\t0\tSID\tv", ".youtube.com"},
		{"#httponly_.bilibili.com\tTRUE\t/\tTRUE\t0\tSESSDATA\tv", ".bilibili.com"},
		{".bilibili.com\tTRUE\t/\tTRUE\t0\tSESSDATA\tv", ".bilibili.com"},
		{"# Netscape HTTP Cookie File", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := cookieLineDomain(c.line); got != c.want {
			t.Errorf("cookieLineDomain(%q) = %q, want %q", c.line, got, c.want)
		}
	}
}

func TestPlatformForURL(t *testing.T) {
	cases := []struct {
		url      string
		wantID   string
		wantOkay bool
	}{
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", "youtube", true},
		{"https://youtu.be/dQw4w9WgXcQ", "youtube", true},
		{"https://m.youtube.com/watch?v=x", "youtube", true},
		{"https://www.youtube.com", "youtube", true},
		{"www.youtube.com/watch?v=x", "youtube", true},
		{"https://www.bilibili.com/video/BV1xx411c7mD", "bilibili", true},
		{"https://b23.tv/abcdefg", "bilibili", true},
		{"https://live.bilibili.com/12345", "bilibili", true},
		{"https://space.bilibili.com/1", "bilibili", true},
		{"https://www.douyin.com/video/123", "", false},
		{"https://vimeo.com/12345", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		p, ok := platformForURL(c.url)
		if ok != c.wantOkay {
			t.Errorf("platformForURL(%q) ok = %v, want %v", c.url, ok, c.wantOkay)
			continue
		}
		if ok && p.ID != c.wantID {
			t.Errorf("platformForURL(%q) = %q, want %q", c.url, p.ID, c.wantID)
		}
	}
}

func TestCookiePlatformsIn(t *testing.T) {
	// 只含 YouTube
	onlyYT := "# Netscape HTTP Cookie File\n" + youtubeCookieLine + "\n" + googleCookieLine + "\n"
	if got := cookiePlatformsIn(onlyYT); len(got) != 1 || got[0].ID != "youtube" {
		t.Fatalf("只含 YouTube 的文件应识别出 youtube,得到 %v", got)
	}

	// 只含哔哩哔哩
	onlyBili := "# Netscape HTTP Cookie File\n" + bilibiliCookieLine + "\n"
	if got := cookiePlatformsIn(onlyBili); len(got) != 1 || got[0].ID != "bilibili" {
		t.Fatalf("只含哔哩哔哩的文件应识别出 bilibili,得到 %v", got)
	}

	// 两个站点都有
	both := "# Netscape HTTP Cookie File\n" + youtubeCookieLine + "\n" + bilibiliCookieLine + "\n" + otherCookieLine + "\n"
	got := cookiePlatformsIn(both)
	if len(got) != 2 {
		t.Fatalf("混合文件应识别出 2 个站点,得到 %d 个: %v", len(got), got)
	}
	// 顺序应与 cookiePlatforms 一致
	if got[0].ID != "youtube" || got[1].ID != "bilibili" {
		t.Fatalf("识别顺序应为 youtube, bilibili,得到 %v", got)
	}

	// 都不认识
	if got := cookiePlatformsIn("# Netscape HTTP Cookie File\n" + otherCookieLine + "\n"); len(got) != 0 {
		t.Fatalf("无关站点不应被识别,得到 %v", got)
	}
	// 空文件
	if got := cookiePlatformsIn(""); len(got) != 0 {
		t.Fatalf("空文本不应识别出站点,得到 %v", got)
	}
}

func TestSplitCookieLines(t *testing.T) {
	yt, _ := cookiePlatformByID("youtube")
	bili, _ := cookiePlatformByID("bilibili")

	text := "# Netscape HTTP Cookie File\n" +
		"# 导出自某扩展\n" +
		youtubeCookieLine + "\n" +
		googleCookieLine + "\n" +
		bilibiliCookieLine + "\n" +
		otherCookieLine + "\n" +
		"#HttpOnly_.youtube.com\tTRUE\t/\tTRUE\t0\tHSID\tv\n" +
		"\n"

	kept, dropped := splitCookieLines(text, yt)
	if len(kept) != 3 {
		t.Fatalf("YouTube 应保留 3 行,得到 %d: %v", len(kept), kept)
	}
	// 哔哩哔哩与其他站点各 1 行被丢弃
	if dropped != 2 {
		t.Fatalf("YouTube 应丢弃 2 行,得到 %d", dropped)
	}
	for _, line := range kept {
		if !isHTTPOnlyLine(line) && !domainMatches(cookieLineDomain(line), yt.Domains) {
			t.Fatalf("保留了不属于 YouTube 的行: %q", line)
		}
	}
	if !isHTTPOnlyLine(kept[2]) {
		t.Fatalf("HttpOnly 前缀的行应被保留,得到 %q", kept[2])
	}

	keptBili, droppedBili := splitCookieLines(text, bili)
	if len(keptBili) != 1 || droppedBili != 4 {
		t.Fatalf("哔哩哔哩应保留 1 行丢弃 4 行,得到 %d/%d", len(keptBili), droppedBili)
	}
}

func TestCookieFileText(t *testing.T) {
	out := cookieFileText([]string{youtubeCookieLine})
	want := "# Netscape HTTP Cookie File\n" + youtubeCookieLine + "\n"
	if out != want {
		t.Fatalf("生成的 cookies 文件格式不对:\n%q\nwant:\n%q", out, want)
	}
	if !looksLikeCookieLines(out) {
		t.Fatal("生成的文本应通过 Netscape 格式校验")
	}
}

// TestCookieImportRoundTrip 走一遍完整往返:混合导出文件 → 识别 → 分站点落盘 → 按链接解析回来。
// os.UserConfigDir 在 Windows 上就是 %AppData%,指到临时目录即可,不会碰到真实数据目录。
func TestCookieImportRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AppData", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	src := filepath.Join(tmp, "all-cookies.txt")
	text := "# Netscape HTTP Cookie File\n# 导出自某扩展\n" +
		youtubeCookieLine + "\n" + googleCookieLine + "\n" + bilibiliCookieLine + "\n" + otherCookieLine + "\n"
	if err := os.WriteFile(src, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}

	read, err := readCookieText(src)
	if err != nil {
		t.Fatalf("读取导出文件失败: %v", err)
	}
	matched := cookiePlatformsIn(read)
	if len(matched) != 2 {
		t.Fatalf("应识别出 2 个站点,得到 %d 个: %v", len(matched), matched)
	}
	for _, p := range matched {
		kept, _ := splitCookieLines(read, p)
		path, err := platformCookiePath(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(cookieFileText(kept)), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// 下载时按链接解析到的必须是刚写进去的那份
	ytPath := platformCookieFile("https://www.youtube.com/watch?v=x")
	if ytPath == "" {
		t.Fatal("YouTube 链接应解析到已导入的 Cookies 文件")
	}
	if filepath.Base(ytPath) != "youtube.txt" {
		t.Fatalf("YouTube 应命中 youtube.txt,得到 %s", filepath.Base(ytPath))
	}
	if n := countCookieLines(ytPath); n != 2 {
		t.Fatalf("youtube.txt 应有 2 行,得到 %d", n)
	}

	biliPath := platformCookieFile("https://www.bilibili.com/video/BV1xx411c7mD")
	if filepath.Base(biliPath) != "bilibili.txt" {
		t.Fatalf("哔哩哔哩应命中 bilibili.txt,得到 %s", filepath.Base(biliPath))
	}
	if n := countCookieLines(biliPath); n != 1 {
		t.Fatalf("bilibili.txt 应有 1 行,得到 %d", n)
	}

	// 两个文件都应落在数据目录的 cookies 子目录里
	if got := filepath.Dir(ytPath); filepath.Base(got) != "cookies" {
		t.Fatalf("Cookies 应存放在 cookies 子目录,得到 %s", got)
	}

	// 未识别的站点不该拿到任何 Cookies(否则等于串号)
	if p := platformCookieFile("https://vimeo.com/12345"); p != "" {
		t.Fatalf("无关站点不应解析到 Cookies,得到 %s", p)
	}

	// 落盘内容必须仍是合法 Netscape 文件
	if data, err := os.ReadFile(ytPath); err != nil || !looksLikeCookieLines(string(data)) {
		t.Fatalf("落盘文件应是合法 Netscape 格式,err=%v", err)
	}

	// CookieStatus 要如实反映导入状态,供设置页展示
	status := (&App{}).CookieStatus()
	list := status.Platforms
	if len(list) != 2 {
		t.Fatalf("状态应包含 2 个站点,得到 %d", len(list))
	}
	for _, item := range list {
		if item.ID == "youtube" && item.Imported != true {
			t.Error("YouTube 应显示为已导入")
		}
		if item.ID == "youtube" && item.Count != 2 {
			t.Errorf("YouTube 条数应为 2,得到 %v", item.Count)
		}
		if item.ID == "bilibili" && item.Count != 1 {
			t.Errorf("哔哩哔哩条数应为 1,得到 %v", item.Count)
		}
	}
	if status.Dir == "" {
		t.Error("状态里应带上 Cookies 目录路径")
	}
}

// TestAppendNetworkArgsPicksPlatformCookie 验证下载参数按链接挑 Cookies 文件的接线:
// 有对应站点文件就用它,没有就回退通用文件,绝不串用别的站点的身份。
func TestAppendNetworkArgsPicksPlatformCookie(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AppData", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	yt, _ := cookiePlatformByID("youtube")
	ytPath, err := platformCookiePath(yt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ytPath, []byte(cookieFileText([]string{youtubeCookieLine})), 0o600); err != nil {
		t.Fatal(err)
	}

	app := &App{}
	settings := Settings{UseBrowserCookie: true, CookieSource: "file", CookieFile: "D:/fallback/cookies.txt"}

	// 有对应站点文件时优先用它
	args := app.appendNetworkArgs(nil, settings, "https://www.youtube.com/watch?v=x")
	if !strings.Contains(strings.Join(args, " "), ytPath) {
		t.Fatalf("YouTube 链接应使用分站点文件,得到 %v", args)
	}

	// 哔哩哔哩没导入:回退通用文件,不能借用 YouTube 那份
	args = app.appendNetworkArgs(nil, settings, "https://www.bilibili.com/video/BV1xx411c7mD")
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "youtube.txt") {
		t.Fatalf("哔哩哔哩不能误用 YouTube 的 Cookies,得到 %v", args)
	}
	if !strings.Contains(joined, "D:/fallback/cookies.txt") {
		t.Fatalf("哔哩哔哩应回退通用文件,得到 %v", args)
	}

	// 完全不认识的站点同样回退通用文件
	args = app.appendNetworkArgs(nil, settings, "https://vimeo.com/12345")
	if !strings.Contains(strings.Join(args, " "), "D:/fallback/cookies.txt") {
		t.Fatalf("其他站点应回退通用文件,得到 %v", args)
	}

	// 关闭 Cookies 时不应追加任何 --cookies
	off := Settings{UseBrowserCookie: false}
	if args := app.appendNetworkArgs(nil, off, "https://www.youtube.com/watch?v=x"); len(args) != 0 {
		t.Fatalf("关闭 Cookies 时不应追加参数,得到 %v", args)
	}

	// 从浏览器读取模式走 --cookies-from-browser
	fromBrowser := Settings{UseBrowserCookie: true, CookieSource: "browser", CookieBrowser: "Microsoft Edge"}
	args = app.appendNetworkArgs(nil, fromBrowser, "https://www.youtube.com/watch?v=x")
	if !strings.Contains(strings.Join(args, " "), "--cookies-from-browser edge") {
		t.Fatalf("浏览器模式应映射为 edge,得到 %v", args)
	}
}

// TestRemovePlatformCookie 覆盖移除逻辑:移除后链接不应再解析到该站点的文件。
func TestRemovePlatformCookie(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AppData", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)

	yt, _ := cookiePlatformByID("youtube")
	path, err := platformCookiePath(yt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(cookieFileText([]string{youtubeCookieLine})), 0o600); err != nil {
		t.Fatal(err)
	}
	if platformCookieFile("https://www.youtube.com/watch?v=x") == "" {
		t.Fatal("写入后应能解析到 Cookies 文件")
	}

	app := &App{}
	if res := app.RemovePlatformCookie("youtube"); res.OK != true {
		t.Fatalf("移除应成功,得到 %v", res)
	}
	if platformCookieFile("https://www.youtube.com/watch?v=x") != "" {
		t.Fatal("移除后不应再解析到 Cookies 文件")
	}
	// 重复移除应当幂等,不报错
	if res := app.RemovePlatformCookie("youtube"); res.OK != true {
		t.Fatalf("重复移除应保持成功,得到 %v", res)
	}
	// 未知站点应被拒绝
	if res := app.RemovePlatformCookie("nope"); res.OK != false {
		t.Fatalf("未知站点应被拒绝,得到 %v", res)
	}
}
