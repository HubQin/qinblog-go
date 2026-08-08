---
name: deploy-qinblog-lnmp-shared
description: Build, push to Tencent Cloud CCR, and deploy QinBlog Go to an existing LNMP Docker Compose stack. Use when the user asks to package, deploy, push image, or update the production deployment of QinBlog.
---

# Deploy QinBlog to LNMP Shared Stack

完整流程：本地构建前端 → 构建 Docker 镜像 → 推送至腾讯云 CCR → 线上 LNMP 共享栈部署。

## 0. 前置检查

- 本地 Docker 可用（`docker --version`）
- 开发环境 MySQL/Redis 运行中（本地开发用，构建部署不需要）
- 腾讯云 CCR 镜像仓库：`ccr.ccs.tencentyun.com/qinblog/qinblog`
- 线上服务器已有 LNMP 栈（MySQL + Redis + nginx）

## 1. 本地构建前端

```bash
cd web
npm run build
# 产物写入 web/public/build/
```

## 2. 构建 Docker 镜像

```bash
# 在项目根目录执行
docker compose -f deploy/docker/docker-compose.yml build

# 或手动构建
docker build -t qinblog:latest -t qinblog:$(date +%Y-%m-%d) .
```

## 3. 推送至腾讯云 CCR

```bash
# 登录（首次需要）
docker login ccr.ccs.tencentyun.com --username 100035609081

# 打标签
docker tag qinblog:latest ccr.ccs.tencentyun.com/qinblog/qinblog:latest
docker tag qinblog:latest ccr.ccs.tencentyun.com/qinblog/qinblog:$(date +%Y-%m-%d)

# 推送
docker push ccr.ccs.tencentyun.com/qinblog/qinblog:latest
docker push ccr.ccs.tencentyun.com/qinblog/qinblog:$(date +%Y-%m-%d)
```

## 4. 线上部署（LNMP 共享栈方式 B）

### 4.1 登录服务器并拉取镜像

```bash
ssh your-server
docker login ccr.ccs.tencentyun.com --username 100035609081
docker pull ccr.ccs.tencentyun.com/qinblog/qinblog:latest
```

### 4.2 准备 qinblog.env

```bash
# 在 LNMP 部署目录下创建
touch qinblog.env
chmod 600 qinblog.env
```

内容模板（参考 `deploy/docker/lnmp-shared/qinblog.env.example`）：

```ini
# 应用
APP_NAME=QinBlog
APP_ENV=production
APP_KEY=your-random-32-byte-base64-key
APP_URL=https://your-domain.com

# 数据库（连接 LNMP 栈中的 mysql 服务）
DB_HOST=mysql
DB_PORT=3306
DB_DATABASE=qinblog
DB_USERNAME=qinblog
DB_PASSWORD=your-db-password

# Redis（连接 LNMP 栈中的 redis 服务）
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=

# 其他配置
MAIL_HOST=
MAIL_PORT=25
MAIL_USERNAME=
MAIL_PASSWORD=
MAIL_FROM_ADDRESS=hello@example.com
MAIL_FROM_NAME=QinBlog
```

### 4.3 编辑 LNMP 的 docker-compose.yml

在 `services:` 下添加 qinblog 服务，并补充数据卷定义：

```yaml
services:
  # ... 已有的 mysql/redis/php/nginx 服务 ...

  qinblog:
    image: ccr.ccs.tencentyun.com/qinblog/qinblog:latest
    restart: unless-stopped
    env_file:
      - ./qinblog.env
    environment:
      APP_ENV: production
      APP_PORT: 8080
      VITE_DEV: "false"
      BLEVE_INDEX_PATH: /app/storage/bleve/posts.bleve
      UPLOAD_PATH: /app/storage/app/public
    ports:
      - "127.0.0.1:8080:8080"
    volumes:
      - qinblog-storage:/app/storage
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_started

volumes:
  # ... 已有的 volumes ...
  qinblog-storage:
```

### 4.4 nginx 反向代理配置

在 nginx 的 `conf.d/` 下添加 `qinblog.conf`：

```nginx
server {
    listen 80;
    server_name blog.your-domain.com;

    location / {
        proxy_pass http://qinblog:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 4.5 初始化数据库

```bash
# 创建数据库（如果不存在）
docker exec lnmp-mysql-1 mysql -uroot -p"$DB_ROOT_PASSWORD" -e "CREATE DATABASE IF NOT EXISTS qinblog CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 导入 schema
docker exec -i lnmp-mysql-1 mysql -uroot -p"$DB_ROOT_PASSWORD" qinblog < deploy/schema.sql
```

### 4.6 启动服务

```bash
docker compose up -d qinblog
```

### 4.7 重建全文索引

```bash
docker compose stop qinblog
docker compose run --rm qinblog /app/bin/indexer
docker compose start qinblog
```

## 5. 验证

```bash
# 检查容器状态
docker compose ps qinblog

# 验证 HTTP 响应
curl -I http://localhost:8080/

# 查看日志
docker compose logs --tail=50 qinblog
```

## 注意事项

- `APP_KEY` 必须是 32 字节的 base64 编码字符串，用于 session 加密。可在本地 `.env` 中获取或在线生成
- `MYSQL_ROOT_PASSWORD` 在 compose 中仅首次初始化生效，已有数据库需手动 `ALTER USER`
- Bleve 索引文件是独占锁，重建索引前必须停掉 qinblog 容器
- 端口绑定 `127.0.0.1:8080:8080` 可防止外部直接访问，仅通过 nginx 反向代理暴露
- `qinblog.env` 含数据库密码和 APP_KEY，务必 `chmod 600` 并排除在 git 之外