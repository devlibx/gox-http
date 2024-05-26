go install github.com/golang/mock/mockgen@v1.6.0

find . -type f -name '*_mock.go' -exec rm {} \;
find . -type f -name '*.go-e' -exec rm {} \;

go mod tidy
go fmt ./...
go generate ./...
go fmt ./...
go test ./... -v

cd v2;
go mod tidy
go fmt ./...
go generate ./...
go fmt ./...
go test ./... -v



