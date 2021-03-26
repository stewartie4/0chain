package chain

import (
	"0chain.net/chaincore/block"
	"0chain.net/smartcontract/minersc"
	"reflect"
	"sync"
	"testing"
)

func TestChain_GetPhaseOfBlock(t *testing.T) {
	type fields struct {
	}
	type args struct {
		b *block.Block
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantPn  minersc.PhaseNode
		wantErr bool
	}{
		{
			name: "ERR",
			args: args{
				b: block.NewBlock("", 1),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chain{
				stateMutex: &sync.RWMutex{},
			}
			gotPn, err := c.GetPhaseOfBlock(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPhaseOfBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotPn, tt.wantPn) {
				t.Errorf("GetPhaseOfBlock() gotPn = %v, want %v", gotPn, tt.wantPn)
			}
		})
	}
}
