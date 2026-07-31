import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// 构建产物输出到 public/build，Go 侧读取 manifest 后以 /build/ 前缀提供静态资源
export default defineConfig({
    plugins: [vue()],
    resolve: {
        alias: {
            // in-DOM 模板（页面内的组件标签）需要运行时编译器
            vue: 'vue/dist/vue.esm-bundler.js',
        },
    },
    base: '/build/',
    build: {
        manifest: true,
        outDir: 'public/build',
        emptyOutDir: true,
        rollupOptions: {
            input: 'src/js/app.js',
        },
    },
    server: {
        port: 5173,
        strictPort: true,
        origin: 'http://localhost:5173',
    },
})
