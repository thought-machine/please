FROM ubuntu:xenial
MAINTAINER peter.ebden@gmail.com

# Basic dependencies, Python and Java
RUN apt-get update && apt-get install -y python2.7 python3.5 python-pip python3-pip openjdk-8-jdk curl unzip git
RUN pip install cffi
RUN pip3 install cffi

# Go
RUN curl -fsSL https://storage.googleapis.com/golang/go1.8.1.linux-amd64.tar.gz | tar -xzC /usr/local
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go

# PyPy
RUN curl -fsSL https://bitbucket.org/pypy/pypy/downloads/pypy2-v5.7.1-linux64.tar.bz2 | tar -xjC /usr/local
RUN mv /usr/local/pypy2-v5.7.1-linux64 /usr/local/pypy
RUN ln -s /usr/local/pypy/bin/pypy /usr/local/bin/pypy
RUN ln -s /usr/local/pypy/bin/libpypy-c.so /usr/lib

# C++
RUN apt-get install -y pkg-config clang
RUN curl -fsSL https://github.com/unittest-cpp/unittest-cpp/releases/download/v2.0.0/unittest-cpp-2.0.0.tar.gz | tar -xzC /tmp
RUN cd /tmp/unittest-cpp-2.0.0 && ./configure --prefix=/usr && make -j4 && make install
RUN ln -s /usr/include/UnitTest++ /usr/include/unittest++
RUN ln -s /usr/lib/libUnitTest++.a /usr/lib/libunittest++.a
RUN rm -r /tmp/unittest-cpp-2.0.0

# Protocol buffers
RUN curl -fsSLo /tmp/protoc.zip https://github.com/google/protobuf/releases/download/v3.2.0/protoc-3.2.0-linux-x86_64.zip
RUN unzip /tmp/protoc.zip -x readme.txt -d /usr

# Locale
RUN locale-gen en_GB.UTF-8

# Welcome message
COPY /motd.txt /etc/motd
RUN echo 'cat /etc/motd' >> /etc/bash.bashrc
WORKDIR /tmp
