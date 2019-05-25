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
	}
}

//CancelViewChange cancels a view change
func (mc *Chain) CancelViewChange(ctx context.Context) {
	if IsDkgDone() {
		Logger.Info("DKG is already done. Canceling the cancel")
		return
	}
	nextMgc := mc.GetNextMagicBlock()

	if nextMgc != nil {
		//ToDo: Store the next magic block also
		Logger.Info("nextMgc is not nil. Change it's parameters", zap.Int64("nextMagicNum", nextMgc.GetMagicBlockNumber()))
	}
	currMgc := mc.GetCurrentMagicBlock()
	verifyMBStore(ctx, currMgc.TypeOfMB, "Printing magicblock before cancel")
	currMgc.EstimatedLastRound = currMgc.EstimatedLastRound + mc.MagicBlockLife
	UpdateMagicBlock(ctx, currMgc)
	verifyMBStore(ctx, currMgc.TypeOfMB, "Printing magicblock after cancel")
	SetDkgDone(1)
	Logger.Info("Canceled Viewchange ", zap.Int64("mbNum", currMgc.GetMagicBlockNumber()), zap.Int64("lastRoundNum", currMgc.EstimatedLastRound))
}

// StartViewChange starts a viewchange for next magicblock
func (mc *Chain) StartViewChange(ctx context.Context, currMgc *chain.MagicBlock) {
	Logger.Info("starting viewchange", zap.Int64("currMgcNumber", currMgc.GetMagicBlockNumber()))
	nextMgc := currMgc.SetupNextMagicBlock()
	mc.SetNextMagicBlock(nextMgc)
	//StoreMagicBlock(ctx, nextMgc)
	StartMbDKG(ctx, nextMgc)
}

