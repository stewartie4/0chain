package miner

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"time"

	metrics "github.com/rcrowley/go-metrics"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/encryption"
	. "0chain.net/core/logging"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// ////////////  BLS-DKG Related stuff  /////////////////////

var dgVrf bls.DKG       // DKG for regular VRF operations
var dgVc bls.DKG        // DKG for View Change operations
var bsVrf bls.SimpleBLS // BLS for regular
var bsVc bls.SimpleBLS  //BLS for View Change operations
var recShares []string
var recSharesMap map[int]string
var recDkgSharesMap map[int]*bls.Dkg
var minerShares map[string]bls.Key
var currRound int64
var isDkgEnabled bool
var k, n int

var vrfTimer metrics.Timer // VRF gen-to-sync timer

func init() {
	vrfTimer = metrics.GetOrRegisterTimer("vrf_time", nil)
}

/*DkgDoneStatus an indicator that DKG (i.e., DKG+VcVRF) status. If false,
DKG is running, if true, it is not running (done/cancel)
*/
var DkgDoneStatus bool //ToDo: Should we manage access to it as it is being used in different threads very frequently?
var selfInd int

var mutex = &sync.RWMutex{}

// IsDkgDone sets DkgDoneStatus
func IsDkgDone() bool {
	//ToDo: Lock
	return DkgDoneStatus
}

// SetDkgDone sets DkgDoneStatus to Done. 0 is false 1 is true. May change to int
func SetDkgDone(status int) {
	//ToDo Lock
	if status == 0 {
		DkgDoneStatus = false
	} else {
		DkgDoneStatus = true
		recSharesMap = nil
		recDkgSharesMap = nil
	}
}

//CancelViewChange cancels a view change
func (mc *Chain) CancelViewChange(ctx context.Context) {
	if IsDkgDone() {
		Logger.Info("DKG is already done. Canceling the cancel")
		return
	}

	currMgc := mc.GetCurrentMagicBlock()
	verifyMBStore(ctx, currMgc.TypeOfMB, "Printing magicblock before cancel")
	currMgc.EstimatedLastRound = chain.CalcLastRound(currMgc.EstimatedLastRound+1, mc.MagicBlockLife)
	UpdateMagicBlock(ctx, currMgc)
	verifyMBStore(ctx, currMgc.TypeOfMB, "Printing magicblock after cancel")
	SetDkgDone(1)
	dgVc = dgVrf
	nextMgc := mc.GetNextMagicBlock()

	if nextMgc != nil {
		//ToDo: Store the next magic block also
		Logger.Info("nextMgc is not nil. Deleting it in cancel", zap.Int64("nextMagicNum", nextMgc.GetMagicBlockNumber()))
		DeleteMagicBlock(ctx, nextMgc)
	}

	Logger.Info("Canceled Viewchange ", zap.Int64("mbNum", currMgc.GetMagicBlockNumber()), zap.Int64("lastRoundNum", currMgc.EstimatedLastRound))
}

// AdjustLastRound  Calc and reset the last round and store
func (mc *Chain) AdjustLastRound(ctx context.Context, mb *chain.MagicBlock, roundNum int64) {
	wasLrn := mb.EstimatedLastRound
	mb.EstimatedLastRound = chain.CalcLastRound(roundNum, mc.MagicBlockLife)
	Logger.Info("Adjusting last round", zap.Int64("lr_was", wasLrn), zap.Int64("changedLrn", mb.EstimatedLastRound))
	UpdateMagicBlock(ctx, mb)
}

// StartViewChange starts a viewchange for next magicblock
func (mc *Chain) StartViewChange(ctx context.Context, currMgc *chain.MagicBlock) {
	Logger.Info("starting viewchange", zap.Int64("currMgcNumber", currMgc.GetMagicBlockNumber()))
	SetDkgDone(0)
	nextMgc, err := currMgc.SetupNextMagicBlock()
	if err != nil {
		Logger.Error("Error in starting viewchange", zap.Int64("currMBNum", currMgc.GetMagicBlockNumber()))

		SetDkgDone(0) //Set it as false, so that it can be canceled
		mc.CancelViewChange(ctx)
		return
	}
	nextMgc.ActiveSetSharders.OneTimeStatusMonitor(ctx)
	mc.SetNextMagicBlock(nextMgc)
	//StoreMagicBlock(ctx, nextMgc)
	StartMbDKG(ctx, nextMgc)
}

// SwitchToNextView Promotes next magic block as current including ActiveSet Changes
func (mc *Chain) SwitchToNextView(ctx context.Context, currMgc *chain.MagicBlock) {

	nmb := mc.GetNextMagicBlock()
	if nmb != nil {
		//dg.SecKeyShareGroup.SetHexString(nmb.SecretKeyGroupStr)
		currMgc.ActiveSetMiners.LogPool("ActiveSetPromote_Before")
		mc.PromoteMagicBlockToCurr(nmb)
		mc.InitChainActiveSetFromMagicBlock(nmb)
		dgVrf = dgVc
		Logger.Info("Promoted next to curr", zap.Int64("current_was", currMgc.GetMagicBlockNumber()), zap.Int64("current_is", mc.GetCurrentMagicBlock().GetMagicBlockNumber()),
			zap.Any("mbtype", nmb.TypeOfMB), zap.Int64("round_num", mc.CurrentRound))
		nmb.ActiveSetMiners.LogPool("ActiveSetPromote_After")
		err := SaveNextAsCurrMagicBlock(ctx, currMgc, nmb)
		if err != nil {
			Logger.DPanic("failed to promote", zap.Int64("currmbnum", currMgc.GetMagicBlockNumber()), zap.Error(err))
			return
		}

		//ToDo: remove this once we know to restart.
		if !verifyMBStore(ctx, nmb.TypeOfMB, "Promoting nextMB to currMB") {
			Logger.DPanic("Failed to store MagicBlock.")
			return
		}

	}

}

