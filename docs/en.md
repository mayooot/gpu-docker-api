# GPU-Docker-API

![license](https://img.shields.io/hexpm/l/plug.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/mayooot/gpu-docker-api)](https://goreportcard.com/badge/github.com/mayooot/gpu-docker-api)

[简体中文](docs%2Fzh-cn.md)

# Overview

Use the Docker Client to invoke NVIDIA Docker to realize the business functions of GPU container.

For example, lifting GPU container configurations, starting containers without cards, and scaling up and
down volume size.

Similar to the operation on container instances in [AutoDL](https://www.autodl.com/docs/env/).

- [GPU-Docker-API](#gpu-docker-api)
- [Overview](#overview)
- [Feature](#feature)
    - [Container](#container)
    - [Volume](#volume)
    - [GPU](#gpu)
    - [Port](#port)
- [Quick Start](#quick-start)
    - [API](#api)
    - [Environmental Preparation](#environmental-preparation)
    - [Build from Source](#build-from-source)
- [Architecture](#architecture)
    - [Component Introduction](#component-introduction)
    - [Architecture Diagram](#architecture-diagram)
    - [Documents](#documents)
- [Contribute](#contribute)
- [Environment](#environment)

First I have to describe to you what a GPU container's directory should look like when it starts. It is as follows:

| name         | path          | performance           | description                                                                                                                                                                                                                                                                                                                      |
|--------------|---------------|-----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| system disk  | /             | local disk, fast      | Data will not be lost when the container is stopped. Generally system dependencies such as the Python installer are located under the system disk, which will be preserved when saving the image. The data will be copied to the new container after the container lifts the GPU and Volume configurations.                      |
| Data Disk    | /root/foo-tmp | Local, Fast           | Use Docker Volume to mount, the data will not be lost when the container is stopped, which will be preserved when saving the image. It is suitable for storing data with high IO requirements for reading and writing. The data will be copied to the new container after the container lifts the GPU and Volume configurations. |
| File Storage | /root/foo-fs  | Network Disk, General | Enables synchronized file sharing across multiple containers, such as NFS.                                                                                                                                                                                                                                                       |

We then discuss update operations (lifting GPU container configurations, scaling up and down volume size,
all of these are update operations, and for ease of understanding, we will use the term "update" below
instead of these specific operations).

When we update a container, a new container is created.

For example, if the old container foo-0 was using 3 graphics
cards, and we want it to use 5 graphics cards, calling the interface creates the new container, foo-1 will be created to
replace foo-0 (foo-0 will not be deleted), similar to how updating a Pod in K8s will be a rolling replacement.

It's worth noting that the new container does not look much different from the old one, except for the parts we
specified
to be updated, and even the software you installed, which will appear in the new container as is.

Not to mention, the
data disk, file storage, environment variables, and port mapping.

which looks pretty cool 😎.

The same is true when updating volume.

# Feature

## Container

- [x] Create GPU container
- [x] Create cardless container
- [x] Patch container GPU configuration
- [x] Patch container volume configuration
- [x] Stop container
- [x] Restart container
- [x] Execute commands inside the container
- [x] Delete container
- [x] Save container as an image
- [x] Get container creation information

## Volume

- [x] Create a volume of the specified capacity size
- [x] Delete volume
- [x] Scale up and down volume capacity size
- [x] Get volume creation information

## GPU

- [x] View GPU Usage

## Port

- [x] View Used Ports

# Quick Start

[👉 Click here to see, my environment](#Environment)

## API

Import [gpu-docker-api.openapi.json](api%2Fgpu-docker-api.openapi.json) to invoke api.

## Environmental Preparation

1. The test environment has already installed the corresponding drivers for the NVIDIA graphics card.
2. Make sure you have NVIDIA Docker installed on your test environment, installation
   tutorial: [NVIDIA Docker Installation](https://zhuanlan.zhihu.com/p/361934132).
3. To support the creation of a volume of the specified capacity size, ensure that Docker's Storage Driver is Overlay2.
   Create
   and format a partition as an XFS file system, and use the mounted directory as the
   Docker Root Dir.
   tutorial: [volume-size-scale-en.md](docs%2Fvolume%2Fvolume-size-scale-en.md)
4. Make sure your test environment has ETCD V3 installed, installation
   tutorial: [ETCD](https://github.com/etcd-io/etcd).
5. Clone and run [detect-gpu](https://github.com/mayooot/detect-gpu).

## Build from source

### Build

~~~
git clone https://github.com/mayooot/gpu-docker-api.git
cd gpu-docker-api
make build
~~~

### Modify configuration file (optional)

~~~
vim etc/config.yaml
~~~

### Run

~~~
./gpu-docker-api-${your_os}-amd64
~~~

# Architecture

The design is inspired by and borrows a lot from Kubernetes.

For example, K8s adds full information about resources (Pods, Deployment, etc.) to the ETCD and then uses the ETCD
version number for rollback.

And workQueue asynchronous processing in Client-go.

## Component Introduction

* gin：Handles HTTP requests and interface routing.

* docker-client：Docker interaction with the server.

* workQueue：Asynchronous processing tasks, for example:

    * When a container/volume is created, add the created information to the ETCD.
    * After deleting a container/volume, delete the full information about the resource from the ETCD.
    * After lifting the GPU/Volume configuration of a container, copy the data of the old container to the new
      container.
    * After scaling up and down the capacity size of a Volume resource, copy the data of the old volume to the new
      volume.

* container/volume VersionMap：

    * Generate version number when creating a container, default is 0, when container is updated, the version number
      will
      be +1.
    * Generate the version number when creating a volume, default is 0, when the volume is updated, the version number
      will
      is +1.

* gpuScheduler：A scheduler that allocates GPU resources and saves the used GPUs.
    * gpuStatusMap：
      Maintain the GPU resources of the server, when the program starts for the first time, call detect-gpu to get all
      the GPU resources, and initialize gpuStatusMap, Key is the UUID of GPU, Value is the usage, 0 means used, 1 means
      unused.

* portScheduler：A scheduler that allocates Port resources and saves the used Ports.
    * usedPortSet:
      Maintains the server's port resources. Ports that are already used are added to this Set.

* docker：The component that actually creates the resources such as container, volume, etc. The [NVIDIA
  Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html) in
  order to schedule GPUs.

* etcd：Save the container/volume creation information. For example:

    * /apis/v1/containers
    * /apis/v1/volumes
    * /apis/v1/gpus/gpuStatusMapKey
    * /apis/v1/ports/usedPortSetKey
    * /apis/v1/versions/containerVersionMapKey
    * /apis/v1/versions/volumeVersionMapKey

* detect-gpu：A simple HTTP server that calls [go-nvml](https://github.com/NVIDIA/go-nvml) to get the GPU of the host
  computer.

## Architecture Diagram

![design.png](docs%2Fdesign.png)

## Documents

* [container-gpu-scale.md](docs%2Fcontainer%2Fcontainer-gpu-scale.md)
* [volume-size-scale-en.md](docs%2Fvolume%2Fvolume-size-scale-en.md)

# Contribute

Feel free to open issues and pull requests. Any feedback is highly appreciated!

# Environment

## Development Environment

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

## Test Environment

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