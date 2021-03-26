package node

import (
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	mocks "0chain.net/mocks/core/datastore"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestGetFetchStrategy(t *testing.T) {
	tests := []struct {
		name string
		typ  int8
		want int
	}{
		{
			name: "Sharder_OK",
			typ:  NodeTypeSharder,
			want: FetchStrategyRandom,
		},
		{
			name: "Miner_OK",
			typ:  NodeTypeMiner,
			want: FetchStrategyNearest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Self.Type = tt.typ
			if got := GetFetchStrategy(); got != tt.want {
				t.Errorf("GetFetchStrategy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPool_RequestEntity(t *testing.T) {
	inactiveN, err := makeTestNode("pbK")
	if err != nil {
		t.Fatal(err)
	}
	inactiveN.Type = NodeTypeMiner
	inactiveN.Status = NodeStatusInactive
	inactiveN.ID = encryption.Hash("inactive node id")
	selfNode := Self.Node
	selfNode.Type = NodeTypeMiner
	selfNode.TimersByURI = make(map[string]metrics.Timer)
	p := NewPool(NodeTypeMiner)
	p.AddNode(inactiveN)
	p.AddNode(selfNode)
	p.ComputeProperties()

	n, err := makeTestNode("pbK")
	if err != nil {
		t.Fatal(err)
	}
	n.ID = encryption.Hash("node id")
	n.Type = NodeTypeSharder
	p2 := NewPool(NodeTypeSharder)
	p2.AddNode(n)
	p2.ComputeProperties()

	rhandler := func(n *Node) bool {
		return true
	}
	requestor := func(urlParams *url.Values, handler datastore.JSONEntityReqResponderF) SendHandler {
		return rhandler
	}

	type fields struct {
		Type              int8
		Nodes             []*Node
		NodesMap          map[string]*Node
		medianNetworkTime uint64
	}
	type args struct {
		ctx       context.Context
		requestor EntityRequestor
		params    *url.Values
		handler   datastore.JSONEntityReqResponderF
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		want     *Node
		selfType int8
	}{
		{
			name: "Nil_Result_OK",
			fields: fields{
				Type:              p.Type,
				Nodes:             p.Nodes,
				NodesMap:          p.NodesMap,
				medianNetworkTime: p.medianNetworkTime,
			},
			args: args{
				ctx:       context.TODO(),
				requestor: requestor,
			},
			selfType: NodeTypeMiner,
			want:     nil,
		},
		{
			name: "Canceled_Context_OK",
			fields: fields{
				Type:              p.Type,
				Nodes:             p.Nodes,
				NodesMap:          p.NodesMap,
				medianNetworkTime: p.medianNetworkTime,
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.TODO())
					cancel()

					return ctx
				}(),
				requestor: requestor,
			},
			selfType: NodeTypeMiner,
			want:     nil,
		},
		{
			name: "OK",
			fields: fields{
				Type:              p2.Type,
				Nodes:             p2.Nodes,
				NodesMap:          p2.NodesMap,
				medianNetworkTime: p2.medianNetworkTime,
			},
			args: args{
				ctx:       context.TODO(),
				requestor: requestor,
			},
			selfType: NodeTypeSharder,
			want:     n,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Self.Type = tt.selfType
			np := &Pool{
				Type:              tt.fields.Type,
				mmx:               sync.RWMutex{},
				Nodes:             tt.fields.Nodes,
				NodesMap:          tt.fields.NodesMap,
				medianNetworkTime: tt.fields.medianNetworkTime,
			}
			if got := np.RequestEntity(tt.args.ctx, tt.args.requestor, tt.args.params, tt.args.handler); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RequestEntity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPool_RequestEntityFromAll(t *testing.T) {
	inactiveN, err := makeTestNode("pbK")
	if err != nil {
		t.Fatal(err)
	}
	inactiveN.Type = NodeTypeMiner
	inactiveN.Status = NodeStatusInactive
	inactiveN.ID = encryption.Hash("inactive node id")
	selfNode := Self.Node
	selfNode.Type = NodeTypeMiner
	selfNode.TimersByURI = make(map[string]metrics.Timer)
	p := NewPool(NodeTypeMiner)
	p.AddNode(inactiveN)
	p.AddNode(selfNode)
	p.ComputeProperties()

	n, err := makeTestNode("pbK")
	if err != nil {
		t.Fatal(err)
	}
	n.ID = encryption.Hash("node id")
	n.Type = NodeTypeSharder
	p2 := NewPool(NodeTypeSharder)
	p2.AddNode(n)
	p2.ComputeProperties()

	rhandler := func(n *Node) bool {
		return true
	}
	requestor := func(urlParams *url.Values, handler datastore.JSONEntityReqResponderF) SendHandler {
		return rhandler
	}

	type fields struct {
		Type              int8
		mmx               sync.RWMutex
		Nodes             []*Node
		NodesMap          map[string]*Node
		medianNetworkTime uint64
	}
	type args struct {
		ctx       context.Context
		requestor EntityRequestor
		params    *url.Values
		handler   datastore.JSONEntityReqResponderF
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		selfType int8
	}{
		{
			name: "Nil_Result_OK",
			fields: fields{
				Type:              p.Type,
				Nodes:             p.Nodes,
				NodesMap:          p.NodesMap,
				medianNetworkTime: p.medianNetworkTime,
			},
			args: args{
				ctx:       context.TODO(),
				requestor: requestor,
			},
			selfType: NodeTypeMiner,
		},
		{
			name: "Canceled_Context_OK",
			fields: fields{
				Type:              p.Type,
				Nodes:             p.Nodes,
				NodesMap:          p.NodesMap,
				medianNetworkTime: p.medianNetworkTime,
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.TODO())
					cancel()

					return ctx
				}(),
				requestor: requestor,
			},
			selfType: NodeTypeMiner,
		},
		{
			name: "OK",
			fields: fields{
				Type:              p2.Type,
				Nodes:             p2.Nodes,
				NodesMap:          p2.NodesMap,
				medianNetworkTime: p2.medianNetworkTime,
			},
			args: args{
				ctx:       context.TODO(),
				requestor: requestor,
			},
			selfType: NodeTypeSharder,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Self.Type = tt.selfType
			np := &Pool{
				Type:              tt.fields.Type,
				mmx:               sync.RWMutex{},
				Nodes:             tt.fields.Nodes,
				NodesMap:          tt.fields.NodesMap,
				medianNetworkTime: tt.fields.medianNetworkTime,
			}

			np.RequestEntityFromAll(tt.args.ctx, tt.args.requestor, tt.args.params, tt.args.handler)
		})
	}
}

func TestNode_RequestEntityFromNode(t *testing.T) {
	rhandler := func(n *Node) bool {
		return true
	}
	requestor := func(urlParams *url.Values, handler datastore.JSONEntityReqResponderF) SendHandler {
		return rhandler
	}

	type fields struct {
		Client                    client.Client
		N2NHost                   string
		Host                      string
		Port                      int
		Path                      string
		Type                      int8
		Description               string
		SetIndex                  int
		Status                    int
		LastActiveTime            time.Time
		ErrorCount                int64
		CommChannel               chan struct{}
		Sent                      int64
		SendErrors                int64
		Received                  int64
		TimersByURI               map[string]metrics.Timer
		SizeByURI                 map[string]metrics.Histogram
		largeMessageSendTime      uint64
		smallMessageSendTime      uint64
		LargeMessagePullServeTime float64
		SmallMessagePullServeTime float64
		mutex                     sync.RWMutex
		mutexInfo                 sync.RWMutex
		ProtocolStats             interface{}
		idBytes                   []byte
		Info                      Info
	}
	type args struct {
		ctx       context.Context
		requestor EntityRequestor
		params    *url.Values
		handler   datastore.JSONEntityReqResponderF
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "FALSE",
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.TODO())
					cancel()

					return ctx
				}(),
				requestor: requestor,
			},
			want: false,
		},
		{
			name: "TRUE",
			args: args{
				ctx:       context.TODO(),
				requestor: requestor,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{
				Client:                    tt.fields.Client,
				N2NHost:                   tt.fields.N2NHost,
				Host:                      tt.fields.Host,
				Port:                      tt.fields.Port,
				Path:                      tt.fields.Path,
				Type:                      tt.fields.Type,
				Description:               tt.fields.Description,
				SetIndex:                  tt.fields.SetIndex,
				Status:                    tt.fields.Status,
				LastActiveTime:            tt.fields.LastActiveTime,
				ErrorCount:                tt.fields.ErrorCount,
				CommChannel:               tt.fields.CommChannel,
				Sent:                      tt.fields.Sent,
				SendErrors:                tt.fields.SendErrors,
				Received:                  tt.fields.Received,
				TimersByURI:               tt.fields.TimersByURI,
				SizeByURI:                 tt.fields.SizeByURI,
				largeMessageSendTime:      tt.fields.largeMessageSendTime,
				smallMessageSendTime:      tt.fields.smallMessageSendTime,
				LargeMessagePullServeTime: tt.fields.LargeMessagePullServeTime,
				SmallMessagePullServeTime: tt.fields.SmallMessagePullServeTime,
				mutex:                     tt.fields.mutex,
				mutexInfo:                 tt.fields.mutexInfo,
				ProtocolStats:             tt.fields.ProtocolStats,
				idBytes:                   tt.fields.idBytes,
				Info:                      tt.fields.Info,
			}
			if got := n.RequestEntityFromNode(tt.args.ctx, tt.args.requestor, tt.args.params, tt.args.handler); got != tt.want {
				t.Errorf("RequestEntityFromNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetRequestHeaders(t *testing.T) {
	options := SendOptions{
		InitialNodeID: "initial node id",
	}

	em := datastore.EntityMetadataImpl{
		Name: "name",
	}

	type args struct {
		req            *http.Request
		options        *SendOptions
		entityMetadata datastore.EntityMetadata
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		wantR *http.Request
	}{
		{
			name: "Zero_Codec_OK",
			args: args{
				req:     httptest.NewRequest(http.MethodGet, "/", nil),
				options: &options,
			},
			want: true,
			wantR: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Set(HeaderRequestChainID, config.GetServerChainID())
				r.Header.Set(HeaderNodeID, Self.Underlying().GetKey())
				r.Header.Set(HeaderInitialNodeID, options.InitialNodeID)
				r.Header.Set(HeaderRequestEntityName, em.Name)
				r.Header.Set(HeaderRequestCODEC, CodecJSON)

				return r
			}(),
		},
		{
			name: "OK",
			args: args{
				req: httptest.NewRequest(http.MethodGet, "/", nil),
				options: &SendOptions{
					InitialNodeID: options.InitialNodeID,
					CODEC:         1,
				},
			},
			want: true,
			wantR: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Set(HeaderRequestChainID, config.GetServerChainID())
				r.Header.Set(HeaderNodeID, Self.Underlying().GetKey())
				r.Header.Set(HeaderInitialNodeID, options.InitialNodeID)
				r.Header.Set(HeaderRequestEntityName, em.Name)
				r.Header.Set(HeaderRequestCODEC, CodecMsgpack)

				return r
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SetRequestHeaders(tt.args.req, tt.args.options, tt.args.entityMetadata); got != tt.want {
				t.Errorf("SetRequestHeaders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequestEntityHandler(t *testing.T) {
	entity := &client.Client{}
	entity.ID = "client id"

	server := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				blob, err := json.Marshal(entity)
				if err != nil {
					t.Fatal(err)
				}

				if _, err := rw.Write(blob); err != nil {
					t.Fatal(err)
				}

				return
			},
		),
	)
	defer server.Close()

	n, err := makeTestNode("pbk")
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	n.N2NHost = u.Hostname()
	if n.Port, err = strconv.Atoi(u.Port()); err != nil {
		t.Fatal(err)
	}

	invalidServer := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				rw.WriteHeader(http.StatusInternalServerError)
			},
		),
	)
	defer invalidServer.Close()

	emptyServer := httptest.NewServer(
		http.HandlerFunc(
			func(http.ResponseWriter, *http.Request) {
				return
			},
		),
	)
	defer emptyServer.Close()

	type args struct {
		uri            string
		options        *SendOptions
		entityMetadata datastore.EntityMetadata

		params  *url.Values
		handler datastore.JSONEntityReqResponderF

		node *Node
	}
	tests := []struct {
		name   string
		args   args
		client *http.Client
		want   bool
	}{
		{
			name: "Invalid_Url_FALSE",
			args: args{
				node: func() *Node {
					n, err := makeTestNode("pbk")
					if err != nil {
						t.Fatal(err)
					}
					n.N2NHost = "}{"

					return n
				}(),
				options: &SendOptions{},
			},
			want: false,
		},
		{
			name: "Client_Err_FALSE",
			args: args{
				uri: "",
				node: func() *Node {
					n, err := makeTestNode("pbk")
					if err != nil {
						t.Fatal(err)
					}

					return n
				}(),
				options: &SendOptions{},
			},
			client: server.Client(),
			want:   false,
		},
		{
			name: "Resp_Status_Not_OK_FALSE",
			args: args{
				uri: "",
				node: func() *Node {
					n, err := makeTestNode("pbk")
					if err != nil {
						t.Fatal(err)
					}
					u, err := url.Parse(invalidServer.URL)
					if err != nil {
						t.Fatal(err)
					}
					n.N2NHost = u.Hostname()
					if n.Port, err = strconv.Atoi(u.Port()); err != nil {
						t.Fatal(err)
					}

					return n
				}(),
				options: &SendOptions{},
			},
			client: invalidServer.Client(),
			want:   false,
		},
		{
			name: "Nil_Entity_Metadata_FALSE",
			args: args{
				uri:     "",
				node:    n,
				options: &SendOptions{},
			},
			client: server.Client(),
			want:   false,
		},
		{
			name: "Response_Entity_Err_FALSE",
			args: args{
				uri: "",
				node: func() *Node {
					n, err := makeTestNode("pbk")
					if err != nil {
						t.Fatal(err)
					}
					u, err := url.Parse(emptyServer.URL)
					if err != nil {
						t.Fatal(err)
					}
					n.N2NHost = u.Hostname()
					if n.Port, err = strconv.Atoi(u.Port()); err != nil {
						t.Fatal(err)
					}

					return n
				}(),
				options: &SendOptions{},
				entityMetadata: &datastore.EntityMetadataImpl{
					Provider: func() datastore.Entity {
						return &mocks.Entity{}
					},
				},
			},
			client: emptyServer.Client(),
			want:   false,
		},
		{
			name: "Handler_Err_FALSE",
			args: args{
				uri:     "",
				node:    n,
				options: &SendOptions{},
				entityMetadata: &datastore.EntityMetadataImpl{
					Provider: func() datastore.Entity {
						return &client.Client{}
					},
				},
				handler: func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
					return nil, errors.New("")
				},
			},
			client: server.Client(),
			want:   false,
		},
		{
			name: "TRUE",
			args: args{
				uri:  "",
				node: n,
				options: &SendOptions{
					Timeout: time.Second,
				},
				entityMetadata: &datastore.EntityMetadataImpl{
					Provider: func() datastore.Entity {
						return &client.Client{}
					},
				},
				handler: func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
					return nil, nil
				},
			},
			client: server.Client(),
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient = tt.client

			requestor := RequestEntityHandler(tt.args.uri, tt.args.options, tt.args.entityMetadata)
			handler := requestor(tt.args.params, tt.args.handler)
			got := handler(tt.args.node)
			if !assert.True(t, got == tt.want) {
				panic(got)
			}
		})
	}
}