// SwitchToNextView Promotes next magic block as current including ActiveSet Changes
func (mc *Chain) SwitchToNextView(ctx context.Context, currMgc *chain.MagicBlock) {

	nmb := mc.GetNextMagicBlock()
	if nmb != nil {
		//dg.SecKeyShareGroup.SetHexString(nmb.SecretKeyGroupStr)
		mc.PromoteMagicBlockToCurr(nmb)
		mc.InitChainActiveSetFromMagicBlock(nmb)
		dgVrf = dgVc
		Logger.Info("Promoted next to curr", zap.Int64("current_was", currMgc.GetMagicBlockNumber()), zap.Int64("current_is", mc.GetCurrentMagicBlock().GetMagicBlockNumber()), zap.Any("mbtype", nmb.TypeOfMB))
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
	xmb, er := getMagicBlockFromStore(ctx, mbtype)
	if er != nil {
		Logger.Error("Failed to get magicblock from store", zap.Error(er))
		return false
	}
	if xmb != nil {
		Logger.Info(mesg, zap.Any("mbtype", xmb.TypeOfMB), zap.Int64("mbnum", xmb.GetMagicBlockNumber()),
			zap.Int("len_of_allminers", len(xmb.AllMiners.Nodes)), zap.Int("len_of_dkgsetminers", len(xmb.DKGSetMiners.Nodes)),
			zap.Int("len_of_activesetminers", len(xmb.ActiveSetMiners.Nodes)), zap.Int("len_of_allsharders", len(xmb.AllSharders.Nodes)),
			zap.Int64("start_round", xmb.StartingRound), zap.Int64("last_round", xmb.EstimatedLastRound), zap.Int64("random_seed", xmb.RandomSeed), zap.Int64("prev_random_seed", xmb.PrevRandomSeed))
	}
	return true
}

// ToDo: fix it. turning off handling restarts for now.
func handleOldDkg(dkgSummary *bls.DKGSummary) bool {
	if dkgSummary.SecretKeyGroupStr != "" {
		Logger.Info("handle old DKG.")
	}
	return false
	/*
		dg.SecKeyShareGroup.SetHexString(dkgSummary.SecretKeyGroupStr)

				if dkgSummary.MagicBlockNumber != mgc.GetMagicBlockNumber() {
					IsDkgDone = true
					//ToDo: Handle this case better
					Logger.Panic("Magic block number is different", zap.Int64("stored_mbnum", dkgSummary.MagicBlockNumber), zap.Int64("current_mbnum", mgc.MagicBlockNumber))
					return
				}
				Logger.Info("got dkg share from db", zap.Int64("mbnum", dkgSummary.MagicBlockNumber))

				mc := GetMinerChain()
				//Add this. We need to wait for enough miners to be active during restart
				//waitForNetworkToBeReadyForBls(ctx)

				if !mgc.IsMinerInActiveSet(node.Self.Node) {
					IsDkgDone = true
					Logger.Panic("Not selected in ActiveSet")
					return
				}
				mc.InitChainActiveSetFromMagicBlock(mgc)
				IsDkgDone = true
				go startProtocol()
				return
	*/
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
	SetDkgDone(0)

	Logger.Info("DKG Setup", zap.Int("K", k), zap.Int("N", n), zap.Bool("DKG Enabled", isDkgEnabled))

	self := node.GetSelfNode(ctx)
	selfInd = self.SetIndex

	if isDkgEnabled {
		dgVc = bls.MakeDKG(k, n, mgc.GetMagicBlockNumber())
		//ToDo: Need to make it per viewchange
		dkgSummary, err := getDKGSummaryFromStore(ctx)
		if dkgSummary.SecretKeyGroupStr != "" {
			if handleOldDkg(dkgSummary) {
				return
			}
		}

		Logger.Info("DKG Not found. Starting afresh", zap.Error(err))

		waitForMbNetworkToBeReady(ctx, mgc)
		if IsDkgDone() {
			Logger.Info("Cannot continue with DKG. It seems canceled. Returning")
			return
		}
		Logger.Info("Starting DKG...")

		minerShares = make(map[string]bls.Key, len(miners.Nodes))

		for _, node := range miners.Nodes {
			forID := bls.ComputeIDdkg(node.SetIndex)
			dgVc.ID = forID

			secShare, _ := dgVc.ComputeDKGKeyShare(forID)

			//Logger.Debug("ComputeDKGKeyShare ", zap.String("secShare", secShare.GetDecString()), zap.Int("miner index", node.SetIndex))
			minerShares[node.GetKey()] = secShare
			if self.SetIndex == node.SetIndex {
				recShares = append(recShares, secShare.GetDecString())
				addToRecSharesMap(self.SetIndex, secShare.GetDecString())
			}

		}
		WaitForMbDKGShares(mgc)

	} else {
		Logger.Info("DKG is not enabled. So, starting protocol")
		SetDkgDone(1)
		go startProtocol()
	}

}

/*
// StartDKG - starts the DKG process
func StartDKG(ctx context.Context) {

	mc := GetMinerChain()

	m2m := mc.Miners

	isDkgEnabled = config.DevConfiguration.IsDkgEnabled
	thresholdByCount := viper.GetInt("server_chain.block.consensus.threshold_by_count")
	k = int(math.Ceil((float64(thresholdByCount) / 100) * float64(mc.Miners.Size())))
	n = mc.Miners.Size()

	Logger.Info("DKG Setup", zap.Int("K", k), zap.Int("N", n), zap.Bool("DKG Enabled", isDkgEnabled))

	self := node.GetSelfNode(ctx)
	selfInd = self.SetIndex

	if isDkgEnabled {
		dg = bls.MakeDKG(k, n, 0)

		dkgSummary, err := getDKGSummaryFromStore(ctx)
		if dkgSummary.SecretKeyGroupStr != "" {
			dg.SecKeyShareGroup.SetHexString(dkgSummary.SecretKeyGroupStr)
			Logger.Info("got dkg share from db")
			waitForNetworkToBeReadyForBls(ctx)
			IsDkgDone = true
			go startProtocol()
			return
		} else {
			Logger.Info("err : reading dkg from db", zap.Error(err))
		}
		waitForNetworkToBeReadyForDKG(ctx)
		Logger.Info("Starting DKG...")

		minerShares = make(map[string]bls.Key, len(m2m.Nodes))

		for _, node := range m2m.Nodes {
			forID := bls.ComputeIDdkg(node.SetIndex)
			dg.ID = forID

			secShare, _ := dg.ComputeDKGKeyShare(forID)

			//Logger.Debug("ComputeDKGKeyShare ", zap.String("secShare", secShare.GetDecString()), zap.Int("miner index", node.SetIndex))
			minerShares[node.GetKey()] = secShare
			if self.SetIndex == node.SetIndex {
				recShares = append(recShares, secShare.GetDecString())
				addToRecSharesMap(self.SetIndex, secShare.GetDecString())
			}

		}
		WaitForDKGShares()
	} else {
		Logger.Info("DKG is not enabled. So, starting protocol")
		IsDkgDone = true
		go startProtocol()
	}

}
*/

// RunVRFForVC run a VRF on the DKG once to rank the miners
func (mc *Chain) RunVRFForVC(ctx context.Context, mb *chain.MagicBlock) {
	vcVrfs := &chain.VCVRFShare{}
	vcVrfs.MagicBlockNumber = mb.GetMagicBlockNumber()
	vcVrfs.Share = GetBlsShareForVC(mb)
	vcVrfs.SetParty(node.Self.Node)
	mb.VcVrfShare = vcVrfs
	Logger.Debug("Appending VCVrfShares", zap.String("vcvrfshare", vcVrfs.Share))
	AppendVCVRFShares(ctx, node.Self.Node.ID, vcVrfs)
	vcVrfs.SetKey(datastore.ToKey(fmt.Sprintf("%v", vcVrfs.MagicBlockNumber)))
	err := SendMbVcVrfShare(mb, vcVrfs)
	if err != nil {
		Logger.Error("Error while sending vcVrfShare", zap.Error(err))
	}
}

func getDKGSummaryFromStore(ctx context.Context) (*bls.DKGSummary, error) {
	dkgSummary := datastore.GetEntity("dkgsummary").(*bls.DKGSummary)
	dkgSummaryMetadata := dkgSummary.GetEntityMetadata()
	dctx := ememorystore.WithEntityConnection(ctx, dkgSummaryMetadata)
	defer ememorystore.Close(dctx)
	err := dkgSummary.Read(dctx, dkgSummary.GetKey())
	return dkgSummary, err
}

func getMagicBlockFromStore(ctx context.Context, mbtype chain.MBType) (*chain.MagicBlock, error) {
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

func storeDKGSummary(ctx context.Context, dkgSummary *bls.DKGSummary) error {
	dkgSummaryMetadata := dkgSummary.GetEntityMetadata()
	dctx := ememorystore.WithEntityConnection(ctx, dkgSummaryMetadata)
	defer ememorystore.Close(dctx)
	err := dkgSummary.Write(dctx)
	if err != nil {
		return err
	}
	con := ememorystore.GetEntityCon(dctx, dkgSummaryMetadata)
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

	go miners.DKGMonitor(ctx)
	//m2m := mc.Miners
	if !mgc.IsMbReadyForDKG() {
		ticker := time.NewTicker(5 * chain.DELTA)
		for ts := range ticker.C {
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

	secShare := minerShares[n.GetKey()]
	dkg := &bls.Dkg{
		Share: secShare.GetDecString()}
	dkg.SetKey(datastore.ToKey("1"))
	Logger.Debug("sending DKG share", zap.Any("recipient", n.GetKey()), zap.Any("share", dkg.Share))
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
		mc := GetMinerChain()

		m2m := mc.Miners
		recSharesMap = make(map[int]string, len(m2m.Nodes))
	}
	recSharesMap[nodeID] = share
}

/*AppendVCVRFShares - Receives VFR shares for view change and processes it */
func AppendVCVRFShares(ctx context.Context, nodeID string, share *chain.VCVRFShare) {
	if !isDkgEnabled {
		Logger.Error("DKG is not enabled. Why are we here?")
		return
	}
	if IsDkgDone() {
		Logger.Info("Dkg Cancelled")
		return
	}
	Logger.Info("Adding vcVrfs", zap.String("sender", nodeID), zap.String("share", share.Share))
	//ToDo: Need to figure out if it is currMB or nextMB
	//mb := GetMinerChain().GetCurrentMagicBlock()
	mb := GetMinerChain().GetMagicBlock(dgVc.MagicBlockNumber)
	if mb == nil {
		Logger.Info("Magicblock not available", zap.Int64("mbNumber", dgVc.MagicBlockNumber))
		return
	}
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
		dkgset := mb.DKGSetMiners
		for _, from := range recFroms {
			recFrom = append(recFrom, ComputeBlsID(dkgset.GetNode(from).SetIndex))
		}

		Logger.Info("VcVrf Consensus reached ...", zap.Int("recSig", len(recSig)), zap.Int("recFrom", len(recFrom)))
		rbOutput := bsVc.CalcRandomBeacon(recSig, recFrom)
		useed, err := strconv.ParseUint(rbOutput[0:16], 16, 64)
		if err != nil {
			panic(err)
		}
		randomSeed := int64(useed)
		Logger.Info("vcVrfs is done :) ...", zap.String("rbOuput", rbOutput), zap.Int64("randomseed", randomSeed))

		mc := GetMinerChain()
		mb.DkgDone(bsVc.SecKeyShareGroup.GetHexString(), randomSeed)

		if !mb.IsMinerInActiveSet(node.Self.Node) {
			SetDkgDone(1)
			Logger.Panic("Not selected in ActiveSet")
			return
		}
		dgVc.SetRandomSeedVC(randomSeed)
		storeDKGSummary(ctx, dgVc.GetDKGSummary())
		getAndPrintStoredDKG(ctx) //For testing purposes. ToDo: Remove this
		SetDkgDone(1)
		if mc.IsCurrentMagicBlock(mb.GetMagicBlockNumber()) {
			Logger.Info("Got next MagicBlock info", zap.Int64("mbNumber", mb.GetMagicBlockNumber()), zap.Int64("mbrrs", mb.RandomSeed), zap.String("type", string(mb.TypeOfMB)))
			mc.InitChainActiveSetFromMagicBlock(mb)
			dgVrf = dgVc
			UpdateMagicBlock(ctx, mb)
			verifyMBStore(ctx, mb.TypeOfMB, "inserting curr mb")
			go startProtocol()
		} else if mc.IsNextMagicBlock(mb.GetMagicBlockNumber()) {
			Logger.Info("Got next MagicBlock info", zap.Int64("mbNumber", mb.GetMagicBlockNumber()), zap.Int64("mbrrs", mb.RandomSeed))
		}
	}

}

func getAndPrintStoredDKG(ctx context.Context) {
	dkgSummary, err := getDKGSummaryFromStore(ctx)
	if err != nil {
		Logger.Error("Error in reading DKGSummaryFromStore", zap.Error(err))
		return
	}
	Logger.Info("Got DKGSummaryFromStore", zap.Int64("stored_mbnum", dkgSummary.MagicBlockNumber), zap.Int64("stored_dkg_vc_vrf", dkgSummary.RandomSeedVC))
}

/*AppendDKGSecShares - Gets the shares by other miners and append to the global array */
func AppendDKGSecShares(ctx context.Context, nodeID int, share string) {
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
			Logger.Debug("Ignoring Share recived again from node : ", zap.Int("Node Id", nodeID))
			return
		}
	}
	recShares = append(recShares, share)
	addToRecSharesMap(nodeID, share)
	//ToDo: We cannot expect everyone to be ready to start. Should we use K?
	if HasAllDKGSharesReceived() {
		Logger.Debug("All the shares are received ...")
		AggregateDKGSecShares(ctx, recShares)
		Logger.Info("DKG is done :) ...")
		bsVc = bls.MakeSimpleBLS(&dgVc)
		mc := GetMinerChain()
		mc.RunVRFForVC(ctx, mc.GetCurrentMagicBlock())
	}

}

