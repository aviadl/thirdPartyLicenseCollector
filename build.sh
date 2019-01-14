#!/usr/bin/env bash

for os in linux darwin; do
    export GOOS="$os"
    echo "build for $GOOS:"
    for arch in 386 amd64; do
        export GOARCH="$arch"
        NAME=bin/thirdPartyLicenseCollector_${GOOS}_${GOARCH}
        if [ $arch = "amd64" ]
        then
            go build -o $NAME .
        fi
    done
done
