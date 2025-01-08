@echo off

:: check if image golang:10235667 existed，if not, build it.
docker images -q golang:10235667 2> nul
docker build -t golang:10235667 -f Dockerfile.10235667 .

:: make sure goenv container is not running or existed
FOR /f "tokens=*" %%i IN ('docker ps -aq name=goenv') DO docker rm -f %%i

:: Compile proto documentation
docker run -d --rm --name goenv -v %cd%/..:/go/src/app -w /go/src/app golang:10235667 sh -c "protoc --go_out=. --go-grpc_out=. *.proto"