// ToDo: remove this method. This is for verifying magic block is stored.
func verifyMBStore(ctx context.Context, mbtype chain.MBType, mesg string) bool {
	xmb, er := GetMagicBlockFromStore(ctx, mbtype)
	if er != nil {
		Logger.Error("Failed to get magicblock from store", zap.Error(er))
		return false
	}
	if xmb != nil {
		Logger.Info(mesg, zap.Any("mbtype", xmb.TypeOfMB), zap.Int64("mbnum", xmb.GetMagicBlockNumber()),
			zap.Int("len_of_allminers", len(xmb.AllMiners.Nodes)), zap.Int("len_of_dkgsetminers", len(xmb.DKGSetMiners.Nodes)),
			zap.Int("len_of_activesetminers", len(xmb.ActiveSetMiners.Nodes)), zap.Int("len_of_allsharders", len(xmb.AllSharders.Nodes)),
			zap.Int64("start_round", xmb.StartingRound), zap.Int64("last_round", xmb.EstimatedLastRound), zap.Int64("random_seed", xmb.RandomSeed),
			zap.Int64("prev_random_seed", xmb.PrevRandomSeed))
	}
	return true
}

// LoadNodesFromDB load nodes information from DB if a magic block exists
func (mc *Chain) LoadNodesFromDB(ctx context.Context) bool {
	mb, err := GetMagicBlockFromStore(ctx, chain.CURR)
	if err != nil {
		Logger.Info("err in getting magicblock from store. Need to start afresh", zap.Error(err))
		return false
	}
	if mb != nil {
		verifyMBStore(ctx, chain.CURR, "verifying mbstore on launch")
		Logger.Info("found magicblock")
		newMb, err := mb.SetupAndInitMagicBlock()
		if err != nil {
			Logger.Error("Error in launching miner from stored data", zap.Error(err))
			return false
		}
		mc.CurrMagicBlock = newMb
		mc.InitChainActiveSetFromMagicBlock(newMb)
		miners := newMb.ActiveSetMiners
		isDkgEnabled = config.DevConfiguration.IsDkgEnabled
		thresholdByCount := viper.GetInt("server_chain.block.consensus.threshold_by_count")
		k = int(math.Ceil((float64(thresholdByCount) / 100) * float64(miners.Size())))
		n = miners.Size()
		recSharesMap = nil
		recDkgSharesMap = nil
		//self := node.GetSelfNode(ctx)

		selfNode := miners.GetNodeFromGNode(node.GetSelfNode(ctx).GNode)
		selfInd = selfNode.SetIndex

		dgVc = bls.MakeDKG(k, n, newMb.GetMagicBlockNumber())
		dgVc.SetRandomSeedVC(newMb.RandomSeed)
		dgVc.SecKeyShareGroup.SetHexString(newMb.SecretKeyGroupStr)
		bsVc = bls.MakeSimpleBLS(&dgVc)
		dgVrf = dgVc

		nmb, err := GetMagicBlockFromStore(ctx, chain.NEXT)
		if err == nil {
			mc.NextMagicBlock = nmb
		}
		return true
	}
	return false

}

//LaunchMiner call this at start or restart of miner
func (mc *Chain) LaunchMiner(ctx context.Context) bool {
	mb, err := GetMagicBlockFromStore(ctx, chain.CURR)
	if err != nil {
		Logger.Info("err in getting magicblock from store. Starting afresh", zap.Error(err))
		StartMbDKG(ctx, mc.GetCurrentMagicBlock())
		return true
	}
	if mb != nil {
		//MB is already loaded
		Logger.Info("LaunchMiner Dkg done")
		SetDkgDone(1)
		mc.SetupChainWorkers(ctx)
		go startProtocol()

		return true
	}
	return false
}

//StartMbDKG  starting DKG from MagicBlock
func StartMbDKG(ctx context.Context, mgc *chain.MagicBlock) {
	miners, err := mgc.GetComputedDKGSet()

	if err != nil {
		Logger.Panic("Error in finding miners for DKG", zap.Error(err))
	}
	if miners == nil {
		Logger.Panic("Could not get miners for DKG")
	}
	Logger.Info("Miners size", zap.Int("Miners", len(miners.Nodes)))
	isDkgEnabled = config.DevConfiguration.IsDkgEnabled
	thresholdByCount := viper.GetInt("server_chain.block.consensus.threshold_by_count")
	k = int(math.Ceil((float64(thresholdByCount) / 100) * float64(miners.Size())))
	n = miners.Size()
	recSharesMap = nil
	recDkgSharesMap = nil
	SetDkgDone(0)

	selfNode := miners.GetNodeFromGNode(node.GetSelfNode(ctx).GNode)
	selfInd = selfNode.SetIndex
	Logger.Info("DKG Setup", zap.Int("selfindex", selfInd), zap.Int("K", k), zap.Int("N", n), zap.Bool("DKG Enabled", isDkgEnabled))

	if isDkgEnabled {
		dgVc = bls.MakeDKG(k, n, mgc.GetMagicBlockNumber())
		waitForMbNetworkToBeReady(ctx, mgc)
		if IsDkgDone() {
			Logger.Info("Cannot continue with DKG. It seems canceled. Returning")
			return
		}
		Logger.Info("Starting DKG...")

		minerShares = make(map[string]bls.Key, len(miners.Nodes))
		var mySecShare string

		for _, node := range miners.Nodes {
			forID, err := bls.ComputeIDdkg(node.SetIndex)
			if err != nil {
				Logger.Error("Error while computeDKG", zap.Int("minerIndex", node.SetIndex), zap.Error(err))
				Logger.Panic("Error in computeDKG")
			}
			dgVc.ID = forID

			secShare, _ := dgVc.ComputeDKGKeyShare(forID)

			Logger.Debug("ComputeDKGKeyShare ", zap.Any("dgVC.ID", dgVc.ID.GetDecString()), zap.String("secShare", secShare.GetDecString()), zap.Int("minerIndex", node.SetIndex))
			minerShares[node.GetKey()] = secShare
			if selfNode.SetIndex == node.SetIndex {
				mySecShare = secShare.GetDecString()
				recShares = append(recShares, mySecShare)
				addToRecSharesMap(selfNode.SetIndex, mySecShare)

			}
		}
		dgVc.SaveVvec()
		vvecStr := dgVc.GetVvecAsString()

		myDg := &bls.Dkg{
			Share: mySecShare,
			Vvec:  vvecStr,
		}
		Logger.Info("Before sending vvec on myDg", zap.Int("len_of_vvec", len(vvecStr)), zap.Any("dgvc.ID", dgVc.ID.GetDecString()), zap.Any("share_at_dgvc[0]", myDg.Vvec[0]))

		addToRecDkgSharesMap(selfNode.SetIndex, myDg)
		WaitForMbDKGShares(mgc)

	} else {
		Logger.Info("DKG is not enabled. So, starting protocol")
		SetDkgDone(1)
		go startProtocol()
	}

}

