# We prefer Ubuntu 17.10 because it makes the installation of some packages simpler
# and is more up-to-date than 16.04.
# If you want to replicate this image on 16.04 the main difference is UnitTest++
# which you'll need to compile your own copy of - their version is too old (we need v2)
# and has some naming idiosyncrasies which are too hard to support).
FROM ubuntu:artful
MAINTAINER peter.ebden@gmail.com

# Most dependencies; Python, Java, Clang and Ruby (only used for fpm)
RUN apt-get update && \
    apt-get install -y python2.7 python3.5 python3-pip libpython2.7-dev openjdk-8-jdk-headless \
    curl unzip git locales pkg-config zlib1g-dev \
    clang libunittest++-dev ruby ruby-dev rubygems golint && \
    apt-get clean
RUN pip3 install cffi

# Go
# We can't install this above because the latest available version for Artful is 1.8
# and we require 1.9 for various minor features.
RUN curl -fsSL https://storage.googleapis.com/golang/go1.9.2.linux-amd64.tar.gz | tar -xzC /usr/local
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt

# Locale
RUN locale-gen en_GB.UTF-8

# fpm
RUN gem install --no-ri --no-rdoc fpm

# Welcome message
COPY /motd.txt /etc/motd
RUN echo 'cat /etc/motd' >> /etc/bash.bashrc
WORKDIR /tmp
