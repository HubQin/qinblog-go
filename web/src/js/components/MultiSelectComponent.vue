<template>
    <div>
        <input type="hidden" :name="name" :value="valueJson">
        <Multiselect
            v-model="value"
            mode="tags"
            :options="opts"
            value-prop="id"
            label="name"
            :object="false"
            :searchable="true"
            :close-on-select="false"
            :create-option="taggable"
            :on-create="handleCreate"
            :placeholder="placeholder"
        />
    </div>
</template>

<script>
import Multiselect from '@vueform/multiselect'

// 多选标签，hidden input 提交 JSON 数组（新标签值为 "名称~随机串" 约定）
export default {
    name: 'MultiSelectComponent',
    components: { Multiselect },
    props: {
        name: { type: String, required: true },
        options: { type: Array, default: () => [] },
        initial: { type: Array, default: () => [] },
        taggable: { type: Boolean, default: true },
        placeholder: { type: String, default: '请添加文章标签（选填，可选择，可直接输入）' },
    },
    data() {
        return {
            opts: this.options.slice(),
            value: this.initial.slice(),
        }
    },
    computed: {
        valueJson() {
            return JSON.stringify(this.value)
        },
    },
    methods: {
        // 新建标签：id 使用 "名称~随机串" 约定，由服务端识别创建
        handleCreate(option) {
            return {
                name: option.name,
                id: option.name + '~' + Math.random().toString(36).substring(2),
            }
        },
    },
}
</script>
