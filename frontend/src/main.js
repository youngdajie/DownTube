import './style.css'

const icon = (name, size = 18) => {
  const paths = {
    download: '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="m7 10 5 5 5-5"/><path d="M12 15V3"/>',
    settings: '<circle cx="12" cy="12" r="3"/><path d="m19.4 15 .1.1a2 2 0 1 1-2.8 2.8l-.1-.1a2 2 0 0 0-3.4 1.4v.3a2 2 0 1 1-4 0v-.2A2 2 0 0 0 5.8 17l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a2 2 0 0 0-1.4-3.4h-.3a2 2 0 1 1 0-4h.2A2 2 0 0 0 3 4.2l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a2 2 0 0 0 3.4-1.4v-.3a2 2 0 1 1 4 0v.2A2 2 0 0 0 16.6 1l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a2 2 0 0 0 1.4 3.4h.3a2 2 0 1 1 0 4h-.2a2 2 0 0 0-1.5 3.8Z" transform="translate(2 2) scale(.83)"/>',
    log: '<path d="M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01"/>',
    plus: '<path d="M12 5v14M5 12h14"/>',
    folder: '<path d="M3 7a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>',
    link: '<path d="M10 13a5 5 0 0 0 7.1 0l3-3A5 5 0 0 0 13 2.9l-1.7 1.7"/><path d="M14 11a5 5 0 0 0-7.1 0l-3 3A5 5 0 0 0 11 21.1l1.7-1.7"/>',
    play: '<path d="m7 4 14 8-14 8z"/>',
    pause: '<path d="M7 4h3v16H7zM14 4h3v16h-3z"/>',
    check: '<path d="m5 12 4 4L19 6"/>',
    more: '<circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/>',
    x: '<path d="m18 6-12 12M6 6l12 12"/>',
    search: '<circle cx="11" cy="11" r="7"/><path d="m20 20-4-4"/>',
    external: '<path d="M15 3h6v6M10 14 21 3"/><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>',
    shield: '<path d="M12 22s8-4 8-11V5l-8-3-8 3v6c0 7 8 11 8 11z"/><path d="m9 12 2 2 4-4"/>',
    sparkle: '<path d="m12 3 1.9 5.8L20 11l-6.1 2.2L12 19l-1.9-5.8L4 11l6.1-2.2L12 3z"/><path d="m19 14 1 2.5 2.5 1-2.5 1L19 21l-1-2.5-2.5-1 2.5-1L19 14z"/>',
    clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
    trash: '<path d="M3 6h18M8 6V4h8v2m3 0-1 14H6L5 6m4 4v6m6-6v6"/>',
    video: '<rect x="3" y="5" width="18" height="14" rx="2"/><path d="m10 9 5 3-5 3z"/>',
    arrow: '<path d="M5 12h14m-6-6 6 6-6 6"/>',
    chevron: '<path d="m9 18 6-6-6-6"/>',
    globe: '<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3a15 15 0 0 1 0 18M12 3a15 15 0 0 0 0 18"/>',
    monitor: '<rect x="3" y="4" width="18" height="13" rx="2"/><path d="M8 21h8m-4-4v4"/>',
    checkcircle: '<circle cx="12" cy="12" r="9"/><path d="m8 12 3 3 5-6"/>',
    minimize: '<path d="M5 12h14"/>',
    maximize: '<rect x="5" y="5" width="14" height="14" rx="1"/>',
    restore: '<path d="M8 8V5h11v11h-3M5 8h11v11H5z"/>',
    info: '<circle cx="12" cy="12" r="9"/><path d="M12 11v5m0-8h.01"/>',
    code: '<path d="m8 8-4 4 4 4m8-8 4 4-4 4m-3-11-2 14"/>',
    layers: '<path d="m12 3 9 5-9 5-9-5 9-5Z"/><path d="m3 12 9 5 9-5M3 16l9 5 9-5"/>',
    cpu: '<rect x="5" y="5" width="14" height="14" rx="2"/><path d="M9 9h6v6H9zM9 1v4m6-4v4M9 19v4m6-4v4M1 9h4m-4 6h4m14-6h4m-4 6h4"/>',
    arrowDown: '<path d="M12 3v12m-5-5 5 5 5-5M5 19h14"/>',
    scale: '<path d="m12 3 9 18H3l9-18Z"/><path d="M12 9v4m0 3h.01"/>',
  }
  return `<svg width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${paths[name] || paths.sparkle}</svg>`
}

const state = {
  page: 'downloads',
  tasks: [],
  filter: '',
  quality: '1080p',
  type: '视频 + 音频',
  folder: '',
  useProxy: false,
  proxyURL: 'http://127.0.0.1:10809',
  autoDetectProxy: true,
  useBrowserCookie: false,
  cookieBrowser: 'Chrome',
  highFrameRate: true,
  openOnComplete: false,
  engineReady: false,
  engineError: '',
  ffmpegReady: false,
  proxyChecked: false,
  proxyReady: false,
  proxyMessage: '',
  windowMaximized: false,
  modal: false,
  notice: '',
}