// RunVRFForVC run a VRF on the DKG once to rank the miners
func (mc *Chain) RunVRFForVC(ctx context.Context, mb *chain.MagicBlock) {
	vcVrfs := &chain.VCVRFShare{}
	vcVrfs.MagicBlockNumber = mb.GetMagicBlockNumber()
	vcVrfs.Share, mb.SignedBlsMessage = GetBlsShareForVC(mb)
	n := mb.DKGSetMiners.GetNodeFromGNode(node.Self.GNode)
	vcVrfs.SetParty(n)
	mb.VcVrfShare = vcVrfs
	ind := n.SetIndex
	if !VerifySigShares(dgVc, vcVrfs.Share, ind, mb.SignedBlsMessage) {
		Logger.Info("vcVrfs not verified", zap.Int("sender", ind), zap.String("signedMessage", mb.SignedBlsMessage), zap.String("share", vcVrfs.Share))
		//Logger.Panic("failed to verify")
	} else {
		Logger.Info("success in verifying vcVrfs ", zap.Int("sender", ind), zap.String("signedMessage", mb.SignedBlsMessage), zap.String("share", vcVrfs.Share))
	}
	AppendVCVRFShares(ctx, n.ID, vcVrfs)
	//vcVrfs.SetKey(datastore.ToKey(fmt.Sprintf("%v", vcVrfs.MagicBlockNumber)))
	err := SendMbVcVrfShare(mb, vcVrfs)
	if err != nil {
		Logger.Error("Error while sending vcVrfShare", zap.Error(err))
	}
}

// GetMagicBlockFromStore reads the magicblock from db, if not exists, returns error
func GetMagicBlockFromStore(ctx context.Context, mbtype chain.MBType) (*chain.MagicBlock, error) {
	cmb := datastore.GetEntity("magicblock").(*chain.MagicBlock)
	cmb.TypeOfMB = mbtype
	mbMetadata := cmb.GetEntityMetadata()
	dctx := ememorystore.WithEntityConnection(ctx, mbMetadata)
	defer ememorystore.Close(dctx)
	err := cmb.Read(dctx, cmb.GetKey())
	return cmb, err
}

// SaveNextAsCurrMagicBlock saves next magic block as current
func SaveNextAsCurrMagicBlock(ctx context.Context, cmb *chain.MagicBlock, nmb *chain.MagicBlock) error {
	mbMetadata := cmb.GetEntityMetadata()
	dctx := ememorystore.WithEntityConnection(ctx, mbMetadata)
	defer ememorystore.Close(dctx)
	err := cmb.Delete(dctx)
	if err != nil {
		return err
	}
	err = nmb.Write(dctx)
	if err != nil {
		return err
	}
	con := ememorystore.GetEntityCon(dctx, mbMetadata)
	err = con.Commit()
	if err != nil {
		return err
	}
	return nil
}

// DeleteMagicBlock deletes the given magicblock from the db
func DeleteMagicBlock(ctx context.Context, cmb *chain.MagicBlock) error {
	mbMetadata := cmb.GetEntityMetadata()
	dctx := ememorystore.WithEntityConnection(ctx, mbMetadata)
	defer ememorystore.Close(dctx)
	err := cmb.Delete(dctx)
	if err != nil {
		return err
	}

	con := ememorystore.GetEntityCon(dctx, mbMetadata)
	err = con.Commit()
	if err != nil {
		return err
	}
	return nil
}

// UpdateMagicBlock replaces old magic block with the new one atomically
func UpdateMagicBlock(ctx context.Context, cmb *chain.MagicBlock) error {
	mbMetadata := cmb.GetEntityMetadata()
	dctx := ememorystore.WithEntityConnection(ctx, mbMetadata)
	defer ememorystore.Close(dctx)
	err := cmb.Delete(dctx)
	if err != nil {
		Logger.Info("error while deleting in updateMagicBlock. Ignoring...", zap.Error(err))
	}
	err = cmb.Write(dctx)
	if err != nil {
		return err
	}
	con := ememorystore.GetEntityCon(dctx, mbMetadata)
	err = con.Commit()
	if err != nil {
		return err
	}
	return nil
}

//StoreMagicBlock Stores a given magic block in DB
func StoreMagicBlock(ctx context.Context, mb *chain.MagicBlock) error {
	mbMetadata := mb.GetEntityMetadata()
	dctx := ememorystore.WithEntityConnection(ctx, mbMetadata)
	defer ememorystore.Close(dctx)
	err := mb.Write(dctx)
	if err != nil {
		return err
	}
	con := ememorystore.GetEntityCon(dctx, mbMetadata)
	err = con.Commit()
	if err != nil {
		return err
	}
	return nil
}

