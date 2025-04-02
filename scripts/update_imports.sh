#!/bin/bash

# 更新导入路径的脚本
# 从 github.com/yourusername/bigdata-manager 更改为 github.com/TejParker/bigdata-manager

# 定义颜色
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # 无颜色

echo -e "${GREEN}开始更新导入路径...${NC}"

# 查找所有Go文件
GO_FILES=$(find . -name "*.go" -type f)

# 遍历所有Go文件并替换导入路径
for file in $GO_FILES; do
    if grep -q "github.com/yourusername/bigdata-manager" $file; then
        echo -e "${YELLOW}处理文件: $file${NC}"
        sed -i 's|github.com/yourusername/bigdata-manager|github.com/TejParker/bigdata-manager|g' $file
    fi
done

# 更新README文件中的仓库URL
README_FILES=$(find . -name "README.md" -type f)
for file in $README_FILES; do
    if grep -q "github.com/yourusername/bigdata-manager" $file; then
        echo -e "${YELLOW}处理文件: $file${NC}"
        sed -i 's|github.com/yourusername/bigdata-manager|github.com/TejParker/bigdata-manager|g' $file
    fi
done

echo -e "${GREEN}导入路径更新完成!${NC}" 