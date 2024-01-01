## 接口示例文档

> ⚠️ **注意**：该文档为手写的示例文档，主要用于解释接口设计，最新版本的接口文档请导入[[gpu-docker-api.openapi.json](gpu-docker-api.openapi.json)]

> ⚠️ **注意**：因为使用` RESTful` 风格的 `API` 设计，所以请求接口中会存在 `Path` 参数，为了方便书写，例子中的`请求接口`中使用 `{Param} ` 的方式来表示。同时会标注，本次请求中使用的 `{Param}` 的值
>
> **📢 关于 Volume 和 Container 的更改操作，如：**
>
> * **更改 Volume 大小**
> * **更改容器的 GPU 配置**
> * **更改容器的 Volume 配置**
>
> **它们更改前后的数据都会存在，比如 Volume 之前存储了一些文件，扩容/缩容后，文件依然存在于新的 Volume 中。**
>
> **比如容器更改 Volume/GPU 前，在容器里安装了 VIM、下载了一些文件，在更改Volume/GPU后，安装的 VIM、文件依然存在于新的容器中。**

## Volume

> 如果要创建/更改 Volume的大小，Docker 应使用 Overlay2 存储引擎，并且将 Docker Root Dir 的目录改为 XFS 文件系统。

### 创建指定大小的 Volume

描述：大小支持的单位有：KB, MB, GB, TB（不区分大小写）

请求方法：`POST`

请求接口：`/api/v1/volumes`

载荷：

~~~json
{
    "name": "rubVol",
    "size": "20GB"
}
~~~

响应：

~~~json
{
    "code": 200,
    "msg": "success",
    "data": {
        "name": "rubVol-0",
        "size": "20GB"
    }
}
~~~

Docker inspect：

~~~
$ docker volume inspect rubVol-0
[
    {
        "CreatedAt": "2023-12-26T06:05:49Z",
        "Driver": "local",
        "Labels": null,
        "Mountpoint": "/localData/docker/volumes/rubVol-0/_data",
        "Name": "rubVol-0",
        "Options": {
            "size": "20GB"
        },
        "Scope": "local"
    }
]
~~~



### 更改 Volume 的大小

> 1. 无论扩容/缩容，如果操作前后大小不变，那么就会跳过。
>
>    例如当前 Volume 大小为 20GB，扩容/缩容后还是20GB。
>
> 2. 更改大小时，会重新创建一个 Volume，比如 foo-0 的大小为 10GB，扩容成 20GB，新的 Volume 名称为 foo-1。此时不能再对 foo-0 进行更改 Volume 操作，因为最新的版本是 foo-1。

#### 扩容

请求方法：`PATCH`

请求接口： `/api/v1/volumes/{name}/size`

Param： `rubVol-0`

载荷：

~~~json
{
    "size": "50GB"
}
~~~

响应：

~~~json
{
    "code": 200,
    "msg": "success",
    "data": {
        "name": "rubVol-1",
        "size": "50GB"
    }
}
~~~

Docker inspect：

~~~
$ docker volume inspect rubVol-1
[
    {
        "CreatedAt": "2023-12-26T06:09:39Z",
        "Driver": "local",
        "Labels": null,
        "Mountpoint": "/localData/docker/volumes/rubVol-1/_data",
        "Name": "rubVol-1",
        "Options": {
            "size": "50GB"
        },
        "Scope": "local"
    }
]
~~~

#### 缩容

描述：如果用户之前的 Volume 已使用的空间大于缩容之后的空间，那么会失败。比如用户使用的 Volume 大小为 10GB，实际使用了 6GB，那么是不能缩容 Volume 为 5GB 的。

请求方法：`PATCH`

请求接口：`/api/v1/volumes/{name}/size`

Param： `rubVol-1`

载荷：

~~~json
{
    "size": "10GB"
}
~~~

响应：

~~~json
{
    "code": 200,
    "msg": "success",
    "data": {
        "name": "rubVol-2",
        "size": "10GB"
    }
}
~~~

Docker inspect：

