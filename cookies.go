package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// 分平台 Cookies:YouTube 与哔哩哔哩对会员、年龄限制、登录可见等资源各自要求对应账号的
// Cookies,混在一份文件里容易互相干扰,所以按站点分开保存、下载时按链接自动选用。
//
// 文件存放在程序数据目录的 cookies 子目录(与 state.json 同级)。导入时只保留属于该站点
// 的数据行,所以就算导出的是整份浏览器 Cookies,也不会把别的站点带进来。

type cookiePlatform struct {
	ID      string
	Name    string
	File    string
	Domains []string
}

// cookiePlatforms 的顺序即界面展示顺序。
// Domains 按后缀匹配:域名等于该值,或以 ".<该值>" 结尾,都算命中。
var cookiePlatforms = []cookiePlatform{
	{
		ID:   "youtube",
		Name: "YouTube",
		File: "youtube.txt",
		// 除 youtube.com 外带上 google.com:YouTube 的账号态有一部分挂在 google.com 域下
		Domains: []string{"youtube.com", "youtu.be", "youtube-nocookie.com", "googlevideo.com", "google.com"},
	},
	{
		ID:      "bilibili",
		Name:    "哔哩哔哩",
		File:    "bilibili.txt",
		Domains: []string{"bilibili.com", "b23.tv", "biliintl.com", "biligame.com", "hdslb.com"},
	},
}

// httpOnlyPrefix 是 Netscape 格式里标记 HttpOnly 的前缀,属于数据行而不是注释。
const httpOnlyPrefix = "#HttpOnly_"

func isHTTPOnlyLine(line string) bool {
	return len(line) >= len(httpOnlyPrefix) && strings.EqualFold(line[:len(httpOnlyPrefix)], httpOnlyPrefix)
}

func isCommentLine(line string) bool {
	return strings.HasPrefix(line, "#") && !isHTTPOnlyLine(line)
}

func cookiePlatformByID(id string) (cookiePlatform, bool) {
	for _, p := range cookiePlatforms {
		if p.ID == id {
			return p, true
		}
	}
	return cookiePlatform{}, false
}

// cookieDataDir 返回存放分平台 Cookies 的目录(与 state.json 同级)。
func cookieDataDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", errors.New("无法确定程序数据目录")
	}
	dir := filepath.Join(base, "DownTube", "cookies")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建 Cookies 目录失败：%v", err)
	}
	return dir, nil
}

func platformCookiePath(p cookiePlatform) (string, error) {
	dir, err := cookieDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, p.File), nil
}

// domainMatches 判断 cookie 域名是否命中后缀列表。
func domainMatches(domain string, suffixes []string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimPrefix(domain, ".")
	if domain == "" {
		return false
	}
	for _, s := range suffixes {
		if domain == s || strings.HasSuffix(domain, "."+s) {
			return true
		}
	}
	return false
}

// cookieLineDomain 取 Netscape 行的域名字段(永远是第一列)。
// #HttpOnly_ 是格式的一部分,比较前要先剥掉;注释行没有域名字段,直接返回空串。
func cookieLineDomain(line string) string {
	if isCommentLine(line) {
		return ""
	}
	s := line
	if isHTTPOnlyLine(s) {
		s = s[len(httpOnlyPrefix):]
	}
	if i := strings.IndexByte(s, '\t'); i >= 0 {
		s = s[:i]
	} else if i := strings.IndexAny(s, " \r\n"); i >= 0 {
		s = s[:i]
	} else {
		return ""
	}
	return strings.TrimSpace(s)
}

// hostOfURL 取链接的主机名,缺协议头时补齐以便解析。
func hostOfURL(rawURL string) string {
	s := strings.TrimSpace(rawURL)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	parsed, err := url.Parse(s)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}

// platformForURL 判断链接属于哪个站点;不认识的站点返回 false。
func platformForURL(rawURL string) (cookiePlatform, bool) {
	host := hostOfURL(rawURL)
	if host == "" {
		return cookiePlatform{}, false
	}
	for _, p := range cookiePlatforms {
		if domainMatches(host, p.Domains) {
			return p, true
		}
	}
	return cookiePlatform{}, false
}

// cookiePlatformsIn 找出文本里出现了哪些站点的 Cookies,保持 cookiePlatforms 的顺序。
func cookiePlatformsIn(text string) []cookiePlatform {
	domains := cookieDomainsIn(text)
	var found []cookiePlatform
	for _, p := range cookiePlatforms {
		for _, d := range domains {
			if domainMatches(d, p.Domains) {
				found = append(found, p)
				break
			}
		}
	}
	return found
}

// cookieDomainsIn 收集文本里所有数据行的域名。
func cookieDomainsIn(text string) []string {
	var domains []string
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" || isCommentLine(line) {
			continue
		}
		if d := cookieLineDomain(line); d != "" {
			domains = append(domains, d)
		}
	}
	return domains
}

