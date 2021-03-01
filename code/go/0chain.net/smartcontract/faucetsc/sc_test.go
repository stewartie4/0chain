package faucetsc

import (
	"context"
	"net/url"
	"reflect"
	"testing"
	"time"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
)

func TestFaucetSmartContract_SetSC(t *testing.T) {
	sc := smartcontractinterface.NewSC("1")

	type fields struct {
		SmartContract *smartcontractinterface.SmartContract
	}
	type args struct {
		sc        *smartcontractinterface.SmartContract
		bcContext smartcontractinterface.BCContextI
	}
	tests := []struct {
		name             string
		fields           fields
		args             args
		wantPersPereodic func(context.Context, url.Values, c_state.StateContextI) (interface{}, error)
	}{
		{
			name:   "Test_FaucetSmartContract_SetSC_OK",
			fields: fields{SmartContract: sc},
			args: args{
				sc: sc,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &FaucetSmartContract{
				SmartContract: tt.fields.SmartContract,
			}
			fc.SetSC(tt.args.sc, tt.args.bcContext)

			pplimit := fc.RestHandlers["/personalPeriodicLimit"]
			if reflect.ValueOf(pplimit).Pointer() != reflect.ValueOf(fc.personalPeriodicLimit).Pointer() {
				t.Error("SetSC() personalPeriodicLimit wrong set result")
			}

			gpLimit := fc.RestHandlers["/globalPerodicLimit"]
			if reflect.ValueOf(gpLimit).Pointer() != reflect.ValueOf(fc.globalPerodicLimit).Pointer() {
				t.Error("SetSC() globalPerodicLimit wrong set result")
			}

			pourAmount := fc.RestHandlers["/pourAmount"]
			if reflect.ValueOf(pourAmount).Pointer() != reflect.ValueOf(fc.pourAmount).Pointer() {
				t.Error("SetSC() pourAmount wrong set result")
			}

			getConfig := fc.RestHandlers["/getConfig"]
			if reflect.ValueOf(getConfig).Pointer() != reflect.ValueOf(fc.getConfigHandler).Pointer() {
				t.Error("SetSC() getConfig wrong set result")
			}

		})
	}
}