~~~
$ docker volume inspect rubVol-2
[
    {
        "CreatedAt": "2023-12-26T06:37:13Z",
        "Driver": "local",
        "Labels": null,
        "Mountpoint": "/localData/docker/volumes/rubVol-2/_data",
        "Name": "rubVol-2",
        "Options": {
            "size": "10GB"
        },
        "Scope": "local"
    }
]
~~~

### 删除 Volume

请求方法：`DELETE`

请求接口：`/api/v1/volumes/{name}`

Param： `rubVol-2`

响应：

~~~json
{
    "code": 200,
    "msg": "success",
    "data": null
}
~~~

## Container

### 创建容器

> 创建容器使用的镜像为 nvidia/cuda:10.0-base，所以创建容器时需要手动 pull 一下。
>
> 在业务中提供 GPU 算力容器时，一般都会使用定制化的镜像，所以没有在创建容器时加入自动拉取的镜像的逻辑。（🤔可能以后会加入）

> 其他参数说明：
>
> 1. binds：代表卷挂载。
>    * 如果以 / 开头，将宿主机的文件夹/文件挂载到容器内。
>    * 不以 / 开头，将 Volume 挂载到容器内。
> 2. env：环境变量，使用 FOO=bar 的格式传递即可。
> 3. Ports：端口映射。使用例子中的格式即可。

#### 创建无卡容器

描述：将 gpuCount 字段设置为 0，即不使用 GPU（无卡容器不能使用 nvidia-smi 命令，同时可能少一些 NVIDIA 显卡驱动或工具）

请求方法：`POST`

请求接口：`/api/v1/containers`

载荷：

~~~json
{
    "imageName": "nvidia/cuda:10.0-base",
    "containerName": "knock",
    "gpuCount": 0,
    "binds": [
        {
            "src": "veil-0",
            "dest": "/root/veil-0"
        },
        {
            "src": "/mynfs/data/ctr-knock",
            "dest": "/root/data/ctr-knock"
        }
    ],
    "env": [
        "USER=foo"
    ],
    "Ports": [
        {
            "hostPort": 2333,
            "containerPort": 22
        }
    ]
}
~~~

响应：

~~~json
{
    "code": 200,
    "msg": "success",
    "data": {
        "id": "444e28edf32f74caf081550ecf03183b47b28307a8a548503aa08275221f4698",
        "name": "knock-0"
    }
}
~~~

#### 创建有卡容器

描述：创建一个 GPU 容器，可进入容器后，使用 nvidia-smi 查看显卡使用情况。如果空闲的卡数小于 gpuCount，那么会创建失败。

请求方法：`POST`

请求接口：`/api/v1/containers`

载荷：

~~~json
{
    "imageName": "nvidia/cuda:10.0-base",
    "containerName": "knockGpu",
    "gpuCount": 3,
    "binds": [
        {
            "src": "veil-0",
            "dest": "/root/veil-0"
        },
        {
            "src": "/mynfs/data/ctr-knock",
            "dest": "/root/data/ctr-knock"
        }
    ],
    "env": [
        "USER=foo"
    ],
    "Ports": [
        {
            "hostPort": 2333,
            "containerPort": 22
        }
    ]
}
~~~

响应：

~~~json
{
    "code": 200,
    "msg": "success",
    "data": {
        "id": "0eaf7ca4bc8c8c639f41ca847fbc49cbf259471d729f544964c4f3d00341b7f8",
        "name": "knockGpu-0"
    }
}
~~~

Docker inspect：

~~~
$ docker inspect knockGpu-0 | grep -A 4 DeviceIDs
                    "DeviceIDs": [
                        "GPU-281d9730-5a26-7c56-12fb-3a3d5a24ab68",
                        "GPU-7a42be89-64fe-5383-c7be-49d199a96b3d",
                        "GPU-dc6d913c-8df4-a9a4-49e6-b82fcba5a6f9"
                    ],
~~~

### 更改容器的 GPU 配置

