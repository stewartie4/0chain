package node

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestInfo_SetMinersMedianNetworkTime(t *testing.T) {
	mmnt := time.Second

	type fields struct {
		mx                      sync.Mutex
		AsOf                    time.Time
		BuildTag                string
		StateMissingNodes       int64
		MinersMedianNetworkTime time.Duration
		AvgBlockTxns            int
	}
	type args struct {
		mmnt time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Info
	}{
		{
			name:   "OK",
			fields: fields{MinersMedianNetworkTime: mmnt},
			args:   args{mmnt: mmnt},
			want:   &Info{MinersMedianNetworkTime: mmnt},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Info{
				mx:                      tt.fields.mx,
				AsOf:                    tt.fields.AsOf,
				BuildTag:                tt.fields.BuildTag,
				StateMissingNodes:       tt.fields.StateMissingNodes,
				MinersMedianNetworkTime: tt.fields.MinersMedianNetworkTime,
				AvgBlockTxns:            tt.fields.AvgBlockTxns,
			}

			i.SetMinersMedianNetworkTime(tt.args.mmnt)
			assert.Equal(t, tt.want, i)
		})
	}
}

func TestInfo_Copy(t *testing.T) {
	i := Info{
		AsOf:                    time.Now(),
		BuildTag:                "build tag",
		StateMissingNodes:       2,
		MinersMedianNetworkTime: 3,
		AvgBlockTxns:            4,
	}

	type fields struct {
		AsOf                    time.Time
		BuildTag                string
		StateMissingNodes       int64
		MinersMedianNetworkTime time.Duration
		AvgBlockTxns            int
	}
	tests := []struct {
		name   string
		fields fields
		wantCp Info
	}{
		{
			name: "OK",
			fields: fields{
				AsOf:                    i.AsOf,
				BuildTag:                i.BuildTag,
				StateMissingNodes:       i.StateMissingNodes,
				MinersMedianNetworkTime: i.MinersMedianNetworkTime,
				AvgBlockTxns:            i.AvgBlockTxns,
			},
			wantCp: i,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Info{
				mx:                      sync.Mutex{},
				AsOf:                    tt.fields.AsOf,
				BuildTag:                tt.fields.BuildTag,
				StateMissingNodes:       tt.fields.StateMissingNodes,
				MinersMedianNetworkTime: tt.fields.MinersMedianNetworkTime,
				AvgBlockTxns:            tt.fields.AvgBlockTxns,
			}
			gotCp := i.Copy()
			assert.Equal(t, tt.wantCp, gotCp)
		})
	}
}
