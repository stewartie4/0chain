package bls

/* DKG implementation */

import (
	"context"
	"fmt"
	"strconv"

	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	. "0chain.net/core/logging"
	"github.com/herumi/bls/ffi/go/bls"
	"go.uber.org/zap"
)

/*DKG - to manage DKG process */
type DKG struct {
	T      int
	N      int
	secKey Key
	mSec   []Key

	secSharesMap      map[PartyID]Key
	receivedSecShares []Key
	GpPubKey          GroupPublicKey

	SecKeyShareGroup Key
	ID               PartyID
	MagicBlockNumber int64
	RandomSeedVC     int64
	Vvec             []bls.PublicKey
	GroupVvec        []bls.PublicKey
}

/* init -  To initialize a point on the curve */
func init() {
	err := bls.Init(bls.CurveFp254BNb)
	if err != nil {
		panic(fmt.Errorf("bls initialization error: %v", err))
	}
}

func CopyDKG(in DKG) DKG {
	out := DKG{}
	out.T = in.T
	out.N = in.N
	out.secKey = in.secKey
	out.mSec = make([]Key, 0, in.T)

	for _, t := range in.mSec {
		out.mSec = append(out.mSec, t)
	}

	out.secSharesMap = make(map[PartyID]Key, in.N)

	for i, n := range in.secSharesMap {
		out.secSharesMap[i] = n
	}

	out.receivedSecShares = make([]Key, 0, in.N)
	for _, s := range in.receivedSecShares {
		out.receivedSecShares = append(out.receivedSecShares, s)
	}

	out.GpPubKey = in.GpPubKey
	out.SecKeyShareGroup = in.SecKeyShareGroup
	out.ID = in.ID
	out.MagicBlockNumber = in.MagicBlockNumber
	out.RandomSeedVC = in.RandomSeedVC

	out.Vvec = make([]bls.PublicKey, 0, len(in.Vvec))
	for _, v := range in.Vvec {
		out.Vvec = append(out.Vvec, v)
	}

	out.GroupVvec = make([]bls.PublicKey, 0, in.T)

	for i, gv := range in.GroupVvec {
		VRFLogger.Info("Appending gvvec", zap.Int("index", i))
		out.GroupVvec = append(out.GroupVvec, gv)
	}

	VRFLogger.Info("Len of gvvec inside", zap.Int("len", len(out.GroupVvec)))
	return out
}

/*MakeDKG - to create a dkg object */
func MakeDKG(t, n int, magicBlockNumber int64) DKG {
	dkg := DKG{
		T:                 t,
		N:                 n,
		secKey:            Key{},
		mSec:              make([]Key, t),
		secSharesMap:      make(map[PartyID]Key, n),
		receivedSecShares: make([]Key, n),
		GpPubKey:          GroupPublicKey{},
		SecKeyShareGroup:  Key{},
		ID:                PartyID{},
		MagicBlockNumber:  magicBlockNumber,
	}

	dkg.secKey.SetByCSPRNG()

	dkg.mSec = dkg.secKey.GetMasterSecretKey(t)

	return dkg
}

/*ComputeIDdkgS - to create an ID of party of type PartyID */
func ComputeIDdkgS(minerID string) PartyID {

	//TODO: minerID here is the index. Change it to miner ID. Neha has fix for this
	var forID PartyID
	err := forID.SetDecString(minerID)
	if err != nil {
		fmt.Printf("Error while computing ID %s\n", forID.GetHexString())
	}

	return forID
}

/*ComputeIDdkg - to create an ID of party of type PartyID */
func ComputeIDdkg(minerID int) (PartyID, error) {

	//TODO: minerID here is the index. Change it to miner ID. Neha has fix for this
	var forID PartyID
	err := forID.SetDecString(strconv.Itoa(minerID + 1))
	if err != nil {
		fmt.Printf("Error while computing ID %s\n", forID.GetHexString())
		return forID, err
	}

	return forID, nil
}

/*ComputeDKGKeyShare - Derive the share for each miner through polynomial substitution method */
func (dkg *DKG) ComputeDKGKeyShare(forID PartyID) (Key, error) {

	var secVec Key
	err := secVec.Set(dkg.mSec, &forID)
	if err != nil {
		return Key{}, nil
	}
	dkg.secSharesMap[forID] = secVec

	return secVec, nil
}

