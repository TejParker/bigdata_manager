FROM golang:1.23.4-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的系统依赖
RUN apk add --no-cache gcc musl-dev git

# 复制go.mod和go.sum文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bigdata-manager-server ./cmd/server/main.go

# 使用小型基础镜像
FROM alpine:3.15

# 安装必要的系统工具
RUN apk add --no-cache ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非root用户
RUN adduser -D -g '' appuser

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/bigdata-manager-server /app/
# 复制配置文件
COPY ./configs/config.yaml /app/configs/

# 使用非root用户运行
USER appuser

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["/app/bigdata-manager-server"] 