func TestToN2NSendEntityHandler(t *testing.T) {
	cl := &client.Client{}
	clHandler := func(ctx context.Context, r *http.Request) (interface{}, error) {
		return cl, nil
	}

	pd := &pushDataCacheEntry{}
	pdHadler := func(ctx context.Context, r *http.Request) (interface{}, error) {
		return pd, nil
	}

	n, err := makeTestNode("pbK")
	if err != nil {
		t.Fatal(err)
	}
	n.ID = encryption.Hash("id")
	RegisterNode(n)

	em := datastore.EntityMetadataImpl{
		Name: "name",
	}
	datastore.RegisterEntityMetadata(em.Name, &em)

	type args struct {
		handler common.JSONResponderF

		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name  string
		args  args
		wantW http.ResponseWriter
	}{
		{
			name: "Nil_Node_OK",
			args: args{
				handler: clHandler,
				w:       httptest.NewRecorder(),
				r: func() *http.Request {
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					r.Header.Set(HeaderNodeID, "unknown id")

					return r
				}(),
			},
			wantW: httptest.NewRecorder(),
		},
		{
			name: "Invalid_Chain_OK",
			args: args{
				handler: clHandler,
				w:       httptest.NewRecorder(),
				r: func() *http.Request {
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					r.Header.Set(HeaderNodeID, n.ID)

					return r
				}(),
			},
			wantW: httptest.NewRecorder(),
		},
		{
			name: "Invalid_Entity_Metadata_OK",
			args: args{
				handler: clHandler,
				w:       httptest.NewRecorder(),
				r: func() *http.Request {
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					r.Header.Set(HeaderNodeID, n.ID)
					r.Header.Set(HeaderRequestChainID, config.GetMainChainID())

					return r
				}(),
			},
			wantW: httptest.NewRecorder(),
		},
		{
			name: "Handler_Err_OK",
			args: args{
				handler: func(ctx context.Context, r *http.Request) (interface{}, error) {
					return nil, errors.New("")
				},
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					r.Header.Set(HeaderNodeID, n.ID)
					r.Header.Set(HeaderRequestChainID, config.GetMainChainID())
					r.Header.Set(HeaderRequestEntityName, em.Name)

					return r
				}(),
			},
			wantW: func() http.ResponseWriter {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				err := errors.New("")
				common.Respond(w, r, nil, err)

				return w
			}(),
		},
		{
			name: "Entity_OK",
			args: args{
				handler: clHandler,
				w:       httptest.NewRecorder(),
				r: func() *http.Request {
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					r.Header.Set(HeaderNodeID, n.ID)
					r.Header.Set(HeaderRequestChainID, config.GetMainChainID())
					r.Header.Set(HeaderRequestEntityName, em.Name)
					r.Header.Set(HeaderRequestCODEC, "JSON")

					return r
				}(),
			},
			wantW: func() http.ResponseWriter {
				w := httptest.NewRecorder()
				w.Header().Set("Content-Encoding", compDecomp.Encoding())
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set(HeaderRequestCODEC, "JSON")

				buffer := datastore.ToJSON(cl)
				cbytes := compDecomp.Compress(buffer.Bytes())
				buffer = bytes.NewBuffer(cbytes)
				sdata := buffer.Bytes()
				if _, err := w.Write(sdata); err != nil {
					t.Fatal(err)
				}

				return w
			}(),
		},
		{
			name: "Push_Data_OK",
			args: args{
				handler: pdHadler,
				w:       httptest.NewRecorder(),
				r: func() *http.Request {
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					r.Header.Set(HeaderNodeID, n.ID)
					r.Header.Set(HeaderRequestChainID, config.GetMainChainID())
					r.Header.Set(HeaderRequestEntityName, em.Name)
					r.Header.Set(HeaderRequestCODEC, "JSON")

					return r
				}(),
			},
			wantW: func() http.ResponseWriter {
				w := httptest.NewRecorder()
				w.Header().Set("Content-Encoding", compDecomp.Encoding())
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set(HeaderRequestCODEC, CodecJSON)
				w.Header().Set(HeaderRequestEntityName, pd.EntityName)

				buffer := bytes.NewBuffer(pd.Data)
				sdata := buffer.Bytes()
				if _, err := w.Write(sdata); err != nil {
					t.Fatal(err)
				}

				return w
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqRespHandlerF := ToN2NSendEntityHandler(tt.args.handler)
			reqRespHandlerF(tt.args.w, tt.args.r)
			assert.Equal(t, tt.wantW, tt.args.w)
		})
	}
}

