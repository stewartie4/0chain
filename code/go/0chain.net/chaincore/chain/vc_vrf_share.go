package chain

import (
	"context"
	"fmt"

	"0chain.net/chaincore/node"
	"0chain.net/core/datastore"
)

// ToDo: check to refactor round.vrfshare and merge this with that

//VCVRFShare - a VRF share for view change process
type VCVRFShare struct {
	datastore.IDField
	MagicBlockNumber int64  `json:"magic_block_number"`
	Share            string `json:"share"`
	party            *node.Node
}

var vcVrfsEntityMetadata *datastore.EntityMetadataImpl

/*GetEntityMetadata - implementing the interface */
func (vcVrfs *VCVRFShare) GetEntityMetadata() datastore.EntityMetadata {
	return vcVrfsEntityMetadata
}

/*GetKey - returns the round number as the key */
func (vcVrfs *VCVRFShare) GetKey() datastore.Key {
	return datastore.ToKey(fmt.Sprintf("%v", vcVrfs.MagicBlockNumber))
}

/*Read - read MB entity from store */
func (vcVrfs *VCVRFShare) Read(ctx context.Context, key datastore.Key) error {
	return vcVrfs.GetEntityMetadata().GetStore().Read(ctx, key, vcVrfs)
}

/*Write - write MB entity to store */
func (vcVrfs *VCVRFShare) Write(ctx context.Context) error {
	return vcVrfs.GetEntityMetadata().GetStore().Write(ctx, vcVrfs)
}

/*Delete - delete round entity from store */
func (vcVrfs *VCVRFShare) Delete(ctx context.Context) error {
	return vcVrfs.GetEntityMetadata().GetStore().Delete(ctx, vcVrfs)
}

/*VCVRFShareProvider - entity provider for client object */
func VCVRFShareProvider() datastore.Entity {
	vcVrfs := &VCVRFShare{}
	return vcVrfs
}

/*SetupVCVRFShareEntity - setup the entity */
func SetupVCVRFShareEntity() {
	//func SetupVCVRFShareEntity(store datastore.Store) {
	vcVrfsEntityMetadata = datastore.MetadataProvider()
	vcVrfsEntityMetadata.Name = "vcvrfs"
	vcVrfsEntityMetadata.Provider = VCVRFShareProvider
	//vcVrfsEntityMetadata.Store = store
	vcVrfsEntityMetadata.IDColumnName = "magic_block_number"
	datastore.RegisterEntityMetadata("vcvrfs", vcVrfsEntityMetadata)
}

//GetMagicBlockNumber - return the magic block number associated with this vrf share
func (vcVrfs *VCVRFShare) GetMagicBlockNumber() int64 {
	return vcVrfs.MagicBlockNumber
}

//SetParty - set the party contributing this vrf share
func (vcVrfs *VCVRFShare) SetParty(party *node.Node) {
	vcVrfs.party = party
}

//GetParty - get the party contributing this vrf share
func (vcVrfs *VCVRFShare) GetParty() *node.Node {
	return vcVrfs.party
}