// WaitForDkgToBeDone is a blocking function waits till DKG process is done if dkg is enabled
func WaitForDkgToBeDone(ctx context.Context) {
	if isDkgEnabled {
		ticker := time.NewTicker(5 * chain.DELTA)
		defer ticker.Stop()

		for ts := range ticker.C {
			if IsDkgDone() {
				Logger.Info("WaitForDkgToBeDone is over.")
				break
			} else {
				Logger.Info("Waiting for DKG process to be over.", zap.Time("ts", ts))
			}
		}
	}
}

func isNetworkReadyForDKG() bool {
	mc := GetMinerChain()
	if isDkgEnabled {
		return mc.AreAllNodesActive()
	}

	return mc.CanStartNetwork()

}

func waitForMbNetworkToBeReady(ctx context.Context, mgc *chain.MagicBlock) {

	miners := mgc.DKGSetMiners
	Logger.Info("Started waiting for MBNetwork to be ready ", zap.Int("len_dkgset", len(miners.Nodes)))
	//go miners.DKGMonitor(ctx)
	Logger.Info("DKGMonitor started ", zap.Int("len_dkgset", len(miners.Nodes)), zap.String("ticker_time", fmt.Sprintf("%v", (5*chain.DELTA))))

	if !mgc.IsMbReadyForDKG() {
		ticker := time.NewTicker(5 * chain.DELTA)
		defer ticker.Stop()
		for ts := range ticker.C {
			Logger.Info("MB Ready Ticking ", zap.Int("len_dkgset", len(miners.Nodes)), zap.String("ticker_time", fmt.Sprintf("%v", (5*chain.DELTA))))

			if IsDkgDone() {
				miners.CancelDKGMonitor()
				Logger.Info("Dkg Cancelled. returning")
				return
			}
			active := miners.GetActiveCount()
			if !isDkgEnabled {
				Logger.Info("waiting for sufficient active nodes", zap.Time("ts", ts), zap.Int("active", active))
			} else {
				Logger.Info("waiting for all nodes to be active", zap.Time("ts", ts), zap.Int("active", active))
			}
			if mgc.IsMbReadyForDKG() {
				miners.CancelDKGMonitor()
				break
			}
		}
	} else {
		Logger.Info(" MBNetwork already ready ", zap.Int("len_dkgset", len(miners.Nodes)))

	}
}

func waitForNetworkToBeReadyForBls(ctx context.Context) {
	mc := GetMinerChain()

	if !mc.CanStartNetwork() {
		ticker := time.NewTicker(5 * chain.DELTA)
		for ts := range ticker.C {
			active := mc.Miners.GetActiveCount()
			Logger.Info("waiting for sufficient active nodes", zap.Time("ts", ts), zap.Int("have", active), zap.Int("need", k))
			if mc.CanStartNetwork() {
				break
			}
		}
	}
}

func waitForNetworkToBeReadyForDKG(ctx context.Context) {

	mc := GetMinerChain()

	if !isNetworkReadyForDKG() {
		ticker := time.NewTicker(5 * chain.DELTA)
		for ts := range ticker.C {
			active := mc.Miners.GetActiveCount()
			if !isDkgEnabled {
				Logger.Info("waiting for sufficient active nodes", zap.Time("ts", ts), zap.Int("active", active))
			} else {
				Logger.Info("waiting for all nodes to be active", zap.Time("ts", ts), zap.Int("active", active))
			}
			if isNetworkReadyForDKG() {
				break
			}
		}
	}
}

func sendMbDKG(mgc *chain.MagicBlock) {

	miners := mgc.DKGSetMiners

	shuffledNodes := miners.GetRandomNodes(miners.Size())

	for _, n := range shuffledNodes {

		if n != nil {
			if selfInd == n.SetIndex {
				//we do not want to send message to ourselves.
				continue
			}
			//ToDo: Optimization Instead of sending, asking for DKG share is better.
			err := SendMbDKGShare(n, mgc)
			if err != nil {
				Logger.Error("DKG Failed sending DKG share", zap.Int("idx", n.SetIndex), zap.Error(err))
			}
		} else {
			Logger.Info("DKG Error in getting node for ", zap.Int("idx", n.SetIndex))
		}
	}

}

// SendMbVcVrfShare sends VCVRFShare to DKGSet miners
func SendMbVcVrfShare(mgc *chain.MagicBlock, vcVrfs *chain.VCVRFShare) error {
	if !isDkgEnabled {
		Logger.Debug("DKG not enabled. Not sending shares")
		return nil
	}
	miners := mgc.DKGSetMiners

	shuffledNodes := miners.GetRandomNodes(miners.Size())
	var err error
	for _, n := range shuffledNodes {

		if n != nil {
			if selfInd == n.SetIndex {
				//we do not want to send message to ourselves.
				continue
			}

			_, err = miners.SendTo(VCVRFSender(vcVrfs), n.GetKey())
			if err != nil {
				Logger.Error("DKG Failed sending DKG share", zap.Int("idx", n.SetIndex), zap.Error(err))
				break
			}
		} else {
			Logger.Info("DKG Error in getting node for ", zap.Int("idx", n.SetIndex))
		}
	}
	//Logger.Debug("sending DKG share", zap.Int("idx", n.SetIndex), zap.Any("share", dkg.Share))

	return err
}

//SendMbDKGShare  sends MB type DKG share to all DKGSet miners
func SendMbDKGShare(n *node.Node, mgc *chain.MagicBlock) error {
	if !isDkgEnabled {
		Logger.Debug("DKG not enabled. Not sending shares")
		return nil
	}
	miners := mgc.DKGSetMiners
	vvecStr := dgVc.GetVvecAsString()

	secShare := minerShares[n.GetKey()]
	dkg := &bls.Dkg{
		Share: secShare.GetDecString(),
		Vvec:  vvecStr,
	}
	dkg.SetKey(datastore.ToKey("1"))
	Logger.Debug("sending DKG share", zap.Any("recipient", n.GetKey()), zap.Any("len_of_vvec_sent", len(dkg.Vvec)), zap.Any("share", dkg.Share))
	_, err := miners.SendTo(DKGShareSender(dkg), n.GetKey())
	return err
}

