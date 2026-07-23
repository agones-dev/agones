module agones.dev/agones/examples/simple-game-server

go 1.26

require agones.dev/agones v1.56.0

require (
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.3 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260414002931-afd174a4e478 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478 // indirect
	google.golang.org/grpc v1.82.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace agones.dev/agones => ../../
