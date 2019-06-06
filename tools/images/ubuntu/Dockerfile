FROM ubuntu:bionic
MAINTAINER peter.ebden@gmail.com

# Go, Python, Java and other dependencies.
RUN apt-get update && \
    apt-get install -y python3 python3-dev python3-pip openjdk-8-jdk-headless \
    curl unzip git locales pkg-config zlib1g-dev && \
    apt-get clean

# Go - we want 1.12 here but the latest package available is 1.10.
RUN curl -fsSL https://dl.google.com/go/go1.12.5.linux-amd64.tar.gz | tar -xzC /usr/local
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt
# Golint
RUN go get golang.org/x/lint/golint && mv ~/go/bin/golint /usr/local/bin && rm -rf ~/go

# Locale
RUN locale-gen en_GB.UTF-8

# Welcome message
COPY /motd.txt /etc/motd
RUN echo 'cat /etc/motd' >> /etc/bash.bashrc
WORKDIR /tmp
