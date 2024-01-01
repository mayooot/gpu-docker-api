# gpu-docker-api

## 介绍

使用 Docker Client 调用 NVIDIA Docker 来实现 GPU 容器的业务功能。如升降 GPU 容器的配置、无卡启动容器、扩缩容 Volume
数据卷。类似于 [AutoDL](https://www.autodl.com/docs/env/) 中关于容器实例的操作。

首先我必须向你描述，一个 GPU 容器启动时，它的目录应该是怎么样的。如下：

| 名称   | 路径            | 性能     | 说明                                                                                            |
|------|---------------|--------|-----------------------------------------------------------------------------------------------|
| 系统盘  | /             | 本地盘，快  | 容器停止后数据不会丢失。一般系统依赖和 Python 安装包都会在系统盘下，保存镜像时会保留这些数据。容器升降 GPU、Volume 配置后，数据会拷贝到新容器。             |
| 数据盘  | /root/foo-tmp | 本地盘，快  | 使用 Docker Volume 挂载，容器停止后数据不会丢失，保存镜像时不会保留这些数据。适合存放读写 IO 要求高的数据。容器升降 GPU、Volume 配置后，数据会拷贝到新容器。 |
| 文件存储 | /root/foo-fs  | 网络盘，一般 | 可以实现多个容器文件同步共享，例如 NFS。                                                                        |

然后我们来讨论一下更新操作（升降 GPU 容器的配置、扩缩容 Volume 数据卷，这些都属于更新操作，为了方便理解，下面将使用 "更新"
来代替这些具体的操作）。

当我们更新一个容器时，会创建一个新的容器，例如旧的容器 foo-0 使用了 3 张显卡，我们想让它使用 5 张，调用接口后就会创建新的容器
foo-1 来代替 foo-0（foo-0 不会被删除），类似于 K8s 中更新一个 Pod，会滚动替换一样。

值得注意的是，新容器看起来和旧容器没什么不同，除了我们指定要更新的部分，甚至你安装的软件，都会原封不动的出现在新容器中。更不用说，数据盘、文件存储、环境变量、端口映射了，这看起来很酷
😎。

更新Volume 时也是如此。

## 实现的功能

### 容器（Container）

- [x] 创建 GPU 容器
- [x] 创建无卡容器
- [x] 升降容器 GPU 配置
- [x] 升降容器 Volume 配置
- [x] 停止容器
- [x] 重启容器
- [x] 在容器内部执行命令
- [x] 删除容器
- [x] 保存容器为镜像

### 卷（Volume）

- [x] 创建指定容量大小的 Volume
- [x] 删除 Volume
- [x] 扩缩容 Volume

### GPU

- [x] 查看 GPU 使用情况

### Port

- [x] 查看已使用的 Ports

## 快速开始

[👉点此查看，我的测试环境信息](#Environment)

### API

你可以通过导入 [gpu-docker-api.openapi.json](api/gpu-docker-api.openapi.json)
或查阅 [gpu-docker-api-sample-interface.md](api/gpu-docker-api-sample-interface.md)  了解并调用接口。

### 环境准备

1. 测试环境已经安装好 NVIDIA 显卡对应的驱动。
2. 确保你的测试环境已安装 NVIDIA Docker，安装教程：[NVIDIA Docker 安装](https://zhuanlan.zhihu.com/p/361934132)。
3. 为支持创建指定大小的 Volume，请确保 Docker 的 Storage Driver 为 Overlay2。创建并格式化一个分区为 XFS 文件系统，将挂载后的目录作为
   Docker Root Dir。
   详细说明：[volume-size-scale.md](docs%2Fvolume%2Fvolume-size-scale.md)
4. 确保你的测试环境已安装 ETCD V3，安装教程：[ETCD](https://github.com/etcd-io/etcd)。
5. 克隆并运行 [detect-gpu](https://github.com/mayooot/detect-gpu)。

### 编译项目

~~~
git clone https://github.com/mayooot/gpu-docker-api.git
cd gpu-docker-api
make build
~~~

### 修改配置文件（可选）

~~~
vim etc/config.yaml
~~~

### 运行项目

~~~
./gpu-docker-api-${your_os}-amd64
~~~

## 架构

设计上受到了许多 Kubernetes 的启发和借鉴。

比如 K8s 将会资源（Pod、Deployment 等）的全量信息添加到 ETCD 中，然后使用 ETCD 的版本号进行回滚。

以及 Client-go 中的 workQueue 异步处理。

### 组件介绍

* gin：处理 HTTP 请求和接口路由。

* docker-client：和服务器的 Docker 交互。

* workQueue：异步处理任务，例如：

    * 创建 Container/Volume 后，将创建的全量信息添加到 ETCD。
    * 删除 Container/Volume 后，删除 ETCD 中关于资源的全量信息。
    * 升降 Container 的 GPU/Volume 配置后，将旧 Container 的数据拷贝到新 Container 中。
    * 升降 Volume 资源的容量大小后，将旧 Volume 的数据拷贝到新的 Volume 中。

* container/volume VersionMap：

    * 创建 Container 时生成版本号，默认为 0，当 Container 被更新后，版本号＋1。
    * 创建 Volume 时生成版本号，默认为 0，当 Volume 被更新后，版本号＋1。

  程序关闭后，会将 VersionMap 写入 ETCD，当程序再次启动时，从 ETCD 中拉取数据并初始化。
* gpuScheduler：分配 GPU 资源的调度器，将容器使用 GPU 的占用情况保存到 gpuStatusMap。
    * gpuStatusMap：
      维护服务器的 GPU 资源，当程序第一次启动时，调用 detect-gpu 获取全部的 GPU 资源，并初始化 gpuStatusMap，Key 为 GPU 的
      UUID，Value 为 使用情况，0 代表未占用，1 代表已占用。

      程序关闭后，会将 gpuStatusMap 写入 ETCD，当程序再次启动时，从 ETCD 中拉取数据并初始化。

* portScheduler：分配 Port 资源的调度器，将容器使用的 Port 资源保存到 usedPortSet。
    * usedPortSet:
      维护服务器的 Port 资源，已经被占用的 Port 会被加入到这个 Set。

      程序关闭后，会将 usedPortSet 写入 ETCD，当程序再次启动时，从 ETCD 中拉取数据并初始化。

* docker：实际创建 Container、Volume等资源的组件，并安装了 NVIDIA Container Toolkit，拥有调度 GPU 的能力。

* etcd：保存 Container/Volume的全量创建信息，以及生成 mod_revision 等 Version 字段用于回滚资源的历史版本。存储在 ETCD
  中资源如下：

    * /apis/v1/containers
    * /apis/v1/volumes
    * /apis/v1/gpus/gpuStatusMapKey
    * /apis/v1/ports/usedPortSetKey
    * /apis/v1/versions/containerVersionMapKey
    * /apis/v1/versions/volumeVersionMapKey

* dete-gpu：调用 go-nvml 的一个小工具，启动时会提供一个 HTTP 接口用于获取 GPU 信息。

### 架构图

![design](docs/design.png)

### 文档

* 容器升降 GPU 资源的实现: [container-gpu-scale.md](docs%2Fcontainer%2Fcontainer-gpu-scale.md)
* Volume 扩缩容的实现: [volume-size-scale.md](docs%2Fvolume%2Fvolume-size-scale.md)

## 贡献代码

欢迎贡献代码或 issue!

## Environment

### 开发环境

~~~ 
$ sw_vers
ProductName:		macOS
ProductVersion:		14.0
BuildVersion:		23A344

$ sysctl -n machdep.cpu.brand_string
Apple M1

$ go version
go version go1.21.5 darwin/arm64
~~~

### 测试环境

~~~
$ cat /etc/issue
Ubuntu 20.04.4 LTS
~~~

~~~
$ docker info
Client: Docker Engine - Community
 Version:    24.0.5
 Context:    default
 Debug Mode: false
 Plugins:
  buildx: Docker Buildx (Docker Inc.)
    Version:  v0.11.2
    Path:     /usr/libexec/docker/cli-plugins/docker-buildx
  compose: Docker Compose (Docker Inc.)
    Version:  v2.20.2
    Path:     /usr/libexec/docker/cli-plugins/docker-compose

Server:
 Containers: 27
  Running: 20
  Paused: 0
  Stopped: 7
 Images: 38
 Server Version: 24.0.5
 Storage Driver: overlay2
  Backing Filesystem: xfs
  Supports d_type: true
  Using metacopy: false
  Native Overlay Diff: true
  userxattr: false
 Logging Driver: json-file
 Cgroup Driver: cgroupfs
 Cgroup Version: 1
 Plugins:
  Volume: local
  Network: bridge host ipvlan macvlan null overlay
  Log: awslogs fluentd gcplogs gelf journald json-file local logentries splunk syslog
 Swarm: inactive
 Runtimes: io.containerd.runc.v2 runc
 Default Runtime: runc
 Init Binary: docker-init
 containerd version: 8165feabfdfe38c65b599c4993d227328c231fca
 runc version: v1.1.8-0-g82f18fe
 init version: de40ad0
 Security Options:
  apparmor
  seccomp
   Profile: builtin
 Kernel Version: 5.4.0-100-generic
 Operating System: Ubuntu 20.04.4 LTS
 OSType: linux
 Architecture: x86_64
 CPUs: 112
 Total Memory: 1.968TiB
 Name: langfang21
 ID: 58c56043-2c92-4d9f-8cb7-14ffa0541531
 Docker Root Dir: /localData/docker
 Debug Mode: false
 Username: *****
 Experimental: false
 Insecure Registries:
  *****
  127.0.0.0/8
 Registry Mirrors:
  *****
  *****
 Live Restore Enabled: false

WARNING: No swap limit support

~~~

~~~
$ nvidia-smi 
Sat Dec  9 09:04:06 2023       
+-----------------------------------------------------------------------------+
| NVIDIA-SMI 525.85.12    Driver Version: 525.85.12    CUDA Version: 12.0     |
|-------------------------------+----------------------+----------------------+
| GPU  Name        Persistence-M| Bus-Id        Disp.A | Volatile Uncorr. ECC |
| Fan  Temp  Perf  Pwr:Usage/Cap|         Memory-Usage | GPU-Util  Compute M. |
|                               |                      |               MIG M. |
|===============================+======================+======================|
|   0  NVIDIA A100 80G...  On   | 00000000:35:00.0 Off |                    0 |
| N/A   46C    P0    73W / 300W |  57828MiB / 81920MiB |      0%      Default |
|                               |                      |             Disabled |
+-------------------------------+----------------------+----------------------+
|   1  NVIDIA A100 80G...  On   | 00000000:36:00.0 Off |                    0 |
| N/A   44C    P0    66W / 300W |  51826MiB / 81920MiB |      0%      Default |
|                               |                      |             Disabled |
+-------------------------------+----------------------+----------------------+
|   2  NVIDIA A100 80G...  On   | 00000000:39:00.0 Off |                    0 |
| N/A   45C    P0    72W / 300W |  12916MiB / 81920MiB |      0%      Default |
|                               |                      |             Disabled |
+-------------------------------+----------------------+----------------------+
|   3  NVIDIA A100 80G...  On   | 00000000:3D:00.0 Off |                    0 |
| N/A   42C    P0    62W / 300W |  12472MiB / 81920MiB |      0%      Default |
|                               |                      |             Disabled |
+-------------------------------+----------------------+----------------------+
|   4  NVIDIA A100 80G...  On   | 00000000:89:00.0 Off |                    0 |
| N/A   48C    P0    72W / 300W |  26140MiB / 81920MiB |      0%      Default |
|                               |                      |             Disabled |
+-------------------------------+----------------------+----------------------+
|   5  NVIDIA A100 80G...  On   | 00000000:8A:00.0 Off |                    0 |
| N/A   40C    P0    45W / 300W |      2MiB / 81920MiB |      0%      Default |
|                               |                      |             Disabled |
+-------------------------------+----------------------+----------------------+
|   6  NVIDIA A100 80G...  On   | 00000000:8D:00.0 Off |                    0 |
| N/A   39C    P0    46W / 300W |      2MiB / 81920MiB |      0%      Default |
|                               |                      |             Disabled |
+-------------------------------+----------------------+----------------------+
|   7  NVIDIA A100 80G...  On   | 00000000:91:00.0 Off |                    0 |
| N/A   39C    P0    46W / 300W |      2MiB / 81920MiB |      0%      Default |
|                               |                      |             Disabled |
+-----------------------------------------------------------------------------+
                                                                               
+-----------------------------------------------------------------------------+
| Processes:                                                                  |
|  GPU   GI   CI        PID   Type   Process name                  GPU Memory |
|        ID   ID                                                   Usage      |
|=============================================================================|
|    0   N/A  N/A    ******      C   ******                            *****MiB |
|    0   N/A  N/A    ******      C   ******                            *****MiB |
|    0   N/A  N/A    ******      C   ******                            *****MiB |
|    0   N/A  N/A    ******      C   ******                            *****MiB |
|    0   N/A  N/A    ******      C   ******                            *****MiB |
|    0   N/A  N/A    ******      C   ******                            *****MiB |
|    0   N/A  N/A    ******      C   ******                            *****MiB |
|    1   N/A  N/A    ******      C   ******                            *****MiB |
|    2   N/A  N/A    ******      C   ******                            *****MiB |
|    3   N/A  N/A    ******      C   ******                            *****MiB |
|    4   N/A  N/A    ******      C   ******                            *****MiB |
|    4   N/A  N/A    ******      C   ******                            *****MiB |
+-----------------------------------------------------------------------------+
~~~

