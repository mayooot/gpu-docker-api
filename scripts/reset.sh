#!/bin/bash

set -o errexit
set -o nounset

ETCD_PREFIX=/gpu-docker-api
MERGE_DIR=./merges

sudo etcdctl del --prefix ${ETCD_PREFIX} > /dev/null

sudo rm -rf ${MERGE_DIR}

echo "\033[32m Reset done. \033[0m"