// WaitForMbDKGShares blocks until DKG process done
func WaitForMbDKGShares(mgc *chain.MagicBlock) bool {

	if !HasAllDKGSharesReceived() {
		ticker := time.NewTicker(5 * chain.DELTA)
		defer ticker.Stop()
		for ts := range ticker.C {
			if HasAllDKGSharesReceived() {
				Logger.Debug("Received sufficient DKG Shares. Sending DKG one moretime and going quiet", zap.Time("ts", ts))
				sendMbDKG(mgc)
				break
			} else if IsDkgDone() {
				Logger.Info("DKG Cancelled.")
				return false
			}
			Logger.Info("waiting for sufficient DKG Shares", zap.Int("Received so far", len(recSharesMap)), zap.Time("ts", ts))
			sendMbDKG(mgc)

		}
	}

	return true

}

/*HasAllDKGSharesReceived returns true if all shares are received */
func HasAllDKGSharesReceived() bool {
	if !isDkgEnabled {
		Logger.Info("DKG not enabled. So, giving a go ahead")
		return true
	}
	mutex.RLock()
	defer mutex.RUnlock()
	//ToDo: Need parameterization
	if len(recSharesMap) >= n {
		return true
	}
	return false
}

func addToRecSharesMap(nodeID int, share string) {
	mutex.Lock()
	defer mutex.Unlock()
	if recSharesMap == nil {
		m2m := GetMinerChain().GetDkgSet()
		recSharesMap = make(map[int]string, len(m2m.Nodes))
	}
	recSharesMap[nodeID] = share
}

func addToRecDkgSharesMap(nodeID int, dg *bls.Dkg) {
	mutex.Lock()
	defer mutex.Unlock()
	if recDkgSharesMap == nil {
		m2m := GetMinerChain().GetDkgSet()
		recDkgSharesMap = make(map[int]*bls.Dkg, len(m2m.Nodes))
	}
	Logger.Info("addToRecDkgSharesMap", zap.Int("len_of_vvecs_received", len(dg.Vvec)), zap.Int("nodeIndex", nodeID), zap.Any("share_at_0_dg.vvec", dg.Vvec[0]))
	recDkgSharesMap[nodeID] = dg
}

//ToDo: remove this. this is for experimenting.
func skipExtras(recFrom []string, recSig []string) ([]string, []string) {
	recSigx := make([]string, 0)
	recFromx := make([]string, 0)

	Logger.Info("lens", zap.Int("len_of_recFrom", len(recFrom)), zap.Int("len_of_recSig", len(recSig)))

	for i := 0; i < k; i++ {
		recSigx = append(recSigx, recSig[i])
	}
	for j := 0; j < k; j++ {
		recFromx = append(recFromx, recFrom[j])
	}
	Logger.Info("lens", zap.Int("len_of_recFromx", len(recFromx)), zap.Int("len_of_recSigx", len(recSigx)))
	return recFromx, recSigx
}

/*AppendVCVRFShares - Receives VFR shares for view change and processes it */
func AppendVCVRFShares(ctx context.Context, nodeID string, share *chain.VCVRFShare) {
	Logger.Info("Append vcVrfs request", zap.Int("senderIndex", node.GetSender(ctx).SetIndex), zap.String("sender", nodeID), zap.String("share", share.Share))

	if !isDkgEnabled {
		Logger.Error("DKG is not enabled. Why are we here?")
		return
	}
	if IsDkgDone() {
		Logger.Info("Dkg Cancelled")
		return
	}

	mb := GetMinerChain().GetMagicBlock(dgVc.MagicBlockNumber)
	if mb == nil {
		Logger.Info("Magicblock not available", zap.Int64("mbNumber", dgVc.MagicBlockNumber))
		return
	}

	/*
		Note: cannot verifySigShare here, as we can be here before VVec is generated
		//ToDo: Handle this after adding but before generating randomseed
		if !VerifySigShares(share.Share, ind, mb.SignedBlsMessage) {
			Logger.Info("Throwing away vcVrfs", zap.Int("sender", ind), zap.String("signedMessage", mb.SignedBlsMessage), zap.String("share", share.Share))
			return
		}
	*/

	Logger.Info("Adding vcVrfs request", zap.Int("senderIndex", node.GetSender(ctx).SetIndex), zap.String("sender", nodeID), zap.String("share", share.Share))

	if mb.IsVcVrfConsensusReached() {
		//adding additional vcvrfs, but we will not process further
		mb.AddToVcVrfSharesMap(nodeID, share)
		Logger.Info("added addtional vcVrfShare", zap.Int64("mb_number", mb.GetMagicBlockNumber()),
			zap.String("sender", nodeID), zap.String("share", share.Share))
		return
	}
	if !mb.AddToVcVrfSharesMap(nodeID, share) {
		Logger.Info("Could not add vcvrf share", zap.Int64("mb_number", mb.GetMagicBlockNumber()),
			zap.String("sender", nodeID), zap.String("share", share.Share))
		return
	}

	if mb.IsVcVrfConsensusReached() {
		recSig, recFroms := mb.GetVcVRFShareInfo()
		recFrom := make([]string, 0)
		recFroms, recSig = skipExtras(recFroms, recSig)
		dkgset := mb.DKGSetMiners
		for _, from := range recFroms {
			s := ComputeBlsID(dkgset.GetNode(from).SetIndex)
			Logger.Info("VCVrf ComputeBlsID", zap.Int64("MBNum", mb.GetMagicBlockNumber()), zap.Int("SetIndex", dkgset.GetNode(from).SetIndex), zap.String("blsId", s))

			recFrom = append(recFrom, s)
		}

		Logger.Info("VcVrf Consensus reached ...", zap.Int("recSig", len(recSig)), zap.Int("recFrom", len(recFrom)))
		rbOutput := bsVc.CalcRandomBeacon(recSig, recFrom)
		useed, err := strconv.ParseUint(rbOutput[0:16], 16, 64)
		if err != nil {
			panic(err)
		}
		randomSeed := int64(useed)
		Logger.Info("vcVrfs is done :) ...", zap.String("rbOuput", rbOutput), zap.Int64("randomseed", randomSeed), zap.String("sec_key", bsVc.SecKeyShareGroup.GetHexString()))

		mc := GetMinerChain()
		mb.DkgDone(bsVc.SecKeyShareGroup.GetHexString(), randomSeed)

		n := mb.ActiveSetMiners.GetNodeFromGNode(node.Self.GNode)
		//ToDo: Remove this check once we know it is always registered
		if n == nil {
			Logger.DPanic("self is not registered", zap.String("shortname", node.Self.GNode.GetPseudoName()))
			return
		}
		if !mb.IsMinerInActiveSet(n) {
			SetDkgDone(1)
			Logger.Panic("Not selected in ActiveSet")
			return
		}
		dgVc.SetRandomSeedVC(randomSeed)

		if mc.IsCurrentMagicBlock(mb.GetMagicBlockNumber()) {
			Logger.Info("Got curr MagicBlock info", zap.Int64("mbNumber", mb.GetMagicBlockNumber()), zap.Int64("mbrrs", mb.RandomSeed), zap.String("type", string(mb.TypeOfMB)))
			mc.InitChainActiveSetFromMagicBlock(mb)
			dgVrf = dgVc
			UpdateMagicBlock(ctx, mb)
			verifyMBStore(ctx, mb.TypeOfMB, "inserting curr mb")
			SetDkgDone(1)
			mc.SetupChainWorkers(ctx)
			go startProtocol()
		} else if mc.IsNextMagicBlock(mb.GetMagicBlockNumber()) {
			Logger.Info("Got next MagicBlock info", zap.Int64("mbNumber", mb.GetMagicBlockNumber()), zap.Int64("mbrrs", mb.RandomSeed))
			SetDkgDone(1)
		}
	}

}

