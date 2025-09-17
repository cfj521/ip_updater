#!/bin/bash
# 版本号管理构建脚本
# 格式: 1.1.x (x自动递增)

MAJOR=1
MINOR=1

# 读取当前版本号
if [ ! -f version.txt ]; then
    echo "1.1.1" > version.txt
fi
VERSION=$(cat version.txt)

# 显示当前版本
echo "Building IP-Updater v${VERSION}"

# 编译Linux版本
echo "Compiling for Linux..."
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=${VERSION}" -o build/ip_updater ./cmd/ip_updater

# 检查编译是否成功
if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo "Build completed successfully!"
echo "Version: ${VERSION}"