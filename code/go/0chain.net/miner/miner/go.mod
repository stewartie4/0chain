module miner

replace 0chain.net/core => ../../core

replace 0chain.net/chaincore => ../../chaincore

replace 0chain.net/smartcontract => ../../smartcontract

replace 0chain.net/miner => ../../miner

replace 0chain.net/sharder => ../../sharder

replace 0chain.net/conductor => ../../conductor

// replace 0chain.net/conductor/conductrpc => ../../conductor/conductrpc

// temporary, for development
replace github.com/0chain/gosdk => ../../gosdk

require (
	0chain.net/chaincore v0.0.0
	0chain.net/conductor v0.0.0-00010101000000-000000000000
	0chain.net/core v0.0.0
	0chain.net/miner v0.0.0
	0chain.net/smartcontract v0.0.0
	github.com/asaskevich/govalidator v0.0.0-20190424111038-f61b66f89f4a // indirect
	github.com/pelletier/go-toml v1.4.0 // indirect
	github.com/spf13/viper v1.7.0
	github.com/valyala/gozstd v1.5.0 // indirect
	go.uber.org/zap v1.15.0
)

go 1.13
