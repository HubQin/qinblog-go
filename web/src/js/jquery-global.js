// 先把 jQuery 挂到全局，bootstrap 4 的插件依赖 window.jQuery
import $ from 'jquery'

window.$ = window.jQuery = $
