FROM golang:1.25-bullseye AS builder
ENV GOPROXY=https://goproxy.cn
ENV CGO_ENABLED=0
WORKDIR /app
# 先复制依赖文件以利用构建缓存
COPY go.mod go.sum ./
RUN go mod download -x
# 再复制源代码
COPY . .
RUN go build -ldflags "-w -s" -o webhook .

FROM debian:bullseye-slim
LABEL maintainer "soulteary <soulteary@gmail.com>"
# 安装 CA 证书（用于 HTTPS 请求）
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*
# 创建非 root 用户
RUN groupadd -r webhook && useradd -r -g webhook webhook
COPY --from=builder /app/webhook /usr/local/bin/webhook
RUN chmod +x /usr/local/bin/webhook
USER webhook
EXPOSE 9000/tcp
CMD ["webhook"]
