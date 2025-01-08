#!/bin/bash

# if no docker image deviceenv.azurecr.io/golang:10235667, pull it from Azure
if [ "$(docker images -q deviceenv.azurecr.io/golang:10235667 2> /dev/null)" == "" ]; then
  docker login deviceenv.azurecr.io --username deviceenv --password 0wWSmpavQA6X9wjj1TRoqi+qDDcffSWR
  docker pull deviceenv.azurecr.io/golang:10235667
fi

# compile proto files
docker run -d --rm --rm --name goenv -v $(pwd)/..:/go/src/app -w /go/src/app deviceenv.azurecr.io/golang:10235667 sh -c 'rm -rf canf22g2 && go clean && protoc --go_out=. --go-grpc_out=. *.proto'