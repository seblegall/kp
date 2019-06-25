# kp - kopy

Kopy is a very simple tool offering a very strong power : ability to copy a file
or a directory from local file system to a **running** container file system (almost) natively.

* No need to install anything else than Kopy
* No need to run any daemon or daemonSet (Kubernetes)
* No need to change and rebuild your container image
* No need to restart your container
* No need to add anything in your Dockerfile

`kp` simply works like a regular `cp` but from local to a running container.
The container may be run by **Docker** or by **Kubernetes**, inside a pod. Kopy support both.

## Usage

Example with a running Docker container :

```bash
kp -c {container_id} /path/to/my/file /tmp
``` 

* `container_id` is the container ID you can get by running `docker ps`
* `/path/to/my/file` is the source file you want to copy. It can be a single file
or a directory. With Kopy you don't even have to precise you want to copy a directory.
* `/tmp` is the path where to copy the file in the container. 

Example with pod from a Kubernetes cluster :

````bash
kp -c {pod_name} -n {namespace} -c {container} /path/to/my/file /tmp 
````

* `pod_name` is the pod running your container
* `namespace` is the namespace running the pod. Default is `default`
* `container` is the specific container running in the pod.
If not set, it will take the first one available.

## Install

````bash
go install github.com/seblegall/kp
````
