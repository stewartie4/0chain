package smartcontractinterface

import "0chain.net/core/util"

type StatesGetter interface {
	GetStateSmartContract(name string) util.MerklePatriciaTrieI
	GetStateSmartContractHash(name string) string
}

type StateInitiator interface {
	InitStateSmartContract(name string, state util.MerklePatriciaTrieI)
}
