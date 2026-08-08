<template>
    <div class="card toc-container hide-on-mobile">
        <h6 class="mb-2">目录</h6>
        <div ref="toc" class="vue-toc"></div>
    </div>
</template>

<script>
// 文章目录：扫描正文里的 h1/h2/h3 生成层级锚点列表
// h1 为顶级；h2 嵌套在最近的 h1 下（无 h1 时自身作顶级，兼容只有 h2/h3 的文章）；h3 嵌套在最近的 h2（或 h1）下
export default {
    name: 'Toc',
    props: {
        contentSelector: { type: String, required: true },
        h1Class: { type: String, default: 'toc-1' },
        h2Class: { type: String, default: 'toc-2' },
        h3Class: { type: String, default: 'toc-3' },
    },
    mounted() {
        this.$nextTick(() => {
            const toc = this.$refs.toc
            const matches = document.querySelectorAll(
                `${this.contentSelector} h1, ${this.contentSelector} h2, ${this.contentSelector} h3`
            )

            if (matches.length === 0) {
                const span = document.createElement('span')
                span.innerText = '暂无数据 ~_~'
                toc.appendChild(span)
                toc.style.textAlign = 'center'
                return
            }

            let lastH1Li = null
            let lastH2Li = null

            const makeLi = (item, cls) => {
                const li = document.createElement('li')
                const a = document.createElement('a')
                a.textContent = item.textContent
                a.title = item.textContent
                a.href = `#${item.id}`
                li.appendChild(a)
                li.classList.add(cls)
                return li
            }
            // 将 li 追加到父级 li 的嵌套 ul（不存在则创建）
            const nest = (parentLi, li) => {
                let ul = parentLi.querySelector(':scope > ul')
                if (!ul) {
                    ul = document.createElement('ul')
                    parentLi.appendChild(ul)
                }
                ul.appendChild(li)
            }
            const topLevel = (li) => {
                const ul = document.createElement('ul')
                ul.appendChild(li)
                toc.appendChild(ul)
            }

            matches.forEach((item) => {
                if (!item.id) {
                    item.id = 'toc-' + Math.random().toString(36).substring(7)
                }
                if (item.tagName === 'H1') {
                    lastH1Li = makeLi(item, this.h1Class)
                    lastH2Li = null
                    topLevel(lastH1Li)
                } else if (item.tagName === 'H2') {
                    lastH2Li = makeLi(item, this.h2Class)
                    if (lastH1Li) {
                        nest(lastH1Li, lastH2Li)
                    } else {
                        topLevel(lastH2Li)
                    }
                } else if (item.tagName === 'H3') {
                    const li = makeLi(item, this.h3Class)
                    if (lastH2Li) {
                        nest(lastH2Li, li)
                    } else if (lastH1Li) {
                        nest(lastH1Li, li)
                    } else {
                        topLevel(li)
                    }
                }
            })
        })
    },
}
</script>
