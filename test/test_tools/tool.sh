#!/bin/sh


if [ "$(cat $1)" != "wibble wibble wibble" ]; then
  echo "content wasn't wibble wibble wibble"
  echo $CONTENT
  exit 1
fi