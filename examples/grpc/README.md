#### Setup Grpc code generation

```shell
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1
  OR 
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1

protoc --go_out=out --go_opt=paths=source_relative --go-grpc_out=out --go-grpc_opt=paths=source_relative proto/helloworld.proto
```

The compiler will read input files proto/helloworld.proto from within the src directory, and write output files
proto/helloworld.pb.go and proto/helloworld_grpc.pb.go to the out directory. The compiler automatically creates nested
output sub-directories if necessary, but will not create the output directory itself.

--go-out and --go-grpc_out -> says where to put the files <br>
--go_opt=paths=source_relative -> paths are relative to this dir

#### Sample server for sample cleint

https://grpc.io/docs/languages/go/quickstart/
