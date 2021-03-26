package node

import (
	"0chain.net/chaincore/client"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"bytes"
	"context"
	"fmt"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

func init() {
	common.ConfigRateLimits()
	SetupHandlers()

	logging.InitLogging("development")
}

func TestWhoAmIHandler(t *testing.T) {
	self := newSelfNode()

	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name  string
		args  args
		self  *SelfNode
		wantW http.ResponseWriter
	}{
		{
			name: "Self_Nil_OK",
			args: args{
				w: httptest.NewRecorder(),
			},
			wantW: httptest.NewRecorder(),
		},
		{
			name: "OK",
			args: args{
				w: httptest.NewRecorder(),
			},
			self: self,
			wantW: func() http.ResponseWriter {
				w := httptest.NewRecorder()
				self.Underlying().Print(w)

				return w
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Self = tt.self

			WhoAmIHandler(tt.args.w, tt.args.r)
			assert.Equal(t, tt.wantW, tt.args.w)
		})
	}
}

func TestNode_PrintSendStats(t *testing.T) {
	n, err := makeTestNode("pbK")
	if err != nil {
		t.Fatal(err)
	}

	uri := "uri 1"
	timer := metrics.NewTimer()
	timer.Update(time.Second)
	timer1 := metrics.NewTimer()
	n.TimersByURI = map[string]metrics.Timer{
		uri:     timer,
		"uri 2": timer1,
	}
	n.SizeByURI = map[string]metrics.Histogram{
		uri: metrics.NewHistogram(metrics.NewUniformSample(1)),
	}

	wantW := "<tr>" +
		fmt.Sprintf("<td>%v</td>", uri) +
		fmt.Sprintf("<td class='number'>%9d</td>", timer.Count()) +
		fmt.Sprintf("<td class='number'>%.2f</td>", scale(timer.Min())) +
		fmt.Sprintf("<td class='number'>%.2f &plusmn;%.2f</td>", timer.Mean()/1000000., timer.StdDev()/1000000.) +
		fmt.Sprintf("<td class='number'>%.2f</td>", scale(timer.Max()))
	sizer := metrics.NewHistogram(metrics.NewUniformSample(256))
	wantW += fmt.Sprintf("<td class='number'>%d</td>", sizer.Min()) +
		fmt.Sprintf("<td class='number'>%.2f &plusmn;%.2f</td>", sizer.Mean(), sizer.StdDev()) +
		fmt.Sprintf("<td class='number'>%d</td>", sizer.Max()) +
		"</tr>"

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
			wantW: wantW,
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
			w := &bytes.Buffer{}
			n.PrintSendStats(w)
			assert.Equal(t, tt.wantW, w.String())
		})
	}
}

func TestStatusHandler(t *testing.T) {
	Self = newSelfNode()

	pbK, prK, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}

	n, err := makeTestNode(pbK)
	if err != nil {
		t.Fatal(err)
	}
	n.ID = encryption.Hash("id")
	n.Status = NodeStatusInactive
	RegisterNode(n)

	activeN, err := makeTestNode("pbK")
	if err != nil {
		t.Fatal(err)
	}
	activeN.Status = NodeStatusActive
	activeN.ID = encryption.Hash("active node id")
	RegisterNode(activeN)

	client.SetClientSignatureScheme("ed25519")

	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name  string
		args  args
		wantW http.ResponseWriter
	}{
		{
			name: "Nil_ID_OK",
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodGet, "/", nil),
			},
			wantW: httptest.NewRecorder(),
		},
		{
			name: "Nil_Node_OK",
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					r.Form = map[string][]string{
						"id": {
							"id",
						},
					}

					return r
				}(),
			},
			wantW: httptest.NewRecorder(),
		},
		{
			name: "Active_Node_OK",
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					r.Form = map[string][]string{
						"id": {
							activeN.ID,
						},
					}

					return r
				}(),
			},
			wantW: func() http.ResponseWriter {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				w := httptest.NewRecorder()
				common.Respond(w, r, Self.Underlying().Info, nil)

				return w
			}(),
		},
		{
			name: "Empty_Hash_And_Data_And_Signature_OK",
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					r.Form = map[string][]string{
						"id": {
							n.ID,
						},
					}

					return r
				}(),
			},
			wantW: httptest.NewRecorder(),
		},
		{
			name: "Invalid_Data_OK",
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					r.Form = map[string][]string{
						"id": {
							n.ID,
						},
						"data": {
							":",
						},
						"hash": {
							"hash",
						},
						"signature": {
							"signature",
						},
					}

					return r
				}(),
			},
			wantW: httptest.NewRecorder(),
		},
		{
			name: "Failed_Validate_Time_OK",
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					ts := strconv.Itoa(int(time.Now().Unix()))

					r := httptest.NewRequest(http.MethodGet, "/", nil)
					r.Form = map[string][]string{
						"id": {
							n.ID,
						},
						"data": {
							"1:" + ts + ":1",
						},
						"hash": {
							"hash",
						},
						"signature": {
							"signature",
						},
					}

					return r
				}(),
			},
			wantW: httptest.NewRecorder(),
		},
		{
			name: "Signature_Failed_OK",
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					ts := strconv.Itoa(int(time.Now().Unix()))

					r := httptest.NewRequest(http.MethodGet, "/", nil)
					r.Form = map[string][]string{
						"id": {
							n.ID,
						},
						"data": {
							"1:" + ts + ":1",
						},
						"hash": {
							"hash",
						},
						"signature": {
							"signature",
						},
					}

					return r
				}(),
			},
			wantW: httptest.NewRecorder(),
		},
		{
			name: "Signature_Failed_OK",
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					ts := strconv.Itoa(int(time.Now().Unix()))

					hash := encryption.Hash("data")
					sign, err := encryption.Sign(prK, hash)
					if err != nil {
						t.Fatal(err)
					}

					r := httptest.NewRequest(http.MethodGet, "/", nil)
					r.Form = map[string][]string{
						"id": {
							n.ID,
						},
						"data": {
							"1:" + ts + ":1",
						},
						"hash": {
							hash,
						},
						"signature": {
							sign,
						},
					}

					return r
				}(),
			},
			wantW: func() http.ResponseWriter {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				w := httptest.NewRecorder()
				common.Respond(w, r, Self.Underlying().Info, nil)

				return w
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			StatusHandler(tt.args.w, tt.args.r)
			assert.Equal(t, tt.wantW, tt.args.w)
		})
	}
}

func TestGetPoolMembersHandler(t *testing.T) {
	nodes = make(map[string]*Node)

	mn, err := makeTestNode("pbK")
	if err != nil {
		t.Fatal(err)
	}
	mn.Type = NodeTypeMiner
	mn.ID = encryption.Hash("miner id")
	RegisterNode(mn)

	sn, err := makeTestNode("pbK")
	if err != nil {
		t.Fatal(err)
	}
	sn.Type = NodeTypeSharder
	sn.ID = encryption.Hash("sharder id")
	RegisterNode(sn)

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
			name: "OK",
			want: &PoolMembers{
				Miners: []string{
					mn.GetN2NURLBase(),
				},
				Sharders: []string{
					sn.GetN2NURLBase(),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetPoolMembersHandler(tt.args.ctx, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPoolMembersHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPoolMembersHandler() got = %v, want %v", got, tt.want)
			}
		})
	}
}
