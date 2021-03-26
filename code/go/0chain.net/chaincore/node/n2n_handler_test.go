package node

import (
	"0chain.net/chaincore/config"
	"context"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func init() {
	SetupN2NHandlers()
}

func makeTestNode(pbK string) (*Node, error) {
	nc := map[interface{}]interface{}{
		"type":       int8(1),
		"public_ip":  "public ip",
		"n2n_ip":     "n2n_ip",
		"port":       8080,
		"id":         "miners node id",
		"public_key": pbK,
	}
	n, err := NewNode(nc)
	if err != nil {
		return nil, err
	}

	return n, nil
}

func TestWithNode(t *testing.T) {
	node, err := makeTestNode("")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.WithValue(context.TODO(), SENDER, node)

	type args struct {
		ctx  context.Context
		node *Node
	}
	tests := []struct {
		name string
		args args
		want context.Context
	}{
		{
			name: "OK",
			args: args{ctx: context.TODO(), node: node},
			want: ctx,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WithNode(tt.args.ctx, tt.args.node); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSender(t *testing.T) {
	node, err := makeTestNode("")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.WithValue(context.TODO(), SENDER, node)

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want *Node
	}{
		{
			name: "OK",
			args: args{ctx: ctx},
			want: node,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSender(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSender() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetHeaders(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(HeaderRequestChainID, config.GetServerChainID())
	r.Header.Set(HeaderNodeID, Self.Underlying().GetKey())

	type args struct {
		req *http.Request
	}
	tests := []struct {
		name string
		args args
		want *http.Request
	}{
		{
			name: "OK",
			args: args{req: r},
			want: r,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetHeaders(tt.args.req)
			assert.Equal(t, tt.want, tt.args.req)
		})
	}
}

func Test_Simple_Setters(t *testing.T) {
	ts := time.Nanosecond
	SetTimeoutSmallMessage(ts)
	assert.Equal(t, ts, TimeoutSmallMessage)

	ts = time.Millisecond
	SetTimeoutLargeMessage(ts)
	assert.Equal(t, ts, TimeoutLargeMessage)

	s := 5
	SetLargeMessageThresholdSize(s)
	assert.Equal(t, s, LargeMessageThreshold)
}
