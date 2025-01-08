@echo off

:: check if image golang:10235667 existed，if not, build it.
docker images -q golang:10235667 2> nul
docker build -t golang:10235667 -f Dockerfile.10235667 .

timeout /t 1
:: make sure goenv container is not running or existed
FOR /f "tokens=*" %%i IN ('docker ps -aq --filter "name=goenv"') DO docker rm -f %%i

:: Compile proto documentation
docker run -d --rm --name goenv -v %cd%/..:/go/src/app -w /go/src/app golang:10235667 sh -c "protoc --go_out=. --go-grpc_out=. *.proto"

timeout /t 1
:: make sure goenv container is not running or existed
FOR /f "tokens=*" %%i IN ('docker ps -aq --filter "name=goenv"') DO docker rm -f %%i

:: env setup and run ./app
docker run -it --rm --name goenv -v %cd%:/go/src/app -v %cd%/../canf22g2:/usr/local/go/src/canf22g2 -w /go/src/app -p 50051:50051 --memory=2g --security-opt seccomp=unconfined golang:10235667 sh -c "./prebuild_pkg.sh && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app && ./app"