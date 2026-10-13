package main

// Wails 会在构建时把这些导出结构体生成到 frontend/wailsjs/go/models.ts,
// 前端由此拿到真实字段类型,不再是一堆 Record<string, any>。
//
// 这些类型是从原先的 map[string]any 返回值逐个反推出来的,**json tag 与旧键名一一对应**。
// 改 tag 等于改前后端契约:界面上不会报错,只会静默变成 undefined。
// contracts_test.go 把每种返回形状的键集锁死,改坏了会立刻失败。

// UpdateResult CheckUpdate 的返回。updateResult 辅助函数总是返回全部 9 个键。
type UpdateResult struct {
	OK        bool   `json:"ok"`
	HasUpdate bool   `json:"hasUpdate"`
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Notes     string `json:"notes"`
	URL       string `json:"url"`
	PageURL   string `json:"pageUrl"`
	Source    string `json:"source"`
	Error     string `json:"error"`
}

// UpdateDownloadResult DownloadUpdate 的返回
type UpdateDownloadResult struct {
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	Path   string `json:"path,omitempty"`
	Size   int64  `json:"size,omitempty"`
	Reused bool   `json:"reused,omitempty"`
}

// UpdateInstallResult InstallUpdate 的返回
type UpdateInstallResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// SystemStatus GetSystemStatus 的返回
type SystemStatus struct {
	EngineReady   bool   `json:"engineReady"`
	EnginePath    string `json:"enginePath"`
	EngineError   string `json:"engineError"`
	FFmpegReady   bool   `json:"ffmpegReady"`
	FFmpegPath    string `json:"ffmpegPath"`
	FFmpegError   string `json:"ffmpegError"`
	FFmpegVersion string `json:"ffmpegVersion"`
	ProxyChecked  bool   `json:"proxyChecked"`
	ProxyReady    bool   `json:"proxyReady"`
	ProxyMessage  string `json:"proxyMessage"`
	DataDir       string `json:"dataDir"`
}

// EncoderStatus GetHEVCEncoderStatus 的返回。
// 未检测到显卡编码器时 encoder/label 为空,未安装 FFmpeg 时 message 为空。
type EncoderStatus struct {
	OK      bool   `json:"ok"`
	GPU     bool   `json:"gpu"`
	Encoder string `json:"encoder,omitempty"`
	Label   string `json:"label,omitempty"`
	Message string `json:"message,omitempty"`
}

// ProxyTestResult TestProxy 的返回
type ProxyTestResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// FFmpegInstallResult InstallFFmpeg 与 ImportFFmpegArchive 的返回
type FFmpegInstallResult struct {
	OK       bool   `json:"ok"`
	Message  string `json:"message,omitempty"`
	Path     string `json:"path,omitempty"`
	Version  string `json:"version,omitempty"`
	Canceled bool   `json:"canceled,omitempty"`
}

// FilePickResult SelectFFmpegFile 的返回
type FilePickResult struct {
	OK       bool   `json:"ok"`
	Message  string `json:"message,omitempty"`
	Path     string `json:"path,omitempty"`
	Version  string `json:"version,omitempty"`
	Canceled bool   `json:"canceled,omitempty"`
}

// CookiePickResult SelectCookieFile 的返回。
//
// 取消分支原本只返回 {"canceled":true}(没有 ok 键),现在会多出一个 ok:false。
// 前端是先判 canceled 再判 ok,所以行为不变;这里选「始终带上 ok」是因为
// 让 TS 拿到 ok: boolean 而不是 ok?: boolean,契约更好读。
type CookiePickResult struct {
	OK       bool   `json:"ok"`
	Message  string `json:"message,omitempty"`
	Canceled bool   `json:"canceled,omitempty"`
	Path     string `json:"path,omitempty"`
	Note     string `json:"note,omitempty"`
}

// CookiePlatformStatus CookieStatus 里单个站点的状态
type CookiePlatformStatus struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	File      string `json:"file"`
	Imported  bool   `json:"imported"`
	Count     int    `json:"count"`
	UpdatedAt string `json:"updatedAt"`
}

// CookieStatusResult CookieStatus 的返回
type CookieStatusResult struct {
	Dir       string                 `json:"dir"`
	Platforms []CookiePlatformStatus `json:"platforms"`
	Error     string                 `json:"error,omitempty"`
}

// CookieImportEntry 导入成功后的逐站点明细(saved 数组的元素)
type CookieImportEntry struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// CookieImportResult ImportPlatformCookie 的返回。
// 与 CookiePickResult 同理,取消分支原本没有 ok 键,现在会补上 ok:false —— 前端先判 canceled,行为不变。
type CookieImportResult struct {
	OK       bool               `json:"ok"`
	Message  string             `json:"message,omitempty"`
	Canceled bool               `json:"canceled,omitempty"`
	Saved    []CookieImportEntry `json:"saved,omitempty"`
	Dropped  int                `json:"dropped,omitempty"`
	Dir      string             `json:"dir,omitempty"`
}

// CookieRemoveResult RemovePlatformCookie 的返回
type CookieRemoveResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
}
