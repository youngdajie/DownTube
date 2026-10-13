package main

// 前后端之间的「线上格式」契约。**这个文件不是前端迁移的遗留物,别顺手删。**
//
// 背景:后端过去用 map[string]any 拼返回值,1.3.0 起改成 contracts.go 里的结构体。
// 当时写这些用例是为了证明改造没改变 JSON 形状 —— 但它们的价值不止于此。
//
// 现在它守的是 **TypeScript 实际挡不住的那一类问题**(已实测,不是推测):
//
//   - Wails 会把 omitempty 字段生成成可选类型(`error?: string`),看着像个防线,
//     但前端写的是 `state.updateError = result?.error || '检查更新失败'` ——
//     那个 `||` 兜底把 undefined 咽掉了,于是 tsc 一声不吭地放行。
//     实测过程:给 UpdateResult.Error 加 omitempty → 重新生成 → tsc 报 0 错。
//   - 但**运行时的行为变了**:带 omitempty 且值为空时,这个键会从 JSON 里整个消失,
//     前端拿到 undefined,于是用户看到的是兜底文案而不是真实原因。
//   - 所以哪些字段必须常驻是刻意区分的,不是随手写的:
//     UpdateResult.Error 常驻(前端拼提示要用);
//     UpdateDownloadResult.Error 带 omitempty(只有失败时才有意义)。
//     谁"顺手统一一下"加个 omitempty,就会踩到上面那个坑。
//
// 实测:给 UpdateResult.Error 加 omitempty,TestContractKeysUnchanged 立刻报
//   「旧: [... error ...] / 新: [... 无 error ...]」,指向性很清楚。
//
// 所以:改了响应结构体、或改了 json tag,这些用例失败是**预期行为** ——
// 先确认这次改动是有意的,再更新下面的期望值。

import (
	"encoding/json"
	"sort"
	"testing"
)

// jsonKeys 返回 marshaled JSON 的顶层键集合(排序后)。
func jsonKeys(t *testing.T, v any) []string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func assertSameKeys(t *testing.T, label string, oldShape map[string]any, newShape any) {
	t.Helper()
	oldKeys := jsonKeys(t, oldShape)
	newKeys := jsonKeys(t, newShape)
	if len(oldKeys) != len(newKeys) {
		t.Errorf("%s 键集不一致\n  旧: %v\n  新: %v", label, oldKeys, newKeys)
		return
	}
	for i := range oldKeys {
		if oldKeys[i] != newKeys[i] {
			t.Errorf("%s 键集不一致\n  旧: %v\n  新: %v", label, oldKeys, newKeys)
			return
		}
	}
}

