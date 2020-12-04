id: k8s
summary: Kubernetes and Docker  
description: Learn about using Please to build and deploy Docker images and Kubernetes manifests 
categories: intermediate
tags: medium
status: Published
authors: Jon Poole
Feedback Link: https://github.com/thought-machine/please

# Kubernetes and Docker
## Overview
Duration: 1

### Prerequisites
- You must have Please installed: [Install Please](https://please.build/quickstart.html)
- You should be comfortable using the existing build rules.
- You should be familiar with [Docker](https://docs.docker.com/get-started/) 
  and [Kubernetes](https://kubernetes.io/docs/tutorials/kubernetes-basics/) 

### What You'll Learn
This codelab is quite long and tries to give an idea of what a complete build pipeline might look like for a docker and
kubernetes based service. You'll learn:

1) How to build a service and bake that into docker image 
2) How to build a kubernetes deployment for that docker image
3) Starting minikube and testing your deployment out
4) Setting up aliases to streamline your dev workflow

### What if I get stuck?

The final result of running through this codelab can be found
[here](https://github.com/thought-machine/please-codelabs/tree/main/k8s) for reference. If you really get stuck
you can find us on [gitter](https://gitter.im/please-build/Lobby)!

## Creating a service
Duration: 3

First up, let create a service to deploy. It's not really important what it does or what language we implement it in. 
For the sake of this codelabs, we'll make a super simple hello world HTTP service in Go.

### Initialising the project
```
$ go mod init github.com/thought-machine/please-codelabs/k8s
go: creating new go.mod: module github.com/thought-machine/please-codelabs/k8s

$ plz init --no_promp
Wrote config template to /home/jpoole/please-codelabs/k8s/.plzconfig, you're now ready to go!

Also wrote wrapper script to pleasew; users can invoke that directly to run Please, even without it installed.
```

### Create the Go service
Create a file `hello_service/main.go`:

```
// Package main implements a hello world http service
package main

import "net/http"

// Server implements the hello world HTTP server
type Server struct {
}

// ServeHTTP just responds with Hello, world! on the writer
func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if _, err := writer.Write([]byte("Hello world!\n")); err != nil {
		panic(err)
	}
}

func main() {
	if err := http.ListenAndServe(":8080", new(Server)); err != nil {
		panic(err)
	}
}
```

Then create a `hello_service/BUILD` file like so:
```python
go_binary(
    name = "hello_service",
    srcs = ["main.go"],
    visibility = ["//hello_service/..."],
)
```

And test it works:
```
$ plz run //hello_service &
[1] 28694
$ curl localhost:8080
Hello, world!
$ pkill hello_service
[1]+  Terminated              plz run //hello_service
```

## Docker build and build systems
Duration: 7

Unfortunately, owing to the way `docker build` works, getting Please (or any build system) to build your image is a no
go. The image ends up being uploaded to a daemon running the background. There's no easy way to get a file based 
artifact out of Docker. This leaves us with two options:

- Distroless image - this approach generates a tarball matching the OSI image format that can easily to pushed to a 
docker registry later. 
- Context tar + script - this approach generates a tarball of the required files for the docker image as well as a shell 
script that can easily be run later to push the image to the docker registry.
                             
The second approach is more widespread and should feel familiar to those who're coming from a standard docker 
background. We'll be using this approach in this codelab however distroless images are worth considering for your 
project.

## Building a Docker image
Duration: 7

### A base image
It's good to create a base image that all our services share. We can install common packages e.g. the glibc 
compatibility library for alpine. Let's create a base docker file for our repo that all our services will use in 
`common/docker/Dockerfile-base`:
```
FROM alpine:3.7

RUN apk update && apk add libc6-compat
```

### Docker build rules
Unlike `go_lbrary()` the docker image build rules aren't built in. They are part of the extra rules found in the 
[pleasings](https://github.com/thought-machine/pleasings/tree/master/docker) repository. Let's create 
`common/docker/BUILD` to build our docker image:

```python
subinclude("///pleasings//docker")

docker_image(
    name = "base",
    dockerfile = "Dockerfile-base",
    visibility = ["PUBLIC"],
)
```

And then let's build that:
```
$ plz build //common/docker:base
Build stopped after 130ms. 1 target failed:
    //common/docker:base
rules/misc_rules.build_defs:555:9: error: You must set buildconfig.default-docker-repo in your .plzconfig to use Docker rules, e.g.
[buildconfig]
default-docker-repo = hub.docker.com
...
```


Oh no! Looks like we missed something. If you've got a docker registry to push to then great otherwise don't worry. This 
is just used to name the image. Let's set this to something sensible in `.plzbuild`:
```
[buildconfig]
default-docker-repo = please-examples
```

And try again:
```
$ plz build //common/docker:base
Build finished; total time 80ms, incrementality 40.0%. Outputs:
//common/docker:base:
  plz-out/bin/common/docker/base.sh
```

### So what's going on?
As promised, the output of the docker image rule is a script that can build the docker image for you. We can have a 
look at what the script is doing:

```
$ cat plz-out/bin/common/docker/base.sh
#!/bin/sh
docker build -t please-examples/base:0d45575ad71adea9861b079e5d56ff0bdc179a1868d06d6b3d102721824c1538 \
    -f Dockerfile-base - < plz-out/gen/common/docker/_base#docker_context.tar.gz
```

There's a couple key things to note:
- The image has been tagged with a hash based on the inputs to the rule. This means that we can always refer 
back to this specific version of this image. 
- It's generated us a `tar.gz` containing all the other files we might need to build the Docker image. 

We can run this script to build the image and push it to the docker daemon as set in our docker env:
```
$ plz run //common/docker:base
```

## Using our base image
Duration: 5

So now we have a base image, let's use it for our docker image. Create a `hello_service/k8s/Dockerfile` for our hello 
service:

```
FROM //common/docker:base

COPY /hello_service /hello_service

ENTRYPOINT [ "/hello_service" ] 
```

And then set up some build rules for that in `hello_service/k8s/BUILD`:

```
subinclude("///pleasings//docker")

docker_image(
    name = "image",
    srcs = ["//hello_service"],
    dockerfile = "Dockerfile",
    base_image = "//common/docker:base",
)
```

## Creating a Kubernetes deployment  
Duration: 5
Let's create `hello_service/k8s/deployment.yaml` for our service:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello
  labels:
    app: hello
spec:
  replicas: 3
  selector:
    matchLabels:
      app: hello
  template:
    metadata:
      labels:
        app: hello
    spec:
      containers:
        - name: main
          image: //hello-service/k8s:image
          ports:
            # This must match the port we start the server on in hello-service/main.go
            - containerPort: 8080
```

Let's also create `hello_service/k8s/service.yaml` for good measure:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: hello-svc
spec:
  selector:
    app: hello
  ports:
    - protocol: TCP
      port: 8080
      targetPort: 8080
```

### Kubernetes rules
Not that we've referenced the image `//hello-service/k8s:image` in the deployment. This is clearly not something 
kubernetes supports. Herein lies the power of the kubernetes rules! The kubernetes rules are able to template your yaml
files substituting in the image with the correct label based on the version of the image we just built! This makes the
deployment much more reproducible.

So lets update `hello_service/k8s/BUILD`:

```python
subinclude("///pleasings//docker", "///pleasings//k8s")

docker_image(
    name = "image",
    srcs = ["//hello_service"],
    dockerfile = "Dockerfile",
    base_image = "//common/docker:base",
)

k8s_config(
    name = "k8s",
    srcs = [
        "deployment.yaml",
        "service.yaml",
    ],
    containers = ["//hello_service/k8s:image"],
)
```

And check that has done the right thing:
```
$ plz build //hello_service/k8se/k8s/templated_deployment.yaml
Build finished; total time 90ms, incrementality 90.9%. Outputs:
//hello_service/k8s:k8s:
  plz-out/gen/hello_service/k8s/templated_deployment.yaml
  plz-out/gen/hello_service/k8s/templated_service.yaml


$ cat plz-out/gen/hello_service/k8s/templated_deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello
  labels:
    app: hello
spec:
  replicas: 3
  selector:
    matchLabels:
      app: hello
  template:
    metadata:
      labels:
        app: hello
    spec:
      containers:
      - name: main
        image: please-example/image:0d45575ad71adea9861b079e5d56ff0bdc179a1868d06d6b3d102721824c1538
        ports:
          # This must match the port we start the server on in hello-service/main.go
          - containerPort: 8080
```

As you can see, this image matches the image we built earlier! 

## Local testing with minikube
Let's tie this all together by deploying our service to minikube! 

### Setting up minikube
We can get Please to download minikube for us. Let's create `tools/minikube/BUILD` to do so:

```
remote_file (
    name = "minikube",
    url = "https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64",
    binary = True,
)
```

And then we can start the cluster like so:
```
$ plz run //tools/minikube -- start
```


### Deploying our service 

First we need to push our images to minikube's docker. To do this we need to point `docker` at minikube:

```
$ eval $(plz run //tools/minikube -- docker-env)
```

Then we can run our deployment scripts:

```
$ plz run //common/docker:base && plz run //hello_service/k8s:image
```

Finally we can then apply our deployments:
```
$ kubectl apply -f plz-out/gen/hello_service/k8s/templated_deployment.yaml 
deployment.apps/hello created

$ kubectl apply -f plz-out/gen/hello_service/k8s/templated_service.yaml 
service/hello-svc created

$ kubectl get po
NAME                    READY   STATUS    RESTARTS   AGE
hello-964988c48-d84nv   1/1     Running   0          16s
hello-964988c48-nfpf4   1/1     Running   0          16s
hello-964988c48-vkpfl   1/1     Running   0          16s
```

And check they're working as we expected:

```
$ kubectl port-forward service/hello-svc 8080:8080 &
[1] 25986
$ curl localhost:8080
Handling connection for 8080
Hello world!
$ pkill kubectl 
[1]+  Terminated              kubectl kubectl port-forward service/hello-svc 8080:8080
```