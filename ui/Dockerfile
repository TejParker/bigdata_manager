FROM node:16-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制依赖文件
COPY package.json package-lock.json ./

# 安装依赖
RUN npm ci

# 复制源代码
COPY . .

# 设置前端API地址
ARG API_URL=http://localhost:8080
ENV VITE_API_BASE_URL=$API_URL

# 构建生产版本
RUN npm run build

# 使用nginx提供静态资源
FROM nginx:alpine

# 设置时区
ENV TZ=Asia/Shanghai

# 复制构建结果到nginx服务目录
COPY --from=builder /app/dist /usr/share/nginx/html

# 复制nginx配置文件
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 暴露端口
EXPOSE 80

# 启动nginx
CMD ["nginx", "-g", "daemon off;"] 