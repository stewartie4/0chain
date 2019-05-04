package chain

import (
	"fmt"
	"math"

	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

//MagicBlock to create and track active sets
type MagicBlock struct {
	MagicBlockNumber   int
	StartingRound      int64
	EstimatedLastRound int64

	/*Miners - this is the pool of miners participating in the blockchain */
	ActiveSetMiners *node.Pool `json:"-"`

	/*Sharders - this is the pool of sharders participaing in the blockchain*/
	ActiveSetSharders *node.Pool `json:"-"`

	/*Miners - this is the pool of all miners */
	AllMiners *node.Pool `json:"-"`

	/*Sharders - this is the pool of all sharders */
	AllSharders *node.Pool `json:"-"`

	/*DKGSetMiners -- this is the pool of all Miners in the DKG process */
	DKGSetMiners *node.Pool `json:"-"`
}

//SetupMagicBlock create and setup magicblock object
func SetupMagicBlock(startingRound int64, life int64) *MagicBlock {
	mb := &MagicBlock{}
	mb.StartingRound = startingRound
	mb.EstimatedLastRound = mb.StartingRound + life
	Logger.Info("Created magic block", zap.Int64("Starting_round", mb.StartingRound), zap.Int64("ending_round", mb.EstimatedLastRound))
	return mb
}

/*ReadNodePools - read the node pools from configuration */
func (mb *MagicBlock) ReadNodePools(configFile string) error {
	nodeConfig := config.ReadConfig(configFile)
	config := nodeConfig.Get("miners")
	if miners, ok := config.([]interface{}); ok {
		if mb.AllMiners == nil {
			//Reading from config file, the node pools need to be initialized
			mb.AllMiners = node.NewPool(node.NodeTypeMiner)

			mb.AllMiners.AddNodes(miners)
			mb.AllMiners.ComputeProperties()

		}

	}
	config = nodeConfig.Get("sharders")
	if sharders, ok := config.([]interface{}); ok {
		if mb.AllSharders == nil {
			//Reading from config file, the node pools need to be initialized
			mb.AllSharders = node.NewPool(node.NodeTypeSharder)
			mb.ActiveSetSharders = node.NewPool(node.NodeTypeSharder)
			mb.AllSharders.AddNodes(sharders)
			mb.AllSharders.ComputeProperties()
			mb.ActiveSetSharders.AddNodes(sharders)
			mb.ActiveSetSharders.ComputeProperties()
		}

	}

	if mb.AllMiners == nil || mb.AllSharders == nil {
		err := common.NewError("configfile_read_err", "Either sharders or miners or both are not found in "+configFile)
		Logger.Info(err.Error())
		return err
	}
	Logger.Info("Added miners", zap.Int("all_miners", len(mb.AllMiners.Nodes)),
		zap.Int("all_sharders", len(mb.AllSharders.Nodes)),
		zap.Int("active_sharders", len(mb.ActiveSetSharders.Nodes)))

	//ToDo: NeedsFix. We need this because Sharders need this right after reading the pool. Fix it.
	mb.GetComputedDKGSet()
	return nil
}

//GetAllMiners gets all miners node pool
func (mb *MagicBlock) GetAllMiners() *node.Pool {
	return mb.AllMiners
}

//GetActiveSetMiners gets all miners in ActiveSet
func (mb *MagicBlock) GetActiveSetMiners() *node.Pool {
	return mb.ActiveSetMiners
}

//GetAllSharders Gets all sharders in the pool
func (mb *MagicBlock) GetAllSharders() *node.Pool {
	return mb.AllSharders
}

//GetActiveSetSharders gets all sharders in the active set
func (mb *MagicBlock) GetActiveSetSharders() *node.Pool {
	return mb.ActiveSetSharders
}

// GetComputedDKGSet select and provide miners set for DKG based on the rules
func (mb *MagicBlock) GetComputedDKGSet() (*node.Pool, *common.Error) {
	mb.DKGSetMiners = node.NewPool(node.NodeTypeMiner)
	miners, err := mb.getDKGSetAfterRules(mb.GetAllMiners())

	if err != nil || miners == nil {
		return miners, err
	}
	mb.DKGSetMiners = miners
	mb.DKGSetMiners.ComputeProperties()
	Logger.Info("returning computed dkg set miners", zap.Int("dkgset_num", mb.DKGSetMiners.Size()))
	return mb.DKGSetMiners, nil
}

func (mb *MagicBlock) getDKGSetAfterRules(allMiners *node.Pool) (*node.Pool, *common.Error) {
	sc := GetServerChain()

	/*
	   Rule#1: if allMiners size is less than the active set required size, you cannopt proceed
	*/
	if allMiners.Size() < sc.ActiveSetMinerMin {
		return nil, common.NewError("too_few_miners", fmt.Sprintf("Need: %v, Have %v", sc.ActiveSetMinerMin, allMiners.Size()))
	}

	var currActiveSetSize int
	if mb.ActiveSetMiners != nil {
		currActiveSetSize = mb.ActiveSetMiners.Size()
	}

	/*
	  Rule#2: DKGSet size cannot be more than increment size of the current active set size;
	  if starting, assume all miners are eligible
	*/
	var dkgSetSize int
	if currActiveSetSize > 0 {
		dkgSetSize = int(math.Ceil((float64(sc.DkgSetMinerIncMax) / 100) * float64(currActiveSetSize)))
	} else {
		dkgSetSize = allMiners.Size()
	}
	if allMiners.Size() > dkgSetSize {
		Logger.Error("Too many miners Need to use stake logic", zap.Int("need", dkgSetSize), zap.Int("have", allMiners.Size()))
	}
	dkgMiners := node.NewPool(node.NodeTypeMiner)

	for _, miner := range allMiners.Nodes {
		dkgMiners.AddNode(miner)
	}

	return dkgMiners, nil
}

// IsMbReadyForDKG are the miners in DKGSet ready for DKG
func (mb *MagicBlock) IsMbReadyForDKG() bool {
	active := mb.DKGSetMiners.GetActiveCount()
	return active >= mb.DKGSetMiners.Size()
}

// DKGDone Tell magic block that DKG is done.
func (mb *MagicBlock) DKGDone() {
	mb.ActiveSetMiners = node.NewPool(node.NodeTypeMiner)
	//This needs more logic. Simplistic approach of all DKGSet moves to ActiveSet for now
	for _, n := range mb.DKGSetMiners.Nodes {
		mb.ActiveSetMiners.AddNode(n)
	}

	mb.ActiveSetMiners.ComputeProperties()
}
