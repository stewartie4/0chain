package client

import (
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/core/memorystore"
	mocks "0chain.net/mocks/core/datastore"
	"context"
	"encoding/hex"
	"errors"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func init() {
	logging.InitLogging("testing")

	clientEntityMetadata = datastore.MetadataProvider()
	clientEntityMetadata.Name = "client"
	clientEntityMetadata.Provider = Provider
	clientEntityMetadata.Store = memorystore.GetStorageProvider()
	datastore.RegisterEntityMetadata("client", clientEntityMetadata)

	common.ConfigRateLimits()
	SetupHandlers()
}

func TestSetClientSignatureScheme(t *testing.T) {
	type args struct {
		scheme string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "OK",
			args: args{scheme: "ed25519"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetClientSignatureScheme(tt.args.scheme)
			if clientSignatureScheme != tt.args.scheme {
				t.Errorf("SetClientSignatureScheme() got = %v, want = %v", clientSignatureScheme, tt.args.scheme)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name string
		want *Client
	}{
		{
			name: "OK",
			want: datastore.GetEntityMetadata("client").Instance().(*Client),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewClient(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_Copy(t *testing.T) {
	cl := NewClient()
	cl.CollectionMemberField.CollectionScore = 123
	cl.CollectionMemberField.EntityCollection = &datastore.EntityCollection{CollectionName: "collection name"}
	cl.IDField.ID = "id"
	cl.VersionField.Version = "1"
	cl.CreationDateField.CreationDate = common.Now()
	cl.PublicKey = "public key"
	cl.PublicKeyBytes = []byte(cl.PublicKey)

	type fields struct {
		CollectionMemberField datastore.CollectionMemberField
		IDField               datastore.IDField
		VersionField          datastore.VersionField
		CreationDateField     datastore.CreationDateField
		PublicKey             string
		PublicKeyBytes        []byte
	}
	tests := []struct {
		name   string
		fields fields
		wantCp *Client
	}{
		{
			name: "OK",
			fields: fields{
				CollectionMemberField: cl.CollectionMemberField,
				IDField:               cl.IDField,
				VersionField:          cl.VersionField,
				CreationDateField:     cl.CreationDateField,
				PublicKey:             cl.PublicKey,
				PublicKeyBytes:        cl.PublicKeyBytes,
			},
			wantCp: cl,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				CollectionMemberField: tt.fields.CollectionMemberField,
				IDField:               tt.fields.IDField,
				VersionField:          tt.fields.VersionField,
				CreationDateField:     tt.fields.CreationDateField,
				PublicKey:             tt.fields.PublicKey,
				PublicKeyBytes:        tt.fields.PublicKeyBytes,
			}
			if gotCp := c.Copy(); !reflect.DeepEqual(gotCp, tt.wantCp) {
				t.Errorf("Copy() = %v, want %v", gotCp, tt.wantCp)
			}
		})
	}
}

func TestClient_Validate(t *testing.T) {
	pbK, _, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	cl := NewClient()
	cl.IDField.ID = encryption.Hash(pbK)
	cl.PublicKeyBytes = []byte(pbK)

	type fields struct {
		CollectionMemberField datastore.CollectionMemberField
		IDField               datastore.IDField
		VersionField          datastore.VersionField
		CreationDateField     datastore.CreationDateField
		PublicKey             string
		PublicKeyBytes        []byte
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
			name:    "Empty_ID_ERR",
			wantErr: true,
		},
		{
			name:    "Different_ID_And_Public_Key_ERR",
			fields:  fields{IDField: cl.IDField},
			wantErr: true,
		},
		{
			name:    "OK",
			fields:  fields{IDField: cl.IDField, PublicKeyBytes: cl.PublicKeyBytes},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				CollectionMemberField: tt.fields.CollectionMemberField,
				IDField:               tt.fields.IDField,
				VersionField:          tt.fields.VersionField,
				CreationDateField:     tt.fields.CreationDateField,
				PublicKey:             tt.fields.PublicKey,
				PublicKeyBytes:        tt.fields.PublicKeyBytes,
			}
			if err := c.Validate(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_Read(t *testing.T) {
	store := mocks.Store{}
	store.On("Read", context.Context(nil), "", new(Client)).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return nil
		},
	)

	clientEntityMetadata.Store = &store

	type fields struct {
		CollectionMemberField datastore.CollectionMemberField
		IDField               datastore.IDField
		VersionField          datastore.VersionField
		CreationDateField     datastore.CreationDateField
		PublicKey             string
		PublicKeyBytes        []byte
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
			c := &Client{
				CollectionMemberField: tt.fields.CollectionMemberField,
				IDField:               tt.fields.IDField,
				VersionField:          tt.fields.VersionField,
				CreationDateField:     tt.fields.CreationDateField,
				PublicKey:             tt.fields.PublicKey,
				PublicKeyBytes:        tt.fields.PublicKeyBytes,
			}
			if err := c.Read(tt.args.ctx, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_Write(t *testing.T) {
	store := mocks.Store{}
	store.On("Write", context.Context(nil), new(Client)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	clientEntityMetadata.Store = &store

	type fields struct {
		CollectionMemberField datastore.CollectionMemberField
		IDField               datastore.IDField
		VersionField          datastore.VersionField
		CreationDateField     datastore.CreationDateField
		PublicKey             string
		PublicKeyBytes        []byte
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
			c := &Client{
				CollectionMemberField: tt.fields.CollectionMemberField,
				IDField:               tt.fields.IDField,
				VersionField:          tt.fields.VersionField,
				CreationDateField:     tt.fields.CreationDateField,
				PublicKey:             tt.fields.PublicKey,
				PublicKeyBytes:        tt.fields.PublicKeyBytes,
			}
			if err := c.Write(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_Delete(t *testing.T) {
	store := mocks.Store{}
	store.On("Delete", context.Context(nil), new(Client)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	clientEntityMetadata.Store = &store

	type fields struct {
		CollectionMemberField datastore.CollectionMemberField
		IDField               datastore.IDField
		VersionField          datastore.VersionField
		CreationDateField     datastore.CreationDateField
		PublicKey             string
		PublicKeyBytes        []byte
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
			c := &Client{
				CollectionMemberField: tt.fields.CollectionMemberField,
				IDField:               tt.fields.IDField,
				VersionField:          tt.fields.VersionField,
				CreationDateField:     tt.fields.CreationDateField,
				PublicKey:             tt.fields.PublicKey,
				PublicKeyBytes:        tt.fields.PublicKeyBytes,
			}
			if err := c.Delete(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_Verify(t *testing.T) {
	SetClientSignatureScheme("ed25519")

	pbK, prK, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	data := encryption.Hash("data")
	sign, err := encryption.Sign(prK, data)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		CollectionMemberField datastore.CollectionMemberField
		IDField               datastore.IDField
		VersionField          datastore.VersionField
		CreationDateField     datastore.CreationDateField
		PublicKey             string
		PublicKeyBytes        []byte
	}
	type args struct {
		signature string
		hash      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:   "TRUE",
			fields: fields{PublicKey: pbK},
			args:   args{hash: data, signature: sign},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				CollectionMemberField: tt.fields.CollectionMemberField,
				IDField:               tt.fields.IDField,
				VersionField:          tt.fields.VersionField,
				CreationDateField:     tt.fields.CreationDateField,
				PublicKey:             tt.fields.PublicKey,
				PublicKeyBytes:        tt.fields.PublicKeyBytes,
			}
			got, err := c.Verify(tt.args.signature, tt.args.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("Verify() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Verify() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_ComputeProperties(t *testing.T) {
	cl := NewClient()
	cl.EntityCollection = cliEntityCollection
	cl.SetPublicKey(hex.EncodeToString(cl.PublicKeyBytes))

	type fields struct {
		CollectionMemberField datastore.CollectionMemberField
		IDField               datastore.IDField
		VersionField          datastore.VersionField
		CreationDateField     datastore.CreationDateField
		PublicKey             string
		PublicKeyBytes        []byte
	}
	tests := []struct {
		name   string
		fields fields
		want   *Client
	}{
		{
			name: "OK",
			fields: fields{
				IDField:           cl.IDField,
				VersionField:      cl.VersionField,
				CreationDateField: cl.CreationDateField,
				PublicKey:         cl.PublicKey,
			},
			want: cl,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				CollectionMemberField: tt.fields.CollectionMemberField,
				IDField:               tt.fields.IDField,
				VersionField:          tt.fields.VersionField,
				CreationDateField:     tt.fields.CreationDateField,
				PublicKey:             tt.fields.PublicKey,
				PublicKeyBytes:        tt.fields.PublicKeyBytes,
			}

			c.ComputeProperties()
			if !assert.Equal(t, c, tt.want) {
				t.Errorf("ComputeProperties() got = %v, want = %v", c, tt.want)
			}
		})
	}
}

func TestGetClients(t *testing.T) {
	clientKey := "cl key"

	clientIDs := make([]string, 0, 1)
	cEntities := make([]datastore.Entity, 0, 1)
	clientIDs = append(clientIDs, clientKey)
	cEntities = append(cEntities, clientEntityMetadata.Instance().(*Client))

	store := mocks.Store{}
	store.On("MultiRead", context.Context(nil), clientEntityMetadata, clientIDs, cEntities).Return(
		func(_ context.Context, entityMetadata datastore.EntityMetadata, keys []datastore.Key, entities []datastore.Entity) error {
			return nil
		},
	)
	store.On("MultiRead", context.TODO(), clientEntityMetadata, clientIDs, cEntities).Return(
		func(ctx context.Context, entityMetadata datastore.EntityMetadata, keys []datastore.Key, entities []datastore.Entity) error {
			return errors.New("")
		},
	)

	clientEntityMetadata.Store = &store

	type args struct {
		ctx     context.Context
		clients map[string]*Client
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Multi_Read_ERR",
			args: args{
				ctx: context.TODO(),
				clients: map[string]*Client{
					clientKey: nil,
				},
			},
			wantErr: true,
		},
		{
			name: "OK",
			args: args{
				clients: map[string]*Client{
					clientKey: nil,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := GetClients(tt.args.ctx, tt.args.clients); (err != nil) != tt.wantErr {
				t.Errorf("GetClients() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetClient(t *testing.T) {
	cacheCl := NewClient()
	cacheCl.ID = "cache cl id"
	if err := cacher.Add(cacheCl.ID, cacheCl); err != nil {
		t.Fatal(err)
	}

	store := mocks.Store{}
	store.On("Read", context.Context(nil), "", NewClient()).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return errors.New("")
		},
	)
	store.On("Read", context.TODO(), "", NewClient()).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return nil
		},
	)

	clientEntityMetadata.Store = &store

	type args struct {
		ctx context.Context
		key datastore.Key
	}
	tests := []struct {
		name    string
		args    args
		want    *Client
		wantErr bool
	}{
		{
			name: "From_Cache_OK",
			args: args{key: cacheCl.ID},
			want: cacheCl,
		},
		{
			name:    "Read_ERR",
			wantErr: true,
		},
		{
			name:    "Read_OK",
			args:    args{ctx: context.TODO()},
			want:    NewClient(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetClient(tt.args.ctx, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetClient() got = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestPutClient(t *testing.T) {
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
			name:    "Not_A_Client_ERR",
			args:    args{entity: &mocks.Entity{}},
			wantErr: true,
		},
		{
			name:    "Put_Handler_ERR",
			args:    args{entity: NewClient()},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PutClient(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("PutClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PutClient() got = %v, want %v", got, tt.want)
			}
		})
	}
}
