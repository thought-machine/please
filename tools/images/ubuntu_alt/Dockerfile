FROM ubuntu:bionic
MAINTAINER peter.ebden@gmail.com

# Most dependencies; Python, Java, Clang
RUN apt-get update && \
    apt-get install -y python3 python3-dev python3-pip openjdk-11-jdk-headless \
    curl unzip git locales pkg-config zlib1g-dev clang && \
    apt-get clean

# Go - we want to test 1.13 beta here but the latest package available is 1.10.
RUN curl -fsSL https://dl.google.com/go/go1.13beta1.linux-amd64.tar.gz | tar -xzC /usr/local
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt

# Locale
RUN locale-gen en_GB.UTF-8

# Custom plzconfig
COPY /plzconfig /etc/plzconfig

# Welcome message
COPY /motd.txt /etc/motd
RUN echo 'cat /etc/motd' >> /etc/bash.bashrc
WORKDIR /tmp