> 1. 如果调用接口前后，容器的 GPU 数量不变，那么会直接跳过。比如有卡容器使用了 3 张卡，调用接口时传递的 gpuCount 仍然为 3。无卡容器的 gpuCount 为 0，调用接口时传递的 gpuCount 仍然为 0。
> 2. 升降 GPU 配置时，会重新创建一个容器，比如 foo-0 容器的 gpuCount 为 3，升级到 5 张卡，新的容器名称为 foo-1。此时不能再对 foo-0 进行更改 GPU/更改 Volume 的操作，因为最新版本是 foo-0。
> 3. 可以将无卡容器变为有卡容器，也可将有卡容器转为无卡容器，当然也可以升降有卡容器的 GPU 卡数。

#### 升级 GPU 配置

请求方法：`PATCH`

请求接口：`/api/v1/containers/{name}/gpu`

Param： `knockGpu-0`

载荷：

~~~json
{
    "gpuCount": 5
}
~~~

响应：

~~~json
{
    "code": 200,
    "msg": "success",
    "data": {
        "id": "f14e23c3b76bb25f67969ac5736f679c2aa09e7c90dd9d64d30629dd0b59c71d",
        "name": "knockGpu-1"
    }
}
~~~

Docker inspect：

~~~
$ docker inspect knockGpu-1 | grep -A 6 DeviceIDs
                    "DeviceIDs": [
                        "GPU-281d9730-5a26-7c56-12fb-3a3d5a24ab68",
                        "GPU-7a42be89-64fe-5383-c7be-49d199a96b3d",
                        "GPU-dc6d913c-8df4-a9a4-49e6-b82fcba5a6f9",
                        "GPU-82fbe07b-200b-1d4c-4fbe-b0b54db86be5",
                        "GPU-36009026-9470-a2e0-73d3-222a63b82e4e"
                    ],
~~~

#### 将有卡容器变为无卡容器

请求方法：`PATCH`

请求接口：`/api/v1/containers/{name}/gpu`

Param： `knockGpu-1`

载荷：

~~~json
{
    "gpuCount": 0
}
~~~

响应：

~~~json
{
    "code": 200,
    "msg": "success",
    "data": {
        "id": "daa1f4e0a2198deadaa92ebc95dbae5ca8c13d8926d4fbf27ee8eedf34f69334",
        "name": "knockGpu-2"
    }
}
~~~

Docker inspect：

~~~
$ docker inspect knockGpu-2 | grep DeviceRequests
            "DeviceRequests": null,
~~~

### 更改容器的 Volume 配置

> 1. 变更的挂载信息必须为 Docker Volume 类型的卷，因为非 Docker Volume 类型的挂载，例如 NFS 目录挂载到容器内部，扩容/缩容、数据的销毁不是由 Docker 管理的。
>
> 2. 如果传递的 oldBind 和 newBind 相同，会直接跳过。
>
> 3. 这里的处理方式和`更改容器的 GPU 配置`不同，具体在`更改前后资源没有发生变化`这一情况。
>
>    更改 GPU 时，如果更改前后卡的数量一样，就跳过。
>
>    而对于 Volume 来说，判断`更改前后容量是否发生变化`，处理逻辑在 `更改 Volume 大小`的接口。
>
>    所以使用该接口时，传递的应该是扩容/缩容好的 Volume（或者一张新的 Volume，不过没测试过这种情况是否可用😢）。

请求方法：`PATCH`

请求接口：`/api/v1/containers/{name}/volume`

Param： `marital-0`

待测试容器的创建信息：

~~~json
{
    "imageName": "nvidia/cuda:10.0-base",
    "containerName": "marital",
    "gpuCount": 1,
    "binds": [
        {
            "src": "aerialVol-0",
            "dest": "/root/aerialVol"
        },
        {
            "src": "/mynfs/data/ctr-marital",
            "dest": "/root/data/ctr-marital"
        }
    ],
    "env": [
        "USER=foo"
    ],
    "Ports": [
        {
            "hostPort": 2333,
            "containerPort": 22
        }
    ]
}
~~~

