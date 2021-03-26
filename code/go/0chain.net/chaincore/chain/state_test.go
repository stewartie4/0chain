package chain

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"container/ring"
	"context"
	"sync"
	"testing"
	"time"
)

func TestChain_ComputeState(t *testing.T) {
	type fields struct {
		LatestFinalizedBlock *block.Block
		blockFetcher         *BlockFetcher
	}
	type args struct {
		ctx context.Context
		b   *block.Block
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		stateDebug bool
		wantErr    bool
	}{
		{
			name: "State_Computed_OK",
			args: args{
				b: func() *block.Block {
					b := block.NewBlock("", 1)
					b.SetStateStatus(block.StateSuccessful)

					return b
				}(),
			},
		},
		{
			name: "Nil_Prev_Block_ERR",
			fields: fields{
				blockFetcher: NewBlockFetcher(),
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.TODO())
					cancel()
					return ctx
				}(),
				b: func() *block.Block {
					b := block.NewBlock("", 1)
					b.HashBlock()

					return b
				}(),
			},
			wantErr: true,
		},
		{
			name: "Prev_Block_Point_To_Itself_ERR",
			fields: fields{
				blockFetcher: NewBlockFetcher(),
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.TODO())
					cancel()
					return ctx
				}(),
				b: func() *block.Block {
					prevB := block.NewBlock("", 1)
					prevB.HashBlock()
					b := block.NewBlock("", 1)
					b.SetPreviousBlock(prevB)
					b.HashBlock()
					b.StateMutex = prevB.StateMutex

					return b
				}(),
			},
			wantErr: true,
		},
		{
			name: "Prev_Block_State_Failed_And_Fail_Get_From_Network_ERR",
			fields: fields{
				blockFetcher:         NewBlockFetcher(),
				LatestFinalizedBlock: block.NewBlock("", 1),
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.TODO())
					cancel()
					return ctx
				}(),
				b: func() *block.Block {
					prevB := block.NewBlock("", 1)
					prevB.HashBlock()
					prevB.SetStateStatus(block.StateFailed)
					b := block.NewBlock("", 1)
					b.SetPreviousBlock(prevB)
					b.HashBlock()

					return b
				}(),
			},
			wantErr: true,
		},
		{
			name: "Prev_Block_State_Computing_And_Failed_New_Computing_Debug_Level_ERR",
			fields: fields{
				blockFetcher:         NewBlockFetcher(),
				LatestFinalizedBlock: block.NewBlock("", 2),
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.TODO())
					cancel()
					return ctx
				}(),
				b: func() *block.Block {
					prevB := block.NewBlock("", 1)
					prevB.HashBlock()
					prevB.SetStateStatus(block.StateComputing)
					b := block.NewBlock("", 1)
					b.SetPreviousBlock(prevB)
					b.HashBlock()

					return b
				}(),
			},
			wantErr:    true,
			stateDebug: true,
		},
		{
			name: "Prev_Block_State_Computing_And_Failed_New_Computing_ERR",
			fields: fields{
				blockFetcher:         NewBlockFetcher(),
				LatestFinalizedBlock: block.NewBlock("", 2),
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.TODO())
					cancel()
					return ctx
				}(),
				b: func() *block.Block {
					prevB := block.NewBlock("", 1)
					prevB.HashBlock()
					prevB.SetStateStatus(block.StateComputing)
					b := block.NewBlock("", 1)
					b.SetPreviousBlock(prevB)
					b.HashBlock()

					return b
				}(),
			},
			wantErr:    true,
			stateDebug: false,
		},
		{
			name: "Prev_Block_State_Computed_ERR",
			fields: fields{
				blockFetcher:         NewBlockFetcher(),
				LatestFinalizedBlock: block.NewBlock("", 2),
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.TODO())
					cancel()
					return ctx
				}(),
				b: func() *block.Block {
					prevB := block.NewBlock("", 1)
					prevB.HashBlock()
					prevB.SetStateStatus(block.StateSuccessful)
					b := block.NewBlock("", 1)
					b.SetPreviousBlock(prevB)
					b.HashBlock()

					return b
				}(),
			},
			wantErr:    true,
			stateDebug: false,
		},
		{
			name: "Prev_Block_State_Computed_And_Nil_Client_State_ERR",
			fields: fields{
				blockFetcher:         NewBlockFetcher(),
				LatestFinalizedBlock: block.NewBlock("", 2),
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.TODO())
					cancel()
					return ctx
				}(),
				b: func() *block.Block {
					prevB := block.NewBlock("", 1)
					prevB.HashBlock()
					prevB.SetStateStatus(block.StateSuccessful)
					b := block.NewBlock("", 1)
					b.SetPreviousBlock(prevB)
					b.HashBlock()

					return b
				}(),
			},
			wantErr:    true,
			stateDebug: false,
		},
		{
			name: "Prev_Block_State_Computed_And_Client_State_Root_Different_With_HashERR",
			fields: fields{
				blockFetcher:         NewBlockFetcher(),
				LatestFinalizedBlock: block.NewBlock("", 2),
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.TODO())
					cancel()
					return ctx
				}(),
				b: func() *block.Block {
					prevB := block.NewBlock("", 1)
					prevB.HashBlock()
					prevB.SetStateStatus(block.StateSuccessful)
					prevB.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1)

					b := block.NewBlock("", 1)
					b.SetPreviousBlock(prevB)
					b.Txns = []*transaction.Transaction{
						{},
					}
					b.ClientStateHash = encryption.RawHash("wrong client state hash")
					b.HashBlock()

					return b
				}(),
			},
			wantErr:    true,
			stateDebug: false,
		},
		{
			name: "OK",
			fields: fields{
				blockFetcher:         NewBlockFetcher(),
				LatestFinalizedBlock: block.NewBlock("", 2),
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.TODO())
					cancel()
					return ctx
				}(),
				b: func() *block.Block {
					prevB := block.NewBlock("", 1)
					prevB.HashBlock()
					prevB.SetStateStatus(block.StateSuccessful)
					prevB.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1)

					b := block.NewBlock("", 1)
					b.SetPreviousBlock(prevB)
					b.Txns = []*transaction.Transaction{
						{},
					}
					b.HashBlock()

					return b
				}(),
			},
			stateDebug: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.stateDebug {
				state.SetDebugLevel(state.DebugLevelBlock)
			} else {
				state.SetDebugLevel(state.DebugLevelNone)
			}

			c := &Chain{
				blocksMutex:          &sync.RWMutex{},
				roundsMutex:          &sync.RWMutex{},
				LatestFinalizedBlock: tt.fields.LatestFinalizedBlock,
				stateMutex:           &sync.RWMutex{},
				blockFetcher:         tt.fields.blockFetcher,
			}
			if err := c.ComputeState(tt.args.ctx, tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("ComputeState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_ComputeOrSyncState(t *testing.T) {
	type fields struct {
		LatestFinalizedBlock *block.Block
		blockFetcher         *BlockFetcher
	}
	type args struct {
		ctx context.Context
		b   *block.Block
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantErr    bool
		stateDebug bool
	}{
		{
			name: "OK",
			fields: fields{
				blockFetcher:         NewBlockFetcher(),
				LatestFinalizedBlock: block.NewBlock("", 2),
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.TODO())
					cancel()
					return ctx
				}(),
				b: func() *block.Block {
					prevB := block.NewBlock("", 1)
					prevB.HashBlock()
					prevB.SetStateStatus(block.StateSuccessful)
					prevB.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1)

					b := block.NewBlock("", 1)
					b.SetPreviousBlock(prevB)
					b.Txns = []*transaction.Transaction{
						{},
					}
					b.HashBlock()

					return b
				}(),
			},
			stateDebug: false,
		},
		{
			name: "Computing_State_Nil_Prev_Block ERR",
			fields: fields{
				blockFetcher:         NewBlockFetcher(),
				LatestFinalizedBlock: block.NewBlock("", 2),
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.TODO())
					cancel()
					return ctx
				}(),
				b: func() *block.Block {
					b := block.NewBlock("", 1)
					b.HashBlock()

					return b
				}(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.stateDebug {
				state.SetDebugLevel(state.DebugLevelBlock)
			} else {
				state.SetDebugLevel(state.DebugLevelNone)
			}

			c := &Chain{
				blocksMutex:          &sync.RWMutex{},
				roundsMutex:          &sync.RWMutex{},
				LatestFinalizedBlock: tt.fields.LatestFinalizedBlock,
				stateMutex:           &sync.RWMutex{},
				blockFetcher:         tt.fields.blockFetcher,
			}
			if err := c.ComputeOrSyncState(tt.args.ctx, tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("ComputeOrSyncState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_SaveChanges(t *testing.T) {
	type fields struct {
		stateDB util.NodeDB
	}
	type args struct {
		ctx context.Context
		b   *block.Block
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "State_Not_Computed_ERR",
			args: args{
				b: func() *block.Block {
					b := block.NewBlock("", 1)
					b.SetStateStatus(block.StateComputing)
					b.HashBlock()

					return b
				}(),
			},
			wantErr: true,
		},
		{
			name: "Nil_Client_State_ERR",
			args: args{
				b: func() *block.Block {
					b := block.NewBlock("", 1)
					b.SetStateStatus(block.StateSuccessful)
					b.ClientState = nil
					b.HashBlock()

					return b
				}(),
			},
			wantErr: true,
		},
		{
			name: "OK",
			fields: fields{
				stateDB: util.NewMemoryNodeDB(),
			},
			args: args{
				b: func() *block.Block {
					b := block.NewBlock("", 1)
					b.SetStateStatus(block.StateSuccessful)
					b.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1)
					b.HashBlock()

					return b
				}(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				stateDB: tt.fields.stateDB,
			}
			if err := c.SaveChanges(tt.args.ctx, tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("SaveChanges() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_UpdateState(t *testing.T) {
	type fields struct {
		IDField                      datastore.IDField
		VersionField                 datastore.VersionField
		CreationDateField            datastore.CreationDateField
		mutexViewChangeMB            sync.RWMutex
		Config                       *Config
		MagicBlockStorage            round.RoundStorage
		PreviousMagicBlock           *block.MagicBlock
		mbMutex                      sync.RWMutex
		LatestFinalizedMagicBlock    *block.Block
		lfmbMutex                    sync.RWMutex
		lfmbSummary                  *block.BlockSummary
		latestOwnFinalizedBlockRound int64
		blocks                       map[datastore.Key]*block.Block
		blocksMutex                  *sync.RWMutex
		rounds                       map[int64]round.RoundI
		roundsMutex                  *sync.RWMutex
		CurrentRound                 int64
		FeeStats                     transaction.TransactionFeeStats
		LatestFinalizedBlock         *block.Block
		lfbMutex                     sync.RWMutex
		lfbSummary                   *block.BlockSummary
		LatestDeterministicBlock     *block.Block
		clientStateDeserializer      state.DeserializerI
		stateDB                      util.NodeDB
		stateMutex                   *sync.RWMutex
		finalizedRoundsChannel       chan round.RoundI
		finalizedBlocksChannel       chan *block.Block
		Stats                        *Stats
		BlockChain                   *ring.Ring
		minersStake                  map[datastore.Key]int
		stakeMutex                   *sync.Mutex
		nodePoolScorer               node.PoolScorer
		GenerateTimeout              int
		genTimeoutMutex              *sync.Mutex
		syncStateTimeout             time.Duration
		bcStuckCheckInterval         time.Duration
		bcStuckTimeThreshold         time.Duration
		retry_wait_time              int
		retry_wait_mutex             *sync.Mutex
		blockFetcher                 *BlockFetcher
		crtCount                     int64
		fetchedNotarizedBlockHandler FetchedNotarizedBlockHandler
		viewChanger                  ViewChanger
		afterFetcher                 AfterFetcher
		magicBlockSaver              MagicBlockSaver
		pruneStats                   *util.PruneStats
		configInfoDB                 string
		configInfoStore              datastore.Store
		RoundF                       round.RoundFactory
		magicBlockStartingRounds     map[int64]*block.Block
		getLFBTicket                 chan *LFBTicket
		updateLFBTicket              chan *LFBTicket
		broadcastLFBTicket           chan *block.Block
		subLFBTicket                 chan chan *LFBTicket
		unsubLFBTicket               chan chan *LFBTicket
		lfbTickerWorkerIsDone        chan struct{}
		syncLFBStateC                chan *block.BlockSummary
		phaseEvents                  chan PhaseEvent
	}
	type args struct {
		b   *block.Block
		txn *transaction.Transaction
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Invalid_Transaction_Type_ERR",
			args: args{
				txn: &transaction.Transaction{
					TransactionType: 57,
				},
				b: func() *block.Block {
					b := block.NewBlock("", 1)
					b.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1)
					b.HashBlock()

					return b
				}(),
			},
			wantErr: true,
		},
		{
			name: "Invalid_State_Context_ERR",
			args: args{
				txn: &transaction.Transaction{
					TransactionType: transaction.TxnTypeData,
					ToClientID:      "unknown to client id",
					Value:           -1,
				},
				b: func() *block.Block {
					b := block.NewBlock("", 1)
					b.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1)
					b.HashBlock()

					return b
				}(),
			},
			wantErr: true,
		},
		{
			name: "Add_Transfer_ERR",
			args: args{
				txn: &transaction.Transaction{
					TransactionType: transaction.TxnTypeData,
					ToClientID:      "unknown to client id",
					Fee:             1,
				},
				b: func() *block.Block {
					b := block.NewBlock("", 1)
					b.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1)
					b.HashBlock()

					return b
				}(),
			},
			wantErr: true,
		},
		//{ TODO
		//	name: "_ERR",
		//	args: args{
		//		txn: &transaction.Transaction{
		//			TransactionType: transaction.TxnTypeData,
		//			ToClientID: "unknown to client id",
		//		},
		//		b: func() *block.Block{
		//			b := block.NewBlock("", 1)
		//			b.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1)
		//			b.HashBlock()
		//
		//			return b
		//		}(),
		//	},
		//	wantErr: true,
		//},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				IDField:                      tt.fields.IDField,
				VersionField:                 tt.fields.VersionField,
				CreationDateField:            tt.fields.CreationDateField,
				mutexViewChangeMB:            tt.fields.mutexViewChangeMB,
				Config:                       tt.fields.Config,
				MagicBlockStorage:            tt.fields.MagicBlockStorage,
				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
				mbMutex:                      tt.fields.mbMutex,
				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
				lfmbMutex:                    tt.fields.lfmbMutex,
				lfmbSummary:                  tt.fields.lfmbSummary,
				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
				blocks:                       tt.fields.blocks,
				blocksMutex:                  tt.fields.blocksMutex,
				rounds:                       tt.fields.rounds,
				roundsMutex:                  tt.fields.roundsMutex,
				CurrentRound:                 tt.fields.CurrentRound,
				FeeStats:                     tt.fields.FeeStats,
				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
				lfbMutex:                     tt.fields.lfbMutex,
				lfbSummary:                   tt.fields.lfbSummary,
				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
				clientStateDeserializer:      tt.fields.clientStateDeserializer,
				stateDB:                      tt.fields.stateDB,
				stateMutex:                   &sync.RWMutex{},
				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
				Stats:                        tt.fields.Stats,
				BlockChain:                   tt.fields.BlockChain,
				minersStake:                  tt.fields.minersStake,
				stakeMutex:                   tt.fields.stakeMutex,
				nodePoolScorer:               tt.fields.nodePoolScorer,
				GenerateTimeout:              tt.fields.GenerateTimeout,
				genTimeoutMutex:              tt.fields.genTimeoutMutex,
				syncStateTimeout:             tt.fields.syncStateTimeout,
				bcStuckCheckInterval:         tt.fields.bcStuckCheckInterval,
				bcStuckTimeThreshold:         tt.fields.bcStuckTimeThreshold,
				retry_wait_time:              tt.fields.retry_wait_time,
				retry_wait_mutex:             tt.fields.retry_wait_mutex,
				blockFetcher:                 tt.fields.blockFetcher,
				crtCount:                     tt.fields.crtCount,
				fetchedNotarizedBlockHandler: tt.fields.fetchedNotarizedBlockHandler,
				viewChanger:                  tt.fields.viewChanger,
				afterFetcher:                 tt.fields.afterFetcher,
				magicBlockSaver:              tt.fields.magicBlockSaver,
				pruneStats:                   tt.fields.pruneStats,
				configInfoDB:                 tt.fields.configInfoDB,
				configInfoStore:              tt.fields.configInfoStore,
				RoundF:                       tt.fields.RoundF,
				magicBlockStartingRounds:     tt.fields.magicBlockStartingRounds,
				getLFBTicket:                 tt.fields.getLFBTicket,
				updateLFBTicket:              tt.fields.updateLFBTicket,
				broadcastLFBTicket:           tt.fields.broadcastLFBTicket,
				subLFBTicket:                 tt.fields.subLFBTicket,
				unsubLFBTicket:               tt.fields.unsubLFBTicket,
				lfbTickerWorkerIsDone:        tt.fields.lfbTickerWorkerIsDone,
				syncLFBStateC:                tt.fields.syncLFBStateC,
				phaseEvents:                  tt.fields.phaseEvents,
			}
			if err := c.UpdateState(tt.args.b, tt.args.txn); (err != nil) != tt.wantErr {
				t.Errorf("UpdateState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
