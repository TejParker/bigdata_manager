#!/bin/bash

# 大数据集群管理平台部署脚本
# 支持Docker、Docker Compose和Kubernetes部署

set -e

# 定义颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # 无颜色

# 显示帮助信息
show_help() {
  echo -e "${GREEN}大数据集群管理平台部署脚本${NC}"
  echo ""
  echo "用法: $0 [选项]"
  echo ""
  echo "选项:"
  echo "  -h, --help              显示帮助信息"
  echo "  -m, --mode <模式>       部署模式: docker, compose, k8s (必选)"
  echo "  -r, --registry <仓库>   镜像仓库地址 (K8s模式下必选)"
  echo "  -n, --namespace <命名空间>  Kubernetes命名空间 (默认: bigdata-manager)"
  echo ""
  echo "示例:"
  echo "  $0 -m docker            使用Docker单机部署"
  echo "  $0 -m compose           使用Docker Compose部署"
  echo "  $0 -m k8s -r my-registry.io  使用Kubernetes部署，镜像从my-registry.io拉取"
  echo ""
}

# 参数解析
MODE=""
REGISTRY=""
NAMESPACE="bigdata-manager"

while [[ $# -gt 0 ]]; do
  case $1 in
    -h|--help)
      show_help
      exit 0
      ;;
    -m|--mode)
      MODE="$2"
      shift 2
      ;;
    -r|--registry)
      REGISTRY="$2"
      shift 2
      ;;
    -n|--namespace)
      NAMESPACE="$2"
      shift 2
      ;;
    *)
      echo -e "${RED}错误: 未知选项 $1${NC}"
      show_help
      exit 1
      ;;
  esac
done

# 验证参数
if [[ -z "$MODE" ]]; then
  echo -e "${RED}错误: 必须指定部署模式 (-m, --mode)${NC}"
  show_help
  exit 1
fi

if [[ "$MODE" == "k8s" && -z "$REGISTRY" ]]; then
  echo -e "${RED}错误: Kubernetes部署模式必须指定镜像仓库 (-r, --registry)${NC}"
  show_help
  exit 1
fi

# 工作目录检查
if [[ ! -f "go.mod" || ! -d "cmd" || ! -d "internal" ]]; then
  echo -e "${RED}错误: 请在项目根目录运行此脚本${NC}"
  exit 1
fi

# Docker部署
docker_deploy() {
  echo -e "${GREEN}开始Docker单机部署...${NC}"
  
  # 构建镜像
  echo -e "${YELLOW}构建Server镜像...${NC}"
  docker build -t bigdata-manager-server:latest -f Dockerfile .
  
  echo -e "${YELLOW}构建Agent镜像...${NC}"
  docker build -t bigdata-manager-agent:latest -f Dockerfile.agent .
  
  echo -e "${YELLOW}构建UI镜像...${NC}"
  docker build -t bigdata-manager-ui:latest -f ui/Dockerfile ./ui
  
  # 创建网络
  docker network create --driver bridge bigdata-net 2>/dev/null || true
  
  # 启动MySQL
  echo -e "${YELLOW}启动MySQL...${NC}"
  docker run -d --name bigdata-mysql \
    --network bigdata-net \
    -e MYSQL_ROOT_PASSWORD=rootpassword \
    -e MYSQL_DATABASE=bigdata_manager \
    -e MYSQL_USER=bigdata \
    -e MYSQL_PASSWORD=bigdata123 \
    -v mysql-data:/var/lib/mysql \
    -p 3306:3306 \
    mysql:8.0
  
  # 等待MySQL启动
  echo -e "${YELLOW}等待MySQL启动...${NC}"
  sleep 20
  
  # 启动Server
  echo -e "${YELLOW}启动Server...${NC}"
  docker run -d --name bigdata-server \
    --network bigdata-net \
    -e DB_HOST=bigdata-mysql \
    -e DB_PORT=3306 \
    -e DB_USER=bigdata \
    -e DB_PASSWORD=bigdata123 \
    -e DB_NAME=bigdata_manager \
    -e JWT_SECRET=yoursecretkey \
    -p 8080:8080 \
    bigdata-manager-server:latest
  
  # 启动UI
  echo -e "${YELLOW}启动UI...${NC}"
  docker run -d --name bigdata-ui \
    --network bigdata-net \
    -p 80:80 \
    bigdata-manager-ui:latest
  
  # 启动演示Agent
  echo -e "${YELLOW}启动演示Agent...${NC}"
  docker run -d --name bigdata-agent-demo \
    --network bigdata-net \
    -e SERVER_ADDRESS=http://bigdata-server:8080 \
    -e HOST_ID=demo-host-1 \
    -v /proc:/host/proc:ro \
    -v /sys:/host/sys:ro \
    -v /var/run/docker.sock:/var/run/docker.sock:ro \
    bigdata-manager-agent:latest
  
  echo -e "${GREEN}Docker部署完成!${NC}"
  echo -e "访问 http://localhost 使用系统"
}

