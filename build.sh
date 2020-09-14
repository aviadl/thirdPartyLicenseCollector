#!/usr/bin/env bash

for os in linux darwin; do
    export GOOS="$os"
    echo "build for $GOOS:"
    for arch in 386 amd64; do
        if [[ "$os" == "darwin" && "$arch" == "386" ]]  # darwin/386 no longer supported in GO 1.15 or later
        then
          continue
        fi
        export GOARCH="$arch"
        NAME=bin/thirdPartyLicenseCollector_${GOOS}_${GOARCH}
        if [ $arch = "amd64" ]
        then
            go build -o $NAME .
        fi
    done
done
