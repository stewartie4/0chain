package statesc

import (
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/util"
	"context"
	"go.uber.org/zap"
	"log"
	"sync"

	. "0chain.net/core/logging"
)

// Errors
var (
	ErrStateSCMismatch = common.NewError("state_sc_mismatch", "Computed state smart contract hash doesn't match with the state hash of the block")
)

type SmartContractState struct {
	mutex sync.RWMutex
	Hash  map[string]util.Key                 `json:"sc_state_hash"`
	state map[string]util.MerklePatriciaTrieI `json:"-"`
}

type StateSCInitiator interface {
	InitStateSmartContract(name string, state util.MerklePatriciaTrieI)
}

var (
	StateSCDBGetter              func(name string) util.NodeDB
	StatesSCBlockInits           func(initiator StateSCInitiator)
	IsSeparateStateSmartContract func(name string) bool
)

func NewSmartContractState() *SmartContractState {
	state := &SmartContractState{
		Hash:  make(map[string]util.Key),
		state: make(map[string]util.MerklePatriciaTrieI),
	}
	return state
}

func (b *SmartContractState) GetStateSmartContract(name string) util.MerklePatriciaTrieI {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.state[name]
}

func (b *SmartContractState) GetStateSmartContractHash(name string) util.Key {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.Hash[name]
}

func (b *SmartContractState) SetStateSmartContractHashesFromRoot() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for nameSC, state := range b.state {
		log.Println("SetStateSmartContractHashesFromRoot name=", nameSC, "old hash=", b.Hash[nameSC], " state root=", state.GetRoot())
		b.Hash[nameSC] = state.GetRoot()
	}
}

func (b *SmartContractState) SetStateSmartContractHashes(nameSC string, key util.Key) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for nameSC, state := range b.state {
		log.Println("SetStateSmartContractHashesFromRoot name=", nameSC, "old hash=", b.Hash[nameSC], " state root=", state.GetRoot())
		b.state[nameSC].SetRoot(key)
		b.Hash[nameSC] = state.GetRoot()
	}
}

func (b *SmartContractState) InitStateSmartContract(name string, state util.MerklePatriciaTrieI) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.state == nil {
		b.state = make(map[string]util.MerklePatriciaTrieI)
	}
	if are, found := b.state[name]; found && are != nil {
		return
	}
	b.state[name] = state
	b.Hash[name] = state.GetRoot()
}

func (b *SmartContractState) GetState() map[string]util.MerklePatriciaTrieI {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	result := make(map[string]util.MerklePatriciaTrieI, len(b.state))
	for name, value := range b.state {
		result[name] = value
	}
	return result
}

func (b *SmartContractState) GetHash() map[string]util.Key {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	result := make(map[string]util.Key, len(b.Hash))
	for name, hash := range b.Hash {
		result[name] = hash
	}
	return result
}
/*

func (b *SmartContractState) CreateFromHash(prev *SmartContractState, version util.Sequence) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.state == nil {
		b.state = make(map[string]util.MerklePatriciaTrieI)
	}

	for name, hash := range b.Hash {
		if _, found := b.state[name]; !found {
			mndb := util.NewMemoryNodeDB()
			var pndb util.NodeDB
			if prev != nil {
				pndb = prev.GetStateSmartContract(name).GetNodeDB()
			} else {
				if state.Debug() {
					Logger.DPanic("set state db - prior state not available")
				}
				pndb = util.NewMemoryNodeDB()
			}
			ndb := util.NewLevelNodeDB(mndb, pndb, false)
			b.state[name] = util.NewMerklePatriciaTrie(ndb, version)
			b.state[name].SetRoot(hash)
		}
	}
}
*/

func (b *SmartContractState) CreateState(prev *SmartContractState, version util.Sequence) {
	if b == nil {
		return
	}
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.state == nil {
		b.state = make(map[string]util.MerklePatriciaTrieI)
	}

	hashes := prev.GetHash()
	prevState := prev.GetState()

	for name, hash := range hashes {
		_, found := b.state[name]
		if !found {
			statePrev := prevState[name]
			var pndb util.NodeDB
			if statePrev == nil {
				if state.Debug() {
					Logger.DPanic("set sc state db - prior state not available")
				} else {
					pndb = util.NewMemoryNodeDB()
					//pndb = StateSCGetter(name).GetNodeDB()
				}
			} else {
				pndb = statePrev.GetNodeDB()
			}
			tdb := util.NewLevelNodeDB(util.NewMemoryNodeDB(), pndb, false)
			b.state[name] = util.NewMerklePatriciaTrie(tdb, version)
			b.Hash[name] = hash
			b.state[name].SetRoot(hash)
		} else {
			//log.Println("replace? root old=", foundState.GetRoot(), " new root=", hash)
			//foundState.SetRoot(hash)
		}
	}
}

func (b *SmartContractState) InitState(version util.Sequence) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.state == nil {
		b.state = make(map[string]util.MerklePatriciaTrieI)
	}
	for name, hash := range b.Hash {
		//_, found := b.state[name]
		//if !found {
		pndb := StateSCDBGetter(name)
		tdb := util.NewLevelNodeDB(util.NewMemoryNodeDB(), pndb, false)
		b.state[name] = util.NewMerklePatriciaTrie(tdb, version)
		//b.Hash[name] = hash
		b.state[name].SetRoot(hash)
		//}
		/* else {
			foundState.SetRoot(hash)
		}*/
	}
}

func (b *SmartContractState) CreateStateForSC(pndb util.NodeDB, name string, version util.Sequence) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	mndb := util.NewMemoryNodeDB()
	ndb := util.NewLevelNodeDB(mndb, pndb, false)
	if b.state == nil {
		b.state = make(map[string]util.MerklePatriciaTrieI)
	}
	b.state[name] = util.NewMerklePatriciaTrie(ndb, version)
}

func (b *SmartContractState) Validate(ctx context.Context) error {
	b.mutex.RLock()
	stateHashes := make(map[string]util.Key, len(b.state))
	for k, v := range b.state {
		stateHashes[k] = v.GetRoot()
	}
	b.mutex.RUnlock()
	hashes := b.GetHash()

	for name, hash := range hashes {
		stateHash, found := stateHashes[name]
		if !found || !hash.EqualTo(stateHash) {
			Logger.Error("compute state sc - state sc hash mismatch",
				zap.String("name_sc", name),
				zap.String("state_sc_hash", util.ToHex(hash)), zap.String("computed_state_hash", util.ToHex(stateHash)))
			return ErrStateSCMismatch
		}
	}
	return nil
}
