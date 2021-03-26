package chain

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"container/ring"
	"context"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestChain_ComputeFinalizedBlock(t *testing.T) {
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
		ctx context.Context
		r   round.RoundI
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *block.Block
	}{
		// TODO: Add test cases.
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
				stateMutex:                   tt.fields.stateMutex,
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
			if got := c.ComputeFinalizedBlock(tt.args.ctx, tt.args.r); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ComputeFinalizedBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_ComputeFinalizedBlock1(t *testing.T) {
	b1 := block.NewBlock("", 0)
	b1.HashBlock()
	b2 := block.NewBlock("", 1)
	b2.HashBlock()
	b2.PrevHash = b1.Hash
	b3 := block.NewBlock("", 2)
	b3.HashBlock()
	b3.PrevBlock = b2
	b4 := block.NewBlock("", 3)
	b4.HashBlock()
	b4.PrevBlock = b3

	type fields struct {
		blocks       map[datastore.Key]*block.Block
		blockFetcher *BlockFetcher
	}
	type args struct {
		ctx context.Context
		r   round.RoundI
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *block.Block
	}{
		{
			name: "Nil_Result_OK",
			args: args{
				r: round.NewRound(1),
			},
			want: nil,
		},
		{
			name: "Nil_Previous_Block_OK",
			fields: fields{
				blocks: map[datastore.Key]*block.Block{
					b2.PrevHash: b1,
				},
				blockFetcher: NewBlockFetcher(),
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.TODO())
					cancel()

					return ctx
				}(),
				r: func() *round.Round {
					r := round.NewRound(1)
					r.AddNotarizedBlock(b2)

					return r
				}(),
			},
			want: nil,
		},
		{
			name: "Tips_Not_Valid_OK",
			fields: fields{
				blocks:       map[datastore.Key]*block.Block{},
				blockFetcher: NewBlockFetcher(),
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.TODO())
					cancel()

					return ctx
				}(),
				r: func() *round.Round {
					r := round.NewRound(2)
					r.AddNotarizedBlock(b3)
					r.AddNotarizedBlock(b4)

					return r
				}(),
			},
			want: nil,
		},
		{
			name: "OK",
			fields: fields{
				blocks:       map[datastore.Key]*block.Block{},
				blockFetcher: NewBlockFetcher(),
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.TODO())
					cancel()

					return ctx
				}(),
				r: func() *round.Round {
					r := round.NewRound(3)
					r.AddNotarizedBlock(b3)
					r.AddNotarizedBlock(b4)

					return r
				}(),
			},
			want: b3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				blocks:       tt.fields.blocks,
				blocksMutex:  &sync.RWMutex{},
				roundsMutex:  &sync.RWMutex{},
				blockFetcher: tt.fields.blockFetcher,
			}
			if got := c.ComputeFinalizedBlock(tt.args.ctx, tt.args.r); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ComputeFinalizedBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}
