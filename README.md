# DownTube

Windows 桌面版 YouTube 下载工具。界面由 Wails + WebView2 提供，任务状态和下载进程由 Go 管理。

## 当前开发版

- Material 3 深色桌面界面，主色为 `#1296db`
- 本地任务队列：单条/批量链接、搜索、移除、清空、单项与顺序批量开始/暂停
- YouTube 链接校验和启动错误说明
- 下载设置和任务队列保存在 `%AppData%\DownTube\state.json`
- 画质、音频格式、帧率、代理、浏览器 Cookies 设置
- 原生目录选择器和 Explorer 文件夹打开
- `yt-dlp.exe` 随便携版提供；FFmpeg 由用户自行安装并加入系统 PATH
- 自动检测 yt-dlp、用户安装的 FFmpeg 和可选代理；高画质合并及 MP3 转换需要 FFmpeg
- WebView2 窗口使用本地打包的前端资源，无需启动浏览器服务器
- 关于页面介绍作者、技术组成和使用免责声明

便携目录包含 `DownTube.exe` 和 `yt-dlp.exe`。若需要合并高画质音视频或下载 MP3，请自行安装 FFmpeg 并将其加入系统 PATH。FFmpeg 未安装时，基础下载仍可使用。

## 环境

- Go 1.27+
- Node.js 18+
- Windows 10/11、WebView2 Runtime

## 构建并运行

```powershell
cd frontend
npm install
npm run build
cd ..
go mod tidy
go run -tags "desktop,dev" .
```

Wails 应用在 Windows 上必须包含 `desktop` 和 `dev` 构建标签；裸跑 `go run .` 或 `go build .` 会显示“Wails applications will not build without the correct build tags”提示框。也可以使用 Wails CLI：`wails dev`。发布构建使用 `wails build`。
