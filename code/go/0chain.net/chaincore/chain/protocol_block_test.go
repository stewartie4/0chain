package chain

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"context"
	"github.com/stretchr/testify/assert"
	"reflect"
	"sync"
	"testing"
)

func TestChain_VerifyTicket(t *testing.T) {
	client.SetClientSignatureScheme("ed25519")

	pbK, prK, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}

	hash := encryption.Hash("data")
	sign, err := encryption.Sign(prK, hash)
	if err != nil {
		t.Fatal(err)
	}

	bvt := block.VerificationTicket{}
	bvt.VerifierID = "verifier id"
	bvt.Signature = sign

	type fields struct {
		MagicBlockStorage round.RoundStorage
	}
	type args struct {
		ctx       context.Context
		blockHash string
		bvt       *block.VerificationTicket
		round     int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Nil_Sender_ERR",
			fields: fields{
				MagicBlockStorage: func() round.RoundStorage {
					mb := block.NewMagicBlock()
					mb.Miners = node.NewPool(node.NodeTypeMiner)
					mb.Miners.NodesMap = map[string]*node.Node{}

					rs := round.NewRoundStartingStorage()
					if err := rs.Put(mb, 0); err != nil {
						t.Fatal(err)
					}

					return rs
				}(),
			},
			args:    args{bvt: &bvt},
			wantErr: true,
		},
		{
			name: "Verifying_ERR",
			fields: fields{
				MagicBlockStorage: func() round.RoundStorage {
					anotherPbK, _, err := encryption.GenerateKeys()
					if err != nil {
						t.Fatal(err)
					}

					n, err := makeTestNode(node.NodeTypeMiner, bvt.VerifierID)
					if err != nil {
						t.Fatal(err)
					}
					n.PublicKey = anotherPbK

					mb := block.NewMagicBlock()
					mb.Miners = node.NewPool(node.NodeTypeMiner)
					mb.Miners.NodesMap = map[string]*node.Node{
						bvt.VerifierID: n,
					}
					mb.Miners.ComputeProperties()

					rs := round.NewRoundStartingStorage()
					if err := rs.Put(mb, 0); err != nil {
						t.Fatal(err)
					}

					return rs
				}(),
			},
			args:    args{bvt: &bvt},
			wantErr: true,
		},
		{
			name: "OK",
			fields: fields{
				MagicBlockStorage: func() round.RoundStorage {

					n, err := makeTestNode(node.NodeTypeMiner, bvt.VerifierID)
					if err != nil {
						t.Fatal(err)
					}
					n.PublicKey = pbK

					mb := block.NewMagicBlock()
					mb.Miners = node.NewPool(node.NodeTypeMiner)
					mb.Miners.NodesMap = map[string]*node.Node{
						bvt.VerifierID: n,
					}
					mb.Miners.ComputeProperties()

					rs := round.NewRoundStartingStorage()
					if err := rs.Put(mb, 0); err != nil {
						t.Fatal(err)
					}

					return rs
				}(),
			},
			args: args{
				blockHash: hash,
				bvt:       &bvt,
				round:     0,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				MagicBlockStorage: tt.fields.MagicBlockStorage,
			}
			if err := c.VerifyTicket(tt.args.ctx, tt.args.blockHash, tt.args.bvt, tt.args.round); (err != nil) != tt.wantErr {
				t.Errorf("VerifyTicket() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_VerifyNotarization(t *testing.T) {
	client.SetClientSignatureScheme("ed25519")

	b := block.NewBlock("", 1)
	b.HashBlock()

	lfmbr := int64(5)
	lfb := block.NewBlock("", lfmbr)

	pbK, prK, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}

	sign, err := encryption.Sign(prK, b.Hash)
	if err != nil {
		t.Fatal(err)
	}

	verID := "id"
	bvt := []*block.VerificationTicket{
		{
			VerifierID: verID,
			Signature:  sign,
		},
	}

	n, err := makeTestNode(node.NodeTypeMiner, verID)
	if err != nil {
		t.Fatal(err)
	}
	n.PublicKey = pbK

	mb := block.NewMagicBlock()
	mb.Miners = node.NewPool(node.NodeTypeMiner)
	mb.Miners.NodesMap[verID] = n
	mb.Miners.ComputeProperties()

	mbs := round.NewRoundStartingStorage()
	if err := mbs.Put(mb, lfmbr); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Config                         *Config
		MagicBlockStorage              round.RoundStorage
		LatestFinalizedBlock           *block.Block
		LatestFinalizedMagicBlockRound int64
		minersStake                    map[datastore.Key]int
	}
	type args struct {
		ctx   context.Context
		b     *block.Block
		bvt   []*block.VerificationTicket
		round int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Nil_BVT_ERR",
			wantErr: true,
		},
		{
			name: "Verifying_Related_Magic_Block_Presence_ERR",
			fields: fields{
				LatestFinalizedBlock: lfb,
				MagicBlockStorage:    mbs,
			},
			args: args{
				bvt: []*block.VerificationTicket{
					{},
				},
				b: func() *block.Block {
					b := block.NewBlock("", 1)
					b.LatestFinalizedMagicBlockRound = lfmbr - 1

					return b
				}(),
			},
			wantErr: true,
		},
		{
			name: "Nil_Ticket_ERR",
			fields: fields{
				LatestFinalizedBlock: lfb,
				MagicBlockStorage:    mbs,
			},
			args: args{
				bvt: []*block.VerificationTicket{
					nil,
				},
				b: block.NewBlock("", 1),
			},
			wantErr: true,
		},
		{
			name: "Duplicate_Ticket_ERR",
			fields: fields{
				LatestFinalizedBlock: lfb,
				MagicBlockStorage:    mbs,
			},
			args: args{
				bvt: []*block.VerificationTicket{
					{
						VerifierID: "dupulicate id",
					},
					{
						VerifierID: "dupulicate id",
					},
				},
				b: block.NewBlock("", 1),
			},
			wantErr: true,
		},
		{
			name: "Block_Not_Notarized_ERR",
			fields: fields{
				Config: &Config{
					ThresholdByCount: 1,
					ThresholdByStake: 1,
				},
				LatestFinalizedBlock: lfb,
				MagicBlockStorage:    mbs,
			},
			args: args{
				bvt: []*block.VerificationTicket{
					{
						VerifierID: "id",
					},
				},
				b: block.NewBlock("", 1),
			},
			wantErr: true,
		},
		{
			name: "Verifying_Ticket_ERR",
			fields: fields{
				Config:               &Config{},
				LatestFinalizedBlock: lfb,
				MagicBlockStorage:    mbs,
			},
			args: args{
				bvt: bvt,
				b:   block.NewBlock("", 1),
			},
			wantErr: true,
		},
		{
			name: "OK",
			fields: fields{
				Config:               &Config{},
				LatestFinalizedBlock: lfb,
				MagicBlockStorage:    mbs,
			},
			args: args{
				bvt: bvt,
				b:   b,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				Config:                       tt.fields.Config,
				MagicBlockStorage:            tt.fields.MagicBlockStorage,
				LatestFinalizedBlock:         tt.fields.LatestFinalizedBlock,
				latestOwnFinalizedBlockRound: tt.fields.LatestFinalizedMagicBlockRound,
				roundsMutex:                  &sync.RWMutex{},
				minersStake:                  tt.fields.minersStake,
			}
			if err := c.VerifyNotarization(tt.args.ctx, tt.args.b, tt.args.bvt, tt.args.round); (err != nil) != tt.wantErr {
				t.Errorf("VerifyNotarization() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_VerifyRelatedMagicBlockPresence(t *testing.T) {
	rNum := int64(5)

	type fields struct {
		MagicBlockStorage    round.RoundStorage
		LatestFinalizedBlock *block.Block
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
				MagicBlockStorage: func() round.RoundStorage {
					rs := round.NewRoundStartingStorage()
					mb := block.NewMagicBlock()
					if err := rs.Put(mb, rNum); err != nil {
						t.Error(err)
					}

					return rs
				}(),
				LatestFinalizedBlock: block.NewBlock("", 1),
			},
			args:    args{b: block.NewBlock("", 1)},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				MagicBlockStorage:    tt.fields.MagicBlockStorage,
				LatestFinalizedBlock: tt.fields.LatestFinalizedBlock,
			}
			if err := c.VerifyRelatedMagicBlockPresence(tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("VerifyRelatedMagicBlockPresence() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_IsBlockNotarized(t *testing.T) {
	type fields struct {
		Config               *Config
		MagicBlockStorage    round.RoundStorage
		LatestFinalizedBlock *block.Block
	}
	type args struct {
		ctx context.Context
		b   *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Notarized_Block_TRUE",
			args: args{
				b: func() *block.Block {
					b := block.NewBlock("", 1)
					b.SetBlockNotarized()

					return b
				}(),
			},
			want: true,
		},
		{
			name: "Verify_Related_Magic_Block_Presence_FALSE",
			fields: fields{
				Config: &Config{},
				MagicBlockStorage: func() round.RoundStorage {
					rs := round.NewRoundStartingStorage()
					mb := block.NewMagicBlock()
					mb.Miners = node.NewPool(node.NodeTypeMiner)
					mb.StartingRound = 1
					if err := rs.Put(mb, 1); err != nil {
						t.Error(err)
					}

					return rs
				}(),
				LatestFinalizedBlock: block.NewBlock("", 1),
			},
			args: args{
				b: block.NewBlock("", 1),
			},
			want: false,
		},
		{
			name: "TRUE",
			fields: fields{
				Config: &Config{},
				MagicBlockStorage: func() round.RoundStorage {
					rs := round.NewRoundStartingStorage()
					mb := block.NewMagicBlock()
					mb.Miners = node.NewPool(node.NodeTypeMiner)
					if err := rs.Put(mb, 1); err != nil {
						t.Error(err)
					}

					return rs
				}(),
				LatestFinalizedBlock: block.NewBlock("", 1),
			},
			args: args{
				b: block.NewBlock("", 1),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				Config:               tt.fields.Config,
				MagicBlockStorage:    tt.fields.MagicBlockStorage,
				LatestFinalizedBlock: tt.fields.LatestFinalizedBlock,
				roundsMutex:          &sync.RWMutex{},
			}
			if got := c.IsBlockNotarized(tt.args.ctx, tt.args.b); got != tt.want {
				t.Errorf("IsBlockNotarized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_AddVerificationTicket(t *testing.T) {
	type args struct {
		ctx context.Context
		b   *block.Block
		bvt *block.VerificationTicket
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TRUE",
			args: args{
				b: func() *block.Block {
					b := block.NewBlock("", 1)
					b.SetBlockNotarized()

					return b
				}(),
				bvt: &block.VerificationTicket{
					VerifierID: "ver id",
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{}
			if got := c.AddVerificationTicket(tt.args.ctx, tt.args.b, tt.args.bvt); got != tt.want {
				t.Errorf("AddVerificationTicket() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_IsFinalizedDeterministically(t *testing.T) {
	n, err := makeTestNode(node.NodeTypeMiner, "id")
	if err != nil {
		t.Fatal(err)
	}

	mb := block.NewMagicBlock()
	mb.Miners = node.NewPool(node.NodeTypeMiner)
	mb.Miners.NodesMap[n.ID] = n
	mb.Miners.ComputeProperties()

	rs := round.NewRoundStartingStorage()
	if err := rs.Put(mb, 1); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Config               *Config
		MagicBlockStorage    round.RoundStorage
		LatestFinalizedBlock *block.Block
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
			name: "Block_Round_Greater_Than_LFB_Round_FALSE",
			fields: fields{
				MagicBlockStorage: func() round.RoundStorage {
					mb := block.NewMagicBlock()
					rs := round.NewRoundStartingStorage()
					if err := rs.Put(mb, 0); err != nil {
						t.Fatal(err)
					}

					return rs
				}(),
				LatestFinalizedBlock: block.NewBlock("", 1),
			},
			args: args{
				b: block.NewBlock("", 2),
			},
			want: false,
		},
		{
			name: "TRUE",
			fields: fields{
				Config:               &Config{},
				MagicBlockStorage:    rs,
				LatestFinalizedBlock: block.NewBlock("", 1),
			},
			args: args{
				b: block.NewBlock("", 1),
			},
			want: true,
		},
		{
			name: "FALSE",
			fields: fields{
				Config: &Config{
					ThresholdByCount: 200,
				},
				MagicBlockStorage:    rs,
				LatestFinalizedBlock: block.NewBlock("", 1),
			},
			args: args{
				b: func() *block.Block {
					b := block.NewBlock("", 1)
					b.UniqueBlockExtensions = map[string]bool{}

					return b
				}(),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				Config:               tt.fields.Config,
				MagicBlockStorage:    tt.fields.MagicBlockStorage,
				LatestFinalizedBlock: tt.fields.LatestFinalizedBlock,
			}
			if got := c.IsFinalizedDeterministically(tt.args.b); got != tt.want {
				t.Errorf("IsFinalizedDeterministically() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetLocalPreviousBlock(t *testing.T) {
	b := block.NewBlock("", 1)
	b.HashBlock()
	prevB := block.NewBlock("", 0)
	prevB.HashBlock()
	b.PrevBlock = prevB

	blocks := map[datastore.Key]*block.Block{
		b.Hash:     b,
		prevB.Hash: prevB,
	}

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
		wantPb *block.Block
	}{
		{
			name: "OK",
			fields: fields{
				blocks: blocks,
			},
			args: args{
				b: func() *block.Block {
					b := block.NewBlock("", 1)
					b.PrevHash = prevB.Hash

					return b
				}(),
			},
			wantPb: prevB,
		},
		{
			name: "Nil_Prev_Block_OK",
			fields: fields{
				blocks: blocks,
			},
			args:   args{b: b},
			wantPb: b.PrevBlock,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				blocks:      tt.fields.blocks,
				blocksMutex: &sync.RWMutex{},
			}
			if gotPb := c.GetLocalPreviousBlock(tt.args.ctx, tt.args.b); !reflect.DeepEqual(gotPb, tt.wantPb) {
				t.Errorf("GetLocalPreviousBlock() = %v, want %v", gotPb, tt.wantPb)
			}
		})
	}
}

func TestChain_UpdateNodeState(t *testing.T) {
	n, err := makeTestNode(node.NodeTypeMiner, "id")
	if err != nil {
		t.Fatal(err)
	}
	n.Status = node.NodeStatusInactive

	mb2 := block.NewMagicBlock()
	mb2.Miners = node.NewPool(node.NodeTypeMiner)
	mb3 := block.NewMagicBlock()
	mb3.Miners = node.NewPool(node.NodeTypeMiner)
	mb3.Miners.NodesMap[n.ID] = n
	mb3.Miners.ComputeProperties()

	rs := round.NewRoundStartingStorage()
	if err := rs.Put(mb2, 1); err != nil {
		t.Fatal(err)
	}
	if err := rs.Put(mb3, 2); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		rounds            map[int64]round.RoundI
		MagicBlockStorage round.RoundStorage
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
			name: "Nil_Round_OK",
			fields: fields{
				MagicBlockStorage: rs,
				rounds: map[int64]round.RoundI{
					0: nil,
				},
			},
			args: args{b: block.NewBlock("", 0)},
			want: &Chain{
				MagicBlockStorage: rs,
				rounds: map[int64]round.RoundI{
					0: nil,
				},
				roundsMutex: &sync.RWMutex{},
				mbMutex:     sync.RWMutex{},
			},
		},
		{
			name: "Nil_Signer_OK",
			fields: fields{
				MagicBlockStorage: rs,
				rounds: map[int64]round.RoundI{
					1: round.NewRound(1),
				},
			},
			args: args{
				b: func() *block.Block {
					b := block.NewBlock("", 1)
					b.VerificationTickets = []*block.VerificationTicket{
						{
							VerifierID: "unknown ver id",
						},
					}

					return b
				}(),
			},
			want: &Chain{
				MagicBlockStorage: rs,
				rounds: map[int64]round.RoundI{
					1: round.NewRound(1),
				},
				roundsMutex: &sync.RWMutex{},
				mbMutex:     sync.RWMutex{},
			},
		},
		{
			name: "OK",
			fields: fields{
				MagicBlockStorage: rs,
				rounds: map[int64]round.RoundI{
					2: round.NewRound(2),
				},
			},
			args: args{
				b: func() *block.Block {
					b := block.NewBlock("", 2)
					b.VerificationTickets = []*block.VerificationTicket{
						{
							VerifierID: n.ID,
						},
					}

					return b
				}(),
			},
			want: &Chain{
				MagicBlockStorage: rs,
				rounds: map[int64]round.RoundI{
					2: round.NewRound(2),
				},
				roundsMutex: &sync.RWMutex{},
				mbMutex:     sync.RWMutex{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				rounds:            tt.fields.rounds,
				roundsMutex:       &sync.RWMutex{},
				mbMutex:           sync.RWMutex{},
				MagicBlockStorage: tt.fields.MagicBlockStorage,
			}

			c.UpdateNodeState(tt.args.b)
			assert.Equal(t, tt.want, c)
		})
	}
}

func TestChain_MergeVerificationTickets(t *testing.T) {
	b := block.NewBlock("", 1)
	vts := []*block.VerificationTicket{
		{
			VerifierID: "ver id",
		},
	}
	b.MergeVerificationTickets(vts)
	b.SetBlockNotarized()

	type fields struct {
	}
	type args struct {
		ctx context.Context
		b   *block.Block
		vts []*block.VerificationTicket
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		wantB  *block.Block
	}{
		{
			name: "OK",
			args: args{
				b: func() *block.Block {
					b := block.NewBlock("", b.Round)
					b.SetBlockNotarized()

					return b
				}(),
				vts: vts,
			},
			wantB: b,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{}
			c.MergeVerificationTickets(tt.args.ctx, tt.args.b, tt.args.vts)
			assert.Equal(t, tt.wantB, tt.args.b)
		})
	}
}

func TestChain_GetPreviousBlock(t *testing.T) {
	pbs := block.NewBlock("", 1)
	pbs.SetStateStatus(block.StateSuccessful)
	pbs.HashBlock()

	pbns := block.NewBlock("", 1)
	pbns.SetStateStatus(block.StateFailed)
	pbns.HashBlock()

	type fields struct {
		blocks map[datastore.Key]*block.Block
	}
	type args struct {
		ctx context.Context
		b   *block.Block
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      *block.Block
		state     int
		wantPanic bool
	}{
		{
			name: "Prev_Block_Not_Nil_OK",
			args: args{
				b: func() *block.Block {
					b := block.NewBlock("", 2)
					b.PrevBlock = pbs
					return b
				}(),
			},
			want: pbs,
		},
		{
			name: "Prev_Block_By_Prev_Hash_Nil",
			fields: fields{
				blocks: map[datastore.Key]*block.Block{
					pbs.Hash: pbs,
				},
			},
			args: args{
				b: func() *block.Block {
					b := block.NewBlock("", 2)
					b.PrevHash = pbs.Hash
					return b
				}(),
			},
			want: pbs,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				blocks:      tt.fields.blocks,
				blocksMutex: &sync.RWMutex{},
				roundsMutex: &sync.RWMutex{},
			}
			if got := c.GetPreviousBlock(tt.args.ctx, tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPreviousBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}
