package node

import (
	"0chain.net/chaincore/client"
	"0chain.net/core/encryption"
	"bytes"
	"fmt"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func init() {
	n, err := makeTestNode("pbK")
	if err != nil {
		panic(err)
	}
	ReadConfig()
	Setup(n)
}

func Test_Simple_Setters_And_Getters(t *testing.T) {
	n := &Node{}

	ec := int64(5)
	n.SetErrorCount(ec)
	assert.Equal(t, ec, n.GetErrorCount())

	n.AddErrorCount(1)
	assert.Equal(t, n.ErrorCount, ec+1)

	n.Info = Info{}
	assert.Equal(t, n.Info, n.GetInfo())
	assert.Equal(t, &n.Info, n.GetInfoPtr())

	at := time.Now()
	n.SetLastActiveTime(at)
	assert.Equal(t, at, n.GetLastActiveTime())

	assert.Equal(t, fmt.Sprintf("http://%v:%v", n.Host, n.Port), n.GetURLBase())

	assert.Equal(t, fmt.Sprintf("http://%v:%v", n.N2NHost, n.Port), n.GetN2NURLBase())

	assert.Equal(t, fmt.Sprintf("%v/_nh/status", n.GetN2NURLBase()), n.GetStatusURL())

	assert.Equal(t, NodeTypeNames[n.Type].Code, n.GetNodeType())

	assert.Equal(t, NodeTypeNames[n.Type].Value, n.GetNodeTypeName())

	lmst := 5.0
	n.SetLargeMessageSendTime(lmst)
	assert.Equal(t, lmst, n.GetLargeMessageSendTime())
	assert.Equal(t, lmst/1000000, n.GetLargeMessageSendTimeSec())

	smst := 5.0
	n.SetSmallMessageSendTime(smst)
	assert.Equal(t, smst, n.GetSmallMessageSendTime())
	assert.Equal(t, smst/1000000, n.GetSmallMessageSendTimeSec())

	assert.Equal(t, fmt.Sprintf("%v%.3d", n.GetNodeTypeName(), n.SetIndex), n.GetPseudoName())

	info := Info{}
	n.SetInfo(info)
	assert.Equal(t, info, n.GetInfo())
}

func TestNode_Equals(t *testing.T) {
	n, err := makeTestNode("pbk")
	if err != nil {
		t.Fatal(err)
	}
	n.ID = encryption.Hash("id")

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
		ProtocolStats             interface{}
		idBytes                   []byte
		Info                      Info
	}
	type args struct {
		n2 *Node
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Equal_Keys_TRUE",
			fields: fields{
				Client:                    n.Client,
				N2NHost:                   n.N2NHost,
				Host:                      n.Host,
				Port:                      n.Port,
				Path:                      n.Path,
				Type:                      n.Type,
				Description:               n.Description,
				SetIndex:                  n.SetIndex,
				Status:                    n.Status,
				LastActiveTime:            n.LastActiveTime,
				ErrorCount:                n.ErrorCount,
				CommChannel:               n.CommChannel,
				Sent:                      n.Sent,
				SendErrors:                n.SendErrors,
				Received:                  n.Received,
				TimersByURI:               n.TimersByURI,
				SizeByURI:                 n.SizeByURI,
				largeMessageSendTime:      n.largeMessageSendTime,
				smallMessageSendTime:      n.smallMessageSendTime,
				LargeMessagePullServeTime: n.LargeMessagePullServeTime,
				SmallMessagePullServeTime: n.SmallMessagePullServeTime,
				ProtocolStats:             n.ProtocolStats,
				idBytes:                   n.idBytes,
				Info:                      n.Info,
			},
			args: args{
				n2: func() *Node {
					n2, err := makeTestNode("pbK")
					if err != nil {
						t.Fatal(err)
					}
					n2.ID = n.ID

					return n2
				}(),
			},
			want: true,
		},
		{
			name: "Equals_Port_And_Host_TRUE",
			fields: fields{
				Client:                    n.Client,
				N2NHost:                   n.N2NHost,
				Host:                      n.Host,
				Port:                      n.Port,
				Path:                      n.Path,
				Type:                      n.Type,
				Description:               n.Description,
				SetIndex:                  n.SetIndex,
				Status:                    n.Status,
				LastActiveTime:            n.LastActiveTime,
				ErrorCount:                n.ErrorCount,
				CommChannel:               n.CommChannel,
				Sent:                      n.Sent,
				SendErrors:                n.SendErrors,
				Received:                  n.Received,
				TimersByURI:               n.TimersByURI,
				SizeByURI:                 n.SizeByURI,
				largeMessageSendTime:      n.largeMessageSendTime,
				smallMessageSendTime:      n.smallMessageSendTime,
				LargeMessagePullServeTime: n.LargeMessagePullServeTime,
				SmallMessagePullServeTime: n.SmallMessagePullServeTime,
				ProtocolStats:             n.ProtocolStats,
				idBytes:                   n.idBytes,
				Info:                      n.Info,
			},
			args: args{
				n2: func() *Node {
					n2, err := makeTestNode("pbK")
					if err != nil {
						t.Fatal(err)
					}

					return n2
				}(),
			},
			want: true,
		},
		{
			name: "FALSE",
			fields: fields{
				Client:                    n.Client,
				N2NHost:                   n.N2NHost,
				Host:                      n.Host,
				Port:                      n.Port,
				Path:                      n.Path,
				Type:                      n.Type,
				Description:               n.Description,
				SetIndex:                  n.SetIndex,
				Status:                    n.Status,
				LastActiveTime:            n.LastActiveTime,
				ErrorCount:                n.ErrorCount,
				CommChannel:               n.CommChannel,
				Sent:                      n.Sent,
				SendErrors:                n.SendErrors,
				Received:                  n.Received,
				TimersByURI:               n.TimersByURI,
				SizeByURI:                 n.SizeByURI,
				largeMessageSendTime:      n.largeMessageSendTime,
				smallMessageSendTime:      n.smallMessageSendTime,
				LargeMessagePullServeTime: n.LargeMessagePullServeTime,
				SmallMessagePullServeTime: n.SmallMessagePullServeTime,
				ProtocolStats:             n.ProtocolStats,
				idBytes:                   n.idBytes,
				Info:                      n.Info,
			},
			args: args{
				n2: func() *Node {
					n2, err := makeTestNode("pbK")
					if err != nil {
						t.Fatal(err)
					}
					n2.Port = 1

					return n2
				}(),
			},
			want: false,
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
				mutex:                     sync.RWMutex{},
				mutexInfo:                 sync.RWMutex{},
				ProtocolStats:             tt.fields.ProtocolStats,
				idBytes:                   tt.fields.idBytes,
				Info:                      tt.fields.Info,
			}
			if got := n.Equals(tt.args.n2); got != tt.want {
				t.Errorf("Equals() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNode_Print(t *testing.T) {
	n, err := makeTestNode("pbk")
	if err != nil {
		t.Fatal(err)
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
	tests := []struct {
		name   string
		fields fields
		wantW  string
	}{
		{
			name: "OK",
			fields: fields{
				Client:                    n.Client,
				N2NHost:                   n.N2NHost,
				Host:                      n.Host,
				Port:                      n.Port,
				Path:                      n.Path,
				Type:                      n.Type,
				Description:               n.Description,
				SetIndex:                  n.SetIndex,
				Status:                    n.Status,
				LastActiveTime:            n.LastActiveTime,
				ErrorCount:                n.ErrorCount,
				CommChannel:               n.CommChannel,
				Sent:                      n.Sent,
				SendErrors:                n.SendErrors,
				Received:                  n.Received,
				TimersByURI:               n.TimersByURI,
				SizeByURI:                 n.SizeByURI,
				largeMessageSendTime:      n.largeMessageSendTime,
				smallMessageSendTime:      n.smallMessageSendTime,
				LargeMessagePullServeTime: n.LargeMessagePullServeTime,
				SmallMessagePullServeTime: n.SmallMessagePullServeTime,
				ProtocolStats:             n.ProtocolStats,
				idBytes:                   n.idBytes,
				Info:                      n.Info,
			},
			wantW: func() string {
				w := &bytes.Buffer{}
				_, err := fmt.Fprintf(w, "%v,%v,%v,%v,%v\n", n.GetNodeType(), n.Host, n.Port, n.GetKey(), n.PublicKey)
				if err != nil {
					t.Fatal(err)
				}

				return w.String()
			}(),
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
			w := &bytes.Buffer{}
			n.Print(w)
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("Print() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

func TestNode_IsActive(t *testing.T) {
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
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "TRUE",
			fields: fields{
				Status: NodeStatusActive,
			},
			want: true,
		},
		{
			name: "FALSE",
			fields: fields{
				Status: NodeStatusInactive,
			},
			want: false,
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
			if got := n.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNode_GetOptimalLargeMessageSendTime(t *testing.T) {
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
	tests := []struct {
		name   string
		fields fields
		want   float64
	}{
		// TODO: Add test cases.
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
			if got := n.GetOptimalLargeMessageSendTime(); got != tt.want {
				t.Errorf("GetOptimalLargeMessageSendTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNode_GetOptimalLargeMessageSendTime1(t *testing.T) {
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
	tests := []struct {
		name   string
		fields fields
		want   float64
	}{}
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
			if got := n.GetOptimalLargeMessageSendTime(); got != tt.want {
				t.Errorf("GetOptimalLargeMessageSendTime() = %v, want %v", got, tt.want)
			}
		})
	}
}
