FROM ubuntu:bionic
MAINTAINER peter.ebden@gmail.com

# Go, Python, Java and other build dependencies.
RUN apt-get update && \
    apt-get install -y python3 python3-dev python3-pip openjdk-8-jdk-headless \
    curl git locales pkg-config zlib1g-dev golang-go golint && \
    apt-get clean

# Locale
RUN locale-gen en_GB.UTF-8

# Welcome message
COPY /motd.txt /etc/motd
RUN echo 'cat /etc/motd' >> /etc/bash.bashrc
WORKDIR /tmp
