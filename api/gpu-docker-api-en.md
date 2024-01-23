---
title: gpu-docker-api-en v1.0.0
language_tabs:
  - shell: Shell
  - http: HTTP
  - javascript: JavaScript
  - ruby: Ruby
  - python: Python
  - php: PHP
  - java: Java
  - go: Go
toc_footers: []
includes: []
search: true
code_clipboard: true
highlight_theme: darkula
headingLevel: 2
generator: "@tarslib/widdershins v4.0.17"

---

# gpu-docker-api-en

> v1.0.0

https://github.com/mayooot/gpu-docker-api

Base URLs:

* <a href="http://127.0.0.1:2378">Develop Env: http://127.0.0.1:2378</a>

# ReplicaSet

## POST Run a container via replicaSet

POST /api/v1/replicaSet

Run a container consists of two parts: create and start.

When the container is created, the creation information is saved to etcd.

> ReplicaSet is just an abstract concept, there is no concrete implementations,it just has to manage docker container, and save the container historical version information, that's all.

> Body Parameters

```json
{
  "imageName": "nvidia/cuda:10.0-base",
  "replicaSetName": "foo",
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
  "containerPorts": [
    "22",
    "443"
  ]
}
```

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|body|body|object| no |none|
|» imageName|body|string| yes |It's not automatically download from docker hub, you need to pull it locally first.|
|» replicaSetName|body|string| yes |Cannot cotainer '-', ReplicaSet managed containers will add version numbers, e.g. foo-0, foo-1.|
|» gpuCount|body|integer| yes |If gpuCount is 0, the gpu is not used.|
|» binds|body|[object]| yes |Bind mounts|
|»» src|body|string| yes |If starts with /, mounts the host's folders or files into the container.If it does not start with /, mount the docker volume into the container.|
|»» dest|body|string| yes |Cannot mount to root direcotry('/').|
|» env|body|[string]| yes |Environment variable, the format is like foo=bar.|
|» containerPorts|body|[string]| yes |The port number inside the container, with a randomly assigned host port number bound to it.|

> Response Examples

> OK