/*GetKeyShareForOther - Get the DKGKeyShare for this Miner specified by the PartyID */
func (dkg *DKG) GetKeyShareForOther(to PartyID) *DKGKeyShare {

	indivShare, ok := dkg.secSharesMap[to]
	if !ok {
		fmt.Println("Share not derived for the miner")
	}

	dShare := &DKGKeyShare{m: indivShare}

	return dShare
}

/*AggregateShares - Each party aggregates the received shares from other party which is calculated for that party */
func (dkg *DKG) AggregateShares() {
	var sec Key

	for i := 0; i < len(dkg.receivedSecShares); i++ {
		sec.Add(&dkg.receivedSecShares[i])
	}
	dkg.SecKeyShareGroup = sec

}

// SetRandomSeedVC set the view change randomseed after it is calculated
func (dkg *DKG) SetRandomSeedVC(vcrs int64) {
	dkg.RandomSeedVC = vcrs
}

// SaveVvec call this once DKG shares are generated
func (dkg *DKG) SaveVvec() int {
	dkg.Vvec = bls.GetMasterPublicKey(dkg.mSec)

	return len(dkg.Vvec)
}

// GetVvec --
func (dkg *DKG) GetVvec() []bls.PublicKey {
	return dkg.Vvec
}

// GetVvecAsString --converts public key to string for messaging purposes
func (dkg *DKG) GetVvecAsString() []string {
	vvecStr := make([]string, 0, len(dkg.Vvec))

	for _, v := range dkg.Vvec {
		vvecStr = append(vvecStr, v.GetHexString())
	}
	return vvecStr
}

// GetVvecFromString converts vvec from incoming DKG to publick keys
func GetVvecFromString(vvecStr []string) []bls.PublicKey {
	vvecpk := make([]bls.PublicKey, 0, len(vvecStr))

	for _, v := range vvecStr {
		var pub bls.PublicKey
		pub.SetHexString(v)
		vvecpk = append(vvecpk, pub)
	}
	return vvecpk
}

// DKGSummary DBObject to store
type DKGSummary struct {
	datastore.IDField
	MagicBlockNumber  int64  `json:"magic_block_number"`
	SecretKeyGroupStr string `json:"secret_key_group_str"`
	RandomSeedVC      int64  `json:"random_seed_vc"`
}

var dkgSummaryMetadata *datastore.EntityMetadataImpl

//GetEntityMetadata entity metadata for DKGSummary
func (dkgSummary *DKGSummary) GetEntityMetadata() datastore.EntityMetadata {
	return dkgSummaryMetadata
}

/*GetKey - returns the MagicBlock number as the key */
func (dkgSummary *DKGSummary) GetKey() datastore.Key {
	return datastore.ToKey(fmt.Sprintf("%v", dkgSummary.MagicBlockNumber))
}

//DKGSummaryProvider the provider for DKG Summary
func DKGSummaryProvider() datastore.Entity {
	dkgSummary := &DKGSummary{}
	return dkgSummary
}

//SetupDKGSummary DKG Summary definition
func SetupDKGSummary(store datastore.Store) {
	dkgSummaryMetadata = datastore.MetadataProvider()
	dkgSummaryMetadata.Name = "dkgsummary"
	dkgSummaryMetadata.DB = "dkgsummarydb"
	dkgSummaryMetadata.Store = store
	dkgSummaryMetadata.Provider = DKGSummaryProvider
	dkgSummaryMetadata.IDColumnName = "magic_block_number"
	datastore.RegisterEntityMetadata("dkgsummary", dkgSummaryMetadata)
}

// SetupDKGDB DKG DB store setup
func SetupDKGDB() {
	db, err := ememorystore.CreateDB("data/rocksdb/dkg")
	if err != nil {
		panic(err)
	}
	ememorystore.AddPool("dkgsummarydb", db)
}

func (dkgSummary *DKGSummary) Read(ctx context.Context, key string) error {
	return dkgSummary.GetEntityMetadata().GetStore().Read(ctx, key, dkgSummary)
}

func (dkgSummary *DKGSummary) Write(ctx context.Context) error {
	return dkgSummary.GetEntityMetadata().GetStore().Write(ctx, dkgSummary)
}

//GetDKGSummary create DKG Summary with ViewChange support
func (dkg *DKG) GetDKGSummary() *DKGSummary {
	dkgSummary := &DKGSummary{
		SecretKeyGroupStr: dkg.SecKeyShareGroup.GetHexString(),
		MagicBlockNumber:  dkg.MagicBlockNumber,
		RandomSeedVC:      dkg.RandomSeedVC,
	}
	return dkgSummary
}
