package state

import (
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"0chain.net/core/util"
	mocks "0chain.net/mocks/core/datastore"
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"reflect"
	"strconv"
	"testing"
)

func init() {
	SetupPartialState(memorystore.GetStorageProvider())
	logging.InitLogging("testing")
}

func TestNewPartialState(t *testing.T) {
	var (
		key util.Key = []byte("key")
		ps           = datastore.GetEntityMetadata("partial_state").Instance().(*PartialState)
	)
	ps.Hash = key
	ps.ComputeProperties()

	type args struct {
		key util.Key
	}
	tests := []struct {
		name string
		args args
		want *PartialState
	}{
		{
			name: "OK",
			args: args{key: key},
			want: ps,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewPartialState(tt.args.key); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPartialState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPartialState_GetKey(t *testing.T) {
	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
		mndb    *util.MemoryNodeDB
		root    util.Node
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.Key
	}{
		{
			name: "Hex_Key_OK",
			want: "!key!",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
				mndb:    tt.fields.mndb,
				root:    tt.fields.root,
			}

			ps.SetKey(tt.want)
			tt.want = datastore.ToKey(ps.Hash)
			if got := ps.GetKey(); got != tt.want {
				t.Errorf("GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPartialState_Read(t *testing.T) {
	store := mocks.Store{}
	store.On("Read", context.Context(nil), "", mock.AnythingOfType("*state.PartialState")).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return nil
		},
	)

	partialStateEntityMetadata.Store = &store

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
		mndb    *util.MemoryNodeDB
		root    util.Node
	}
	type args struct {
		ctx context.Context
		key datastore.Key
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
				mndb:    tt.fields.mndb,
				root:    tt.fields.root,
			}
			if err := ps.Read(tt.args.ctx, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPartialState_GetScore(t *testing.T) {
	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
		mndb    *util.MemoryNodeDB
		root    util.Node
	}
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		{
			name: "OK",
			want: 0, // not implemented
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
				mndb:    tt.fields.mndb,
				root:    tt.fields.root,
			}
			if got := ps.GetScore(); got != tt.want {
				t.Errorf("GetScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPartialState_Write(t *testing.T) {
	store := mocks.Store{}
	store.On("Write", context.Context(nil), mock.AnythingOfType("*state.PartialState")).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	partialStateEntityMetadata.Store = &store

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
		mndb    *util.MemoryNodeDB
		root    util.Node
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
				mndb:    tt.fields.mndb,
				root:    tt.fields.root,
			}
			if err := ps.Write(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPartialState_Delete(t *testing.T) {
	store := mocks.Store{}
	store.On("Delete", context.Context(nil), mock.AnythingOfType("*state.PartialState")).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	partialStateEntityMetadata.Store = &store

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
		mndb    *util.MemoryNodeDB
		root    util.Node
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
				mndb:    tt.fields.mndb,
				root:    tt.fields.root,
			}
			if err := ps.Delete(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPartialState_GetRoot(t *testing.T) {
	root := util.NewValueNode()

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
		mndb    *util.MemoryNodeDB
		root    util.Node
	}
	tests := []struct {
		name   string
		fields fields
		want   util.Node
	}{
		{
			name:   "OK",
			fields: fields{root: root},
			want:   root,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
				mndb:    tt.fields.mndb,
				root:    tt.fields.root,
			}
			if got := ps.GetRoot(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRoot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPartialState_GetNodeDB(t *testing.T) {
	db := util.NewMemoryNodeDB()

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
		mndb    *util.MemoryNodeDB
		root    util.Node
	}
	tests := []struct {
		name   string
		fields fields
		want   util.NodeDB
	}{
		{
			name:   "OK",
			fields: fields{mndb: db},
			want:   db,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
				mndb:    tt.fields.mndb,
				root:    tt.fields.root,
			}
			if got := ps.GetNodeDB(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetNodeDB() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPartialState_UnmarshalJSON(t *testing.T) {
	ps := NewPartialState([]byte("key"))
	ps.Nodes = []util.Node{}
	blob, err := json.Marshal(ps)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
		mndb    *util.MemoryNodeDB
		root    util.Node
	}
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    *PartialState
	}{
		{
			name:    "OK",
			args:    args{data: blob},
			want:    ps,
			wantErr: false,
		},
		{
			name:    "ERR",
			args:    args{data: []byte("}{")},
			want:    ps,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
				mndb:    tt.fields.mndb,
				root:    tt.fields.root,
			}
			if err := ps.UnmarshalJSON(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, ps)
			}
		})
	}
}

func TestPartialState_UnmarshalPartialState(t *testing.T) {
	ps := PartialState{
		Version: "1",
	}
	ps.SetKey(encryption.Hash("data"))

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
		mndb    *util.MemoryNodeDB
		root    util.Node
	}
	type args struct {
		obj map[string]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Invalid_Root_ERR",
			args: args{
				obj: map[string]interface{}{
					"root": 124,
				},
			},
			wantErr: true,
		},
		{
			name: "No_Root_ERR",
			args: args{
				obj: map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "No_Version_ERR",
			args: args{
				obj: map[string]interface{}{
					"root": ps.Hash,
				},
			},
			wantErr: true,
		},
		{
			name: "No_Nodes_ERR",
			args: args{
				obj: map[string]interface{}{
					"root":    ps.Hash,
					"version": ps.Version,
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid_Nodes_ERR",
			args: args{
				obj: map[string]interface{}{
					"root":    ps.Hash,
					"version": ps.Version,
					"nodes": []interface{}{
						1,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Node_Decoding_ERR",
			args: args{
				obj: map[string]interface{}{
					"root":    []byte("root"),
					"version": ps.Version,
					"nodes": []interface{}{
						"!",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "OK",
			args: args{
				obj: map[string]interface{}{
					"root":    ps.Hash,
					"version": ps.Version,
					"nodes": []interface{}{
						base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(util.NodeTypeValueNode) + "node")),
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
				mndb:    tt.fields.mndb,
				root:    tt.fields.root,
			}
			if err := ps.UnmarshalPartialState(tt.args.obj); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalPartialState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPartialState_MarshalJSON(t *testing.T) {
	ps := PartialState{
		Version: "1",
		Nodes: []util.Node{
			util.NewValueNode(),
		},
	}
	ps.SetKey(encryption.Hash("data"))

	mapPS := map[string]interface{}{
		"root":    util.ToHex(ps.Hash),
		"version": ps.Version,
		"nodes": [][]byte{
			ps.Nodes[0].Encode(),
		},
	}

	blob, err := json.Marshal(mapPS)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
		mndb    *util.MemoryNodeDB
		root    util.Node
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			name:    "OK",
			fields:  fields(ps),
			want:    blob,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
				mndb:    tt.fields.mndb,
				root:    tt.fields.root,
			}

			got, err := ps.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPartialState_AddNode(t *testing.T) {
	nodes := []util.Node{
		util.NewValueNode(),
	}
	node := util.NewValueNode()

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
		mndb    *util.MemoryNodeDB
		root    util.Node
	}
	type args struct {
		node util.Node
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *PartialState
	}{
		{
			name: "OK",
			fields: fields{
				Nodes: nodes,
			},
			args: args{node: node},
			want: &PartialState{
				Nodes: append(nodes, node),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
				mndb:    tt.fields.mndb,
				root:    tt.fields.root,
			}

			ps.AddNode(tt.args.node)
			assert.Equal(t, tt.want, ps)
		})
	}
}

func TestPartialState_SaveState(t *testing.T) {
	ps := NewPartialState(util.Key("key"))
	ps.mndb = util.NewMemoryNodeDB()
	db := util.NewMemoryNodeDB()
	db.PutNode(util.Key("node key"), util.NewFullNode(&util.SecureSerializableValue{Buffer: []byte("data")}))

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
		mndb    *util.MemoryNodeDB
		root    util.Node
	}
	type args struct {
		ctx     context.Context
		stateDB util.NodeDB
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    *PartialState
	}{
		{
			name:    "OK",
			fields:  fields(*ps),
			args:    args{stateDB: db},
			wantErr: false,
			want: func() *PartialState {
				mndb := *ps.mndb
				util.MergeState(nil, &mndb, db)

				ps := *ps
				ps.mndb = &mndb
				return &ps
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
				mndb:    tt.fields.mndb,
				root:    tt.fields.root,
			}
			if err := ps.SaveState(tt.args.ctx, tt.args.stateDB); (err != nil) != tt.wantErr {
				t.Errorf("SaveState() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.want, ps)
		})
	}
}

func TestPartialState_ComputeProperties(t *testing.T) {
	ps := PartialState{
		Nodes: []util.Node{
			util.NewFullNode(&util.SecureSerializableValue{[]byte("value")}),
		},
	}
	ps.mndb = ps.newNodeDB()
	ps.root = ps.mndb.ComputeRoot()
	ps.Hash = ps.root.GetHashBytes()

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
		mndb    *util.MemoryNodeDB
		root    util.Node
	}
	tests := []struct {
		name   string
		fields fields
		want   *PartialState
	}{
		{
			name: "OK",
			fields: fields{
				Hash:  ps.Hash,
				Nodes: ps.Nodes,
			},
			want: &ps,
		},
		{
			name: "OK2",
			fields: fields{
				Nodes: ps.Nodes,
			},
			want: &PartialState{
				Nodes: ps.Nodes,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
				mndb:    tt.fields.mndb,
				root:    tt.fields.root,
			}

			ps.ComputeProperties()
			assert.Equal(t, tt.want, ps)
		})
	}
}

func TestPartialState_Validate(t *testing.T) {
	ps := PartialState{
		Nodes: []util.Node{
			util.NewFullNode(&util.SecureSerializableValue{[]byte("value")}),
		},
	}
	ps.mndb = ps.newNodeDB()
	ps.root = ps.mndb.ComputeRoot()

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
		mndb    *util.MemoryNodeDB
		root    util.Node
	}
	type args struct {
		ctx context.Context
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
				Hash:    ps.Hash,
				Version: ps.Version,
				Nodes:   ps.Nodes,
				mndb:    ps.mndb,
				root:    ps.root,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
				mndb:    tt.fields.mndb,
				root:    tt.fields.root,
			}
			if err := ps.Validate(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
