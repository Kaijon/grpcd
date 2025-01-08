@echo off

:: check if image golang:npm existed，if not, build it.
docker images -q golang:npm 2> nul
docker build -t golang:npm -f Dockerfile.npm .

:: Compile proto documentation
docker run -d --rm -v %cd%/..:/go/src/app -w /go/src/app golang:npm sh -c "protoc --js_out=import_style=commonjs:. --grpc-web_out=import_style=commonjs,mode=grpcweb:. *.proto"