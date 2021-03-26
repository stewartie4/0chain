package node

import (
	"0chain.net/chaincore/client"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
)

func init() {
	sp := memorystore.GetStorageProvider()
	clientEM := datastore.MetadataProvider()
	clientEM.Name = "client"
	clientEM.Provider = client.Provider
	clientEM.Store = sp
	client.SetEntityMetadata(clientEM)
}

func TestSetSendHeaders(t *testing.T) {
	key := "key"
	txn := client.Provider()
	txn.SetKey(key)

	scheme := encryption.NewED25519Scheme()
	if err := scheme.GenerateKeys(); err != nil {
		t.Fatal(err)
	}
	Self.SetSignatureScheme(scheme)

	type args struct {
		req     *http.Request
		entity  datastore.Entity
		options *SendOptions
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantReq *http.Request
	}{
		{
			name: "TRUE",
			args: args{
				req:    httptest.NewRequest(http.MethodGet, "/", nil),
				entity: txn,
				options: &SendOptions{
					InitialNodeID:  "initial node id",
					MaxRelayLength: 2,
				},
			},
			want: true,
		},
		{
			name: "TRUE2",
			args: args{
				req:    httptest.NewRequest(http.MethodGet, "/", nil),
				entity: txn,
				options: &SendOptions{
					InitialNodeID:  "initial node id",
					CODEC:          1,
					MaxRelayLength: 2,
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SetSendHeaders(tt.args.req, tt.args.entity, tt.args.options); got != tt.want {
				t.Errorf("SetSendHeaders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetMaxConcurrentRequests(t *testing.T) {
	type args struct {
		maxConcurrentRequests int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "OK",
			args: args{maxConcurrentRequests: 5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetMaxConcurrentRequests(tt.args.maxConcurrentRequests)
			assert.Equal(t, MaxConcurrentRequests, tt.args.maxConcurrentRequests)
		})
	}
}

func TestPool_SendAll(t *testing.T) {
	n, err := makeTestNode("")
	if err != nil {
		t.Fatal(err)
	}
	n.Type = NodeTypeMiner

	p := NewPool(NodeTypeMiner)
	p.AddNode(n)
	p.computeNodesArray()

	type fields struct {
		Type              int8
		Nodes             []*Node
		NodesMap          map[string]*Node
		medianNetworkTime uint64
	}
	type args struct {
		handler SendHandler
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []*Node
	}{
		{
			name: "OK",
			fields: fields{
				Type:              p.Type,
				Nodes:             p.Nodes,
				NodesMap:          p.NodesMap,
				medianNetworkTime: p.medianNetworkTime,
			},
			args: args{
				handler: func(n *Node) bool {
					return true
				},
			},
			want: []*Node{n},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			np := &Pool{
				Type:              tt.fields.Type,
				mmx:               sync.RWMutex{},
				Nodes:             tt.fields.Nodes,
				NodesMap:          tt.fields.NodesMap,
				medianNetworkTime: tt.fields.medianNetworkTime,
			}
			if got := np.SendAll(tt.args.handler); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SendAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPool_SendTo(t *testing.T) {
	to := "to"

	n, err := makeTestNode("")
	if err != nil {
		t.Fatal(err)
	}
	n.Type = NodeTypeMiner

	p := NewPool(NodeTypeMiner)

	type fields struct {
		Type              int8
		mmx               sync.RWMutex
		Nodes             []*Node
		NodesMap          map[string]*Node
		medianNetworkTime uint64
	}
	type args struct {
		handler SendHandler
		to      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Unknown_Node_FALSE",
			fields: fields{
				Type:              p.Type,
				Nodes:             p.Nodes,
				NodesMap:          make(map[string]*Node),
				medianNetworkTime: p.medianNetworkTime,
			},
			args: args{
				to: "unknown id",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "Self_Node_FALSE",
			fields: fields{
				Type:  p.Type,
				Nodes: p.Nodes,
				NodesMap: map[string]*Node{
					to: Self.Node,
				},
				medianNetworkTime: p.medianNetworkTime,
			},
			args: args{
				to: to,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "Self_Node_TRUE",
			fields: fields{
				Type:  p.Type,
				Nodes: p.Nodes,
				NodesMap: map[string]*Node{
					to: n,
				},
				medianNetworkTime: p.medianNetworkTime,
			},
			args: args{
				handler: func(n *Node) bool {
					return true
				},
				to: to,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			np := &Pool{
				Type:              tt.fields.Type,
				mmx:               tt.fields.mmx,
				Nodes:             tt.fields.Nodes,
				NodesMap:          tt.fields.NodesMap,
				medianNetworkTime: tt.fields.medianNetworkTime,
			}
			got, err := np.SendTo(tt.args.handler, tt.args.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendTo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SendTo() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPool_SendOne(t *testing.T) {
	t.Skip()

	activeN, err := makeTestNode("")
	if err != nil {
		t.Fatal(err)
	}
	activeN.Type = NodeTypeMiner
	inactiveN, err := makeTestNode("")
	if err != nil {
		t.Fatal(err)
	}
	inactiveN.Type = NodeTypeMiner
	inactiveN.Status = NodeStatusInactive

	p := NewPool(NodeTypeMiner)
	p.AddNode(activeN)
	p.AddNode(inactiveN)
	p.computeNodesArray()

	type fields struct {
		Type              int8
		Nodes             []*Node
		NodesMap          map[string]*Node
		medianNetworkTime uint64
	}
	type args struct {
		handler SendHandler
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Node
	}{
		{
			name: "OK",
			fields: fields{
				Type:              p.Type,
				Nodes:             p.Nodes,
				NodesMap:          p.NodesMap,
				medianNetworkTime: p.medianNetworkTime,
			},
			args: args{
				handler: func(n *Node) bool {
					return true
				},
			},
			want: activeN,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			np := &Pool{
				Type:              tt.fields.Type,
				mmx:               sync.RWMutex{},
				Nodes:             tt.fields.Nodes,
				NodesMap:          tt.fields.NodesMap,
				medianNetworkTime: tt.fields.medianNetworkTime,
			}
			if got := np.SendOne(tt.args.handler); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SendOne() = %v, want %v", got, tt.want)
			}
		})
	}
}
