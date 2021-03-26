package chain

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
	"0chain.net/core/util"
	"container/ring"
	"reflect"
	"sync"
	"testing"
	"time"
)

func init() {
	sp := memorystore.GetStorageProvider()
	block.SetupEntity(sp)
	round.SetupEntity(sp)
}

func TestInfo_GetKey(t *testing.T) {
	type fields struct {
		TimeStamp       *time.Time
		FinalizedRound  int64
		FinalizedCount  int64
		BlockHash       string
		ClientStateHash util.Key
		ChainWeight     float64
	}
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		{
			name:   "OK",
			fields: fields{FinalizedRound: 1},
			want:   1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &Info{
				TimeStamp:       tt.fields.TimeStamp,
				FinalizedRound:  tt.fields.FinalizedRound,
				FinalizedCount:  tt.fields.FinalizedCount,
				BlockHash:       tt.fields.BlockHash,
				ClientStateHash: tt.fields.ClientStateHash,
				ChainWeight:     tt.fields.ChainWeight,
			}
			if got := info.GetKey(); got != tt.want {
				t.Errorf("GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInfo_GetTime(t *testing.T) {
	ts := time.Now()

	type fields struct {
		TimeStamp       *time.Time
		FinalizedRound  int64
		FinalizedCount  int64
		BlockHash       string
		ClientStateHash util.Key
		ChainWeight     float64
	}
	tests := []struct {
		name   string
		fields fields
		want   *time.Time
	}{
		{
			name:   "OK",
			fields: fields{TimeStamp: &ts},
			want:   &ts,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &Info{
				TimeStamp:       tt.fields.TimeStamp,
				FinalizedRound:  tt.fields.FinalizedRound,
				FinalizedCount:  tt.fields.FinalizedCount,
				BlockHash:       tt.fields.BlockHash,
				ClientStateHash: tt.fields.ClientStateHash,
				ChainWeight:     tt.fields.ChainWeight,
			}
			if got := info.GetTime(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_UpdateChainInfo(t *testing.T) {
	type fields struct {
		IDField                      datastore.IDField
		VersionField                 datastore.VersionField
		CreationDateField            datastore.CreationDateField
		Config                       *Config
		MagicBlockStorage            round.RoundStorage
		PreviousMagicBlock           *block.MagicBlock
		LatestFinalizedMagicBlock    *block.Block
		lfmbSummary                  *block.BlockSummary
		latestOwnFinalizedBlockRound int64
		blocks                       map[datastore.Key]*block.Block
		rounds                       map[int64]round.RoundI
		CurrentRound                 int64
		FeeStats                     transaction.TransactionFeeStats
		LatestFinalizedBlock         *block.Block
		lfbSummary                   *block.BlockSummary
		LatestDeterministicBlock     *block.Block
		clientStateDeserializer      state.DeserializerI
		stateDB                      util.NodeDB
		finalizedRoundsChannel       chan round.RoundI
		finalizedBlocksChannel       chan *block.Block
		Stats                        *Stats
		BlockChain                   *ring.Ring
		minersStake                  map[datastore.Key]int
		nodePoolScorer               node.PoolScorer
		GenerateTimeout              int
		retry_wait_time              int
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
		phaseEvents                  chan PhaseEvent
	}
	type args struct {
		b *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "OK",
			args: args{b: block.NewBlock("", 1)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				IDField:                      tt.fields.IDField,
				VersionField:                 tt.fields.VersionField,
				CreationDateField:            tt.fields.CreationDateField,
				Config:                       tt.fields.Config,
				MagicBlockStorage:            tt.fields.MagicBlockStorage,
				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
				lfmbSummary:                  tt.fields.lfmbSummary,
				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
				blocks:                       tt.fields.blocks,
				rounds:                       tt.fields.rounds,
				CurrentRound:                 tt.fields.CurrentRound,
				FeeStats:                     tt.fields.FeeStats,
				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
				lfbSummary:                   tt.fields.lfbSummary,
				LatestDeterministicBlock:     tt.fields.LatestDeterministicBlock,
				clientStateDeserializer:      tt.fields.clientStateDeserializer,
				stateDB:                      tt.fields.stateDB,
				finalizedRoundsChannel:       tt.fields.finalizedRoundsChannel,
				finalizedBlocksChannel:       tt.fields.finalizedBlocksChannel,
				Stats:                        tt.fields.Stats,
				BlockChain:                   tt.fields.BlockChain,
				minersStake:                  tt.fields.minersStake,
				nodePoolScorer:               tt.fields.nodePoolScorer,
				GenerateTimeout:              tt.fields.GenerateTimeout,
				retry_wait_time:              tt.fields.retry_wait_time,
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
				phaseEvents:                  tt.fields.phaseEvents,
			}

			c.UpdateChainInfo(tt.args.b)
		})
	}
}

func TestChain_UpdateRoundInfo(t *testing.T) {
	ch, ok := Provider().(*Chain)
	if !ok {
		t.Error("can`t use another entity")
	}

	type fields struct {
		IDField                      datastore.IDField
		VersionField                 datastore.VersionField
		CreationDateField            datastore.CreationDateField
		Config                       *Config
		MagicBlockStorage            round.RoundStorage
		PreviousMagicBlock           *block.MagicBlock
		LatestFinalizedMagicBlock    *block.Block
		lfmbSummary                  *block.BlockSummary
		latestOwnFinalizedBlockRound int64
		blocks                       map[datastore.Key]*block.Block
		rounds                       map[int64]round.RoundI
		CurrentRound                 int64
		FeeStats                     transaction.TransactionFeeStats
		LatestFinalizedBlock         *block.Block
		lfbSummary                   *block.BlockSummary
		LatestDeterministicBlock     *block.Block
		clientStateDeserializer      state.DeserializerI
		stateDB                      util.NodeDB
		finalizedRoundsChannel       chan round.RoundI
		finalizedBlocksChannel       chan *block.Block
		Stats                        *Stats
		BlockChain                   *ring.Ring
		minersStake                  map[datastore.Key]int
		nodePoolScorer               node.PoolScorer
		GenerateTimeout              int
		retry_wait_time              int
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
		phaseEvents                  chan PhaseEvent
	}
	type args struct {
		r round.RoundI
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "OK",
			args: args{r: round.NewRound(1)},
			fields: fields{
				IDField:                      ch.IDField,
				VersionField:                 ch.VersionField,
				CreationDateField:            ch.CreationDateField,
				Config:                       ch.Config,
				MagicBlockStorage:            ch.MagicBlockStorage,
				PreviousMagicBlock:           ch.PreviousMagicBlock,
				LatestFinalizedMagicBlock:    ch.LatestFinalizedMagicBlock,
				lfmbSummary:                  ch.lfbSummary,
				latestOwnFinalizedBlockRound: ch.latestOwnFinalizedBlockRound,
				blocks:                       ch.blocks,
				rounds:                       ch.rounds,
				CurrentRound:                 ch.CurrentRound,
				FeeStats:                     ch.FeeStats,
				LatestFinalizedBlock:         ch.LatestFinalizedBlock,
				lfbSummary:                   ch.lfbSummary,
				LatestDeterministicBlock:     ch.LatestDeterministicBlock,
				clientStateDeserializer:      ch.clientStateDeserializer,
				stateDB:                      ch.stateDB,
				finalizedRoundsChannel:       ch.finalizedRoundsChannel,
				finalizedBlocksChannel:       ch.finalizedBlocksChannel,
				Stats:                        ch.Stats,
				BlockChain:                   ch.BlockChain,
				minersStake:                  ch.minersStake,
				nodePoolScorer:               ch.nodePoolScorer,
				GenerateTimeout:              ch.GenerateTimeout,
				retry_wait_time:              ch.retry_wait_time,
				blockFetcher:                 ch.blockFetcher,
				crtCount:                     ch.crtCount,
				fetchedNotarizedBlockHandler: ch.fetchedNotarizedBlockHandler,
				viewChanger:                  ch.viewChanger,
				afterFetcher:                 ch.afterFetcher,
				magicBlockSaver:              ch.magicBlockSaver,
				pruneStats:                   ch.pruneStats,
				configInfoDB:                 ch.configInfoDB,
				configInfoStore:              ch.configInfoStore,
				RoundF:                       ch.RoundF,
				magicBlockStartingRounds:     ch.magicBlockStartingRounds,
				getLFBTicket:                 ch.getLFBTicket,
				updateLFBTicket:              ch.updateLFBTicket,
				broadcastLFBTicket:           ch.broadcastLFBTicket,
				subLFBTicket:                 ch.subLFBTicket,
				unsubLFBTicket:               ch.unsubLFBTicket,
				lfbTickerWorkerIsDone:        ch.lfbTickerWorkerIsDone,
				phaseEvents:                  ch.phaseEvents,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				IDField:                      tt.fields.IDField,
				VersionField:                 tt.fields.VersionField,
				CreationDateField:            tt.fields.CreationDateField,
				Config:                       tt.fields.Config,
				MagicBlockStorage:            tt.fields.MagicBlockStorage,
				PreviousMagicBlock:           tt.fields.PreviousMagicBlock,
				LatestFinalizedMagicBlock:    tt.fields.LatestFinalizedMagicBlock,
				lfmbSummary:                  tt.fields.lfmbSummary,
				latestOwnFinalizedBlockRound: tt.fields.latestOwnFinalizedBlockRound,
				blocks:                       tt.fields.blocks,
				blocksMutex:                  &sync.RWMutex{},
				rounds:                       tt.fields.rounds,
				roundsMutex:                  &sync.RWMutex{},
				CurrentRound:                 tt.fields.CurrentRound,
				FeeStats:                     tt.fields.FeeStats,
				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
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
				stakeMutex:                   &sync.Mutex{},
				nodePoolScorer:               tt.fields.nodePoolScorer,
				GenerateTimeout:              tt.fields.GenerateTimeout,
				genTimeoutMutex:              &sync.Mutex{},
				retry_wait_time:              tt.fields.retry_wait_time,
				retry_wait_mutex:             &sync.Mutex{},
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
				phaseEvents:                  tt.fields.phaseEvents,
			}

			c.UpdateRoundInfo(tt.args.r)
		})
	}
}
