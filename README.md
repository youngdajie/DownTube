# DownTube

Windows 桌面版 YouTube 下载工具。界面由 Wails + WebView2 提供，任务状态和下载进程由 Go 管理。

## 功能

- 界面为扁平化专业工具风格:纯色表面、细边框、克制的圆角与阴影,不用渐变光晕和装饰性元素;主色 `#1296db` 只用于主按钮、激活态与进度等关键位置;支持深色/浅色双主题,标题栏一键切换并记住选择
- 下载页为两步式流程:粘贴来源(支持批量/单个模式、剪贴板粘贴)→ 队列下载(任务列表、搜索、单项与顺序批量开始/暂停、清空);链接解析完成后,队列有空闲并发位会自动开始下载
- **不止 YouTube**:下载内核是 yt-dlp,凡是它支持的站点都能尝试下载;下载页列出常见支持站点,并说明站点改版、地区限制或需要登录时可能解析失败,不保证每个链接都能下载
- **并发下载**:可同时下载多个任务(设置中可选 1–5 个,默认 3),任务行实时显示阶段——「下载视频中 / 下载音频中 / 合并中 / 转换音频中」
- **格式支持**:视频默认输出 H.264 + AAC 的 MP4(兼容性最好),可在设置中选择「H.265 · 下载后转码」获得 HEVC 文件;H.265 转码默认优先**显卡加速**(自动探测 NVIDIA NVENC / Intel QSV / AMD AMF,不可用或编码失败时回退 CPU),可在设置中改为纯 CPU 编码;转码按分段执行并支持**断点续转**——暂停或关闭程序后重新开始会复用已完成的分段,无需从头转码;拼接使用绝对路径并对特殊字符转义(文件名含 `#` 等符号也能正常产出成品),万一拼接仍失败会自动整段重新转码,不会卡在同一步反复失败;音频支持 MP3 与 FLAC(无损);「视频（带音频）」下载为合并后的单个文件,遇到同名无声旧文件会自动改名保留并重新下载合并
- **下载可靠性**:断点续传(保留 `.part` 文件)、网络层自动重试,任务失败后还会按设置自动重试(默认 2 次,退避间隔);重试使下载恢复推进后计数会归零,之后再失败从第 1 次重新计;常见错误会附带中文排查建议
- 设置页按分类组织(下载位置、画质与格式、网络与代理、Cookies、下载引擎),支持恢复默认 / 撤销 / 保存;下载引擎页可一键打开程序数据目录
- 本地任务队列保存在 `%AppData%\DownTube\state.json`,界面可切换深色/浅色,选择会记住
- 画质、音频格式、帧率、代理、浏览器 Cookies 设置;导入的 cookies.txt 会自动校验格式并修复常见问题(BOM、缺失文件头)
- **分站点 Cookies**:YouTube 与哔哩哔哩各存一份独立 Cookies,下载时按链接自动选用;从文件导入时自动识别站点并把对应部分存到 `%AppData%\DownTube\cookies`,不属于目标站点的行会被过滤掉(导出整份浏览器 Cookies 也不会串号);其他站点可继续用一份通用 cookies.txt 兜底
- 原生目录选择器和 Explorer 文件夹打开
- `yt-dlp.exe` 随便携版提供;FFmpeg 有三种方式配置:自动下载安装、从 GitHub 加速镜像下载 `.7z` 后一键导入(自动解压到程序目录并关联 bin 下的可执行文件)、手动选择已有的 `ffmpeg.exe`
- 自动检测 yt-dlp、FFmpeg 和可选代理;高画质合并及 MP3 转换需要 FFmpeg
- WebView2 窗口使用本地打包的前端资源,无需启动浏览器服务器
- 关于页面展示版本信息与作者链接
- **检查更新**:启动后静默检查一次(不打扰),也可在「关于」页手动检查;有新版本时展示版本号、更新说明与来源,导航栏「关于」会亮起提示标记
- **一键更新**:应用内直接下载安装包(CNB 优先、国内直连;失败自动切 GitHub 兜底),下载完点「立即安装」即结束程序、静默覆盖安装并自动重启到新版,全程无需手动下载和点下一步;已下载的安装包会复用,不必重复下载

