package chain

import (
	"container/ring"
	"context"
	"fmt"
	"math"
	"os"
	"runtime/pprof"
	"sort"
	"sync"
	"time"

	"0chain.net/chaincore/client"
	"0chain.net/core/ememorystore"
	"0chain.net/core/encryption"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

//PreviousBlockUnavailable - to indicate an error condition when the previous block of a given block is not available
const PreviousBlockUnavailable = "previous_block_unavailable"

//ErrPreviousBlockUnavailable - error for previous block is not available
var ErrPreviousBlockUnavailable = common.NewError(PreviousBlockUnavailable, "Previous block is not available")
var ErrInsufficientChain = common.NewError("insufficient_chain", "Chain length not sufficient to perform the logic")

const (
	NOTARIZED = 1
	FINALIZED = 2
)

const (
	AllMiners  = 11
	Generator  = 12
	Generators = 13
)

/*ServerChain - the chain object of the chain  the server is responsible for */
var ServerChain *Chain

/*SetServerChain - set the server chain object */
func SetServerChain(c *Chain) {
	ServerChain = c
}

/*GetServerChain - returns the chain object for the server chain */
func GetServerChain() *Chain {
	return ServerChain
}

/*BlockStateHandler - handles the block state changes */
type BlockStateHandler interface {
	UpdatePendingBlock(ctx context.Context, b *block.Block, txns []datastore.Entity)
	UpdateFinalizedBlock(ctx context.Context, b *block.Block)
}

/*Chain - data structure that holds the chain data*/
type Chain struct {
	datastore.IDField
	datastore.VersionField
	datastore.CreationDateField

	//Chain config goes into this object
	*Config

	/*Miners - this is the pool of miners in active set */
	Miners *node.Pool `json:"-"`

	/*Sharders - this is the pool of sharders in active set*/
	Sharders *node.Pool `json:"-"`

	/*Blobbers - this is the pool of blobbers */
	Blobbers *node.Pool `json:"-"`

	/* This is a cache of blocks that may include speculative blocks */
	blocks      map[datastore.Key]*block.Block
	blocksMutex *sync.RWMutex

	rounds      map[int64]round.RoundI
	roundsMutex *sync.RWMutex

	CurrentRound             int64        `json:"-"`
	CurrentMagicBlock        *block.Block `json:"-"`
	LatestFinalizedBlock     *block.Block `json:"latest_finalized_block,omitempty"` // Latest block on the chain the program is aware of
	LatestDeterministicBlock *block.Block `json:"latest_deterministic_block,omitempty"`

	clientStateDeserializer state.DeserializerI
	stateDB                 util.NodeDB
	stateMutex              *sync.RWMutex

	finalizedRoundsChannel chan round.RoundI
	finalizedBlocksChannel chan *block.Block

	*Stats `json:"-"`

	BlockChain *ring.Ring `json:"-"`

	minersStake map[datastore.Key]int
	stakeMutex  *sync.Mutex

	nodePoolScorer node.PoolScorer

	GenerateTimeout int `json:"-"`
	genTimeoutMutex *sync.Mutex

	retry_wait_time  int
	retry_wait_mutex *sync.Mutex

	blockFetcher *BlockFetcher

	crtCount int64 // Continuous/Current Round Timeout Count

	fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler

	pruneStats *util.PruneStats

	CurrMagicBlock *MagicBlock `json:"curr_magic_block,omitempty"`
	NextMagicBlock *MagicBlock `json:"next_magic_block,omitempty"`
	MagicBlockLife int64       `json:"magic_block_life,omitempty"`
	configInfoDB   string

	configInfoStore datastore.Store
	RoundF          round.RoundFactory `json:"-"`
}

var chainEntityMetadata *datastore.EntityMetadataImpl

/*GetEntityMetadata - implementing the interface */
func (c *Chain) GetEntityMetadata() datastore.EntityMetadata {
	return chainEntityMetadata
}

/*Validate - implementing the interface */
func (c *Chain) Validate(ctx context.Context) error {
	if datastore.IsEmpty(c.ID) {
		return common.InvalidRequest("chain id is required")
	}
	if datastore.IsEmpty(c.OwnerID) {
		return common.InvalidRequest("owner id is required")
	}
	return nil
}

/*Read - store read */
func (c *Chain) Read(ctx context.Context, key datastore.Key) error {
	return c.GetEntityMetadata().GetStore().Read(ctx, key, c)
}

/*Write - store read */
func (c *Chain) Write(ctx context.Context) error {
	return c.GetEntityMetadata().GetStore().Write(ctx, c)
}

/*Delete - store read */
func (c *Chain) Delete(ctx context.Context) error {
	Logger.Error("here in chain delete")
	return c.GetEntityMetadata().GetStore().Delete(ctx, c)
}

//NewChainFromConfig - create a new chain from config
func NewChainFromConfig() *Chain {
	chain := Provider().(*Chain)
	chain.ID = datastore.ToKey(config.Configuration.ChainID)
	chain.Decimals = int8(viper.GetInt("server_chain.decimals"))
	chain.BlockSize = viper.GetInt32("server_chain.block.max_block_size")
	chain.MinBlockSize = viper.GetInt32("server_chain.block.min_block_size")
	chain.MaxByteSize = viper.GetInt64("server_chain.block.max_byte_size")
	chain.NumGenerators = viper.GetInt("server_chain.block.generators")
	chain.NotariedBlocksCounts = make([]int64, chain.NumGenerators+1)
	chain.NumReplicators = viper.GetInt("server_chain.block.replicators")
	chain.ThresholdByCount = viper.GetInt("server_chain.block.consensus.threshold_by_count")
	chain.ThresholdByStake = viper.GetInt("server_chain.block.consensus.threshold_by_stake")
	chain.OwnerID = viper.GetString("server_chain.owner")
	chain.ValidationBatchSize = viper.GetInt("server_chain.block.validation.batch_size")
	chain.RoundRange = viper.GetInt64("server_chain.round_range")
	chain.MagicBlockLife = viper.GetInt64("server_chain.magic_block_life")
	chain.TxnMaxPayload = viper.GetInt("server_chain.transaction.payload.max_size")
	chain.PruneStateBelowCount = viper.GetInt("server_chain.state.prune_below_count")
	verificationTicketsTo := viper.GetString("server_chain.messages.verification_tickets_to")
	if verificationTicketsTo == "" || verificationTicketsTo == "all_miners" || verificationTicketsTo == "11" {
		chain.VerificationTicketsTo = AllMiners
	} else {
		chain.VerificationTicketsTo = Generator
	}
	chain.HealthyRoundNumber = viper.GetInt64("server_chain.health_check.round")
	chain.BatchSyncSize = viper.GetInt("server_chain.health_check.batch_sync_size")
	chain.BlockProposalMaxWaitTime = viper.GetDuration("server_chain.block.proposal.max_wait_time") * time.Millisecond
	waitMode := viper.GetString("server_chain.block.proposal.wait_mode")
	if waitMode == "static" {
		chain.BlockProposalWaitMode = BlockProposalWaitStatic
	} else if waitMode == "dynamic" {
		chain.BlockProposalWaitMode = BlockProposalWaitDynamic
	}
	chain.ReuseTransactions = viper.GetBool("server_chain.block.reuse_txns")
	chain.SetSignatureScheme(viper.GetString("server_chain.client.signature_scheme"))

	chain.MinActiveSharders = viper.GetInt("server_chain.block.sharding.min_active_sharders")
	chain.MinActiveReplicators = viper.GetInt("server_chain.block.sharding.min_active_replicators")
	chain.SmartContractTimeout = viper.GetDuration("server_chain.smart_contract.timeout") * time.Millisecond
	chain.RoundTimeoutSofttoMin = viper.GetInt("server_chain.round_timeouts.softto_min")
	chain.RoundTimeoutSofttoMult = viper.GetInt("server_chain.round_timeouts.softto_mult")
	chain.RoundRestartMult = viper.GetInt("server_chain.round_timeouts.round_restart_mult")
	chain.ActiveSetMinerMax = viper.GetInt("server_chain.miner_active_set_max_size")
	chain.ActiveSetMinerMin = viper.GetInt("server_chain.miner_active_set_min_size")
	chain.DkgSetMinerIncMax = viper.GetInt("server_chain.miner_dkg_set_inc_max")

	return chain
}

/*Provider - entity provider for chain object */
func Provider() datastore.Entity {
	c := &Chain{}
	c.Config = &Config{}
	c.Initialize()
	c.Version = "1.0"

	c.blocks = make(map[string]*block.Block)
	c.blocksMutex = &sync.RWMutex{}

	c.rounds = make(map[int64]round.RoundI)
	c.roundsMutex = &sync.RWMutex{}

	c.retry_wait_mutex = &sync.Mutex{}
	c.genTimeoutMutex = &sync.Mutex{}
	c.stateMutex = &sync.RWMutex{}
	c.stakeMutex = &sync.Mutex{}
	c.InitializeCreationDate()
	c.nodePoolScorer = node.NewHashPoolScorer(encryption.NewXORHashScorer())
	c.Miners = node.NewPool(node.NodeTypeMiner)
	c.Sharders = node.NewPool(node.NodeTypeSharder)
	c.Blobbers = node.NewPool(node.NodeTypeBlobber)
	c.Stats = &Stats{}
	c.blockFetcher = NewBlockFetcher()
	return c
}

/*Initialize - intializes internal datastructures to start again */
func (c *Chain) Initialize() {
	c.CurrentRound = 0
	c.LatestFinalizedBlock = nil
	c.CurrentMagicBlock = nil
	c.BlocksToSharder = 1
	c.VerificationTicketsTo = AllMiners
	c.ValidationBatchSize = 2000
	c.finalizedRoundsChannel = make(chan round.RoundI, 128)
	c.finalizedBlocksChannel = make(chan *block.Block, 128)
	c.clientStateDeserializer = &state.Deserializer{}
	c.stateDB = stateDB
	c.BlockChain = ring.New(10000)
	c.minersStake = make(map[datastore.Key]int)
}

/*SetupEntity - setup the entity */
func SetupEntity(store datastore.Store) {
	chainEntityMetadata = datastore.MetadataProvider()
	chainEntityMetadata.Name = "chain"
	chainEntityMetadata.Provider = Provider
	chainEntityMetadata.Store = store
	datastore.RegisterEntityMetadata("chain", chainEntityMetadata)
	SetupStateDB()
}

var stateDB *util.PNodeDB

//SetupStateDB - setup the state db
func SetupStateDB() {
	db, err := util.NewPNodeDB("data/rocksdb/state", "/0chain/log/rocksdb/state")
	if err != nil {
		panic(err)
	}
	stateDB = db
}

func (c *Chain) SetupConfigInfoDB() {
	c.configInfoDB = "configdb"
	c.configInfoStore = ememorystore.GetStorageProvider()
	db, err := ememorystore.CreateDB("data/rocksdb/config")
	if err != nil {
		panic(err)
	}
	ememorystore.AddPool(c.configInfoDB, db)
}

func (c *Chain) GetConfigInfoDB() string {
	return c.configInfoDB
}

func (c *Chain) GetConfigInfoStore() datastore.Store {
	return c.configInfoStore
}

func (c *Chain) getInitialState() util.Serializable {
	tokens := viper.GetInt64("server_chain.tokens")
	balance := &state.State{}
	balance.SetTxnHash("0000000000000000000000000000000000000000000000000000000000000000")
	var cents int64 = 1
	for i := int8(0); i < c.Decimals; i++ {
		cents *= 10
	}
	balance.Balance = state.Balance(tokens * cents)
	return balance
}

/*setupInitialState - setup the initial state based on configuration */
func (c *Chain) setupInitialState() util.MerklePatriciaTrieI {
	pmt := util.NewMerklePatriciaTrie(c.stateDB, util.Sequence(0))
	pmt.Insert(util.Path(c.OwnerID), c.getInitialState())
	pmt.SaveChanges(c.stateDB, false)
	Logger.Info("initial state root", zap.Any("hash", util.ToHex(pmt.GetRoot())))
	return pmt
}

/*GenerateGenesisBlock - Create the genesis block for the chain */
func (c *Chain) GenerateGenesisBlock(hash string) (round.RoundI, *block.Block) {
	c.GenesisBlockHash = hash
	gb := block.NewBlock(c.GetKey(), 0)
	gb.Hash = hash
	gb.ClientState = c.setupInitialState()
	gb.SetStateStatus(block.StateSuccessful)
	gb.SetBlockState(block.StateNotarized)
	gb.ClientStateHash = gb.ClientState.GetRoot()
	gr := round.NewRound(0)
	c.SetRandomSeed(gr, 839695260482366273)
	gr.ComputeMinerRanks(c.Miners)
	gr.Block = gb
	gr.AddNotarizedBlock(gb)
	return gr, gb
}

/*AddGenesisBlock - adds the genesis block to the chain */
func (c *Chain) AddGenesisBlock(b *block.Block) {
	if b.Round != 0 {
		return
	}
	c.LatestFinalizedBlock = b // Genesis block is always finalized
	c.LatestDeterministicBlock = b
	c.CurrentMagicBlock = b // Genesis block is always a magic block
	c.blocks[b.Hash] = b
	return
}

/*AddBlock - adds a block to the cache */
func (c *Chain) AddBlock(b *block.Block) *block.Block {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()
	return c.addBlock(b)
}

/*AddNotarizedBlockToRound - adds notarized block to cache and sync  info from notarized block to round  */
func (c *Chain) AddNotarizedBlockToRound(r round.RoundI, b *block.Block) (*block.Block, round.RoundI) {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()

	/*
		Since this nb mostly from a diff node, addBlock will return local block with the same hash if exists.
		Either way the block content is same, but we will get it from the local.
	*/
	b = c.addBlock(b)

	if b.Round == c.CurrentRound {
		Logger.Info("Adding a notarized block for current round", zap.Int64("Round", r.GetRoundNumber()))
	}

	//Get round data insync as it is the notarized block
	if r.GetRandomSeed() != b.RoundRandomSeed || r.GetTimeoutCount() != b.RoundTimeoutCount {
		Logger.Info("AddNotarizedBlockToRound round and block random seed different", zap.Int64("Round", r.GetRoundNumber()), zap.Int64("Round_rrs", r.GetRandomSeed()), zap.Int64("Block_rrs", b.RoundRandomSeed))
		r.SetRandomSeedForNotarizedBlock(b.RoundRandomSeed)
		r.SetTimeoutCount(b.RoundTimeoutCount)
		r.ComputeMinerRanks(c.Miners)
	}

	c.SetRoundRank(r, b)
	if b.PrevBlock != nil {
		b.ComputeChainWeight()
	}
	return b, r
}

/*AddRoundBlock - add a block for a given round to the cache */
func (c *Chain) AddRoundBlock(r round.RoundI, b *block.Block) *block.Block {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()
	b2 := c.addBlock(b)
	if b2 != b {
		return b2
	}
	b.RoundRandomSeed = r.GetRandomSeed()
	b.RoundTimeoutCount = r.GetTimeoutCount()
	c.SetRoundRank(r, b)
	if b.PrevBlock != nil {
		b.ComputeChainWeight()
	}
	return b
}

func (c *Chain) addBlock(b *block.Block) *block.Block {
	if eb, ok := c.blocks[b.Hash]; ok {
		if eb != b {
			c.MergeVerificationTickets(common.GetRootContext(), eb, b.VerificationTickets)
		}
		return eb
	}
	c.blocks[b.Hash] = b
	if b.PrevBlock == nil {
		if pb, ok := c.blocks[b.PrevHash]; ok {
			b.SetPreviousBlock(pb)
		} else {
			c.AsyncFetchNotarizedPreviousBlock(b)
		}
	}
	for pb := b.PrevBlock; pb != nil && pb != c.LatestDeterministicBlock; pb = pb.PrevBlock {
		pb.AddUniqueBlockExtension(b)
		if c.IsFinalizedDeterministically(pb) {
			c.LatestDeterministicBlock = pb
			break
		}
	}
	return b
}

/*AsyncFetchNotarizedPreviousBlock - async fetching of the previous notarized block */
func (c *Chain) AsyncFetchNotarizedPreviousBlock(b *block.Block) {
	c.blockFetcher.AsyncFetchPreviousBlock(b)
}

/*AsyncFetchNotarizedBlock - async fetching of a notarized block */
func (c *Chain) AsyncFetchNotarizedBlock(hash string) {
	c.blockFetcher.AsyncFetchBlock(hash)
}

/*GetBlock - returns a known block for a given hash from the cache */
func (c *Chain) GetBlock(ctx context.Context, hash string) (*block.Block, error) {
	c.blocksMutex.RLock()
	defer c.blocksMutex.RUnlock()
	return c.getBlock(ctx, hash)
}

func (c *Chain) getBlock(ctx context.Context, hash string) (*block.Block, error) {
	if b, ok := c.blocks[datastore.ToKey(hash)]; ok {
		return b, nil
	}
	return nil, common.NewError(datastore.EntityNotFound, fmt.Sprintf("Block with hash (%v) not found", hash))
}

/*DeleteBlock - delete a block from the cache */
func (c *Chain) DeleteBlock(ctx context.Context, b *block.Block) {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()
	if _, ok := c.blocks[b.Hash]; !ok {
		return
	}
	delete(c.blocks, b.Hash)
}

/*GetRoundBlocks - get the blocks for a given round */
func (c *Chain) GetRoundBlocks(round int64) []*block.Block {
	blocks := make([]*block.Block, 0, 1)
	c.blocksMutex.RLock()
	defer c.blocksMutex.RUnlock()
	for _, b := range c.blocks {
		if b.Round == round {
			blocks = append(blocks, b)
		}
	}
	return blocks
}

/*DeleteBlocksBelowRound - delete all the blocks below this round */
func (c *Chain) DeleteBlocksBelowRound(round int64) {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()
	ts := common.Now() - 60
	blocks := make([]*block.Block, 0, 1)
	for _, b := range c.blocks {
		if b.Round < round && b.CreationDate < ts && b.Round < c.LatestDeterministicBlock.Round {
			Logger.Debug("found block to delete", zap.Int64("round", round), zap.Int64("block_round", b.Round), zap.Int64("current_round", c.CurrentRound), zap.Int64("lf_round", c.LatestFinalizedBlock.Round))
			blocks = append(blocks, b)
		}
	}
	Logger.Info("delete blocks below round", zap.Int64("round", c.CurrentRound), zap.Int64("below_round", round), zap.Any("before", ts), zap.Int("total", len(c.blocks)), zap.Int("count", len(blocks)))
	for _, b := range blocks {
		b.Clear()
		delete(c.blocks, b.Hash)
	}

}

/*DeleteBlocks - delete a list of blocks */
func (c *Chain) DeleteBlocks(blocks []*block.Block) {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()
	for _, b := range blocks {
		b.Clear()
		delete(c.blocks, b.Hash)
	}
}

/*PruneChain - prunes the chain */
func (c *Chain) PruneChain(ctx context.Context, b *block.Block) {
	c.DeleteBlocksBelowRound(b.Round - 50)
}

/*ValidateMagicBlock - validate the block for a given round has the right magic block */
func (c *Chain) ValidateMagicBlock(ctx context.Context, b *block.Block) bool {
	//TODO: This needs to take the round number into account and go backwards as needed to validate
	return b.MagicBlockHash == c.CurrentMagicBlock.Hash
}

//IsRoundGenerator - is this miner a generator for this round
func (c *Chain) IsRoundGenerator(r round.RoundI, nd *node.Node) bool {
	return r.GetMinerRank(nd) < c.NumGenerators
}

/*GetGenerators - get all the block generators for a given round */
func (c *Chain) GetGenerators(r round.RoundI) []*node.Node {
	miners := r.GetMinersByRank(c.Miners)
	return miners[:c.NumGenerators]
}

/*IsBlockSharder - checks if the sharder can store the block in the given round */
func (c *Chain) IsBlockSharder(b *block.Block, sharder *node.Node) bool {
	if c.NumReplicators <= 0 {
		return true
	}
	scores := c.nodePoolScorer.ScoreHashString(c.Sharders, b.Hash)
	return sharder.IsInTop(scores, c.NumReplicators)
}

func (c *Chain) IsBlockSharderFromHash(bHash string, sharder *node.Node) bool {
	if c.NumReplicators <= 0 {
		return true
	}
	scores := c.nodePoolScorer.ScoreHashString(c.Sharders, bHash)
	return sharder.IsInTop(scores, c.NumReplicators)
}

/*CanShardBlockWithReplicators - checks if the sharder can store the block with nodes that store this block*/
func (c *Chain) CanShardBlockWithReplicators(hash string, sharder *node.Node) (bool, []*node.Node) {
	if c.NumReplicators <= 0 {
		return true, nil
	}
	scores := c.nodePoolScorer.ScoreHashString(c.Sharders, hash)
	return sharder.IsInTopWithNodes(scores, c.NumReplicators)
}

//GetBlockSharders - get the list of sharders who would be replicating the block
func (c *Chain) GetBlockSharders(b *block.Block) []string {
	var sharders []string
	//TODO: sharders list needs to get resolved per the magic block of the block
	var sharderPool = c.Sharders
	var sharderNodes = sharderPool.Nodes
	if c.NumReplicators > 0 {
		scores := c.nodePoolScorer.ScoreHashString(sharderPool, b.Hash)
		sharderNodes = node.GetTopNNodes(scores, c.NumReplicators)
	}
	for _, sharder := range sharderNodes {
		sharders = append(sharders, sharder.GetKey())
	}
	return sharders
}

/*ValidGenerator - check whether this block is from a valid generator */
func (c *Chain) ValidGenerator(r round.RoundI, b *block.Block) bool {
	miner := c.Miners.GetNode(b.MinerID)
	if miner == nil {
		return false
	}

	isGen := c.IsRoundGenerator(r, miner)
	if !isGen {
		//This is a Byzantine condition?
		Logger.Info("Received a block from non-generator", zap.String("miner", miner.GetPseudoName()), zap.Int64("RRS", r.GetRandomSeed()))
		gens := c.GetGenerators(r)

		Logger.Info("Generators are: ", zap.Int64("round", r.GetRoundNumber()))
		for _, n := range gens {
			Logger.Info("generator", zap.String("Node", n.GetPseudoName()))
		}
	}
	return isGen
}

/*GetNotarizationThresholdCount - gives the threshold count for block to be notarized*/
func (c *Chain) GetNotarizationThresholdCount() int {
	notarizedPercent := float64(c.ThresholdByCount) / 100
	thresholdCount := float64(c.Miners.Size()) * notarizedPercent
	return int(math.Ceil(thresholdCount))
}

// AreAllNodesActive - use this to check if all nodes needs to be active as in DKG
func (c *Chain) AreAllNodesActive() bool {
	active := c.Miners.GetActiveCount()
	return active >= c.Miners.Size()
}

/*CanStartNetwork - check whether the network can start */
func (c *Chain) CanStartNetwork() bool {
	active := c.Miners.GetActiveCount()
	threshold := c.GetNotarizationThresholdCount()
	return active >= threshold && c.CanShardBlocks()
}

// InitCurrMagicBlock to be called if current magic block is found while launching. Calling this any other time may lead to bad consequences
func (c *Chain) InitCurrMagicBlock(ctx, mgc *MagicBlock) {
	c.CurrMagicBlock = mgc
}

// ReadNodePools reads node pools from the given file and stores in Magic Block
func (c *Chain) ReadNodePools(ctx context.Context, configFile string) error {
	if c.CurrMagicBlock == nil {
		c.CurrMagicBlock = SetupMagicBlock(CURR, 0, 0, 0, CalcLastRound(0, c.MagicBlockLife), c.ActiveSetMinerMax, c.ActiveSetMinerMin)
	}
	err := c.CurrMagicBlock.ReadNodePools(configFile)
	if err != nil {
		//Error is already logged inside ReadNodePools
		return err
	}
	return nil
}

// AddARegisteredMiner Store a registered miner in MagicBlocks
func (c *Chain) AddARegisteredMiner(id, publicKey, shortName, hostName string, port int) {
	//if nmb exists, first add it there
	nmb := c.GetNextMagicBlock()
	if nmb != nil {
		nmb.AddARegisteredMiner(id, publicKey, shortName, hostName, port)
	}
	cmb := c.GetCurrentMagicBlock()
	if cmb != nil {
		cmb.AddARegisteredMiner(id, publicKey, shortName, hostName, port)
	}
	if nmb == nil && cmb == nil {
		Logger.Info("Both nmb and cmb are nil.")
	}
}

// ComputeActiveSetMinersForSharder -- temp fix for sharders
func (c *Chain) ComputeActiveSetMinersForSharder() {
	mgc := c.GetCurrentMagicBlock()
	mgc.ComputeActiveSetMinersForSharder()
	c.SetActiveSetMiners(mgc.GetActiveSetMiners())
	c.SetActiveSetSharders(mgc.GetActiveSetSharders())
}

// InitChainActiveSetFromMagicBlock initializes Chain active nodes from magic block's activeset
func (c *Chain) InitChainActiveSetFromMagicBlock(mgc *MagicBlock) {
	//ToDo: Figure out how to lock these as it can be in use by n2n.
	c.SetActiveSetMiners(mgc.GetActiveSetMiners())
	c.SetActiveSetSharders(mgc.GetActiveSetSharders())
}

// SetActiveSetMiners sets chain miners from the given pool
func (c *Chain) SetActiveSetMiners(activeSetMiners *node.Pool) {
	c.Miners = activeSetMiners
	c.Miners.ComputeProperties()
	c.InitializeMinerPool()
}

// SetActiveSetSharders sets active sharders from the given pool
func (c *Chain) SetActiveSetSharders(activeSetSharders *node.Pool) {
	c.Sharders = activeSetSharders
	c.Sharders.ComputeProperties()
}

/*ChainHasTransaction - indicates if this chain has the transaction */
func (c *Chain) ChainHasTransaction(ctx context.Context, b *block.Block, txn *transaction.Transaction) (bool, error) {
	var pb = b
	for cb := b; cb != nil; pb, cb = cb, c.GetPreviousBlock(ctx, cb) {
		if cb.Round == 0 {
			return false, nil
		}
		if cb.HasTransaction(txn.Hash) {
			return true, nil
		}
		if cb.CreationDate < txn.CreationDate-common.Timestamp(transaction.TXN_TIME_TOLERANCE) {
			return false, nil
		}
	}
	if false {
		Logger.Debug("chain has txn", zap.Int64("round", b.Round), zap.Int64("upto_round", pb.Round), zap.Any("txn_ts", txn.CreationDate), zap.Any("upto_block_ts", pb.CreationDate))
	}
	return false, ErrInsufficientChain
}

func (c *Chain) updateMiningStake(minerID datastore.Key, stake int) {
	c.stakeMutex.Lock()
	defer c.stakeMutex.Unlock()
	c.minersStake[minerID] = stake
}

func (c *Chain) getMiningStake(minerID datastore.Key) int {
	return c.minersStake[minerID]
}

//InitializeMinerPool - initialize the miners after their configuration is read
func (c *Chain) InitializeMinerPool() {
	for _, nd := range c.Miners.Nodes {
		ms := &MinerStats{}
		ms.GenerationCountByRank = make([]int64, c.NumGenerators)
		ms.FinalizationCountByRank = make([]int64, c.NumGenerators)
		ms.VerificationTicketsByRank = make([]int64, c.NumGenerators)
		nd.ProtocolStats = ms
	}
}

/*AddRound - Add Round to the block */
func (c *Chain) AddRound(r round.RoundI) round.RoundI {
	c.roundsMutex.Lock()
	defer c.roundsMutex.Unlock()
	roundNumber := r.GetRoundNumber()
	er, ok := c.rounds[roundNumber]
	if ok {
		return er
	}
	c.rounds[roundNumber] = r
	return r
}

/*GetRound - get a round */
func (c *Chain) GetRound(roundNumber int64) round.RoundI {
	c.roundsMutex.RLock()
	defer c.roundsMutex.RUnlock()
	round, ok := c.rounds[roundNumber]
	if !ok {
		return nil
	}
	return round
}

/*DeleteRound - delete a round and associated block data */
func (c *Chain) DeleteRound(ctx context.Context, r round.RoundI) {
	c.roundsMutex.Lock()
	defer c.roundsMutex.Unlock()
	delete(c.rounds, r.GetRoundNumber())
}

/*DeleteRoundsBelow - delete rounds below */
func (c *Chain) DeleteRoundsBelow(ctx context.Context, roundNumber int64) {
	c.roundsMutex.Lock()
	defer c.roundsMutex.Unlock()
	rounds := make([]round.RoundI, 0, 1)
	for _, r := range c.rounds {
		if r.GetRoundNumber() < roundNumber-10 && r.GetRoundNumber() != 0 {
			rounds = append(rounds, r)
		}
	}
	for _, r := range rounds {
		r.Clear()
		delete(c.rounds, r.GetRoundNumber())
	}
}

/*SetRandomSeed - set the random seed for the round */
func (c *Chain) SetRandomSeed(r round.RoundI, randomSeed int64) bool {
	c.roundsMutex.Lock()
	defer c.roundsMutex.Unlock()
	if r.HasRandomSeed() && randomSeed == r.GetRandomSeed() {
		return false
	}
	r.Lock()
	defer r.Unlock()
	r.SetRandomSeed(randomSeed)
	r.ComputeMinerRanks(c.Miners)
	roundNumber := r.GetRoundNumber()
	if roundNumber > c.CurrentRound {
		c.CurrentRound = roundNumber
	}
	return true
}

func (c *Chain) getBlocks() []*block.Block {
	c.blocksMutex.RLock()
	defer c.blocksMutex.RUnlock()
	var bl []*block.Block
	for _, v := range c.blocks {
		bl = append(bl, v)
	}
	return bl
}

func (c *Chain) GetActivesetNodePoolForRound(roundNum int64) *node.Pool {
	currMB := c.GetCurrentMagicBlock()
	if currMB.DoesRoundBelongToMagicBlock(roundNum) {
		return currMB.GetActiveSetMiners()
	}
	nextMB := c.GetNextMagicBlock()
	if nextMB != nil && nextMB.DoesRoundBelongToMagicBlock(roundNum) {
		return nextMB.GetActiveSetMiners()
	}
	Logger.Error("Temp Fix: Returning currMB for activeset", zap.Int64("roundNum", roundNum))
	//ToDo: Temp fix until we get the SC support. For now, it works because we do not additional miners
	return currMB.GetActiveSetMiners()
	//return nil
}

// GetDkgSetNodePool Handy function to get the DkgSet Nodepool
func (c *Chain) GetDkgSetNodePool(mbType MBType) *node.Pool {
	if mbType == CURR {
		if c.GetCurrentMagicBlock() == nil {
			return nil
		}
		return c.GetCurrentMagicBlock().DKGSetMiners
	} else {
		if c.GetNextMagicBlock() == nil {
			return nil
		}
		return c.GetNextMagicBlock().DKGSetMiners
	}
}

//GetActivesetSharder gets node from Activeset Nodepool
func (c *Chain) GetActivesetSharder(bgNode *node.GNode) *node.Node {

	np := c.GetCurrentMagicBlock().GetActiveSetSharders()
	if np == nil {
		//ToDo: Handle this better
		Logger.Error("No Activeset for sharder node pool yet. Returning")
		return nil
	}
	return np.GetNodeFromGNode(bgNode)
}

//GetDkgSetMiner gets node from Activeset Nodepool
func (c *Chain) GetDkgSetMiner(bgNode *node.GNode, mbType MBType) *node.Node {

	np := c.GetDkgSetNodePool(mbType)
	if np == nil {
		//ToDo: Handle this better
		Logger.Error("No DKGSet node pool yet. Returning")
		return nil
	}
	return np.GetNodeFromGNode(bgNode)
}

//GetActivesetMinerForRound gets node from Activeset Nodepool
func (c *Chain) GetActivesetMinerForRound(roundNum int64, bgNode *node.GNode) *node.Node {
	var np *node.Pool
	if node.Self.Type == node.NodeTypeSharder {
		np = c.Miners
	} else {
		np = c.GetActivesetNodePoolForRound(roundNum)
	}
	if np == nil {
		//ToDo: Handle this better
		Logger.Error("No Activeset node pool yet. Returning", zap.Int64("roundNum", roundNum))
		return nil
	}
	return np.GetNodeFromGNode(bgNode)
}

//SetRoundRank - set the round rank of the block
func (c *Chain) SetRoundRank(r round.RoundI, b *block.Block) {
	bgNode := node.GetNode(b.MinerID)
	if bgNode == nil {
		//ToDo: Handle this better
		Logger.Error("Node is not registered", zap.Int64("roundNum", r.GetRoundNumber()), zap.String("blockhash", b.Hash), zap.Any("shortname", bgNode.GetPseudoName()))
		return
	}

	bNode := c.GetActivesetMinerForRound(r.GetRoundNumber(), bgNode)
	if bNode == nil {
		//ToDo: Handle this better
		Logger.Error("Node is not part of Activeset", zap.Int64("roundNum", r.GetRoundNumber()), zap.String("blockhash", b.Hash), zap.Any("shortname", bgNode.GetPseudoName()))
		return
	}

	rank := r.GetMinerRank(bNode)
	if rank >= c.NumGenerators {
		pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		Logger.DPanic(fmt.Sprintf("Round# %v generator miner ID %v rank is greater than num generators. State= %v, rank= %v, generators = %v, rrs = %v", r.GetRoundNumber(), bNode.GetPseudoName(), r.GetState(), rank, c.NumGenerators, r.GetRandomSeed()))
	}
	b.RoundRank = rank
	//TODO: Remove this log
	Logger.Info(fmt.Sprintf("Round# %v generator miner ID %v State= %v, rank= %v", r.GetRoundNumber(), bNode.GetPseudoName(), r.GetState(), rank))
}

func (c *Chain) SetGenerationTimeout(newTimeout int) {
	c.genTimeoutMutex.Lock()
	defer c.genTimeoutMutex.Unlock()
	c.GenerateTimeout = newTimeout
}

func (c *Chain) GetGenerationTimeout() int {
	c.genTimeoutMutex.Lock()
	defer c.genTimeoutMutex.Unlock()
	return c.GenerateTimeout
}

func (c *Chain) GetRetryWaitTime() int {
	c.retry_wait_mutex.Lock()
	defer c.retry_wait_mutex.Unlock()
	return c.retry_wait_time
}

func (c *Chain) SetRetryWaitTime(newWaitTime int) {
	c.retry_wait_mutex.Lock()
	defer c.retry_wait_mutex.Unlock()
	c.retry_wait_time = newWaitTime
}

/*GetUnrelatedBlocks - get blocks that are not related to the chain of the given block */
func (c *Chain) GetUnrelatedBlocks(maxBlocks int, b *block.Block) []*block.Block {
	c.blocksMutex.Lock()
	defer c.blocksMutex.Unlock()
	var blocks []*block.Block
	var chain = make(map[datastore.Key]*block.Block)
	var prevRound = b.Round
	for pb := b.PrevBlock; pb != nil; pb = pb.PrevBlock {
		prevRound = pb.Round
		chain[pb.Hash] = pb
	}
	for _, rb := range c.blocks {
		if rb.Round >= prevRound && rb.Round < b.Round && common.WithinTime(int64(b.CreationDate), int64(rb.CreationDate), transaction.TXN_TIME_TOLERANCE) {
			if _, ok := chain[rb.Hash]; !ok {
				blocks = append(blocks, rb)
			}
			if len(blocks) >= maxBlocks {
				break
			}
		}
	}
	sort.SliceStable(blocks, func(i, j int) bool { return blocks[i].Round > blocks[j].Round })
	return blocks
}

//ResetRoundTimeoutCount - reset the counter
func (c *Chain) ResetRoundTimeoutCount() {
	c.crtCount = 0
}

//IncrementRoundTimeoutCount - increment the counter
func (c *Chain) IncrementRoundTimeoutCount() {
	c.crtCount++
}

//GetRoundTimeoutCount - get the counter
func (c *Chain) GetRoundTimeoutCount() int64 {
	return c.crtCount
}

//SetSignatureScheme - set the client signature scheme to be used by this chain
func (c *Chain) SetSignatureScheme(sigScheme string) {
	c.ClientSignatureScheme = sigScheme
	client.SetClientSignatureScheme(c.ClientSignatureScheme)
}

//GetSignatureScheme - get the signature scheme used by this chain
func (c *Chain) GetSignatureScheme() encryption.SignatureScheme {
	return encryption.GetSignatureScheme(c.ClientSignatureScheme)
}

//CanShardBlocks - is the network able to effectively shard the blocks?
func (c *Chain) CanShardBlocks() bool {
	return c.Sharders.GetActiveCount()*100 >= c.Sharders.Size()*c.MinActiveSharders
}

//CanReplicateBlock - can the given block be effectively replicated?
func (c *Chain) CanReplicateBlock(b *block.Block) bool {
	if c.NumReplicators <= 0 || c.MinActiveReplicators == 0 {
		return c.CanShardBlocks()
	}
	scores := c.nodePoolScorer.ScoreHashString(c.Sharders, b.Hash)
	arCount := 0
	minScore := scores[c.NumReplicators-1].Score
	for i := 0; i < len(scores); i++ {
		if scores[i].Score < minScore {
			break
		}
		if scores[i].Node.IsActive() {
			arCount++
			if arCount*100 >= c.NumReplicators*c.MinActiveReplicators {
				return true
			}
		}
	}
	return false
}

//SetFetchedNotarizedBlockHandler - setter for FetchedNotarizedBlockHandler
func (c *Chain) SetFetchedNotarizedBlockHandler(fnbh FetchedNotarizedBlockHandler) {
	c.fetchedNotarizedBlockHandler = fnbh
}

//GetPruneStats - get the current prune stats
func (c *Chain) GetPruneStats() *util.PruneStats {
	return c.pruneStats
}

// IsNextMagicBlock given a magic block number returns true if it is current magic block
func (c *Chain) IsNextMagicBlock(mbNumber int64) bool {
	if c.NextMagicBlock != nil && c.NextMagicBlock.GetMagicBlockNumber() == mbNumber {
		return true
	}
	return false
}

// IsCurrentMagicBlock given a magic block number returns true if it is current magic block
func (c *Chain) IsCurrentMagicBlock(mbNumber int64) bool {
	if c.CurrMagicBlock != nil && c.CurrMagicBlock.GetMagicBlockNumber() == mbNumber {
		return true
	}
	return false
}

// GetMagicBlock returns the magic block for the given number if it is current or next; nil otherwise
func (c *Chain) GetMagicBlock(mbNumber int64) *MagicBlock {
	if c.CurrMagicBlock != nil && c.CurrMagicBlock.GetMagicBlockNumber() == mbNumber {
		return c.CurrMagicBlock
	} else if c.NextMagicBlock != nil && c.NextMagicBlock.GetMagicBlockNumber() == mbNumber {
		return c.NextMagicBlock
	} else {
		currMbNumber := int64(0)
		if c.CurrMagicBlock != nil {
			currMbNumber = c.CurrMagicBlock.GetMagicBlockNumber()
		}
		nextMbNumber := int64(0)
		if c.NextMagicBlock != nil {
			nextMbNumber = c.NextMagicBlock.GetMagicBlockNumber()
		}
		Logger.Error("Tried to access nonexistant magicblock", zap.Int64("requested_mb_number", mbNumber), zap.Int64("curr_mb_number", currMbNumber), zap.Int64("next_mb_number", nextMbNumber))
		return nil
	}
}

//GetCurrentMagicBlock returns current magic block
func (c *Chain) GetCurrentMagicBlock() *MagicBlock {
	//ToDo: Needs mutex?
	return c.CurrMagicBlock
}

//GetNextMagicBlock returns next magic block
func (c *Chain) GetNextMagicBlock() *MagicBlock {
	return c.NextMagicBlock
}

//SetNextMagicBlock sets next magic block
func (c *Chain) SetNextMagicBlock(mb *MagicBlock) {
	c.NextMagicBlock = mb
}

//PromoteMagicBlockToCurr -moves next magic block to current
func (c *Chain) PromoteMagicBlockToCurr(nmb *MagicBlock) {
	//ToDo: Needs mutex?
	c.CurrMagicBlock = nmb
	c.CurrMagicBlock.TypeOfMB = CURR
	//ToDo: Save this to DB
}

//InitBlockState - initialize the block's state with the database state
func (c *Chain) InitBlockState(b *block.Block) {
	if err := b.InitStateDB(c.stateDB); err != nil {
		Logger.Error("init block state", zap.Int64("round", b.Round), zap.String("state", util.ToHex(b.ClientStateHash)), zap.Error(err))
	} else {
		Logger.Info("init block state successful", zap.Int64("round", b.Round), zap.String("state", util.ToHex(b.ClientStateHash)))
	}
}