const backend = () => window.go?.main?.App
window.addEventListener('error', event => backend()?.ReportFrontendError?.(`${event.message} (${event.filename}:${event.lineno})`))
window.addEventListener('unhandledrejection', event => backend()?.ReportFrontendError?.(`unhandled rejection: ${event.reason}`))
function applySnapshot(snapshot) {
  if (!snapshot) return
  state.tasks = snapshot.tasks || []
  const s = snapshot.settings || {}
  state.quality = s.quality || state.quality
  state.type = s.downloadType || state.type
  state.folder = s.downloadDir || state.folder
  state.useProxy = !!s.useProxy
  state.proxyURL = s.proxyURL || state.proxyURL
  state.autoDetectProxy = s.autoDetectProxy !== false
  state.useBrowserCookie = !!s.useBrowserCookie
  state.cookieBrowser = s.cookieBrowser || 'Chrome'
  state.highFrameRate = s.highFrameRate !== false
  state.openOnComplete = !!s.openOnComplete
}
async function refreshSnapshot() {
  const api = backend()
  if (!api?.Snapshot) return
  try { applySnapshot(await api.Snapshot()); render() } catch (error) { console.error(error); toast(`读取本地数据失败：${error.message}`) }
}

const root = document.querySelector('#app')
function closeSelect(select, restoreFocus = false) {
  if (!select?.classList.contains('is-open')) return
  select.classList.remove('is-open')
  select.querySelector('.select-trigger')?.setAttribute('aria-expanded', 'false')
  if (restoreFocus) select.querySelector('.select-trigger')?.focus()
}
function closeAllSelects(except = null) {
  root.querySelectorAll('.md-select.is-open').forEach(select => { if (select !== except) closeSelect(select) })
}
root.addEventListener('click', event => {
  const trigger = event.target.closest('.select-trigger')
  if (trigger) {
    const select = trigger.closest('.md-select')
    const shouldOpen = !select.classList.contains('is-open')
    closeAllSelects(select)
    select.classList.toggle('is-open', shouldOpen)
    trigger.setAttribute('aria-expanded', String(shouldOpen))
    return
  }
  const option = event.target.closest('.select-option')
  if (option) {
    const select = option.closest('.md-select')
    const setting = select.dataset.setting
    const value = option.dataset.value
    if (setting === 'quality') state.quality = value
    if (setting === 'downloadType') state.type = value
    if (setting === 'cookieBrowser') state.cookieBrowser = value
    select.querySelector('.select-current').textContent = value
    select.querySelectorAll('.select-option').forEach(item => {
      const selected = item === option
      item.classList.toggle('selected', selected)
      item.setAttribute('aria-selected', String(selected))
    })
    closeSelect(select)
    return
  }
  if (!event.target.closest('.md-select')) closeAllSelects()
})
root.addEventListener('keydown', event => {
  const select = event.target.closest('.md-select')
  if (!select) return
  const options = [...select.querySelectorAll('.select-option')]
  const trigger = select.querySelector('.select-trigger')
  if (event.target === trigger && ['ArrowDown', 'ArrowUp'].includes(event.key)) {
    event.preventDefault()
    closeAllSelects(select)
    select.classList.add('is-open')
    trigger.setAttribute('aria-expanded', 'true')
    const selectedIndex = Math.max(0, options.findIndex(option => option.getAttribute('aria-selected') === 'true'))
    options[event.key === 'ArrowDown' ? selectedIndex : (selectedIndex || options.length - 1)].focus()
    return
  }
  if (!event.target.matches('.select-option')) return
  const index = options.indexOf(event.target)
  if (event.key === 'Escape') { event.preventDefault(); closeSelect(select, true) }
  else if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    options[(index + (event.key === 'ArrowDown' ? 1 : -1) + options.length) % options.length].focus()
  } else if (event.key === 'Home' || event.key === 'End') {
    event.preventDefault(); options[event.key === 'Home' ? 0 : options.length - 1].focus()
  }
})
document.addEventListener('pointerdown', event => { if (!event.target.closest('.md-select')) closeAllSelects() })
const taskCount = () => state.tasks.length
const fmt = (n) => String(n).padStart(2, '0')
const now = () => { const d = new Date(); return `${fmt(d.getHours())}:${fmt(d.getMinutes())}:${fmt(d.getSeconds())}` }
const safe = (s) => String(s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]))

