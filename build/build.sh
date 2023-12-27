#!/bin/bash
# 在构建之前，先进入到项目根目录

set -o errexit
set -o nounset
set -o pipefail

echo "Building..."
# 清理并构建
make clean && make linux
echo "Build completed."