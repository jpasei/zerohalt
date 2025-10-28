#!/bin/bash

export TARGET_OS=linux

rm -rf build/

mkdir -p build

docker compose run --rm format

for i in amd64 arm64; do
    export TARGET_ARCH=$i
    docker compose run --rm build
done
