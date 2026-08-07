# QinBlog Go

QinBlog 博客的 Go 重构版：由原 Laravel 5.8 项目完整对等迁移，使用 **Gin + GORM + html/template** 服务端渲染，前端保持"服务端渲染 + Vue 组件撒点"架构并升级为 **Vue 3 + Vite**。复用原有 MySQL 库表结构，无需数据迁移。

## 功能特性

- **前台**：文章列表/详情（slug 301 跳转、Redis 浏览量统计）、归档、分类、标签、专题、搜索、用户主页、关于页
- **写作**：Markdown 编辑器（EasyMDE，支持图片拖拽上传）、新建标签/专题（`name~` 约定）、slug 自动生成（百度翻译，未配置时回退拼音）
- **互动**：评论与嵌套回复（管理员免审）、评论/回复通知（导航栏红点 + 通知列表）
- **认证**：注册（邮箱验证）、登录/登出（remember me）、密码重置、GitHub OAuth；兼容 Laravel bcrypt `$2y$` 哈希，老账号可直接登录
- **搜索**：bleve v2 全文索引 + gse 中文分词，文章保存/删除实时同步索引
- **后台**：仪表盘、文章/评论（含批量审核）/分类/标签/专题/栏目/友链/用户管理、站点设置
- **调度**：内嵌 cron，每日 00:05 将 Redis 中昨日浏览量同步到 `posts.view_count`

## 技术栈

