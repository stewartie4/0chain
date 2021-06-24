package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"

	"0chain.net/chaincore/chain/state"
)

func Test_Consumers_Decode(t *testing.T) {
	t.Parallel()

	list := mockConsumers()
	blob, _ := json.Marshal(list)

	tests := [2]struct {
		name    string
		blob    []byte
		want    Consumers
		wantErr bool
	}{
		{
			name: "OK",
			blob: blob,
			want: list,
		},
		{
			name:    "ERR",
			blob:    []byte(":"), // invalid json,
			wantErr: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := Consumers{}
			if err := got.Decode(test.blob); (err != nil) != test.wantErr {
				t.Errorf("Decode() error: %v | want: %v", err, nil)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_Consumers_Encode(t *testing.T) {
	t.Parallel()

	list := mockConsumers()
	blob, _ := json.Marshal(list)

	tests := [1]struct {
		name string
		list Consumers
		want []byte
	}{
		{
			name: "OK",
			list: list,
			want: blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.list.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_Consumers_contains(t *testing.T) {
	t.Parallel()

	const scID = "sc_id"

	list := mockConsumers()
	sci, cons := mockStateContextI(), mockConsumer()
	if _, err := sci.InsertTrieNode(nodeUID(scID, cons.ID, consumerType), &cons); err != nil {
		t.Fatalf("InsertTrieNode() got: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name string
		scID string
		cons *Consumer
		list Consumers
		sci  state.StateContextI
		want bool
	}{
		{
			name: "FALSE",
			scID: scID,
			cons: &Consumer{ID: "consumer_not_present_id"},
			list: list,
			sci:  sci,
			want: false,
		},
		{
			name: "InNodeList_TRUE",
			scID: scID,
			cons: list.Nodes[0],
			list: list,
			want: true,
		},
		{
			name: "InStateContext_TRUE",
			scID: scID,
			cons: &cons,
			list: list,
			sci:  sci,
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.list.contains(test.scID, test.cons, test.sci); got != test.want {
				t.Errorf("contains() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_sortedConsumers_add(t *testing.T) {
	t.Parallel()

	type args struct {
		consumer *Consumer
	}
	tests := []struct {
		name string
		m    sortedConsumers
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.add(tt.args.consumer); got != tt.want {
				t.Errorf("add() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sortedConsumers_get(t *testing.T) {
	t.Parallel()

	type args struct {
		id string
	}
	tests := []struct {
		name  string
		m     sortedConsumers
		args  args
		want  *Consumer
		want1 bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.m.get(tt.args.id)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("get() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("get() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_sortedConsumers_getIndex(t *testing.T) {
	t.Parallel()

	type args struct {
		id string
	}
	tests := []struct {
		name  string
		m     sortedConsumers
		args  args
		want  int
		want1 bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.m.getIndex(tt.args.id)
			if got != tt.want {
				t.Errorf("getIndex() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("getIndex() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_sortedConsumers_remove(t *testing.T) {
	t.Parallel()

	type args struct {
		id string
	}
	tests := []struct {
		name string
		m    sortedConsumers
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.remove(tt.args.id); got != tt.want {
				t.Errorf("remove() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sortedConsumers_removeByIndex(t *testing.T) {
	t.Parallel()

	type args struct {
		idx int
	}
	tests := []struct {
		name string
		m    sortedConsumers
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}

func Test_sortedConsumers_update(t *testing.T) {
	t.Parallel()

	type args struct {
		consumer *Consumer
	}
	tests := []struct {
		name string
		m    sortedConsumers
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.update(tt.args.consumer); got != tt.want {
				t.Errorf("update() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_extractConsumers(t *testing.T) {
	t.Parallel()

	sci, list := mockStateContextI(), mockConsumers()
	if _, err := sci.InsertTrieNode(AllConsumersKey, &list); err != nil {
		t.Fatalf("InsertTrieNode() got: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name    string
		sci     state.StateContextI
		want    *Consumers
		wantErr bool
	}{
		{
			name:    "OK",
			sci:     mockStateContextI(),
			want:    &Consumers{},
			wantErr: false,
		},
		{
			name:    "Nodes_OK",
			sci:     sci,
			want:    &list,
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			got, err := extractConsumers(test.sci)
			if (err != nil) != test.wantErr {
				t.Errorf("extractConsumers() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("extractConsumers() got = %v, want %v", got, test.want)
			}
		})
	}
}