/*AppendDKGSecShares - Gets the shares by other miners and append to the global array */
func AppendDKGSecShares(ctx context.Context, nodeID int, dg *bls.Dkg) {
	Logger.Info("Received DKG Shares", zap.Int("nodeIndex", nodeID), zap.String("share", dg.Share))
	if IsDkgDone() {
		Logger.Info("Dkg is over. Ignoring the incoming message")
		return
	}
	if !isDkgEnabled {
		Logger.Error("DKG is not enabled. Why are we here?")
		return
	}

	if recSharesMap != nil {
		if _, ok := recSharesMap[nodeID]; ok {
			Logger.Info("Ignoring Share recived again from node : ", zap.Int("Node Id", nodeID))
			return
		}
	}
	if recDkgSharesMap != nil {
		if _, ok := recDkgSharesMap[nodeID]; ok {
			Logger.Debug("Ignoring Share recived again from node : ", zap.Int("Node Id", nodeID))
			return
		}
	}
	addToRecDkgSharesMap(nodeID, dg)

	share := dg.Share
	recShares = append(recShares, share)
	addToRecSharesMap(nodeID, share)
	if HasAllDKGSharesReceived() {
		Logger.Debug("All the shares are received ...")
		AggregateDKGSecShares(ctx, recShares)
		dgVc.GroupVvec = ComputeVvec(ctx, recDkgSharesMap, k, n)
		Logger.Info("DKG is done :) Onto VcVRF...")
		bsVc = bls.MakeSimpleBLS(&dgVc)
		mc := GetMinerChain()
		mb := mc.GetNextMagicBlock()
		if mb == nil {
			mb = mc.GetCurrentMagicBlock()
		}
		mc.RunVRFForVC(ctx, mb)
	}

}

// VerifySigShares - Verify the bls sig share is correct
func VerifySigShares(dg bls.DKG, sigShare string, fromID int, msgString string) bool {
	senderID := ComputeBlsID(fromID)
	Logger.Info("PrintGroupsVvec from verifySigshares")
	PrintGroupsVvec(dg.GroupVvec)
	err := bls.VerifyVrf(sigShare, senderID, fromID, msgString, dg.GroupVvec)
	if err != nil {
		Logger.Error("VerifySigShares failed", zap.Int("len_of_groupvvec", len(dgVc.GroupVvec)), zap.Int("fromID", fromID), zap.Error(err))
		return false

	}

	return true

}

/*GetBlsThreshold Handy api for now.  */
func GetBlsThreshold() int {
	//return dg.T
	return k
}

/*ComputeBlsIDS Handy API to get the ID used in the library */
func ComputeBlsIDS(key string) string {
	computeID := bls.ComputeIDdkgS(key)
	return computeID.GetDecString()
}

/*ComputeBlsID Handy API to get the ID used in the library */
func ComputeBlsID(key int) string {
	computeID, err := bls.ComputeIDdkg(key)
	if err != nil {
		Logger.Error("Eror in computeIDdkg", zap.Int("index", key), zap.Error(err))
	}
	return computeID.GetDecString()
}

// ComputeVvec compute group vvec from individual vvec
func ComputeVvec(ctx context.Context, dkgSharesMap map[int]*bls.Dkg, t, num int) []bls.VerificationKey {
	numVvecs := len(dkgSharesMap)
	Logger.Info("ComputeVvec", zap.Int("len_of_vvecs", numVvecs), zap.Int("t", t), zap.Int("num", num))

	Vvecs := make([][]bls.VerificationKey, numVvecs)

	for i := 0; i < numVvecs; i++ {
		dgs := dkgSharesMap[i]
		Vvec := bls.GetVvecFromString(dgs.Vvec)

		Logger.Info("Got vVec in ComputeVvec", zap.Int("entries_in_vvec", len(Vvec)), zap.Any("share_at_0", Vvec[0].GetHexString()))

		Vvecs[i] = make([]bls.VerificationKey, t)
		for j := range Vvecs[i] {
			Vvecs[i][j] = Vvec[j]

		}

	}
	groupsVvec := bls.CalcGroupsVvec(Vvecs, t, num)
	PrintGroupsVvec(groupsVvec)
	return groupsVvec
}

