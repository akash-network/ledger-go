module github.com/cosmos/ledger-cosmos-go

go 1.19

require (
	github.com/btcsuite/btcd/btcec/v2 v2.3.2
	github.com/stretchr/testify v1.8.1
	github.com/zondax/ledger-go v0.12.2
)

replace (
	github.com/zondax/hid => github.com/troian/hid v0.11.0
	github.com/zondax/ledger-go => ../
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/zondax/hid v0.9.1 // indirect
	golang.org/x/net v0.0.0-20201021035429-f5854403a974 // indirect
	golang.org/x/sys v0.0.0-20210119212857-b64e53b001e4 // indirect
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/grpc v1.50.1 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
