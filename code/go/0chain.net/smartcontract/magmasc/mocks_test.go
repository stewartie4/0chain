package magmasc

import (
	"log"
	"strconv"
	"strings"

	magma "github.com/magma/augmented-networks/accounting/protos"
	"github.com/stretchr/testify/mock"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	tx "0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	// mockStateContext implements mocked chain state context interface.
	mockStateContext struct {
		mocks.StateContextI
		store map[datastore.Key]util.Serializable
	}

	// mockSmartContract implements mocked mocked smart contract interface.
	mockSmartContract struct {
		mocks.SmartContractInterface
		ID string
		SC *MagmaSmartContract
	}

	// mockInvalidJson implements mocked util.Serializable interface for invalid json.
	mockInvalidJson struct{ ID datastore.Key }
)

// Decode implements util.Serializable interface.
func (m *mockInvalidJson) Decode([]byte) error {
	return errDecodeData
}

// Encode implements util.Serializable interface.
func (m *mockInvalidJson) Encode() []byte {
	return []byte(":")
}

func mockAcknowledgment() *Acknowledgment {
	return &Acknowledgment{
		AccessPointID: "access_point_id",
		ConsumerID:    "consumer_id",
		ProviderID:    "provider_id",
		SessionID:     "session_id",
		ProviderTerms: mockProviderTerms(),
	}
}

func mockBilling() *Billing {
	ackn := mockAcknowledgment()
	bill := Billing{
		SessionID: "session_id",
		DataUsage: mockDataUsage(),
	}

	volume := bill.DataUsage.DownloadBytes + bill.DataUsage.UploadBytes
	bill.Amount = int64(volume * ackn.ProviderTerms.GetPrice())

	return &bill
}

func mockConsumer() Consumer {
	return Consumer{ID: "consumer_id"}
}

func mockConsumers() Consumers {
	list := Consumers{Nodes: &consumersSorted{}}
	for i := 0; i < 10; i++ {
		list.Nodes.add(&Consumer{ID: "consumer_id" + strconv.Itoa(i)})
	}

	return list
}

func mockDataUsage() *DataUsage {
	return &DataUsage{
		DownloadBytes: 1000,
		UploadBytes:   1000,
		SessionID:     "session_id",
		SessionTime:   1 * 60, // 1 minute
	}
}

func mockSmartContractI() *mockSmartContract {
	argBlob := mock.AnythingOfType("[]uint8")
	argSci := mock.AnythingOfType("*magmasc.mockStateContext")
	argStr := mock.AnythingOfType("string")
	argTxn := mock.AnythingOfType("*transaction.Transaction")

	smartContract := mockSmartContract{
		ID: "sc_id",
		SC: &MagmaSmartContract{
			SmartContract: &sci.SmartContract{
				RestHandlers:                nil,
				SmartContractExecutionStats: nil,
			},
		},
	}

	smartContract.On("Execute", argTxn, argStr, argBlob, argSci).Return(
		func(txn *tx.Transaction, funcName string, blob []byte, sci chain.StateContextI) string {
			if _, err := smartContract.SC.Execute(txn, funcName, blob, sci); errIs(err, errInvalidFuncName) {
				return ""
			}

			return funcName
		},
		func(txn *tx.Transaction, funcName string, blob []byte, sci chain.StateContextI) error {
			if _, err := smartContract.SC.Execute(txn, funcName, blob, sci); errIs(err, errInvalidFuncName) {
				return err
			}

			return nil
		},
	)

	return &smartContract
}

func mockMagmaSmartContract() *MagmaSmartContract {
	return &MagmaSmartContract{
		SmartContract: &sci.SmartContract{
			RestHandlers:                nil,
			SmartContractExecutionStats: nil,
		},
	}
}

func mockProvider() Provider {
	return Provider{
		ID:    "provider_id",
		Terms: mockProviderTerms(),
	}
}

