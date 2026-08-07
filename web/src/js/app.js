// 样式
import '@fortawesome/fontawesome-free/css/all.min.css'
import '@fortawesome/fontawesome-free/css/v4-shims.min.css'
import 'highlight.js/styles/base16/tomorrow-night.css'
import 'easymde/dist/easymde.min.css'
import '@vueform/multiselect/themes/default.css'
import 'katex/dist/katex.min.css'
import '../sass/app.scss'

// jQuery 全局挂载须先于 bootstrap 执行
import './jquery-global'
import 'popper.js'
import 'bootstrap'

import hljs from 'highlight.js'

// 图片拖动上传（挂到 window.inlineAttachment，供编辑器组件使用）
import 'inline-attachment/src/inline-attachment'
import 'inline-attachment/src/codemirror-4.inline-attachment'

import { createApp } from 'vue'
import SimpleMdeComponent from './components/SimpleMdeComponent.vue'
import MultiSelectComponent from './components/MultiSelectComponent.vue'
import SingleSelectComponent from './components/SingleSelectComponent.vue'
import Toc from './components/Toc.vue'

window.hljs = hljs

document.addEventListener('DOMContentLoaded', () => {
    // mermaid 代码块转为 .mermaid 容器（须先于 hljs，避免高亮破坏源码）
    document.querySelectorAll('pre code.language-mermaid').forEach((el) => {
        const div = document.createElement('div')
        div.className = 'mermaid'
        div.textContent = el.textContent
        el.closest('pre').replaceWith(div)
    })

    // 代码高亮
    document.querySelectorAll('pre code').forEach((el) => hljs.highlightElement(el))

    // mermaid 图表渲染（按需加载 chunk）
    if (document.querySelector('.mermaid')) {
        import('mermaid').then(({ default: mermaid }) => {
            mermaid.initialize({ startOnLoad: false })
            mermaid.run({ querySelector: '.mermaid' })
        })
    }

    // LaTeX 公式渲染（KaTeX auto-render，pre/code/textarea 内默认忽略）
    // 作用范围：文章正文、关于页及评论（均带 .markdown-body，不含 textarea）
    if (document.querySelector('.markdown-body:not(textarea)')) {
        import('katex/dist/contrib/auto-render.mjs').then(({ default: renderMathInElement }) => {
            document.querySelectorAll('.markdown-body').forEach((el) => {
                renderMathInElement(el, {
                    delimiters: [
                        { left: '$$', right: '$$', display: true },
                        { left: '$', right: '$', display: false },
                        { left: '\\(', right: '\\)', display: false },
                        { left: '\\[', right: '\\]', display: true },
                    ],
                    throwOnError: false,
                })
            })
        })
    }

    // 评论回复表单展开/收起（partials/comments 里的 .reply-btn）
    document.querySelectorAll('.reply-btn').forEach((btn) => {
        btn.addEventListener('click', (e) => {
            e.preventDefault()
            const target = document.querySelector(btn.dataset.target)
            if (target) {
                target.classList.toggle('show')
            }
        })
    })

    // 返回顶部按钮
    const backToTop = document.createElement('div')
    backToTop.id = 'back-to-top'
    document.body.appendChild(backToTop)
    backToTop.addEventListener('click', () => {
        window.scrollTo({ top: 0, behavior: 'smooth' })
    })
    window.addEventListener('scroll', () => {
        backToTop.classList.toggle('back-to-top-show', window.scrollY > 300)
    })

    // Vue 撒点：页面通过内联脚本设置 window.__vueOptions = { el, data }
    const opts = window.__vueOptions
    if (opts && opts.el && document.querySelector(opts.el)) {
        const data = opts.data || {}
        const app = createApp({
            data() {
                return data
            },
        })
        app.component('simple-mde-component', SimpleMdeComponent)
        app.component('multi-select-component', MultiSelectComponent)
        app.component('single-select-component', SingleSelectComponent)
        app.component('toc', Toc)
        // in-DOM 模板不使用插值语法（与 Go 模板冲突），仅用属性绑定
        app.mount(opts.el)
    }
})
