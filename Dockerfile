# 第一阶段：构建 Go 应用
FROM golang:1.23.5 AS builder

# 设置国内 Go 代理
ENV GOPROXY=https://goproxy.cn,direct
ENV GO111MODULE=on

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum 并拉取依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并构建
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bgproxy main.go

# 第二阶段：构建最终镜像
FROM openjdk:8-jre

# 创建工作目录
WORKDIR /app

# 复制构建产物到运行环境中
COPY --from=builder /app/bgproxy .

# 添加执行权限
RUN chmod +x ./bgproxy

# 启动入口
CMD ["./bgproxy"]