便携目录包含 `DownTube.exe` 和 `yt-dlp.exe`。若需要合并高画质音视频或下载 MP3,可在应用内的 FFmpeg 安装弹窗中自动下载,或从弹窗提供的加速镜像下载 `.7z` 安装包后导入;FFmpeg 未安装时,基础下载仍可使用。

## 环境

- Go 1.27.2+（`go.mod` 已声明该最低版本；Go 1.21+ 会自动切换到修复后的工具链）
- Node.js 18+
- Windows 10/11、WebView2 Runtime

## 构建并运行

```powershell
cd frontend
npm install
npm run build      # 会先跑 tsc --noEmit，类型不过就不产出
cd ..
go mod tidy
go run -tags "desktop,dev" .
```

前端是 TypeScript（`src/main.ts`），类型定义从 Wails 生成的 `wailsjs/go/models.ts` 取，后端改字段名前端会立刻报错。单独跑类型检查用 `npm run typecheck`。

Wails 应用在 Windows 上必须包含 `desktop` 和 `dev` 构建标签；裸跑 `go run .` 或 `go build .` 会显示“Wails applications will not build without the correct build tags”提示框。也可以使用 Wails CLI：`wails dev`。发布构建使用 `wails build`。

打包安装包（需要 `makensis` 在 PATH 里）：

```bash
wails build -nsis -clean -platform windows/amd64
```

`build/windows/installer/project.nsi` 在 Wails 模板基础上加了三段逻辑，支撑应用内一键更新，改动时注意保留：安装前先 `taskkill` 结束正在运行的旧版本（并轮询等它真正退出，超时才强杀）；静默安装（`/S`）完成后自动重启新版；文件带 UTF-8 BOM，否则含中文注释会报 `Bad text encoding`。

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
4. 推标签（两个平台各自触发自动发版）：`git tag -a V1.3.0 -m "1.3.0" && git push origin refs/tags/V1.3.0 && git push github refs/tags/V1.3.0`

标签格式必须是 `V` + `wails.json` 里的版本号，对不上流水线会直接红掉 ——
这样就不会出现「安装包是 0.2.0、Release 却挂在 V0.3.0 上」的情况。

之后两条流水线各自自动跑完：交叉编译 → 打 NSIS 安装包 → 在本平台建 Release（绑定同一个 `V<版本号>` 标签）→ 挂 `DownTube-Setup-<版本号>.exe` → 删掉历史 Release 回收存储额度。CNB 侧配置在 `.cnb.yml`，GitHub 侧配置在 `.github/workflows/release.yml`；GitHub 侧还可以用 `gh workflow run release.yml` 手动触发一次「只构建、不上架」的健康检查。

### 安装包命名不能随意改

应用内的检查更新按固定规则拼地址，改命名会直接让老版本用户收不到更新：

| 用途 | 地址 |
| --- | --- |
| 版本号来源（CNB） | `/-/git/raw/main/wails.json` 里的 `info.productVersion` |
| 更新说明（CNB） | `/-/git/raw/main/docs/releases/<版本号>.md` |
| 安装包（CNB 主源） | `/-/releases/download/V<版本号>/DownTube-Setup-<版本号>.exe` |
| 版本号 / 说明 / 安装包（GitHub 备源） | `releases/latest` 接口 + 同名附件 |

所以：安装包文件名必须保持 `DownTube-Setup-<版本号>.exe`；`docs/releases/<版本号>.md` 要在打标签前提交到 `main`（它就是更新弹窗里显示的说明）；版本号本身必须比老版本大，否则检测逻辑判定「已是最新」。应用内下载只允许来自 `cnb.cool` / `github.com` / `githubusercontent.com` 的 https 地址，白名单之外一律拒绝。

标签推上去之后本地删不掉（`cnb git delete-tag` 需要 `repo-release:rw`，本地令牌没有，流水线的可信令牌才有）。这一版流水线跑挂了想重来，两条路：

- 只是流水线/CI 的问题、应用代码没动：把标签挪到新提交再强推（`git tag -f V0.2.0 <sha>` + `git push -f`）
- 应用代码也改了：那就按新版本号推一个新标签

Release 附件只放流水线自己构建出来的安装包，不当网盘用。
