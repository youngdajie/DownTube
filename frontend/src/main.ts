import './style.css'
import './animations.css'

// ============================================================
// 图标:统一线性描边风格,颜色继承自 CSS currentColor
// ============================================================
const ICON_PATHS: Record<string, string> = {
  download: '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="m7 10 5 5 5-5"/><path d="M12 15V3"/>',
  settings: '<circle cx="12" cy="12" r="3"/><path d="m19.4 15 .1.1a2 2 0 1 1-2.8 2.8l-.1-.1a2 2 0 0 0-3.4 1.4v.3a2 2 0 1 1-4 0v-.2A2 2 0 0 0 5.8 17l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a2 2 0 0 0-1.4-3.4h-.3a2 2 0 1 1 0-4h.2A2 2 0 0 0 3 4.2l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a2 2 0 0 0 3.4-1.4v-.3a2 2 0 1 1 4 0v.2A2 2 0 0 0 16.6 1l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a2 2 0 0 0 1.4 3.4h.3a2 2 0 1 1 0 4h-.2a2 2 0 0 0-1.5 3.8Z" transform="translate(2 2) scale(.83)"/>',
  log: '<path d="M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  folder: '<path d="M3 7a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>',
  link: '<path d="M10 13a5 5 0 0 0 7.1 0l3-3A5 5 0 0 0 13 2.9l-1.7 1.7"/><path d="M14 11a5 5 0 0 0-7.1 0l-3 3A5 5 0 0 0 11 21.1l1.7-1.7"/>',
  play: '<path d="m7 4 14 8-14 8z"/>',
  pause: '<path d="M7 4h3v16H7zM14 4h3v16h-3z"/>',
  check: '<path d="m5 12 4 4L19 6"/>',
  x: '<path d="m18 6-12 12M6 6l12 12"/>',
  search: '<circle cx="11" cy="11" r="7"/><path d="m20 20-4-4"/>',
  external: '<path d="M15 3h6v6M10 14 21 3"/><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>',
  shield: '<path d="M12 22s8-4 8-11V5l-8-3-8 3v6c0 7 8 11 8 11z"/><path d="m9 12 2 2 4-4"/>',
  clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
  trash: '<path d="M3 6h18M8 6V4h8v2m3 0-1 14H6L5 6m4 4v6m6-6v6"/>',
  video: '<rect x="3" y="5" width="18" height="14" rx="2"/><path d="m10 9 5 3-5 3z"/>',
  arrow: '<path d="M5 12h14m-6-6 6 6-6 6"/>',
  chevron: '<path d="m9 18 6-6-6-6"/>',
  globe: '<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3a15 15 0 0 1 0 18M12 3a15 15 0 0 0 0 18"/>',
  checkcircle: '<circle cx="12" cy="12" r="9"/><path d="m8 12 3 3 5-6"/>',
  minimize: '<path d="M5 12h14"/>',
  maximize: '<rect x="5" y="5" width="14" height="14" rx="1"/>',
  restore: '<path d="M8 8V5h11v11h-3M5 8h11v11H5z"/>',
  info: '<circle cx="12" cy="12" r="9"/><path d="M12 11v5m0-8h.01"/>',
  code: '<path d="m8 8-4 4 4 4m8-8 4 4-4 4m-3-11-2 14"/>',
  layers: '<path d="m12 3 9 5-9 5-9-5 9-5Z"/><path d="m3 12 9 5 9-5M3 16l9 5 9-5"/>',
  cpu: '<rect x="5" y="5" width="14" height="14" rx="2"/><path d="M9 9h6v6H9zM9 1v4m6-4v4M9 19v4m6-4v4M1 9h4m-4 6h4m14-6h4m-4 6h4"/>',
  file: '<path d="M14 3v5h5"/><path d="M6 2h8l6 6v12a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2z"/>',
  bilibili: '<rect x="3" y="7" width="18" height="12" rx="3"/><path d="m7 3.5 2.8 3.5M17 3.5l-2.8 3.5"/><path d="M8.8 12.4v2.2M15.2 12.4v2.2"/>',
  github: '<path d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4"/><path d="M9 18c-4.51 2-5-2-7-2"/>',
  sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M4.93 4.93l1.41 1.41m11.32 11.32 1.41 1.41M2 12h2m16 0h2M4.93 19.07l1.41-1.41m11.32-11.32 1.41-1.41"/>',
  moon: '<path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>',
  clipboard: '<rect x="8" y="2" width="8" height="4" rx="1"/><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/>',
  package: '<path d="M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z"/><path d="m3.3 7 8.7 5 8.7-5"/><path d="M12 22V12"/>',
  cloudDown: '<path d="M12 13v8"/><path d="m8 17 4 4 4-4"/><path d="M4.4 14.9A7 7 0 1 1 15.7 8h1.8a4.5 4.5 0 0 1 2.5 8.2"/>',
  rotateCcw: '<path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/>',
  undo: '<path d="M9 14 4 9l5-5"/><path d="M4 9h10.5a5.5 5.5 0 0 1 0 11H11"/>',
  save: '<path d="M15.2 3a2 2 0 0 1 1.4.6l3.8 3.8a2 2 0 0 1 .6 1.4V19a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2z"/><path d="M17 21v-7a1 1 0 0 0-1-1H8a1 1 0 0 0-1 1v7"/><path d="M7 3v4a1 1 0 0 0 1 1h7"/>',
  sliders: '<path d="M4 21v-7M4 10V3M12 21v-9M12 8V3M20 21v-5M20 12V3M2 14h4M10 8h4M18 16h4"/>',
  key: '<path d="m21 2-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0 3 3L22 7l-3-3"/>',
  toolbox: '<rect x="3" y="8" width="18" height="12" rx="2"/><path d="M7 8V6a2 2 0 0 1 2-2h6a2 2 0 0 1 2 2v2"/><path d="M3 13h18"/>',
  broadcast: '<path d="M4.9 19.1a10 10 0 0 1 0-14.2"/><path d="M7.8 16.2a6 6 0 0 1 0-8.4"/><path d="M16.2 7.8a6 6 0 0 1 0 8.4"/><path d="M19.1 4.9a10 10 0 0 1 0 14.2"/><circle cx="12" cy="12" r="1.5"/>',
}
const icon = (name: string, size = 17) =>
  `<svg width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${ICON_PATHS[name] || ICON_PATHS.info}</svg>`

// 构建时由 vite.config.js 从 wails.json 注入,界面上不要写死版本号。
const APP_VERSION = __APP_VERSION__

// FFmpeg 在线下载镜像(与安装弹窗一一对应)
const FFMPEG_MIRRORS = [
  {
    id: 'ghfast',
    name: 'GitHub 加速 · GHFast',
    desc: '通过开发者提供的链接下载',
    file: 'ffmpeg-9.0.2-essentials_build.7z',
    meta: ['Windows x64', '7Z', '约 34 MB'],
    url: 'https://ghfast.top/github.com/GyanD/codexffmpeg/releases/download/9.0.2/ffmpeg-9.0.2-essentials_build.7z',
  },
  {
    id: 'ghproxy',
    name: 'GitHub 加速 · GH-Proxy',
    desc: '通过开发者提供的链接下载',
    file: 'ffmpeg-9.0.2-essentials_build.7z',
    meta: ['Windows x64', '7Z', '约 34 MB'],
    url: 'https://gh-proxy.com/https://github.com/GyanD/codexffmpeg/releases/download/9.0.2/ffmpeg-9.0.2-essentials_build.7z',
  },
]

const SETTINGS_SECTIONS = [
  { id: 'location', icon: 'folder', title: '下载位置', sub: '保存目录与完成后行为', desc: '视频文件保存到哪里，以及下载完成后要做什么' },
  { id: 'format', icon: 'sliders', title: '画质与格式', sub: '默认清晰度与内容类型', desc: '默认下载的视频清晰度、内容类型与帧率偏好' },
  { id: 'network', icon: 'globe', title: '网络与代理', sub: '代理连接与连通性检测', desc: '通过代理访问站点，并在启动时检测可用性' },
  { id: 'cookies', icon: 'key', title: 'Cookies', sub: '浏览器验证与文件导入', desc: '下载需要登录或年龄验证的视频时，提供 Cookies' },
  { id: 'engine', icon: 'cpu', title: '下载引擎', sub: 'yt-dlp、FFmpeg 与程序数据', desc: '下载所依赖的本地组件，以及程序数据的存放位置' },
]

/**
 * 元素查询助手。运行时行为与 `document.querySelector` 完全一致,
 * 只是把「这个选择器拿到的是什么元素」的类型断言集中到一处,
 * 免得 1700 多行模板代码里散落一地的 `as HTMLInputElement`。
 *
 * 用 `$` 这个短名字是因为它会出现在上百个调用点;它不是 jQuery,只是本文件的局部别名。
 */
const $ = <T extends Element = HTMLElement>(selector: string): T => document.querySelector(selector) as T

const STATUS_CLASS: Record<string, string> = { 等待中: 'waiting', 准备中: 'running', 已暂停: 'paused', 已完成: 'done', 失败: 'failed' }

/**
 * 全局界面状态。
 *
 * **刻意不写索引签名** —— 之前用 `[key: string]: any` 兜住未声明字段,结果 59 个字段里
 * 只有 8 个真正受类型约束,`state.maxConcurrnt` 这类拼写错误根本不会被发现。
 * 现在每个字段都必须显式声明,漏一个 tsc 就会报错(而不是悄悄放行)。
 *
 * 字面量取值收成联合类型,拼错会在编译期被发现;取值开放的地方用 string。
 */
interface AppState {
  // —— 视图状态:取值封闭,收成联合类型 ——
  page: 'downloads' | 'settings' | 'toolbox' | 'about'
  theme: 'light' | 'dark'
  queueStep: 'add' | 'list'
  linkMode: 'batch' | 'single'
  videoCodec: 'auto' | 'hevc'
  hevcEncoder: 'auto' | 'cpu'

  // —— 任务队列 ——
  tasks: Task[]
  filter: string
  draftLink: string

  // —— 下载设置(取值来自后端 Settings 或自定义下拉,保持 string) ——
  quality: string
  type: string
  folder: string
  useProxy: boolean
  proxyURL: string
  autoDetectProxy: boolean
  maxConcurrent: number
  autoRetry: number
  highFrameRate: boolean
  openOnComplete: boolean

  // —— Cookies ——
  useBrowserCookie: boolean
  cookieSource: string
  cookieBrowser: string
  cookieFile: string
  cookiePlatforms: CookiePlatform[]
  cookieDir: string
  /** 正在进行中的 Cookies 操作(导入/移除),用于按钮置灰;空串表示空闲 */
  cookieBusy: string
  browserRunning: boolean

  // —— 内核与 FFmpeg ——
  encoderHint: string
  engineReady: boolean
  engineError: string
  ffmpegReady: boolean
  ffmpegPath: string
  ffmpegVersion: string
  dataDir: string
  ffmpegModal: boolean
  ffmpegBusy: boolean
  ffmpegProgress: number
  ffmpegPhase: string
  ffmpegMessage: string

  // —— 代理 ——
  proxyChecked: boolean
  proxyReady: boolean
  proxyMessage: string

  // —— 界面交互 ——
  windowMaximized: boolean
  enterAnim: boolean
  settingsSection: string
  settingsDirty: boolean
  /** 保存前的设置草稿快照,用于「撤销更改」;存的是本地草稿形状,不是后端 Settings */
  settingsBaseline: Record<string, any> | null

  // —— 版本更新 ——
  updateChecking: boolean
  updateChecked: boolean
  updateHas: boolean
  updateLatest: string
  updateNotes: string
  updateUrl: string
  updatePageUrl: string
  updateSource: string
  updatePhase: string
  updateProgress: number
  updateMessage: string
  updateError: string

  // —— 百宝箱:FFmpeg 工具 ——
  toolboxVideo: string
  toolboxVideoInfo: string
  toolboxAudio: string
  toolboxAudioInfo: string
  /** 是否用显卡加速重编码;不勾选则走流复制(无损、瞬间完成) */
  toolboxReencode: boolean
  toolboxBusy: boolean
  toolboxPhase: string
  toolboxProgress: number
  toolboxMessage: string
  toolboxResult: string
}

const state: AppState = {
  page: 'downloads',
  theme: document.documentElement.dataset.theme === 'light' ? 'light' : 'dark',
  queueStep: 'add',
  linkMode: 'batch',
  draftLink: '',
  tasks: [],
  filter: '',
  quality: '1080p',
  type: '视频（带音频）',
  folder: '',
  useProxy: false,
  proxyURL: 'http://127.0.0.1:10809',
  autoDetectProxy: true,
  useBrowserCookie: false,
  cookieSource: 'browser',
  cookieBrowser: 'Chrome',
  cookieFile: '',
  cookiePlatforms: [],
  cookieDir: '',
  cookieBusy: '',
  browserRunning: false,
  highFrameRate: true,
  openOnComplete: false,
  maxConcurrent: 3,
  autoRetry: 2,
  videoCodec: 'auto',
  hevcEncoder: 'auto',
  encoderHint: '',
  engineReady: false,
  engineError: '',
  ffmpegReady: false,
  ffmpegPath: '',
  ffmpegVersion: '',
  dataDir: '',
  proxyChecked: false,
  proxyReady: false,
  proxyMessage: '',
  windowMaximized: false,
  ffmpegModal: false,
  ffmpegBusy: false,
  ffmpegProgress: 0,
  ffmpegPhase: '',
  ffmpegMessage: '',
  settingsSection: 'location',
  settingsDirty: false,
  settingsBaseline: null,
  enterAnim: false,
  /* 版本更新 */
  updateChecking: false,
  updateChecked: false,
  updateHas: false,
  updateLatest: '',
  updateNotes: '',
  updateUrl: '',
  updatePageUrl: '',
  updateSource: '',
  updatePhase: '',
  updateProgress: 0,
  updateMessage: '',
  updateError: '',

  /* 百宝箱:FFmpeg 工具 */
  toolboxVideo: '',
  toolboxVideoInfo: '',
  toolboxAudio: '',
  toolboxAudioInfo: '',
  toolboxReencode: false,
  toolboxBusy: false,
  toolboxPhase: '',
  toolboxProgress: 0,
  toolboxMessage: '',
  toolboxResult: '',
}

