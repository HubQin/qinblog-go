# 开发交接文档（Session Handoff）

> 本文档由 Laravel → Go 重构会话生成，供在 Qoder 中单独打开 `qinblog-go` 目录继续开发时快速恢复上下文。
> 原 Laravel 项目位于上级目录 `../`（QinBlog，Laravel 5.8），可随时对照。

## 当前状态

重构已**全部完成并通过验收**（2025 年会话）：

- `go build ./...` / `go test ./...` 全部通过
- 浏览器端到端验证完成，12 张截图存于 `storage/verify/`
- 验证覆盖：首页/详情/归档/分类/标签/专题/搜索、老账号 `$2y$` 密码登录、发文全链路（拼音 slug + `name~` 新建标签）、评论与嵌套回复、通知红点与清零、后台全部 CRUD + 批量审核、设置保存后前台生效

演示账号：`admin@example.com` / `demo@example.com`，密码均为 `password`。

## 本机验收环境

| 组件 | 说明 |
|---|---|
| MySQL | Docker 容器 `qinblog-mysql`，端口 **3307**（3306 被本机其他项目容器占用），库 `qin-blog`，homestead/secret |
| Redis | Docker 容器 `qinblog-redis`，端口 **6380**（6379 同理被占用） |
| 初始化 | `deploy/schema.sql`（11 表 DDL + 分类/栏目种子 + 演示数据），挂载为 `/docker-entrypoint-initdb.d` 自动执行 |
| .env | 已配置好指向 3307/6380，`VITE_DEV=false` |

启动命令见 `README.md` 的"快速开始"。

## 开发环境坑点（重要）

1. **系统环境变量 `GOOS=linux`**：本机全局设置了 `GOOS=linux`，任何 go 命令前必须先 `Set-Item Env:GOOS windows`（PowerShell），否则编译产物/测试全错。建议加 `-buildvcs=false`。
2. **模板不热更新**：Go 服务启动时一次性解析全部模板（`internal/render`），改动 `web/templates/` 后必须重启服务。
3. **前端改动**：改 `web/src/` 后需 `cd web && npm run build` 重新产出 manifest；或 `.env` 设 `VITE_DEV=true` 并跑 `npm run dev` 走 dev server。
4. **PowerShell 中文**：不要用 PowerShell 文本替换工具处理中文模板文件（编码会坏），一律用编辑器/IDE 工具改。
5. **重启服务**：`Get-NetTCPConnection -LocalPort 8080` 找 PID → `Stop-Process` → 重新 `go run ./cmd/server`。

## 架构速查

模块名 `github.com/qin/qinblog`，Go 1.25。

| 目录 | 职责 |
|---|---|
| `cmd/server` | Web 入口，内嵌 cron（每日 00:05 同步 Redis 浏览量到 `posts.view_count`） |
| `cmd/indexer` | bleve 全量索引重建 |
| `internal/config` | godotenv 加载 + settings 表读写（启动时 AutoMigrate + 写缺省值） |
| `internal/database` | MySQL(GORM) / Redis(go-redis v9) 连接 |
| `internal/models` | 11 张表模型，表名/字段与 Laravel 完全一致 |
| `internal/handlers/web` | 前台：posts/comments/auth/users/topics/tags/categories/notifications/pages |
| `internal/handlers/admin` | 后台：仪表盘 + 9 组 CRUD |
| `internal/middleware` | session、auth、admin、CSRF、flash、sidebar 数据注入 |
| `internal/services` | post（核心业务）、search(bleve)、upload、mail、translate、viewcount、notification |
| `internal/render` | 模板渲染层：layout/partial 组织、模板函数（asset/route/str_limit/timeago/json/safe/dict…）、Vite manifest 解析 |
| `web/templates` | Go 模板（对照 `../resources/views` 迁移） |
| `web/src` | Vite + Vue 3：EasyMDE 编辑器、MultiSelect/SingleSelect（@vueform/multiselect）、TOC 四个组件，全局注册 + 页面 `createApp` 挂载 `#app` |

## 关键业务约定（与 Laravel 对等）

- **评论多态**：`commentable_type` 写入字符串 `App\Post`（兼容存量数据）；`parent_id` 区分评论/回复。
- **密码兼容**：Laravel bcrypt `$2y$` 前缀转 `$2a$` 后用 x/crypto/bcrypt 校验（见 services 单测）。
- **新建标签/专题**：前端传 `name~随机串`，服务端 `handleTag` 识别 `~` 后缀创建新记录。
- **slug 生成**：留空时异步（goroutine）走百度翻译，未配置 appid 回退拼音（mozillazg/go-pinyin）；唯一性用 `slug RLIKE` 查重。
- **计数维护**：`tags.post_count`、`categories.post_count`、`posts.comment_count`、`users.notification_count` 均由 service 层写操作维护（对应 Laravel Observer），**直接改库后计数会不一致**，需手动 UPDATE。
- **侧边栏缓存**：Redis 键 `cache:categories` / `cache:tags` / `cache:archives` / `cache:links` / `cache:columns`，写操作后清理；直接改库需手动 `DEL`。
- **评论审核**：admin 用户评论自动 `approved=1`，普通用户需后台批量审核。
- **body 字段**：存 HTML（同 Laravel），编辑时 html-to-markdown 转回 Markdown 给 EasyMDE。
- **后台权限**：`users.is_admin`，未迁移 laravel-admin 的 RBAC 表。
- **站点设置**：`settings` 表（key-value），替代 Laravel 的 `config/site.php` 文件。

## 未完成 / 可继续的方向

以下按计划属"已知偏差"或未在本机实测（无外部依赖环境）：

- [ ] **邮件链路实测**：注册验证邮件、密码重置邮件代码已写（gomail + 签名 token），但 `.env` 未配 SMTP，未实测发信。
- [ ] **GitHub OAuth 实测**：代码已写（x/oauth2），`.env` 未配 client id/secret。
- [ ] **图片上传实测**：`posts/upload_post_image` 接口与 EasyMDE 拖拽对接已写，浏览器验证时未走真实文件上传。
- [ ] 生产部署：目前 `go run` 方式运行，可补 Dockerfile / systemd 单元。
- [ ] 原库真实数据迁移演练（目前用的是 schema.sql 演示数据）。

## 验证证据

- 截图：`storage/verify/*.png`（home / post-show / post-create / select-open / search / notification-badge / notifications / admin-dashboard / admin-posts / admin-comments / about 等）
- 单测：`internal/services`（slug/tag/excerpt/bcrypt/bleve）、`internal/render`、`internal/support`
- 重构计划原文：会话计划文件（Spec）已全部落实，偏差点见 README"与 Laravel 版的差异"
