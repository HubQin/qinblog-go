<template>
    <div class="simplemde-component">
        <textarea :name="name" ref="textarea"></textarea>
    </div>
</template>

<script>
import EasyMDE from 'easymde'

// Markdown 编辑器（原 SimpleMDE 已停维护，换用 API 兼容的 EasyMDE）
export default {
    name: 'SimpleMdeComponent',
    props: {
        name: { type: String, default: 'body' },
        initial: { type: String, default: '' },
        uploadUrl: { type: String, default: '/posts/upload_post_image' },
    },
    data() {
        return { easymde: null }
    },
    mounted() {
        this.easymde = new EasyMDE({
            autofocus: true,
            status: false,
            autosave: { enabled: false },
            element: this.$refs.textarea,
            forceSync: true, // 同步回 textarea，随表单提交
            indentWithTabs: false,
            initialValue: this.initial || '',
            lineWrapping: true,
            parsingConfig: {
                allowAtxHeaderWithoutSpace: true,
                strikethrough: true,
                underscoresBreakWords: true,
            },
            placeholder: 'Type here...',
            insertTexts: {
                horizontalRule: ['', '\n\n-----\n\n'],
                image: ['![](https://', ')'],
                link: ['[', '](https://)'],
                table: ['', '\n\n| Column 1 | Column 2 | Column 3 |\n| -------- | -------- | -------- |\n| Text     | Text      | Text     |\n\n'],
            },
            spellChecker: false,
            renderingConfig: {
                codeSyntaxHighlighting: true, // 预览代码高亮
            },
            autoDownloadFontAwesome: false, // 使用项目自带的 font awesome
            tabSize: 4,
            toolbar: [
                { name: 'bold', action: EasyMDE.toggleBold, className: 'fa fa-bold', title: '加粗' },
                { name: 'italic', action: EasyMDE.toggleItalic, className: 'fa fa-italic', title: '斜体' },
                { name: 'strikethrough', action: EasyMDE.toggleStrikethrough, className: 'fa fa-strikethrough', title: '删除线' },
                '|',
                { name: 'heading', action: EasyMDE.toggleHeadingSmaller, className: 'fa fa-header', title: '标题' },
                { name: 'heading-1', action: EasyMDE.toggleHeading1, className: 'fa fa-header fa-header-x fa-header-1', title: '一级标题' },
                { name: 'heading-2', action: EasyMDE.toggleHeading2, className: 'fa fa-header fa-header-x fa-header-2', title: '二级标题' },
                { name: 'heading-3', action: EasyMDE.toggleHeading3, className: 'fa fa-header fa-header-x fa-header-3', title: '三级标题' },
                '|',
                { name: 'code', action: EasyMDE.toggleCodeBlock, className: 'fa fa-code', title: '代码' },
                { name: 'quote', action: EasyMDE.toggleBlockquote, className: 'fa fa-quote-left', title: '引用' },
                { name: 'unordered-list', action: EasyMDE.toggleUnorderedList, className: 'fa fa-list-ul', title: '无序列表' },
                { name: 'ordered-list', action: EasyMDE.toggleOrderedList, className: 'fa fa-list-ol', title: '有序列表' },
                { name: 'horizontal-rule', action: EasyMDE.drawHorizontalRule, className: 'fa fa-minus', title: '插入水平线' },
                '|',
                { name: 'link', action: EasyMDE.drawLink, className: 'fa fa-link', title: '创建链接' },
                { name: 'image', action: EasyMDE.drawImage, className: 'fa fa-picture-o', title: '插入图片' },
                { name: 'table', action: EasyMDE.drawTable, className: 'fa fa-table', title: '插入表格' },
                '|',
                { name: 'preview', action: EasyMDE.togglePreview, className: 'fa fa-eye no-disable', title: '预览' },
                { name: 'side-by-side', action: EasyMDE.toggleSideBySide, className: 'fa fa-columns no-disable no-mobile', title: '编辑并预览' },
                { name: 'fullscreen', action: EasyMDE.toggleFullScreen, className: 'fa fa-arrows-alt no-disable no-mobile', title: '全屏' },
                {
                    name: 'guide',
                    action: () => window.open('https://github.com/pudongping/Markdown-Syntax-CN'),
                    className: 'fa fa-question-circle',
                    title: '帮助',
                },
            ],
        })

        // 拖拽/粘贴上传图片（inline-attachment 对接 codemirror）
        if (window.inlineAttachment) {
            const csrfMeta = document.head.querySelector('meta[name="csrf-token"]')
            window.inlineAttachment.editors.codemirror4.attach(this.easymde.codemirror, {
                uploadUrl: this.uploadUrl,
                extraHeaders: { 'X-CSRF-Token': csrfMeta ? csrfMeta.content : '' },
                progressText: '![图片上传中...]()',
                urlText: '![image]({filename})',
                errorText: '图片上传失败！',
            })
        }
    },
    beforeUnmount() {
        if (this.easymde) {
            this.easymde.toTextArea()
            this.easymde = null
        }
    },
}
</script>
