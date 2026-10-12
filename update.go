package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ============================================================
// 应用内更新
// ------------------------------------------------------------
// 检查:优先 CNB 仓库(国内直连,读 wails.json 版本号 + 探测安装包是否就绪),
// 兜底 GitHub releases API。
// 更新:应用内下载安装包 → 以管理员权限静默启动 NSIS 安装程序(/S),
// 安装程序负责结束旧进程、覆盖文件,完成后自动重启新版。
// ============================================================

const (
	updateRepoCNB    = "youngdajie/downtube"
	updateRepoGitHub = "youngdajie/DownTube"
	updateAssetName  = "DownTube-Setup-%s.exe" // 发版流水线按「版本号」统一命名安装包
	updateTempDir    = "DownTube-Update"
)

var updateVersionPattern = regexp.MustCompile(`^\d+(\.\d+){1,3}$`)

// parseVersionNums 把 "1.2.3" 解析为 [1,2,3];遇非数字后缀(如 -beta)即截断。
func parseVersionNums(v string) []int {
	v = strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(v), "v"), "V")
	parts := strings.Split(v, ".")
	nums := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		i := 0
		for i < len(part) && part[i] >= '0' && part[i] <= '9' {
			i++
		}
		if i == 0 {
			break
		}
		n, err := strconv.Atoi(part[:i])
		if err != nil {
			break
		}
		nums = append(nums, n)
		if i < len(part) {
			break
		}
	}
	return nums
}

// compareVersions 比较两个版本号:1 表示 a 更新,-1 表示 b 更新,0 表示相同。
func compareVersions(a, b string) int {
	pa, pb := parseVersionNums(a), parseVersionNums(b)
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x != y {
			if x > y {
				return 1
			}
			return -1
		}
	}
	return 0
}

func cnbSetupURL(version string) string {
	return fmt.Sprintf("https://cnb.cool/%s/-/releases/download/V%s/%s",
		updateRepoCNB, version, fmt.Sprintf(updateAssetName, version))
}

func githubSetupURL(version string) string {
	return fmt.Sprintf("https://github.com/%s/releases/download/V%s/%s",
		updateRepoGitHub, version, fmt.Sprintf(updateAssetName, version))
}

func cnbReleasePage(version string) string {
	return fmt.Sprintf("https://cnb.cool/%s/-/releases/tag/V%s", updateRepoCNB, version)
}

func githubReleasePage(version string) string {
	return fmt.Sprintf("https://github.com/%s/releases/tag/V%s", updateRepoGitHub, version)
}

// allowedUpdateHost 限制应用内下载只能来自官方发布渠道,避免被诱导下载任意文件。
func allowedUpdateHost(rawURL string) bool {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Scheme != "https" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "cnb.cool" || strings.HasSuffix(host, ".cnb.cool") {
		return true
	}
	if host == "github.com" || strings.HasSuffix(host, ".github.com") {
		return true
	}
	if host == "githubusercontent.com" || strings.HasSuffix(host, ".githubusercontent.com") {
		return true
	}
	return false
}

// proxyForHost GitHub 走用户配置的代理(国内访问常用),CNB 直连。
func proxyForHost(rawURL, proxyURL string) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	if strings.Contains(strings.ToLower(u.Hostname()), "github") {
		return proxyURL
	}
	return ""
}

func newUpdateClient(proxyURL string, timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if proxyURL != "" {
		if u, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}
	return &http.Client{Transport: transport, Timeout: timeout}
}

func (a *App) proxySettings() (bool, string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.store.Settings.UseProxy, a.store.Settings.ProxyURL
}

// fetchCNBLatestVersion 读取 CNB 仓库 main 分支 wails.json 里的版本号。
func (a *App) fetchCNBLatestVersion() (string, error) {
	client := newUpdateClient("", 6*time.Second)
	resp, err := client.Get(fmt.Sprintf("https://cnb.cool/%s/-/git/raw/main/wails.json", updateRepoCNB))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	var payload struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	version := strings.TrimSpace(payload.Info.ProductVersion)
	if version == "" {
		return "", errors.New("wails.json 中没有 productVersion")
	}
	return version, nil
}