// VerifySigShares - Verify the bls sig share is correct
func VerifySigShares() bool {
	//TBD
	return true
}

/*GetBlsThreshold Handy api for now. move this to protocol_vrf */
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
	computeID := bls.ComputeIDdkg(key)
	return computeID.GetDecString()
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

	//storeDKGSummary(ctx, dg.GetDKGSummary()) Use this for non-viewchange DKG
	Logger.Info("Computed DKG")
	Logger.Debug("the aggregated sec share",
		zap.String("sec_key_share_grp", dgVc.SecKeyShareGroup.GetDecString()),
		zap.String("gp_public_key", dgVc.GpPubKey.GetHexString()))
	return nil
}

// GetBlsShareForVC - Start the BLS process
func GetBlsShareForVC(mb *chain.MagicBlock) string {
	if !isDkgEnabled {
		Logger.Debug("returning standard string as DKG is not enabled.")
		return encryption.Hash("0chain")
	}

	msg := strconv.FormatInt(mb.PrevRandomSeed, 10)
	if msg == "0" {
		msg = "0chain"
	}
	Logger.Info("DKG getBlsShareForVC ", zap.Int64("mb_number", mb.GetMagicBlockNumber()), zap.String("msg", msg))

	bsVc.Msg = fmt.Sprintf("%v%v", mb.GetMagicBlockNumber(), msg)
	sigShare := bsVc.SignMsg()
	return sigShare.GetHexString()

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
	var rbOutput string
	prevRseed := int64(0)
	if r.GetRoundNumber()-1 == 0 {

		Logger.Debug("The corner case for round 1 when pr is nil :", zap.Int64("round", r.GetRoundNumber()))
		rbOutput = encryption.Hash("0chain")
	} else {
		prevRseed = pr.RandomSeed
		rbOutput = strconv.FormatInt(pr.RandomSeed, 16) //pr.VRFOutput
	}

	bsVrf.Msg = fmt.Sprintf("%v%v%v", r.GetRoundNumber(), r.GetTimeoutCount(), rbOutput)

	Logger.Info("Bls sign vrfshare calculated for ", zap.Int64("round", r.GetRoundNumber()), zap.Int("roundtimeout", r.GetTimeoutCount()), zap.Int64("prev_rseed", prevRseed), zap.Any("bls_msg", bsVrf.Msg))

	sigShare := bsVrf.SignMsg()
	return sigShare.GetHexString()

}

//  ///////////  End fo BLS-DKG Related Stuff   ////////////////

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
		Logger.Debug("VRF ", zap.String("rboOutput", rbOutput), zap.Int64("Round", mr.Number))
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
			zap.Int64("Prev_rseed", pr.GetRandomSeed()))
	}
	if !r.VrfStartTime.IsZero() {
		vrfTimer.UpdateSince(r.VrfStartTime)
	} else {
		Logger.Info("VrfStartTime is zero", zap.Int64("round", r.GetRoundNumber()))
	}
	mc.startRound(ctx, r, seed)

}