| 组件 | 选型 |
|---|---|
| Web 框架 | [Gin](https://github.com/gin-gonic/gin) |
| ORM | [GORM](https://gorm.io/)（MySQL，沿用原 Laravel 表结构） |
| 模板 | Go `html/template`（layout + partials 渲染层） |
| Session | gin-contrib/sessions（Redis store）+ 自实现 CSRF / Flash |
| 全文搜索 | [bleve v2](https://github.com/blevesearch/bleve) + [gse](https://github.com/go-ego/gse) 中文分词 |
| Markdown | goldmark（md→html）+ html-to-markdown（编辑回显）+ bluemonday（XSS 过滤） |
| 图片处理 | disintegration/imaging（上传图 resize） |
| 缓存/计数 | go-redis/v9 |
| 前端 | Vite + Vue 3 + Bootstrap 4 + EasyMDE + @vueform/multiselect |
| 调度 | robfig/cron |

## 项目结构

```
qinblog-go/
├── cmd/
│   ├── server/          # Web 服务入口（内嵌 cron 调度）
│   └── indexer/         # bleve 全量索引重建命令
├── internal/
│   ├── config/          # .env 加载、站点设置
│   ├── database/        # MySQL / Redis 连接
│   ├── models/          # GORM 模型（11 张表）
│   ├── handlers/
│   │   ├── web/         # 前台 handlers
│   │   └── admin/       # 后台 handlers
│   ├── middleware/      # session、auth、admin、CSRF、flash、sidebar
│   ├── services/        # post、search、upload、mail、translate、viewcount、notification
│   ├── render/          # 模板渲染层（模板函数、Vite manifest）
│   ├── router/          # 路由注册
│   └── support/         # 工具函数
├── web/
│   ├── templates/       # Go 模板（layouts / partials / posts / auth / admin / ...）
│   ├── src/             # Vite + Vue 3 源码（js / sass）
│   └── public/          # 静态资源、Vite 构建产物（build/）
├── deploy/
│   ├── schema.sql       # 建表 DDL + 种子数据 + 演示数据
│   └── docker/          # docker compose 部署（app + MySQL + Redis）
├── Dockerfile           # 前端构建 + Go 静态编译 + alpine 运行时
├── storage/             # bleve 索引、上传文件
├── .env.example
└── go.mod
```

## 快速开始

### 1. 准备 MySQL 与 Redis

已有数据库可跳过。本地验收可用 Docker（`deploy/schema.sql` 会自动初始化 11 张表、分类/栏目种子和演示数据）：

```bash
docker run -d --name qinblog-mysql -p 3307:3306 \
  -e MYSQL_ROOT_PASSWORD=secret \
  -e MYSQL_DATABASE=qin-blog \
  -e MYSQL_USER=homestead -e MYSQL_PASSWORD=secret \
  -v ./deploy:/docker-entrypoint-initdb.d \
  mysql:8

docker run -d --name qinblog-redis -p 6380:6379 redis:7
```

### 2. 配置

```bash
cp .env.example .env
# 按实际环境修改 DB_*、REDIS_* 等配置
```

主要配置项：

| 变量 | 说明 |
|---|---|
| `APP_KEY` | Session/签名密钥（base64） |
| `DB_*` / `REDIS_*` | 数据库与 Redis 连接 |
| `MAIL_*` | SMTP（邮箱验证、找回密码；留空则跳过发信） |
| `GITHUB_CLIENT_*` | GitHub OAuth（可选） |
| `BAIDU_TRANSLATE_*` | 百度翻译 API，用于 slug 生成（留空回退拼音） |
| `VITE_DEV` | `true` 时模板加载 Vite dev server 资源 |

### 3. 前端构建

```bash
cd web
npm install
npm run build      # 生产构建（产物 + manifest 输出到 public/build/）
# 或 npm run dev   # 开发模式（配合 .env 的 VITE_DEV=true）
```

### 4. 重建搜索索引（首次运行）

```bash
go run ./cmd/indexer
```

### 5. 启动服务

```bash
go run ./cmd/server
# 访问 http://localhost:8080
```

演示账号（使用 `deploy/schema.sql` 初始化时）：

| 账号 | 密码 | 角色 |
|---|---|---|
| admin@example.com | password | 管理员 |
| demo@example.com | password | 普通用户 |

## Docker 部署

镜像为三阶段构建（Vite 前端 → Go 静态编译 → alpine 运行时，非 root 用户运行），compose 编排 app + MySQL + Redis：

```bash
cd deploy/docker
cp .env.example .env          # 修改 APP_KEY、DB_PASSWORD、APP_URL 等
docker compose up -d --build
# 访问 http://localhost:8080（宿主端口由 .env 的 APP_HOST_PORT 决定）
```

首次启动会自动完成：MySQL 执行 `deploy/schema.sql` 建表 + 种子数据 → app 检测到索引目录为空自动全量建 bleve 索引 → 启动 Web 服务。

| 说明 | 内容 |
|---|---|
| 配置 | `deploy/docker/.env`（含密钥，勿提交）；`DB_HOST`/`REDIS_HOST` 等服务连接由 compose 固定注入 |
| 数据卷 | `qinblog_mysql-data`、`qinblog_redis-data`、`qinblog_app-storage`（bleve 索引 + 上传文件） |
| 端口 | app `APP_HOST_PORT`（默认 8080）；MySQL `3308`、Redis `6381` 仅为调试映射，生产可删 |
| 反向代理 | `APP_URL` 填真实域名；容器内固定监听 8080 |

常用运维命令：

```bash
docker compose logs -f app          # 查看日志
docker compose up -d --build app    # 更新代码后重建并滚动重启
docker compose down                 # 停止（保留数据卷）
docker compose down -v              # 停止并清空数据（含库与索引）

# 重建全文索引（bleve 为独占锁，须先停 app）
docker compose stop app
docker compose --profile tools run --rm indexer
docker compose start app
```

## 测试

```bash
go test ./...
```

覆盖：唯一 slug 生成、标签处理（新建/计数/删空）、摘要提取、Laravel `$2y$` bcrypt 兼容校验、bleve 中文索引与检索、模板渲染函数等。

## 与 Laravel 版的差异

- 后台由 laravel-admin 改为自建（`/admin`），权限简化为 `users.is_admin`，不再使用 admin_users/RBAC 表
- 站点设置由 `config/site.php` 文件改为 `settings` 数据库表，可在后台"站点设置"中维护
- 全文搜索由 TNTSearch + jieba 换为 bleve + gse
- 队列任务（slug 翻译）改为后台 goroutine；计划任务改为内嵌 cron
- 文章 `body` 字段继续存 HTML（与 Laravel 行为一致），编辑时转回 Markdown