// fetchCNBReleaseNotes 取该版本的发布说明(docs/releases/<版本>.md),失败返回空串。
func (a *App) fetchCNBReleaseNotes(version string) string {
	client := newUpdateClient("", 6*time.Second)
	resp, err := client.Get(fmt.Sprintf("https://cnb.cool/%s/-/git/raw/main/docs/releases/%s.md", updateRepoCNB, version))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

// remoteFileExists 探测下载地址是否可用(跟随重定向,最终 200 视为存在)。
func (a *App) remoteFileExists(rawURL string) bool {
	client := newUpdateClient("", 8*time.Second)
	req, err := http.NewRequest(http.MethodHead, rawURL, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return true
		}
	}
	// 个别 CDN 对 HEAD 支持不佳,退一步用 GET 只读极小一段。
	req2, err2 := http.NewRequest(http.MethodGet, rawURL, nil)
	if err2 != nil {
		return false
	}
	req2.Header.Set("Range", "bytes=0-0")
	resp2, err2 := client.Do(req2)
	if err2 != nil {
		return false
	}
	defer resp2.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp2.Body, 1))
	return resp2.StatusCode == http.StatusOK || resp2.StatusCode == http.StatusPartialContent
}

type githubAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Body    string        `json:"body"`
	HTMLURL string        `json:"html_url"`
	Assets  []githubAsset `json:"assets"`
}

// githubLatestRelease 查询 GitHub 最新 Release(网络可达时使用;走代理更快)。
func (a *App) githubLatestRelease(proxyURL string) (version, notes, downloadURL, pageURL string, err error) {
	client := newUpdateClient(proxyURL, 8*time.Second)
	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", updateRepoGitHub), nil)
	if err != nil {
		return "", "", "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "DownTube-Updater")
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", "", "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", "", "", err
	}
	var payload githubRelease
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", "", "", "", err
	}
	version = strings.TrimPrefix(strings.TrimSpace(payload.TagName), "V")
	if version == "" {
		return "", "", "", "", errors.New("GitHub Release 没有标签名")
	}
	want := fmt.Sprintf(updateAssetName, version)
	for _, asset := range payload.Assets {
		if asset.Name == want {
			downloadURL = asset.URL
			break
		}
	}
	if downloadURL == "" {
		for _, asset := range payload.Assets {
			if strings.HasSuffix(strings.ToLower(asset.Name), ".exe") {
				downloadURL = asset.URL
				break
			}
		}
	}
	pageURL = payload.HTMLURL
	if pageURL == "" {
		pageURL = githubReleasePage(version)
	}
	return version, strings.TrimSpace(payload.Body), downloadURL, pageURL, nil
}

func updateResult(ok, hasUpdate bool, current, latest, notes, downloadURL, pageURL, source, errText string) map[string]any {
	return map[string]any{
		"ok":        ok,
		"hasUpdate": hasUpdate,
		"current":   current,
		"latest":    latest,
		"notes":     notes,
		"url":       downloadURL,
		"pageUrl":   pageURL,
		"source":    source,
		"error":     errText,
	}
}