Volume 的配置：

~~~json
$ docker volume inspect aerialVol-0 
[
    {
        "CreatedAt": "2023-12-27T02:39:36Z",
        "Driver": "local",
        "Labels": null,
        "Mountpoint": "/localData/docker/volumes/aerialVol-0/_data",
        "Name": "aerialVol-0",
        "Options": {
            "size": "20GB"
        },
        "Scope": "local"
    }
]
// 对 Volume 进行扩容后
$ docker volume inspect aerialVol-1
[
    {
        "CreatedAt": "2023-12-27T02:42:16Z",
        "Driver": "local",
        "Labels": null,
        "Mountpoint": "/localData/docker/volumes/aerialVol-1/_data",
        "Name": "aerialVol-1",
        "Options": {
            "size": "40GB"
        },
        "Scope": "local"
    }
]
~~~

载荷：

~~~json
{
    "type": "volume",
    "oldBind": {
        "src": "aerialVol-0",
        "dest": "/root/aerialVol"
    },
    "newBind": {
        "src": "aerialVol-1",
        "dest": "/root/aerialVol"
    }
}
~~~

响应：

~~~json
{
    "code": 200,
    "msg": "success",
    "data": {
        "id": "671eacb8514c92fa62e296785c1164b0a70f4c5fc28b525a210f870acef94e2b",
        "name": "marital-1"
    }
}
~~~

Docker inspect：

~~~json
$ docker inspect marital-1 | grep -A 3 Binds
            "Binds": [
                "aerialVol-1:/root/aerialVol",
                "/mynfs/data/ctr-marital:/root/data/ctr-marital"
            ],
~~~

### 停止容器

描述：如果容器是有卡容器，那么停止时会释放占用的 GPU 资源。

请求方法：`PATCH`

请求接口：`/api/v1/containers/{name}/stop`

Param： `sarcastic-0`

响应：

~~~json
{
    "code": 200,
    "msg": "success",
    "data": null
}
~~~

### 重启容器

描述：停止的无卡容器重启时，直接 docker restart。

停止的有卡容器重启时，会重新为它创建一个新容器，同时重新申请它之前使用的指定数量的 GPU（使用的卡号可能发生变化，例如之前使用0、 1、 2 号卡，新容器可能使用 3、 4、 5 号卡）。

⚠️**不用担心的是，它和之前的容器一模一样，只是看起来容器名称有些变化。**

请求方法：`PATCH`

请求接口：`/api/v1/containers/{name}/stop`

Param： `sarcastic-0`

响应：

~~~json
{
    "code": 200,
    "msg": "success",
    "data": {
        "id": "cc19da17f809b19e05f5baf85b873248d70de0903f390e85fb08cbc1cda29000",
        "name": "sarcastic-1"
    }
}
~~~

### 删除容器

描述：删除有卡容器时，会释放它所占用的 GPU 资源，如果指定了 delEtcdInfoAndVersionRecord 参数为 true，那么删除容器时也会删除掉 ETCD 和 VersionMap 中关于它的记录。

通过一个具体的例子来解释，比如当前有一个 foo-0 容器，它经过一次升级 GPU 配置，变成了 foo-1。

此时，要删除 foo-1，如果指定了 delEtcdInfoAndVersionRecord 为 true，那么 ETCD 中关于 foo 的描述会被删除，以为记录版本的 Map 中会移除 foo。就好像 foo-0、foo-1 从来没有来过。

当然 foo-0 还没有被删除，当你把 foo-0 删除时，你就可以再次用 foo 作为名字创建容器，新的描述会被添加到 ETCD，然后 {k: foo, v: 0} 会被添加到 VersionMap 中。

如果 delEtcdInfoAndVersionRecord为 false，我们删除了 foo-1，此时我们仍然可以在 foo-1 的基础上继续变更配置，生成一个 foo-2。这适用于释放资源。

所以，如果要单纯的释放资源，delEtcdInfoAndVersionRecord 应为 false。如果确定这个要抹除掉一个容器的历史版本，应为 true。

