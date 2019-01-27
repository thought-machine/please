#!/bin/sh
exec plz-out/bin/src/please run -p -- //third_party/go:buildifier $@ `git ls-files | grep BUILD`
