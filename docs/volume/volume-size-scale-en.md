## Scale Up And Down Docker Volume

## Prerequisites

I am using the following storage driver and filesystem in my test environment:

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

It should be noted that when we choose different Storage Driver, the disk structure of image, container and volume
storage is different.

Also, they do not necessarily support the `Set Volume Size` feature. For example, when we use DeviceMapper.

~~~
$ docker volume create --driver local --opt=o=size=20 --name data
~~~

Considering that when installing a new version of Docker, the default Storage Driver is Overlay2, but it is not
supported to set the size, we can set the size by changing the path of Docker Root Dir, and then use the xfs filesystem
so that we can set the size in the following way:

~~~
$ docker volume create --opt size=20GB --name foo
$ docker volume inspect foo
[
    {
        "CreatedAt": "2023-12-09T12:28:20Z",
        "Driver": "local",
        "Labels": null,
        "Mountpoint":"/localData/docker/volumes/foo/_data",
        "Name": "foo",
        "Options": {
            "size": "20GB"
        },
        "Scope": "local"
    }
]
~~~

## Implementation

As with Docker, it's easy to start a Nginx process when we use `docker run -d -P nginx:latest`, but the isolation of
Groups and Namespace behind it, as well as the image, container storage, etc., we don't need to think
about. Just need to remember simple commands.

The same is true for our project, where you just call a simple api.

It can be summarised as follows:

1. create a volume of size 10Gi named foo-0 and now want to expand it to 20Gi.
2. create a new 20Gi volume called foo-1, and move the contents of foo-0's actual storage directory to foo-1's storage
   directory.
3. Pull the full volume information from ETCD that created the container and modify the volume mount contents. Use this
   information to recreate a container, submit it to the ETCD, and return to the user the new container's ID, Name.