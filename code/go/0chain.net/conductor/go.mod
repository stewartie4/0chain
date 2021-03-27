module 0chain.net/conductor

go 1.14

replace 0chain.net/core => ../core

replace 0chain.net/chaincore => ../chaincore

replace 0chain.net/smartcontract => ../smartcontract

replace 0chain.net/conductor => ./

// Dev only.
replace github.com/0chain/gosdk => ../gosdk

require (
	0chain.net/chaincore v0.0.0
	0chain.net/core v0.0.0
	github.com/mitchellh/mapstructure v1.3.1
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/spf13/viper v1.7.0
)
