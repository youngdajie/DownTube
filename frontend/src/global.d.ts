// Wails 与 Vite 注入的全局对象、宿主持有的样式资源,以及本项目自用的少量类型别名。
// 这些都不是从代码里 import 来的东西,必须在此声明,否则 TS 不认识。

import type { main } from '../wailsjs/go/models'
// 后端导出方法的类型直接复用 Wails 自动生成的 App.d.ts —— 那个文件里大量返回
// Record<string, any>,在这里重写一份只会与生成结果漂移;直接引用则后端改签名这里跟着变。
import type * as WailsBindings from '../wailsjs/go/main/App'

declare global {
  interface Window {
    /** Wails 注入的前后端桥。浏览器里直接开 vite 时不存在,调用点一律用可选链。 */
    go?: { main?: { App?: typeof WailsBindings } }
    /** Wails 运行时:事件订阅与剪贴板 */
    runtime?: {
      EventsOn?: (event: string, callback: (data: any) => void) => void
      EventsOff?: (event: string, ...args: any[]) => void
      ClipboardGetText?: () => Promise<string>
      ClipboardSetText?: (text: string) => Promise<boolean>
      [key: string]: any
    }
  }

  /** vite.config.js 用 define 注入的版本号,与 wails.json 同一个值 */
  const __APP_VERSION__: string

  /** 任务对象,字段与 Go 侧 main.Task 一致 */
  type Task = main.Task
  /** 设置对象,字段与 Go 侧 main.Settings 一致 */
  type Settings = main.Settings
  /** 后端快照 */
  type Snapshot = main.Snapshot

  // 下面这几个都来自 Wails 生成的 models.ts(Go 侧原本返回 map[string]any,
  // 已在 contracts.go 里改成结构体)。用 type 别名导出,免得 main.ts 里到处写 main. 前缀。
  /** 单个站点的 Cookies 状态(generated) */
  type CookiePlatform = main.CookiePlatformStatus
  /** CookieStatus 的返回(generated) */
  type CookieStatus = main.CookieStatusResult
  /** 系统状态(generated) */
  type SystemStatus = main.SystemStatus
  /** 显卡编码器状态(generated) */
  type EncoderStatus = main.EncoderStatus
  /** 检查更新结果(generated) */
  type UpdateInfo = main.UpdateResult

  /** 自定义下拉控件的选项:纯字符串,或 { value, label } */
  type SelectOption = string | { value: string; label: string; [key: string]: any }
}

// Vite 负责处理样式资源,这里只需让 TS 认识「副作用导入」这种写法
declare module '*.css'

export {}