func mockProviders() Providers {
	list := Providers{Nodes: &providersSorted{}}
	for i := 0; i < 10; i++ {
		list.Nodes.add(&Provider{ID: "provider_id" + strconv.Itoa(i)})
	}

	return list
}

func mockProviderTerms() *ProviderTerms {
	return &ProviderTerms{
		Terms: mockTerms(),
		QoS:   mockQoS(),
	}
}

func mockStateContextI() *mockStateContext {
	argStr := mock.AnythingOfType("string")

	stateContext := mockStateContext{store: make(map[datastore.Key]util.Serializable)}

	stateContext.On("AddTransfer", mock.AnythingOfType("*state.Transfer")).Return(
		func(transfer *state.Transfer) error {
			if transfer.Amount < 0 {
				return errNegativeValue
			}
			if transfer.ToClientID == "not_present_id" {
				return util.ErrValueNotPresent
			}
			return nil
		},
	)
	stateContext.On("GetClientBalance", argStr).Return(
		func(id datastore.Key) state.Balance {
			if id == "consumer_id" {
				return 1000000000000 // 1000 * 1e9 units equal to one thousand coins
			}
			return 0
		},
		func(id datastore.Key) error {
			if id == "" {
				return util.ErrNodeNotFound
			}
			if id == "not_present_id" {
				return util.ErrValueNotPresent
			}
			return nil
		},
	)
	stateContext.On("GetTransaction").Return(
		func() *tx.Transaction {
			return &tx.Transaction{ToClientID: "to_client_id"}
		},
	)
	stateContext.On("GetTrieNode", argStr).Return(
		func(id datastore.Key) util.Serializable {
			if val, ok := stateContext.store[id]; ok {
				return val
			}
			return nil
		},
		func(id datastore.Key) error {
			if strings.Contains(id, "not_present_id") {
				return util.ErrValueNotPresent
			}
			if _, ok := stateContext.store[id]; ok {
				return nil
			}
			return util.ErrNodeNotFound
		},
	)
	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.Acknowledgment")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			stateContext.store[id] = val
			return ""
		},
		func(_ datastore.Key, _ util.Serializable) error { return nil },
	)
	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.Billing")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			stateContext.store[id] = val
			return ""
		},
		func(_ datastore.Key, _ util.Serializable) error { return nil },
	)
	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.Consumer")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			stateContext.store[id] = val
			return ""
		},
		func(_ datastore.Key, _ util.Serializable) error { return nil },
	)
	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.Consumers")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			stateContext.store[id] = val
			return ""
		},
		func(_ datastore.Key, _ util.Serializable) error { return nil },
	)
	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.mockInvalidJson")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			stateContext.store[id] = val
			return ""
		},
		func(_ datastore.Key, _ util.Serializable) error { return nil },
	)
	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.Provider")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			stateContext.store[id] = val
			return ""
		},
		func(_ datastore.Key, _ util.Serializable) error { return nil },
	)
	stateContext.On("InsertTrieNode", argStr, mock.AnythingOfType("*magmasc.Providers")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			stateContext.store[id] = val
			return ""
		},
		func(_ datastore.Key, _ util.Serializable) error { return nil },
	)

	nodeInvalid := mockInvalidJson{ID: "invalid_json_id"}
	if _, err := stateContext.InsertTrieNode(nodeInvalid.ID, &nodeInvalid); err != nil {
		log.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	return &stateContext
}

func mockTerms() Terms {
	return Terms{
		Price:     1,
		Volume:    0,
		ExpiredAt: common.Now() + providerTermsProlongDuration,
	}
}

func mockTokenPool() *tokenPool {
	pool := tokenPool{
		PayerID: "client_id",
		PayeeID: "delegate_id",
	}

	pool.ID = "pool_id"
	pool.Balance = 1000

	return &pool
}

func mockQoS() magma.QoS {
	return magma.QoS{
		DownloadMbps: 1,
		UploadMbps:   1,
	}
}
