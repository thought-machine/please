FROM alpine:3.6
MAINTAINER peter.ebden@gmail.com

RUN apk update && apk add bash

# Having the dynamic linker under this name makes it easier for x86-64-linux-gnu
# binaries to work.
RUN ln -s /lib/ld-musl-x86_64.so.1 /lib/ld-linux-x86-64.so.2 && ln -s /lib /lib64
