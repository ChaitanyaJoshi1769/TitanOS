module github.com/ChaitanyaJoshi1769/TitanOS/services/node-agent

go 1.22

require (
	github.com/ChaitanyaJoshi1769/TitanOS/services/scheduler v0.1.0
	google.golang.org/grpc v1.60.0
	google.golang.org/protobuf v1.31.0
)

require (
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231212172506-995d672761c0 // indirect
)

replace github.com/ChaitanyaJoshi1769/TitanOS/services/scheduler => ../scheduler
