## gpu-docker-api
调用 Docker Client 实现 GPU 容器的一些业务操作，例如：
* 无卡启动容器
* 升降 GPU 配置
* Docker Volume 扩缩容

以及容器的基本操作，例如：
* 容器启动、暂停、重启、删除
* 卷创建、删除

### 升降配置
比如启动一个 Gpu 容器，使用 `nvidia/cuda:10.0-base` 镜像，使用的是 0 号卡槽的 NVIDIA A100 80G GPU，现在想使用三张同类型的卡。
首先调用创建容器接口：`/api/v1/containers`，载荷：
~~~json
{
    "imageName": "nvidia/cuda:10.0-base",
    "containerName": "foo",
    "gpuCount": 1,
    "gpuNumbers": [],
    "cardless": false,
    "binds": []
}
~~~

然后调用升降 GPU 容器的接口：/api/v1/containers/${container_name}/gpu，载荷：
~~~json
{
    "gpuCount": 3
}
~~~

效果如下：

before：

![image-20231208173128300](https://bertram-li-bucket.oss-cn-beijing.aliyuncs.com/markdown-img/image-20231208173128300.png)

after：

![image-20231208173201102](https://bertram-li-bucket.oss-cn-beijing.aliyuncs.com/markdown-img/image-20231208173201102.png)
