#!/bin/bash
# 版本号管理构建脚本
# 格式: 1.1.x (x自动递增)

MAJOR=1
MINOR=1

# 读取当前构建号
if [ ! -f version.txt ]; then
    echo "1" > version.txt
fi
BUILD=$(cat version.txt)

# 显示当前版本
VERSION="${MAJOR}.${MINOR}.${BUILD}"
echo "Building IP-Updater v${VERSION}"

# 编译Linux版本
echo "Compiling for Linux..."
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=${VERSION}" -o build/ip_updater ./cmd/ip_updater

# 检查编译是否成功
if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

# 编译成功，递增构建号
NEW_BUILD=$((BUILD + 1))
echo "${NEW_BUILD}" > version.txt

echo "Build completed successfully!"
echo "Version: ${VERSION}"
echo "Next build will be: ${MAJOR}.${MINOR}.${NEW_BUILD}"