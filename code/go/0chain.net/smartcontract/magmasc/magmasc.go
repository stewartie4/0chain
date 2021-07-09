package magmasc

import (
	"context"
	"net/url"

	chain "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	tx "0chain.net/chaincore/transaction"
)

type (
	// MagmaSmartContract represents smartcontractinterface.SmartContractInterface
	// implementation allows interacting with Magma.
	MagmaSmartContract struct {
		*sci.SmartContract
	}
)

var (
	// Ensure MagmaSmartContract implements smartcontractinterface.SmartContractInterface.
	_ sci.SmartContractInterface = (*MagmaSmartContract)(nil)
)

// NewMagmaSmartContract creates smartcontractinterface.SmartContractInterface
// and sets provided smartcontractinterface.SmartContract to corresponding
// MagmaSmartContract field and configures RestHandlers and SmartContractExecutionStats.
func NewMagmaSmartContract() sci.SmartContractInterface {
	msc := MagmaSmartContract{
		SmartContract: sci.NewSC(Address),
	}

	// Magma smart contract REST handlers
	msc.RestHandlers["/acknowledgmentAccepted"] = msc.acknowledgmentAccepted
	msc.RestHandlers["/acknowledgmentAcceptedVerify"] = msc.acknowledgmentAcceptedVerify
	msc.RestHandlers["/acknowledgmentExist"] = msc.acknowledgmentExist
	msc.RestHandlers["/allConsumers"] = msc.allConsumers
	msc.RestHandlers["/allProviders"] = msc.allProviders
	msc.RestHandlers["/billingFetch"] = msc.billingFetch
	msc.RestHandlers["/providerTerms"] = msc.providerTerms

	// consumer setup section
	msc.SmartContractExecutionStats[consumerAcceptTerms] = mtRegisterTimer(msc.ID, consumerAcceptTerms)
	msc.SmartContractExecutionStats[consumerRegister] = mtRegisterTimer(msc.ID, consumerRegister)
	msc.SmartContractExecutionStats[consumerSessionStop] = mtRegisterTimer(msc.ID, consumerSessionStop)

	// provider setup section
	msc.SmartContractExecutionStats[providerDataUsage] = mtRegisterTimer(msc.ID, providerDataUsage)
	msc.SmartContractExecutionStats[providerRegister] = mtRegisterTimer(msc.ID, providerRegister)
	msc.SmartContractExecutionStats[providerTermsUpdate] = mtRegisterTimer(msc.ID, providerTermsUpdate)

	return &msc
}

// Execute implements smartcontractinterface.SmartContractInterface.
func (m *MagmaSmartContract) Execute(txn *tx.Transaction, call string, blob []byte, sci chain.StateContextI) (string, error) {
	switch call {
	// consumer's function list
	case consumerAcceptTerms:
		return m.consumerAcceptTerms(txn, blob, sci)
	case consumerRegister:
		return m.consumerRegister(txn, sci)
	case consumerSessionStop:
		return m.consumerSessionStop(txn, blob, sci)

	// provider's function list
	case providerDataUsage:
		return m.providerDataUsage(txn, blob, sci)
	case providerRegister:
		return m.providerRegister(txn, blob, sci)
	case providerTermsUpdate:
		return m.providerTermsUpdate(txn, blob, sci)
	}

	return "", errInvalidFuncName
}

// GetAddress implements smartcontractinterface.SmartContractInterface.
func (m *MagmaSmartContract) GetAddress() string {
	return Address
}

// GetExecutionStats implements smartcontractinterface.SmartContractInterface.
func (m *MagmaSmartContract) GetExecutionStats() map[string]interface{} {
	return m.SmartContractExecutionStats
}

// GetHandlerStats implements smartcontractinterface.SmartContractInterface.
func (m *MagmaSmartContract) GetHandlerStats(ctx context.Context, params url.Values) (interface{}, error) {
	return m.SmartContract.HandlerStats(ctx, params)
}

// GetName implements smartcontractinterface.SmartContractInterface.
func (m *MagmaSmartContract) GetName() string {
	return Name
}

// GetRestPoints implements smartcontractinterface.SmartContractInterface.
func (m *MagmaSmartContract) GetRestPoints() map[string]sci.SmartContractRestHandler {
	return m.RestHandlers
}
