package node

import (
	"context"
	"reflect"
	"testing"
)

func TestGetNodeContext(t *testing.T) {
	tests := []struct {
		name string
		want context.Context
	}{
		{
			name: "OK",
			want: context.WithValue(context.Background(), SelfNodeKey, Self),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetNodeContext(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetNodeContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSelfNode(t *testing.T) {
	ctx := GetNodeContext()

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want *SelfNode
	}{
		{
			name: "OK",
			args: args{ctx: ctx},
			want: Self,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSelfNode(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSelfNode() = %v, want %v", got, tt.want)
			}
		})
	}
}
