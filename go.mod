module github.com/zondax/ledger-go

go 1.13

require (
	github.com/golang/protobuf v1.5.2
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	github.com/zondax/hid v0.9.0
	google.golang.org/grpc v1.42.0
	google.golang.org/protobuf v1.27.1
)

replace (
	github.com/zondax/hid => github.com/troian/hid v0.10.0
)