func TestFaucetSmartContract_updateLimits(t *testing.T) {
	type fields struct {
		SmartContract *smartcontractinterface.SmartContract
	}
	type args struct {
		t         *transaction.Transaction
		inputData []byte
		balances  c_state.StateContextI
		gn        *GlobalNode
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "TestFaucetSmartContract_updateLimits_BadRequest",
			args: args{
				t:         makeTestTxWithOwner_OK_and_BadRequest(),
				inputData: []byte(testTxnDataOK),
				balances:  newTestEmptyBalances(),
				gn:        globalNode1(),
			},
			wantErr: true,
		},
		{
			name: "TestFaucetSmartContract_updateLimits_BadOwner",
			args: args{
				t:         makeTestTx1Ok(),
				inputData: []byte(testTxnDataOK),
				balances:  newTestEmptyBalances(),
				gn:        globalNode1(),
			},
			wantErr: true,
		},
		{
			name: "TestFaucetSmartContract_updateLimits_RequestOK_NodeData_wrong",
			args: args{
				t:         makeTestTxWithOwner_OK_and_RequestOK(),
				inputData: []byte(txnDataOk()),
				balances:  newTestEmptyBalances(),
				gn:        globalNode2(),
			},
			wantErr: false,
			want:    string(globalNode2().Encode()),
		},
		{
			name: "TestFaucetSmartContract_updateLimits_RequestOK_NodeData_Ok",
			args: args{
				t:         makeTestTxWithOwner_OK_and_RequestOK(),
				inputData: []byte(txnDataOk()),
				balances:  newTestEmptyBalances(),
				gn:        globalNode1(),
			},
			wantErr: false,
			want:    string(globalNode1().Encode()),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &FaucetSmartContract{
				SmartContract: tt.fields.SmartContract,
			}
			got, err := fc.updateLimits(tt.args.t, tt.args.inputData, tt.args.balances, tt.args.gn)
			if (err != nil) != tt.wantErr {
				t.Errorf("updateLimits() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("updateLimits() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFaucetSmartContract_getUserNode(t *testing.T) {
	type fields struct {
		SmartContract *smartcontractinterface.SmartContract
	}
	type args struct {
		id        string
		globalKey string
		balances  c_state.StateContextI
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *UserNode
		wantErr bool
	}{
		{
			name: "TestFaucetSmartContract_getUserNode_ValueNotPresent",
			args: args{
				id:        clientID1,
				globalKey: globalNode1Ok,
				balances:  newTestEmptyBalances(),
			},
			wantErr: true,
			want: &UserNode{
				ID:        clientID1,
				StartTime: time.Time{},
				Used:      0,
			},
		},
		{
			name: "TestFaucetSmartContract_getUserNode_OK",
			args: args{
				id:        clientID1,
				globalKey: globalNode1Ok,
				balances:  newTestEmptyBalancesWithValue(datastore.Key(globalNode1Ok + clientID1)),
			},
			wantErr: false,
			want:    newEmptyUserNode(),
		},
		{
			name: "TestFaucetSmartContract_getUserNode_OK_100Used",
			args: args{
				id:        clientID1,
				globalKey: globalNode1Ok,
				balances:  newTest100BalancesWithValue(datastore.Key(globalNode1Ok + clientID1)),
			},
			wantErr: false,
			want:    newEmptyUserNodeWith100(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &FaucetSmartContract{
				SmartContract: tt.fields.SmartContract,
			}
			got, err := fc.getUserNode(tt.args.id, tt.args.globalKey, tt.args.balances)
			if (err != nil) != tt.wantErr {
				t.Errorf("getUserNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getUserNode() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFaucetSmartContract_getUserVariables(t *testing.T) {
	type fields struct {
		SmartContract *smartcontractinterface.SmartContract
	}
	type args struct {
		t        *transaction.Transaction
		gn       *GlobalNode
		balances c_state.StateContextI
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *UserNode
	}{
		{
			name: "TestFaucetSmartContract_getUserVariables_With0used",
			args: args{
				t:        makeTestTx1Ok(),
				gn:       globalNode1(),
				balances: newTest100BalancesWithValue(datastore.Key(globalNode1Ok + clientID1)),
			},
			want: newEmptyUserNode(),
		},
		{
			name: "TestFaucetSmartContract_getUserVariables_With100Used",
			args: args{
				t:        makeTestTx1Ok(),
				gn:       globalNode1WithReset(),
				balances: newTest100BalancesWithValue(datastore.Key(globalNode1Ok + clientID1)),
			},
			want: newEmptyUserNodeWith100(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &FaucetSmartContract{
				SmartContract: tt.fields.SmartContract,
			}
			if got := fc.getUserVariables(tt.args.t, tt.args.gn, tt.args.balances); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getUserVariables() = %v, want %v", got, tt.want)
			}
		})
	}
}

//func TestFaucetSmartContract_pour(t *testing.T) {
//	type fields struct {
//		SmartContract *smartcontractinterface.SmartContract
//	}
//	type args struct {
//		t         *transaction.Transaction
//		inputData []byte
//		balances  c_state.StateContextI
//		gn        *GlobalNode
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		want    string
//		wantErr bool
//	}{
//		{
//			name: "Pour_OK",
//			args: args{
//				t:         makeTestTxWithOwner_OK_and_RequestOK(),
//				inputData: []byte(txnDataOk()),
//				balances:  newTestEmptyBalances(),
//				gn:        globalNode1(),
//			},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			fc := &FaucetSmartContract{
//				SmartContract: tt.fields.SmartContract,
//			}
//			got, err := fc.pour(tt.args.t, tt.args.inputData, tt.args.balances, tt.args.gn)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("pour() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if got != tt.want {
//				t.Errorf("pour() got = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
