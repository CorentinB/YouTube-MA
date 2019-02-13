#!/usr/bin/env bash

appname=$1
tag=$2
[ -z "$tag" ] && echo "Usage: build <appname> <version>" && exit 1
[ -z "$appname" ] && echo "Usage: build <appname> <version>" && exit 1

name=${appname}-${tag}-windows.exe
GOOS="windows" GOARCH="amd64" go build -ldflags="-s -w" -o $name
gzip -f $name
echo $name

name=${appname}-${tag}-linux
GOOS="linux" GOARCH="amd64" go build -ldflags="-s -w" -o $name
gzip -f $name
echo $name

name=${appname}-${tag}-osx
GOOS="darwin" GOARCH="amd64" go build -ldflags="-s -w" -o $name
gzip -f $name
echo $name

name=${appname}-${tag}-freebsd
GOOS="freebsd" GOARCH="amd64" go build -ldflags="-s -w" -o $name
gzip -f $name
echo $name