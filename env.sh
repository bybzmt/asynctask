#!/bin/bash
CUR=$(cd `dirname $0`; pwd)

export GOPATH="${CUR}/lib"

has=$(echo "${PATH}" | grep "${CUR}")
if [ "$has" == "" ]; then
    export PATH="${PATH}:${CUR}/lib/bin"
fi
