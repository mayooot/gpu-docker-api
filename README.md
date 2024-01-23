# GPU-Docker-API

![license](https://img.shields.io/hexpm/l/plug.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/mayooot/gpu-docker-api)](https://goreportcard.com/badge/github.com/mayooot/gpu-docker-api)

[简体中文](docs%2Fzh-cn.md)
> ⚠️注意：中文文档可能落后于英文文档，请以英文文档为准。

Try to keep it simple.

# Overview

Use the Docker Client to invoke NVIDIA Docker to realize the business functions of GPU container.

For example, lifting GPU container configurations, starting containers without cards, and scaling up and
down volume size.

Similar to the operation on container instances in [AutoDL](https://www.autodl.com/docs/env/).

- [GPU-Docker-API](#gpu-docker-api)
- [Overview](#overview)
- [Feature](#feature)
    - [ReplicaSet](#replicaset)
    - [Volume](#volume)
    - [Resource](#resource)
- [Quick Start](#quick-start)
    - [How To Use API](#how-to-use-api)
    - [Environmental Preparation](#environmental-preparation)
    - [Build From Source](#build-from-source)
    - [Download From Release](#download-from-release)
    - [Run](#run)
    - [How To Reset](#how-to-reset)
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

---

Last but not least, you can see that we're using a ReplicaSet instead of a container, which if you're familiar with K8s,
you probably already know what that means, you can
see [ReplicaSet](https://kubernetes.io/zh-cn/docs/concepts/workloads/controllers/replicaset/).

In this project, ReplicaSet is just a concept, there is no concrete implementation, responsible for managing the
container's history version,
and implement the function of rollback to the specified version.

# Feature

## ReplicaSet

- [x] Run a container via replicaSet
- [x] Commit container as an image via replicaSet
- [x] Execute a command in the container via replicaSet
- [x] Patch a container via replicaSet
- [x] Rollback a container via replicaSet
- [x] Stop a container via replicaSet
- [x] Restart a container via replicaSet
- [x] Pause a replicaSet via replicaSet
- [x] Continue a replicaSet via replicaSet
- [x] Get version info about replicaSet
- [x] Get all version info about replicaSet
- [x] Delete a container via replicaSet

## Volume

- [x] Create a volume
- [x] Patch a volume
- [x] Get version info about a volume
- [x] Get all version info about a volume
- [x] Delete a volume

## Resource

- [x] Get gpu usage status
- [x] Get port usage status

# Quick Start

[👉 Click here to see, my environment](#Environment)

## How To Use API

Select any of the following.

* Import [gpu-docker-api-en.openapi.json](api%2Fgpu-docker-api-en.openapi.json) to [ApiFox](https://apifox.com).
* View [gpu-docker-api-en.md](api%2Fgpu-docker-api-en.md).

## Environmental Preparation

1. The Linux servers has installed NVIDIA GPU drivers, NVIDIA Docker, ETCD V3.

2. [Optional] If you want to specify the size of the docker volume, you need to specify the Docker `Storage Driver`
   as `Overlay2`,
   and set the `Docker Root Dir` to the `XFS` file system.

## Build From Source

~~~
$ git clone https://github.com/mayooot/gpu-docker-api.git
$ cd gpu-docker-api
$ make build
~~~

## Download From Release

[release](https://github.com/mayooot/gpu-docker-api/releases)

## Run

You can get help and the default configuration with `-h` parameter.

~~~
$ ./gpu-docker-api-linux-amd64 -h
GPU-DOCKER-API
 BRANCH: feat/union-patch-and-version-control
 Version: v0.0.2-12-gc29670a
 COMMIT: c29670a1dfa8bc5470e282ce9b214398baab3a15
 GoVersion: go1.21.4
 BuildTime: 2024-01-23T13:55:51+0800

Usage of ./gpu-docker-api-linux-amd64:
  -a, --addr string        Address of gpu-docker-routers server,format: ip:port (default "0.0.0.0:2378")
  -e, --etcd string        Address of etcd server,format: ip:port (default "0.0.0.0:2379")
  -l, --logLevel string    Log level, optional: release (default "debug")
  -p, --portRange string   Port range of docker container,format: startPort-endPort (default "40000-65535")
pflag: help requested
~~~

And enjoy it.

~~~
$ ./gpu-docker-api-linux-amd64
~~~

## How To Reset

As you know, we save some information in etcd and locally, so when you want to delete them,
you can use this [reset.sh](scripts%2Freset.sh).

Or if you downloaded the executable file from release, you can use the following command to get it and place it with
executable file.

```
wget https://github.com/mayooot/gpu-docker-api/blob/main/scripts/reset.sh
```

# Architecture

The design is inspired by and borrows a lot from Kubernetes.

For example, K8s adds full information about resources (Pods, Deployment, etc.) to the ETCD and then uses the ETCD
version number for rollback.

And workQueue asynchronous processing in Client-Go.

## Component Introduction

* gin：Handles HTTP requests and interface routing.

* docker-client：Docker interaction with the server.

* workQueue：Asynchronous processing tasks, for example:

    * When a container/volume is created, add the created information to the ETCD.
    * After deleting a container/volume, delete the full information about the resource from the ETCD.

* container/volume VersionMap：

    * Generate version number when creating a container, default is 1, when container is updated, the version number
      will
      be +1.
    * Generate the version number when creating a volume, default is 1, when the volume is updated, the version number
      will
      is +1.

* gpuScheduler：A scheduler that allocates GPU resources and saves the used GPUs.
    * gpuStatusMap：
      Maintain the GPU resources of the server, when the program starts for the first time, call `nvidia-smi` to get all
      the GPU resources, and initialize gpuStatusMap.
      Key is the UUID of GPU, Value is the usage, 0 means used, 1 means
      unused.

* portScheduler：A scheduler that allocates Port resources and saves the used Ports.
    * usedPortSet:
      Maintains the server's port resources. Ports that are already used are added to this Set.

* docker：The component that actually creates the resources such as container, volume, etc. The [NVIDIA
  Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html) in
  order to schedule GPUs.

* etcd：Save the container/volume creation information. The following keys are currently in use:

    * /gpu-docker-api/apis/v1/containers
    * /gpu-docker-api/apis/v1/volumes
    * /gpu-docker-api/apis/v1/gpus/gpuStatusMapKey
    * /gpu-docker-api/apis/v1/ports/usedPortSetKey
    * /gpu-docker-api/apis/v1/merges/containerMergeMapKey
    * /gpu-docker-api/apis/v1/versions/containerVersionMapKey
    * /gpu-docker-api/apis/v1/versions/volumeVersionMapKey

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