# Docker Compose部署
compose_deploy() {
  echo -e "${GREEN}开始Docker Compose部署...${NC}"
  
  # 检查docker-compose文件
  if [[ ! -f "docker-compose.yml" ]]; then
    echo -e "${RED}错误: docker-compose.yml文件不存在${NC}"
    exit 1
  fi
  
  # 启动服务
  docker-compose up -d
  
  echo -e "${GREEN}Docker Compose部署完成!${NC}"
  echo -e "访问 http://localhost 使用系统"
}

# Kubernetes部署
k8s_deploy() {
  echo -e "${GREEN}开始Kubernetes部署...${NC}"
  
  # 检查kubectl
  if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}错误: 未安装kubectl命令${NC}"
    exit 1
  fi
  
  # 检查K8s配置文件
  if [[ ! -d "k8s" ]]; then
    echo -e "${RED}错误: k8s目录不存在${NC}"
    exit 1
  fi
  
  # 构建镜像
  echo -e "${YELLOW}构建Server镜像...${NC}"
  docker build -t ${REGISTRY}/bigdata-manager-server:latest -f Dockerfile .
  
  echo -e "${YELLOW}构建Agent镜像...${NC}"
  docker build -t ${REGISTRY}/bigdata-manager-agent:latest -f Dockerfile.agent .
  
  echo -e "${YELLOW}构建UI镜像...${NC}"
  docker build -t ${REGISTRY}/bigdata-manager-ui:latest -f ui/Dockerfile ./ui
  
  # 推送镜像
  echo -e "${YELLOW}推送镜像到仓库...${NC}"
  docker push ${REGISTRY}/bigdata-manager-server:latest
  docker push ${REGISTRY}/bigdata-manager-agent:latest
  docker push ${REGISTRY}/bigdata-manager-ui:latest
  
  # 创建命名空间
  echo -e "${YELLOW}创建命名空间...${NC}"
  kubectl apply -f k8s/namespace.yaml
  
  # 替换变量
  echo -e "${YELLOW}部署服务...${NC}"
  sed "s/\${REGISTRY}/${REGISTRY}/g" k8s/server.yaml | kubectl apply -f -
  sed "s/\${REGISTRY}/${REGISTRY}/g" k8s/ui.yaml | kubectl apply -f -
  sed "s/\${REGISTRY}/${REGISTRY}/g" k8s/agent.yaml | kubectl apply -f -
  kubectl apply -f k8s/mysql.yaml
  
  echo -e "${GREEN}Kubernetes部署完成!${NC}"
  echo -e "请配置DNS或修改hosts，将bigdata-manager.example.com指向集群的Ingress IP"
}

# 执行部署
case $MODE in
  docker)
    docker_deploy
    ;;
  compose)
    compose_deploy
    ;;
  k8s)
    k8s_deploy
    ;;
  *)
    echo -e "${RED}错误: 未知的部署模式: $MODE${NC}"
    show_help
    exit 1
    ;;
esac 