function render() {
  root.innerHTML = `
    <div class="app-window">
    <div class="window-titlebar" id="windowTitlebar"><div class="window-title-left"><div class="window-brand"><img class="window-brand-icon" src="/downtube-icon.svg" alt=""><span>DownTube</span></div><div class="window-engine-status ${state.engineReady ? 'is-ready' : 'is-missing'}" title="${state.engineReady ? 'yt-dlp 已就绪' : 'yt-dlp 未配置'}"><span class="status-led"></span><span>${state.engineReady ? '下载引擎就绪' : '下载引擎未配置'}</span></div></div><div class="window-controls"><button class="window-control" id="minimizeWindow" title="最小化" aria-label="最小化">${icon('minimize', 15)}</button><button class="window-control" id="maximizeWindow" title="最大化" aria-label="最大化">${icon(state.windowMaximized ? 'restore' : 'maximize', 13)}</button><button class="window-control close" id="closeWindow" title="关闭" aria-label="关闭">${icon('x', 15)}</button></div></div>
    <div class="shell">
      <aside class="sidebar">
        <div class="brand"><img class="brand-icon" src="/downtube-icon.svg" alt=""><div><div class="brand-name">down<span>tube</span></div><div class="brand-caption">DESKTOP DOWNLOADER</div></div></div>
        <div class="workspace-label">工作区</div>
        <nav class="nav">
          <button class="nav-item ${state.page === 'downloads' ? 'active' : ''}" data-page="downloads">${icon('download')}<span>下载队列</span><em>${taskCount()}</em></button>
          <button class="nav-item ${state.page === 'settings' ? 'active' : ''}" data-page="settings">${icon('settings')}<span>下载设置</span></button>
          <button class="nav-item ${state.page === 'about' ? 'active' : ''}" data-page="about">${icon('info')}<span>关于</span></button>
        </nav>
        <div class="sidebar-bottom"></div>
      </aside>
      <main class="main">
        <section class="content">${state.page === 'downloads' ? renderDownloads() : state.page === 'settings' ? renderSettings() : renderAbout()}</section>
        <footer class="statusbar"><div><span class="status-led"></span><span>本地数据已保存</span><i></i><span>${state.engineReady ? `yt-dlp 已就绪${state.ffmpegReady ? ' · FFmpeg 已就绪' : ' · FFmpeg 请自行安装'}` : '等待配置 yt-dlp'}</span></div><span class="version">DOWNTUBE <b>0.1.0</b> <span class="demo-tag">DEV</span></span></footer>
      </main>
    </div>
    ${state.modal ? renderModal() : ''}
    ${state.notice ? `<div class="toast">${icon('checkcircle', 18)}<span>${safe(state.notice)}</span></div>` : ''}
    </div>
  `
  bind()
  syncWindowButtons()
}

function renderDownloads() {
  const filtered = state.tasks.filter(t => `${t.title} ${t.url} ${t.status}`.toLowerCase().includes(state.filter.toLowerCase()))
  const completed = state.tasks.filter(t => t.status === '已完成').length
  const active = state.tasks.filter(t => t.status === '准备中').length
  const canStart = state.tasks.some(t => ['等待中', '已暂停', '失败'].includes(t.status))
  return `
    <div class="page-heading"><div><div class="eyebrow">DOWNLOADS <span class="eyebrow-line"></span></div><h1>下载队列</h1><p>添加链接，任务会保存在这台电脑上。</p></div><div class="heading-actions">${active ? `<button class="tonal-btn" id="pauseAll">${icon('pause', 16)}全部暂停</button>` : `<button class="tonal-btn" id="startAll" ${canStart ? '' : 'disabled'}>${icon('play', 16)}全部开始</button>`}<button class="primary-btn" id="addTask">${icon('plus', 17)}添加链接</button></div></div>
    ${!state.engineReady ? `<div class="engine-warning"><div class="warning-icon">${icon('download', 18)}</div><div><b>下载引擎未配置</b><span>${safe(state.engineError || '请安装 yt-dlp，并将 yt-dlp.exe 放到程序目录或加入 PATH。')}</span></div><button class="text-btn" id="engineSetupBanner">获取 yt-dlp ${icon('arrow', 14)}</button></div>` : !state.ffmpegReady ? `<div class="engine-warning optional"><div class="warning-icon">${icon('video', 18)}</div><div><b>FFmpeg 未安装</b><span>基础下载仍可使用；合并高画质音视频或转换 MP3 需要自行安装 FFmpeg 并加入系统 PATH。</span></div><button class="text-btn" id="ffmpegSetup">安装说明 ${icon('arrow', 14)}</button></div>` : ''}
    <div class="stat-row"><div class="stat-card"><div class="stat-top"><span>总任务</span><div class="stat-icon purple">${icon('download', 17)}</div></div><div class="stat-value">${fmt(taskCount())}<small>个任务</small></div><div class="stat-foot">当前队列中的全部项目</div></div><div class="stat-card"><div class="stat-top"><span>正在下载</span><div class="stat-icon blue">${icon('arrow', 17)}</div></div><div class="stat-value">${fmt(active)}<small>个任务</small></div><div class="stat-foot"><span class="mini-led"></span> 下载引擎运行中</div></div><div class="stat-card"><div class="stat-top"><span>已完成</span><div class="stat-icon green">${icon('check', 17)}</div></div><div class="stat-value">${fmt(completed)}<small>个任务</small></div><div class="stat-foot">已保存到本地设备</div></div></div>
    <section class="panel queue-panel"><div class="panel-head"><div><h2>任务列表</h2><span class="panel-sub">你的最近下载</span></div><div class="queue-actions"><label class="searchbox">${icon('search', 16)}<input id="searchInput" placeholder="搜索任务..." value="${safe(state.filter)}" /></label><button class="subtle-btn" id="clearBtn" title="清空列表">${icon('trash', 15)}清空</button></div></div>
      ${filtered.length ? `<div class="table-wrap"><table><thead><tr><th class="col-video">视频</th><th>状态</th><th class="col-progress">进度</th><th>画质</th><th class="col-actions"></th></tr></thead><tbody>${filtered.map(taskRow).join('')}</tbody></table></div>` : `<div class="empty-state"><div class="empty-icon">${icon('video', 25)}</div><h3>${state.filter ? '没有匹配的任务' : '队列还是空的'}</h3><p>${state.filter ? '试试其他关键词。' : '粘贴 YouTube 链接来创建下载任务。'}</p>${!state.filter ? `<button class="subtle-btn" id="emptyAdd">${icon('plus', 15)}添加第一个链接</button>` : ''}</div>`}
      <div class="queue-foot"><span>显示 <b>${filtered.length}</b> 个任务</span><button class="text-btn" id="folderBtn">${icon('folder', 14)}打开保存位置 ${icon('external', 13)}</button></div>
    </section>
    <div class="tip-banner"><div class="tip-icon">${icon('link', 16)}</div><div><b>支持批量添加</b><span>每行一个 YouTube 链接。可下载视频、MP3 音频，并设置代理或浏览器 Cookies。</span></div><button class="tip-dismiss" id="dismissTip" aria-label="关闭">${icon('x', 15)}</button></div>
  `
}

