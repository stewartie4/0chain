package httpclientutil

import (
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
)

func init() {
	logging.InitLogging("development")
}

func TestTransaction_ComputeHashAndSign(t1 *testing.T) {
	txn := NewTransactionEntity("id", "chainID", "public key")
	txn.CreationDate = 0

	_, prK, err := encryption.GenerateKeys()
	if err != nil {
		t1.Fatal(err)
	}
	handler := func(h string) (string, error) {
		return encryption.Sign(prK, h)
	}

	want := *txn
	hashdata := fmt.Sprintf("%v:%v:%v:%v:%v", want.CreationDate, want.ClientID,
		want.ToClientID, want.Value, encryption.Hash(want.TransactionData))
	want.Hash = encryption.Hash(hashdata)
	want.Signature, err = handler(want.Hash)
	if err != nil {
		t1.Fatal(err)
	}

	type fields struct {
		Hash              string
		Version           string
		ClientID          string
		PublicKey         string
		ToClientID        string
		ChainID           string
		TransactionData   string
		Value             int64
		Signature         string
		CreationDate      common.Timestamp
		Fee               int64
		TransactionType   int
		TransactionOutput string
		OutputHash        string
	}
	type args struct {
		handler Signer
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    *Transaction
	}{
		{
			name:    "OK",
			fields:  fields(*txn),
			args:    args{handler: handler},
			wantErr: false,
			want:    &want,
		},
		{
			name:   "ERR",
			fields: fields(*txn),
			args: args{
				handler: func(h string) (string, error) {
					return "", errors.New("")
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Transaction{
				Hash:              tt.fields.Hash,
				Version:           tt.fields.Version,
				ClientID:          tt.fields.ClientID,
				PublicKey:         tt.fields.PublicKey,
				ToClientID:        tt.fields.ToClientID,
				ChainID:           tt.fields.ChainID,
				TransactionData:   tt.fields.TransactionData,
				Value:             tt.fields.Value,
				Signature:         tt.fields.Signature,
				CreationDate:      tt.fields.CreationDate,
				Fee:               tt.fields.Fee,
				TransactionType:   tt.fields.TransactionType,
				TransactionOutput: tt.fields.TransactionOutput,
				OutputHash:        tt.fields.OutputHash,
			}
			if err := t.ComputeHashAndSign(tt.args.handler); (err != nil) != tt.wantErr {
				t1.Errorf("ComputeHashAndSign() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				assert.Equal(t1, tt.want, t)
			}
		})
	}
}

func TestNewHTTPRequest(t *testing.T) {
	var (
		url  = "/"
		data = []byte("data")
		id   = "id"
		pKey = "pkey"
	)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Access-Control-Allow-Origin", "*")
	req.Header.Set("X-App-Client-ID", id)
	req.Header.Set("X-App-Client-Key", pKey)

	type args struct {
		method string
		url    string
		data   []byte
		ID     string
		pkey   string
	}
	tests := []struct {
		name    string
		args    args
		want    *http.Request
		wantErr bool
	}{
		{
			name: "OK",
			args: args{
				method: "",
				url:    url,
				data:   data,
				ID:     id,
				pkey:   pKey,
			},
			want:    req,
			wantErr: false,
		},
		{
			name: "ERR",
			args: args{
				url: string(rune(0x7f)),
			},
			want:    req,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewHTTPRequest(tt.args.method, tt.args.url, tt.args.data, tt.args.ID, tt.args.pkey)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHTTPRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil && !tt.wantErr {
				assert.Equal(t, tt.want.URL, got.URL)
				tt.want.URL = nil
				got.URL = nil

				assert.Equal(t, tt.want.Body, got.Body)
			}
		})
	}
}

func TestSendPostRequest(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
			},
		),
	)
	defer server.Close()

	type args struct {
		url  string
		data []byte
		ID   string
		pkey string
		wg   *sync.WaitGroup
	}
	tests := []struct {
		name    string
		args    args
		client  *http.Client
		want    []byte
		wantErr bool
	}{
		{
			name: "Request_Creating_ERR",
			args: args{
				url: string(rune(0x7f)),
			},
			wantErr: true,
		},
		{
			name:   "Client_ERR",
			client: server.Client(),
			args: args{
				url: "/",
			},
			wantErr: true,
		},
		{
			name:   "OK",
			client: server.Client(),
			args: args{
				url: server.URL,
			},
			want:    make([]byte, 0, 512),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient = tt.client

			got, err := SendPostRequest(tt.args.url, tt.args.data, tt.args.ID, tt.args.pkey, tt.args.wg)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendPostRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SendPostRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSendMultiPostRequest(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
			},
		),
	)
	defer server.Close()

	type args struct {
		urls []string
		data []byte
		ID   string
		pkey string
	}
	tests := []struct {
		name   string
		client *http.Client
		args   args
	}{
		{
			name:   "OK",
			client: server.Client(),
			args: args{
				urls: []string{
					server.URL,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient = tt.client

			SendMultiPostRequest(tt.args.urls, tt.args.data, tt.args.ID, tt.args.pkey)
		})
	}
}

func TestSendTransaction(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
			},
		),
	)
	defer server.Close()

	type args struct {
		txn  *Transaction
		urls []string
		ID   string
		pkey string
	}
	tests := []struct {
		name   string
		args   args
		client *http.Client
	}{
		{
			name:   "OK",
			client: server.Client(),
			args: args{
				urls: []string{
					server.URL,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient = tt.client

			SendTransaction(tt.args.txn, tt.args.urls, tt.args.ID, tt.args.pkey)
		})
	}
}

