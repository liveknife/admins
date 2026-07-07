import { createCdnPlugin } from "./cdn";
import vue from "@vitejs/plugin-vue";
import { viteBuildInfo } from "./info";
import svgLoader from "vite-svg-loader";
import Icons from "unplugin-icons/vite";
import type { PluginOption } from "vite";
import vueJsx from "@vitejs/plugin-vue-jsx";
import tailwindcss from "@tailwindcss/vite";
import { configCompressPlugin } from "./compress";
import removeNoMatch from "vite-plugin-router-warn";
import { visualizer } from "rollup-plugin-visualizer";
import removeConsole from "vite-plugin-remove-console";
import { codeInspectorPlugin } from "code-inspector-plugin";

export function getPluginsList(
  VITE_CDN: boolean,
  VITE_COMPRESSION: ViteCompression,
  VITE_CODE_INSPECTOR: boolean
): PluginOption[] {
  const lifecycle = process.env.npm_lifecycle_event;
  return [
    tailwindcss(),
    vue(),
    // Vue JSX/TSX 支持
    vueJsx(),
    /**
     * 页面按住组合键时可以定位组件源码。
     * Mac 默认组合键：Option + Shift
     * Windows 默认组合键：Alt + Shift
     * 文档：https://inspector.fe-dev.cn/guide/start.html
     */
    VITE_CODE_INSPECTOR
      ? codeInspectorPlugin({
          bundler: "vite",
          hideConsole: true
        })
      : null,
    viteBuildInfo(),
    /**
     * 开发环境移除 vue-router 动态路由的无效匹配警告。
     * 参考：https://github.com/vuejs/router/issues/521
     * 参考：https://github.com/vuejs/router/issues/359
     */
    removeNoMatch(),
    // SVG 组件化支持
    svgLoader(),
    // 自动按需加载图标
    Icons({
      compiler: "vue3",
      scale: 1
    }),
    VITE_CDN ? createCdnPlugin() : null,
    configCompressPlugin(VITE_COMPRESSION),
    // 线上环境删除 console
    removeConsole({ external: ["src/assets/iconfont/iconfont.js"] }),
    // 打包分析
    lifecycle === "report"
      ? visualizer({ open: true, brotliSize: true, filename: "report.html" })
      : (null as any)
  ];
}