function taskRow(t) {
  const finished = t.status === '已完成'
  const pending = t.status === '等待中'
  const short = t.url.replace(/^https?:\/\//, '').replace(/\/$/, '')
  const failed = t.status === '失败'
  return `<tr><td><div class="video-cell"><div class="thumb ${t.id % 3 === 0 ? 'thumb-green' : t.id % 2 ? 'thumb-violet' : 'thumb-blue'}"><span>${icon('video', 20)}</span><i>YT</i></div><div class="video-meta"><b title="${safe(t.title)}">${safe(t.title)}</b><span>${safe(short)}${t.error ? ` · ${safe(t.error)}` : ''}</span></div></div></td><td><span class="badge ${finished ? 'badge-done' : pending ? 'badge-pending' : failed ? 'badge-failed' : 'badge-active'}"><i></i>${safe(t.status)}</span></td><td><div class="progress-cell"><div class="progress-track"><span class="${finished ? 'progress-done' : ''}" style="width:${t.progress}%"></span></div><small>${t.progress}%</small></div></td><td><span class="quality-pill">${safe(t.quality)}</span></td><td><div class="row-actions"><button class="row-btn" data-action="${t.status === '准备中' ? 'pause' : 'start'}" data-id="${t.id}" title="${t.status === '准备中' ? '暂停' : '开始'}">${icon(t.status === '准备中' ? 'pause' : 'play', 15)}</button><button class="row-btn danger-hover" data-action="remove" data-id="${t.id}" title="移除">${icon('x', 15)}</button></div></td></tr>`
}

function selectControl(setting, options, value, ariaLabel) {
  const id = `select-${setting}`
  return `<div class="md-select" data-setting="${setting}"><button class="select-trigger" type="button" id="${id}" aria-label="${ariaLabel}" aria-haspopup="listbox" aria-expanded="false" aria-controls="${id}-menu"><span class="select-current">${safe(value)}</span><span class="select-chevron">${icon('chevron', 17)}</span></button><div class="select-menu" id="${id}-menu" role="listbox" aria-labelledby="${id}">${options.map(option => `<button class="select-option ${option === value ? 'selected' : ''}" type="button" role="option" tabindex="-1" data-value="${safe(option)}" aria-selected="${option === value}"><span>${safe(option)}</span><span class="select-check">${icon('check', 15)}</span></button>`).join('')}</div></div>`
}

function renderSettings() {
  return `<div class="page-heading"><div><div class="eyebrow">PREFERENCES <span class="eyebrow-line"></span></div><h1>下载设置</h1><p>按照你的习惯配置下载体验</p></div><div class="heading-actions"><button class="subtle-btn" id="resetSettings">恢复默认</button><button class="primary-btn" id="saveSettings">保存设置</button></div></div>
    <div class="settings-layout"><div class="settings-main">
      <section class="panel settings-panel"><div class="settings-title"><div class="setting-icon purple">${icon('video', 18)}</div><div><h2>视频偏好</h2><p>选择下载视频时使用的默认画质与格式</p></div></div>
        <div class="form-grid"><div class="field"><span>默认视频画质</span>${selectControl('quality', ['2160p · 4K', '1440p', '1080p', '720p', '480p', '最佳可用'], state.quality, '默认视频画质')}</div><div class="field"><span>下载内容</span>${selectControl('downloadType', ['视频 + 音频', '仅视频', '仅音频 · MP3'], state.type, '下载内容')}</div></div>
        <div class="setting-divider"></div><div class="toggle-row"><div class="toggle-copy"><b>优先选择高帧率</b><small>可用时优先下载 60 fps 视频</small></div><label class="switch"><input id="highFrameRate" type="checkbox" ${state.highFrameRate ? 'checked' : ''}><span></span></label></div><div class="toggle-row"><div class="toggle-copy"><b>下载完成后打开文件夹</b><small>自动定位到已保存的视频</small></div><label class="switch"><input id="openOnComplete" type="checkbox" ${state.openOnComplete ? 'checked' : ''}><span></span></label></div>
      </section>
      <section class="panel settings-panel"><div class="settings-title"><div class="setting-icon blue">${icon('folder', 18)}</div><div><h2>保存位置</h2><p>选择视频文件在这台设备上的保存位置</p></div></div><label class="field"><span>下载文件夹</span><div class="folder-field"><div class="folder-path">${icon('folder', 16)}<input id="folderPath" value="${safe(state.folder)}" /></div><button class="subtle-btn" id="browseFolder">浏览...</button></div><small class="helper-text">视频将保存到所选文件夹，文件名会尽量保留原视频标题。</small></label></section>
      <section class="panel settings-panel"><div class="settings-title"><div class="setting-icon amber">${icon('globe', 18)}</div><div><h2>网络与隐私</h2><p>配置代理连接与浏览器验证方式</p></div></div>
        <div class="toggle-row"><div class="toggle-copy"><b>使用代理</b><small>通过指定代理连接 YouTube</small></div><label class="switch"><input id="proxyToggle" type="checkbox" ${state.useProxy ? 'checked' : ''}><span></span></label></div><label class="field proxy-address ${state.useProxy ? '' : 'hidden'}" id="proxyAddress"><span>代理地址</span><input id="proxyURL" class="text-input" value="${safe(state.proxyURL)}" placeholder="http://127.0.0.1:10809" /></label>
        <div class="toggle-row"><div class="toggle-copy"><b>启动时检测代理</b><small>启动时确认代理端口是否可连接</small></div><label class="switch"><input id="autoDetectToggle" type="checkbox" ${state.autoDetectProxy ? 'checked' : ''}><span></span></label></div><div class="setting-divider"></div><div class="toggle-row"><div class="toggle-copy"><b>使用浏览器 Cookies</b><small>用于访问需要登录或验证的视频</small></div><label class="switch"><input id="cookiesToggle" type="checkbox" ${state.useBrowserCookie ? 'checked' : ''}><span></span></label></div>
        <div class="cookie-browser ${state.useBrowserCookie ? '' : 'hidden'}" id="browserSelect"><span>读取 Cookies 的浏览器</span>${selectControl('cookieBrowser', ['Chrome', 'Microsoft Edge', 'Firefox', 'Brave'], state.cookieBrowser, '读取 Cookies 的浏览器')}</div><div class="proxy-actions"><button class="subtle-btn" id="testProxy">${icon('shield', 15)}测试代理</button><span class="proxy-result ${state.proxyChecked && !state.proxyReady ? 'proxy-failed' : ''}">${state.proxyChecked ? safe(state.proxyMessage) : state.useProxy ? '点击测试连接' : '代理已关闭'}</span></div>
      </section>
    </div><aside class="settings-side"><div class="side-note"><div class="note-mark">${icon('shield', 19)}</div><h3>你的下载，保持私密</h3><p>所有任务和设置都保存在本地设备上。我们不会上传或收集你的视频链接。</p><div class="note-divider"></div><div class="note-foot">${icon('checkcircle', 15)} 本地处理 · 无云端同步</div></div><div class="side-version"><span>应用版本</span><b>DownTube <i>0.1.0 Dev</i></b><button id="aboutBtn">关于 DownTube ${icon('chevron', 13)}</button></div></aside></div>`
}

function renderAbout() {
  return `<div class="page-heading"><div><div class="eyebrow">ABOUT <span class="eyebrow-line"></span></div><h1>关于 DownTube</h1><p>一个专注于本地任务管理的 Windows 桌面工具。</p></div></div>
    <section class="about-hero panel"><div class="about-identity"><img class="about-app-icon" src="/downtube-icon.svg" alt="DownTube 图标"><div class="about-heading"><div class="about-kicker">DESKTOP MEDIA TOOL</div><div class="about-name">DownTube</div><p>YouTube 下载任务管理工具</p></div></div><div class="about-hero-meta"><span class="about-version"><i></i> 开发版 0.1.0</span><div class="about-author"><span>DESIGNED BY</span><b>小杰</b></div></div><svg class="about-orbit" viewBox="0 0 220 180" aria-hidden="true"><circle cx="132" cy="88" r="70"/><circle cx="132" cy="88" r="48"/><path d="M132 18v140M62 88h140M82 38l100 100m0-100L82 138"/><circle cx="132" cy="88" r="8"/></svg></section>
    <section class="about-section"><div class="about-section-heading"><div><div class="about-overline">THE STACK</div><h2>技术组成</h2><p>各组件分工明确，协同完成桌面下载流程。</p></div><span class="about-section-index">01 <i>/</i> 02</span></div><div class="tech-grid">
      <article class="tech-card"><div class="tech-card-top"><span class="tech-icon go">${icon('code', 22)}</span><span class="tech-index">01</span></div><div class="tech-card-copy"><h3>Go</h3><p>管理下载任务、本地设置、进程控制与 Windows 系统集成。</p></div><div class="tech-card-foot"><span>APPLICATION CORE</span><span>GO</span></div></article>
      <article class="tech-card"><div class="tech-card-top"><span class="tech-icon wails">${icon('layers', 22)}</span><span class="tech-index">02</span></div><div class="tech-card-copy"><h3>Wails · WebView2</h3><p>连接 Go 后端与 Windows 桌面窗口，使用系统 WebView 呈现界面。</p></div><div class="tech-card-foot"><span>WINDOWS DESKTOP</span><span>WAILS</span></div></article>
      <article class="tech-card"><div class="tech-card-top"><span class="tech-icon ui">${icon('cpu', 22)}</span><span class="tech-index">03</span></div><div class="tech-card-copy"><h3>HTML · CSS · JavaScript</h3><p>构建 Material 3 风格交互界面，并以 <code>#1296db</code> 作为品牌主色。</p></div><div class="tech-card-foot"><span>INTERFACE</span><span>WEBVIEW</span></div></article>
      <article class="tech-card"><div class="tech-card-top"><span class="tech-icon engine">${icon('arrowDown', 22)}</span><span class="tech-index">04</span></div><div class="tech-card-copy"><h3>yt-dlp · FFmpeg</h3><p>yt-dlp 负责解析与下载；FFmpeg 负责合并音视频和转换格式。</p></div><div class="tech-card-foot"><span>MEDIA ENGINE</span><span>LOCAL</span></div></article>
    </div></section>
    <section class="disclaimer-card"><div class="disclaimer-layout"><div class="disclaimer-heading"><div class="disclaimer-icon">${icon('scale', 20)}</div><div><div class="about-overline">PLEASE READ</div><h2>使用说明与免责声明</h2><p>请在使用前阅读并遵守相关规则。</p></div></div><div class="disclaimer-copy"><p>本程序仅提供本地下载任务管理和媒体处理功能。请仅下载你有权保存或使用的内容，并遵守 YouTube 及相关网站的服务条款、版权规则和所在地法律。</p><p>不得将本程序用于侵犯他人权益、未经授权传播内容或规避访问限制。用户应自行确认下载行为的合法性，并对使用本程序产生的行为和后果负责。</p><p>本程序按现状提供。开发者不保证第三方网站、下载引擎或媒体资源持续可用，也不对第三方内容及下载结果承担责任。</p></div></div><div class="disclaimer-rule"><span>LOCAL FIRST</span><i></i><span>USER RESPONSIBILITY</span><i></i><span>AS IS</span></div></section>
    <div class="about-footnote"><span>${icon('shield', 14)}</span><p>任务列表与设置仅保存在本机。DownTube 不提供视频内容，也不托管下载文件。</p><span class="footnote-version">DOWNTUBE · 0.1.0</span></div>`
}

function renderModal() {
  return `<div class="modal-backdrop" id="modalBackdrop"><div class="modal"><div class="modal-head"><div><h2>添加下载链接</h2><p>粘贴一个或多个 YouTube 视频链接</p></div><button class="icon-btn" id="closeModal">${icon('x', 17)}</button></div><label class="field modal-field"><span>视频链接</span><textarea id="linkInput" rows="4" placeholder="https://www.youtube.com/watch?v=...&#10;每行一个链接，支持批量添加"></textarea><small>支持 youtube.com 和 youtu.be 链接</small></label><div class="modal-foot"><span>下载画质 <b>${safe(state.quality)}</b></span><div><button class="subtle-btn" id="cancelModal">取消</button><button class="primary-btn" id="confirmAdd">${icon('plus', 15)}加入队列</button></div></div></div></div>`
}

function toast(message) {
  state.notice = message; render();
  setTimeout(() => { state.notice = ''; render() }, 2300)
}

function bind() {
  document.querySelector('#minimizeWindow')?.addEventListener('click', () => window.runtime?.WindowMinimise?.())
  document.querySelector('#maximizeWindow')?.addEventListener('click', () => { window.runtime?.WindowToggleMaximise?.(); setTimeout(syncWindowButtons, 80) })
  document.querySelector('#closeWindow')?.addEventListener('click', () => window.runtime?.Quit?.())
  document.querySelector('#windowTitlebar')?.addEventListener('dblclick', event => {
    if (event.target.closest('.window-controls')) return
    window.runtime?.WindowToggleMaximise?.(); setTimeout(syncWindowButtons, 80)
  })
  document.querySelectorAll('[data-page]').forEach(el => el.addEventListener('click', () => { state.page = el.dataset.page; render() }))
  document.querySelector('#addTask')?.addEventListener('click', () => { state.modal = true; render(); setTimeout(() => document.querySelector('#linkInput')?.focus(), 30) })
  document.querySelector('#emptyAdd')?.addEventListener('click', () => { state.modal = true; render() })
  document.querySelector('#closeModal')?.addEventListener('click', closeModal)
  document.querySelector('#cancelModal')?.addEventListener('click', closeModal)
  document.querySelector('#modalBackdrop')?.addEventListener('click', e => { if (e.target.id === 'modalBackdrop') closeModal() })
  document.querySelector('#confirmAdd')?.addEventListener('click', addLinks)
  document.querySelector('#linkInput')?.addEventListener('keydown', e => { if (e.ctrlKey && e.key === 'Enter') addLinks() })
  document.querySelector('#searchInput')?.addEventListener('input', e => { const pos = e.target.selectionStart; state.filter = e.target.value; render(); const input = document.querySelector('#searchInput'); input.focus(); input.setSelectionRange(pos, pos) })
  document.querySelector('#clearBtn')?.addEventListener('click', async () => { if (backend()?.ClearTasks) applySnapshot(await backend().ClearTasks()); else state.tasks = []; render(); toast('下载队列已清空') })
  document.querySelector('#dismissTip')?.addEventListener('click', e => e.currentTarget.closest('.tip-banner').remove())
  document.querySelector('#folderBtn')?.addEventListener('click', openFolder)
  document.querySelector('#engineSetupBanner')?.addEventListener('click', () => backend()?.OpenEngineDownloads?.())
  document.querySelector('#ffmpegSetup')?.addEventListener('click', () => backend()?.OpenFFmpegDownloads?.())
  document.querySelector('#startAll')?.addEventListener('click', startAll)
  document.querySelector('#pauseAll')?.addEventListener('click', pauseAll)
  document.querySelector('#browseFolder')?.addEventListener('click', browse)
  document.querySelector('#folderPath')?.addEventListener('change', e => state.folder = e.target.value)
  document.querySelector('#highFrameRate')?.addEventListener('change', e => state.highFrameRate = e.target.checked)
  document.querySelector('#openOnComplete')?.addEventListener('change', e => state.openOnComplete = e.target.checked)
  document.querySelector('#cookiesToggle')?.addEventListener('change', e => { state.useBrowserCookie = e.target.checked; document.querySelector('#browserSelect').classList.toggle('hidden', !e.target.checked) })
  document.querySelector('#proxyToggle')?.addEventListener('change', e => { state.useProxy = e.target.checked; document.querySelector('#proxyAddress').classList.toggle('hidden', !e.target.checked) })
  document.querySelector('#autoDetectToggle')?.addEventListener('change', e => state.autoDetectProxy = e.target.checked)
  document.querySelector('#proxyURL')?.addEventListener('change', e => state.proxyURL = e.target.value)
  document.querySelector('#testProxy')?.addEventListener('click', testProxy)
  document.querySelector('#saveSettings')?.addEventListener('click', saveSettings)
  document.querySelector('#resetSettings')?.addEventListener('click', () => { state.quality = '1080p'; state.type = '视频 + 音频'; state.useProxy = false; state.autoDetectProxy = true; state.useBrowserCookie = false; render(); toast('已恢复默认设置，请保存') })
  document.querySelector('#aboutBtn')?.addEventListener('click', () => { state.page = 'about'; render() })
  document.querySelectorAll('[data-action]').forEach(btn => btn.addEventListener('click', () => handleTask(btn.dataset.action, Number(btn.dataset.id))))
}

async function syncWindowButtons() {
  if (!window.runtime?.WindowIsMaximised) return
  try {
    state.windowMaximized = await window.runtime.WindowIsMaximised()
    const button = document.querySelector('#maximizeWindow')
    if (button) {
      button.innerHTML = icon(state.windowMaximized ? 'restore' : 'maximize', 13)
      button.title = state.windowMaximized ? '还原' : '最大化'
      button.setAttribute('aria-label', button.title)
    }
  } catch { /* Wails window controls are unavailable in browser preview. */ }
}

function closeModal() { state.modal = false; render() }
async function startAll() {
  if (backend()?.StartAllTasks) applySnapshot(await backend().StartAllTasks())
  else toast('全部开始需要在桌面程序中运行')
  await refreshSystemStatus()
  render()
}
async function pauseAll() {
  if (backend()?.PauseAllTasks) applySnapshot(await backend().PauseAllTasks())
  else state.tasks.forEach(t => { if (t.status === '准备中') t.status = '已暂停' })
  render(); toast('下载任务已暂停')
}
async function addLinks() {
  const input = document.querySelector('#linkInput')
  const lines = input.value.split(/[\n\r\t ,;]+/).map(s => s.trim()).filter(Boolean)
  const urls = lines.filter(s => { try { const u = new URL(s); return ['youtube.com', 'www.youtube.com', 'm.youtube.com', 'youtu.be', 'www.youtu.be'].includes(u.hostname) } catch { return false } })
  if (!urls.length) { input.classList.add('invalid'); input.focus(); return }
  let snapshot
  if (backend()?.AddLinks) snapshot = await backend().AddLinks(urls.join('\n'))
  else {
    const fresh = urls.map((url, i) => ({ id: Date.now() + i, url, title: '待获取视频信息', status: '等待中', progress: 0, quality: state.quality }))
    state.tasks.unshift(...fresh)
  }
  if (snapshot) applySnapshot(snapshot)
  state.modal = false; state.page = 'downloads'; render(); toast(`已加入 ${urls.length} 个链接`)
}

async function handleTask(action, id) {
  const t = state.tasks.find(x => x.id === id)
  if (action === 'remove') {
    if (backend()?.RemoveTask) applySnapshot(await backend().RemoveTask(id))
    else state.tasks = state.tasks.filter(x => x.id !== id)
    toast('已从队列移除'); render(); return
  }
  if (!t) return
  if (action === 'start') {
    if (backend()?.StartTask) applySnapshot(await backend().StartTask(id))
    else { t.status = '失败'; t.error = '桌面版下载引擎尚未配置' }
    toast(t.status === '失败' ? '下载引擎尚未配置' : '任务已开始')
  }
  if (action === 'pause') {
    if (backend()?.PauseTask) applySnapshot(await backend().PauseTask(id))
    else t.status = '已暂停'
    toast('下载已暂停')
  }
  render()
}

async function browse() {
  if (window.go?.main?.App?.SelectFolder) { const dir = await window.go.main.App.SelectFolder(); if (dir) state.folder = dir }
  else toast('桌面版中可使用原生文件夹选择器')
  render()
}

async function openFolder() {
  if (backend()?.OpenDownloadFolder) {
    try { await backend().OpenDownloadFolder() } catch (error) { toast(`无法打开文件夹：${error.message}`) }
  } else toast('请在桌面版中打开下载文件夹')
}

async function saveSettings() {
  const settings = { quality: state.quality, downloadType: state.type, downloadDir: document.querySelector('#folderPath')?.value || state.folder, useProxy: state.useProxy, proxyURL: document.querySelector('#proxyURL')?.value || state.proxyURL, autoDetectProxy: state.autoDetectProxy, useBrowserCookie: state.useBrowserCookie, cookieBrowser: state.cookieBrowser, highFrameRate: state.highFrameRate, openOnComplete: state.openOnComplete }
  state.folder = settings.downloadDir
  if (backend()?.SaveSettings) applySnapshot(await backend().SaveSettings(settings))
  await refreshSystemStatus()
  render(); toast('设置已保存到本机')
}

async function refreshSystemStatus() {
  if (!backend()?.GetSystemStatus) return
  try {
    const status = await backend().GetSystemStatus()
    state.engineReady = !!status.engineReady
    state.engineError = status.engineError || ''
    state.ffmpegReady = !!status.ffmpegReady
    state.proxyChecked = !!status.proxyChecked
    state.proxyReady = !!status.proxyReady
    state.proxyMessage = status.proxyMessage || ''
    render()
  } catch (error) { console.error(error) }
}

async function testProxy() {
  if (!state.useProxy) { toast('请先启用代理并填写代理地址'); return }
  if (backend()?.TestProxy) {
    try {
      const result = await backend().TestProxy(document.querySelector('#proxyURL')?.value || state.proxyURL)
      state.proxyChecked = true; state.proxyReady = !!result.ok; state.proxyMessage = result.message
      render(); toast(result.message)
    } catch (error) { toast(`代理测试失败：${error.message}`) }
  } else toast('代理测试仅在桌面版可用')
}

render()
refreshSnapshot()
refreshSystemStatus()
window.addEventListener('resize', syncWindowButtons)

if (window.runtime?.EventsOn) {
  window.runtime.EventsOn('state:updated', (snapshot) => { applySnapshot(snapshot); render() })
}