func TestToS2MSendEntityHandler(t *testing.T) {
	cl := &client.Client{}
	clHandler := func(ctx context.Context, r *http.Request) (interface{}, error) {
		return cl, nil
	}

	pd := &pushDataCacheEntry{}
	pdHadler := func(ctx context.Context, r *http.Request) (interface{}, error) {
		return pd, nil
	}

	type args struct {
		handler common.JSONResponderF
		w       http.ResponseWriter
		r       *http.Request
	}
	tests := []struct {
		name  string
		args  args
		wantW http.ResponseWriter
	}{
		{
			name: "Handler_Err_OK",
			args: args{
				handler: func(ctx context.Context, r *http.Request) (interface{}, error) {
					return nil, errors.New("")
				},
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodGet, "/", nil),
			},
			wantW: func() http.ResponseWriter {
				w := httptest.NewRecorder()

				r := httptest.NewRequest(http.MethodGet, "/", nil)
				err := errors.New("")
				common.Respond(w, r, nil, err)

				return w
			}(),
		},
		{
			name: "Entity_OK",
			args: args{
				handler: clHandler,
				w:       httptest.NewRecorder(),
				r: func() *http.Request {
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					r.Header.Set(HeaderRequestCODEC, "JSON")

					return r
				}(),
			},
			wantW: func() http.ResponseWriter {
				w := httptest.NewRecorder()
				w.Header().Set(HeaderRequestCODEC, "JSON")
				w.Header().Set("Content-Encoding", compDecomp.Encoding())
				w.Header().Set("Content-Type", "application/json")

				buffer := datastore.ToJSON(cl)
				cbytes := compDecomp.Compress(buffer.Bytes())
				buffer = bytes.NewBuffer(cbytes)
				sdata := buffer.Bytes()
				if _, err := w.Write(sdata); err != nil {
					t.Fatal(err)
				}

				return w
			}(),
		},
		{
			name: "Cache_Entity_OK",
			args: args{
				handler: pdHadler,
				w:       httptest.NewRecorder(),
				r:       httptest.NewRequest(http.MethodGet, "/", nil),
			},
			wantW: func() http.ResponseWriter {
				w := httptest.NewRecorder()
				w.Header().Set(HeaderRequestCODEC, CodecJSON)
				w.Header().Set(HeaderRequestEntityName, pd.EntityName)
				w.Header().Set("Content-Encoding", compDecomp.Encoding())
				w.Header().Set("Content-Type", "application/json")

				buffer := bytes.NewBuffer(pd.Data)
				sdata := buffer.Bytes()
				if _, err := w.Write(sdata); err != nil {
					t.Fatal(err)
				}

				return w
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqRespHandlerF := ToS2MSendEntityHandler(tt.args.handler)
			reqRespHandlerF(tt.args.w, tt.args.r)
			assert.Equal(t, tt.wantW, tt.args.w)
		})
	}
}
