<template>
    <div class="card toc-container hide-on-mobile">
        <h6 class="mb-2">目录</h6>
        <div ref="toc" class="vue-toc"></div>
    </div>
</template>

<script>
// 文章目录：扫描正文里的 h2/h3 生成锚点列表
export default {
    name: 'Toc',
    props: {
        contentSelector: { type: String, required: true },
        h2Class: { type: String, default: 'toc-2' },
        h3Class: { type: String, default: 'toc-3' },
    },
    mounted() {
        this.$nextTick(() => {
            const toc = this.$refs.toc
            const matches = document.querySelectorAll(
                `${this.contentSelector} h2, ${this.contentSelector} h3`
            )

            if (matches.length === 0) {
                const span = document.createElement('span')
                span.innerText = '暂无数据 ~_~'
                toc.appendChild(span)
                toc.style.textAlign = 'center'
                return
            }

            matches.forEach((item) => {
                if (!item.id) {
                    item.id = 'toc-' + Math.random().toString(36).substring(7)
                }
                if (item.tagName === 'H2') {
                    const ul = document.createElement('ul')
                    const li = document.createElement('li')
                    const a = document.createElement('a')

                    a.textContent = item.textContent
                    a.title = item.textContent
                    a.href = `#${item.id}`

                    li.appendChild(a)
                    li.classList.add(this.h2Class)
                    ul.appendChild(li)
                    toc.appendChild(ul)
                }
                if (item.tagName === 'H3') {
                    const ul = document.createElement('ul')
                    const li = document.createElement('li')
                    const a = document.createElement('a')

                    const lastUl = toc.lastElementChild
                    if (!lastUl) return
                    const lastLi = lastUl.lastElementChild
                    if (!lastLi) return

                    a.textContent = item.textContent
                    a.title = item.textContent
                    a.href = `#${item.id}`
                    li.appendChild(a)
                    li.classList.add(this.h3Class)

                    ul.appendChild(li)
                    lastLi.appendChild(ul)
                }
            })
        })
    },
}
</script>
