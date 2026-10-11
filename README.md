# DownTube

Windows 桌面版 YouTube 下载工具。界面由 Wails + WebView2 提供，任务状态和下载进程由 Go 管理。

## 功能

- 界面为扁平化专业工具风格:纯色表面、细边框、克制的圆角与阴影,不用渐变光晕和装饰性元素;主色 `#1296db` 只用于主按钮、激活态与进度等关键位置;支持深色/浅色双主题,标题栏一键切换并记住选择
- 下载页为两步式流程:粘贴来源(支持批量/单个模式、剪贴板粘贴)→ 队列下载(任务列表、搜索、单项与顺序批量开始/暂停、清空)
- **并发下载**:可同时下载多个任务(设置中可选 1–5 个,默认 3),任务行实时显示阶段——「下载视频中 / 下载音频中 / 合并中 / 转换音频中」
- **格式支持**:视频默认输出 H.264 + AAC 的 MP4(兼容性最好),可在设置中选择「H.265 · 下载后转码」获得 HEVC 文件;音频支持 MP3 与 FLAC(无损)
- **下载可靠性**:断点续传(保留 `.part` 文件)、网络层自动重试,任务失败后还会按设置自动重试(默认 2 次,退避间隔);常见错误会附带中文排查建议
- 设置页按分类组织(下载位置、画质与格式、网络与代理、Cookies、下载引擎),支持恢复默认 / 撤销 / 保存;下载引擎页可一键打开程序数据目录
- 本地任务队列保存在 `%AppData%\DownTube\state.json`,界面可切换深色/浅色,选择会记住
- 画质、音频格式、帧率、代理、浏览器 Cookies 设置;导入的 cookies.txt 会自动校验格式并修复常见问题(BOM、缺失文件头)
- 原生目录选择器和 Explorer 文件夹打开
- `yt-dlp.exe` 随便携版提供;FFmpeg 有三种方式配置:自动下载安装、从 GitHub 加速镜像下载 `.7z` 后一键导入(自动解压到程序目录并关联 bin 下的可执行文件)、手动选择已有的 `ffmpeg.exe`
- 自动检测 yt-dlp、FFmpeg 和可选代理;高画质合并及 MP3 转换需要 FFmpeg
- WebView2 窗口使用本地打包的前端资源,无需启动浏览器服务器
- 关于页面展示版本信息与作者链接

便携目录包含 `DownTube.exe` 和 `yt-dlp.exe`。若需要合并高画质音视频或下载 MP3,可在应用内的 FFmpeg 安装弹窗中自动下载,或从弹窗提供的加速镜像下载 `.7z` 安装包后导入;FFmpeg 未安装时,基础下载仍可使用。

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
| tag 与安装包名 | `.cnb.yml` 与 `.github/workflows/release.yml` 读取后拼成 `V<版本号>` |

发一个新版本：

1. 改 `wails.json` 的 `info.productVersion`（顺手同步 `frontend/package.json` 的 `version`）
2. 版本说明写进 `docs/releases/<版本号>.md`（Release 页的正文就是它，不写就退回 tag 附注）
3. 提交并推送到 GitHub 与 CNB
4. 推标签（两个平台各自触发自动发版）：`git tag -a V1.2.0 -m "1.2.0" && git push origin refs/tags/V1.2.0 && git push github refs/tags/V1.2.0`

标签格式必须是 `V` + `wails.json` 里的版本号，对不上流水线会直接红掉 ——
这样就不会出现「安装包是 0.2.0、Release 却挂在 V0.3.0 上」的情况。

之后两条流水线各自自动跑完：交叉编译 → 打 NSIS 安装包 → 在本平台建 Release（绑定同一个 `V<版本号>` 标签）→ 挂 `DownTube-Setup-<版本号>-<编号>.exe` → 删掉历史 Release 回收存储额度。CNB 侧配置在 `.cnb.yml`，GitHub 侧配置在 `.github/workflows/release.yml`；GitHub 侧还可以用 `gh workflow run release.yml` 手动触发一次「只构建、不上架」的健康检查。

标签推上去之后本地删不掉（`cnb git delete-tag` 需要 `repo-release:rw`，本地令牌没有，流水线的可信令牌才有）。这一版流水线跑挂了想重来，两条路：

- 只是流水线/CI 的问题、应用代码没动：把标签挪到新提交再强推（`git tag -f V0.2.0 <sha>` + `git push -f`）
- 应用代码也改了：那就按新版本号推一个新标签

Release 附件只放流水线自己构建出来的安装包，不当网盘用。
