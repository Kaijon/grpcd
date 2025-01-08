rm -f go.mod go.sum
go mod init app
go mod tidy
go get google.golang.org/grpc
go get google.golang.org/grpc/codes
go get google.golang.org/grpc/status
go get google.golang.org/protobuf/reflect/protoreflect
go get google.golang.org/protobuf/runtime/protoimpl
go get github.com/eclipse/paho.mqtt.golang
go install github.com/jstemmer/go-junit-report/v2@latest

