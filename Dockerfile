# QinBlog Go 生产镜像：前端 Vite 构建 + Go 静态编译 + 精简运行时

# ---------- 阶段 1：前端构建（产物 web/public/build + manifest） ----------
FROM node:20-alpine AS frontend

WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ---------- 阶段 2：Go 构建（静态二进制，CGO 关闭） ----------
FROM golang:1.25-alpine AS backend

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/

ENV CGO_ENABLED=0 GOOS=linux
# buildvcs=false：构建上下文不含 .git，避免 VCS 戳记失败
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/server ./cmd/server && \
    go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/indexer ./cmd/indexer

# ---------- 阶段 3：运行镜像 ----------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 qinblog

WORKDIR /app

COPY --from=backend /out/server /out/indexer /app/bin/
# 模板与静态资源按相对路径读取（render.Init("web/templates") / r.Static("web/public/...")）
COPY web/templates/ /app/web/templates/
COPY web/public/images/ /app/web/public/images/
COPY --from=frontend /src/web/public/build/ /app/web/public/build/

# 索引与上传目录，compose 中挂载为数据卷
RUN mkdir -p /app/storage/bleve /app/storage/app/public && \
    chown -R qinblog:qinblog /app/storage

ENV TZ=Asia/Shanghai \
    APP_ENV=production \
    APP_PORT=8080 \
    BLEVE_INDEX_PATH=/app/storage/bleve/posts.bleve \
    UPLOAD_PATH=/app/storage/app/public

USER qinblog
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -qO- "http://127.0.0.1:${APP_PORT}/" >/dev/null 2>&1 || exit 1

# 首次启动（索引目录不存在）自动全量建索引，随后启动 Web 服务
CMD ["sh", "-c", "[ -d \"$BLEVE_INDEX_PATH\" ] || /app/bin/indexer; exec /app/bin/server"]
