#!/bin/bash
# go to the project root directory and execute bash `build/build.sh`

set -o errexit
set -o nounset
set -o pipefail

echo "Building..."
make clean && make linux
echo "Build completed."