```json
{
  "code": 200,
  "msg": "Success",
  "data": {
    "name": "foo-1"
  }
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» code|integer|true|none||none|
|» msg|string|true|none||none|
|» data|object|true|none||none|
|»» name|string|true|none||Container name of the new version created|

## POST Commit container as an image via replicaSet

POST /api/v1/replicaSet/{name}/commit

Commit replicaSet the current version of the container as an image.

> Body Parameters

```json
{
  "newImageName": "foo-v1-image"
}
```

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|name|path|string| yes |ReplicaSet Name|
|body|body|object| no |none|
|» newImageName|body|string| yes |The name of the image you want|

> Response Examples

> OK

```json
{
  "code": 200,
  "msg": "Success",
  "data": {
    "imageName": "foo-v1-image"
  }
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» code|integer|true|none||none|
|» msg|string|true|none||none|
|» data|object|true|none||none|
|»» imageName|string|true|none||Saved Image Name|

## PATCH Rollback a container via replicaSet

PATCH /api/v1/replicaSet/{name}/rollback

Rollback replicaSet the current version of the container toa specific version.

You can get all history of ReplicaSet version, via '/api/v1/replicaSet/:name/history'

> Body Parameters

```json
{
  "version": 1
}
```

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|name|path|string| yes |ReplicaSet Name|
|body|body|object| no |none|
|» version|body|integer| yes |Specific Version|

> Response Examples

> OK

```json
{
  "code": 200,
  "msg": "Success",
  "data": {
    "containerName": "foo-6"
  }
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» code|integer|true|none||none|
|» msg|string|true|none||none|
|» data|object|true|none||none|
|»» containerName|string|true|none||Container name of the new version created|

## PATCH Patch a container via replicaSet

PATCH /api/v1/replicaSet/{name}

Patch the configuration of the latest version of existing container via create a new container and copy the old container system data to the new container.

Including changing the number of gpu and updating the volume binding.

If you request body is empty(e.g. {}), it will recreate a container based on the existing configuration.

> The old version container will be deleted.
> For example, if foo-1 calls the `/api/v1/replicaSet/:name` and create foo-2, foo-1 will be deleted.

> Body Parameters

```json
{
  "gpuPatch": {
    "gpuCount": 1
  },
  "volumePatch": {
    "oldBind": {
      "src": "veil-0",
      "dest": "/root/veil-0"
    },
    "newBind": {
      "src": "veil-1",
      "dest": "/root/veil-1"
    }
  }
}
```

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|name|path|string| yes |none|
|body|body|object| no |none|
|» gpuPatch|body|object| yes |none|
|»» gpuCount|body|integer| yes |To adjust the number of gpus, it is so simple.|
|» volumePatch|body|object| yes |First find the mount information by matching oldBind, then change it.|
|»» oldBind|body|object| yes |The old binding information, which you can get via '/api/v1/replicaSet/:name'.|
|»»» src|body|string| yes |none|
|»»» dest|body|string| yes |none|
|»» newBind|body|object| yes |If it is a docker volume, make sure it already exists.|
|»»» src|body|string| yes |none|
|»»» dest|body|string| yes |none|

#### Description

**»» gpuCount**: To adjust the number of gpus, it is so simple.
If you don't want to use the gpu, set it to 0.

> Response Examples

> OK

```json
{
  "code": 200,
  "msg": "Success",
  "data": {
    "containerName": "foo-2"
  }
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» code|integer|true|none||none|
|» msg|string|true|none||none|
|» data|object|true|none||none|
|»» containerName|string|true|none||none|

## DELETE Delete a container via replicaSet

DELETE /api/v1/replicaSet/{name}

Delete a replicaSet also delete the container and cannot be recovered.

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|name|path|string| yes |ReplicaSet Name|

> Response Examples

> OK

```json
{
  "code": 200,
  "msg": "Success",
  "data": null
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» code|integer|true|none||none|
|» msg|string|true|none||none|
|» data|null|true|none||none|

## GET  Get version info about replicaSet

GET /api/v1/replicaSet/{name}

Get information about the current version of the replicaset, which you can use to see the port number that the container exposes to the host or other information when the cotainer was created.

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|name|path|string| yes |ReplicaSet Name|

> Response Examples

> OK

```json
{
  "code": 200,
  "msg": "Success",
  "data": {
    "info": {
      "version": 8,
      "createTime": "2024-01-22 07:36:04",
      "config": {
        "Hostname": "",
        "Domainname": "",
        "User": "",
        "AttachStdin": false,
        "AttachStdout": false,
        "AttachStderr": false,
        "ExposedPorts": {
          "22/tcp": {},
          "443/tcp": {}
        },
        "Tty": true,
        "OpenStdin": true,
        "StdinOnce": false,
        "Env": [
          "USER=foo",
          "CONTAINER_VERSION=8"
        ],
        "Cmd": null,
        "Image": "nvidia/cuda:10.0-base",
        "Volumes": null,
        "WorkingDir": "",
        "Entrypoint": null,
        "OnBuild": null,
        "Labels": null
      },
      "hostConfig": {
        "Binds": [
          "veil-0:/root/veil-0",
          "/mynfs/data/ctr-knock:/root/data/ctr-knock"
        ],
        "ContainerIDFile": "",
        "LogConfig": {
          "Type": "",
          "Config": null
        },
        "NetworkMode": "",
        "PortBindings": {
          "22/tcp": [
            {
              "HostIp": "",
              "HostPort": "40000"
            }
          ],
          "443/tcp": [
            {
              "HostIp": "",
              "HostPort": "40001"
            }
          ]
        },
        "RestartPolicy": {
          "Name": "",
          "MaximumRetryCount": 0
        },
        "AutoRemove": false,
        "VolumeDriver": "",
        "VolumesFrom": null,
        "ConsoleSize": [
          0,
          0
        ],
        "CapAdd": null,
        "CapDrop": null,
        "CgroupnsMode": "",
        "Dns": null,
        "DnsOptions": null,
        "DnsSearch": null,
        "ExtraHosts": null,
        "GroupAdd": null,
        "IpcMode": "",
        "Cgroup": "",
        "Links": null,
        "OomScoreAdj": 0,
        "PidMode": "",
        "Privileged": false,
        "PublishAllPorts": false,
        "ReadonlyRootfs": false,
        "SecurityOpt": null,
        "UTSMode": "",
        "UsernsMode": "",
        "ShmSize": 0,
        "Isolation": "",
        "CpuShares": 0,
        "Memory": 0,
        "NanoCpus": 0,
        "CgroupParent": "",
        "BlkioWeight": 0,
        "BlkioWeightDevice": null,
        "BlkioDeviceReadBps": null,
        "BlkioDeviceWriteBps": null,
        "BlkioDeviceReadIOps": null,
        "BlkioDeviceWriteIOps": null,
        "CpuPeriod": 0,
        "CpuQuota": 0,
        "CpuRealtimePeriod": 0,
        "CpuRealtimeRuntime": 0,
        "CpusetCpus": "",
        "CpusetMems": "",
        "Devices": null,
        "DeviceCgroupRules": null,
        "DeviceRequests": [
          {
            "Driver": "nvidia",
            "Count": 0,
            "DeviceIDs": [
              "GPU-bc85a406-0357-185f-a56c-afb49572bdbe",
              "GPU-c6b3ca5f-c1ac-8171-582b-737b70a6bbce",
              "GPU-dc6d913c-8df4-a9a4-49e6-b82fcba5a6f9"
            ],
            "Capabilities": [
              [
                "gpu"
              ]
            ],
            "Options": null
          }
        ],
        "MemoryReservation": 0,
        "MemorySwap": 0,
        "MemorySwappiness": null,
        "OomKillDisable": null,
        "PidsLimit": null,
        "Ulimits": null,
        "CpuCount": 0,
        "CpuPercent": 0,
        "IOMaximumIOps": 0,
        "IOMaximumBandwidth": 0,
        "MaskedPaths": null,
        "ReadonlyPaths": null
      },
      "networkingConfig": {
        "EndpointsConfig": null
      },
      "platform": {
        "architecture": "",
        "os": ""
      },
      "containerName": "foo-8"
    }
  }
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» code|integer|true|none||none|
|» msg|string|true|none||none|
|» data|object|true|none||none|
|»» info|object|true|none||Container Creation Information|
|»»» version|integer|true|none||none|
|»»» createTime|string|true|none||none|
|»»» config|object|true|none||none|
|»»»» Hostname|string|true|none||none|
|»»»» Domainname|string|true|none||none|
|»»»» User|string|true|none||none|
|»»»» AttachStdin|boolean|true|none||none|
|»»»» AttachStdout|boolean|true|none||none|
|»»»» AttachStderr|boolean|true|none||none|
|»»»» ExposedPorts|object|true|none||none|
|»»»»» 22|object|false|none||none|
|»»»»» 443|object|false|none||none|
|»»»» Tty|boolean|true|none||none|
|»»»» OpenStdin|boolean|true|none||none|
|»»»» StdinOnce|boolean|true|none||none|
|»»»» Env|[string]|true|none||none|
|»»»» Cmd|null|true|none||none|
|»»»» Image|string|true|none||none|
|»»»» Volumes|null|true|none||none|
|»»»» WorkingDir|string|true|none||none|
|»»»» Entrypoint|null|true|none||none|
|»»»» OnBuild|null|true|none||none|
|»»»» Labels|null|true|none||none|
|»»» hostConfig|object|true|none||none|
|»»»» Binds|[string]|true|none||none|
|»»»» ContainerIDFile|string|true|none||none|
|»»»» LogConfig|object|true|none||none|
|»»»»» Type|string|true|none||none|
|»»»»» Config|null|true|none||none|
|»»»» NetworkMode|string|true|none||none|
|»»»» PortBindings|object|true|none||none|
|»»»»» 22|[object]|false|none||none|
|»»»»»» HostIp|string|false|none||none|
|»»»»»» HostPort|string|false|none||none|
|»»»»» 443|[object]|false|none||none|
|»»»»»» HostIp|string|false|none||none|
|»»»»»» HostPort|string|false|none||none|
|»»»» RestartPolicy|object|true|none||none|
|»»»»» Name|string|true|none||none|
|»»»»» MaximumRetryCount|integer|true|none||none|
|»»»» AutoRemove|boolean|true|none||none|
|»»»» VolumeDriver|string|true|none||none|
|»»»» VolumesFrom|null|true|none||none|
|»»»» ConsoleSize|[integer]|true|none||none|
|»»»» CapAdd|null|true|none||none|
|»»»» CapDrop|null|true|none||none|
|»»»» CgroupnsMode|string|true|none||none|
|»»»» Dns|null|true|none||none|
|»»»» DnsOptions|null|true|none||none|
|»»»» DnsSearch|null|true|none||none|
|»»»» ExtraHosts|null|true|none||none|
|»»»» GroupAdd|null|true|none||none|
|»»»» IpcMode|string|true|none||none|
|»»»» Cgroup|string|true|none||none|
|»»»» Links|null|true|none||none|
|»»»» OomScoreAdj|integer|true|none||none|
|»»»» PidMode|string|true|none||none|
|»»»» Privileged|boolean|true|none||none|
|»»»» PublishAllPorts|boolean|true|none||none|
|»»»» ReadonlyRootfs|boolean|true|none||none|
|»»»» SecurityOpt|null|true|none||none|
|»»»» UTSMode|string|true|none||none|
|»»»» UsernsMode|string|true|none||none|
|»»»» ShmSize|integer|true|none||none|
|»»»» Isolation|string|true|none||none|
|»»»» CpuShares|integer|true|none||none|
|»»»» Memory|integer|true|none||none|
|»»»» NanoCpus|integer|true|none||none|
|»»»» CgroupParent|string|true|none||none|
|»»»» BlkioWeight|integer|true|none||none|
|»»»» BlkioWeightDevice|null|true|none||none|
|»»»» BlkioDeviceReadBps|null|true|none||none|
|»»»» BlkioDeviceWriteBps|null|true|none||none|
|»»»» BlkioDeviceReadIOps|null|true|none||none|
|»»»» BlkioDeviceWriteIOps|null|true|none||none|
|»»»» CpuPeriod|integer|true|none||none|
|»»»» CpuQuota|integer|true|none||none|
|»»»» CpuRealtimePeriod|integer|true|none||none|
|»»»» CpuRealtimeRuntime|integer|true|none||none|
|»»»» CpusetCpus|string|true|none||none|
|»»»» CpusetMems|string|true|none||none|
|»»»» Devices|null|true|none||none|
|»»»» DeviceCgroupRules|null|true|none||none|
|»»»» DeviceRequests|[object]|true|none||none|
|»»»»» Driver|string|false|none||none|
|»»»»» Count|integer|false|none||none|
|»»»»» DeviceIDs|[string]|false|none||none|
|»»»»» Capabilities|[array]|false|none||none|
|»»»»» Options|null|false|none||none|
|»»»» MemoryReservation|integer|true|none||none|
|»»»» MemorySwap|integer|true|none||none|
|»»»» MemorySwappiness|null|true|none||none|
|»»»» OomKillDisable|null|true|none||none|
|»»»» PidsLimit|null|true|none||none|
|»»»» Ulimits|null|true|none||none|
|»»»» CpuCount|integer|true|none||none|
|»»»» CpuPercent|integer|true|none||none|
|»»»» IOMaximumIOps|integer|true|none||none|
|»»»» IOMaximumBandwidth|integer|true|none||none|
|»»»» MaskedPaths|null|true|none||none|
|»»»» ReadonlyPaths|null|true|none||none|
|»»» networkingConfig|object|true|none||none|
|»»»» EndpointsConfig|null|true|none||none|
|»»» platform|object|true|none||none|
|»»»» architecture|string|true|none||none|
|»»»» os|string|true|none||none|
|»»» containerName|string|true|none||none|

## PATCH Stop a container via replicaSet

PATCH /api/v1/replicaSet/{name}/stop

Stop the current version of the replicaSet container, gpu and port will be released.

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|name|path|string| yes |ReplicaSet Name|

> Response Examples

> OK

```json
{
  "code": 200,
  "msg": "Success",
  "data": null
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» code|integer|true|none||none|
|» msg|string|true|none||none|
|» data|null|true|none||none|

## PATCH Pause a replicaSet via replicaSet

PATCH /api/v1/replicaSet/{name}/pause

Pause the current version of the replicaSet container, gpu and port will not be release.

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|name|path|string| yes |ReplicaSet Name|

> Response Examples

> OK

```json
{
  "code": 200,
  "msg": "Success",
  "data": null
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» code|integer|true|none||none|
|» msg|string|true|none||none|
|» data|null|true|none||none|

## PATCH Continue a replicaSet via replicaSet

PATCH /api/v1/replicaSet/{name}/continue

Continue to run the current version of the replicaSet container, it will call `docker restart`.

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|name|path|string| yes |ReplicaSet Name|

> Response Examples

> OK

```json
{
  "code": 200,
  "msg": "Success",
  "data": null
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» code|integer|true|none||none|
|» msg|string|true|none||none|
|» data|null|true|none||none|

## PATCH Restart a container via replicaSet

PATCH /api/v1/replicaSet/{name}/restart

Restart the current version of the replicaSet container by recreate a container, it will reapply gpu and port.

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|name|path|string| yes |ReplicaSet Name|

> Response Examples

> OK

```json
{
  "code": 200,
  "msg": "Success",
  "data": {
    "containerName": "foo-7"
  }
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» code|integer|true|none||none|
|» msg|string|true|none||none|
|» data|object|true|none||none|
|»» containerName|string|true|none||Container name of the new version created|

## POST Execute a command in the container via replicaSet

POST /api/v1/replicaSet/{name}/execute

Execute a command in the replicaSet current version of the container.

> Body Parameters

```json
{
  "cmd": [
    "nvidia-smi"
  ]
}
```

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|name|path|string| yes |ReplicaSet Name|
|body|body|object| no |none|
|» cmd|body|[string]| yes |One Command|

> Response Examples

> OK

```json
{
  "code": 200,
  "msg": "Success",
  "data": {
    "stdout": "Mon Jan 22 03:01:02 2024       \n+-----------------------------------------------------------------------------+\n| NVIDIA-SMI 525.85.12    Driver Version: 525.85.12    CUDA Version: 12.0     |\n|-------------------------------+----------------------+----------------------+\n| GPU  Name        Persistence-M| Bus-Id        Disp.A | Volatile Uncorr. ECC |\n| Fan  Temp  Perf  Pwr:Usage/Cap|         Memory-Usage | GPU-Util  Compute M. |\n|                               |                      |               MIG M. |\n|===============================+======================+======================|\n|   0  NVIDIA A100 80G...  On   | 00000000:36:00.0 Off |                    0 |\n| N/A   42C    P0    65W / 300W |  51827MiB / 81920MiB |      0%      Default |\n|                               |                      |             Enabled* |\n+-------------------------------+----------------------+----------------------+\n|   1  NVIDIA A100 80G...  On   | 00000000:89:00.0 Off |                    0 |\n| N/A   47C    P0    70W / 300W |  12481MiB / 81920MiB |      0%      Default |\n|                               |                      |             Enabled* |\n+-------------------------------+----------------------+----------------------+\n|   2  NVIDIA A100 80G...  On   | 00000000:8A:00.0 Off |                    0 |\n| N/A   39C    P0    45W / 300W |      0MiB / 81920MiB |      0%      Default |\n|                               |                      |             Enabled* |\n+-------------------------------+----------------------+----------------------+\n                                                                               \n+-----------------------------------------------------------------------------+\n| Processes:                                                                  |\n|  GPU   GI   CI        PID   Type   Process name                  GPU Memory |\n|        ID   ID                                                   Usage      |\n|=============================================================================|\n+-----------------------------------------------------------------------------+\n"
  }
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» code|integer|true|none||none|
|» msg|string|true|none||none|
|» data|object|true|none||none|
|»» stdout|string|true|none||Response|

## GET Get all version info about replicaSet

GET /api/v1/replicaSet/{name}/history

Get information about all historical versions of the replicaSet.

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|name|path|string| yes |ReplicaSet Name|

> Response Examples

> OK

```json
{
  "code": 200,
  "msg": "Success",
  "data": {
    "history": [
      {
        "version": 8,
        "createTime": "2024-01-22 07:36:04",
        "status": {
          "version": 8,
          "createTime": "2024-01-22 07:36:04",
          "config": {
            "Hostname": "",
            "Domainname": "",
            "User": "",
            "AttachStdin": false,
            "AttachStdout": false,
            "AttachStderr": false,
            "ExposedPorts": {
              "22/tcp": {},
              "443/tcp": {}
            },
            "Tty": true,
            "OpenStdin": true,
            "StdinOnce": false,
            "Env": [
              "USER=foo",
              "CONTAINER_VERSION=8"
            ],
            "Cmd": null,
            "Image": "nvidia/cuda:10.0-base",
            "Volumes": null,
            "WorkingDir": "",
            "Entrypoint": null,
            "OnBuild": null,
            "Labels": null
          },
          "hostConfig": {
            "Binds": [
              "veil-0:/root/veil-0",
              "/mynfs/data/ctr-knock:/root/data/ctr-knock"
            ],
            "ContainerIDFile": "",
            "LogConfig": {
              "Type": "",
              "Config": null
            },
            "NetworkMode": "",
            "PortBindings": {
              "22/tcp": [
                {
                  "HostIp": "",
                  "HostPort": "40000"
                }
              ],
              "443/tcp": [
                {
                  "HostIp": "",
                  "HostPort": "40001"
                }
              ]
            },
            "RestartPolicy": {
              "Name": "",
              "MaximumRetryCount": 0
            },
            "AutoRemove": false,
            "VolumeDriver": "",
            "VolumesFrom": null,
            "ConsoleSize": [
              0,
              0
            ],
            "CapAdd": null,
            "CapDrop": null,
            "CgroupnsMode": "",
            "Dns": null,
            "DnsOptions": null,
            "DnsSearch": null,
            "ExtraHosts": null,
            "GroupAdd": null,
            "IpcMode": "",
            "Cgroup": "",
            "Links": null,
            "OomScoreAdj": 0,
            "PidMode": "",
            "Privileged": false,
            "PublishAllPorts": false,
            "ReadonlyRootfs": false,
            "SecurityOpt": null,
            "UTSMode": "",
            "UsernsMode": "",
            "ShmSize": 0,
            "Isolation": "",
            "CpuShares": 0,
            "Memory": 0,
            "NanoCpus": 0,
            "CgroupParent": "",
            "BlkioWeight": 0,
            "BlkioWeightDevice": null,
            "BlkioDeviceReadBps": null,
            "BlkioDeviceWriteBps": null,
            "BlkioDeviceReadIOps": null,
            "BlkioDeviceWriteIOps": null,
            "CpuPeriod": 0,
            "CpuQuota": 0,
            "CpuRealtimePeriod": 0,
            "CpuRealtimeRuntime": 0,
            "CpusetCpus": "",
            "CpusetMems": "",
            "Devices": null,
            "DeviceCgroupRules": null,
            "DeviceRequests": [
              {
                "Driver": "nvidia",
                "Count": 0,
                "DeviceIDs": [
                  "GPU-bc85a406-0357-185f-a56c-afb49572bdbe",
                  "GPU-c6b3ca5f-c1ac-8171-582b-737b70a6bbce",
                  "GPU-dc6d913c-8df4-a9a4-49e6-b82fcba5a6f9"
                ],
                "Capabilities": [
                  [
                    "gpu"
                  ]
                ],
                "Options": null
              }
            ],
            "MemoryReservation": 0,
            "MemorySwap": 0,
            "MemorySwappiness": null,
            "OomKillDisable": null,
            "PidsLimit": null,
            "Ulimits": null,
            "CpuCount": 0,
            "CpuPercent": 0,
            "IOMaximumIOps": 0,
            "IOMaximumBandwidth": 0,
            "MaskedPaths": null,
            "ReadonlyPaths": null
          },
          "networkingConfig": {
            "EndpointsConfig": null
          },
          "platform": {
            "architecture": "",
            "os": ""
          },
          "containerName": "foo-8"
        }
      },
      {
        "version": 7,
        "createTime": "2024-01-22 06:45:11",
        "status": {
          "version": 7,
          "createTime": "2024-01-22 06:45:11",
          "config": {
            "Hostname": "",
            "Domainname": "",
            "User": "",
            "AttachStdin": false,
            "AttachStdout": false,
            "AttachStderr": false,
            "ExposedPorts": {
              "22/tcp": {},
              "443/tcp": {}
            },
            "Tty": true,
            "OpenStdin": true,
            "StdinOnce": false,
            "Env": [
              "USER=foo",
              "CONTAINER_VERSION=7"
            ],
            "Cmd": null,
            "Image": "nvidia/cuda:10.0-base",
            "Volumes": null,
            "WorkingDir": "",
            "Entrypoint": null,
            "OnBuild": null,
            "Labels": null
          },
          "hostConfig": {
            "Binds": [
              "veil-0:/root/veil-0",
              "/mynfs/data/ctr-knock:/root/data/ctr-knock"
            ],
            "ContainerIDFile": "",
            "LogConfig": {
              "Type": "",
              "Config": null
            },
            "NetworkMode": "",
            "PortBindings": {
              "22/tcp": [
                {
                  "HostIp": "",
                  "HostPort": "40000"
                }
              ],
              "443/tcp": [
                {
                  "HostIp": "",
                  "HostPort": "40001"
                }
              ]
            },
            "RestartPolicy": {
              "Name": "",
              "MaximumRetryCount": 0
            },
            "AutoRemove": false,
            "VolumeDriver": "",
            "VolumesFrom": null,
            "ConsoleSize": [
              0,
              0
            ],
            "CapAdd": null,
            "CapDrop": null,
            "CgroupnsMode": "",
            "Dns": null,
            "DnsOptions": null,
            "DnsSearch": null,
            "ExtraHosts": null,
            "GroupAdd": null,
            "IpcMode": "",
            "Cgroup": "",
            "Links": null,
            "OomScoreAdj": 0,
            "PidMode": "",
            "Privileged": false,
            "PublishAllPorts": false,
            "ReadonlyRootfs": false,
            "SecurityOpt": null,
            "UTSMode": "",
            "UsernsMode": "",
            "ShmSize": 0,
            "Isolation": "",
            "CpuShares": 0,
            "Memory": 0,
            "NanoCpus": 0,
            "CgroupParent": "",
            "BlkioWeight": 0,
            "BlkioWeightDevice": null,
            "BlkioDeviceReadBps": null,
            "BlkioDeviceWriteBps": null,
            "BlkioDeviceReadIOps": null,
            "BlkioDeviceWriteIOps": null,
            "CpuPeriod": 0,
            "CpuQuota": 0,
            "CpuRealtimePeriod": 0,
            "CpuRealtimeRuntime": 0,
            "CpusetCpus": "",
            "CpusetMems": "",
            "Devices": null,
            "DeviceCgroupRules": null,
            "DeviceRequests": [
              {
                "Driver": "nvidia",
                "Count": 0,
                "DeviceIDs": [
                  "GPU-bc85a406-0357-185f-a56c-afb49572bdbe",
                  "GPU-c6b3ca5f-c1ac-8171-582b-737b70a6bbce",
                  "GPU-dc6d913c-8df4-a9a4-49e6-b82fcba5a6f9"
                ],
                "Capabilities": [
                  [
                    "gpu"
                  ]
                ],
                "Options": null
              }
            ],
            "MemoryReservation": 0,
            "MemorySwap": 0,
            "MemorySwappiness": null,
            "OomKillDisable": null,
            "PidsLimit": null,
            "Ulimits": null,
            "CpuCount": 0,
            "CpuPercent": 0,
            "IOMaximumIOps": 0,
            "IOMaximumBandwidth": 0,
            "MaskedPaths": null,
            "ReadonlyPaths": null
          },
          "networkingConfig": {
            "EndpointsConfig": null
          },
          "platform": {
            "architecture": "",
            "os": ""
          },
          "containerName": "foo-7"
        }
      },
      {
        "version": 6,
        "createTime": "2024-01-22 06:37:05",
        "status": {
          "version": 6,
          "createTime": "2024-01-22 06:37:05",
          "config": {
            "Hostname": "",
            "Domainname": "",
            "User": "",
            "AttachStdin": false,
            "AttachStdout": false,
            "AttachStderr": false,
            "ExposedPorts": {
              "22/tcp": {},
              "443/tcp": {}
            },
            "Tty": true,
            "OpenStdin": true,
            "StdinOnce": false,
            "Env": [
              "USER=foo",
              "CONTAINER_VERSION=6"
            ],
            "Cmd": null,
            "Image": "nvidia/cuda:10.0-base",
            "Volumes": null,
            "WorkingDir": "",
            "Entrypoint": null,
            "OnBuild": null,
            "Labels": null
          },
          "hostConfig": {
            "Binds": [
              "veil-0:/root/veil-0",
              "/mynfs/data/ctr-knock:/root/data/ctr-knock"
            ],
            "ContainerIDFile": "",
            "LogConfig": {
              "Type": "",
              "Config": null
            },
            "NetworkMode": "",
            "PortBindings": {
              "22/tcp": [
                {
                  "HostIp": "",
                  "HostPort": "40004"
                }
              ],
              "443/tcp": [
                {
                  "HostIp": "",
                  "HostPort": "40005"
                }
              ]
            },
            "RestartPolicy": {
              "Name": "",
              "MaximumRetryCount": 0
            },
            "AutoRemove": false,
            "VolumeDriver": "",
            "VolumesFrom": null,
            "ConsoleSize": [
              0,
              0
            ],
            "CapAdd": null,
            "CapDrop": null,
            "CgroupnsMode": "",
            "Dns": null,
            "DnsOptions": null,
            "DnsSearch": null,
            "ExtraHosts": null,
            "GroupAdd": null,
            "IpcMode": "",
            "Cgroup": "",
            "Links": null,
            "OomScoreAdj": 0,
            "PidMode": "",
            "Privileged": false,
            "PublishAllPorts": false,
            "ReadonlyRootfs": false,
            "SecurityOpt": null,
            "UTSMode": "",
            "UsernsMode": "",
            "ShmSize": 0,
            "Isolation": "",
            "CpuShares": 0,
            "Memory": 0,
            "NanoCpus": 0,
            "CgroupParent": "",
            "BlkioWeight": 0,
            "BlkioWeightDevice": null,
            "BlkioDeviceReadBps": null,
            "BlkioDeviceWriteBps": null,
            "BlkioDeviceReadIOps": null,
            "BlkioDeviceWriteIOps": null,
            "CpuPeriod": 0,
            "CpuQuota": 0,
            "CpuRealtimePeriod": 0,
            "CpuRealtimeRuntime": 0,
            "CpusetCpus": "",
            "CpusetMems": "",
            "Devices": null,
            "DeviceCgroupRules": null,
            "DeviceRequests": [
              {
                "Driver": "nvidia",
                "Count": 0,
                "DeviceIDs": [
                  "GPU-bc85a406-0357-185f-a56c-afb49572bdbe",
                  "GPU-c6b3ca5f-c1ac-8171-582b-737b70a6bbce",
                  "GPU-dc6d913c-8df4-a9a4-49e6-b82fcba5a6f9"
                ],
                "Capabilities": [
                  [
                    "gpu"
                  ]
                ],
                "Options": null
              }
            ],
            "MemoryReservation": 0,
            "MemorySwap": 0,
            "MemorySwappiness": null,
            "OomKillDisable": null,
            "PidsLimit": null,
            "Ulimits": null,
            "CpuCount": 0,
            "CpuPercent": 0,
            "IOMaximumIOps": 0,
            "IOMaximumBandwidth": 0,
            "MaskedPaths": null,
            "ReadonlyPaths": null
          },
          "networkingConfig": {
            "EndpointsConfig": null
          },
          "platform": {
            "architecture": "",
            "os": ""
          },
          "containerName": "foo-6"
        }
      },
      {
        "version": 5,
        "createTime": "2024-01-22 06:36:32",
        "status": {
          "version": 5,
          "createTime": "2024-01-22 06:36:32",
          "config": {
            "Hostname": "",
            "Domainname": "",
            "User": "",
            "AttachStdin": false,
            "AttachStdout": false,
            "AttachStderr": false,
            "ExposedPorts": {
              "22/tcp": {},
              "443/tcp": {}
            },
            "Tty": true,
            "OpenStdin": true,
            "StdinOnce": false,
            "Env": [
              "USER=foo",
              "CONTAINER_VERSION=5"
            ],
            "Cmd": null,
            "Image": "nvidia/cuda:10.0-base",
            "Volumes": null,
            "WorkingDir": "",
            "Entrypoint": null,
            "OnBuild": null,
            "Labels": null
          },
          "hostConfig": {
            "Binds": [
              "veil-1:/root/veil-1",
              "/mynfs/data/ctr-knock:/root/data/ctr-knock"
            ],
            "ContainerIDFile": "",
            "LogConfig": {
              "Type": "",
              "Config": null
            },
            "NetworkMode": "",
            "PortBindings": {
              "22/tcp": [
                {
                  "HostIp": "",
                  "HostPort": "40000"
                }
              ],
              "443/tcp": [
                {
                  "HostIp": "",
                  "HostPort": "40001"
                }
              ]
            },
            "RestartPolicy": {
              "Name": "",
              "MaximumRetryCount": 0
            },
            "AutoRemove": false,
            "VolumeDriver": "",
            "VolumesFrom": null,
            "ConsoleSize": [
              0,
              0
            ],
            "CapAdd": null,
            "CapDrop": null,
            "CgroupnsMode": "",
            "Dns": null,
            "DnsOptions": null,
            "DnsSearch": null,
            "ExtraHosts": null,
            "GroupAdd": null,
            "IpcMode": "",
            "Cgroup": "",
            "Links": null,
            "OomScoreAdj": 0,
            "PidMode": "",
            "Privileged": false,
            "PublishAllPorts": false,
            "ReadonlyRootfs": false,
            "SecurityOpt": null,
            "UTSMode": "",
            "UsernsMode": "",
            "ShmSize": 0,
            "Isolation": "",
            "CpuShares": 0,
            "Memory": 0,
            "NanoCpus": 0,
            "CgroupParent": "",
            "BlkioWeight": 0,
            "BlkioWeightDevice": null,
            "BlkioDeviceReadBps": null,
            "BlkioDeviceWriteBps": null,
            "BlkioDeviceReadIOps": null,
            "BlkioDeviceWriteIOps": null,
            "CpuPeriod": 0,
            "CpuQuota": 0,
            "CpuRealtimePeriod": 0,
            "CpuRealtimeRuntime": 0,
            "CpusetCpus": "",
            "CpusetMems": "",
            "Devices": null,
            "DeviceCgroupRules": null,
            "DeviceRequests": [
              {
                "Driver": "nvidia",
                "Count": 0,
                "DeviceIDs": [
                  "GPU-bc85a406-0357-185f-a56c-afb49572bdbe",
                  "GPU-c6b3ca5f-c1ac-8171-582b-737b70a6bbce",
                  "GPU-dc6d913c-8df4-a9a4-49e6-b82fcba5a6f9"
                ],
                "Capabilities": [
                  [
                    "gpu"
                  ]
                ],
                "Options": null
              }
            ],
            "MemoryReservation": 0,
            "MemorySwap": 0,
            "MemorySwappiness": null,
            "OomKillDisable": null,
            "PidsLimit": null,
            "Ulimits": null,
            "CpuCount": 0,
            "CpuPercent": 0,
            "IOMaximumIOps": 0,
            "IOMaximumBandwidth": 0,
            "MaskedPaths": null,
            "ReadonlyPaths": null
          },
          "networkingConfig": {
            "EndpointsConfig": null
          },
          "platform": {
            "architecture": "",
            "os": ""
          },
          "containerName": "foo-5"
        }
      },
      {
        "version": 4,
        "createTime": "2024-01-22 06:34:20",
        "status": {
          "version": 4,
          "createTime": "2024-01-22 06:34:20",
          "config": {
            "Hostname": "",
            "Domainname": "",
            "User": "",
            "AttachStdin": false,
            "AttachStdout": false,
            "AttachStderr": false,
            "ExposedPorts": {
              "22/tcp": {},
              "443/tcp": {}
            },
            "Tty": true,
            "OpenStdin": true,
            "StdinOnce": false,
            "Env": [
              "USER=foo",
              "CONTAINER_VERSION=4"
            ],
            "Cmd": null,
            "Image": "nvidia/cuda:10.0-base",
            "Volumes": null,
            "WorkingDir": "",
            "Entrypoint": null,
            "OnBuild": null,
            "Labels": null
          },
          "hostConfig": {
            "Binds": [
              "veil-0:/root/veil-0",
              "/mynfs/data/ctr-knock:/root/data/ctr-knock"
            ],
            "ContainerIDFile": "",
            "LogConfig": {
              "Type": "",
              "Config": null
            },
            "NetworkMode": "",
            "PortBindings": {
              "22/tcp": [
                {
                  "HostIp": "",
                  "HostPort": "40004"
                }
              ],
              "443/tcp": [
                {
                  "HostIp": "",
                  "HostPort": "40005"
                }
              ]
            },
            "RestartPolicy": {
              "Name": "",
              "MaximumRetryCount": 0
            },
            "AutoRemove": false,
            "VolumeDriver": "",
            "VolumesFrom": null,
            "ConsoleSize": [
              0,
              0
            ],
            "CapAdd": null,
            "CapDrop": null,
            "CgroupnsMode": "",
            "Dns": null,
            "DnsOptions": null,
            "DnsSearch": null,
            "ExtraHosts": null,
            "GroupAdd": null,
            "IpcMode": "",
            "Cgroup": "",
            "Links": null,
            "OomScoreAdj": 0,
            "PidMode": "",
            "Privileged": false,
            "PublishAllPorts": false,
            "ReadonlyRootfs": false,
            "SecurityOpt": null,
            "UTSMode": "",
            "UsernsMode": "",
            "ShmSize": 0,
            "Isolation": "",
            "CpuShares": 0,
            "Memory": 0,
            "NanoCpus": 0,
            "CgroupParent": "",
            "BlkioWeight": 0,
            "BlkioWeightDevice": null,
            "BlkioDeviceReadBps": null,
            "BlkioDeviceWriteBps": null,
            "BlkioDeviceReadIOps": null,
            "BlkioDeviceWriteIOps": null,
            "CpuPeriod": 0,
            "CpuQuota": 0,
            "CpuRealtimePeriod": 0,
            "CpuRealtimeRuntime": 0,
            "CpusetCpus": "",
            "CpusetMems": "",
            "Devices": null,
            "DeviceCgroupRules": null,
            "DeviceRequests": [
              {
                "Driver": "nvidia",
                "Count": 0,
                "DeviceIDs": [
                  "GPU-bc85a406-0357-185f-a56c-afb49572bdbe",
                  "GPU-c6b3ca5f-c1ac-8171-582b-737b70a6bbce",
                  "GPU-dc6d913c-8df4-a9a4-49e6-b82fcba5a6f9"
                ],
                "Capabilities": [
                  [
                    "gpu"
                  ]
                ],
                "Options": null
              }
            ],
            "MemoryReservation": 0,
            "MemorySwap": 0,
            "MemorySwappiness": null,
            "OomKillDisable": null,
            "PidsLimit": null,
            "Ulimits": null,
            "CpuCount": 0,
            "CpuPercent": 0,
            "IOMaximumIOps": 0,
            "IOMaximumBandwidth": 0,
            "MaskedPaths": null,
            "ReadonlyPaths": null
          },
          "networkingConfig": {
            "EndpointsConfig": null
          },
          "platform": {
            "architecture": "",
            "os": ""
          },
          "containerName": "foo-4"
        }
      },
      {
        "version": 3,
        "createTime": "2024-01-22 06:07:48",
        "status": {
          "version": 3,
          "createTime": "2024-01-22 06:07:48",
          "config": {
            "Hostname": "",
            "Domainname": "",
            "User": "",
            "AttachStdin": false,
            "AttachStdout": false,
            "AttachStderr": false,
            "ExposedPorts": {
              "22/tcp": {},
              "443/tcp": {}
            },
            "Tty": true,
            "OpenStdin": true,
            "StdinOnce": false,
            "Env": [
              "USER=foo",
              "CONTAINER_VERSION=3"
            ],
            "Cmd": null,
            "Image": "nvidia/cuda:10.0-base",
            "Volumes": null,
            "WorkingDir": "",
            "Entrypoint": null,
            "OnBuild": null,
            "Labels": null
          },
          "hostConfig": {
            "Binds": [
              "veil-1:/root/veil-1",
              "/mynfs/data/ctr-knock:/root/data/ctr-knock"
            ],
            "ContainerIDFile": "",
            "LogConfig": {
              "Type": "",
              "Config": null
            },
            "NetworkMode": "",
            "PortBindings": {
              "22/tcp": [
                {
                  "HostIp": "",
                  "HostPort": "40000"
                }
              ],
              "443/tcp": [
                {
                  "HostIp": "",
                  "HostPort": "40001"
                }
              ]
            },
            "RestartPolicy": {
              "Name": "",
              "MaximumRetryCount": 0
            },
            "AutoRemove": false,
            "VolumeDriver": "",
            "VolumesFrom": null,
            "ConsoleSize": [
              0,
              0
            ],
            "CapAdd": null,
            "CapDrop": null,
            "CgroupnsMode": "",
            "Dns": null,
            "DnsOptions": null,
            "DnsSearch": null,
            "ExtraHosts": null,
            "GroupAdd": null,
            "IpcMode": "",
            "Cgroup": "",
            "Links": null,
            "OomScoreAdj": 0,
            "PidMode": "",
            "Privileged": false,
            "PublishAllPorts": false,
            "ReadonlyRootfs": false,
            "SecurityOpt": null,
            "UTSMode": "",
            "UsernsMode": "",
            "ShmSize": 0,
            "Isolation": "",
            "CpuShares": 0,
            "Memory": 0,
            "NanoCpus": 0,
            "CgroupParent": "",
            "BlkioWeight": 0,
            "BlkioWeightDevice": null,
            "BlkioDeviceReadBps": null,
            "BlkioDeviceWriteBps": null,
            "BlkioDeviceReadIOps": null,
            "BlkioDeviceWriteIOps": null,
            "CpuPeriod": 0,
            "CpuQuota": 0,
            "CpuRealtimePeriod": 0,
            "CpuRealtimeRuntime": 0,
            "CpusetCpus": "",
            "CpusetMems": "",
            "Devices": null,
            "DeviceCgroupRules": null,
            "DeviceRequests": [
              {
                "Driver": "nvidia",
                "Count": 0,
                "DeviceIDs": [
                  "GPU-dc6d913c-8df4-a9a4-49e6-b82fcba5a6f9",
                  "GPU-bc85a406-0357-185f-a56c-afb49572bdbe",
                  "GPU-c6b3ca5f-c1ac-8171-582b-737b70a6bbce"
                ],
                "Capabilities": [
                  [
                    "gpu"
                  ]
                ],
                "Options": null
              }
            ],
            "MemoryReservation": 0,
            "MemorySwap": 0,
            "MemorySwappiness": null,
            "OomKillDisable": null,
            "PidsLimit": null,
            "Ulimits": null,
            "CpuCount": 0,
            "CpuPercent": 0,
            "IOMaximumIOps": 0,
            "IOMaximumBandwidth": 0,
            "MaskedPaths": null,
            "ReadonlyPaths": null
          },
          "networkingConfig": {
            "EndpointsConfig": null
          },
          "platform": {
            "architecture": "",
            "os": ""
          },
          "containerName": "foo-3"
        }
      },
      {
        "version": 2,
        "createTime": "2024-01-22 06:07:34",
        "status": {
          "version": 2,
          "createTime": "2024-01-22 06:07:34",
          "config": {
            "Hostname": "",
            "Domainname": "",
            "User": "",
            "AttachStdin": false,
            "AttachStdout": false,
            "AttachStderr": false,
            "ExposedPorts": {
              "22/tcp": {},
              "443/tcp": {}
            },
            "Tty": true,
            "OpenStdin": true,
            "StdinOnce": false,
            "Env": [
              "USER=foo",
              "CONTAINER_VERSION=2"
            ],
            "Cmd": null,
            "Image": "nvidia/cuda:10.0-base",
            "Volumes": null,
            "WorkingDir": "",
            "Entrypoint": null,
            "OnBuild": null,
            "Labels": null
          },
          "hostConfig": {
            "Binds": [
              "veil-1:/root/veil-1",
              "/mynfs/data/ctr-knock:/root/data/ctr-knock"
            ],
            "ContainerIDFile": "",
            "LogConfig": {
              "Type": "",
              "Config": null
            },
            "NetworkMode": "",
            "PortBindings": {
              "22/tcp": [
                {
                  "HostIp": "",
                  "HostPort": "40002"
                }
              ],
              "443/tcp": [
                {
                  "HostIp": "",
                  "HostPort": "40003"
                }
              ]
            },
            "RestartPolicy": {
              "Name": "",
              "MaximumRetryCount": 0
            },
            "AutoRemove": false,
            "VolumeDriver": "",
            "VolumesFrom": null,
            "ConsoleSize": [
              0,
              0
            ],
            "CapAdd": null,
            "CapDrop": null,
            "CgroupnsMode": "",
            "Dns": null,
            "DnsOptions": null,
            "DnsSearch": null,
            "ExtraHosts": null,
            "GroupAdd": null,
            "IpcMode": "",
            "Cgroup": "",
            "Links": null,
            "OomScoreAdj": 0,
            "PidMode": "",
            "Privileged": false,
            "PublishAllPorts": false,
            "ReadonlyRootfs": false,
            "SecurityOpt": null,
            "UTSMode": "",
            "UsernsMode": "",
            "ShmSize": 0,
            "Isolation": "",
            "CpuShares": 0,
            "Memory": 0,
            "NanoCpus": 0,
            "CgroupParent": "",
            "BlkioWeight": 0,
            "BlkioWeightDevice": null,
            "BlkioDeviceReadBps": null,
            "BlkioDeviceWriteBps": null,
            "BlkioDeviceReadIOps": null,
            "BlkioDeviceWriteIOps": null,
            "CpuPeriod": 0,
            "CpuQuota": 0,
            "CpuRealtimePeriod": 0,
            "CpuRealtimeRuntime": 0,
            "CpusetCpus": "",
            "CpusetMems": "",
            "Devices": null,
            "DeviceCgroupRules": null,
            "DeviceRequests": [
              {
                "Driver": "nvidia",
                "Count": 0,
                "DeviceIDs": [
                  "GPU-dc6d913c-8df4-a9a4-49e6-b82fcba5a6f9"
                ],
                "Capabilities": [
                  [
                    "gpu"
                  ]
                ],
                "Options": null
              }
            ],
            "MemoryReservation": 0,
            "MemorySwap": 0,
            "MemorySwappiness": null,
            "OomKillDisable": null,
            "PidsLimit": null,
            "Ulimits": null,
            "CpuCount": 0,
            "CpuPercent": 0,
            "IOMaximumIOps": 0,
            "IOMaximumBandwidth": 0,
            "MaskedPaths": null,
            "ReadonlyPaths": null
          },
          "networkingConfig": {
            "EndpointsConfig": null
          },
          "platform": {
            "architecture": "",
            "os": ""
          },
          "containerName": "foo-2"
        }
      },
      {
        "version": 1,
        "createTime": "2024-01-22 05:56:48",
        "status": {
          "version": 1,
          "createTime": "2024-01-22 05:56:48",
          "config": {
            "Hostname": "",
            "Domainname": "",
            "User": "",
            "AttachStdin": false,
            "AttachStdout": false,
            "AttachStderr": false,
            "ExposedPorts": {
              "22/tcp": {},
              "443/tcp": {}
            },
            "Tty": true,
            "OpenStdin": true,
            "StdinOnce": false,
            "Env": [
              "USER=foo",
              "CONTAINER_VERSION=1"
            ],
            "Cmd": null,
            "Image": "nvidia/cuda:10.0-base",
            "Volumes": null,
            "WorkingDir": "",
            "Entrypoint": null,
            "OnBuild": null,
            "Labels": null
          },
          "hostConfig": {
            "Binds": [
              "veil-0:/root/veil-0",
              "/mynfs/data/ctr-knock:/root/data/ctr-knock"
            ],
            "ContainerIDFile": "",
            "LogConfig": {
              "Type": "",
              "Config": null
            },
            "NetworkMode": "",
            "PortBindings": {
              "22/tcp": [
                {
                  "HostIp": "",
                  "HostPort": "40000"
                }
              ],
              "443/tcp": [
                {
                  "HostIp": "",
                  "HostPort": "40001"
                }
              ]
            },
            "RestartPolicy": {
              "Name": "",
              "MaximumRetryCount": 0
            },
            "AutoRemove": false,
            "VolumeDriver": "",
            "VolumesFrom": null,
            "ConsoleSize": [
              0,
              0
            ],
            "CapAdd": null,
            "CapDrop": null,
            "CgroupnsMode": "",
            "Dns": null,
            "DnsOptions": null,
            "DnsSearch": null,
            "ExtraHosts": null,
            "GroupAdd": null,
            "IpcMode": "",
            "Cgroup": "",
            "Links": null,
            "OomScoreAdj": 0,
            "PidMode": "",
            "Privileged": false,
            "PublishAllPorts": false,
            "ReadonlyRootfs": false,
            "SecurityOpt": null,
            "UTSMode": "",
            "UsernsMode": "",
            "ShmSize": 0,
            "Isolation": "",
            "CpuShares": 0,
            "Memory": 0,
            "NanoCpus": 0,
            "CgroupParent": "",
            "BlkioWeight": 0,
            "BlkioWeightDevice": null,
            "BlkioDeviceReadBps": null,
            "BlkioDeviceWriteBps": null,
            "BlkioDeviceReadIOps": null,
            "BlkioDeviceWriteIOps": null,
            "CpuPeriod": 0,
            "CpuQuota": 0,
            "CpuRealtimePeriod": 0,
            "CpuRealtimeRuntime": 0,
            "CpusetCpus": "",
            "CpusetMems": "",
            "Devices": null,
            "DeviceCgroupRules": null,
            "DeviceRequests": [
              {
                "Driver": "nvidia",
                "Count": 0,
                "DeviceIDs": [
                  "GPU-bc85a406-0357-185f-a56c-afb49572bdbe",
                  "GPU-c6b3ca5f-c1ac-8171-582b-737b70a6bbce",
                  "GPU-dc6d913c-8df4-a9a4-49e6-b82fcba5a6f9"
                ],
                "Capabilities": [
                  [
                    "gpu"
                  ]
                ],
                "Options": null
              }
            ],
            "MemoryReservation": 0,
            "MemorySwap": 0,
            "MemorySwappiness": null,
            "OomKillDisable": null,
            "PidsLimit": null,
            "Ulimits": null,
            "CpuCount": 0,
            "CpuPercent": 0,
            "IOMaximumIOps": 0,
            "IOMaximumBandwidth": 0,
            "MaskedPaths": null,
            "ReadonlyPaths": null
          },
          "networkingConfig": {
            "EndpointsConfig": null
          },
          "platform": {
            "architecture": "",
            "os": ""
          },
          "containerName": "foo-1"
        }
      }
    ]
  }
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» code|integer|true|none||none|
|» msg|string|true|none||none|
|» data|object|true|none||none|
|»» history|[object]|true|none||none|
|»»» version|integer|true|none||none|
|»»» createTime|string|true|none||none|
|»»» status|object|true|none||none|
|»»»» version|integer|true|none||none|
|»»»» createTime|string|true|none||none|
|»»»» config|object|true|none||none|
|»»»»» Hostname|string|true|none||none|
|»»»»» Domainname|string|true|none||none|
|»»»»» User|string|true|none||none|
|»»»»» AttachStdin|boolean|true|none||none|
|»»»»» AttachStdout|boolean|true|none||none|
|»»»»» AttachStderr|boolean|true|none||none|
|»»»»» ExposedPorts|object|true|none||none|
|»»»»»» 22|object|false|none||none|
|»»»»»» 443|object|false|none||none|
|»»»»» Tty|boolean|true|none||none|
|»»»»» OpenStdin|boolean|true|none||none|
|»»»»» StdinOnce|boolean|true|none||none|
|»»»»» Env|[string]|true|none||none|
|»»»»» Cmd|null|true|none||none|
|»»»»» Image|string|true|none||none|
|»»»»» Volumes|null|true|none||none|
|»»»»» WorkingDir|string|true|none||none|
|»»»»» Entrypoint|null|true|none||none|
|»»»»» OnBuild|null|true|none||none|
|»»»»» Labels|null|true|none||none|
|»»»» hostConfig|object|true|none||none|
|»»»»» Binds|[string]|true|none||none|
|»»»»» ContainerIDFile|string|true|none||none|
|»»»»» LogConfig|object|true|none||none|
|»»»»»» Type|string|true|none||none|
|»»»»»» Config|null|true|none||none|
|»»»»» NetworkMode|string|true|none||none|
|»»»»» PortBindings|object|true|none||none|
|»»»»»» 22|[object]|false|none||none|
|»»»»»»» HostIp|string|true|none||none|
|»»»»»»» HostPort|string|true|none||none|
|»»»»»» 443|[object]|false|none||none|
|»»»»»»» HostIp|string|true|none||none|
|»»»»»»» HostPort|string|true|none||none|
|»»»»» RestartPolicy|object|true|none||none|
|»»»»»» Name|string|true|none||none|
|»»»»»» MaximumRetryCount|integer|true|none||none|
|»»»»» AutoRemove|boolean|true|none||none|
|»»»»» VolumeDriver|string|true|none||none|
|»»»»» VolumesFrom|null|true|none||none|
|»»»»» ConsoleSize|[integer]|true|none||none|
|»»»»» CapAdd|null|true|none||none|
|»»»»» CapDrop|null|true|none||none|
|»»»»» CgroupnsMode|string|true|none||none|
|»»»»» Dns|null|true|none||none|
|»»»»» DnsOptions|null|true|none||none|
|»»»»» DnsSearch|null|true|none||none|
|»»»»» ExtraHosts|null|true|none||none|
|»»»»» GroupAdd|null|true|none||none|
|»»»»» IpcMode|string|true|none||none|
|»»»»» Cgroup|string|true|none||none|
|»»»»» Links|null|true|none||none|
|»»»»» OomScoreAdj|integer|true|none||none|
|»»»»» PidMode|string|true|none||none|
|»»»»» Privileged|boolean|true|none||none|
|»»»»» PublishAllPorts|boolean|true|none||none|
|»»»»» ReadonlyRootfs|boolean|true|none||none|
|»»»»» SecurityOpt|null|true|none||none|
|»»»»» UTSMode|string|true|none||none|
|»»»»» UsernsMode|string|true|none||none|
|»»»»» ShmSize|integer|true|none||none|
|»»»»» Isolation|string|true|none||none|
|»»»»» CpuShares|integer|true|none||none|
|»»»»» Memory|integer|true|none||none|
|»»»»» NanoCpus|integer|true|none||none|
|»»»»» CgroupParent|string|true|none||none|
|»»»»» BlkioWeight|integer|true|none||none|
|»»»»» BlkioWeightDevice|null|true|none||none|
|»»»»» BlkioDeviceReadBps|null|true|none||none|
|»»»»» BlkioDeviceWriteBps|null|true|none||none|
|»»»»» BlkioDeviceReadIOps|null|true|none||none|
|»»»»» BlkioDeviceWriteIOps|null|true|none||none|
|»»»»» CpuPeriod|integer|true|none||none|
|»»»»» CpuQuota|integer|true|none||none|
|»»»»» CpuRealtimePeriod|integer|true|none||none|
|»»»»» CpuRealtimeRuntime|integer|true|none||none|
|»»»»» CpusetCpus|string|true|none||none|
|»»»»» CpusetMems|string|true|none||none|
|»»»»» Devices|null|true|none||none|
|»»»»» DeviceCgroupRules|null|true|none||none|
|»»»»» DeviceRequests|[object]|true|none||none|
|»»»»»» Driver|string|true|none||none|
|»»»»»» Count|integer|true|none||none|
|»»»»»» DeviceIDs|[string]|true|none||none|
|»»»»»» Capabilities|[array]|true|none||none|
|»»»»»» Options|null|true|none||none|
|»»»»» MemoryReservation|integer|true|none||none|
|»»»»» MemorySwap|integer|true|none||none|
|»»»»» MemorySwappiness|null|true|none||none|
|»»»»» OomKillDisable|null|true|none||none|
|»»»»» PidsLimit|null|true|none||none|
|»»»»» Ulimits|null|true|none||none|
|»»»»» CpuCount|integer|true|none||none|
|»»»»» CpuPercent|integer|true|none||none|
|»»»»» IOMaximumIOps|integer|true|none||none|
|»»»»» IOMaximumBandwidth|integer|true|none||none|
|»»»»» MaskedPaths|null|true|none||none|
|»»»»» ReadonlyPaths|null|true|none||none|
|»»»» networkingConfig|object|true|none||none|
|»»»»» EndpointsConfig|null|true|none||none|
|»»»» platform|object|true|none||none|
|»»»»» architecture|string|true|none||none|
|»»»»» os|string|true|none||none|
|»»»» containerName|string|true|none||none|

# Volume

## POST Create a volume

POST /api/v1/volumes

Create a volume, you can specify the size and name.

> Body Parameters

```json
{
  "name": "bar",
  "size": "20GB"
}
```

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|body|body|object| no |none|
|» name|body|string| yes |Volume Name|
|» size|body|string| yes |Supported Units: KB, MB, GB, TB|

> Response Examples

> OK

```json
{
  "code": 200,
  "msg": "Success",
  "data": {
    "name": "bar-1",
    "size": "20GB"
  }
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» code|integer|true|none||none|
|» msg|string|true|none||none|
|» data|object|true|none||none|
|»» name|string|true|none||Volume name of the new version created|
|»» size|string|true|none||Size|

## DELETE Delete a volume

DELETE /api/v1/volumes/{name}

Delete a volume.

The record in etcd will be deleted and cannot be recovered.

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|name|path|string| yes |Volume Name|

> Response Examples

> OK

```json
{
  "code": 200,
  "msg": "Success",
  "data": null
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» code|integer|true|none||none|
|» msg|string|true|none||none|
|» data|null|true|none||none|

## GET Get version info about a volume

GET /api/v1/volumes/{name}

Get information about the current version of the volume.

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|name|path|string| yes |Volume Name|

> Response Examples

> OK

```json
{
  "code": 200,
  "msg": "Success",
  "data": {
    "Info": {
      "version": 1,
      "createTime": "2024-01-23 02:07:31",
      "opt": {
        "Driver": "local",
        "DriverOpts": {
          "size": "20GB"
        },
        "Name": "bar-1"
      }
    }
  }
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» code|integer|true|none||none|
|» msg|string|true|none||none|
|» data|object|true|none||none|
|»» Info|object|true|none||none|
|»»» version|integer|true|none||none|
|»»» createTime|string|true|none||none|
|»»» opt|object|true|none||none|
|»»»» Driver|string|true|none||none|
|»»»» DriverOpts|object|true|none||none|
|»»»»» size|string|true|none||none|
|»»»» Name|string|true|none||none|

## PATCH Patch a volume

PATCH /api/v1/volumes/{name}/size

Patch the size of the latest version of an existing volume via create a new volume and copy the old volume data to the new volume.

Including expand and shrink of two operations, if the size is the same before and after the operation,it will be skipped.

If the size already used is larger than the size after shrink, then shrink operation will fail.

> The old version volume will be deleted.
> For example, if bar-1 calls the `/api/v1/volumes/:name/size` and create bar-2, bar-1 will be deleted.

> Body Parameters

```json
{
  "size": "50GB"
}
```

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|name|path|string| yes |Volume Name|
|body|body|object| no |none|
|» size|body|string| yes |Size|

> Response Examples

> OK

```json
{
  "code": 200,
  "msg": "Success",
  "data": {
    "name": "bar-2",
    "size": "50GB"
  }
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» code|integer|true|none||none|
|» msg|string|true|none||none|
|» data|object|true|none||none|
|»» name|string|true|none||Volume name of the new version created|
|»» size|string|true|none||Volume size of the new version created|

## GET Get all version info about a volume

GET /api/v1/volumes/{name}/history

Get information about all historical versions of the volume.

### Params

|Name|Location|Type|Required|Description|
|---|---|---|---|---|
|name|path|string| yes |Volume Name|

> Response Examples

> OK

```json
{
  "code": 200,
  "msg": "Success",
  "data": {
    "history": [
      {
        "version": 2,
        "createTime": "2024-01-23 02:17:30",
        "status": {
          "version": 2,
          "createTime": "2024-01-23 02:17:30",
          "opt": {
            "Driver": "local",
            "DriverOpts": {
              "size": "50GB"
            },
            "Name": "bar-2"
          }
        }
      },
      {
        "version": 1,
        "createTime": "2024-01-23 02:17:28",
        "status": {
          "version": 1,
          "createTime": "2024-01-23 02:17:28",
          "opt": {
            "Driver": "local",
            "DriverOpts": {
              "size": "20GB"
            },
            "Name": "bar-1"
          }
        }
      }
    ]
  }
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» code|integer|true|none||none|
|» msg|string|true|none||none|
|» data|object|true|none||none|
|»» history|[object]|true|none||none|
|»»» version|integer|true|none||none|
|»»» createTime|string|true|none||none|
|»»» status|object|true|none||none|
|»»»» version|integer|true|none||none|
|»»»» createTime|string|true|none||none|
|»»»» opt|object|true|none||none|
|»»»»» Driver|string|true|none||none|
|»»»»» DriverOpts|object|true|none||none|
|»»»»»» size|string|true|none||none|
|»»»»» Name|string|true|none||none|

# Resource

## GET Get gpu usage status

GET /api/v1/resources/gpus

0 means not used, 1 means used.

> Response Examples

> OK

```json
{
  "code": 200,
  "msg": "Success",
  "data": {
    "gpus": {
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
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» code|integer|true|none||none|
|» msg|string|true|none||none|
|» data|object|true|none||none|
|»» gpus|object|true|none||key: uuid, value: status|
|»»» GPU-04adce59-e7fc-19ed-6800-bc09e5f8fa31|integer|true|none||none|
|»»» GPU-281d9730-5a26-7c56-12fb-3a3d5a24ab68|integer|true|none||none|
|»»» GPU-36009026-9470-a2e0-73d3-222a63b82e4e|integer|true|none||none|
|»»» GPU-7a42be89-64fe-5383-c7be-49d199a96b3d|integer|true|none||none|
|»»» GPU-82fbe07b-200b-1d4c-4fbe-b0b54db86be5|integer|true|none||none|
|»»» GPU-bc85a406-0357-185f-a56c-afb49572bdbe|integer|true|none||none|
|»»» GPU-c6b3ca5f-c1ac-8171-582b-737b70a6bbce|integer|true|none||none|
|»»» GPU-dc6d913c-8df4-a9a4-49e6-b82fcba5a6f9|integer|true|none||none|

## GET Get port usage status

GET /api/v1/resources/ports

> Response Examples

> OK

```json
{
  "code": 200,
  "msg": "Success",
  "data": {
    "ports": {
      "StartPort": 40000,
      "EndPort": 65535,
      "AvailableCount": 25536,
      "UsedPortSet": {
        "40000": {},
        "40001": {}
      }
    }
  }
}
```

### Responses

|HTTP Status Code |Meaning|Description|Data schema|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|OK|Inline|

### Responses Data Schema

HTTP Status Code **200**

|Name|Type|Required|Restrictions|Title|description|
|---|---|---|---|---|---|
|» code|integer|true|none||none|
|» msg|string|true|none||none|
|» data|object|true|none||none|
|»» ports|object|true|none||none|
|»»» StartPort|integer|true|none||Start of available port numbers|
|»»» EndPort|integer|true|none||End of available port numbers|
|»»» AvailableCount|integer|true|none||Number of available port numbers|
|»»» UsedPortSet|object|true|none||Used port number|

# Data Schema

