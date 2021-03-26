package chain

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	chmocks "0chain.net/mocks/chaincore/chain"
	mocks "0chain.net/mocks/core/datastore"
	"context"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"os"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"
)

func init() {
	config.SetupDefaultConfig()
	config.SetupConfig()
	//config.SetupSmartContractConfig()

	block.SetupEntity(&mocks.Store{})
	block.SetupBlockSummaryEntity(&mocks.Store{})

	round.SetupEntity(&mocks.Store{})

	logging.InitLogging("testing")

	if err := os.MkdirAll("data/rocksdb/state", 0700); err != nil {
		panic(err)
	}
	SetupEntity(&mocks.Store{})
	CloseStateDB()
	if err := os.RemoveAll("data"); err != nil {
		panic(err)
	}
}

func TestChain_Validate(t *testing.T) {
	type fields struct {
		IDField datastore.IDField
		Config  *Config
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Empty_ID_ERR",
			fields: fields{
				IDField: datastore.IDField{ID: ""},
			},
			wantErr: true,
		},
		{
			name: "Empty_Owner_ID_ERR",
			fields: fields{
				IDField: datastore.IDField{ID: "id"},
				Config:  &Config{OwnerID: ""},
			},
			wantErr: true,
		},
		{
			name: "OK",
			fields: fields{
				IDField: datastore.IDField{ID: "id"},
				Config:  &Config{OwnerID: "owner id"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				IDField:           tt.fields.IDField,
				mutexViewChangeMB: sync.RWMutex{},
				Config:            tt.fields.Config,
				mbMutex:           sync.RWMutex{},
				lfmbMutex:         sync.RWMutex{},
				blocksMutex:       &sync.RWMutex{},
				roundsMutex:       &sync.RWMutex{},
				lfbMutex:          sync.RWMutex{},
				stateMutex:        &sync.RWMutex{},
				stakeMutex:        &sync.Mutex{},
				genTimeoutMutex:   &sync.Mutex{},
				retry_wait_mutex:  &sync.Mutex{},
			}
			if err := c.Validate(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewChainFromConfig(t *testing.T) {
	// setup config and ch
	ch := Provider().(*Chain)

	config.Configuration.ChainID = "0afc093ffb509f059c55478bc1a60351cef7b4e9c008a53a6cc8241ca8617dfe"
	ch.ID = datastore.ToKey(config.Configuration.ChainID)
	ch.Decimals = int8(viper.GetInt("server_chain.decimals"))
	ch.BlockSize = viper.GetInt32("server_chain.block.max_block_size")
	ch.MinBlockSize = viper.GetInt32("server_chain.block.min_block_size")
	ch.MaxByteSize = viper.GetInt64("server_chain.block.max_byte_size")
	ch.NumGenerators = viper.GetInt("server_chain.block.generators")
	ch.NotariedBlocksCounts = make([]int64, ch.NumGenerators+1)
	ch.NumReplicators = viper.GetInt("server_chain.block.replicators")
	ch.ThresholdByCount = viper.GetInt("server_chain.block.consensus.threshold_by_count")
	ch.ThresholdByStake = viper.GetInt("server_chain.block.consensus.threshold_by_stake")
	ch.OwnerID = viper.GetString("server_chain.owner")
	ch.ValidationBatchSize = viper.GetInt("server_chain.block.validation.batch_size")
	ch.RoundRange = viper.GetInt64("server_chain.round_range")
	ch.TxnMaxPayload = viper.GetInt("server_chain.transaction.payload.max_size")
	ch.PruneStateBelowCount = viper.GetInt("server_chain.state.prune_below_count")
	verificationTicketsTo := viper.GetString("server_chain.messages.verification_tickets_to")
	if verificationTicketsTo == "" || verificationTicketsTo == "all_miners" || verificationTicketsTo == "11" {
		ch.VerificationTicketsTo = AllMiners
	} else {
		ch.VerificationTicketsTo = Generator
	}

	conf := &ch.HCCycleScan[DeepScan]

	conf.Enabled = viper.GetBool("server_chain.health_check.deep_scan.enabled")
	conf.BatchSize = viper.GetInt64("server_chain.health_check.deep_scan.batch_size")
	conf.Window = viper.GetInt64("server_chain.health_check.deep_scan.window")

	conf.SettleSecs = viper.GetInt("server_chain.health_check.deep_scan.settle_secs")
	conf.Settle = time.Duration(conf.SettleSecs) * time.Second

	conf.RepeatIntervalMins = viper.GetInt("server_chain.health_check.deep_scan.repeat_interval_mins")
	conf.RepeatInterval = time.Duration(conf.RepeatIntervalMins) * time.Minute

	conf.ReportStatusMins = viper.GetInt("server_chain.health_check.deep_scan.report_status_mins")
	conf.ReportStatus = time.Duration(conf.ReportStatusMins) * time.Minute

	conf = &ch.HCCycleScan[ProximityScan]

	conf.Enabled = viper.GetBool("server_chain.health_check.proximity_scan.enabled")
	conf.BatchSize = viper.GetInt64("server_chain.health_check.proximity_scan.batch_size")
	conf.Window = viper.GetInt64("server_chain.health_check.proximity_scan.window")

	conf.SettleSecs = viper.GetInt("server_chain.health_check.proximity_scan.settle_secs")
	conf.Settle = time.Duration(conf.SettleSecs) * time.Second

	conf.RepeatIntervalMins = viper.GetInt("server_chain.health_check.proximity_scan.repeat_interval_mins")
	conf.RepeatInterval = time.Duration(conf.RepeatIntervalMins) * time.Minute

	conf.ReportStatusMins = viper.GetInt("server_chain.health_check.proximity_scan.report_status_mins")
	conf.ReportStatus = time.Duration(conf.ReportStatusMins) * time.Minute

	ch.HealthShowCounters = viper.GetBool("server_chain.health_check.show_counters")

	ch.BlockProposalMaxWaitTime = viper.GetDuration("server_chain.block.proposal.max_wait_time") * time.Millisecond
	waitMode := viper.GetString("server_chain.block.proposal.wait_mode")
	if waitMode == "static" {
		ch.BlockProposalWaitMode = BlockProposalWaitStatic
	} else if waitMode == "dynamic" {
		ch.BlockProposalWaitMode = BlockProposalWaitDynamic
	}
	ch.ReuseTransactions = viper.GetBool("server_chain.block.reuse_txns")
	ch.SetSignatureScheme(viper.GetString("server_chain.client.signature_scheme"))

	ch.MinActiveSharders = viper.GetInt("server_chain.block.sharding.min_active_sharders")
	ch.MinActiveReplicators = viper.GetInt("server_chain.block.sharding.min_active_replicators")
	ch.SmartContractTimeout = viper.GetDuration("server_chain.smart_contract.timeout") * time.Millisecond
	ch.RoundTimeoutSofttoMin = viper.GetInt("server_chain.round_timeouts.softto_min")
	ch.RoundTimeoutSofttoMult = viper.GetInt("server_chain.round_timeouts.softto_mult")
	ch.RoundRestartMult = viper.GetInt("server_chain.round_timeouts.round_restart_mult")

	tests := []struct {
		name string
		want *Chain
	}{
		{
			name: "OK",
			want: ch,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewChainFromConfig()

			// clean up channels and mutexes due to impossibility of comparasion
			tt.want.blocksMutex = nil
			got.blocksMutex = nil
			tt.want.roundsMutex = nil
			got.roundsMutex = nil
			tt.want.finalizedRoundsChannel = nil
			got.finalizedRoundsChannel = nil
			tt.want.finalizedBlocksChannel = nil
			got.finalizedBlocksChannel = nil
			tt.want.blockFetcher = nil
			got.blockFetcher = nil
			tt.want.getLFBTicket = nil
			got.getLFBTicket = nil
			tt.want.updateLFBTicket = nil
			got.updateLFBTicket = nil
			tt.want.broadcastLFBTicket = nil
			got.broadcastLFBTicket = nil
			tt.want.subLFBTicket = nil
			got.subLFBTicket = nil
			tt.want.unsubLFBTicket = nil
			got.unsubLFBTicket = nil
			tt.want.lfbTickerWorkerIsDone = nil
			got.lfbTickerWorkerIsDone = nil
			tt.want.syncLFBStateC = nil
			got.syncLFBStateC = nil
			tt.want.phaseEvents = nil
			got.phaseEvents = nil

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestChain_Simple_Setters_And_Getters(t *testing.T) {
	ch := NewChainFromConfig()

	threshold := time.Second
	ch.SetBCStuckTimeThreshold(threshold)
	assert.Equal(t, threshold, ch.bcStuckTimeThreshold)

	interval := time.Second
	ch.SetBCStuckCheckInterval(interval)
	assert.Equal(t, interval, ch.bcStuckCheckInterval)

	syncStateTimeout := time.Second
	ch.SetSyncStateTimeout(interval)
	assert.Equal(t, syncStateTimeout, ch.syncStateTimeout)

	ch.configInfoDB = "config info db"
	assert.Equal(t, ch.configInfoDB, ch.GetConfigInfoDB())

	ch.configInfoStore = &mocks.Store{}
	assert.Equal(t, ch.configInfoStore, ch.GetConfigInfoStore())

	generationTimeout := 10
	ch.SetGenerationTimeout(generationTimeout)
	assert.Equal(t, generationTimeout, ch.GetGenerationTimeout())

	retryWaitTime := 10
	ch.SetRetryWaitTime(retryWaitTime)
	assert.Equal(t, retryWaitTime, ch.GetRetryWaitTime())

	ch.ResetRoundTimeoutCount()
	assert.Equal(t, int64(0), ch.crtCount)

	ch.IncrementRoundTimeoutCount()
	assert.Equal(t, int64(1), ch.crtCount)

	assert.Equal(t, int64(1), ch.GetRoundTimeoutCount())

	ch.SetSignatureScheme("ed25519")
	assert.Equal(t, encryption.NewED25519Scheme(), ch.GetSignatureScheme())

	fnbh := &chmocks.FetchedNotarizedBlockHandler{}
	ch.SetFetchedNotarizedBlockHandler(fnbh)
	assert.Equal(t, fnbh, ch.fetchedNotarizedBlockHandler)

	viewChanger := &chmocks.ViewChanger{}
	ch.SetViewChanger(viewChanger)
	assert.Equal(t, viewChanger, ch.viewChanger)

	afterFetcher := &chmocks.AfterFetcher{}
	ch.SetAfterFetcher(afterFetcher)
	assert.Equal(t, afterFetcher, ch.afterFetcher)

	pruneStats := &util.PruneStats{}
	ch.pruneStats = pruneStats
	assert.Equal(t, pruneStats, ch.GetPruneStats())

	r := int64(1)
	ch.SetLatestOwnFinalizedBlockRound(r)
	assert.Equal(t, r, ch.LatestOwnFinalizedBlockRound())

	lfmb := block.NewBlock("", 1)
	ch.LatestFinalizedMagicBlock = lfmb
	assert.Equal(t, lfmb, ch.GetLatestFinalizedMagicBlock())

	ch.lfmbSummary = lfmb.GetSummary()
	assert.Equal(t, lfmb.GetSummary(), ch.GetLatestFinalizedMagicBlockSummary())

	SetServerChain(ch)
	assert.Equal(t, ch, GetServerChain())

	currRound := int64(5)
	ch.SetCurrentRound(currRound)
	assert.Equal(t, currRound, ch.GetCurrentRound())

	ch.lfbSummary = &block.BlockSummary{}
	assert.Equal(t, ch.lfbSummary, ch.GetLatestFinalizedBlockSummary())
}

func TestChain_GetBlockStateNode(t *testing.T) {
	db := util.NewMemoryNodeDB()

	value := &util.SecureSerializableValue{Buffer: []byte("data")}
	n := util.NewLeafNode(util.Path(value.GetHash()), 0, value)
	if err := db.PutNode(n.GetHashBytes(), n); err != nil {
		t.Fatal(err)
	}

	mpt := util.NewMerklePatriciaTrie(db, 1)
	mpt.Root = db.ComputeRoot().GetHashBytes()

	b := block.NewBlock("", 1)
	b.ClientState = mpt

	type args struct {
		block *block.Block
		path  string
	}
	tests := []struct {
		name     string
		args     args
		wantSeri util.Serializable
		wantErr  bool
	}{
		{
			name: "Nil_Client_Block_State_ERR",
			args: args{
				block: &block.Block{
					ClientState: nil,
				},
			},
			wantErr: true,
		},
		{
			name: "OK",
			args: args{
				block: b,
				path:  string(value.Encode()),
			},
			wantSeri: value,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &Chain{
				mutexViewChangeMB: sync.RWMutex{},
				mbMutex:           sync.RWMutex{},
				lfmbMutex:         sync.RWMutex{},
				blocksMutex:       &sync.RWMutex{},
				roundsMutex:       &sync.RWMutex{},
				lfbMutex:          sync.RWMutex{},
				stateMutex:        &sync.RWMutex{},
				stakeMutex:        &sync.Mutex{},
				genTimeoutMutex:   &sync.Mutex{},
				retry_wait_mutex:  &sync.Mutex{},
			}
			gotSeri, err := mc.GetBlockStateNode(tt.args.block, tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBlockStateNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.wantSeri, gotSeri)
			}
		})
	}
}

func Test_mbRoundOffset(t *testing.T) {
	type args struct {
		rn int64
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "Same_OK",
			args: args{
				rn: ViewChangeOffset - 1,
			},
			want: ViewChangeOffset - 1,
		},
		{
			name: "OK",
			args: args{
				rn: ViewChangeOffset + 1,
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mbRoundOffset(tt.args.rn); got != tt.want {
				t.Errorf("mbRoundOffset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetCurrentMagicBlock(t *testing.T) {
	lmb := block.NewMagicBlock()
	lmb.Hash = encryption.Hash("magic block data")

	mbs := round.NewRoundStartingStorage()
	if err := mbs.Put(lmb, 0); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		MagicBlockStorage round.RoundStorage
		CurrentRound      int64
	}
	tests := []struct {
		name   string
		fields fields
		want   *block.MagicBlock
	}{
		{
			name: "Zero_Round_OK",
			fields: fields{
				CurrentRound:      0,
				MagicBlockStorage: mbs,
			},
			want: lmb,
		},
		{
			name: "OK",
			fields: fields{
				CurrentRound:      1,
				MagicBlockStorage: mbs,
			},
			want: lmb,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				mutexViewChangeMB: sync.RWMutex{},
				MagicBlockStorage: tt.fields.MagicBlockStorage,
				mbMutex:           sync.RWMutex{},
				lfmbMutex:         sync.RWMutex{},
				blocksMutex:       &sync.RWMutex{},
				roundsMutex:       &sync.RWMutex{},
				CurrentRound:      tt.fields.CurrentRound,
				lfbMutex:          sync.RWMutex{},
				stateMutex:        &sync.RWMutex{},
				stakeMutex:        &sync.Mutex{},
				genTimeoutMutex:   &sync.Mutex{},
				retry_wait_mutex:  &sync.Mutex{},
			}
			if got := c.GetCurrentMagicBlock(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCurrentMagicBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetLatestMagicBlock(t *testing.T) {
	lmb := block.NewMagicBlock()
	lmb.Hash = encryption.Hash("magic block data")

	mbs := round.NewRoundStartingStorage()
	if err := mbs.Put(lmb, 0); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		MagicBlockStorage round.RoundStorage
	}
	tests := []struct {
		name      string
		fields    fields
		want      *block.MagicBlock
		wantPanic bool
	}{
		{
			name: "OK",
			fields: fields{
				MagicBlockStorage: mbs,
			},
			want: lmb,
		},
		{
			name: "PANIC",
			fields: fields{
				MagicBlockStorage: func() round.RoundStorage {
					mbs := round.NewRoundStartingStorage()
					if err := mbs.Put(nil, 0); err != nil {
						t.Fatal(err)
					}

					return mbs
				}(),
			},
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("GetLatestMagicBlock() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			c := &Chain{
				mutexViewChangeMB: sync.RWMutex{},
				MagicBlockStorage: tt.fields.MagicBlockStorage,
				mbMutex:           sync.RWMutex{},
				lfmbMutex:         sync.RWMutex{},
				blocksMutex:       &sync.RWMutex{},
				lfbMutex:          sync.RWMutex{},
				stateMutex:        &sync.RWMutex{},
				stakeMutex:        &sync.Mutex{},
				genTimeoutMutex:   &sync.Mutex{},
				retry_wait_mutex:  &sync.Mutex{},
			}
			if got := c.GetLatestMagicBlock(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetLatestMagicBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetPrevMagicBlock(t *testing.T) {
	mb := block.NewMagicBlock()
	mb.Hash = encryption.Hash("magic block data")
	mb2 := block.NewMagicBlock()
	mb2.Hash = encryption.Hash("magic block 2 data")

	prevMB := block.NewMagicBlock()
	prevMB.Hash = encryption.Hash("prev mb block data")

	type fields struct {
		MagicBlockStorage  round.RoundStorage
		PreviousMagicBlock *block.MagicBlock
	}
	type args struct {
		round int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *block.MagicBlock
	}{
		{
			name: "Prev_From_Chain_OK",
			fields: fields{
				MagicBlockStorage: func() round.RoundStorage {
					mbs := round.NewRoundStartingStorage()
					if err := mbs.Put(mb, 0); err != nil {
						t.Fatal(err)
					}

					return mbs
				}(),
				PreviousMagicBlock: prevMB,
			},
			args: args{round: 0},
			want: prevMB,
		},
		{
			name: "Nil_Entity_From_Storage_OK",
			fields: fields{
				MagicBlockStorage: func() round.RoundStorage {
					mbs := round.NewRoundStartingStorage()
					if err := mbs.Put(nil, 0); err != nil {
						t.Fatal(err)
					}
					if err := mbs.Put(mb2, 1); err != nil {
						t.Fatal(err)
					}

					return mbs
				}(),
				PreviousMagicBlock: prevMB,
			},
			args: args{round: 1},
			want: prevMB,
		},
		{
			name: "OK",
			fields: fields{
				MagicBlockStorage: func() round.RoundStorage {
					mbs := round.NewRoundStartingStorage()
					if err := mbs.Put(mb, 0); err != nil {
						t.Fatal(err)
					}
					if err := mbs.Put(mb2, 1); err != nil {
						t.Fatal(err)
					}

					return mbs
				}(),
			},
			args: args{round: 1},
			want: mb,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				mutexViewChangeMB:  sync.RWMutex{},
				MagicBlockStorage:  tt.fields.MagicBlockStorage,
				PreviousMagicBlock: tt.fields.PreviousMagicBlock,
				mbMutex:            sync.RWMutex{},
				lfmbMutex:          sync.RWMutex{},
				blocksMutex:        &sync.RWMutex{},
				lfbMutex:           sync.RWMutex{},
				stateMutex:         &sync.RWMutex{},
				stakeMutex:         &sync.Mutex{},
				genTimeoutMutex:    &sync.Mutex{},
				retry_wait_mutex:   &sync.Mutex{},
			}
			if got := c.GetPrevMagicBlock(tt.args.round); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPrevMagicBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetMagicBlock(t *testing.T) {
	mb := block.NewMagicBlock()
	mb.Hash = encryption.Hash("magic block data")

	type fields struct {
		MagicBlockStorage round.RoundStorage
	}
	type args struct {
		round int64
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      *block.MagicBlock
		wantPanic bool
	}{
		{
			name: "PANIC",
			fields: fields{
				MagicBlockStorage: func() round.RoundStorage {
					mbs := round.NewRoundStartingStorage()
					if err := mbs.Put(nil, 0); err != nil {
						t.Fatal(err)
					}
					if err := mbs.Put(nil, 1); err != nil {
						t.Fatal(err)
					}

					return mbs
				}(),
			},
			args:      args{round: 0},
			wantPanic: true,
		},
		{
			name: "OK",
			fields: fields{
				MagicBlockStorage: func() round.RoundStorage {
					mbs := round.NewRoundStartingStorage()
					if err := mbs.Put(mb, 0); err != nil {
						t.Fatal(err)
					}

					return mbs
				}(),
			},
			args: args{round: 0},
			want: mb,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("GetLatestMagicBlock() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			c := &Chain{
				mutexViewChangeMB: sync.RWMutex{},
				MagicBlockStorage: tt.fields.MagicBlockStorage,
				mbMutex:           sync.RWMutex{},
				lfmbMutex:         sync.RWMutex{},
				blocksMutex:       &sync.RWMutex{},
				lfbMutex:          sync.RWMutex{},
				stateMutex:        &sync.RWMutex{},
				stakeMutex:        &sync.Mutex{},
				genTimeoutMutex:   &sync.Mutex{},
				retry_wait_mutex:  &sync.Mutex{},
			}
			if got := c.GetMagicBlock(tt.args.round); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetMagicBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetPrevMagicBlockFromMB(t *testing.T) {
	mb := block.NewMagicBlock()
	mb.StartingRound = 0
	mb.Hash = encryption.Hash("magic block data")
	mb2 := block.NewMagicBlock()
	mb2.StartingRound = 1
	mb2.Hash = encryption.Hash("magic block 2 data")

	type fields struct {
		MagicBlockStorage round.RoundStorage
	}
	type args struct {
		mb *block.MagicBlock
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantPmb *block.MagicBlock
	}{
		{
			name: "OK",
			fields: fields{
				MagicBlockStorage: func() round.RoundStorage {
					mbs := round.NewRoundStartingStorage()
					if err := mbs.Put(mb, 0); err != nil {
						t.Fatal(err)
					}
					if err := mbs.Put(mb2, 1); err != nil {
						t.Fatal(err)
					}

					return mbs
				}(),
			},
			args: args{mb: mb},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				mutexViewChangeMB: sync.RWMutex{},
				MagicBlockStorage: tt.fields.MagicBlockStorage,
				mbMutex:           sync.RWMutex{},
				lfmbMutex:         sync.RWMutex{},
				blocksMutex:       &sync.RWMutex{},
				lfbMutex:          sync.RWMutex{},
				stateMutex:        &sync.RWMutex{},
				stakeMutex:        &sync.Mutex{},
				genTimeoutMutex:   &sync.Mutex{},
				retry_wait_mutex:  &sync.Mutex{},
			}
			if gotPmb := c.GetPrevMagicBlockFromMB(tt.args.mb); !reflect.DeepEqual(gotPmb, tt.wantPmb) {
				t.Errorf("GetPrevMagicBlockFromMB() = %v, want %v", gotPmb, tt.wantPmb)
			}
		})
	}
}

func TestChain_Read(t *testing.T) {
	store := mocks.Store{}
	store.On("Read", context.Context(nil), "", mock.AnythingOfType("*chain.Chain")).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return nil
		},
	)

	chainEntityMetadata.Store = &store

	type args struct {
		ctx context.Context
		key datastore.Key
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				mutexViewChangeMB: sync.RWMutex{},
				mbMutex:           sync.RWMutex{},
				lfmbMutex:         sync.RWMutex{},
				blocksMutex:       &sync.RWMutex{},
				lfbMutex:          sync.RWMutex{},
				stateMutex:        &sync.RWMutex{},
				stakeMutex:        &sync.Mutex{},
				genTimeoutMutex:   &sync.Mutex{},
				retry_wait_mutex:  &sync.Mutex{},
			}
			if err := c.Read(tt.args.ctx, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_Write(t *testing.T) {
	store := mocks.Store{}
	store.On("Write", context.Context(nil), mock.AnythingOfType("*chain.Chain")).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	chainEntityMetadata.Store = &store

	type fields struct {
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				mutexViewChangeMB: sync.RWMutex{},
				mbMutex:           sync.RWMutex{},
				lfmbMutex:         sync.RWMutex{},
				blocksMutex:       &sync.RWMutex{},
				lfbMutex:          sync.RWMutex{},
				stateMutex:        &sync.RWMutex{},
				stakeMutex:        &sync.Mutex{},
				genTimeoutMutex:   &sync.Mutex{},
				retry_wait_mutex:  &sync.Mutex{},
			}
			if err := c.Write(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_Delete(t *testing.T) {
	store := mocks.Store{}
	store.On("Delete", context.Context(nil), mock.AnythingOfType("*chain.Chain")).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	chainEntityMetadata.Store = &store

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				mutexViewChangeMB: sync.RWMutex{},
				mbMutex:           sync.RWMutex{},
				lfmbMutex:         sync.RWMutex{},
				blocksMutex:       &sync.RWMutex{},
				lfbMutex:          sync.RWMutex{},
				stateMutex:        &sync.RWMutex{},
				stakeMutex:        &sync.Mutex{},
				genTimeoutMutex:   &sync.Mutex{},
				retry_wait_mutex:  &sync.Mutex{},
			}
			if err := c.Delete(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSetupStateDB(t *testing.T) {
	tests := []struct {
		name      string
		wantPanic bool
	}{
		{
			name:      "OK",
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("SetupStateDB() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()
			SetupStateDB()
		})
	}
}

func TestChain_SetupConfigInfoDB(t *testing.T) {
	if err := os.MkdirAll("data/rocksdb/config", 0700); err != nil {
		panic(err)
	}

	tests := []struct {
		name      string
		wantPanic bool
	}{
		{
			name:      "OK",
			wantPanic: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("SetupConfigInfoDB() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			c := &Chain{
				mutexViewChangeMB: sync.RWMutex{},
				mbMutex:           sync.RWMutex{},
				lfmbMutex:         sync.RWMutex{},
				blocksMutex:       &sync.RWMutex{},
				lfbMutex:          sync.RWMutex{},
				stateMutex:        &sync.RWMutex{},
				stakeMutex:        &sync.Mutex{},
				genTimeoutMutex:   &sync.Mutex{},
				retry_wait_mutex:  &sync.Mutex{},
			}

			c.SetupConfigInfoDB()
		})
	}

	if err := os.RemoveAll("data"); err != nil {
		panic(err)
	}
}

func TestChain_SetupConfigInfoDB_Panic(t *testing.T) {

	tests := []struct {
		name      string
		wantPanic bool
	}{
		{
			name:      "PANIC",
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("SetupConfigInfoDB() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			c := &Chain{
				mutexViewChangeMB: sync.RWMutex{},
				mbMutex:           sync.RWMutex{},
				lfmbMutex:         sync.RWMutex{},
				blocksMutex:       &sync.RWMutex{},
				lfbMutex:          sync.RWMutex{},
				stateMutex:        &sync.RWMutex{},
				stakeMutex:        &sync.Mutex{},
				genTimeoutMutex:   &sync.Mutex{},
				retry_wait_mutex:  &sync.Mutex{},
			}

			c.SetupConfigInfoDB()
		})
	}
}

// TODO fix tests
//func TestChain_GenerateGenesisBlock(t *testing.T) {
//	chainKey := "chain key"
//	ownerID := "owner id"
//	stateDB := util.NewMemoryNodeDB()
//
//	balance := &state.State{
//		Balance: state.Balance(200000000),
//	}
//	if err := balance.SetTxnHash("0000000000000000000000000000000000000000000000000000000000000000"); err != nil {
//		t.Fatal(err)
//	}
//
//	mpt := util.NewMerklePatriciaTrie(stateDB, util.Sequence(0))
//	if _, err := mpt.Insert(util.Path(ownerID), balance); err != nil {
//		t.Fatal(err)
//	}
//	if err := mpt.SaveChanges(stateDB, false); err != nil {
//		t.Fatal(err)
//	}
//
//	mb := block.NewMagicBlock()
//	mb.Miners = node.NewPool(node.NodeTypeMiner)
//	mb.Sharders = node.NewPool(node.NodeTypeSharder)
//
//	gb := block.NewBlock(chainKey, 0)
//	gb.ClientState = mpt
//	gb.SetStateStatus(block.StateSuccessful)
//	gb.SetBlockState(block.StateNotarized)
//	gb.ClientStateHash = gb.ClientState.GetRoot()
//	gb.MagicBlock = mb
//	gb.Hash = gb.ComputeHash()
//
//	gr := round.NewRound(0)
//	gr.SetRandomSeed(839695260482366273, 0)
//	gr.Block = gb
//	gr.AddNotarizedBlock(gb)
//
//	type fields struct {
//		Config            *Config
//		MagicBlockStorage round.RoundStorage
//		stateDB           util.NodeDB
//	}
//	type args struct {
//		hash              string
//		genesisMagicBlock *block.MagicBlock
//		initStates *state.InitStates
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   round.RoundI
//		want1  *block.Block
//	}{
//		{
//			name: "OK",
//			fields: fields{
//				Config: &Config{
//					OwnerID: ownerID,
//				},
//				stateDB: stateDB,
//				MagicBlockStorage: func() round.RoundStorage {
//					st := round.NewRoundStartingStorage()
//					if err := st.Put(mb, 0); err != nil {
//						t.Fatal(err)
//					}
//
//					return st
//				}(),
//			},
//			args:  args{hash: gb.Hash, genesisMagicBlock: mb},
//			want:  gr,
//			want1: gb,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &Chain{
//				mutexViewChangeMB: sync.RWMutex{},
//				Config:            tt.fields.Config,
//				MagicBlockStorage: tt.fields.MagicBlockStorage,
//				mbMutex:           sync.RWMutex{},
//				lfmbMutex:         sync.RWMutex{},
//				blocksMutex:       &sync.RWMutex{},
//				roundsMutex:       &sync.RWMutex{},
//				lfbMutex:          sync.RWMutex{},
//				stateDB:           tt.fields.stateDB,
//				stateMutex:        &sync.RWMutex{},
//				stakeMutex:        &sync.Mutex{},
//				genTimeoutMutex:   &sync.Mutex{},
//				retry_wait_mutex:  &sync.Mutex{},
//			}
//			got, got1 := c.GenerateGenesisBlock(tt.args.hash, tt.args.genesisMagicBlock, tt.args.initStates)
//			assert.Equal(t, tt.want, got)
//			assert.Equal(t, tt.want1, got1)
//		})
//	}
//}

func TestChain_AddGenesisBlock(t *testing.T) {
	ch := &Chain{
		blocks: make(map[string]*block.Block),
	}
	b := block.NewBlock("", 0)
	b.HashBlock()
	ch.SetLatestFinalizedMagicBlock(b)
	ch.SetLatestFinalizedBlock(b)
	ch.SetLatestDeterministicBlock(b)
	ch.blocks[b.Hash] = b

	type fields struct {
		blocks map[datastore.Key]*block.Block
	}
	type args struct {
		b *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Chain
	}{
		{
			name: "OK",
			fields: fields{
				blocks: make(map[string]*block.Block),
			},
			args: args{b: b},
			want: ch,
		},
		{
			name: "Not_A_Zero_Round_OK",
			args: args{b: block.NewBlock("", 1)},
			want: &Chain{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				blocks: tt.fields.blocks,
			}

			c.AddGenesisBlock(tt.args.b)
			assert.Equal(t, tt.want, c)
		})
	}
}

func TestChain_AddLoadedFinalizedBlocks(t *testing.T) {
	ch := &Chain{
		blocks: make(map[string]*block.Block),
	}
	lfb := block.NewBlock("", 0)
	lfb.HashBlock()
	lfmb := block.NewBlock("", 1)

	ch.SetLatestFinalizedMagicBlock(lfmb)
	ch.SetLatestFinalizedBlock(lfb)
	ch.blocks[lfb.Hash] = lfb

	type fields struct {
		blocks map[datastore.Key]*block.Block
	}
	type args struct {
		lfb  *block.Block
		lfmb *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Chain
	}{
		{
			name: "OK",
			fields: fields{
				blocks: make(map[string]*block.Block),
			},
			args: args{lfb: lfb, lfmb: lfmb},
			want: ch,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				blocks: tt.fields.blocks,
			}

			c.AddLoadedFinalizedBlocks(tt.args.lfb, tt.args.lfmb)
			assert.Equal(t, tt.want, c)
		})
	}
}

func TestChain_AddBlockNoPrevious(t *testing.T) {
	b := block.NewBlock("", 1)
	b.HashBlock()

	type fields struct {
		blocks      map[datastore.Key]*block.Block
		blocksMutex *sync.RWMutex
	}
	type args struct {
		b *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *block.Block
	}{
		{
			name: "OK",
			fields: fields{
				blocks: make(map[string]*block.Block),
			},
			args: args{b: b},
			want: b,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				blocks:      tt.fields.blocks,
				blocksMutex: &sync.RWMutex{},
			}
			if got := c.AddBlockNoPrevious(tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AddBlockNoPrevious() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_AddBlock(t *testing.T) {
	b := block.NewBlock("", 1)
	b.HashBlock()

	type fields struct {
		blocks map[datastore.Key]*block.Block
	}
	type args struct {
		b *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *block.Block
	}{
		{
			name: "OK",
			fields: fields{
				blocks: make(map[string]*block.Block),
			},
			args: args{b: b},
			want: b,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				blocks:      tt.fields.blocks,
				blocksMutex: &sync.RWMutex{},
			}
			if got := c.AddBlock(tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AddBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_AddNotarizedBlockToRound(t *testing.T) {
	rrs := int64(5)

	b := block.NewBlock("", 1)
	b.SetRoundRandomSeed(rrs)
	b.HashBlock()
	r := round.NewRound(1)

	wantB := block.NewBlock("", 1)
	wantB.SetRoundRandomSeed(rrs)
	wantB.Hash = b.Hash
	wantB.ChainWeight = 0
	wantR := round.NewRound(1)
	wantR.SetRandomSeedForNotarizedBlock(b.GetRoundRandomSeed(), 0)

	type fields struct {
		MagicBlockStorage round.RoundStorage
		blocks            map[datastore.Key]*block.Block
	}
	type args struct {
		r round.RoundI
		b *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *block.Block
		want1  round.RoundI
	}{
		{
			name: "OK",
			fields: fields{
				MagicBlockStorage: func() round.RoundStorage {
					rs := round.NewRoundStartingStorage()
					mb := block.NewMagicBlock()
					mb.Miners = node.NewPool(node.NodeTypeMiner)
					if err := rs.Put(mb, 1); err != nil {
						t.Fatal(err)
					}

					return rs
				}(),
				blocks: make(map[string]*block.Block),
			},
			args:  args{b: b, r: r},
			want:  wantB,
			want1: wantR,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				MagicBlockStorage: tt.fields.MagicBlockStorage,
				blocks:            tt.fields.blocks,
				blocksMutex:       &sync.RWMutex{},
				roundsMutex:       &sync.RWMutex{},
			}
			got, got1 := c.AddNotarizedBlockToRound(tt.args.r, tt.args.b)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want1, got1)
		})
	}
}

func TestChain_AddRoundBlock(t *testing.T) {
	prevB := block.NewBlock("", 1)
	prevB.HashBlock()
	b := block.NewBlock("", 2)
	b.ComputeHash()

	r := round.NewRound(1)

	type fields struct {
		blocks            map[datastore.Key]*block.Block
		MagicBlockStorage round.RoundStorage
	}
	type args struct {
		r round.RoundI
		b *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *block.Block
	}{
		{
			name: "OK",
			fields: fields{
				blocks: map[string]*block.Block{
					prevB.Hash: prevB,
					b.Hash:     b,
				},
				MagicBlockStorage: func() round.RoundStorage {
					rs := round.NewRoundStartingStorage()
					mb := block.NewMagicBlock()
					mb.Miners = node.NewPool(node.NodeTypeMiner)
					if err := rs.Put(mb, 2); err != nil {
						t.Fatal(err)
					}

					return rs
				}(),
			},
			args: args{b: b, r: r},
			want: func() *block.Block {
				b := block.NewBlock("", 2)

				return b
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				blocks:            tt.fields.blocks,
				blocksMutex:       &sync.RWMutex{},
				MagicBlockStorage: tt.fields.MagicBlockStorage,
			}
			got := c.AddRoundBlock(tt.args.r, tt.args.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestChain_GetBlock(t *testing.T) {
	b := block.NewBlock("", 1)
	b.HashBlock()

	type fields struct {
		blocks map[datastore.Key]*block.Block
	}
	type args struct {
		ctx  context.Context
		hash string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *block.Block
		wantErr bool
	}{
		{
			name: "OK",
			fields: fields{
				blocks: map[string]*block.Block{
					b.Hash: b,
				},
			},
			args: args{hash: b.Hash},
			want: b,
		},
		{
			name: "Unknown_Block_ERR",
			fields: fields{
				blocks: map[string]*block.Block{
					b.Hash: b,
				},
			},
			want:    block.NewBlock("", 0),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				blocks:      tt.fields.blocks,
				blocksMutex: &sync.RWMutex{},
			}
			got, err := c.GetBlock(tt.args.ctx, tt.args.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBlock() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_DeleteBlock(t *testing.T) {
	b := block.NewBlock("", 1)
	b.HashBlock()

	type fields struct {
		blocks map[datastore.Key]*block.Block
	}
	type args struct {
		ctx context.Context
		b   *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Chain
	}{
		{
			name: "OK",
			fields: fields{
				blocks: map[string]*block.Block{
					b.Hash: b,
				},
			},
			args: args{b: b},
			want: &Chain{
				blocks:      make(map[string]*block.Block),
				blocksMutex: &sync.RWMutex{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				blocks:      tt.fields.blocks,
				blocksMutex: &sync.RWMutex{},
			}

			c.DeleteBlock(tt.args.ctx, tt.args.b)
			assert.Equal(t, tt.want, c)
		})
	}
}

func TestChain_GetRoundBlocks(t *testing.T) {
	r := int64(1)
	b := block.NewBlock("", r)
	b.HashBlock()
	b1 := block.NewBlock("", r)
	b1.MinerID = "miner id"
	b1.HashBlock()

	type fields struct {
		blocks map[datastore.Key]*block.Block
	}
	type args struct {
		round int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []*block.Block
	}{
		{
			name: "OK",
			fields: fields{
				blocks: map[datastore.Key]*block.Block{
					b.Hash:  b,
					b1.Hash: b1,
				},
			},
			args: args{round: r},
			want: []*block.Block{
				b,
				b1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				blocks:      tt.fields.blocks,
				blocksMutex: &sync.RWMutex{},
			}

			got := c.GetRoundBlocks(tt.args.round)
			sort.Slice(got,
				func(i int, j int) bool {
					return len(got[i].MinerID) > len(got[j].MinerID)
				},
			)
			sort.Slice(tt.want,
				func(i int, j int) bool {
					return len(tt.want[i].MinerID) > len(tt.want[j].MinerID)
				},
			)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestChain_DeleteBlocksBelowRound(t *testing.T) {
	blocks := make(map[string]*block.Block)
	for i := int64(0); i < 5; i++ {
		b := block.NewBlock("", i)
		b.CreationDate = common.Timestamp(0)
		b.HashBlock()
		blocks[b.Hash] = b
	}

	ldb := block.NewBlock("", 6)
	lfb := block.NewBlock("", 6)

	type fields struct {
		blocks                   map[datastore.Key]*block.Block
		LatestDeterministicBlock *block.Block
		LatestFinalizedBlock     *block.Block
	}
	type args struct {
		round int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Chain
	}{
		{
			name: "OK",
			fields: fields{
				blocks:                   blocks,
				LatestDeterministicBlock: ldb,
				LatestFinalizedBlock:     lfb,
			},
			args: args{round: 5},
			want: &Chain{
				blocks:                   make(map[string]*block.Block),
				LatestDeterministicBlock: ldb,
				blocksMutex:              &sync.RWMutex{},
				roundsMutex:              &sync.RWMutex{},
				LatestFinalizedBlock:     lfb,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				blocks:                   tt.fields.blocks,
				blocksMutex:              &sync.RWMutex{},
				roundsMutex:              &sync.RWMutex{},
				LatestDeterministicBlock: tt.fields.LatestDeterministicBlock,
				LatestFinalizedBlock:     tt.fields.LatestFinalizedBlock,
			}

			c.DeleteBlocksBelowRound(tt.args.round)
			assert.Equal(t, tt.want, c)
		})
	}
}

func TestChain_DeleteBlocks(t *testing.T) {
	blocks := make(map[string]*block.Block)
	delBlocks := make([]*block.Block, 0)
	for i := int64(0); i < 5; i++ {
		b := block.NewBlock("", i)
		b.CreationDate = common.Timestamp(0)
		b.HashBlock()
		blocks[b.Hash] = b

		delBlocks = append(delBlocks, b)
	}

	type fields struct {
		blocks map[datastore.Key]*block.Block
	}
	type args struct {
		blocks []*block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Chain
	}{
		{
			name: "OK",
			fields: fields{
				blocks: blocks,
			},
			args: args{blocks: delBlocks},
			want: &Chain{
				blocks:      make(map[string]*block.Block),
				blocksMutex: &sync.RWMutex{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				blocks:      tt.fields.blocks,
				blocksMutex: &sync.RWMutex{},
			}

			c.DeleteBlocks(tt.args.blocks)
			assert.Equal(t, tt.want, c)
		})
	}
}

func TestChain_PruneChain(t *testing.T) {
	blocks := make(map[string]*block.Block)
	for i := int64(0); i < 5; i++ {
		b := block.NewBlock("", i)
		b.CreationDate = common.Timestamp(0)
		b.HashBlock()
		blocks[b.Hash] = b
	}

	ldb := block.NewBlock("", 6)
	lfb := block.NewBlock("", 6)

	type fields struct {
		blocks                   map[datastore.Key]*block.Block
		LatestDeterministicBlock *block.Block
		LatestFinalizedBlock     *block.Block
	}
	type args struct {
		in0 context.Context
		b   *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Chain
	}{
		{
			name: "OK",
			fields: fields{
				blocks:                   blocks,
				LatestDeterministicBlock: ldb,
				LatestFinalizedBlock:     lfb,
			},
			args: args{b: block.NewBlock("", 60)},
			want: &Chain{
				blocks:                   make(map[string]*block.Block),
				LatestDeterministicBlock: ldb,
				blocksMutex:              &sync.RWMutex{},
				roundsMutex:              &sync.RWMutex{},
				LatestFinalizedBlock:     lfb,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				blocks:                   tt.fields.blocks,
				blocksMutex:              &sync.RWMutex{},
				roundsMutex:              &sync.RWMutex{},
				LatestDeterministicBlock: tt.fields.LatestDeterministicBlock,
				LatestFinalizedBlock:     tt.fields.LatestFinalizedBlock,
			}

			c.PruneChain(tt.args.in0, tt.args.b)
			assert.Equal(t, tt.want, c)
		})
	}
}

func TestChain_IsBlockSharder(t *testing.T) {
	mb := block.NewMagicBlock()
	mb.Sharders = node.NewPool(node.NodeTypeSharder)

	n, err := makeTestNode(node.NodeTypeSharder, "id")
	if err != nil {
		t.Fatal(err)
	}
	mb.Sharders.NodesMap[n.ID] = n

	mb.Sharders.ComputeProperties()

	type fields struct {
		Config            *Config
		MagicBlockStorage round.RoundStorage
		nodePoolScorer    node.PoolScorer
	}
	type args struct {
		b       *block.Block
		sharder *node.Node
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Zero_Replicators_TRUE",
			fields: fields{
				Config: &Config{
					NumReplicators: 0,
				},
			},
			want: true,
		},
		{
			name: "TRUE",
			fields: fields{
				Config: &Config{
					NumReplicators: 1,
				},
				MagicBlockStorage: func() round.RoundStorage {
					rs := round.NewRoundStartingStorage()
					if err := rs.Put(mb, 1); err != nil {
						t.Fatal(err)
					}

					return rs
				}(),
				nodePoolScorer: node.NewHashPoolScorer(&encryption.XORHashScorer{}),
			},
			args: args{
				b:       block.NewBlock("", 1),
				sharder: n,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				Config:            tt.fields.Config,
				MagicBlockStorage: tt.fields.MagicBlockStorage,
				mbMutex:           sync.RWMutex{},
				nodePoolScorer:    tt.fields.nodePoolScorer,
			}
			if got := c.IsBlockSharder(tt.args.b, tt.args.sharder); got != tt.want {
				t.Errorf("IsBlockSharder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_IsBlockSharderFromHash(t *testing.T) {
	mb := block.NewMagicBlock()
	mb.Sharders = node.NewPool(node.NodeTypeSharder)

	n, err := makeTestNode(node.NodeTypeSharder, "id")
	if err != nil {
		t.Fatal(err)
	}
	mb.Sharders.NodesMap[n.ID] = n

	mb.Sharders.ComputeProperties()

	type fields struct {
		Config            *Config
		MagicBlockStorage round.RoundStorage
		nodePoolScorer    node.PoolScorer
	}
	type args struct {
		nRound  int64
		bHash   string
		sharder *node.Node
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Zero_Replicators_TRUE",
			fields: fields{
				Config: &Config{
					NumReplicators: 0,
				},
			},
			want: true,
		},
		{
			name: "TRUE",
			fields: fields{
				Config: &Config{
					NumReplicators: 1,
				},
				MagicBlockStorage: func() round.RoundStorage {
					rs := round.NewRoundStartingStorage()
					if err := rs.Put(mb, 1); err != nil {
						t.Fatal(err)
					}

					return rs
				}(),
				nodePoolScorer: node.NewHashPoolScorer(&encryption.XORHashScorer{}),
			},
			args: args{
				nRound:  1,
				sharder: n,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				Config:            tt.fields.Config,
				MagicBlockStorage: tt.fields.MagicBlockStorage,
				mbMutex:           sync.RWMutex{},
				nodePoolScorer:    tt.fields.nodePoolScorer,
			}
			if got := c.IsBlockSharderFromHash(tt.args.nRound, tt.args.bHash, tt.args.sharder); got != tt.want {
				t.Errorf("IsBlockSharder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetNotarizationThresholdCount(t *testing.T) {
	type fields struct {
		Config *Config
	}
	type args struct {
		minersNumber int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name: "OK",
			fields: fields{
				Config: &Config{
					ThresholdByCount: 100,
				},
			},
			args: args{
				minersNumber: 5,
			},
			want: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				Config: tt.fields.Config,
			}
			if got := c.GetNotarizationThresholdCount(tt.args.minersNumber); got != tt.want {
				t.Errorf("GetNotarizationThresholdCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_AddRound(t *testing.T) {
	r := round.NewRound(1)
	r1 := round.NewRound(2)

	type fields struct {
		rounds map[int64]round.RoundI
	}
	type args struct {
		r round.RoundI
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   round.RoundI
	}{
		{
			name: "OK",
			fields: fields{
				rounds: map[int64]round.RoundI{
					r.GetRoundNumber(): r,
				},
			},
			args: args{r: r},
			want: r,
		},
		{
			name: "Unknown_Round_OK",
			fields: fields{
				rounds: map[int64]round.RoundI{
					r.GetRoundNumber(): r,
				},
			},
			args: args{r: r1},
			want: r1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				rounds:      tt.fields.rounds,
				roundsMutex: &sync.RWMutex{},
			}
			if got := c.AddRound(tt.args.r); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AddRound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetRound(t *testing.T) {
	r := round.NewRound(1)

	type fields struct {
		rounds map[int64]round.RoundI
	}
	type args struct {
		roundNumber int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   round.RoundI
	}{
		{
			name: "OK",
			fields: fields{
				rounds: map[int64]round.RoundI{
					r.GetRoundNumber(): r,
				},
			},
			args: args{roundNumber: r.GetRoundNumber()},
			want: r,
		},
		{
			name: "Unknown_Round_OK",
			fields: fields{
				rounds: map[int64]round.RoundI{
					r.GetRoundNumber(): r,
				},
			},
			args: args{roundNumber: 2},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				rounds:      tt.fields.rounds,
				roundsMutex: &sync.RWMutex{},
			}
			if got := c.GetRound(tt.args.roundNumber); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_DeleteRound(t *testing.T) {
	r := round.NewRound(1)

	type fields struct {
		rounds map[int64]round.RoundI
	}
	type args struct {
		ctx context.Context
		r   round.RoundI
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Chain
	}{
		{
			name: "OK",
			fields: fields{
				rounds: map[int64]round.RoundI{
					r.GetRoundNumber(): r,
				},
			},
			args: args{r: r},
			want: &Chain{
				rounds:      make(map[int64]round.RoundI),
				roundsMutex: &sync.RWMutex{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				rounds:      tt.fields.rounds,
				roundsMutex: &sync.RWMutex{},
			}

			c.DeleteRound(tt.args.ctx, tt.args.r)
			assert.Equal(t, tt.want, c)
		})
	}
}

func TestChain_DeleteRoundsBelow(t *testing.T) {
	r := round.NewRound(1)
	r1 := round.NewRound(2)

	type fields struct {
		rounds map[int64]round.RoundI
	}
	type args struct {
		ctx         context.Context
		roundNumber int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Chain
	}{
		{
			name: "OK",
			fields: fields{
				rounds: map[int64]round.RoundI{
					r.GetRoundNumber():  r,
					r1.GetRoundNumber(): r1,
				},
			},
			args: args{roundNumber: 13},
			want: &Chain{
				rounds:      make(map[int64]round.RoundI),
				roundsMutex: &sync.RWMutex{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				rounds:      tt.fields.rounds,
				roundsMutex: &sync.RWMutex{},
			}

			c.DeleteRoundsBelow(tt.args.ctx, tt.args.roundNumber)
			assert.Equal(t, tt.want, c)
		})
	}
}

func TestChain_ValidateMagicBlock(t *testing.T) {
	mb := block.NewBlock("", 1)
	mb.HashBlock()

	b := block.NewBlock("", 1)
	b.LatestFinalizedMagicBlockHash = mb.Hash

	type fields struct {
		LatestFinalizedMagicBlock *block.Block
	}
	type args struct {
		ctx context.Context
		mr  *round.Round
		b   *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "TRUE",
			fields: fields{
				LatestFinalizedMagicBlock: mb,
			},
			args: args{b: b, mr: round.NewRound(b.Round)},
			want: true,
		},
		{
			name: "FAlSE",
			fields: fields{
				LatestFinalizedMagicBlock: mb,
			},
			args: args{b: block.NewBlock("", 1), mr: round.NewRound(b.Round)},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				LatestFinalizedMagicBlock: tt.fields.LatestFinalizedMagicBlock,
			}
			if got := c.ValidateMagicBlock(tt.args.ctx, tt.args.mr, tt.args.b); got != tt.want {
				t.Errorf("ValidateMagicBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func makeTestNode(typ int8, id string) (*node.Node, error) {
	nc := map[interface{}]interface{}{
		"type":        typ,
		"public_ip":   "public ip",
		"n2n_ip":      "n2n ip",
		"port":        8080,
		"id":          id,
		"public_key":  "public key",
		"description": "description",
	}

	return node.NewNode(nc)
}

func TestChain_GetGenerators(t *testing.T) {
	mb := block.NewMagicBlock()
	mb.Miners = node.NewPool(node.NodeTypeMiner)

	n, err := makeTestNode(node.NodeTypeMiner, "id")
	if err != nil {
		t.Fatal(err)
	}
	mb.Miners.NodesMap[n.ID] = n

	mb.Miners.ComputeProperties()

	type fields struct {
		Config            *Config
		MagicBlockStorage round.RoundStorage
	}
	type args struct {
		r round.RoundI
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []*node.Node
	}{
		{
			name: "OK",
			fields: fields{
				Config: &Config{
					NumGenerators: 0,
				},
				MagicBlockStorage: func() round.RoundStorage {
					rs := round.NewRoundStartingStorage()
					if err := rs.Put(mb, 1); err != nil {
						t.Fatal(err)
					}

					return rs
				}(),
			},
			args: args{
				r: func() round.RoundI {
					r := round.NewRound(1)
					return r
				}(),
			},
			want: make([]*node.Node, 0, 1),
		},
		{
			name: "OK",
			fields: fields{
				Config: &Config{
					NumGenerators: 2,
				},
				MagicBlockStorage: func() round.RoundStorage {
					rs := round.NewRoundStartingStorage()
					if err := rs.Put(mb, 1); err != nil {
						t.Fatal(err)
					}

					return rs
				}(),
			},
			args: args{
				r: func() round.RoundI {
					r := round.NewRound(1)
					return r
				}(),
			},
			want: []*node.Node{
				n,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				Config:            tt.fields.Config,
				MagicBlockStorage: tt.fields.MagicBlockStorage,
			}
			if got := c.GetGenerators(tt.args.r); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetGenerators() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_CanShardBlockWithReplicators(t *testing.T) {
	mb := block.NewMagicBlock()
	mb.Sharders = node.NewPool(node.NodeTypeSharder)

	n, err := makeTestNode(node.NodeTypeSharder, "id")
	if err != nil {
		t.Fatal(err)
	}
	mb.Sharders.NodesMap[n.ID] = n

	mb.Sharders.ComputeProperties()

	rs := round.NewRoundStartingStorage()
	if err := rs.Put(mb, 1); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Config            *Config
		MagicBlockStorage round.RoundStorage
		nodePoolScorer    node.PoolScorer
	}
	type args struct {
		nRound  int64
		hash    string
		sharder *node.Node
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
		want1  []*node.Node
	}{
		{
			name: "Zero_Replicators_TRUE",
			fields: fields{
				Config: &Config{
					NumReplicators: 0,
				},
				MagicBlockStorage: rs,
			},
			want: true,
			want1: []*node.Node{
				n,
			},
		},
		{
			name: "TRUE",
			fields: fields{
				Config: &Config{
					NumReplicators: 1,
				},
				MagicBlockStorage: rs,
				nodePoolScorer:    node.NewHashPoolScorer(&encryption.XORHashScorer{}),
			},
			args: args{
				nRound:  1,
				sharder: n,
			},
			want: true,
			want1: []*node.Node{
				n,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				Config:            tt.fields.Config,
				MagicBlockStorage: tt.fields.MagicBlockStorage,
				mbMutex:           sync.RWMutex{},
				nodePoolScorer:    tt.fields.nodePoolScorer,
			}
			got, got1 := c.CanShardBlockWithReplicators(tt.args.nRound, tt.args.hash, tt.args.sharder)
			if got != tt.want {
				t.Errorf("CanShardBlockWithReplicators() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("CanShardBlockWithReplicators() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestChain_GetBlockSharders(t *testing.T) {
	mb := block.NewMagicBlock()
	mb.Sharders = node.NewPool(node.NodeTypeSharder)

	n, err := makeTestNode(node.NodeTypeSharder, "id")
	if err != nil {
		t.Fatal(err)
	}
	mb.Sharders.NodesMap[n.ID] = n

	mb.Sharders.ComputeProperties()

	rs := round.NewRoundStartingStorage()
	if err := rs.Put(mb, 1); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Config            *Config
		MagicBlockStorage round.RoundStorage
		nodePoolScorer    node.PoolScorer
	}
	type args struct {
		b *block.Block
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantSharders []string
	}{
		{
			name: "OK",
			fields: fields{
				Config: &Config{
					NumReplicators: 1,
				},
				MagicBlockStorage: rs,
				nodePoolScorer:    node.NewHashPoolScorer(&encryption.XORHashScorer{}),
			},
			args: args{b: block.NewBlock("", 1)},
			wantSharders: []string{
				n.GetKey(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				Config:            tt.fields.Config,
				MagicBlockStorage: tt.fields.MagicBlockStorage,
				mbMutex:           sync.RWMutex{},
				nodePoolScorer:    tt.fields.nodePoolScorer,
			}
			if gotSharders := c.GetBlockSharders(tt.args.b); !reflect.DeepEqual(gotSharders, tt.wantSharders) {
				t.Errorf("GetBlockSharders() = %v, want %v", gotSharders, tt.wantSharders)
			}
		})
	}
}

func TestChain_ValidGenerator(t *testing.T) {
	mb := block.NewMagicBlock()
	mb.Miners = node.NewPool(node.NodeTypeMiner)

	n, err := makeTestNode(node.NodeTypeMiner, "id")
	if err != nil {
		t.Fatal(err)
	}
	mb.Miners.NodesMap[n.ID] = n

	mb.Miners.ComputeProperties()

	rs := round.NewRoundStartingStorage()
	if err := rs.Put(mb, 1); err != nil {
		t.Fatal(err)
	}

	b := block.NewBlock("", 1)
	b.MinerID = n.GetKey()

	type fields struct {
		Config            *Config
		MagicBlockStorage round.RoundStorage
		nodePoolScorer    node.PoolScorer
	}
	type args struct {
		r round.RoundI
		b *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Nil_Miner_FALSE",
			fields: fields{
				Config: &Config{
					NumReplicators: 1,
				},
				MagicBlockStorage: rs,
				nodePoolScorer:    node.NewHashPoolScorer(&encryption.XORHashScorer{}),
			},
			args: args{b: block.NewBlock("", 1), r: round.NewRound(1)},
			want: false,
		},
		{
			name: "FALSE",
			fields: fields{
				Config: &Config{
					NumReplicators: 1,
					NumGenerators:  2,
				},
				MagicBlockStorage: rs,
				nodePoolScorer:    node.NewHashPoolScorer(&encryption.XORHashScorer{}),
			},
			args: args{b: b, r: round.NewRound(1)},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				Config:            tt.fields.Config,
				MagicBlockStorage: tt.fields.MagicBlockStorage,
				mbMutex:           sync.RWMutex{},
				nodePoolScorer:    tt.fields.nodePoolScorer,
			}
			if got := c.ValidGenerator(tt.args.r, tt.args.b); got != tt.want {
				t.Errorf("ValidGenerator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_AreAllNodesActive(t *testing.T) {
	mb := block.NewMagicBlock()
	mb.Miners = node.NewPool(node.NodeTypeMiner)

	n, err := makeTestNode(node.NodeTypeMiner, "id")
	if err != nil {
		t.Fatal(err)
	}
	mb.Miners.NodesMap[n.ID] = n

	mb.Miners.ComputeProperties()

	type fields struct {
		MagicBlockStorage round.RoundStorage
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "TRUE",
			fields: fields{
				MagicBlockStorage: func() round.RoundStorage {
					rs := round.NewRoundStartingStorage()
					if err := rs.Put(mb, 1); err != nil {
						t.Fatal(err)
					}

					return rs
				}(),
			},
			want: true,
		},
		{
			name: "FALSE",
			fields: fields{
				MagicBlockStorage: func() round.RoundStorage {
					rs := round.NewRoundStartingStorage()
					mb := block.NewMagicBlock()
					n := *n
					n.Status = node.NodeStatusInactive
					mb.Miners = node.NewPool(node.NodeTypeMiner)
					mb.Miners.NodesMap[n.ID] = &n
					mb.Miners.ComputeProperties()

					if err := rs.Put(mb, 1); err != nil {
						t.Fatal(err)
					}

					return rs
				}(),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				MagicBlockStorage: tt.fields.MagicBlockStorage,
				roundsMutex:       &sync.RWMutex{},
			}
			if got := c.AreAllNodesActive(); got != tt.want {
				t.Errorf("AreAllNodesActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_ChainHasTransaction(t *testing.T) {
	txn := transaction.Transaction{}
	txn.Hash = encryption.Hash("txn hash")
	txn.CreationDate = common.Now()

	type args struct {
		ctx context.Context
		b   *block.Block
		txn *transaction.Transaction
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Prev_Block_Zero_Round_FALSE",
			args: args{
				b: block.NewBlock("", 0),
			},
			want: false,
		},
		{
			name: "No_Txn_In_Block_TRUE",
			args: args{
				b: func() *block.Block {
					b := block.NewBlock("", 1)
					b.TxnsMap = map[string]bool{
						txn.Hash: true,
					}

					return b
				}(),
				txn: &txn,
			},
			want: true,
		},
		{
			name: "FALSE",
			args: args{
				b: func() *block.Block {
					b := block.NewBlock("", 1)
					b.CreationDate = 0

					return b
				}(),
				txn: &txn,
			},
			want: false,
		},
		{
			name: "ERR",
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.TODO())
					cancel()

					return ctx
				}(),
				b:   block.NewBlock("", 1),
				txn: &txn,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				blocksMutex:  &sync.RWMutex{},
				blockFetcher: NewBlockFetcher(),
				roundsMutex:  &sync.RWMutex{},
			}
			got, err := c.ChainHasTransaction(tt.args.ctx, tt.args.b, tt.args.txn)
			if (err != nil) != tt.wantErr {
				t.Errorf("ChainHasTransaction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ChainHasTransaction() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_SetRandomSeed(t *testing.T) {
	randomSeed := int64(5)
	r := round.NewRound(1)
	r.SetRandomSeed(randomSeed, 0)

	type fields struct {
		MagicBlockStorage round.RoundStorage
	}
	type args struct {
		r          round.RoundI
		randomSeed int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "FALSE",
			args: args{r: r, randomSeed: randomSeed},
			want: false,
		},
		{
			name: "TRUE",
			fields: fields{
				MagicBlockStorage: func() round.RoundStorage {
					mb := block.NewMagicBlock()
					mb.Miners = node.NewPool(node.NodeTypeMiner)
					rs := round.NewRoundStartingStorage()
					if err := rs.Put(mb, r.GetRoundNumber()); err != nil {
						t.Fatal(err)
					}

					return rs
				}(),
			},
			args: args{r: r, randomSeed: 0},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				roundsMutex:       &sync.RWMutex{},
				MagicBlockStorage: tt.fields.MagicBlockStorage,
			}
			if got := c.SetRandomSeed(tt.args.r, tt.args.randomSeed); got != tt.want {
				t.Errorf("SetRandomSeed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_CanStartNetwork(t *testing.T) {
	type fields struct {
		MagicBlockStorage round.RoundStorage
		Config            *Config
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "TRUE",
			fields: fields{
				MagicBlockStorage: func() round.RoundStorage {
					mb := block.NewMagicBlock()
					mb.Miners = node.NewPool(node.NodeTypeMiner)
					mb.Sharders = node.NewPool(node.NodeTypeSharder)

					rs := round.NewRoundStartingStorage()
					if err := rs.Put(mb, 1); err != nil {
						t.Fatal(err)
					}

					return rs
				}(),
				Config: &Config{
					ThresholdByCount: 100,
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				MagicBlockStorage: tt.fields.MagicBlockStorage,
				roundsMutex:       &sync.RWMutex{},
				Config:            tt.fields.Config,
			}
			if got := c.CanStartNetwork(); got != tt.want {
				t.Errorf("CanStartNetwork() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetUnrelatedBlocks(t *testing.T) {
	b := block.NewBlock("", 2)
	b.HashBlock()

	type fields struct {
		blocks map[string]*block.Block
	}
	type args struct {
		maxBlocks int
		b         *block.Block
	}
	tests := []struct {
		name string
		fields
		args args
		want []*block.Block
	}{
		{
			name: "OK",
			fields: fields{
				blocks: map[string]*block.Block{
					b.Hash: b,
				},
			},
			args: args{
				b: func() *block.Block {
					b1 := block.NewBlock("", 3)
					b1.Hash = b.Hash
					prevB := block.NewBlock("", 1)
					prevB.HashBlock()
					b1.PrevBlock = prevB

					return b1
				}(),
			},
			want: []*block.Block{
				b,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				blocks:      tt.fields.blocks,
				blocksMutex: &sync.RWMutex{},
			}
			if got := c.GetUnrelatedBlocks(tt.args.maxBlocks, tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetUnrelatedBlocks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_CanShardBlocksSharders(t *testing.T) {
	type fields struct {
		Config *Config
	}
	type args struct {
		sharders *node.Pool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "TRUE",
			fields: fields{
				Config: &Config{
					MinActiveSharders: 1,
				},
			},
			args: args{sharders: node.NewPool(node.NodeTypeSharder)},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				Config: tt.fields.Config,
			}
			if got := c.CanShardBlocksSharders(tt.args.sharders); got != tt.want {
				t.Errorf("CanShardBlocksSharders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_CanShardBlocks(t *testing.T) {
	rNum := int64(5)

	type fields struct {
		Config            *Config
		MagicBlockStorage round.RoundStorage
	}
	type args struct {
		nRound int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "TRUE",
			fields: fields{
				Config: &Config{
					MinActiveSharders: 1,
				},
				MagicBlockStorage: func() round.RoundStorage {
					mb := block.NewMagicBlock()
					mb.Sharders = node.NewPool(node.NodeTypeSharder)

					rs := round.NewRoundStartingStorage()
					if err := rs.Put(mb, rNum); err != nil {
						t.Fatal(err)
					}

					return rs
				}(),
			},
			args: args{nRound: rNum},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				Config:            tt.fields.Config,
				MagicBlockStorage: tt.fields.MagicBlockStorage,
			}
			if got := c.CanShardBlocks(tt.args.nRound); got != tt.want {
				t.Errorf("CanShardBlocks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_CanReplicateBlock(t *testing.T) {
	rNum := int64(5)

	type fields struct {
		Config            *Config
		MagicBlockStorage round.RoundStorage
		nodePoolScorer    node.PoolScorer
	}
	type args struct {
		b *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "TRUE",
			fields: fields{
				Config: &Config{
					MinActiveSharders: 1,
				},
				MagicBlockStorage: func() round.RoundStorage {
					mb := block.NewMagicBlock()
					mb.Sharders = node.NewPool(node.NodeTypeSharder)

					rs := round.NewRoundStartingStorage()
					if err := rs.Put(mb, rNum); err != nil {
						t.Fatal(err)
					}

					return rs
				}(),
			},
			args: args{b: block.NewBlock("", rNum)},
			want: true,
		},
		{
			name: "Empty_Scores_TRUE",
			fields: fields{
				Config: &Config{
					MinActiveSharders:    1,
					NumReplicators:       1,
					MinActiveReplicators: 1,
				},
				MagicBlockStorage: func() round.RoundStorage {
					mb := block.NewMagicBlock()
					mb.Sharders = node.NewPool(node.NodeTypeSharder)

					rs := round.NewRoundStartingStorage()
					if err := rs.Put(mb, rNum); err != nil {
						t.Fatal(err)
					}

					return rs
				}(),
				nodePoolScorer: node.NewHashPoolScorer(&encryption.XORHashScorer{}),
			},
			args: args{b: block.NewBlock("", rNum)},
			want: true,
		},
		{
			name: "TRUE2",
			fields: fields{
				Config: &Config{
					MinActiveSharders:    1,
					NumReplicators:       1,
					MinActiveReplicators: 1,
				},
				MagicBlockStorage: func() round.RoundStorage {
					mb := block.NewMagicBlock()
					mb.Sharders = node.NewPool(node.NodeTypeSharder)

					n, err := makeTestNode(node.NodeTypeSharder, "id")
					if err != nil {
						t.Fatal(err)
					}
					mb.Sharders.NodesMap[n.ID] = n
					mb.Sharders.ComputeProperties()

					rs := round.NewRoundStartingStorage()
					if err := rs.Put(mb, rNum); err != nil {
						t.Fatal(err)
					}

					return rs
				}(),
				nodePoolScorer: node.NewHashPoolScorer(&encryption.XORHashScorer{}),
			},
			args: args{b: block.NewBlock("", rNum)},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				Config:            tt.fields.Config,
				MagicBlockStorage: tt.fields.MagicBlockStorage,
				nodePoolScorer:    tt.fields.nodePoolScorer,
			}
			if got := c.CanReplicateBlock(tt.args.b); got != tt.want {
				t.Errorf("CanReplicateBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_HasClientStateStored(t *testing.T) {
	key := encryption.Hash("data")
	value := util.SecureSerializableValue{Buffer: []byte("value")}
	n := util.NewFullNode(&value)

	stateDB := util.NewMemoryNodeDB()
	if err := stateDB.PutNode(util.Key(key), n); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		stateDB util.NodeDB
	}
	type args struct {
		clientStateHash util.Key
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "TRUE",
			fields: fields{
				stateDB: stateDB,
			},
			args: args{clientStateHash: util.Key(key)},
			want: true,
		},
		{
			name: "FALSE",
			fields: fields{
				stateDB: stateDB,
			},
			args: args{clientStateHash: util.Key("unknown key")},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				stateDB: tt.fields.stateDB,
			}
			if got := c.HasClientStateStored(tt.args.clientStateHash); got != tt.want {
				t.Errorf("HasClientStateStored() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_InitBlockState(t *testing.T) {
	key := encryption.Hash("data")
	value := util.SecureSerializableValue{Buffer: []byte("value")}
	n := util.NewFullNode(&value)

	stateDB := util.NewMemoryNodeDB()
	if err := stateDB.PutNode(util.Key(key), n); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		stateDB util.NodeDB
	}
	type args struct {
		b *block.Block
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "OK",
			fields: fields{
				stateDB: stateDB,
			},
			args: args{
				b: func() *block.Block {
					b := block.NewBlock("", 1)
					b.ClientStateHash = util.Key(key)

					return b
				}(),
			},
			wantErr: false,
		},
		{
			name: "ERR",
			fields: fields{
				stateDB: stateDB,
			},
			args: args{
				b: func() *block.Block {
					b := block.NewBlock("", 1)
					b.ClientStateHash = util.Key("unknown hash")

					return b
				}(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				stateDB: tt.fields.stateDB,
			}
			if err := c.InitBlockState(tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("InitBlockState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_IsActiveInChain(t *testing.T) {
	node.Self.ID = "ID"

	type fields struct {
		MagicBlockStorage    round.RoundStorage
		LatestFinalizedBlock *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "TRUE",
			fields: fields{
				MagicBlockStorage: func() round.RoundStorage {
					n, err := makeTestNode(node.NodeTypeMiner, "id")
					if err != nil {
						t.Fatal(err)
					}

					mb := block.NewMagicBlock()
					mb.Miners = node.NewPool(node.NodeTypeMiner)
					mb.Miners.NodesMap = map[string]*node.Node{
						node.Self.Node.ID: n,
					}
					mb.Sharders = node.NewPool(node.NodeTypeSharder)
					mb.ComputeProperties()

					ps := round.NewRoundStartingStorage()
					if err := ps.Put(mb, 0); err != nil {
						t.Fatal(err)
					}

					return ps
				}(),
				LatestFinalizedBlock: block.NewBlock("", 0),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				MagicBlockStorage:    tt.fields.MagicBlockStorage,
				LatestFinalizedBlock: tt.fields.LatestFinalizedBlock,
				roundsMutex:          &sync.RWMutex{},
			}
			if got := c.IsActiveInChain(); got != tt.want {
				t.Errorf("IsActiveInChain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_SetupNodes(t *testing.T) {
	type args struct {
		mb *block.MagicBlock
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "OK",
			args: args{
				mb: func() *block.MagicBlock {
					mb := block.NewMagicBlock()
					mn, err := makeTestNode(node.NodeTypeMiner, "id")
					if err != nil {
						t.Fatal(err)
					}
					sn, err := makeTestNode(node.NodeTypeSharder, "id")
					if err != nil {
						t.Fatal(err)
					}

					mb.Miners = node.NewPool(node.NodeTypeMiner)
					mb.Miners.NodesMap = map[string]*node.Node{
						mn.ID: mn,
					}
					mb.Sharders = node.NewPool(node.NodeTypeSharder)
					mb.Sharders.NodesMap = map[string]*node.Node{
						sn.ID: sn,
					}

					return mb
				}(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{}

			c.SetupNodes(tt.args.mb)
		})
	}
}
