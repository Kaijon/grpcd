@echo off

:: if ../canf22g2 folder exist, move here else exist
if exist "..\canf22g2" (
    xcopy /E /I "..\canf22g2" ".\canf22g2"
) else (
    echo No canf22g2 folder found in parent directory
    exit /b 1
)

:: make sure goenv container is not running or existed
FOR /f "tokens=*" %%i IN ('docker ps -aq -f name=goenv') DO docker rm -f %%i

:: env setup and run ./app
docker run -it --rm --name goenv -v %cd%:/go/src/app -w /go/src/app -p 50051:50051 golang:10235667 sh -c "cp -r canf22g2 /usr/local/go/src && ./prebuild_pkg.sh && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app && ./app"