#!/bin/bash

# move ../canf22g2 to current folder if the folder exists else break
if [ -d "../canf22g2" ]; then
  cp -r ../canf22g2 .
else
  echo "No canf22g2 folder found in parent directory"
  exit 1
fi

# environment setup & Run ./app
#docker run -it --rm  --name goenv -v $(pwd):/go/src/app -w /go/src/app -p 50051:50051 golang:10235667 sh -c 'cp -r canf22g2 /usr/local/go/src && ./prebuild_pkg.sh && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build . '
docker run -it --rm  --name goenv -v $(pwd):/go/src/app -w /go/src/app -p 50051:50051 golang:10235667 sh -c 'cp -r canf22g2 /usr/local/go/src && ./prebuild_pkg.sh && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build . '
#docker run -it --rm  --name goenv -v $(pwd):/go/src/app -w /go/src/app -p 50051:50051 golang:10235667 bash
