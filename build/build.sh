#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

# 切换到上级目录
cd ../ > /dev/null

echo "Building..."

# 清理并构建
make clean && make linux

echo "Build completed."

# 返回之前的目录
cd - > /dev/null
