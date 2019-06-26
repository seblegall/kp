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


## requirements and limitations

**Kopy doesn't have any requirements and (almost) any limitations.**

Of course, since Kopy purpose is to copy-past files from local to a running
container you will need a running container either on Docker or on Kubernetes. That's it !

Internally, Kopy read the files to copy and write a *tar* to the input of a *untar*
command running from inside the container. It means that your container must be able to
execute a tar command. Which is the case for **almost** all distro-based containers.

*(I haven't checked yet, but it shouldn't work for [distroless](https://github.com/GoogleContainerTools/distroless) containers)*


## Known issues

Kopy haven't any known issues. It just works.

However, if you find one, feel free to create an issue and
I will explain you why your issue is not relevant.

## The code seems familiar to you?

keep calm and breathe. It's perfectly normal.

Most of the code used in Kopy is copied from the [Skaffold](https://github.com/GoogleContainerTools/skaffold)
code base without any reference to the authors. I've just stolen it because I found it super smart and
I though I could became a Github rock star if I had the idea myself.