// splitCookieLines 挑出属于该平台的数据行,同时返回被丢弃的行数。
// 注释行(各家扩展写的文件头、导出时间等)统一丢弃,写盘时补标准文件头。
func splitCookieLines(text string, p cookiePlatform) (kept []string, dropped int) {
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" || isCommentLine(line) {
			continue
		}
		if domainMatches(cookieLineDomain(line), p.Domains) {
			kept = append(kept, line)
			continue
		}
		dropped++
	}
	return kept, dropped
}

func cookieFileText(lines []string) string {
	return "# Netscape HTTP Cookie File\n" + strings.Join(lines, "\n") + "\n"
}

// countCookieLines 数一个 cookies 文件里的数据行,用于界面展示。
func countCookieLines(filePath string) int {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0
	}
	return len(cookieDomainsIn(string(data)))
}

// platformCookieFile 返回该链接对应站点已导入的 Cookies 文件路径;没导入过返回空串。
func platformCookieFile(rawURL string) string {
	p, ok := platformForURL(rawURL)
	if !ok {
		return ""
	}
	path, err := platformCookiePath(p)
	if err != nil {
		return ""
	}
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return ""
	}
	return path
}

// readCookieText 读取并做基本校验,不改动用户的原文件。
func readCookieText(filePath string) (string, error) {
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
	text := string(bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}))
	if !looksLikeCookieLines(text) {
		return "", errors.New("文件不像 Netscape 格式的 cookies.txt，请用浏览器扩展重新导出(不要选 JSON 格式)")
	}
	return text, nil
}

// CookieStatus 返回分平台 Cookies 的导入状态,供设置页展示。
func (a *App) CookieStatus() map[string]any {
	dir, dirErr := cookieDataDir()
	if dirErr != nil {
		dir = ""
	}
	platforms := []map[string]any{}
	for _, p := range cookiePlatforms {
		item := map[string]any{
			"id": p.ID, "name": p.Name, "file": p.File,
			"imported": false, "count": 0, "updatedAt": "",
		}
		if dir != "" {
			path := filepath.Join(dir, p.File)
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				item["imported"] = true
				item["count"] = countCookieLines(path)
				item["updatedAt"] = info.ModTime().Format("2006-01-02 15:04")
			}
		}
		platforms = append(platforms, item)
	}
	status := map[string]any{"dir": dir, "platforms": platforms}
	if dirErr != nil {
		status["error"] = dirErr.Error()
	}
	return status
}

// ImportPlatformCookie 选择一个 cookies.txt,识别其中的站点并分别保存到数据目录。
// 同时含两个站点的文件会一次写入两个文件,不属于这两个站点的行会被丢弃。
func (a *App) ImportPlatformCookie() map[string]any {
	chosen, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 cookies.txt 文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "Cookies 文件 (*.txt)", Pattern: "*.txt"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil || chosen == "" {
		return map[string]any{"canceled": true}
	}
	chosen = filepath.Clean(chosen)
	text, err := readCookieText(chosen)
	if err != nil {
		return map[string]any{"ok": false, "message": err.Error()}
	}
	matched := cookiePlatformsIn(text)
	if len(matched) == 0 {
		return map[string]any{"ok": false, "message": "没有识别到 YouTube 或哔哩哔哩的 Cookies，请确认导出时选的是对应站点"}
	}
	dir, err := cookieDataDir()
	if err != nil {
		return map[string]any{"ok": false, "message": err.Error()}
	}
	saved := []map[string]any{}
	dropped := 0
	for _, p := range matched {
		kept, drop := splitCookieLines(text, p)
		dropped += drop
		if len(kept) == 0 {
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, p.File), []byte(cookieFileText(kept)), 0o600); err != nil {
			return map[string]any{"ok": false, "message": fmt.Sprintf("保存 %s 的 Cookies 失败：%v", p.Name, err)}
		}
		saved = append(saved, map[string]any{"id": p.ID, "name": p.Name, "count": len(kept)})
	}
	if len(saved) == 0 {
		return map[string]any{"ok": false, "message": "文件里属于目标站点的 Cookies 是空的，请重新导出"}
	}
	return map[string]any{"ok": true, "saved": saved, "dropped": dropped, "dir": dir}
}

// RemovePlatformCookie 删除某个站点已保存的 Cookies。
func (a *App) RemovePlatformCookie(platformID string) map[string]any {
	p, ok := cookiePlatformByID(platformID)
	if !ok {
		return map[string]any{"ok": false, "message": "未知的站点"}
	}
	path, err := platformCookiePath(p)
	if err != nil {
		return map[string]any{"ok": false, "message": err.Error()}
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return map[string]any{"ok": false, "message": fmt.Sprintf("删除失败：%v", err)}
	}
	return map[string]any{"ok": true, "id": p.ID, "name": p.Name}
}

// OpenCookieFolder 打开存放分平台 Cookies 的目录。
func (a *App) OpenCookieFolder() error {
	dir, err := cookieDataDir()
	if err != nil {
		return err
	}
	return exec.Command("explorer.exe", dir).Start()
}
