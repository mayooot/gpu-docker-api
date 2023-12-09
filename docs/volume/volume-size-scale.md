# Docker Volume 扩缩容

## 前提
我在测试环境中，使用的存储驱动和文件系统如下：
~~~
Storage Driver: overlay2
Backing Filesystem: xfs
~~~
~~~
$ docker info | grep "Storage Driver"
 Storage Driver: overlay2
 
$ docker info | grep "Docker Root Dir"
 Docker Root Dir: /localData/docker
 
$ df -Th | grep -m 1 "/localData"
/dev/mapper/example-lvdata xfs 10T 2.3T 7.7T 30% /localData
~~~



需要注意的是，当我们选择不同的 Storage Driver 时，镜像、容器、卷存储的磁盘结构是不一样的。

同时它们对`设置 Volume 大小`这个功能不一定支持。比如 Device Mapper 是可以的，可以使用下面的方式设置大小
~~~
$ docker -s devicemapper
$ docker volume create --driver local --opt=o=size=20 --name data
~~~
考虑到安装新版 Docker 时，默认的 Storage Driver 是 Overlay2，但它是不支持设置大小，我们可以通过修改 Docker Root Dir 的路径，然后使用 xfs 文件系统，这样可以通过下面的方式来制定大小

~~~
$ docker volume create --opt size=20GB --name foo
$ docker volume inspect foo
[
    {
        "CreatedAt": "2023-12-09T12:28:20Z",
        "Driver": "local",
        "Labels": null,
        "Mountpoint": "/localData/docker/volumes/foo/_data",
        "Name": "foo",
        "Options": {
            "size": "20GB"
        },
        "Scope": "local"
    }
]
~~~

## 实现

和 Docker 一样，当我们使用 `docker run -d -P nginx:latest`时，很容易的就能启动一个 Nginx 进程，但是背后的 Cgroups和 Namespace 的隔离，以及镜像、容器存储等，我们是不需要考虑的。只需要记住简单的命令。

我们的这个程序也是这样，只是完成了配置工作，剩下的实现也非常简单。

可以概括为以下内容：

扩容：

1. 创建一个大小为 10Gi 的 Volume，名为 foo，现在想要扩容到 20Gi
2. 重新创建一个大小为 20Gi 的 Volume，名为 bar，然后将 foo 的实际存储目录下的内容 move 到 bar 的存储目录
3. 从 ETCD 拉取创建容器的全量信息，修改卷挂载内容。使用这些信息重新创建一个容器，然后提交到 ETCD，最后返回给用户新容器的 ID、Name

