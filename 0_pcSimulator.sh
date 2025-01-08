#!/bin/bash

# if no docker image golang:10235667, build it
if [ "$(docker images -q golang:10235667 2> /dev/null)" == "" ]; then
    docker build -t golang:10235667 -f Dockerfile.10235667 .
fi

# Ensure no container named "goenv" is running or exists
docker ps -aq -f name=goenv | xargs -r docker rm -f

# compile proto files
docker run -d --rm --name goenv  -v $(pwd)/..:/go/src/app -w /go/src/app golang:10235667 sh -c 'protoc --go_out=. --go-grpc_out=. *.proto'

# Ensure no container named "goenv" is running or exists before running the app
docker ps -aq -f name=goenv | xargs -r docker rm -f

# environment setup & Run ./app
docker run -it --rm  --name goenv -v $(pwd):/go/src/app -v $(pwd)/../canf22g2:/usr/local/go/src/canf22g2 -w /go/src/app -p 50051:50051 --memory=2g --security-opt seccomp=unconfined golang:10235667 sh -c './prebuild_pkg.sh && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app && ./app'