// CheckUpdate 检查是否有新版本。current 由界面传入(wails.json 注入的那份)。
// 判定规则:CNB 版本号 > 当前版本,且对应安装包已挂到 Release 才算「可更新」;
// CNB 不通或安装包未就绪时,再问 GitHub。
func (a *App) CheckUpdate(current string) map[string]any {
	current = strings.TrimSpace(current)
	if current == "" {
		return updateResult(false, false, "", "", "", "", "", "", "无法确定当前版本")
	}
	useProxy, proxyURL := a.proxySettings()
	if !useProxy {
		proxyURL = ""
	}

	var cnbErr string
	if version, err := a.fetchCNBLatestVersion(); err == nil {
		if compareVersions(version, current) <= 0 {
			return updateResult(true, false, current, version, "", "", "", "cnb", "")
		}
		downloadURL := cnbSetupURL(version)
		if a.remoteFileExists(downloadURL) {
			return updateResult(true, true, current, version, a.fetchCNBReleaseNotes(version), downloadURL, cnbReleasePage(version), "cnb", "")
		}
		// 版本号已升但安装包还没挂上(发版构建进行中),尝试 GitHub。
		cnbErr = "新版本安装包尚未就绪"
	} else {
		cnbErr = err.Error()
	}

	if version, notes, downloadURL, pageURL, err := a.githubLatestRelease(proxyURL); err == nil {
		if compareVersions(version, current) <= 0 {
			return updateResult(true, false, current, version, "", "", "", "github", "")
		}
		if downloadURL == "" {
			downloadURL = githubSetupURL(version)
		}
		if pageURL == "" {
			pageURL = githubReleasePage(version)
		}
		return updateResult(true, true, current, version, notes, downloadURL, pageURL, "github", "")
	}
	return updateResult(false, false, current, "", "", "", "", "", "检查更新失败:"+cnbErr)
}

// DownloadUpdate 下载安装包到临时目录,进度通过 update:progress 事件上报。
// url 必须来自官方发布渠道(见 allowedUpdateHost);version 用于命名与校验。
// 通道不通时会自动尝试另一个官方源(CNB ↔ GitHub),国内直连 CNB 更快。
func (a *App) DownloadUpdate(rawURL, version string) map[string]any {
	version = strings.TrimSpace(version)
	if !updateVersionPattern.MatchString(version) {
		return map[string]any{"ok": false, "error": "版本号格式不正确"}
	}
	if !allowedUpdateHost(rawURL) {
		return map[string]any{"ok": false, "error": "不接受该下载地址"}
	}

	a.mu.Lock()
	if a.updateDownloading {
		a.mu.Unlock()
		return map[string]any{"ok": false, "error": "正在下载更新,请稍候"}
	}
	a.updateDownloading = true
	useProxy, proxyURL := a.store.Settings.UseProxy, a.store.Settings.ProxyURL
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.updateDownloading = false
		a.mu.Unlock()
	}()
	if !useProxy {
		proxyURL = ""
	}

	targetDir := filepath.Join(os.TempDir(), updateTempDir)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return map[string]any{"ok": false, "error": "无法创建临时目录:" + err.Error()}
	}
	dest := filepath.Join(targetDir, fmt.Sprintf(updateAssetName, version))

	// 已下载过且校验通过时直接复用(应用重启后不必重复下载)。
	if err := validateInstallerFile(dest); err == nil {
		info, _ := os.Stat(dest)
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		a.emitUpdateProgress("downloaded", size, size, "安装包已就绪")
		return map[string]any{"ok": true, "path": dest, "size": size, "reused": true}
	}

	// 主源 + 备用源:同一版本在两个平台都有安装包,一路失败自动换另一路。
	candidates := []string{rawURL}
	if strings.Contains(rawURL, "cnb.cool") {
		candidates = append(candidates, githubSetupURL(version))
	} else {
		candidates = append(candidates, cnbSetupURL(version))
	}

	var lastErr string
	for i, source := range candidates {
		if i > 0 {
			a.emitUpdateProgress("download", 0, 0, "主源不可用,正在切换备用源…")
		}
		_ = os.Remove(dest)
		if err := a.downloadUpdateFile(source, dest, proxyForHost(source, proxyURL)); err != nil {
			lastErr = err.Error()
			continue
		}
		if err := validateInstallerFile(dest); err != nil {
			lastErr = err.Error()
			continue
		}
		info, _ := os.Stat(dest)
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		a.emitUpdateProgress("downloaded", size, size, "下载完成")
		return map[string]any{"ok": true, "path": dest, "size": size}
	}
	_ = os.Remove(dest)
	return map[string]any{"ok": false, "error": "下载失败:" + lastErr + "(可稍后重试,或在设置中配置代理)"}
}

