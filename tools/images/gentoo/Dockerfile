FROM gentoo/stage3-amd64

# Base system & locales
COPY make.conf /etc/portage/make.conf
COPY locale.gen /etc/locale.gen
RUN emerge --sync -q && locale-gen && eselect locale set en_GB.utf8 && env-update && source /etc/profile && emerge -q portage

# Python
RUN emerge -q python:3.5 net-misc/curl unzip dev-vcs/git dev-python/cffi
RUN emerge -q --newuse world
# Currently disabled due to interesting situation with it needing a masked openssl.
# It's not compulsory to run Please but would be nicer if we had it.
# RUN emerge pypy

# Go and Java, protobufs, linter
# Unsure why sandbox breaks here?
RUN FEATURES="-sandbox -usersandbox" emerge -q dev-lang/go
RUN emerge -q virtual/jdk dev-libs/protobuf dev-go/golint

WORKDIR /tmp
