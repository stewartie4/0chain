module 0chain.net/smartcontract/multisigsc/test

replace 0chain.net/core => ../../../core

replace 0chain.net/chaincore => ../../../chaincore

replace 0chain.net/smartcontract => ../../../smartcontract

replace 0chain.net/conductor => ../../../conductor

// Dev only.
replace github.com/0chain/gosdk => ../../../gosdk

require (
	0chain.net/chaincore v0.0.0
	0chain.net/core v0.0.0
	0chain.net/smartcontract v0.0.0
	github.com/spf13/viper v1.7.0
	go.uber.org/zap v1.15.0
)

go 1.13
