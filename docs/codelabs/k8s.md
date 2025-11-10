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

This codelab uses Golang for the example service however the language used for this service isn't that important. Just 
make sure you're able to build a binary in whatever your preferred language is.  

### What you'll learn
This codelab is quite long and tries to give an idea of what a complete build pipeline might look like for a docker and
kubernetes based project. You'll learn:

- How to build a service and bake that into docker image 
- How to build a kubernetes deployment for that docker image
- Starting minikube and testing your deployment out
- Setting up aliases to streamline your dev workflow

### What if I get stuck?

The final result of running through this codelab can be found
[here](https://github.com/thought-machine/please-codelabs/tree/main/kubernetes_and_docker) for reference. If you really get stuck
you can find us on [gitter](https://gitter.im/please-build/Lobby)!

## Creating a service
Duration: 5

First up, let's create a service to deploy. It's not really important what it does or what language we implement it in. 
For this codelab, we'll make a simple hello world HTTP service in go.

### Initialising the project
```bash
plz init 
go mod init github.com/example/module
plz init plugin go
```

### Set up the Go plugin

Add a go toolchain to `third_party/go/BUILD`
```go
go_toolchain(
    name = "toolchain",
    version = "1.20",
)
```

And configure the plugin:
```
[Plugin "go"]
Target = //plugins:go
ImportPath = github.com/example/module
GoTool = //third_party/go:toolchain|go
```

For more information on this, check out the Go codelab. 

### Creating a Go service
Create a file `hello_service/service.go`:

```golang
package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "This is my website!")
	})

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, HTTP!")
	})

	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal("Error starting the server: ", err)
	}
}
```

Then create a `hello_service/BUILD` file like so:
```python
go_binary(
  name = "hello_service",
  srcs = ["service.go"],
  visibility = ["//hello_service/k8s:all"],
)
```

And test it works:

```bash
plz run //hello_service:hello_service && curl localhost:8000
pkill hello_service
```

The output should look like this:
```bash
[1] 28694
Hello, world!
[1]+  Terminated              plz run //hello_service
```

## Building a Docker image
Duration: 5

Before we create a docker image for our service, it can be useful to create a base image that all our services share. 
This can be used this to install language runtimes e.g. a python interpreter. If you're using a language that requires
a runtime, this is where you should install it. In this case, we're using Go so this isn't strictly necessary. 

Let's create a base docker file for our repo that all our services will use in `common/docker/Dockerfile-base`:
```
FROM ubuntu:22.04

RUN apt update -y && apt upgrade -y
```

### Docker build rules

To use the docker build rules, we need to install the docker plugin, as well as the shell plugin which it requires:

```bash
plz init plugin shell && plz init plugin docker
```

We can then build a set of scripts that help us build, and push our docker images. Add the following to `common/docker/BUILD`:

```python
docker_image(
    name = "base",
    dockerfile = "Dockerfile-base",
    visibility = ["PUBLIC"],
)
```

And then let's build that:
```
$ plz build //common/docker:base
Build finished; total time 80ms, incrementality 40.0%. Outputs:
//common/docker:base:
  plz-out/bin/common/docker/base.sh
```

### So what's going on?
As promised, the output of the docker image rule is a script that can build the docker image for you. We can have a 
look at what the script is doing:

```bash
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
```bash
plz run //common/docker:base
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
docker_image(
    name = "image",
    srcs = ["//hello_service"],
    dockerfile = "Dockerfile",
    base_image = "//common/docker:base",
)
```

Let's build this and have a look at the script it generates: 

```
$ plz build //hello_service/k8s:image
Build finished; total time 100ms, incrementality 100.0%. Outputs:
//hello_service/k8s:image:
  plz-out/bin/hello_service/k8s/image.sh

$ cat plz-out/bin/hello_service/k8s/image.sh
#!/bin/sh
./plz-out/bin/common/docker/base.sh \
  && docker build -t please-example/image:0d45575ad71adea9861b079e5d56ff0bdc179a1868d06d6b3d102721824c1538 -f \
  Dockerfile - < plz-out/gen/hello_service/k8s/_image#docker_context.tar.gz
```

Note, this script takes care of building the base image for us, so we don't have to orchestrate this ourselves.

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
          image: //hello_service/k8s:image
          ports:
            # This must match the port we start the server on in hello-service/main.py
            - containerPort: 8000
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
      port: 8000
      targetPort: 8000
```

### Kubernetes rules
Note that we've referenced the image `//hello-service/k8s:image` in the deployment. The kubernetes rules are able to 
template your yaml files substituting in the image with the correct label based on the version of the image we just 
built! This ties all the images and kubernetes manifests together based on the current state of the repo making the
deployment much more reproducible!

To add the kubernetes rules, run `plz init plugin k8s`.

Let's update `hello_service/k8s/BUILD` to build these manifests:

```python
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
    containers = [":image"],
)
```

And check that has done the right thing:
```
$ plz build //hello_service/k8s
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
          # This must match the port we start the server on in hello-service/main.py
          - containerPort: 8000
```

As you can see, this image matches the image we built earlier! These rules also provide a useful script for pushing 
the manifests to kubernetes:

```
$ plz build //hello_service/k8s:k8s_push
Build finished; total time 140ms, incrementality 100.0%. Outputs:
//hello_service/k8s:k8s_push:
  plz-out/bin/hello_service/k8s/k8s_push.sh

$ cat plz-out/bin/hello_service/k8s/k8s_push.sh
#!/bin/sh
kubectl apply -f plz-out/gen/hello_service/k8s/templated_deployment.yaml && \
kubectl apply -f plz-out/gen/hello_service/k8s/templated_service.yaml
```

## Local testing with minikube
Duration: 5

Let's tie this all together by deploying our service to minikube! 

### Setting up minikube
We can get Please to download minikube for us. Let's create `third_party/binary/BUILD` to do so:

```
remote_file (
    name = "minikube",
    url = f"https://storage.googleapis.com/minikube/releases/latest/minikube-{CONFIG.OS}-{CONFIG.ARCH}",
    binary = True,
)
```

And then we can start the cluster like so:
```bash
plz run //third_party/binary:minikube -- start
```

### Deploying our service 

First we need to push our images to minikube's docker. To do this we need to point `docker` at minikube:

```bash
eval $(plz run //third_party/binary:minikube -- docker-env)
```

Then we can run our deployment scripts:

```bash
plz run //hello_service/k8s:image_load && plz run //hello_service/k8s:k8s_push
```

And check they're working as we expected:

```
$ kubectl port-forward service/hello-svc 8000:8000 && curl localhost:8000

[1] 25986
Hello world!

$ pkill kubectl 
[1]+  Terminated              kubectl kubectl port-forward service/hello-svc 8000:8000
```

## Please deploy
Duration: 5

Here we have learnt about the provided targets we need to run to get our changes deployed to minikube, however it's a 
bit of a ritual. Let's look at consolidating this into a single command. Luckily the generated targets are labeled so 
this is as simple as: 

```bash
plz run sequential --include docker-build --include k8s-push //hello_service/... 
```

We can then set up an alias for this in `.plzconfig`:

```
[alias "deploy"]
cmd = run sequential --include docker-build --include k8s-push
; Enable tab completion for build labels
positionallabels = true
```

This is used like: 

```bash
plz deploy //hello_service/...
```

## Docker build and build systems
Duration: 7

To finish this off, it's worth talking about the challenges with building docker images from Docker files in a 
file based build system. 

Integrating a build system with `docker build` is notoriously difficult. Build systems have trouble building your image 
as `docker build` sends the image to a daemon running in the background. There's no easy way to get a file based artifact 
out of Docker without this extra infrastructure. The built in rules produce a number of scripts to help build, load, 
push and save images:

```
docker_image(
    name = "image",
    srcs = [":example"],
    base_image = ":base",
    run_args = "-p 8000:8000",
    visibility = ["//k8s/example:all"],
)
```

This single target produces the following sub-targets:

- `:image_fqn` target contains the fully qualified name of the generated image. Each image gets tagged with the hash
of its inputs so this can be relied upon to uniquely identify this image. 
- `:image` & `:image_load` are the same script. This script loads the image into the local docker daemon. It will 
also make sure the base image is build and loaded first. 
- `:image_push` will load and push the image to the docker registry as configured by your local machines docker 
environment. 
- `:image_save` will load and then save the image to a `.tar` in `plz-out/gen`
- `:image_run` will run the image in the local docker env

There are two ways we anticipate these targets to be used as part of a CI/CD pipeline:

- The build server can be given access to the docker registry, and the images can be loaded directly with `:image_push`.
- The build server can save the images out to an offline image tarball with `:image_save`. These can be exported as 
artifacts from the build server. Another stage of the CI/CD pipeline can then push these to the docker registry via 
`docker load`.