func TestMakeGetRequest(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				data := map[string]interface{}{
					"key": "value",
				}
				blob, err := json.Marshal(data)
				if err != nil {
					t.Fatal(err)
				}

				if _, err := rw.Write(blob); err != nil {
					t.Fatal(err)
				}
			},
		),
	)
	defer server.Close()

	errServer := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				rw.WriteHeader(http.StatusInternalServerError)
			},
		),
	)
	defer server.Close()

	type args struct {
		remoteUrl string
		result    interface{}
	}
	tests := []struct {
		name    string
		args    args
		client  *http.Client
		wantErr bool
	}{
		{
			name: "Request_Creating_ERR",
			args: args{
				remoteUrl: string(rune(0x7f)),
			},
			wantErr: true,
		},
		{
			name:   "Client_ERR",
			client: server.Client(),
			args: args{
				remoteUrl: "/",
			},
			wantErr: true,
		},
		{
			name:   "Resp_Status_Not_Ok_ERR",
			client: errServer.Client(),
			args: args{
				remoteUrl: errServer.URL,
			},
			wantErr: true,
		},
		{
			name:   "JSON_Decoding_ERR",
			client: server.Client(),
			args: args{
				remoteUrl: server.URL,
				result:    "}{",
			},
			wantErr: true,
		},
		{
			name:   "OK",
			client: server.Client(),
			args: args{
				remoteUrl: server.URL,
				result:    &map[string]interface{}{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient = tt.client

			if err := MakeGetRequest(tt.args.remoteUrl, tt.args.result); (err != nil) != tt.wantErr {
				t.Errorf("MakeGetRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMakeClientBalanceRequest(t *testing.T) {
	balance := state.Balance(5)
	server := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				st := &state.State{
					Balance: balance,
				}
				blob, err := json.Marshal(st)
				if err != nil {
					t.Fatal(err)
				}

				if _, err := rw.Write(blob); err != nil {
					t.Fatal(err)
				}
			},
		),
	)
	defer server.Close()

	invServer := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				if _, err := rw.Write([]byte("}{")); err != nil {
					t.Fatal(err)
				}
			},
		),
	)
	defer invServer.Close()

	errServer := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				rw.WriteHeader(http.StatusInternalServerError)
			},
		),
	)
	defer invServer.Close()

	type args struct {
		clientID  string
		urls      []string
		consensus int
	}
	tests := []struct {
		name    string
		args    args
		want    state.Balance
		wantErr bool
	}{
		{
			name: "ERR",
			args: args{
				urls: []string{
					"worng url",
					errServer.URL,
					invServer.URL,
				},
			},
			wantErr: true,
		},
		{
			name: "Empty_ERR",
			args: args{
				urls: []string{},
			},
			wantErr: true,
		},
		{
			name: "Consensus_ERR",
			args: args{
				urls: []string{
					server.URL,
				},
				consensus: 200,
			},
			wantErr: true,
		},
		{
			name: "OK",
			args: args{
				urls: []string{
					server.URL,
				},
				consensus: 0,
			},
			want: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MakeClientBalanceRequest(tt.args.clientID, tt.args.urls, tt.args.consensus)
			if (err != nil) != tt.wantErr {
				t.Errorf("MakeClientBalanceRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MakeClientBalanceRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetTransactionStatus(t *testing.T) {
	//balance := state.Balance(5)
	//server := httptest.NewServer(
	//	http.HandlerFunc(
	//		func(rw http.ResponseWriter, r *http.Request) {
	//			st := &state.State{
	//				Balance: balance,
	//			}
	//			blob, err := json.Marshal(st)
	//			if err != nil {
	//				t.Fatal(err)
	//			}
	//
	//			if _, err := rw.Write(blob); err != nil {
	//				t.Fatal(err)
	//			}
	//		},
	//	),
	//)
	//defer server.Close()

	invServer := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				if _, err := rw.Write([]byte("}{")); err != nil {
					t.Fatal(err)
				}
			},
		),
	)
	defer invServer.Close()

	errServer := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				rw.WriteHeader(http.StatusInternalServerError)
			},
		),
	)
	defer invServer.Close()

	nilTxnServer := httptest.NewServer(
		http.HandlerFunc(
			func(rw http.ResponseWriter, r *http.Request) {
				txn := Transaction{
					Hash: encryption.Hash("data"),
				}
				blob, err := json.Marshal(&txn)
				if err != nil {
					t.Fatal(err)
				}

				data := map[string]interface{}{
					"txn": blob,
				}
				blob, err = json.Marshal(data)
				if err != nil {
					t.Fatal(err)
				}

				if _, err := rw.Write(blob); err != nil {
					t.Fatal(err)
				}
			},
		),
	)
	defer invServer.Close()

	type args struct {
		txnHash string
		urls    []string
		sf      int
	}
	tests := []struct {
		name    string
		args    args
		want    *Transaction
		wantErr bool
	}{
		{
			name: "ERR",
			args: args{
				urls: []string{
					"worng url",
					errServer.URL,
					invServer.URL,
					nilTxnServer.URL,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTransactionStatus(tt.args.txnHash, tt.args.urls, tt.args.sf)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTransactionStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTransactionStatus() got = %v, want %v", got, tt.want)
			}
		})
	}
}