<!--TODO：其实可以只传入 foo，如果delEtcdInfoAndVersionRecord为 true，那么就删除 ETCD、VersionMap 中的数据，然后依次删除 foo-0、foo-1... foo-n。然后没有单独的删除容器，可能比较好。🤔-->

请求方法：`DELETE`

请求接口：`/api/v1/containers/{name}`

Param： `sarcastic-0`

载荷：

~~~json
{
    "force": true,
    "delEtcdInfoAndVersionRecord": true
}
~~~

响应：

~~~json
{
    "code": 200,
    "msg": "success",
    "data": null
}
~~~

### 提交容器为镜像

描述：镜像名称默认为容器 ID。

请求方法：`POST`

请求接口：`/api/v1/containers/{name}/commit`

Param： `advocate-0`

载荷：

~~~json
{
    "newImageName": "advocate-0-12-27"
}
~~~

响应：

~~~json
{
    "code": 200,
    "msg": "success",
    "data": {
        "container": "advocate-0",
        "imageName": "advocate-0-12-27"
    }
}
~~~

docker images：

~~~
$docker images | grep advocate-0-12-27
advocate-0-12-27	latest	1727f61e77ba   15 minutes ago   109MB
~~~

### 容器内执行命令

描述：相当于 docker exec，但是不能向在宿主机一样进入容器，只能将标准输出返回，当你传递一些命令给容器时。

请求方法：`POST`

请求接口：`/api/v1/containers/{name}/execute`

Param：`dilute-0`

载荷：

~~~json
{
    "cmd": ["nvidia-smi"]
}
~~~

响应：

~~~json
{
    "code": 200,
    "msg": "success",
    "data": {
        "stdout": "Wed Dec 27 09:11:22 2023       \n+-----------------------------------------------------------------------------+\n| NVIDIA-SMI 525.85.12    Driver Version: 525.85.12    CUDA Version: 12.0     |\n|-------------------------------+----------------------+----------------------+\n| GPU  Name        Persistence-M| Bus-Id        Disp.A | Volatile Uncorr. ECC |\n| Fan  Temp  Perf  Pwr:Usage/Cap|         Memory-Usage | GPU-Util  Compute M. |\n|                               |                      |               MIG M. |\n|===============================+======================+======================|\n|   0  NVIDIA A100 80G...  On   | 00000000:39:00.0 Off |                    0 |\n| N/A   39C    P0    47W / 300W |      0MiB / 81920MiB |      0%      Default |\n|                               |                      |             Disabled |\n+-------------------------------+----------------------+----------------------+\n                                                                               \n+-----------------------------------------------------------------------------+\n| Processes:                                                                  |\n|  GPU   GI   CI        PID   Type   Process name                  GPU Memory |\n|        ID   ID                                                   Usage      |\n|=============================================================================|\n|  No running processes found                                                 |\n+-----------------------------------------------------------------------------+\n"
    }
}
~~~



## GPU

### 查看 GPU 使用情况

描述：key：GPU UUID  value：占用情况，0 代表未被占用，1 代表已被占用

请求方法：`GET`

请求接口：`/api/v1/gpus`

响应：

~~~json
{
    "code": 200,
    "msg": "success",
    "data": {
        "gpuStatus": {
            "GPU-04adce59-e7fc-19ed-6800-bc09e5f8fa31": 0,
            "GPU-281d9730-5a26-7c56-12fb-3a3d5a24ab68": 0,
            "GPU-36009026-9470-a2e0-73d3-222a63b82e4e": 0,
            "GPU-7a42be89-64fe-5383-c7be-49d199a96b3d": 0,
            "GPU-82fbe07b-200b-1d4c-4fbe-b0b54db86be5": 0,
            "GPU-bc85a406-0357-185f-a56c-afb49572bdbe": 0,
            "GPU-c6b3ca5f-c1ac-8171-582b-737b70a6bbce": 0,
            "GPU-dc6d913c-8df4-a9a4-49e6-b82fcba5a6f9": 0
        }
    }
}
~~~
