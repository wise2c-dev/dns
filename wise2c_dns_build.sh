#! /bin/sh

export ARCH=amd64
export VERSION=1.14.10
export PKG=k8s.io/dns

GOBINPATH=`pwd`/bin

export GOBIN=$GOBINPATH

if [ ! -d $GOBINPATH ]; then
  mkdir $GOBINPATH
fi

./build/build.sh 