// downloadUpdateFile 单文件下载,每 200ms 上报一次进度。
func (a *App) downloadUpdateFile(rawURL, dest, proxyURL string) error {
	client := newUpdateClient(proxyURL, 0)
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "DownTube-Updater")
	a.emitUpdateProgress("download", 0, 0, "正在连接下载源…")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	total := resp.ContentLength
	buf := make([]byte, 1<<20)
	var received int64
	last := time.Now()
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				out.Close()
				return werr
			}
			received += int64(n)
			if time.Since(last) >= 200*time.Millisecond {
				message := fmt.Sprintf("已下载 %.1f MB", float64(received)/1048576)
				if total > 0 {
					message = fmt.Sprintf("已下载 %.1f / %.1f MB", float64(received)/1048576, float64(total)/1048576)
				}
				a.emitUpdateProgress("download", received, total, message)
				last = time.Now()
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			out.Close()
			return rerr
		}
	}
	if err := out.Close(); err != nil {
		return err
	}
	if total > 0 && received != total {
		return errors.New("下载不完整")
	}
	return nil
}

// validateInstallerFile 基本校验:存在、是 PE 可执行文件、体积合理。
func validateInstallerFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return errors.New("安装包不存在")
	}
	if info.Size() < 4<<20 {
		return errors.New("安装包体积异常,可能未下载完整")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	head := make([]byte, 2)
	if _, err := io.ReadFull(f, head); err != nil {
		return errors.New("安装包不可读")
	}
	if head[0] != 'M' || head[1] != 'Z' {
		return errors.New("安装包格式不正确")
	}
	return nil
}

// InstallUpdate 以管理员权限静默启动已下载的安装包(会弹一次 UAC 确认)。
// 安装程序随后会结束本进程、覆盖文件并在安装完成后重新拉起新版。
// 若用户在 UAC 中取消,这里会返回错误,应用保持运行不受影响。
func (a *App) InstallUpdate(version string) map[string]any {
	version = strings.TrimSpace(version)
	if !updateVersionPattern.MatchString(version) {
		return map[string]any{"ok": false, "error": "版本号格式不正确"}
	}
	path := filepath.Join(os.TempDir(), updateTempDir, fmt.Sprintf(updateAssetName, version))
	if err := validateInstallerFile(path); err != nil {
		return map[string]any{"ok": false, "error": "安装包已失效,请重新下载"}
	}
	if err := launchInstallerElevated(path); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	a.emitUpdateProgress("install", 0, 0, "安装程序已启动")
	// 安装程序已接手:稍候主动退出,让它顺利覆盖文件并重启新版。
	go func() {
		time.Sleep(1500 * time.Millisecond)
		if a.ctx != nil {
			runtime.Quit(a.ctx)
		}
	}()
	return map[string]any{"ok": true}
}

// OpenUpdatePage 打开发布页(网络受限或自动下载失败时的手动入口)。
func (a *App) OpenUpdatePage(rawURL string) {
	raw := strings.TrimSpace(rawURL)
	if !allowedUpdateHost(raw) {
		return
	}
	runtime.BrowserOpenURL(a.ctx, raw)
}

// emitUpdateProgress 广播下载/安装进度。
func (a *App) emitUpdateProgress(phase string, received, total int64, message string) {
	percent := 0
	if total > 0 {
		percent = int(received * 100 / total)
		if percent > 100 {
			percent = 100
		}
	}
	if phase == "downloaded" {
		percent = 100
	}
	if a.ctx == nil || a.disableEvents {
		return
	}
	runtime.EventsEmit(a.ctx, "update:progress", map[string]any{
		"phase": phase, "percent": percent, "message": message,
		"received": received, "total": total,
	})
}
