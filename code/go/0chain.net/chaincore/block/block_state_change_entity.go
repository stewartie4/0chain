package block

import (
	"context"
	"encoding/json"

	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"go.uber.org/zap"
)

//StateChange - an entity that captures all changes to the state by a given block
type StateChange struct {
	state.PartialState
	StateSmartContract map[string]*state.PartialState `json:"state_smart_contract,omitempty"`
	Block              string                         `json:"block"`
}

//NewBlockStateChange - if the block state computation is successfully completed, provide the changes
func NewBlockStateChange(b *Block) *StateChange {
	bsc := datastore.GetEntityMetadata("block_state_change").Instance().(*StateChange)
	bsc.Block = b.Hash
	bsc.Hash = b.ClientState.GetRoot()
	changes := b.ClientState.GetChangeCollector().GetChanges()
	bsc.Nodes = make([]util.Node, len(changes))
	for idx, change := range changes {
		bsc.Nodes[idx] = change.New
	}

	if b.SmartContextStates != nil {
		statesSC := b.SmartContextStates.GetState()
		bsc.StateSmartContract = make(map[string]*state.PartialState)
		for nameSC, stateSC := range statesSC {
			partialState := state.PartialState{
				Hash: stateSC.GetRoot(),
			}
			changes := stateSC.GetChangeCollector().GetChanges()
			partialState.Nodes = make([]util.Node, len(changes))
			for idx, change := range changes {
				partialState.Nodes[idx] = change.New
			}
			bsc.StateSmartContract[nameSC] = &partialState
		}
	}
	bsc.ComputeProperties()
	return bsc
}

var stateChangeEntityMetadata *datastore.EntityMetadataImpl

/*StateChangeProvider - a block summary instance provider */
func StateChangeProvider() datastore.Entity {
	sc := &StateChange{}
	sc.Version = "1.0"
	return sc
}

/*GetEntityMetadata - implement interface */
func (sc *StateChange) GetEntityMetadata() datastore.EntityMetadata {
	return stateChangeEntityMetadata
}

/*Read - store read */
func (sc *StateChange) Read(ctx context.Context, key datastore.Key) error {
	return sc.GetEntityMetadata().GetStore().Read(ctx, key, sc)
}

/*Write - store read */
func (sc *StateChange) Write(ctx context.Context) error {
	return sc.GetEntityMetadata().GetStore().Write(ctx, sc)
}

/*Delete - store read */
func (sc *StateChange) Delete(ctx context.Context) error {
	return sc.GetEntityMetadata().GetStore().Delete(ctx, sc)
}

/*SetupStateChange - setup the block summary entity */
func SetupStateChange(store datastore.Store) {
	stateChangeEntityMetadata = datastore.MetadataProvider()
	stateChangeEntityMetadata.Name = "block_state_change"
	stateChangeEntityMetadata.Provider = StateChangeProvider
	stateChangeEntityMetadata.Store = store
	stateChangeEntityMetadata.IDColumnName = "hash"
	datastore.RegisterEntityMetadata("block_state_change", stateChangeEntityMetadata)
}

//MarshalJSON - implement Marshaler interface
func (sc *StateChange) MarshalJSON() ([]byte, error) {
	var data = make(map[string]interface{})
	data["block"] = sc.Block
	if sc.StateSmartContract != nil {
		dataSmartContracts := make(map[string]json.RawMessage)
		for nameSC, partialState := range sc.StateSmartContract {
			partialObj := make(map[string]interface{})
			partialData, err := partialState.MartialPartialState(partialObj)
			if err != nil {
				Logger.Error("marshal json - state sc change", zap.Error(err))
				return nil, err
			}
			dataSmartContracts[nameSC] = partialData
		}
		data["state_smart_contract"] = dataSmartContracts
	}
	return sc.MartialPartialState(data)
}

//UnmarshalJSON - implement Unmarshaler interface
func (sc *StateChange) UnmarshalJSON(data []byte) error {
	var obj map[string]interface{}
	err := json.Unmarshal(data, &obj)
	if err != nil {
		Logger.Error("unmarshal json - state change", zap.Error(err))
		return err
	}
	if block, ok := obj["block"]; ok {
		if sc.Block, ok = block.(string); !ok {
			Logger.Error("unmarshal json - invalid block hash", zap.Any("obj", obj))
			return common.ErrInvalidData
		}
	} else {
		Logger.Error("unmarshal json - invalid block hash", zap.Any("obj", obj))
		return common.ErrInvalidData
	}

	if dataSmartContractsObj, ok := obj["state_smart_contract"]; ok {
		dataSmartContracts, ok := dataSmartContractsObj.(map[string]interface{})
		if !ok {
			Logger.Error("unmarshal json - invalid block state_smart_contract", zap.Any("obj", obj))
			return common.ErrInvalidData
		}
		statesSC := make(map[string]*state.PartialState, len(dataSmartContracts))
		for nameSC, partialStateObj := range dataSmartContracts {
			partialState := datastore.GetEntityMetadata("partial_state").Instance().(*state.PartialState)
			err := partialState.UnmarshalPartialState(partialStateObj.(map[string]interface{}))
			if err != nil {
				Logger.Error("unmarshal json - state sc change", zap.Error(err))
				return err
			}
			statesSC[nameSC] = partialState
		}
		if len(statesSC) > 0 {
			sc.StateSmartContract = statesSC
		}
	}

	return sc.UnmarshalPartialState(obj)
}

func (sc *StateChange) ComputeProperties() {
	for _, partialState := range sc.StateSmartContract {
		partialState.ComputeProperties()
	}
	sc.PartialState.ComputeProperties()
}

func (sc *StateChange) Validate(ctx context.Context) error {
	for _, partialState := range sc.StateSmartContract {
		if err := partialState.Validate(ctx); err != nil {
			return err
		}
	}
	return sc.PartialState.Validate(ctx)
}

//
//func (sc *StateChange) SaveState(ctx context.Context, stateDB util.NodeDB) error {
//	for _, partialState := range sc.StateSmartContract {
//		//FIXME: GET stateDB
//		//db := ""
//		if err := partialState.SaveState(ctx,stateDB); err != nil {
//			return err
//		}
//	}
//	return sc.PartialState.SaveState(ctx, stateDB)
//}