// ToDo: remove this helper function once we know vvec is working.
func PrintGroupsVvec(groupsVvec []bls.VerificationKey) {
	Logger.Info("PrintGroupsVvec", zap.Int("groupsVvec_len", len(groupsVvec)))

	for i, v := range groupsVvec {
		Logger.Info("PrintGroupsVvec", zap.Int("index", i), zap.Any("vvec_entry", v.GetHexString()))
	}
}

// AggregateDKGSecShares - Each miner adds the shares to get the secKey share for group
func AggregateDKGSecShares(ctx context.Context, recShares []string) error {

	secShares := make([]bls.Key, len(recShares))
	for i := 0; i < len(recShares); i++ {
		err := secShares[i].SetDecString(recShares[i])
		if err != nil {
			Logger.Error("Aggregation of DKG shares not done", zap.Error(err))
		}
	}
	var sec bls.Key

	for i := 0; i < len(secShares); i++ {
		sec.Add(&secShares[i])
	}
	dgVc.SecKeyShareGroup = sec

	Logger.Info("Computed DKG",
		zap.String("sec_key_share_grp", dgVc.SecKeyShareGroup.GetDecString()),
		zap.String("gp_public_key", dgVc.GpPubKey.GetHexString()))
	return nil
}

// GetBlsShareForVC - Start the BLS process
func GetBlsShareForVC(mb *chain.MagicBlock) (string, string) {
	if !isDkgEnabled {
		Logger.Debug("returning standard string as DKG is not enabled.")
		return encryption.Hash("0chain"), "0chain"
	}

	msg := strconv.FormatInt(mb.PrevRandomSeed, 10)
	if msg == "0" {
		msg = "0chain"
	}
	Logger.Info("DKG getBlsShareForVC ", zap.Int64("mb_number", mb.GetMagicBlockNumber()), zap.String("msg", msg))
	bsVc.Msg = fmt.Sprintf("%v%v", mb.GetMagicBlockNumber(), msg)
	sigShare := bsVc.SignMsg()
	Logger.Info("getBlsShareForVC signedMessage", zap.Any("bsVC.ID", bsVc.ID), zap.String("signedMessage", bsVc.Msg), zap.String("sigShare", sigShare.GetHexString()))

	return sigShare.GetHexString(), bsVc.Msg

}

// GetBlsShare - Start the BLS process
func GetBlsShare(ctx context.Context, r, pr *round.Round) string {
	r.VrfStartTime = time.Now()
	if !isDkgEnabled {
		Logger.Debug("returning standard string as DKG is not enabled.")
		return encryption.Hash("0chain")
	}
	Logger.Debug("DKG getBlsShare ", zap.Int64("Round Number", r.Number))

	bsVrf = bls.MakeSimpleBLS(&dgVrf)

	currRound = r.Number
	var err error
	bsVrf.Msg, err = GetMinerChain().GetBlsMessageForRound(r)

	if err != nil {
		//ToDo: Return err here
		return "0"
	}
	sigShare := bsVrf.SignMsg()
	return sigShare.GetHexString()

}

//  ///////////  End fo BLS-DKG Related Stuff   ////////////////

//GetBlsMessageForRound given a round, get a message for BLS to sign
func (mc *Chain) GetBlsMessageForRound(r *round.Round) (string, error) {

	var rbOutput string
	prevRseed := int64(0)
	prevRoundNumber := r.GetRoundNumber() - 1
	if prevRoundNumber == 0 {

		Logger.Debug("The corner case for round 1 when pr is nil :", zap.Int64("round", r.GetRoundNumber()))
		rbOutput = encryption.Hash("0chain")
	} else {
		pr := mc.GetMinerRound(prevRoundNumber)
		if pr == nil {
			//This should never happen
			Logger.Error("could not find round object for non-zero round", zap.Int64("PrevRoundNum", prevRoundNumber))
			return "", common.NewError("no_prev_round", "Could not find the previous round")
		}
		prevRseed = pr.RandomSeed
		rbOutput = strconv.FormatInt(pr.RandomSeed, 16) //pr.VRFOutput
	}
	blsMsg := fmt.Sprintf("%v%v%v", r.GetRoundNumber(), r.GetTimeoutCount(), rbOutput)

	Logger.Info("Bls sign vrfshare calculated for ", zap.Int64("round", r.GetRoundNumber()), zap.Int("roundtimeout", r.GetTimeoutCount()),
		zap.Int64("prev_rseed", prevRseed), zap.Any("bls_msg", blsMsg), zap.String("sec_key", bsVrf.SecKeyShareGroup.GetHexString()))

	return blsMsg, nil
}

