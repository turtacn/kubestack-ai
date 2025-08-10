# 多阶段构建。Multi-stage build.
FROM golang:1.21-alpine AS builder

# 安装必要工具。Install necessary tools.
RUN apk add --no-cache git ca-certificates tzdata

# 设置工作目录。Set work directory.
WORKDIR /app

# 复制依赖文件。Copy dependency files.
COPY go.mod go.sum ./

# 下载依赖。Download dependencies.
RUN go mod download

# 复制源代码。Copy source code.
COPY . .

# 构建应用。Build application.
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X github.com/turtacn/kubestack-ai/internal/version.Version=${VERSION} \
              -X github.com/turtacn/kubestack-ai/internal/version.BuildTime=${BUILD_TIME} \
              -X github.com/turtacn/kubestack-ai/internal/version.GitCommit=${GIT_COMMIT} \
              -w -s" \
    -o ksa ./cmd/ksa

# 最终镜像。Final image.
FROM alpine:latest

# 安装依赖。Install dependencies.
RUN apk --no-cache add ca-certificates tzdata curl

# 创建用户。Create user.
RUN addgroup -g 1001 kubestack && \
    adduser -u 1001 -G kubestack -s /bin/sh -D kubestack

# 设置工作目录。Set work directory.
WORKDIR /home/kubestack

# 复制二进制文件。Copy binary.
COPY --from=builder /app/kubestack-ai /usr/local/bin/

# 创建配置和插件目录。Create config and plugin directories.
RUN mkdir -p /etc/kubestack-ai /var/lib/kubestack-ai/plugins && \
    chown -R kubestack:kubestack /etc/kubestack-ai /var/lib/kubestack-ai

# 切换用户。Switch user.
USER kubestack

# 健康检查。Health check.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ksa version || exit 1

# 入口点。Entrypoint.
ENTRYPOINT ["ksa"]

# 默认命令。Default command.
CMD ["--help"]

#Personal.AI order the ending
