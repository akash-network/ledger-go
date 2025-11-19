module github.com/cosmos/ledger-cosmos-go

go 1.23.0

require (
	github.com/btcsuite/btcd/btcec/v2 v2.3.2
	github.com/stretchr/testify v1.11.1
	github.com/zondax/ledger-go v0.15.0
)

replace (
	github.com/zondax/hid => github.com/troian/hid v0.14.0
	github.com/zondax/ledger-go => ../
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/zondax/golem v0.27.0 // indirect
	github.com/zondax/hid v0.9.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250811230008-5f3141c8851a // indirect
	google.golang.org/grpc v1.75.0 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
