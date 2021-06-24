package magmasc

import (
	"strconv"

	magma "github.com/magma/augmented-networks/accounting/protos"
	"github.com/stretchr/testify/mock"

	"0chain.net/chaincore/mocks"
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

func mockAcknowledgment() Acknowledgment {
	return Acknowledgment{
		AccessPointID: "access_point_id",
		ConsumerID:    "consumer_id",
		ProviderID:    "provider_id",
		SessionID:     "session_id",
		ProviderTerms: mockProviderTerms(),
	}
}

func mockBilling() Billing {
	return Billing{{Amount: 1}, {Amount: 2}, {Amount: 3}, {Amount: 4}, {Amount: 5}}
}

func mockConsumer() Consumer {
	return Consumer{ID: "consumer_id"}
}

func mockConsumers() Consumers {
	list := Consumers{Nodes: make(sortedConsumers, 10)}
	for i := range list.Nodes {
		list.Nodes[i] = &Consumer{ID: "consumer_id" + strconv.Itoa(i)}
	}

	return list
}

func mockDataUsage() DataUsage {
	return DataUsage{
		Amount:        0,
		DownloadBytes: 1111,
		UploadBytes:   1111,
		SessionID:     "session_id",
		Timestamp:     common.Now(),
	}
}

func mockProvider() Provider {
	return Provider{
		ID:    "provider_id",
		Terms: mockProviderTerms(),
	}
}

func mockProviders() Providers {
	list := Providers{Nodes: make(sortedProviders, 10)}
	for i := range list.Nodes {
		list.Nodes[i] = &Provider{ID: "consumer_id" + strconv.Itoa(i)}
	}

	return list
}

func mockProviderTerms() ProviderTerms {
	return ProviderTerms{
		Terms: mockTerms(),
		QoS:   mockQoS(),
	}
}

func mockStateContextI() *mockStateContext {
	sci := mockStateContext{store: make(map[datastore.Key]util.Serializable)}
	mockStringArg := mock.AnythingOfType("string")

	sci.On("InsertTrieNode", mockStringArg, mock.AnythingOfType("*magmasc.mockInvalidJson")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			sci.store[id] = val
			return ""
		},
		func(_ datastore.Key, _ util.Serializable) error { return nil },
	)
	sci.On("InsertTrieNode", mockStringArg, mock.AnythingOfType("*magmasc.Provider")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			sci.store[id] = val
			return ""
		},
		func(_ datastore.Key, _ util.Serializable) error { return nil },
	)
	sci.On("InsertTrieNode", mockStringArg, mock.AnythingOfType("*magmasc.Providers")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			sci.store[id] = val
			return ""
		},
		func(_ datastore.Key, _ util.Serializable) error { return nil },
	)
	sci.On("InsertTrieNode", mockStringArg, mock.AnythingOfType("*magmasc.Consumer")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			sci.store[id] = val
			return ""
		},
		func(_ datastore.Key, _ util.Serializable) error { return nil },
	)
	sci.On("InsertTrieNode", mockStringArg, mock.AnythingOfType("*magmasc.Consumers")).Return(
		func(id datastore.Key, val util.Serializable) datastore.Key {
			sci.store[id] = val
			return ""
		},
		func(_ datastore.Key, _ util.Serializable) error { return nil },
	)
	sci.On("GetTrieNode", mockStringArg).Return(
		func(id datastore.Key) util.Serializable {
			if val, ok := sci.store[id]; ok {
				return val
			}
			return nil
		},
		func(id datastore.Key) error {
			if _, ok := sci.store[id]; ok {
				return nil
			}
			return util.ErrValueNotPresent
		},
	)

	return &sci
}

func mockTerms() Terms {
	return Terms{
		Price:     1,
		Volume:    0,
		ExpiredAt: common.Now() + common.Timestamp(providerTermsProlongDuration),
	}
}

func mockQoS() magma.QoS {
	return magma.QoS{
		DownloadMbps: 1.1,
		UploadMbps:   1.1,
	}
}