//AddVRFShare - implement the interface for the RoundRandomBeacon protocol
func (mc *Chain) AddVRFShare(ctx context.Context, mr *Round, vrfs *round.VRFShare) bool {
	Logger.Info("DKG AddVRFShare", zap.Int64("Round", mr.GetRoundNumber()), zap.Int("RoundTimeoutCount", mr.GetTimeoutCount()),
		zap.Int("Sender", vrfs.GetParty().SetIndex), zap.Int("vrf_timeoutcount", vrfs.GetRoundTimeoutCount()),
		zap.String("vrf_share", vrfs.Share))

	if vrfs.GetRoundTimeoutCount() != mr.GetTimeoutCount() {
		//Keep VRF timeout and round timeout in sync. Same vrfs will comeback during soft timeouts
		Logger.Info("TOC_FIX VRF Timeout > round timeout", zap.Int("vrfs_timeout", vrfs.GetRoundTimeoutCount()), zap.Int("round_timeout", mr.GetTimeoutCount()))
		return false
	}

	ind := vrfs.GetParty().SetIndex
	blsMsg, err := mc.GetBlsMessageForRound(mr.Round)
	if err == nil {
		if !VerifySigShares(dgVrf, vrfs.Share, ind, blsMsg) {
			Logger.Info("Throwing away vrfs", zap.Int("sender", ind), zap.String("signedMessage", blsMsg), zap.String("share", vrfs.Share))
			Logger.Panic("failed to verify") //ToDo: remove this panic once we know vvec is working
			return false
		} else {
			Logger.Info("success in verifying vrfs ", zap.Int("sender", ind), zap.String("signedMessage", blsMsg), zap.String("share", vrfs.Share))
		}
	} else {
		Logger.Info("could not get bls message. SKIPPING verifySigShares")
	}

	if len(mr.GetVRFShares()) >= GetBlsThreshold() {
		//ignore VRF shares coming after threshold is reached to avoid locking issues.
		//Todo: Remove this logging
		mr.AddAdditionalVRFShare(vrfs)
		Logger.Info("Ignoring VRFShare. Already at threshold", zap.Int64("Round", mr.GetRoundNumber()), zap.Int("VRF_Shares", len(mr.GetVRFShares())))
		return false
	}
	if mr.AddVRFShare(vrfs, GetBlsThreshold()) {
		mc.ThresholdNumBLSSigReceived(ctx, mr)
		return true
	}

	Logger.Info("Could not add VRFshare", zap.Int64("Round", mr.GetRoundNumber()), zap.Int("Sender", vrfs.GetParty().SetIndex))

	return false
}

/*ThresholdNumBLSSigReceived do we've sufficient BLSshares */
func (mc *Chain) ThresholdNumBLSSigReceived(ctx context.Context, mr *Round) {

	if mr.IsVRFComplete() {
		//BLS has completed already for this round, But, received a BLS message from a node now
		Logger.Info("DKG ThresholdNumSigReceived VRF is already completed.", zap.Int64("round", mr.GetRoundNumber()))
		return
	}

	shares := mr.GetVRFShares()
	if len(shares) >= GetBlsThreshold() {
		Logger.Debug("VRF Hurray we've threshold BLS shares")
		if !isDkgEnabled {
			//We're still waiting for threshold number of VRF shares, even though DKG is not enabled.

			rbOutput := "" //rboutput will ignored anyway
			mc.computeRBO(ctx, mr, rbOutput)

			return
		}
		beg := time.Now()
		recSig, recFrom := getVRFShareInfo(mr)

		rbOutput := bsVrf.CalcRandomBeacon(recSig, recFrom)
		Logger.Info("VRF ", zap.String("rboOutput", rbOutput), zap.Int64("Round", mr.Number), zap.String("sec_key", bsVrf.SecKeyShareGroup.GetHexString()))
		mc.computeRBO(ctx, mr, rbOutput)
		end := time.Now()

		diff := end.Sub(beg)

		if diff > (time.Duration(k) * time.Millisecond) {
			Logger.Info("DKG RBO Calc ***SLOW****", zap.Int64("Round", mr.GetRoundNumber()), zap.Int("VRF_shares", len(shares)), zap.Any("time_taken", diff))

		}
	} else {
		//TODO: remove this log
		Logger.Info("Not yet reached threshold", zap.Int("vrfShares_num", len(shares)), zap.Int("threshold", GetBlsThreshold()))
	}
}

func (mc *Chain) computeRBO(ctx context.Context, mr *Round, rbo string) {
	Logger.Debug("DKG computeRBO")
	if mr.IsVRFComplete() {
		Logger.Info("DKG computeRBO RBO is already completed")
		return
	}

	pr := mc.GetRound(mr.GetRoundNumber() - 1)
	mc.computeRoundRandomSeed(ctx, pr, mr, rbo)

}

func getVRFShareInfo(mr *Round) ([]string, []string) {
	recSig := make([]string, 0)
	recFrom := make([]string, 0)
	mr.Mutex.Lock()
	defer mr.Mutex.Unlock()

	shares := mr.GetVRFShares()
	for _, share := range shares {
		n := share.GetParty()
		Logger.Debug("VRF Printing from shares: ", zap.Int("Miner Index = ", n.SetIndex), zap.Any("Share = ", share.Share))

		recSig = append(recSig, share.Share)
		recFrom = append(recFrom, ComputeBlsID(n.SetIndex))
	}

	return recSig, recFrom
}

func (mc *Chain) computeRoundRandomSeed(ctx context.Context, pr round.RoundI, r *Round, rbo string) {

	var seed int64
	if isDkgEnabled {
		useed, err := strconv.ParseUint(rbo[0:16], 16, 64)
		if err != nil {
			panic(err)
		}
		seed = int64(useed)
	} else {
		if pr != nil {
			if mpr := pr.(*Round); mpr.IsVRFComplete() {
				seed = rand.New(rand.NewSource(pr.GetRandomSeed())).Int63()
			}
		} else {
			Logger.Error("pr is null! Let go this round...")
			return
		}
	}
	r.Round.SetVRFOutput(rbo)
	if pr != nil {
		//Todo: Remove this log later.
		Logger.Info("Starting round with vrf", zap.Int64("round", r.GetRoundNumber()),
			zap.Int("roundtimeout", r.GetTimeoutCount()),
			zap.Int64("rseed", seed), zap.Int64("prev_round", pr.GetRoundNumber()),
			//zap.Int("Prev_roundtimeout", pr.GetTimeoutCount()),
			zap.Int64("Prev_rseed", pr.GetRandomSeed()), zap.String("sec_key", bsVrf.SecKeyShareGroup.GetHexString()))
	}
	if !r.VrfStartTime.IsZero() {
		vrfTimer.UpdateSince(r.VrfStartTime)
	} else {
		Logger.Info("VrfStartTime is zero", zap.Int64("round", r.GetRoundNumber()))
	}
	mc.startRound(ctx, r, seed)

}