// TestContractKeysUnchanged 把「旧代码里的 map 字面量」与「新结构体」逐个对照。
//
// 这是本次改动的核心保证:json tag 只要写错一个字母,界面不会报错,
// 只会静默变成 undefined —— 所以必须在编译期用测试挡住。
// 下面的 map 字面量是从旧实现逐个 return 语句抄下来的,不是从新代码反推的。
func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestContractKeysUnchanged(t *testing.T) {
	cases := []struct {
		name string
		old  map[string]any
		new  any
	}{
		{
			name: "CheckUpdate / updateResult",
			old: map[string]any{
				"ok": true, "hasUpdate": true, "current": "1.2.8", "latest": "1.2.9",
				"notes": "n", "url": "u", "pageUrl": "p", "source": "cnb", "error": "",
			},
			new: UpdateResult{OK: true, HasUpdate: true, Current: "1.2.8", Latest: "1.2.9",
				Notes: "n", URL: "u", PageURL: "p", Source: "cnb"},
		},
		{
			name: "GetSystemStatus",
			old: map[string]any{
				"engineReady": true, "enginePath": "e", "engineError": "", "ffmpegReady": true,
				"ffmpegPath": "f", "ffmpegError": "", "ffmpegVersion": "v",
				"proxyChecked": true, "proxyReady": false, "proxyMessage": "m", "dataDir": "d",
			},
			new: SystemStatus{EngineReady: true, EnginePath: "e", FFmpegReady: true, FFmpegPath: "f",
				FFmpegVersion: "v", ProxyChecked: true, ProxyReady: false, ProxyMessage: "m", DataDir: "d"},
		},
		{
			name: "GetHEVCEncoderStatus(显卡可用)",
			old:  map[string]any{"ok": true, "gpu": true, "encoder": "nvenc", "label": "L"},
			new:  EncoderStatus{OK: true, GPU: true, Encoder: "nvenc", Label: "L"},
		},
		{
			name: "GetHEVCEncoderStatus(回退 CPU)",
			old:  map[string]any{"ok": true, "gpu": false, "encoder": "cpu", "label": "L", "message": "m"},
			new:  EncoderStatus{OK: true, GPU: false, Encoder: "cpu", Label: "L", Message: "m"},
		},
		{
			name: "DownloadUpdate(成功)",
			old:  map[string]any{"ok": true, "path": "p", "size": 123, "reused": true},
			new:  UpdateDownloadResult{OK: true, Path: "p", Size: 123, Reused: true},
		},
		{
			name: "DownloadUpdate(复用已有)",
			old:  map[string]any{"ok": true, "path": "p", "size": 123},
			new:  UpdateDownloadResult{OK: true, Path: "p", Size: 123},
		},
		{
			name: "DownloadUpdate(失败)",
			old:  map[string]any{"ok": false, "error": "e"},
			new:  UpdateDownloadResult{Error: "e"},
		},
		{
			name: "InstallUpdate(成功)",
			old:  map[string]any{"ok": true},
			new:  UpdateInstallResult{OK: true},
		},
		{
			name: "InstallUpdate(失败)",
			old:  map[string]any{"ok": false, "error": "e"},
			new:  UpdateInstallResult{Error: "e"},
		},
		{
			name: "TestProxy",
			old:  map[string]any{"ok": true, "message": "m"},
			new:  ProxyTestResult{OK: true, Message: "m"},
		},
		{
			name: "InstallFFmpeg(成功)",
			old:  map[string]any{"ok": true, "path": "p", "version": "v"},
			new:  FFmpegInstallResult{OK: true, Path: "p", Version: "v"},
		},
		{
			name: "ImportFFmpegArchive(取消)",
			old:  map[string]any{"ok": false, "canceled": true},
			new:  FFmpegInstallResult{Canceled: true},
		},
		{
			name: "InstallFFmpeg(失败)",
			old:  map[string]any{"ok": false, "message": "m"},
			new:  FFmpegInstallResult{Message: "m"},
		},
		{
			name: "SelectFFmpegFile(成功)",
			old:  map[string]any{"ok": true, "path": "p", "version": "v"},
			new:  FilePickResult{OK: true, Path: "p", Version: "v"},
		},
		{
			name: "SelectFFmpegFile(取消)",
			old:  map[string]any{"ok": false, "canceled": true},
			new:  FilePickResult{Canceled: true},
		},
		{
			name: "SelectCookieFile(成功)",
			old:  map[string]any{"ok": true, "path": "p", "note": "n"},
			new:  CookiePickResult{OK: true, Path: "p", Note: "n"},
		},
		{
			name: "SelectCookieFile(失败)",
			old:  map[string]any{"ok": false, "message": "m"},
			new:  CookiePickResult{Message: "m"},
		},
		{
			name: "CookieStatus",
			old: map[string]any{"dir": "d", "platforms": []map[string]any{
				{"id": "youtube", "name": "YouTube", "file": "youtube.txt", "imported": true, "count": 12, "updatedAt": "2026-10-13 10:00"},
			}},
			new: CookieStatusResult{Dir: "d", Platforms: []CookiePlatformStatus{
				{ID: "youtube", Name: "YouTube", File: "youtube.txt", Imported: true, Count: 12, UpdatedAt: "2026-10-13 10:00"},
			}},
		},
		{
			name: "CookieStatus(取目录失败,多 error 键)",
			old:  map[string]any{"dir": "", "platforms": []map[string]any{}, "error": "e"},
			new:  CookieStatusResult{Platforms: []CookiePlatformStatus{}, Error: "e"},
		},
		{
			name: "ImportPlatformCookie(成功)",
			old: map[string]any{"ok": true,
				"saved":   []map[string]any{{"id": "youtube", "name": "YouTube", "count": 12}},
				"dropped": 3, "dir": "d"},
			new: CookieImportResult{OK: true,
				Saved:   []CookieImportEntry{{ID: "youtube", Name: "YouTube", Count: 12}},
				Dropped: 3, Dir: "d"},
		},
		{
			name: "ImportPlatformCookie(失败)",
			old:  map[string]any{"ok": false, "message": "m"},
			new:  CookieImportResult{Message: "m"},
		},
		{
			name: "RemovePlatformCookie(成功)",
			old:  map[string]any{"ok": true, "id": "youtube", "name": "YouTube"},
			new:  CookieRemoveResult{OK: true, ID: "youtube", Name: "YouTube"},
		},
		{
			name: "RemovePlatformCookie(失败)",
			old:  map[string]any{"ok": false, "message": "m"},
			new:  CookieRemoveResult{Message: "m"},
		},
	}
	for _, c := range cases {
		assertSameKeys(t, c.name, c.old, c.new)
	}
}

