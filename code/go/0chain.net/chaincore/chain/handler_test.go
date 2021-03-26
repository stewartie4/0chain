package chain

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func init() {
	common.ConfigRateLimits()
	SetupHandlers()
}

func TestGetChainHandler(t *testing.T) {
	type args struct {
		ctx context.Context
		r   *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "ERR", // TODO
			args: args{
				ctx: context.TODO(),
				r:   httptest.NewRequest(http.MethodGet, "/", nil),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetChainHandler(tt.args.ctx, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetChainHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetChainHandler() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLatestBlockFeeStatsHandler(t *testing.T) {
	ch := &Chain{
		FeeStats: transaction.TransactionFeeStats{
			MaxFees:  5,
			MeanFees: 2,
			MinFees:  1,
		},
	}
	SetServerChain(ch)

	type args struct {
		ctx context.Context
		r   *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name:    "OK",
			want:    ch.FeeStats,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LatestBlockFeeStatsHandler(tt.args.ctx, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("LatestBlockFeeStatsHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LatestBlockFeeStatsHandler() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPutChainHandler(t *testing.T) {
	type args struct {
		ctx    context.Context
		entity datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "ERR", // TODO
			args: args{
				ctx:    context.TODO(),
				entity: block.NewBlock("", 1),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PutChainHandler(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("PutChainHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PutChainHandler() got = %v, want %v", got, tt.want)
			}
		})
	}
}
