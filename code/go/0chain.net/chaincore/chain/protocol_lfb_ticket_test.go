package chain

import (
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"fmt"
	"reflect"
	"testing"
)

func TestLFBTicketEntityMetadata_GetName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "OK",
			want: "lfb_ticket",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lfbtem := &LFBTicketEntityMetadata{}
			if got := lfbtem.GetName(); got != tt.want {
				t.Errorf("GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLFBTicketEntityMetadata_GetDB(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "OK",
			want: "lfb_ticket.db",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lfbtem := &LFBTicketEntityMetadata{}
			if got := lfbtem.GetDB(); got != tt.want {
				t.Errorf("GetDB() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLFBTicketEntityMetadata_Instance(t *testing.T) {
	tests := []struct {
		name string
		want datastore.Entity
	}{
		{
			name: "OK",
			want: new(LFBTicket),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lfbtem := &LFBTicketEntityMetadata{}
			if got := lfbtem.Instance(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Instance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLFBTicketEntityMetadata_GetStore(t *testing.T) {
	tests := []struct {
		name string
		want datastore.Store
	}{
		{
			name: "OK",
			want: nil, // not implemented
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lfbtem := &LFBTicketEntityMetadata{}
			if got := lfbtem.GetStore(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetStore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLFBTicketEntityMetadata_GetIDColumnName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "OK",
			want: "", // not implemented
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lfbtem := &LFBTicketEntityMetadata{}
			if got := lfbtem.GetIDColumnName(); got != tt.want {
				t.Errorf("GetIDColumnName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLFBTicket_Hash(t *testing.T) {
	lfbt := LFBTicket{
		Round:     5,
		SharderID: "sharder id",
		LFBHash:   "lfb hash",
	}
	data := fmt.Sprintf("%d:%s:%s", lfbt.Round, lfbt.SharderID, lfbt.LFBHash)

	type fields struct {
		Round     int64
		SharderID string
		LFBHash   string
		Sign      string
		Senders   []string
		IsOwn     bool
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "OK",
			fields: fields{
				Round:     lfbt.Round,
				SharderID: lfbt.SharderID,
				LFBHash:   lfbt.LFBHash,
			},
			want: encryption.Hash(data),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lfbt := &LFBTicket{
				Round:     tt.fields.Round,
				SharderID: tt.fields.SharderID,
				LFBHash:   tt.fields.LFBHash,
				Sign:      tt.fields.Sign,
				Senders:   tt.fields.Senders,
				IsOwn:     tt.fields.IsOwn,
			}
			if got := lfbt.Hash(); got != tt.want {
				t.Errorf("Hash() = %v, want %v", got, tt.want)
			}
		})
	}
}