// TestContractNestedPlatformKeys 单独锁 platforms / saved 两个嵌套数组的元素键。
// 顶层键对了但元素键写错的情况,靠上面那个用例是发现不了的。
func TestContractNestedPlatformKeys(t *testing.T) {
	platformOld := map[string]any{
		"id": "bilibili", "name": "哔哩哔哩", "file": "bilibili.txt",
		"imported": false, "count": 0, "updatedAt": "",
	}
	platformNew := CookiePlatformStatus{ID: "bilibili", Name: "哔哩哔哩", File: "bilibili.txt"}
	assertSameKeys(t, "CookiePlatformStatus", platformOld, platformNew)

	entryOld := map[string]any{"id": "bilibili", "name": "哔哩哔哩", "count": 88}
	entryNew := CookieImportEntry{ID: "bilibili", Name: "哔哩哔哩", Count: 88}
	assertSameKeys(t, "CookieImportEntry", entryOld, entryNew)
}

// TestContractCanceledBranchesAddOK 记录一处**有意为之**的形状变化:
// 取消分支原本是 {"canceled":true}(没有 ok 键),结构化之后会多出 ok:false。
// 前端是先判 canceled 再判 ok,所以行为不变;这个用例把这个差异钉死,
// 避免以后有人"顺手修好"它时无声地改变行为。
func TestContractCanceledBranchesAddOK(t *testing.T) {
	for _, name := range []string{"CookiePickResult", "CookieImportResult"} {
		var got []string
		switch name {
		case "CookiePickResult":
			got = jsonKeys(t, CookiePickResult{Canceled: true})
		case "CookieImportResult":
			got = jsonKeys(t, CookieImportResult{Canceled: true})
		}
		found := map[string]bool{}
		for _, k := range got {
			found[k] = true
		}
		if !found["canceled"] {
			t.Errorf("%s 取消分支必须带 canceled 键,实际 %v", name, got)
		}
		if !found["ok"] {
			t.Errorf("%s 取消分支现在会带上 ok:false(有意变化),实际 %v", name, got)
		}
	}
}

// TestSystemStatusLiveFields 对真实方法做一次序列化核对。
// GetSystemStatus 里手改过一处:原先从 TestProxy 的 map 取 ok/message,
// 结构化之后改成读字段。这条用例确保那条路径没被改坏。
func TestSystemStatusLiveFields(t *testing.T) {
	app := &App{}
	status := app.GetSystemStatus()
	raw, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}
	want := []string{"dataDir", "engineError", "enginePath", "engineReady", "ffmpegError",
		"ffmpegPath", "ffmpegReady", "ffmpegVersion", "proxyChecked", "proxyMessage", "proxyReady"}
	if len(got) != len(want) {
		t.Fatalf("键数不对\n  期望 %d 个: %v\n  实际 %d 个: %v", len(want), want, len(got), keysOf(got))
	}
	for _, k := range want {
		if _, ok := got[k]; !ok {
			t.Errorf("缺少键 %q", k)
		}
	}
	// 代理检测关闭时,这两个字段必须是零值而不是缺失 —— 前端会直接读它们
	if got["proxyChecked"] != false {
		t.Errorf("proxyChecked 应为 false,得到 %v", got["proxyChecked"])
	}
	if got["proxyMessage"] != "代理检测已关闭" {
		t.Errorf("proxyMessage 文案不符,得到 %v", got["proxyMessage"])
	}
}

// TestProxyLiveResult 单独确认 TestProxy 的结构化结果。
func TestProxyLiveResult(t *testing.T) {
	app := &App{}
	if res := app.TestProxy("   "); res.OK || res.Message == "" {
		t.Errorf("空地址应返回 ok=false 且带提示,得到 %+v", res)
	}
	if res := app.TestProxy("http://127.0.0.1:1"); res.OK || res.Message == "" {
		t.Errorf("不可连地址应返回 ok=false 且带提示,得到 %+v", res)
	}
}
