import { defineConfig } from 'vite'
import { readFileSync } from 'node:fs'

// 版本号只在 wails.json 里写一次，两处消费：
//   - wails.json 的 info.productVersion 会被 wails 写进 exe 的版本资源
//     （资源管理器属性页看到的那个版本）；
//   - 这里再注入到前端，界面上显示的是同一个值。
// 不会出现「属性页已经 0.3.0、界面还写 0.2.0」的错位。
// CNB 流水线同样从 wails.json 读版本号，用来定 tag 名和安装包文件名。
const productVersion = JSON.parse(
  readFileSync(new URL('../wails.json', import.meta.url), 'utf8')
).info.productVersion

export default defineConfig({
  base: '/',
  define: {
    __APP_VERSION__: JSON.stringify(productVersion),
  },
  build: { outDir: 'dist', emptyOutDir: true },
})
