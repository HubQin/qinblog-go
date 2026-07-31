<template>
    <div>
        <input type="hidden" :name="name" :value="value === null || value === undefined ? '' : value">
        <Multiselect
            v-model="value"
            :options="opts"
            value-prop="id"
            label="name"
            :object="false"
            :searchable="true"
            :can-clear="false"
            :can-deselect="false"
            :create-option="taggable"
            :on-create="handleCreate"
            :placeholder="placeholder"
        >
            <template v-if="hasIcon" #singlelabel="{ value }">
                <div class="multiselect-single-label multiselect-category">
                    <svg class="icon" aria-hidden="true"><use :xlink:href="'#' + value.icon"></use></svg>
                    <span class="single-label-slot">{{ value.name }}</span>
                </div>
            </template>
            <template v-if="hasIcon" #option="{ option }">
                <div class="multiselect-category">
                    <svg class="icon" aria-hidden="true"><use :xlink:href="'#' + option.icon"></use></svg>
                    <span class="option-slot">{{ option.name }}</span>
                </div>
            </template>
        </Multiselect>
    </div>
</template>

<script>
import Multiselect from '@vueform/multiselect'

// 单选下拉（分类/专题），taggable 时允许输入新名称创建（值为 "名称~随机串" 约定）
export default {
    name: 'SingleSelectComponent',
    components: { Multiselect },
    props: {
        name: { type: String, required: true },
        options: { type: Array, default: () => [] },
        initial: { type: [Number, String], default: 0 },
        taggable: { type: Boolean, default: false },
        placeholder: { type: String, default: '请选择' },
    },
    data() {
        return {
            opts: this.options.slice(),
            value: this.initial && this.initial !== 0 && this.initial !== '0' ? this.initial : null,
        }
    },
    computed: {
        hasIcon() {
            return this.opts.some((o) => o.icon)
        },
    },
    methods: {
        // 新建项：id 使用 "名称~随机串" 约定，由服务端识别创建
        handleCreate(option) {
            return {
                name: option.name,
                id: option.name + '~' + Math.random().toString(36).substring(2),
            }
        },
    },
}
</script>

<style>
.multiselect-category {
    display: flex;
    align-items: center;
    justify-content: flex-start;
}
.multiselect-category .icon {
    width: 1em;
    height: 1em;
}
.multiselect-category .option-slot,
.multiselect-category .single-label-slot {
    margin-left: 10px;
    font-size: 14px;
}
</style>