let scrollReset = false
const root = $('#app')
const backend = () => window.go?.main?.App
window.addEventListener('error', event => backend()?.ReportFrontendError?.(`${event.message} (${event.filename}:${event.lineno})`))
window.addEventListener('unhandledrejection', event => backend()?.ReportFrontendError?.(`unhandled rejection: ${event.reason}`))

const taskCount = () => state.tasks.length
const ESCAPE_MAP: Record<string, string> = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }
const safe = (s: unknown) => String(s ?? '').replace(/[&<>"']/g, c => ESCAPE_MAP[c] ?? c)

// notesForDisplay 把 Release 说明里的 Markdown 压成纯文本。
// 更新弹窗用 pre-wrap 直接显示文本，不渲染 Markdown，压平后才不会露出 ### 和 ** 这类标记。
function notesForDisplay(raw: string) {
  return String(raw ?? '')
    .replace(/^```[^\n]*$/gm, '')
    .replace(/^[^\S\n]*#{1,6}[^\S\n]*DownTube[^\S\n]*[\d.]*[^\S\n]*$/im, '') // 首行标题与弹窗标题重复，去掉
    .replace(/^[^\S\n]*#{1,6}[^\S\n]+/gm, '')
    .replace(/^[^\S\n]*[-*+][^\S\n]+/gm, '· ')
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/\*\*([^*]+)\*\*/g, '$1')
    .replace(/__([^_]+)__/g, '$1')
    .replace(/`([^`]+)`/g, '$1')
    .replace(/\n{3,}/g, '\n\n')
    .trim()
}
// isSupportedURL 粗筛粘贴的内容像不像能交给 yt-dlp 的链接。
// yt-dlp 支持 1700 多个站点,没法逐一列举域名,这里只做格式判断;
// 具体能不能下载由解析阶段决定,失败时会给出对应站点的提示。
const isSupportedURL = (raw: string) => {
  try {
    const u = new URL(raw)
    if (u.protocol !== 'http:' && u.protocol !== 'https:') return false
    return u.hostname.includes('.')
  } catch { return false }
}
const shortURL = (url: string) => String(url || '').replace(/^https?:\/\//, '').replace(/\/$/, '')

// ============================================================
// 主题
// ============================================================
function toggleTheme() {
  const next = state.theme === 'dark' ? 'light' : 'dark'
  state.theme = next
  document.documentElement.dataset.theme = next
  try { localStorage.setItem('downtube-theme', next) } catch { /* 存储不可用时仅本次生效 */ }
  const button = $('#themeToggle')
  if (button) {
    button.innerHTML = icon(next === 'dark' ? 'sun' : 'moon', 16)
    button.title = next === 'dark' ? '切换到浅色模式' : '切换到深色模式'
  }
}

// ============================================================
// 数据同步
// ============================================================
function applySnapshot(snapshot: Snapshot) {
  if (!snapshot) return
  state.tasks = snapshot.tasks || []
  if (state.settingsDirty) return
  const s: Record<string, any> = snapshot.settings || {}
  state.quality = s.quality || state.quality
  state.type = s.downloadType || state.type
  state.folder = s.downloadDir || state.folder
  state.useProxy = !!s.useProxy
  state.proxyURL = s.proxyURL || state.proxyURL
  state.autoDetectProxy = s.autoDetectProxy !== false
  state.useBrowserCookie = !!s.useBrowserCookie
  state.cookieSource = s.cookieSource || 'browser'
  state.cookieBrowser = s.cookieBrowser || 'Chrome'
  state.cookieFile = s.cookieFile || ''
  state.ffmpegPath = s.ffmpegPath || ''
  state.highFrameRate = s.highFrameRate !== false
  state.openOnComplete = !!s.openOnComplete
  state.maxConcurrent = Math.max(1, Math.min(6, parseInt(s.maxConcurrent, 10) || 3))
  state.autoRetry = Math.max(0, Math.min(5, s.autoRetry == null ? 2 : parseInt(s.autoRetry, 10) || 0))
  state.videoCodec = s.videoCodec === 'hevc' ? 'hevc' : 'auto'
  state.hevcEncoder = s.hevcEncoder === 'cpu' ? 'cpu' : 'auto'
}
async function refreshSnapshot() {
  const api = backend()
  if (!api?.Snapshot) return
  try {
    applySnapshot(await api.Snapshot())
    // 打开时如果已有任务在下载,直接进入队列视图
    if (state.tasks.some(t => t.status === '准备中')) state.queueStep = 'list'
    render()
  } catch (error) { console.error(error); toast(`读取本地数据失败：${(error as Error).message}`) }
}

// ============================================================
// 渲染
// ============================================================
function render() {
  const active = document.activeElement as (Element & { selectionStart?: number | null }) | null
  const activeId = active && active !== document.body && active.id ? active.id : ''
  const selStart = active && typeof active.selectionStart === 'number' ? active.selectionStart : null
  const oldContent = root.querySelector('.content')
  const keepScroll = scrollReset ? 0 : (oldContent ? oldContent.scrollTop : 0)
  scrollReset = false

  root.innerHTML = `
    <div class="app-window">
      <header class="titlebar" id="windowTitlebar">
        <div class="titlebar-title">DownTube</div>
        <div class="titlebar-drag"></div>
        <div class="titlebar-tools">
          <button class="theme-toggle" id="themeToggle" title="${state.theme === 'dark' ? '切换到浅色模式' : '切换到深色模式'}" aria-label="切换主题">${icon(state.theme === 'dark' ? 'sun' : 'moon', 16)}</button>
        </div>
        <div class="window-controls">
          <button class="window-control" id="minimizeWindow" title="最小化" aria-label="最小化">${icon('minimize', 15)}</button>
          <button class="window-control" id="maximizeWindow" title="最大化" aria-label="最大化">${icon(state.windowMaximized ? 'restore' : 'maximize', 13)}</button>
          <button class="window-control close" id="closeWindow" title="关闭" aria-label="关闭">${icon('x', 15)}</button>
        </div>
      </header>
      <div class="shell">
        <aside class="sidebar">
          <nav class="nav">
            <button class="nav-item ${state.page === 'downloads' ? 'active' : ''}" data-page="downloads">${icon('download')}<span>下载队列</span><em>${state.tasks.length || ''}</em></button>
            <button class="nav-item ${state.page === 'settings' ? 'active' : ''}" data-page="settings">${icon('settings')}<span>设置</span></button>
            <button class="nav-item ${state.page === 'toolbox' ? 'active' : ''}" data-page="toolbox">${icon('toolbox')}<span>百宝箱</span></button>
            <button class="nav-item ${state.page === 'about' ? 'active' : ''}" data-page="about">${icon('info')}<span>关于</span>${state.updateHas ? '<i class="nav-badge"></i>' : ''}</button>
          </nav>
        </aside>
        <main class="main">
          <div class="content ${state.enterAnim ? 'page-enter' : ''}">
            <div class="page">${state.page === 'downloads' ? renderDownloads() : state.page === 'settings' ? renderSettings() : state.page === 'toolbox' ? renderToolbox() : renderAbout()}</div>
          </div>
          ${renderStatusbar()}
        </main>
      </div>
      ${state.ffmpegModal ? renderFFmpegModal() : ''}
    </div>
  `
  bind()
  syncWindowButtons()
  if (state.page === 'settings' && state.settingsSection === 'format') refreshEncoderHint()
  state.enterAnim = false
  const newContent = root.querySelector('.content')
  if (newContent) newContent.scrollTop = keepScroll
  if (activeId) {
    const el = document.getElementById(activeId) as (HTMLElement & { setSelectionRange?: (a: number, b: number) => void }) | null
    if (el) {
      el.focus()
      if (selStart !== null && typeof el.setSelectionRange === 'function') {
        try { el.setSelectionRange(selStart, selStart) } catch { /* 不支持选区的控件忽略 */ }
      }
    }
  }
}

function renderStatusbar() {
  const engine = state.engineReady
    ? `<span class="led ok"></span><span>yt-dlp 已就绪</span>`
    : `<span class="led warn"></span><span>yt-dlp 未配置</span>`
  const ffmpeg = state.ffmpegReady
    ? `<span class="status-sep">·</span><span>FFmpeg 已就绪</span>`
    : `<span class="status-sep">·</span><span>FFmpeg 未安装</span>`
  const proxy = state.useProxy
    ? `<span class="status-sep">·</span><span>代理${state.proxyChecked ? (state.proxyReady ? '已连通' : '不可用') : '待检测'}</span>`
    : ''
  return `<footer class="statusbar"><div class="statusbar-left">${engine}${ffmpeg}${proxy}</div><span class="version">DownTube v${APP_VERSION}</span></footer>`
}

// ---------------- 下载页 ----------------
function renderDownloads() {
  const total = taskCount()
  const running = state.tasks.filter(t => t.status === '准备中').length
  const banner = !state.engineReady
    ? `<div class="notice-bar warn"><span class="notice-icon">${icon('download', 18)}</span><div class="notice-copy"><b>下载引擎未配置</b><small>${safe(state.engineError || '请将 yt-dlp.exe 放到程序目录，或加入系统 PATH')}</small></div><button class="btn btn-ghost btn-sm" id="engineSetupBanner">获取 yt-dlp</button></div>`
    : !state.ffmpegReady
      ? `<div class="notice-bar info"><span class="notice-icon">${icon('video', 18)}</span><div class="notice-copy"><b>FFmpeg 未安装</b><small>合并高画质音视频与 MP3 转换需要此组件，基础下载不受影响</small></div><button class="btn btn-ghost btn-sm" id="ffmpegSetup">去安装</button></div>`
      : ''
  const stepListSmall = total ? `${total} 个任务${running ? ` · ${running} 进行中` : ''}` : '暂无任务'
  return `
    ${banner}
    <div class="card">
      <div class="steps">
        <button class="step ${state.queueStep === 'add' ? 'active' : ''}" id="stepAdd">
          <span class="step-num">1</span>
          <span class="step-copy"><b>添加链接</b><small>粘贴来源</small></span>
        </button>
        <button class="step ${state.queueStep === 'list' ? 'active' : ''}" id="stepList" ${total ? '' : 'disabled'}>
          <span class="step-num">2</span>
          <span class="step-copy"><b>队列下载</b><small id="stepListSmall">${stepListSmall}</small></span>
        </button>
      </div>
      ${state.queueStep === 'add' ? renderAddView() : renderQueueView()}
    </div>
  `
}

function renderAddView() {
  const single = state.linkMode === 'single'
  return `<div class="parse-view">
    <h1 class="parse-title">解析链接</h1>
    <p class="parse-desc">${single ? '粘贴一个视频链接，解析后加入下载队列' : '粘贴一个或多个视频链接，每行一个，解析后加入下载队列'}</p>
    <div class="tabs">
      <button class="tab ${single ? '' : 'active'}" data-mode="batch">批量链接</button>
      <button class="tab ${single ? 'active' : ''}" data-mode="single">单个视频</button>
    </div>
    <textarea id="linkInput" class="parse-input ${single ? 'single' : ''}" spellcheck="false" placeholder="${single ? 'https://www.youtube.com/watch?v=…' : 'https://www.youtube.com/watch?v=…&#10;https://www.bilibili.com/video/BV…'}">${safe(state.draftLink)}</textarea>
    <div class="field-hint">每行一个链接；支持视频、短链、播放列表与直播回放</div>
    <div class="parse-actions">
      <button class="btn btn-ghost" id="pasteLinks">${icon('clipboard', 15)}粘贴链接</button>
      <div class="parse-submit">
        <span class="kbd-hint">Ctrl + Enter</span>
        <button class="btn btn-primary" id="parseBtn">开始解析</button>
      </div>
    </div>
    <div class="support-row">
      <span class="support-label">支持来源</span>
      <span class="support-item">${icon('video', 14)}视频与短链</span>
      <span class="support-item">${icon('layers', 14)}播放列表</span>
      <span class="support-item">${icon('broadcast', 14)}直播回放</span>
      <span class="support-item">${icon('log', 14)}多行批量</span>
    </div>
    <div class="sites-note">
      <div class="sites-head"><b>支持的站点</b><span>下载内核为 yt-dlp，只要它支持，这里多半也能下</span></div>
      <div class="sites-group"><span class="sites-label">最佳</span><p class="sites-best"><b>YouTube</b><b>哔哩哔哩</b></p></div>
      <div class="sites-group"><span class="sites-label">其他</span><p class="sites-weak">抖音、西瓜视频、优酷、爱奇艺、腾讯视频、芒果TV、微博、TikTok、Twitter/X、Instagram、Facebook、Vimeo、Twitch、Reddit、Dailymotion</p></div>
      <div class="sites-group"><span class="sites-label">音乐</span><p class="sites-weak">SoundCloud、Bandcamp、网易云音乐、QQ音乐</p></div>
      <p class="sites-caveat">YouTube 与哔哩哔哩适配最充分，画质选项、字幕、播放列表和 Cookies 都按这两个站点调过；抖音视频建议用左侧<b>「百宝箱」</b>里的 Edge 插件下载，效果更好。站点改版、地区限制、需要登录或会员内容都可能解析失败，<b>不保证每个链接都能下载</b>；需要登录的内容（如 YouTube 年龄限制、哔哩哔哩大会员）请到「设置 → Cookies」按站点导入。另外，<b>播放列表、频道、合集与 UP 主空间的链接会把它下面的每个视频都逐个下载</b>，而队列里只显示一条任务，可能需要非常久。</p>
    </div>
  </div>`
}

function filterTasks() {
  const keyword = state.filter.trim().toLowerCase()
  if (!keyword) return state.tasks
  return state.tasks.filter(t => `${t.title} ${t.url} ${t.status}`.toLowerCase().includes(keyword))
}

function queueSummary() {
  const done = state.tasks.filter(t => t.status === '已完成').length
  const running = state.tasks.filter(t => t.status === '准备中').length
  const waiting = state.tasks.filter(t => t.status === '等待中' || t.status === '已暂停').length
  const parts = [`共 ${state.tasks.length} 个任务`]
  if (running) parts.push(`${running} 进行中`)
  if (waiting) parts.push(`${waiting} 等待中`)
  if (done) parts.push(`${done} 已完成`)
  return parts.join(' · ')
}

function queueRunButtons() {
  const running = state.tasks.some(t => t.status === '准备中')
  if (running) return `<button class="btn btn-ghost btn-sm" id="pauseAll">${icon('pause', 14)}全部暂停</button>`
  return `<button class="btn btn-ghost btn-sm" id="startAll" ${state.tasks.length ? '' : 'disabled'}>${icon('play', 14)}全部开始</button>`
}

function renderQueueView() {
  const filtered = filterTasks()
  return `
    <div class="card-head">
      <div><h2>任务列表</h2><span class="card-sub" id="queueSub">${queueSummary()}</span></div>
      <div class="card-head-actions">
        <label class="searchbox">${icon('search', 15)}<input id="searchInput" placeholder="搜索任务" value="${safe(state.filter)}" /></label>
        <span id="queueRunBtns">${queueRunButtons()}</span>
        <button class="btn btn-ghost btn-sm" id="clearBtn">${icon('trash', 14)}清空</button>
      </div>
    </div>
    <div id="queueTableWrap">${renderTable(filtered)}</div>
    <div class="card-foot">
      <span id="queueFootCount">显示 ${filtered.length} / ${state.tasks.length} 个任务</span>
      <button class="text-link" id="openFolderBtn">${icon('folder', 14)}打开保存位置</button>
    </div>
  `
}

function renderTable(list: Task[]) {
  if (!state.tasks.length) {
    return `<div class="empty"><span class="empty-icon">${icon('video', 22)}</span><b>队列还是空的</b><p>在「添加链接」里粘贴链接，任务会显示在这里</p></div>`
  }
  if (!list.length) {
    return `<div class="empty"><span class="empty-icon">${icon('search', 22)}</span><b>没有匹配的任务</b><p>试试其他关键词</p></div>`
  }
  return `<table class="queue-table">
    <colgroup><col class="col-task"><col class="col-status"><col class="col-progress"><col class="col-quality"><col class="col-actions"></colgroup>
    <thead><tr><th>视频</th><th>状态</th><th>进度</th><th>画质</th><th style="text-align:right">操作</th></tr></thead>
    <tbody id="queueBody">${list.map(taskRow).join('')}</tbody>
  </table>`
}

function taskSig(t: Task) {
  return `${t.status}|${t.phase || ''}|${t.progress}|${t.title}|${t.error || ''}|${t.quality}`
}

function taskRow(t: Task) {
  const statusClass = STATUS_CLASS[t.status] || 'waiting'
  const statusText = t.phase || t.status
  const active = t.status === '准备中'
  const canStart = ['等待中', '已暂停', '失败'].includes(t.status)
  const sub = t.error ? `<span class="task-sub error">${safe(t.error)}</span>` : `<span class="task-sub">${safe(shortURL(t.url))}</span>`
  return `<tr data-id="${t.id}" data-sig="${safe(taskSig(t))}">
    <td><div class="task-cell"><span class="task-thumb">${icon('video', 16)}</span><div class="task-copy"><b class="task-title" title="${safe(t.title)}">${safe(t.title)}</b>${sub}</div></div></td>
    <td><span class="status ${statusClass}"><i></i>${safe(statusText)}</span></td>
    <td><div class="progress-cell"><div class="progress-track"><span class="progress-fill ${t.status === '已完成' ? 'is-done' : ''}" style="width:${t.progress}%"></span></div><span class="progress-num">${t.progress}%</span></div></td>
    <td><span class="tag">${safe(t.quality || '默认')}</span></td>
    <td><div class="row-actions">
      ${canStart ? `<button class="row-btn" data-action="start" data-id="${t.id}" title="开始">${icon('play', 14)}</button>` : ''}
      ${active ? `<button class="row-btn" data-action="pause" data-id="${t.id}" title="暂停">${icon('pause', 14)}</button>` : ''}
      <button class="row-btn danger" data-action="remove" data-id="${t.id}" title="移除">${icon('x', 14)}</button>
    </div></td>
  </tr>`
}

// ---------------- 设置页 ----------------
function renderSettings() {
  const section = SETTINGS_SECTIONS.find(s => s.id === state.settingsSection) || SETTINGS_SECTIONS[0]
  return `
    <div class="settings-head">
      <h1>设置</h1>
      <div class="settings-head-actions">
        <button class="btn btn-ghost btn-sm" id="resetSettings">${icon('rotateCcw', 14)}恢复默认</button>
        <button class="btn btn-ghost btn-sm" id="revertSettings">${icon('undo', 14)}撤销</button>
        <button class="btn btn-primary btn-sm" id="saveSettings">${icon('save', 14)}保存</button>
      </div>
    </div>
    <div class="settings-layout">
      <nav class="settings-nav">
        ${SETTINGS_SECTIONS.map(s => `
          <button class="snav-item ${s.id === state.settingsSection ? 'active' : ''}" data-section="${s.id}">
            <span class="snav-icon">${icon(s.icon, 17)}</span>
            <span class="snav-copy"><b>${s.title}</b><small>${s.sub}</small></span>
          </button>`).join('')}
      </nav>
      <section class="settings-content">
        <div class="set-head">
          <span class="set-head-icon">${icon(section.icon, 18)}</span>
          <div><h2>${section.title}</h2><p>${section.desc}</p></div>
        </div>
        ${renderSettingsSection(section.id)}
      </section>
    </div>
  `
}

// renderCookiePlatformRow 渲染分站点 Cookies 的一行：站点名 + 导入状态 + 导入/移除。
function renderCookiePlatformRow(p: CookiePlatform) {
  const busy = state.cookieBusy === p.id
  const imported = !!p.imported
  const detail = imported
    ? `已导入 · ${p.count} 条${p.updatedAt ? ` · ${p.updatedAt} 更新` : ''}`
    : '未导入'
  return `<div class="row-split cookie-plat">
    <div class="row-split-copy"><b>${safe(p.name)}</b><small class="${imported ? 'ok' : ''}">${safe(detail)}</small></div>
    <div class="cookie-plat-actions">
      <button class="btn btn-ghost btn-sm" data-cookie-import="${safe(p.id)}" ${busy ? 'disabled' : ''}>${busy ? '处理中…' : (imported ? '重新导入' : '导入')}</button>
      ${imported ? `<button class="btn btn-ghost btn-sm" data-cookie-remove="${safe(p.id)}" ${busy ? 'disabled' : ''}>移除</button>` : ''}
    </div>
  </div>`
}

function renderSettingsSection(id: string) {
  if (id === 'location') {
    return `
      <div class="set-card">
        <div class="set-card-head"><b>保存位置</b><p>视频将保存到所选文件夹，文件名尽量保留原标题</p></div>
        <div class="set-card-body">
          <label class="field"><span>下载文件夹</span>
            <div class="path-field">
              <div class="path-input">${icon('folder', 15)}<input id="folderPath" value="${safe(state.folder)}" placeholder="选择保存文件夹" /></div>
              <button class="btn btn-ghost" id="browseFolder">浏览…</button>
            </div>
          </label>
          <div class="divider"></div>
          <div class="row-split">
            <div class="row-split-copy"><b>下载完成后打开文件夹</b><small>自动定位到已保存的视频</small></div>
            <label class="switch"><input id="openOnComplete" type="checkbox" ${state.openOnComplete ? 'checked' : ''}><span></span></label>
          </div>
        </div>
      </div>
      <div class="set-card">
        <div class="set-card-head"><b>下载行为</b><p>同时下载的任务数量，以及出错时的处理方式</p></div>
        <div class="set-card-body">
          <div class="form-grid">
            <div class="field"><span>同时下载的任务数</span>${selectControl('maxConcurrent', [1, 2, 3, 4, 5].map(n => ({ value: String(n), label: `${n} 个` })), String(state.maxConcurrent), '同时下载的任务数')}</div>
            <div class="field"><span>下载失败时自动重试</span>${selectControl('autoRetry', [0, 1, 2, 3].map(n => ({ value: String(n), label: n === 0 ? '不重试' : `${n} 次` })), String(state.autoRetry), '自动重试次数')}</div>
          </div>
        </div>
      </div>`
  }
  if (id === 'format') {
    return `
      <div class="set-card">
        <div class="set-card-head"><b>画质与格式</b><p>新任务默认采用以下设置；H.265 在下载完成后用 FFmpeg 转码，耗时与视频长度相关</p></div>
        <div class="set-card-body">
          <div class="form-grid">
            <div class="field"><span>默认视频画质</span>${selectControl('quality', ['2160p · 4K', '1440p', '1080p', '720p', '480p', '最佳可用'], state.quality, '默认视频画质')}</div>
            <div class="field"><span>下载内容</span>${selectControl('downloadType', ['视频（带音频）', '仅视频', '仅音频 · MP3', '仅音频 · FLAC'], state.type, '下载内容')}</div>
            <div class="field"><span>视频编码</span>${selectControl('videoCodec', [{ value: 'auto', label: '智能匹配 · H.264 优先' }, { value: 'hevc', label: 'H.265 · 下载后转码' }], state.videoCodec, '视频编码')}</div>
            <div class="field"><span>H.265 编码器</span>${selectControl('hevcEncoder', [{ value: 'auto', label: '智能加速 · 优先显卡' }, { value: 'cpu', label: 'CPU · 压缩率更高' }], state.hevcEncoder, 'H.265 编码器')}<small class="field-hint" id="encoderHint">${safe(state.encoderHint)}</small></div>
          </div>
          <div class="divider"></div>
          <div class="row-split">
            <div class="row-split-copy"><b>优先选择高帧率</b><small>可用时优先下载 60 fps 视频</small></div>
            <label class="switch"><input id="highFrameRate" type="checkbox" ${state.highFrameRate ? 'checked' : ''}><span></span></label>
          </div>
        </div>
      </div>`
  }
  if (id === 'network') {
    return `
      <div class="set-card">
        <div class="set-card-head"><b>代理连接</b><p>通过本机代理访问站点，适用于网络受限的环境</p></div>
        <div class="set-card-body">
          <div class="row-split">
            <div class="row-split-copy"><b>使用代理</b><small>下载与解析均通过指定代理连接</small></div>
            <label class="switch"><input id="proxyToggle" type="checkbox" ${state.useProxy ? 'checked' : ''}><span></span></label>
          </div>
          <label class="field ${state.useProxy ? '' : 'hidden'}" id="proxyAddressField"><span>代理地址</span><input id="proxyURL" class="input" value="${safe(state.proxyURL)}" placeholder="http://127.0.0.1:10809" /></label>
          <div class="row-split">
            <div class="row-split-copy"><b>启动时检测代理</b><small>启动时确认代理端口是否可连接</small></div>
            <label class="switch"><input id="autoDetectToggle" type="checkbox" ${state.autoDetectProxy ? 'checked' : ''}><span></span></label>
          </div>
          <div class="divider"></div>
          <div class="proxy-row">
            <button class="btn btn-ghost btn-sm" id="testProxy">${icon('shield', 14)}测试连接</button>
            <span class="proxy-result ${state.proxyChecked ? (state.proxyReady ? 'ok' : 'bad') : ''}">${state.proxyChecked ? safe(state.proxyMessage) : state.useProxy ? '尚未测试' : '代理已关闭'}</span>
          </div>
        </div>
      </div>`
  }
  if (id === 'cookies') {
    return `
      <div class="set-card">
        <div class="set-card-head"><b>Cookies 来源</b><p>用于访问需要登录或年龄验证的视频</p></div>
        <div class="set-card-body">
          <div class="row-split">
            <div class="row-split-copy"><b>使用 Cookies</b><small>随请求携带浏览器身份信息</small></div>
            <label class="switch"><input id="cookiesToggle" type="checkbox" ${state.useBrowserCookie ? 'checked' : ''}><span></span></label>
          </div>
          <div class="cookie-panel ${state.useBrowserCookie ? '' : 'hidden'}" id="cookiePanel">
            <div class="cookie-tabs" role="tablist">
              <button type="button" class="cookie-tab ${state.cookieSource === 'browser' ? 'active' : ''}" data-cookie-source="browser" role="tab" aria-selected="${state.cookieSource === 'browser'}">${icon('globe', 13)}从浏览器读取</button>
              <button type="button" class="cookie-tab ${state.cookieSource === 'file' ? 'active' : ''}" data-cookie-source="file" role="tab" aria-selected="${state.cookieSource === 'file'}">${icon('file', 13)}从文件导入</button>
            </div>
            <div class="${state.cookieSource === 'browser' ? '' : 'hidden'}" id="cookieSourceBrowser">
              <div class="field"><span>读取 Cookies 的浏览器</span>${selectControl('cookieBrowser', ['Chrome', 'Microsoft Edge', 'Firefox', 'Brave'], state.cookieBrowser, '读取 Cookies 的浏览器')}</div>
              <div class="hint ${state.browserRunning ? 'warn' : ''}" id="browserRunningHint" style="margin-top:10px">${state.browserRunning ? safe(state.cookieBrowser) + ' 正在运行 —— 读取 Cookies 会失败，请彻底退出后再试' : '此方式需完全退出浏览器(含后台进程)后使用'}</div>
            </div>
            <div class="${state.cookieSource === 'file' ? '' : 'hidden'}" id="cookieSourceFile">
              <div class="cookie-plat-list">
                ${state.cookiePlatforms.map(renderCookiePlatformRow).join('')}
              </div>
              <div class="hint" style="margin-top:10px">用浏览器扩展(如 Get cookies.txt)导出 Netscape 格式的 cookies.txt，导入时自动识别站点，分别存到程序数据目录，下载时按链接自动选用</div>
              <div class="cookie-plat-foot">
                <span class="hint">其他站点</span>
                <div class="path-input">${icon('file', 15)}<input id="cookieFilePath" value="${safe(state.cookieFile)}" placeholder="可选的通用 cookies.txt" /></div>
                <button class="btn btn-ghost" id="browseCookieFile">浏览…</button>
                <button class="btn btn-ghost" id="openCookieFolder">打开目录</button>
              </div>
            </div>
          </div>
        </div>
      </div>`
  }
  // engine
  return `
    <div class="set-card">
      <div class="set-card-head"><b>yt-dlp</b><p>负责解析视频信息与下载媒体流</p></div>
      <div class="set-card-body">
        <div class="engine-row">
          <span class="engine-badge ${state.engineReady ? 'ok' : 'warn'}">${icon('download', 18)}</span>
          <div class="engine-copy">
            <b>${state.engineReady ? '已就绪' : '未配置'}</b>
            <small>${state.engineReady ? 'yt-dlp.exe 已随程序提供，可直接使用' : safe(state.engineError || '将 yt-dlp.exe 放到程序目录，或加入系统 PATH')}</small>
          </div>
          ${state.engineReady ? '' : `<button class="btn btn-ghost btn-sm" id="engineSetupBtn">获取 yt-dlp</button>`}
        </div>
      </div>
    </div>
    <div class="set-card">
      <div class="set-card-head"><b>FFmpeg</b><p>合并高清音视频与格式转换所需的媒体组件</p></div>
      <div class="set-card-body">
        <div class="engine-row">
          <span class="engine-badge ${state.ffmpegReady ? 'ok' : 'warn'}">${icon('video', 18)}</span>
          <div class="engine-copy">
            <b>${state.ffmpegReady ? '已就绪' : '未安装'}</b>
            <small>${state.ffmpegReady ? safe(state.ffmpegVersion || state.ffmpegPath || '已检测到可用组件') : '未安装时，包含独立音频流的高画质视频将无法合并'}</small>
          </div>
          <button class="btn btn-ghost btn-sm" id="ffmpegManageBtn">${state.ffmpegReady ? '更换' : '去安装'}</button>
        </div>
      </div>
    </div>
    <div class="set-card">
      <div class="set-card-head"><b>程序数据</b><p>任务队列、界面设置与自动安装的组件保存在这里</p></div>
      <div class="set-card-body">
        <div class="engine-row">
          <span class="engine-badge">${icon('folder', 18)}</span>
          <div class="engine-copy">
            <b>数据目录</b>
            <small>${safe(state.dataDir || '任务队列与设置的本地存放位置')}</small>
          </div>
          <button class="btn btn-ghost btn-sm" id="openDataFolder">${icon('external', 13)}打开目录</button>
        </div>
      </div>
    </div>`
}

function selectControl(setting: string, options: SelectOption[], value: unknown, ariaLabel?: string) {
  const id = `select-${setting}`
  const items = options.map(o => (typeof o === 'string' ? { value: o, label: o } : o))
  const current = items.find(o => o.value === String(value))
  const currentLabel = current ? current.label : String(value)
  return `<div class="md-select" data-setting="${setting}"><button class="select-trigger" type="button" id="${id}" aria-label="${ariaLabel}" aria-haspopup="listbox" aria-expanded="false" aria-controls="${id}-menu"><span class="select-current">${safe(currentLabel)}</span><span class="select-chevron">${icon('chevron', 16)}</span></button><div class="select-menu" id="${id}-menu" role="listbox" aria-labelledby="${id}">${items.map(item => `<button class="select-option ${item.value === String(value) ? 'selected' : ''}" type="button" role="option" tabindex="-1" data-value="${safe(item.value)}" aria-selected="${item.value === String(value)}"><span>${safe(item.label)}</span><span class="select-check">${icon('check', 15)}</span></button>`).join('')}</div></div>`
}

// ---------------- 关于页 ----------------
// 更新区块:按状态渲染 —— 无更新时提示已是最新,发现新版显示版本与说明,
// 下载中显示进度,下载完成显示安装按钮。
function renderUpdateBlock() {
  if (state.updateChecking && !state.updateChecked && !state.updateHas) return ''
  if (state.updatePhase === 'downloading') {
    const indeterminate = state.updateProgress <= 0
    return `<div class="update-block">
      <div class="update-block-head"><b>正在下载 v${safe(state.updateLatest)}</b><span id="updateProgressPct">${indeterminate ? '' : state.updateProgress + '%'}</span></div>
      <div class="ff-track ${indeterminate ? 'indeterminate' : ''}"><span id="updateProgressBar" style="width:${state.updateProgress}%"></span></div>
      <p class="update-hint" id="updateProgressLabel">${safe(state.updateMessage || '正在连接下载源…')}</p>
    </div>`
  }
  if (state.updatePhase === 'downloaded' || state.updatePhase === 'installing') {
    const installing = state.updatePhase === 'installing'
    return `<div class="update-block ready">
      <div class="update-block-head"><b>v${safe(state.updateLatest)} 已就绪</b></div>
      <p class="update-hint">${installing ? '安装程序已启动，程序即将自动关闭并完成更新' : '点击「立即安装」后程序会自动关闭，安装完成后重新打开'}</p>
      <button class="btn btn-primary btn-sm" id="installUpdateBtn" ${installing ? 'disabled' : ''}>${installing ? '正在安装…' : '立即安装并重启'}</button>
    </div>`
  }
  if (state.updateHas) {
    const source = state.updateSource === 'github' ? 'GitHub' : 'CNB'
    return `<div class="update-block">
      <div class="update-block-head"><b>发现新版本 v${safe(state.updateLatest)}</b><span class="update-source">来自 ${source}</span></div>
      ${state.updateNotes ? `<div class="update-notes">${safe(notesForDisplay(state.updateNotes))}</div>` : ''}
      ${state.updateError ? `<p class="update-hint error">${safe(state.updateError)}</p>` : ''}
      <div class="update-actions">
        <button class="btn btn-primary btn-sm" id="downloadUpdateBtn">下载更新</button>
        <button class="btn btn-ghost btn-sm" id="openReleaseBtn">查看发布页</button>
      </div>
    </div>`
  }
  if (state.updateChecked) {
    if (state.updateError) return `<p class="update-hint error">${safe(state.updateError)}</p>`
    return `<p class="update-ok">已是最新版本</p>`
  }
  return ''
}

// ============================================================
// 百宝箱:推荐工具 + FFmpeg 工具箱
// ============================================================

const TAMPERMONKEY_REPO = 'https://github.com/WhiteSevs/TamperMonkeyScript'

/** 抖音网页端 Edge 插件推荐 */
function renderToolboxRecommend() {
  return `<div class="tb-card">
    <div class="tb-card-head">
      <span class="tb-card-icon">${icon('shield', 18)}</span>
      <div><h2>推荐</h2><span class="card-sub">抖音网页端更好用的办法</span></div>
    </div>
    <div class="tb-recommend">
      <div class="tb-recommend-body">
        <h3>Tampermonkey 脚本 · Edge 插件</h3>
        <p>给 Edge 装上油猴插件、导入这个脚本，抖音网页端就能<b>直接下载视频</b>，页面上的干扰元素也会一并清掉</p>
        <p class="tb-recommend-tip">${icon('checkcircle', 14)}下载抖音视频优先用这个插件，比在下载队列里贴链接更稳，画质也更完整</p>
      </div>
      <div class="tb-recommend-actions">
        <button class="btn btn-primary btn-sm" id="tbOpenRepo">${icon('external', 14)}打开仓库</button>
        <button class="btn btn-ghost btn-sm" id="tbCopyRepo">${icon('clipboard', 14)}复制地址</button>
      </div>
      <div class="tb-repo">${icon('github', 14)}<span>${TAMPERMONKEY_REPO}</span></div>
    </div>
  </div>`
}

/** FFmpeg 工具箱:合并 / 分离 */
function renderToolboxFFmpeg() {
  const canStart = !!state.toolboxVideo && !state.toolboxBusy
  return `<div class="tb-card">
    <div class="tb-card-head">
      <span class="tb-card-icon">${icon('sliders', 18)}</span>
      <div><h2>FFmpeg 工具</h2><span class="card-sub">音视频合并与分离，支持显卡硬件加速</span></div>
    </div>

    <div class="tb-panel">
      <div class="tb-panel-title">${icon('layers', 15)}视频音频合并</div>
      <p class="tb-panel-desc">把外部音频合并进视频，替换原来的音轨。适合下载到无声视频、需要另外配一条音轨的情况</p>
      <div class="tb-files">
        <div class="tb-file">
          <span class="tb-file-tag">视频</span>
          <span class="tb-file-name" id="tbVideoName">${state.toolboxVideo ? safe(shortURL(state.toolboxVideo)) : '未选择'}</span>
          <button class="btn btn-ghost btn-sm" id="tbPickVideo" ${state.toolboxBusy ? 'disabled' : ''}>选择</button>
        </div>
        <div class="tb-file">
          <span class="tb-file-tag">音频</span>
          <span class="tb-file-name" id="tbAudioName">${state.toolboxAudio ? safe(shortURL(state.toolboxAudio)) : '未选择'}</span>
          <button class="btn btn-ghost btn-sm" id="tbPickAudio" ${state.toolboxBusy ? 'disabled' : ''}>选择</button>
        </div>
      </div>
      <div class="tb-actions">
        <button class="btn btn-primary btn-sm" id="tbMerge" ${canStart && state.toolboxAudio ? '' : 'disabled'}>${icon('layers', 14)}开始合并</button>
      </div>
    </div>

    <div class="tb-panel">
      <div class="tb-panel-title">${icon('undo', 15)}视频分离</div>
      <p class="tb-panel-desc">把视频拆成独立的视频流和音频流两个文件。适合只想要其中一路，或者想自己重新拼一次的情况</p>
      <div class="tb-actions">
        <button class="btn btn-primary btn-sm" id="tbSplit" ${canStart ? '' : 'disabled'}>${icon('undo', 14)}开始分离</button>
      </div>
    </div>

    <label class="tb-switch">
      <input type="checkbox" id="tbReencode" ${state.toolboxReencode ? 'checked' : ''} ${state.toolboxBusy ? 'disabled' : ''}>
      <span>显卡加速重编码（输出 H.265，文件更小）</span>
    </label>
    <p class="tb-hint">${icon('info', 13)}不勾选时是<b>无损流复制</b>：不重新编码，瞬间完成，画质不变。合并和分离本来就不需要重编码，只有想让文件变小才需要勾选</p>

    <div class="tb-progress-wrap" id="tbProgressWrap" ${state.toolboxBusy || state.toolboxResult ? '' : 'hidden'}>
      <div class="tb-progress-head">
        <span id="tbPhase">${safe(state.toolboxMessage || (state.toolboxBusy ? '处理中…' : '已完成'))}</span>
        <span id="tbPercent">${Math.round(state.toolboxProgress)}%</span>
      </div>
      <div class="tb-progress"><div class="tb-progress-bar" id="tbProgressBar" style="width:${Math.round(state.toolboxProgress)}%"></div></div>
      <div class="tb-progress-actions">
        <button class="btn btn-ghost btn-sm" id="tbCancel" ${state.toolboxBusy ? '' : 'disabled'}>取消</button>
      </div>
    </div>
    ${state.toolboxResult ? `<p class="tb-result ${state.toolboxResult.startsWith('失败') ? 'error' : ''}">${safe(state.toolboxResult)}</p>` : ''}
    <p class="tb-outdir-note">${icon('folder', 13)}输出文件夹<b>每次都会弹窗询问</b>，不自动保存，也不记住上次的位置</p>
  </div>`
}

function renderToolbox() {
  return `<div class="tb-wrap">
    ${renderToolboxRecommend()}
    ${renderToolboxFFmpeg()}
  </div>`
}

function renderAbout() {
  return `<div class="about-wrap">
    <div class="about-card">
      <div class="about-brand">
        <img class="about-logo" src="/downtube-icon.svg" alt="DownTube 图标" />
        <div>
          <div class="about-overline">DESKTOP DOWNLOAD TOOL</div>
          <h1 class="about-title">DownTube</h1>
          <p class="about-desc">一个专注 YouTube 本地下载与任务管理的桌面工具，但又不仅限于 YouTube</p>
        </div>
      </div>
      <div class="about-divider"></div>
      <div class="about-info">
        <div class="info-row">
          <span>当前版本</span>
          <b>v${APP_VERSION}</b>
          <button class="btn btn-ghost btn-sm update-check-btn" id="checkUpdateBtn" ${state.updateChecking ? 'disabled' : ''}>${state.updateChecking ? '检查中…' : '检查更新'}</button>
        </div>
        <div class="info-row"><span>应用名称</span><b>DownTube</b></div>
      </div>
      <div id="updateArea">${renderUpdateBlock()}</div>
      <h3 class="about-links-title">作者链接</h3>
      <div class="link-list">
        <div class="link-row">
          <span class="link-icon">${icon('github', 18)}</span>
          <div class="link-copy"><b>GitHub 仓库</b><span>github.com/youngdajie/DownTube</span></div>
          <button class="btn btn-ghost btn-sm" id="linkGithub">打开</button>
        </div>
        <div class="link-row">
          <span class="link-icon">${icon('code', 18)}</span>
          <div class="link-copy"><b>CNB 仓库</b><span>cnb.cool/youngdajie/downtube</span></div>
          <button class="btn btn-ghost btn-sm" id="linkCnb">打开</button>
        </div>
        <div class="link-row">
          <span class="link-icon">${icon('bilibili', 18)}</span>
          <div class="link-copy"><b>哔哩哔哩主页</b><span>space.bilibili.com/646325099</span></div>
          <button class="btn btn-ghost btn-sm" id="linkBilibili">打开</button>
        </div>
        <div class="link-row">
          <span class="link-icon">${icon('globe', 18)}</span>
          <div class="link-copy"><b>个人网站</b><span>www.yangjie.site</span></div>
          <button class="btn btn-ghost btn-sm" id="linkHomepage">打开</button>
        </div>
      </div>
    </div>
    <p class="about-note">任务与设置仅保存在本机，不会上传或收集任何链接。本工具不提供视频内容，请仅下载你有权保存的内容，并遵守相关网站的服务条款与版权规则</p>
  </div>`
}

// ---------------- FFmpeg 弹窗 ----------------
function renderFFmpegModal() {
  const busy = state.ffmpegBusy
  const showProgress = state.ffmpegBusy || state.ffmpegPhase === 'done'
  const indeterminate = state.ffmpegPhase === 'extract' && state.ffmpegProgress < 100
  return `<div class="modal-backdrop" id="ffmpegBackdrop"><div class="modal ffmpeg-modal">
    <div class="modal-head">
      <div><h2>安装 FFmpeg</h2><p>合并高清音视频与 MP3 转换所需的媒体组件，安装后自动生效</p></div>
      <button class="icon-btn" id="closeFFmpegModal" aria-label="关闭">${icon('x', 16)}</button>
    </div>
    <div class="ff-section">
      <div class="ff-section-title">在线下载</div>
      <p class="ff-hint">点击「前往下载」会在浏览器中打开镜像链接，下载得到 <b>ffmpeg-9.0.2-essentials_build.7z</b>；完成后回到这里点击「导入安装包」选择该文件，程序会自动解压并配置</p>
      <div class="mirror-list">
        ${FFMPEG_MIRRORS.map(m => `
          <div class="mirror-item">
            <div class="mirror-top">
              <span class="mirror-icon">${icon('cloudDown', 18)}</span>
              <div class="mirror-copy"><b>${m.name}</b><small>${m.desc}</small></div>
              <span class="mirror-tag">外部来源</span>
            </div>
            <div class="mirror-bottom">
              <span class="mirror-file">${m.file}</span>
              <span class="mirror-meta">${m.meta.map(x => `<span>${x}</span>`).join('')}</span>
              <button class="btn btn-ghost btn-sm mirror-action" data-url="${m.url}" ${busy ? 'disabled' : ''}>${icon('external', 13)}前往下载</button>
            </div>
          </div>`).join('')}
      </div>
    </div>
    <div class="ff-section">
      <div class="ff-section-title">从本地安装</div>
      <div class="ff-actions">
        <button class="btn btn-ghost" id="ffmpegImportArchive" ${busy ? 'disabled' : ''}>${icon('package', 15)}导入安装包(.7z / .zip)</button>
        <button class="btn btn-ghost" id="ffmpegPickFile" ${busy ? 'disabled' : ''}>${icon('file', 15)}选择 ffmpeg.exe</button>
      </div>
      <p class="ff-hint" style="margin:10px 0 0">导入后会解压安装到程序目录，自动关联 bin 下的可执行文件；也可以直接选择已下载的 ffmpeg.exe</p>
    </div>
    <div class="ff-section">
      <div class="ff-section-title">自动安装</div>
      <button class="btn btn-primary btn-block" id="ffmpegAutoInstall" ${busy ? 'disabled' : ''}>${icon('download', 15)}自动下载安装(约 110 MB)</button>
      <p class="ff-hint" style="margin:9px 0 0">从官方地址下载压缩包并解压，会走已配置的代理；国内网络较慢时建议使用上方镜像手动下载</p>
    </div>
    ${showProgress ? `<div class="ff-progress">
      <div class="ff-progress-head"><span id="ffmpegProgressLabel">${safe(state.ffmpegMessage || '准备中…')}</span><b id="ffmpegProgressPct">${indeterminate ? '' : state.ffmpegProgress + '%'}</b></div>
      <div class="ff-track ${indeterminate ? 'indeterminate' : ''}"><span id="ffmpegProgressBar" style="width:${state.ffmpegProgress}%"></span></div>
    </div>` : ''}
    <div class="modal-foot">
      <span>${state.ffmpegReady ? '重新安装会替换程序目录内的版本' : '安装完成后自动生效，无需重启'}</span>
      <button class="btn btn-ghost btn-sm" id="ffmpegModalCancel">关闭</button>
    </div>
  </div></div>`
}

// ============================================================
// Toast
// ============================================================
function toast(message: string) {
  document.querySelectorAll('.toast').forEach(el => el.remove())
  const el = document.createElement('div')
  el.className = 'toast'
  el.innerHTML = `${icon('checkcircle', 17)}<span>${safe(message)}</span>`
  $('.app-window')?.appendChild(el)
  setTimeout(() => el.remove(), 2400)
}

// ============================================================
// 局部更新(后台事件驱动,避免打断输入与滚动)
// ============================================================
function patchDownloads() {
  const badge = $('.nav-item[data-page="downloads"] em')
  if (badge) badge.textContent = state.tasks.length ? String(state.tasks.length) : ''
  const stepListSmall = $('#stepListSmall')
  if (stepListSmall) {
    const total = taskCount()
    const running = state.tasks.filter(t => t.status === '准备中').length
    stepListSmall.textContent = total ? `${total} 个任务${running ? ` · ${running} 进行中` : ''}` : '暂无任务'
  }
  if (state.queueStep !== 'list') return
  const sub = $('#queueSub')
  if (sub) sub.textContent = queueSummary()
  const runBtns = $('#queueRunBtns')
  if (runBtns) runBtns.innerHTML = queueRunButtons()
  const wrap = $('#queueTableWrap')
  if (!wrap) return
  const filtered = filterTasks()
  const bodyEl = wrap.querySelector('#queueBody')
  const wanted = filtered.map(t => t.id)
  const current = bodyEl ? [...(bodyEl.children as unknown as HTMLElement[])].map(tr => Number(tr.dataset.id)) : null
  if (!bodyEl || JSON.stringify(wanted) !== JSON.stringify(current)) {
    wrap.innerHTML = renderTable(filtered)
  } else {
    for (const tr of [...(bodyEl.children as unknown as HTMLElement[])]) {
      const t = state.tasks.find(x => x.id === Number(tr.dataset.id))
      if (!t || tr.dataset.sig === taskSig(t)) continue
      const holder = document.createElement('tbody')
      holder.innerHTML = taskRow(t)
      tr.replaceWith(holder.firstElementChild!)
    }
  }
  const foot = $('#queueFootCount')
  if (foot) foot.textContent = `显示 ${filtered.length} / ${state.tasks.length} 个任务`
}

function refreshQueueList() {
  const wrap = $('#queueTableWrap')
  if (!wrap) return
  const filtered = filterTasks()
  wrap.innerHTML = renderTable(filtered)
  const foot = $('#queueFootCount')
  if (foot) foot.textContent = `显示 ${filtered.length} / ${state.tasks.length} 个任务`
}

// ============================================================
// 绑定与全局委托
// ============================================================
function closeSelect(select: HTMLElement | null, restoreFocus = false) {
  if (!select?.classList.contains('is-open')) return
  select.classList.remove('is-open')
  select.querySelector('.select-trigger')?.setAttribute('aria-expanded', 'false')
  if (restoreFocus) select.querySelector<HTMLElement>('.select-trigger')?.focus()
}
function closeAllSelects(except: HTMLElement | null = null) {
  root.querySelectorAll<HTMLElement>('.md-select.is-open').forEach(select => { if (select !== except) closeSelect(select) })
}

root.addEventListener('click', event => {
  // 任务行操作
  const actionBtn = (event.target as HTMLElement).closest<HTMLElement>('[data-action]')
  if (actionBtn) { handleTask(actionBtn.dataset.action!, Number(actionBtn.dataset.id)); return }
  // 队列运行控制(会被局部更新替换,必须走委托)
  if ((event.target as HTMLElement).closest<HTMLElement>('#startAll')) { startAll(); return }
  if ((event.target as HTMLElement).closest<HTMLElement>('#pauseAll')) { pauseAll(); return }
  // 导航
  const navBtn = (event.target as HTMLElement).closest<HTMLElement>('[data-page]')
  if (navBtn) {
    const page = navBtn.dataset.page
    if (page !== state.page) {
      state.enterAnim = true
      scrollReset = true
      state.page = page as AppState['page']
      if (page === 'settings') { captureSettingsBaseline(); refreshBrowserRunning(); refreshCookieStatus() }
      render()
    }
    return
  }
  // 设置分类
  const sectionBtn = (event.target as HTMLElement).closest<HTMLElement>('[data-section]')
  if (sectionBtn) {
    if (sectionBtn.dataset.section !== state.settingsSection) {
      state.settingsSection = sectionBtn.dataset.section!
      scrollReset = true
      render()
    }
    return
  }
  // 解析模式 tab
  const modeBtn = (event.target as HTMLElement).closest<HTMLElement>('[data-mode]')
  if (modeBtn) {
    if (modeBtn.dataset.mode !== state.linkMode) { state.linkMode = modeBtn.dataset.mode as AppState['linkMode']; render(); $('#linkInput')?.focus() }
    return
  }
  // 外部下载镜像
  const urlBtn = (event.target as HTMLElement).closest<HTMLElement>('[data-url]')
  if (urlBtn) {
    const url = urlBtn.dataset.url
    const api = backend()
    if (api?.OpenExternalURL) api.OpenExternalURL(url!)
    else window.open(url, '_blank')
    return
  }
  // Cookies 来源 tab
  const cookieTab = (event.target as HTMLElement).closest<HTMLElement>('[data-cookie-source]')
  if (cookieTab) { setCookieSource(cookieTab.dataset.cookieSource!); return }
  // 分站点 Cookies 导入/移除
  const cookieImport = (event.target as HTMLElement).closest<HTMLElement>('[data-cookie-import]')
  if (cookieImport) { importPlatformCookie(cookieImport.dataset.cookieImport!); return }
  const cookieRemove = (event.target as HTMLElement).closest<HTMLElement>('[data-cookie-remove]')
  if (cookieRemove) { removePlatformCookie(cookieRemove.dataset.cookieRemove!); return }
  // 自定义下拉
  const trigger = (event.target as HTMLElement).closest<HTMLElement>('.select-trigger')
  if (trigger) {
    const select = trigger.closest<HTMLElement>('.md-select')!
    const shouldOpen = !select.classList.contains('is-open')
    closeAllSelects(select)
    select.classList.toggle('is-open', shouldOpen)
    trigger.setAttribute('aria-expanded', String(shouldOpen))
    return
  }
  const option = (event.target as HTMLElement).closest<HTMLElement>('.select-option')
  if (option) {
    const select = option.closest<HTMLElement>('.md-select')!
    const setting = select.dataset.setting
    const value = option.dataset.value!
    if (setting === 'quality') state.quality = value
    if (setting === 'downloadType') state.type = value
    if (setting === 'cookieBrowser') state.cookieBrowser = value
    if (setting === 'maxConcurrent') state.maxConcurrent = Math.max(1, Math.min(6, parseInt(value!, 10) || 3))
    if (setting === 'autoRetry') state.autoRetry = Math.max(0, Math.min(5, parseInt(value!, 10) || 0))
    if (setting === 'videoCodec') state.videoCodec = value === 'hevc' ? 'hevc' : 'auto'
    if (setting === 'hevcEncoder') state.hevcEncoder = value === 'cpu' ? 'cpu' : 'auto'
    state.settingsDirty = true
    select.querySelector<HTMLElement>('.select-current')!.textContent = option.querySelector<HTMLElement>('span')!.textContent
    select.querySelectorAll<HTMLElement>('.select-option').forEach(item => {
      const selected = item === option
      item.classList.toggle('selected', selected)
      item.setAttribute('aria-selected', String(selected))
    })
    closeSelect(select)
    if (setting === 'cookieBrowser') refreshBrowserRunning()
    return
  }
  if (!(event.target as HTMLElement).closest<HTMLElement>('.md-select')) closeAllSelects()
})

root.addEventListener('keydown', event => {
  const select = (event.target as HTMLElement).closest<HTMLElement>('.md-select')
  if (!select) return
  const options = [...select.querySelectorAll<HTMLElement>('.select-option')]
  const trigger = select.querySelector('.select-trigger')
  if (event.target === trigger && ['ArrowDown', 'ArrowUp'].includes((event as KeyboardEvent).key)) {
    event.preventDefault()
    closeAllSelects(select)
    select.classList.add('is-open')
    trigger!.setAttribute('aria-expanded', 'true')
    const selectedIndex = Math.max(0, options.findIndex(option => option.getAttribute('aria-selected') === 'true'))
    options[(event as KeyboardEvent).key === 'ArrowDown' ? selectedIndex : (selectedIndex || options.length - 1)].focus()
    return
  }
  if (!(event.target as HTMLElement).matches('.select-option')) return
  const index = options.indexOf(event.target as HTMLElement)
  if ((event as KeyboardEvent).key === 'Escape') { event.preventDefault(); closeSelect(select, true) }
  else if ((event as KeyboardEvent).key === 'ArrowDown' || (event as KeyboardEvent).key === 'ArrowUp') {
    event.preventDefault()
    options[(index + ((event as KeyboardEvent).key === 'ArrowDown' ? 1 : -1) + options.length) % options.length].focus()
  } else if ((event as KeyboardEvent).key === 'Home' || (event as KeyboardEvent).key === 'End') {
    event.preventDefault(); options[(event as KeyboardEvent).key === 'Home' ? 0 : options.length - 1].focus()
  }
})
document.addEventListener('pointerdown', event => { if (!(event.target as HTMLElement).closest<HTMLElement>('.md-select')) closeAllSelects() })

function bind() {
  // 标题栏
  $('#themeToggle')?.addEventListener('click', toggleTheme)
  $('#minimizeWindow')?.addEventListener('click', () => window.runtime?.WindowMinimise?.())
  $('#maximizeWindow')?.addEventListener('click', () => { window.runtime?.WindowToggleMaximise?.(); setTimeout(syncWindowButtons, 80) })
  $('#closeWindow')?.addEventListener('click', () => window.runtime?.Quit?.())
  $('#windowTitlebar')?.addEventListener('dblclick', event => {
    if ((event.target as HTMLElement).closest<HTMLElement>('.window-controls') || (event.target as HTMLElement).closest<HTMLElement>('.titlebar-tools')) return
    window.runtime?.WindowToggleMaximise?.(); setTimeout(syncWindowButtons, 80)
  })

  // 下载页
  $('#stepAdd')?.addEventListener('click', () => {
    if (state.queueStep !== 'add') { state.queueStep = 'add'; scrollReset = true; render(); $('#linkInput')?.focus() }
  })
  $('#stepList')?.addEventListener('click', () => {
    if (state.queueStep !== 'list') { state.queueStep = 'list'; scrollReset = true; render() }
  })
  const linkInput = $('#linkInput')
  linkInput?.addEventListener('input', e => { state.draftLink = (e.target as HTMLInputElement).value })
  linkInput?.addEventListener('keydown', e => { if (e.ctrlKey && (e as KeyboardEvent).key === 'Enter') parseLinks() })
  $('#pasteLinks')?.addEventListener('click', pasteFromClipboard)
  $('#parseBtn')?.addEventListener('click', parseLinks)
  $('#searchInput')?.addEventListener('input', e => { state.filter = (e.target as HTMLInputElement).value; refreshQueueList() })
  $('#clearBtn')?.addEventListener('click', async () => {
    if (backend()?.ClearTasks) applySnapshot(await backend()!.ClearTasks())
    else state.tasks = []
    render(); toast('下载队列已清空')
  })
  $('#openFolderBtn')?.addEventListener('click', openFolder)
  $('#engineSetupBanner')?.addEventListener('click', () => backend()?.OpenEngineDownloads?.())
  $('#ffmpegSetup')?.addEventListener('click', () => { state.ffmpegModal = true; render() })

  // 设置页
  $('#resetSettings')?.addEventListener('click', resetSettings)
  $('#revertSettings')?.addEventListener('click', revertSettings)
  $('#saveSettings')?.addEventListener('click', saveSettings)
  $('#browseFolder')?.addEventListener('click', browse)
  $('#browseCookieFile')?.addEventListener('click', browseCookieFile)
  $('#openCookieFolder')?.addEventListener('click', () => backend()?.OpenCookieFolder?.())
  $('#folderPath')?.addEventListener('change', e => { state.folder = (e.target as HTMLInputElement).value; state.settingsDirty = true })
  $('#cookieFilePath')?.addEventListener('change', e => { state.cookieFile = (e.target as HTMLInputElement).value; state.settingsDirty = true })
  $('#highFrameRate')?.addEventListener('change', e => { state.highFrameRate = (e.target as HTMLInputElement).checked; state.settingsDirty = true })
  $('#openOnComplete')?.addEventListener('change', e => { state.openOnComplete = (e.target as HTMLInputElement).checked; state.settingsDirty = true })
  $('#proxyToggle')?.addEventListener('change', e => {
    state.useProxy = (e.target as HTMLInputElement).checked
    state.settingsDirty = true
    $('#proxyAddressField')?.classList.toggle('hidden', !(e.target as HTMLInputElement).checked)
  })
  $('#autoDetectToggle')?.addEventListener('change', e => { state.autoDetectProxy = (e.target as HTMLInputElement).checked; state.settingsDirty = true })
  $('#proxyURL')?.addEventListener('change', e => { state.proxyURL = (e.target as HTMLInputElement).value; state.settingsDirty = true })
  $('#testProxy')?.addEventListener('click', testProxy)
  $('#cookiesToggle')?.addEventListener('change', e => {
    state.useBrowserCookie = (e.target as HTMLInputElement).checked
    state.settingsDirty = true
    $('#cookiePanel')?.classList.toggle('hidden', !(e.target as HTMLInputElement).checked)
    if ((e.target as HTMLInputElement).checked) refreshBrowserRunning()
  })
  $('#engineSetupBtn')?.addEventListener('click', () => backend()?.OpenEngineDownloads?.())
  $('#ffmpegManageBtn')?.addEventListener('click', () => { state.ffmpegModal = true; render() })
  $('#openDataFolder')?.addEventListener('click', openDataFolder)

  // 关于页
  $('#linkGithub')?.addEventListener('click', () => backend()?.OpenProjectRepo?.())
  $('#linkCnb')?.addEventListener('click', () => backend()?.OpenCNBRepo?.())
  $('#linkBilibili')?.addEventListener('click', () => backend()?.OpenBilibili?.())
  $('#linkHomepage')?.addEventListener('click', () => backend()?.OpenHomepage?.())

  // 百宝箱:推荐
  $('#tbOpenRepo')?.addEventListener('click', () => backend()?.OpenExternalURL?.(TAMPERMONKEY_REPO))
  $('#tbCopyRepo')?.addEventListener('click', async () => {
    try {
      await window.runtime?.ClipboardSetText?.(TAMPERMONKEY_REPO)
      toast('仓库地址已复制')
    } catch { toast('复制失败,请手动选中复制') }
  })

  // 百宝箱:FFmpeg 工具
  $('#tbPickVideo')?.addEventListener('click', pickToolboxVideo)
  $('#tbPickAudio')?.addEventListener('click', pickToolboxAudio)
  $('#tbMerge')?.addEventListener('click', () => startToolboxTask('merge'))
  $('#tbSplit')?.addEventListener('click', () => startToolboxTask('split'))
  $('#tbCancel')?.addEventListener('click', cancelToolboxTask)
  $('#tbReencode')?.addEventListener('change', (e) => {
    state.toolboxReencode = (e.target as HTMLInputElement).checked
  })

  // 版本更新
  $('#checkUpdateBtn')?.addEventListener('click', () => checkUpdate(true))
  $('#downloadUpdateBtn')?.addEventListener('click', downloadUpdate)
  $('#installUpdateBtn')?.addEventListener('click', installUpdate)
  $('#openReleaseBtn')?.addEventListener('click', () => {
    const url = state.updatePageUrl || state.updateUrl
    if (url) backend()?.OpenUpdatePage?.(url)
  })

  // FFmpeg 弹窗
  $('#closeFFmpegModal')?.addEventListener('click', closeFFmpegModal)
  $('#ffmpegModalCancel')?.addEventListener('click', closeFFmpegModal)
  $('#ffmpegBackdrop')?.addEventListener('click', e => { if ((e.target as HTMLElement).id === 'ffmpegBackdrop') closeFFmpegModal() })
  $('#ffmpegAutoInstall')?.addEventListener('click', installFFmpeg)
  $('#ffmpegImportArchive')?.addEventListener('click', importFFmpegArchive)
  $('#ffmpegPickFile')?.addEventListener('click', pickFFmpegFile)
}

async function syncWindowButtons() {
  if (!window.runtime?.WindowIsMaximised) return
  try {
    state.windowMaximized = await window.runtime.WindowIsMaximised()
    const button = $('#maximizeWindow')
    if (button) {
      button.innerHTML = icon(state.windowMaximized ? 'restore' : 'maximize', 13)
      button.title = state.windowMaximized ? '还原' : '最大化'
      button.setAttribute('aria-label', button.title)
    }
  } catch { /* 浏览器预览中没有窗口 API */ }
}

// ============================================================
// 下载流程
// ============================================================
// looksLikeManyVideos 判断链接是否可能对应「视频很多」的页面（播放列表、频道、合集、UP 主空间）。
// 这类页面的地址看起来和单个视频一模一样，但下载内核会把页面下的每个视频都拉下来，
// 而任务列表里只显示一条，很容易让人以为卡住了或在反复重试。
// 这里只做提示、不阻断 —— 用户可能确实想要整张列表。
const MANY_VIDEOS_PATTERNS = [
  /[?&]list=/,            // YouTube 播放列表参数
  /\/playlist\b/,
  /\/channel\//,
  /\/@[\w.\-]+/,
  /\/c\/[\w.\-]+/,
  /\/user\//,
  /\/videos\b/,
  /\/series\b/,
  /\/album\b/,
  /space\.bilibili\.com/,  // B 站 UP 主空间
  /\/medialist\//,
  /\/bangumi\//,
  /\/festival\//,
  /collection/,
  /[?&]sid=/,            // B 站合集参数
  /[?&]season_id=/,
]
const looksLikeManyVideos = (raw: string) => {
  try {
    const u = new URL(raw)
    // 域名也要参与匹配:B 站 UP 主空间是 space.bilibili.com/<数字>，特征在 host 上
    const target = u.host + u.pathname + u.search
    return MANY_VIDEOS_PATTERNS.some(re => re.test(target))
  } catch { return false }
}

async function parseLinks() {
  const input = $('#linkInput')
  const raw = (state.draftLink || '').trim()
  const parts: string[] = raw.split(/[\n\r\t ,;]+/).map((s: string) => s.trim()).filter(Boolean)
  const urls = parts.filter(isSupportedURL)
  let list = urls
  if (state.linkMode === 'single') {
    if (urls.length > 1) toast('单个视频模式只解析第一条链接')
    list = urls.slice(0, 1)
  }
  if (!list.length) {
    input?.classList.add('invalid')
    input?.focus()
    toast('没有找到可下载的链接，请检查粘贴的内容')
    return
  }
  let snapshot
  if (backend()?.AddLinks) snapshot = await backend()!.AddLinks(list.join('\n'))
  else {
    // 没有后端桥时(浏览器里调试)的兜底:先造占位任务,后端就绪后会用真实快照覆盖
    const fresh: Task[] = list.map((url: string, i: number) => ({ id: Date.now() + i, url, title: '正在获取视频信息…', status: '等待中', progress: 0, quality: state.quality, createdAt: '' } as Task))
    state.tasks.unshift(...fresh)
  }
  if (snapshot) applySnapshot(snapshot)
  state.draftLink = ''
  state.queueStep = 'list'
  scrollReset = true
  render()
  // 提示放在成功提示之后：先告诉用户已加入，再补一句风险提醒
  toast(list.length > 1 ? `已加入 ${list.length} 个任务` : '已加入 1 个任务')
  const risky = list.filter(looksLikeManyVideos)
  if (risky.length) {
    setTimeout(() => toast('其中 ' + risky.length + ' 个链接像是播放列表或合集页，会把页面下的每个视频都逐个下载，可能需要很久'), 2600)
  }
}

async function pasteFromClipboard() {
  let text = ''
  try { if (window.runtime?.ClipboardGetText) text = await window.runtime.ClipboardGetText() } catch { /* 忽略 */ }
  if (!text) { try { text = await navigator.clipboard.readText() } catch { /* 忽略 */ } }
  text = (text || '').trim()
  if (!text) { toast('剪贴板里没有可用的文本'); return }
  const existing = state.draftLink.trim()
  state.draftLink = existing ? existing + '\n' + text : text
  render()
  const input = $<HTMLInputElement>('#linkInput')
  input?.focus()
  input?.setSelectionRange(state.draftLink.length, state.draftLink.length)
}

async function handleTask(action: string, id: number) {
  const t = state.tasks.find(x => x.id === id)
  if (action === 'remove') {
    if (backend()?.RemoveTask) applySnapshot(await backend()!.RemoveTask(id))
    else state.tasks = state.tasks.filter(x => x.id !== id)
    render(); toast('已从队列移除'); return
  }
  if (!t) return
  if (action === 'start') {
    if (t.status === '已完成') { toast('该任务已经下载完毕'); return }
    if (backend()?.StartTask) applySnapshot(await backend()!.StartTask(id))
    else { t.status = '失败'; t.error = '桌面版下载引擎尚未配置' }
    const fresh = state.tasks.find(x => x.id === id)
    toast(fresh && fresh.status === '失败' ? '下载引擎尚未配置' : '任务已开始')
  }
  if (action === 'pause') {
    if (backend()?.PauseTask) applySnapshot(await backend()!.PauseTask(id))
    else t.status = '已暂停'
    toast('下载已暂停')
  }
  render()
}

async function startAll() {
  const startable = state.tasks.some(t => ['等待中', '已暂停', '失败'].includes(t.status))
  if (state.tasks.length && !startable) {
    toast(state.tasks.some(t => t.status === '已完成') ? '已经下载完毕' : '没有可开始的任务')
    return
  }
  if (backend()?.StartAllTasks) applySnapshot(await backend()!.StartAllTasks())
  else toast('全部开始需要在桌面程序中运行')
  render()
}

async function pauseAll() {
  if (backend()?.PauseAllTasks) applySnapshot(await backend()!.PauseAllTasks())
  else state.tasks.forEach(t => { if (t.status === '准备中') t.status = '已暂停' })
  render(); toast('下载任务已暂停')
}

async function openFolder() {
  if (backend()?.OpenDownloadFolder) {
    try { await backend()!.OpenDownloadFolder() } catch (error) { toast(`无法打开文件夹：${(error as Error).message}`) }
  } else toast('请在桌面版中打开下载文件夹')
}

async function openDataFolder() {
  if (backend()?.OpenDataFolder) {
    try { await backend()!.OpenDataFolder() } catch (error) { toast(`无法打开数据目录：${(error as Error).message}`) }
  } else toast('请在桌面版中打开数据目录')
}

// ============================================================
// 版本更新
// ============================================================
async function checkUpdate(manual = false) {
  const api = backend()
  if (!api?.CheckUpdate) { if (manual) toast('请在桌面版中检查更新'); return }
  if (state.updateChecking) return
  state.updateChecking = true
  renderUpdateArea()
  try {
    const result = await api.CheckUpdate(APP_VERSION)
    state.updateChecking = false
    state.updateChecked = true
    if (!result?.ok) {
      state.updateHas = false
      state.updateError = result?.error || '检查更新失败'
      if (manual) toast(state.updateError)
    } else if (result.hasUpdate) {
      const versionChanged = state.updateLatest && state.updateLatest !== result.latest
      if (versionChanged) { state.updatePhase = ''; state.updateProgress = 0 }
      state.updateHas = true
      state.updateLatest = result.latest || ''
      state.updateNotes = result.notes || ''
      state.updateUrl = result.url || ''
      state.updatePageUrl = result.pageUrl || ''
      state.updateSource = result.source || ''
      state.updateError = ''
      if (manual) toast(`发现新版本 v${result.latest}`)
    } else {
      state.updateHas = false
      state.updateError = ''
      if (manual) toast(`已是最新版本 v${APP_VERSION}`)
    }
  } catch (error) {
    state.updateChecking = false
    state.updateChecked = true
    state.updateHas = false
    state.updateError = (error as Error).message || '检查更新失败'
    if (manual) toast(`检查更新失败：${state.updateError}`)
  }
  markUpdateBadge()
  renderUpdateArea()
}

async function downloadUpdate() {
  const api = backend()
  if (!api?.DownloadUpdate || !state.updateUrl) return
  state.updatePhase = 'downloading'
  state.updateProgress = 0
  state.updateMessage = '正在连接下载源…'
  state.updateError = ''
  renderUpdateArea()
  try {
    const result = await api.DownloadUpdate(state.updateUrl, state.updateLatest)
    if (result?.ok) {
      state.updatePhase = 'downloaded'
      state.updateProgress = 100
      renderUpdateArea()
      toast('更新包已下载完成')
      return
    }
    state.updatePhase = ''
    state.updateError = result?.error || '下载更新失败'
    renderUpdateArea()
    toast(state.updateError)
  } catch (error) {
    state.updatePhase = ''
    state.updateError = (error as Error).message || '下载更新失败'
    renderUpdateArea()
    toast(`下载更新失败：${state.updateError}`)
  }
}

async function installUpdate() {
  const api = backend()
  if (!api?.InstallUpdate) return
  if (state.updatePhase === 'installing') return
  state.updatePhase = 'installing'
  state.updateError = ''
  renderUpdateArea()
  try {
    const result = await api.InstallUpdate(state.updateLatest)
    if (!result?.ok) {
      state.updatePhase = 'downloaded'
      state.updateError = result?.error || '无法启动安装程序'
      renderUpdateArea()
      toast(state.updateError)
      return
    }
    // 安装程序已启动:安装包会结束本进程,完成后自动重启新版
    state.updatePhase = 'installing'
    renderUpdateArea()
  } catch (error) {
    state.updatePhase = 'downloaded'
    state.updateError = (error as Error).message || '无法启动安装程序'
    renderUpdateArea()
    toast(`无法启动安装程序：${state.updateError}`)
  }
}

// renderUpdateArea 只重绘关于页的更新区块,避免整页重绘打断交互。
function renderUpdateArea() {
  if (state.page !== 'about') return
  const btn = $('#checkUpdateBtn')
  if (btn) {
    (btn as HTMLButtonElement).disabled = state.updateChecking
    btn.textContent = state.updateChecking ? '检查中…' : '检查更新'
  }
  const block = $('#updateArea')
  if (!block) { render(); return }
  block.outerHTML = `<div id="updateArea">${renderUpdateBlock()}</div>`
  bindUpdateArea()
}

function bindUpdateArea() {
  $('#downloadUpdateBtn')?.addEventListener('click', downloadUpdate)
  $('#installUpdateBtn')?.addEventListener('click', installUpdate)
  $('#openReleaseBtn')?.addEventListener('click', () => {
    const url = state.updatePageUrl || state.updateUrl
    if (url) backend()?.OpenUpdatePage?.(url)
  })
}

// ============================================================
// 设置页逻辑
// ============================================================
function settingsSnapshot() {
  return {
    quality: state.quality, type: state.type, folder: state.folder,
    useProxy: state.useProxy, proxyURL: state.proxyURL, autoDetectProxy: state.autoDetectProxy,
    useBrowserCookie: state.useBrowserCookie, cookieSource: state.cookieSource,
    cookieBrowser: state.cookieBrowser, cookieFile: state.cookieFile,
    highFrameRate: state.highFrameRate, openOnComplete: state.openOnComplete,
    maxConcurrent: state.maxConcurrent, autoRetry: state.autoRetry,
    videoCodec: state.videoCodec,
    hevcEncoder: state.hevcEncoder,
  }
}
function captureSettingsBaseline() {
  state.settingsBaseline = settingsSnapshot()
  state.settingsDirty = false
}
function revertSettings() {
  if (!state.settingsBaseline) { toast('没有可撤销的修改'); return }
  Object.assign(state, state.settingsBaseline, { settingsDirty: false })
  render(); toast('已撤销未保存的修改')
}
function resetSettings() {
  state.quality = '1080p'
  state.type = '视频（带音频）'
  state.useProxy = false
  state.autoDetectProxy = true
  state.useBrowserCookie = false
  state.highFrameRate = true
  state.openOnComplete = false
  state.maxConcurrent = 3
  state.autoRetry = 2
  state.videoCodec = 'auto'
  state.hevcEncoder = 'auto'
  state.settingsDirty = true
  render(); toast('已恢复默认值，点击「保存」生效')
}

async function browse() {
  if (window.go?.main?.App?.SelectFolder) { const dir = await window.go.main.App.SelectFolder(); if (dir) { state.folder = dir; state.settingsDirty = true } }
  else toast('桌面版中可使用原生文件夹选择器')
  render()
}

async function browseCookieFile() {
  const api = backend()
  if (!api?.SelectCookieFile) { toast('桌面版中可使用文件选择器'); return }
  try {
    const result = await api.SelectCookieFile()
    if (!result || result.canceled) return
    if (!result.ok) { toast(result.message || 'Cookies 文件不可用'); return }
    state.cookieFile = result.path!
    state.settingsDirty = true
    render()
    toast(result.note ? `已选择 Cookies 文件：${result.note}` : '已选择 Cookies 文件')
  } catch (error) { toast(`选择文件失败：${(error as Error).message}`) }
}

function setCookieSource(source: string) {
  state.cookieSource = source
  state.settingsDirty = true
  root.querySelectorAll<HTMLElement>('.cookie-tab').forEach(tab => {
    const active = tab.dataset.cookieSource === source
    tab.classList.toggle('active', active)
    tab.setAttribute('aria-selected', String(active))
  })
  $('#cookieSourceBrowser')?.classList.toggle('hidden', source !== 'browser')
  $('#cookieSourceFile')?.classList.toggle('hidden', source !== 'file')
  if (source === 'browser') refreshBrowserRunning()
  else refreshCookieStatus()
}

// refreshCookieStatus 拉取各站点已导入的 Cookies 状态。
async function refreshCookieStatus() {
  const api = backend()
  if (!api?.CookieStatus) return
  try {
    const status = await api.CookieStatus()
    state.cookiePlatforms = Array.isArray(status?.platforms) ? status.platforms : []
    state.cookieDir = status?.dir || ''
    patchCookiePlatforms()
  } catch { /* 读取失败时保持现状 */ }
}

// patchCookiePlatforms 只替换分站点列表，避免整页重绘打断正在编辑的输入框。
function patchCookiePlatforms() {
  const list = $('.cookie-plat-list')
  if (!list) return
  list.innerHTML = state.cookiePlatforms.map(renderCookiePlatformRow).join('')
}

// importPlatformCookie 选择一个 cookies.txt，由后端识别站点并分别保存。
async function importPlatformCookie(platformId: string) {
  const api = backend()
  if (!api?.ImportPlatformCookie) { toast('桌面版中可使用文件选择器'); return }
  state.cookieBusy = platformId || ''
  patchCookiePlatforms()
  try {
    const result = await api.ImportPlatformCookie()
    if (!result || result.canceled) { state.cookieBusy = ''; patchCookiePlatforms(); return }
    if (!result.ok) {
      toast(result.message || '导入失败')
      state.cookieBusy = ''
      patchCookiePlatforms()
      return
    }
    const saved = Array.isArray(result.saved) ? result.saved : []
    state.cookieBusy = ''
    await refreshCookieStatus()
    const summary = saved.map(item => `${item.name} ${item.count} 条`).join('、')
    const dropped = Number(result.dropped) || 0
    toast(`已保存 ${summary}${dropped ? `，忽略了 ${dropped} 条其他站点的 Cookie` : ''}`)
  } catch (error) {
    state.cookieBusy = ''
    patchCookiePlatforms()
    toast(`导入失败：${(error as Error).message}`)
  }
}

// removePlatformCookie 删除某个站点已保存的 Cookies。
async function removePlatformCookie(platformId: string) {
  const api = backend()
  if (!api?.RemovePlatformCookie) return
  const target = state.cookiePlatforms.find(p => p.id === platformId)
  state.cookieBusy = platformId
  patchCookiePlatforms()
  try {
    const result = await api.RemovePlatformCookie(platformId)
    state.cookieBusy = ''
    if (!result?.ok) { toast(result?.message || '移除失败'); patchCookiePlatforms(); return }
    await refreshCookieStatus()
    toast(`已移除 ${target?.name || ''} 的 Cookies`)
  } catch (error) {
    state.cookieBusy = ''
    patchCookiePlatforms()
    toast(`移除失败：${(error as Error).message}`)
  }
}

async function refreshBrowserRunning() {
  const api = backend()
  if (!api?.CheckBrowserRunning) return
  if (state.page !== 'settings' || !state.useBrowserCookie || state.cookieSource !== 'browser') return
  try {
    const running = await api.CheckBrowserRunning(state.cookieBrowser)
    if (running !== state.browserRunning) {
      state.browserRunning = running
      const hint = $('#browserRunningHint')
      if (hint) {
        hint.classList.toggle('warn', running)
        hint.textContent = running
          ? state.cookieBrowser + ' 正在运行 —— 读取 Cookies 会失败，请彻底退出后再试'
          : '此方式需完全退出浏览器(含后台进程)后使用'
      }
    }
  } catch { /* 检测失败时保持现状 */ }
}

async function saveSettings() {
  const settings = {
    quality: state.quality,
    downloadType: state.type,
    downloadDir: $<HTMLInputElement>('#folderPath')?.value || state.folder,
    useProxy: state.useProxy,
    proxyURL: $<HTMLInputElement>('#proxyURL')?.value || state.proxyURL,
    autoDetectProxy: state.autoDetectProxy,
    useBrowserCookie: state.useBrowserCookie,
    cookieSource: state.cookieSource,
    cookieBrowser: state.cookieBrowser,
    cookieFile: $<HTMLInputElement>('#cookieFilePath')?.value || state.cookieFile,
    ffmpegPath: state.ffmpegPath,
    highFrameRate: state.highFrameRate,
    openOnComplete: state.openOnComplete,
    maxConcurrent: state.maxConcurrent,
    autoRetry: state.autoRetry,
    videoCodec: state.videoCodec,
    hevcEncoder: state.hevcEncoder,
  }
  state.folder = settings.downloadDir
  state.proxyURL = settings.proxyURL
  state.settingsDirty = false
  if (backend()?.SaveSettings) applySnapshot(await backend()!.SaveSettings(settings))
  state.encoderHint = ''
  state.settingsBaseline = settingsSnapshot()
  await refreshSystemStatus()
  render(); toast('设置已保存到本机')
}

// 检测并展示可用的 H.265 显卡编码器(探测结果由后端缓存,仅首次较慢)
async function refreshEncoderHint() {
  const el = $('#encoderHint')
  if (!el) return
  if (state.encoderHint) { el.textContent = state.encoderHint; return }
  el.textContent = '正在检测显卡编码器…'
  const api = backend()
  if (!api?.GetHEVCEncoderStatus) { el.textContent = ''; return }
  try {
    const st = await api.GetHEVCEncoderStatus()
    let text = ''
    if (!st || st.ok === false) text = (st && st.message) || ''
    else if (st.gpu) text = `已检测到 ${st.label}，转码速度更快`
    else text = st.message || '未检测到可用的显卡编码器，将使用 CPU 转码'
    state.encoderHint = text
    const live = $('#encoderHint')
    if (live) live.textContent = text
  } catch { el.textContent = '' }
}

async function testProxy() {
  if (!state.useProxy) { toast('请先启用代理并填写代理地址'); return }
  if (backend()?.TestProxy) {
    try {
      const result = await backend()!.TestProxy($<HTMLInputElement>('#proxyURL')?.value || state.proxyURL)
      state.proxyChecked = true
      state.proxyReady = !!result.ok
      state.proxyMessage = result.message
      render(); toast(result.message)
    } catch (error) { toast(`代理测试失败：${(error as Error).message}`) }
  } else toast('代理测试仅在桌面版可用')
}

// ============================================================
// 系统状态 / FFmpeg
// ============================================================
async function refreshSystemStatus() {
  if (!backend()?.GetSystemStatus) return
  try {
    const status = await backend()!.GetSystemStatus()
    state.engineReady = !!status.engineReady
    state.engineError = status.engineError || ''
    state.ffmpegReady = !!status.ffmpegReady
    state.ffmpegPath = status.ffmpegPath || state.ffmpegPath
    state.ffmpegVersion = status.ffmpegVersion || ''
    state.dataDir = status.dataDir || state.dataDir
    state.proxyChecked = !!status.proxyChecked
    state.proxyReady = !!status.proxyReady
    state.proxyMessage = status.proxyMessage || ''
    render()
  } catch (error) { console.error(error) }
}

function closeFFmpegModal() {
  if (state.ffmpegBusy) return
  state.ffmpegModal = false
  state.ffmpegPhase = ''
  state.ffmpegProgress = 0
  state.ffmpegMessage = ''
  render()
}

// ============================================================
// 百宝箱:FFmpeg 工具的交互
// 输出目录由后端每次弹窗询问,前端只负责选文件、看进度、拿结果。
// ============================================================

async function pickToolboxVideo() {
  const api = backend()
  if (!api?.PickToolboxVideo) { toast('桌面版中可使用文件选择器'); return }
  try {
    const res = await api.PickToolboxVideo()
    if (!res || res.canceled) return
    if (!res.ok) { toast(res.message || '无法读取该视频'); return }
    state.toolboxVideo = res.path!
    state.toolboxVideoInfo = res.version || ''
    state.toolboxResult = ''
    render()
  } catch (error) { toast(`选择视频失败：${(error as Error).message}`) }
}

async function pickToolboxAudio() {
  const api = backend()
  if (!api?.PickToolboxAudio) { toast('桌面版中可使用文件选择器'); return }
  try {
    const res = await api.PickToolboxAudio()
    if (!res || res.canceled) return
    if (!res.ok) { toast(res.message || '无法读取该音频'); return }
    state.toolboxAudio = res.path!
    state.toolboxAudioInfo = res.version || ''
    state.toolboxResult = ''
    render()
  } catch (error) { toast(`选择音频失败：${(error as Error).message}`) }
}

/** 启动合并或分离。文件校验、输出目录弹窗都在后端,前端只管发起。 */
async function startToolboxTask(kind: 'merge' | 'split') {
  const api = backend()
  if (!api) return
  if (kind === 'merge' && (!state.toolboxVideo || !state.toolboxAudio)) {
    toast('请先选择视频文件与音频文件'); return
  }
  if (kind === 'split' && !state.toolboxVideo) {
    toast('请先选择视频文件'); return
  }
  state.toolboxBusy = true
  state.toolboxProgress = 0
  state.toolboxResult = ''
  state.toolboxMessage = kind === 'merge' ? '正在准备合并…' : '正在准备分离…'
  render()
  try {
    const res = kind === 'merge'
      ? await api.MergeVideoAudio!(state.toolboxVideo, state.toolboxAudio, state.toolboxReencode)
      : await api.SplitVideoStream!(state.toolboxVideo, state.toolboxReencode)
    if (!res || res.canceled) {
      // 用户在输出目录弹窗点了取消
      state.toolboxBusy = false
      state.toolboxMessage = ''
      state.toolboxResult = ''
      render()
      return
    }
    if (!res.ok) {
      state.toolboxBusy = false
      state.toolboxProgress = 0
      state.toolboxMessage = ''
      state.toolboxResult = `失败：${res.message || '无法启动'}`
      render()
      toast(res.message || '无法启动')
    }
    // ok 时保持 busy,等 toolbox:progress / toolbox:finished 事件推进
  } catch (error) {
    state.toolboxBusy = false
    state.toolboxMessage = ''
    state.toolboxResult = `失败：${(error as Error).message}`
    render()
  }
}

async function cancelToolboxTask() {
  if (!state.toolboxBusy) return
  await backend()?.CancelToolboxTask?.()
}

/** 进度事件:更新进度条,不整页重绘(避免打断文件选择等交互) */
function patchToolboxProgress() {
  const wrap = $('#tbProgressWrap')
  if (!wrap) { render(); return }
  wrap.hidden = state.toolboxBusy || !!state.toolboxResult
  const phase = $('#tbPhase')
  if (phase) phase.textContent = state.toolboxMessage || (state.toolboxBusy ? '处理中…' : '已完成')
  const percent = $('#tbPercent')
  if (percent) percent.textContent = `${Math.round(state.toolboxProgress)}%`
  const bar = $('#tbProgressBar')
  if (bar) bar.style.width = `${Math.round(state.toolboxProgress)}%`
  const cancel = $('#tbCancel')
  if (cancel) (cancel as HTMLButtonElement).disabled = !state.toolboxBusy
}

async function finishFFmpegResult(result: any) {
  if (result && result.canceled) {
    state.ffmpegBusy = false
    state.ffmpegPhase = ''
    state.ffmpegMessage = ''
    render()
    return
  }
  if (result && result.ok) {
    state.ffmpegBusy = false
    state.ffmpegPhase = 'done'
    state.ffmpegProgress = 100
    state.ffmpegMessage = '安装完成'
    if (result.version) state.ffmpegVersion = result.version
    if (result.path) state.ffmpegPath = result.path
    render()
    toast('FFmpeg 已就绪')
    await refreshSystemStatus()
    setTimeout(() => {
      if (state.ffmpegPhase === 'done') {
        state.ffmpegModal = false
        state.ffmpegPhase = ''
        state.ffmpegProgress = 0
        state.ffmpegMessage = ''
        render()
      }
    }, 1500)
  } else {
    state.ffmpegBusy = false
    state.ffmpegPhase = ''
    state.ffmpegMessage = ''
    render()
    toast((result && result.message) || '操作未完成，请重试')
  }
}

async function installFFmpeg() {
  const api = backend()
  if (!api?.InstallFFmpeg) { toast('该功能需要在桌面程序中运行'); return }
  if (state.ffmpegBusy) return
  state.ffmpegBusy = true
  state.ffmpegProgress = 0
  state.ffmpegPhase = 'download'
  state.ffmpegMessage = '正在准备下载…'
  render()
  try {
    await finishFFmpegResult(await api.InstallFFmpeg())
  } catch (error) {
    state.ffmpegBusy = false
    state.ffmpegPhase = ''
    state.ffmpegMessage = ''
    render()
    toast('安装失败：' + ((error as Error).message || '未知错误'))
  }
}

async function importFFmpegArchive() {
  const api = backend()
  if (!api?.ImportFFmpegArchive) { toast('该功能需要在桌面程序中运行'); return }
  if (state.ffmpegBusy) return
  state.ffmpegBusy = true
  state.ffmpegProgress = 0
  state.ffmpegPhase = 'extract'
  state.ffmpegMessage = '等待选择安装包…'
  render()
  try {
    await finishFFmpegResult(await api.ImportFFmpegArchive())
  } catch (error) {
    state.ffmpegBusy = false
    state.ffmpegPhase = ''
    state.ffmpegMessage = ''
    render()
    toast('导入失败：' + ((error as Error).message || '未知错误'))
  }
}

async function pickFFmpegFile() {
  const api = backend()
  if (!api?.SelectFFmpegFile) { toast('该功能需要在桌面程序中运行'); return }
  try {
    const result = await api.SelectFFmpegFile()
    if (result && result.canceled) return
    if (result && result.ok) {
      if (result.path) state.ffmpegPath = result.path
      if (result.version) state.ffmpegVersion = result.version
      state.ffmpegModal = false
      render()
      toast('FFmpeg 已配置')
      await refreshSystemStatus()
    } else {
      toast((result && result.message) || '所选文件无效')
    }
  } catch (error) { toast('选择失败：' + ((error as Error).message || '未知错误')) }
}

function updateFFmpegProgressUI() {
  const bar = $('#ffmpegProgressBar')
  if (!bar) return
  const track = bar.closest<HTMLElement>('.ff-track')
  const pct = $('#ffmpegProgressPct')
  const label = $('#ffmpegProgressLabel')
  const indeterminate = state.ffmpegPhase === 'extract' && state.ffmpegProgress < 100
  track?.classList.toggle('indeterminate', indeterminate)
  bar.style.width = state.ffmpegProgress + '%'
  if (label && state.ffmpegMessage) label.textContent = state.ffmpegMessage
  if (pct) pct.textContent = indeterminate ? '' : state.ffmpegProgress + '%'
}

// ============================================================
// 启动
// ============================================================
render()
refreshSnapshot()
refreshSystemStatus()
setInterval(() => { if (state.page === 'settings') refreshBrowserRunning() }, 4000)
document.addEventListener('contextmenu', event => {
  if ((event.target as HTMLElement).closest<HTMLElement>('input, textarea')) return
  event.preventDefault()
})
window.addEventListener('resize', syncWindowButtons)

if (window.runtime?.EventsOn) {
  window.runtime.EventsOn('state:updated', (snapshot) => {
    applySnapshot(snapshot)
    if (state.page === 'downloads') patchDownloads()
    else {
      const badge = $('.nav-item[data-page="downloads"] em')
      if (badge) badge.textContent = state.tasks.length ? String(state.tasks.length) : ''
    }
  })
  window.runtime.EventsOn('ffmpeg:progress', (data) => {
    if (!data) return
    if (typeof data.percent === 'number') state.ffmpegProgress = data.percent
    if (data.phase) state.ffmpegPhase = data.phase
    if (data.message) state.ffmpegMessage = data.message
    if (!$('#ffmpegProgressBar')) { render(); return }
    updateFFmpegProgressUI()
  })
  window.runtime.EventsOn('update:progress', (data) => {
    if (!data) return
    if (typeof data.percent === 'number') state.updateProgress = data.percent
    if (data.message) state.updateMessage = data.message
    if (state.updatePhase !== 'downloading') return
    const bar = $('#updateProgressBar')
    if (!bar) return
    bar.style.width = state.updateProgress + '%'
    const pct = $('#updateProgressPct')
    if (pct) pct.textContent = state.updateProgress > 0 ? state.updateProgress + '%' : ''
    const label = $('#updateProgressLabel')
    if (label && state.updateMessage) label.textContent = state.updateMessage
  })

  // 百宝箱:FFmpeg 工具进度
  window.runtime.EventsOn('toolbox:progress', (data) => {
    if (!data) return
    if (typeof data.percent === 'number') state.toolboxProgress = data.percent
    if (data.message) state.toolboxMessage = data.message
    if (state.page !== 'toolbox') return
    patchToolboxProgress()
  })

  window.runtime.EventsOn('toolbox:finished', (data) => {
    if (!data) return
    state.toolboxBusy = false
    state.toolboxProgress = data.ok ? 100 : state.toolboxProgress
    state.toolboxMessage = data.ok ? '完成' : ''
    state.toolboxResult = data.ok
      ? `已完成,输出目录:${data.output || ''}`
      : `${data.message || '处理失败'}`
    if (state.page !== 'toolbox') return
    render()
  })
}

// 启动后静默检查一次更新:失败不打扰,有新版本时在侧边栏「关于」加提示点。
setTimeout(async () => {
  const api = backend()
  if (!api?.CheckUpdate) return
  try {
    const result = await api.CheckUpdate(APP_VERSION)
    if (!result?.ok) return
    if (result.hasUpdate) {
      state.updateChecked = true
      state.updateHas = true
      state.updateLatest = result.latest || ''
      state.updateNotes = result.notes || ''
      state.updateUrl = result.url || ''
      state.updatePageUrl = result.pageUrl || ''
      state.updateSource = result.source || ''
      markUpdateBadge()
      if (state.page === 'about') renderUpdateArea()
    }
  } catch { /* 静默,忽略 */ }
}, 2500)

// markUpdateBadge 在侧边栏「关于」上加/去提示点。
function markUpdateBadge() {
  const nav = $('.nav-item[data-page="about"]')
  if (!nav) return
  const existing = nav.querySelector('.nav-badge')
  if (state.updateHas && !existing) {
    nav.insertAdjacentHTML('beforeend', '<i class="nav-badge"></i>')
  } else if (!state.updateHas && existing) {
    existing.remove()
  }
}
