package smartcontractinterface

import "0chain.net/core/util"

type StateInitiator interface {
	InitStateSmartContract(name string, state util.MerklePatriciaTrieI)
}
