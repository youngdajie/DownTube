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

打包安装包（需要 `makensis` 在 PATH 里）：

```bash
wails build -nsis -clean -platform windows/amd64
```

## 版本号与发版

版本号只在 `wails.json` 的 `info.productVersion` 里写一次，三处都由它推导：

| 消费方 | 说明 |
| --- | --- |
| exe 版本资源 | wails 构建时写进 PE 资源，资源管理器属性页看到的版本 |
| 界面上的版本 | `frontend/vite.config.js` 通过 `define` 注入 `__APP_VERSION__` |
| tag 与安装包名 | `.cnb.yml` 流水线读取后拼成 `V<版本号>` |

发一个新版本：

1. 改 `wails.json` 的 `info.productVersion`（顺手同步 `frontend/package.json` 的 `version`）
2. 版本说明写进 `docs/releases/<版本号>.md`（Release 页的正文就是它，不写就退回 tag 附注）
3. 提交并推送到 GitHub 与 CNB
4. 推标签：`git tag -a V0.2.0 -m "0.2.0" && git push cnb refs/tags/V0.2.0`

标签格式必须是 `V` + `wails.json` 里的版本号，对不上流水线会直接红掉 ——
这样就不会出现「安装包是 0.2.0、Release 却挂在 V0.3.0 上」的情况。

之后 CNB 流水线自动跑完：交叉编译 → 打 NSIS 安装包 → 建 Release（绑定 `V<版本号>` 标签）→ 挂 `DownTube-Setup-<版本号>-<流水线编号>.exe` → 删掉历史 Release 回收存储额度。

标签推上去之后本地删不掉（`cnb git delete-tag` 需要 `repo-release:rw`，本地令牌没有，流水线的可信令牌才有）。这一版流水线跑挂了想重来，两条路：

- 只是流水线/CI 的问题、应用代码没动：把标签挪到新提交再强推（`git tag -f V0.2.0 <sha>` + `git push -f`）
- 应用代码也改了：那就按新版本号推一个新标签

Release 附件只放流水线自己构建出来的安装包，不当网盘用。
