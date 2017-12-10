FROM ubuntu:xenial
MAINTAINER peter.ebden@gmail.com

# Basic dependencies, Python and Java
RUN apt-get update && apt-get install -y python2.7 libpython2.7-dev openjdk-8-jdk curl unzip git locales pkg-config
RUN pip install cffi
RUN pip3 install cffi

# Go
RUN curl -fsSL https://storage.googleapis.com/golang/go1.9.1.linux-amd64.tar.gz | tar -xzC /usr/local
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt

# C++
# Ubuntu's packaged version of UnitTest++ is older than we'd like and has some
# idiosyncratic naming (unittest++ vs. UnitTest++ etc).
RUN apt-get install -y clang
RUN curl -fsSL https://github.com/unittest-cpp/unittest-cpp/releases/download/v2.0.0/unittest-cpp-2.0.0.tar.gz | tar -xzC /tmp
RUN cd /tmp/unittest-cpp-2.0.0 && ./configure --prefix=/usr && make -j4 && make install
RUN rm -r /tmp/unittest-cpp-2.0.0

# Protocol buffers
RUN curl -fsSLo /tmp/protoc.zip https://github.com/google/protobuf/releases/download/v3.2.0/protoc-3.2.0-linux-x86_64.zip
RUN unzip /tmp/protoc.zip -x readme.txt -d /usr

# Locale
RUN locale-gen en_GB.UTF-8

# Ruby (for fpm)
RUN apt-get install -y ruby ruby-dev rubygems
RUN gem install --no-ri --no-rdoc fpm

# Welcome message
COPY /motd.txt /etc/motd
RUN echo 'cat /etc/motd' >> /etc/bash.bashrc

# Linters etc
WORKDIR /tmp
RUN go get github.com/golang/lint/golint && mv ~/go/bin/golint /usr/local/bin && rm -rf ~/go
