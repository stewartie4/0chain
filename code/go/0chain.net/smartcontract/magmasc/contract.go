package magmasc

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/rcrowley/go-metrics"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	tx "0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

// acknowledgment tries to extract Acknowledgment with given id param.
func (m *MagmaSmartContract) acknowledgment(id datastore.Key, sci chain.StateContextI) (*Acknowledgment, error) {
	ackn := Acknowledgment{SessionID: id}

	data, err := sci.GetTrieNode(ackn.uid(m.ID))
	if err != nil {
		return nil, errWrap(errCodeFetchData, "retrieve acknowledgment from state failed", err)
	}
	if err = json.Unmarshal(data.Encode(), &ackn); err != nil {
		return nil, errWrap(errCodeFetchData, "decode acknowledgment failed", err)
	}
	if err = ackn.validate(); err != nil {
		return nil, errWrap(errCodeFetchData, "validate acknowledgment failed", err)
	}

	return &ackn, nil
}

// acknowledgmentAcceptedVerify tries to extract Acknowledgment with given id param,
// validate and verifies others IDs from values for equality with extracted acknowledgment.
func (m *MagmaSmartContract) acknowledgmentAcceptedVerify(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	ackn, err := m.acknowledgment(vals.Get("session_id"), sci)
	if err != nil {
		return nil, err
	}

	switch {
	case ackn.AccessPointID != vals.Get("access_point_id"):
		return nil, errNew(errCodeBadRequest, "verify access point id failed")

	case ackn.ConsumerID != vals.Get("consumer_id"):
		return nil, errNew(errCodeBadRequest, "verify consumer id failed")

	case ackn.ProviderID != vals.Get("provider_id"):
		return nil, errNew(errCodeBadRequest, "verify provider id failed")

	case ackn.SessionID != vals.Get("session_id"):
		return nil, errNew(errCodeBadRequest, "verify session id failed")
	}

	return ackn, nil
}

func (m *MagmaSmartContract) acknowledgmentAccepted(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	ackn, err := m.acknowledgment(vals.Get("id"), sci)
	if err != nil {
		return nil, err
	}

	return ackn, nil
}

// allConsumers represents MagmaSmartContract handler.
// Returns all registered Consumer's nodes stores in
// provided state.StateContextI with AllConsumersKey.
func (m *MagmaSmartContract) allConsumers(_ context.Context, _ url.Values, sci chain.StateContextI) (interface{}, error) {
	consumers, err := extractConsumers(sci)
	if err != nil {
		return nil, errWrap(errCodeFetchData, "extract consumers list from state failed", err)
	}

	return consumers.Nodes.Sorted, nil
}

// allProviders represents MagmaSmartContract handler.
// Returns all registered Provider's nodes stores in
// provided state.StateContextI with AllProvidersKey.
func (m *MagmaSmartContract) allProviders(_ context.Context, _ url.Values, sci chain.StateContextI) (interface{}, error) {
	providers, err := extractProviders(sci)
	if err != nil {
		return nil, errWrap(errCodeFetchData, "extract providers list from state failed", err)
	}

	return providers.Nodes.Sorted, nil
}

// billingData tries to extract Billing data with given id param.
func (m *MagmaSmartContract) billingData(blob []byte, sci chain.StateContextI) (*Acknowledgment, *Billing, error) {
	var dataUsage DataUsage
	if err := dataUsage.Decode(blob); err != nil {
		return nil, nil, errWrap(errCodeDataUsage, "decode data usage failed", err)
	}
	if err := dataUsage.validate(); err != nil {
		return nil, nil, errWrap(errCodeDataUsage, "validate data usage failed", err)
	}

	ackn, err := m.acknowledgment(dataUsage.SessionID, sci)
	if err != nil {
		return nil, nil, errWrap(errCodeDataUsage, "extract acknowledgment failed", err)
	}

	data, err := sci.GetTrieNode(dataUsage.uid(m.ID))
	if err != nil && !errIs(err, util.ErrValueNotPresent) {
		return nil, nil, errWrap(errCodeDataUsage, "retrieve billing data failed", err)
	}

	billing := Billing{}
	if data != nil { // decode previous saved data
		if err = billing.Decode(data.Encode()); err != nil {
			return nil, nil, errWrap(errCodeDataUsage, "decode billing data failed", err)
		}
	}

	volume := dataUsage.DownloadBytes + dataUsage.UploadBytes
	dataUsage.Amount = volume * ackn.ProviderTerms.Price
	billing.DataUsage = append(billing.DataUsage, &dataUsage) // update billing data
	if _, err = sci.InsertTrieNode(dataUsage.uid(m.ID), &billing); err != nil {
		return nil, nil, errWrap(errCodeDataUsage, "save consumer to state failed", err)
	}

	return ackn, &billing, nil
}

// consumerAcceptTerms checks input for validity, sets the client's id
// from transaction to Acknowledgment.ConsumerID,
// set's hash of transaction to Acknowledgment.ID and inserts
// resulted Acknowledgment in provided state.StateContextI.
func (m *MagmaSmartContract) consumerAcceptTerms(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	var ackn Acknowledgment
	if err := ackn.Decode(blob); err != nil {
		return "", errWrap(errCodeAcceptTerms, "decode acknowledgment data failed", err)
	}
	if err := ackn.validate(); err != nil {
		return "", errWrap(errCodeAcceptTerms, "received acknowledgment is invalid", err)
	}

	provider, err := extractProvider(m.ID, ackn.ProviderID, sci)
	if err != nil {
		return "", errWrap(errCodeAcceptTerms, "extract provider terms failed", err)
	}
	if provider.Terms.expired() {
		return "", errNew(errCodeAcceptTerms, "provider terms is expired")
	}

	ackn.ConsumerID = txn.ClientID
	ackn.ProviderTerms = provider.Terms

	if _, err = m.tokenPoolCreate(txn.Hash, &ackn, sci); err != nil {
		return "", errNew(errCodeAcceptTerms, "provider terms is expired")
	}
	if _, err = sci.InsertTrieNode(ackn.uid(m.ID), &ackn); err != nil {
		return "", errWrap(errCodeAcceptTerms, "save acknowledgment failed", err)
	}
	if err = m.providerUpdate(provider.termsIncrease(), sci); err != nil {
		return "", errWrap(errCodeAcceptTerms, "provider increase terms failed", err)
	}

	return string(ackn.Encode()), nil
}

// consumerPoolsCreate creates consumer pools for given ID and insert it into blockchain state.
// if consumerPools with given ID already exist it returns errConsumerAlreadyExists.
// Also returns error occurred while inserting a new consumerPools into blockchain state.
func (m *MagmaSmartContract) consumerPoolsCreate(id datastore.Key, sci chain.StateContextI) (*consumerPools, error) {
	pools := consumerPools{UID: consumerUID(m.ID, id)}
	if _, err := sci.GetTrieNode(pools.UID); !errIs(err, util.ErrValueNotPresent) {
		return nil, errWrap(errCodeInsertData, "create consumer pools failed", errConsumerAlreadyExists)
	}

	if _, err := sci.InsertTrieNode(pools.UID, &pools); err != nil {
		return nil, errWrap(errCodeInsertData, "insert consumer pools failed", err)
	}

	return &pools, nil
}

// consumerPoolsFetch fetches existed consumer pools.
func (m *MagmaSmartContract) consumerPoolsFetch(id datastore.Key, sci chain.StateContextI) (*consumerPools, error) {
	data, err := sci.GetTrieNode(consumerUID(m.ID, id))
	if err != nil {
		return nil, errWrap(errCodeFetchData, "fetch consumer pools failed", err)
	}

	pools := consumerPools{}
	if err = json.Unmarshal(data.Encode(), &pools); err != nil {
		return nil, errWrap(errCodeFetchData, "decode consumer pools failed", err)
	}
	if pools.Pools == nil {
		pools.Pools = make(map[datastore.Key]datastore.Key)
	}

	return &pools, nil
}

// consumerRegister allows registering Consumer in blockchain and creates
// Consumer with Consumer.ID (equals to transaction client ID), adds it to all Consumers list,
// creates consumerPools for new Consumer and saves results in provided state.StateContextI.
func (m *MagmaSmartContract) consumerRegister(txn *tx.Transaction, sci chain.StateContextI) (string, error) {
	list, err := extractConsumers(sci)
	if err != nil {
		return "", errWrap(errCodeConsumerReg, "extract consumers list from state failed", err)
	}

	consumer := Consumer{ID: txn.ClientID}
	if list.contains(m.ID, &consumer, sci) {
		return "", errWrap(errCodeConsumerReg, "consumer id: "+consumer.ID, errConsumerAlreadyExists)

	}
	if _, err = m.consumerPoolsCreate(consumer.ID, sci); err != nil {
		return "", errWrap(errCodeConsumerReg, "create consumer pools failed", err)
	}

	// save the all consumers
	list.Nodes.add(&consumer)
	if _, err = sci.InsertTrieNode(AllConsumersKey, list); err != nil {
		return "", errWrap(errCodeConsumerReg, "save consumers list to state failed", err)
	}
	// save the new consumer
	uid := nodeUID(m.ID, consumer.ID, consumerType)
	if _, err = sci.InsertTrieNode(uid, &consumer); err != nil {
		return "", errWrap(errCodeConsumerReg, "save consumer to state failed", err)
	}

	return string(consumer.Encode()), nil
}

// consumerSessionStop checks input for validity and complete the session with
// stake spent tokens and refunds remaining balance by billing data.
func (m *MagmaSmartContract) consumerSessionStop(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	ackn := Acknowledgment{ConsumerID: txn.ClientID}
	if err := ackn.Decode(blob); err != nil {
		return "", errWrap(errCodeSessionStop, "decode acknowledgment data failed", err)
	}
	if err := ackn.validate(); err != nil {
		return "", errWrap(errCodeSessionStop, "provided acknowledgment is invalid", err)
	}

	billing, dataUsage := Billing{}, DataUsage{SessionID: ackn.SessionID}
	data, err := sci.GetTrieNode(dataUsage.uid(m.ID))
	if err != nil && !errIs(err, util.ErrValueNotPresent) {
		return "", errWrap(errCodeSessionStop, "retrieve billing data failed", err)
	}
	if data != nil { // decode previous saved data
		if err = billing.Decode(data.Encode()); err != nil {
			return "", errWrap(errCodeSessionStop, "decode billing data failed", err)
		}
	}

	pools, err := m.consumerPoolsFetch(txn.ClientID, sci)
	if err != nil {
		return "", errWrap(errCodeSessionStop, "fetch consumer pools failed", err)
	}
	balance, err := pools.tokenPollBalance(&ackn, sci)
	if err != nil {
		return "", errWrap(errCodeSessionStop, "fetch token pool balance failed", err)
	}

	amount := billing.Amount()
	if _, err = m.tokenPollSpend(&ackn, amount, sci); err != nil {
		return "", errWrap(errCodeSessionStop, "spend token pool failed", err)
	}
	if balance > amount {
		if _, err = m.tokenPoolRefund(&ackn, sci); err != nil {
			return "", errWrap(errCodeSessionStop, "refund token pool failed", err)
		}
	}
	if _, err = m.providerTermsDecrease(ackn.ProviderID, sci); err != nil {
		return "", errWrap(errCodeSessionStop, "update provider terms failed", err)
	}

	return string(billing.Encode()), nil
}

// providerDataUsage updates the Provider billing session.
func (m *MagmaSmartContract) providerDataUsage(_ *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	ackn, billing, err := m.billingData(blob, sci)
	if err != nil {
		return "", errWrap(errCodeDataUsage, "extract billing data failed", err)
	}

	pools, err := m.consumerPoolsFetch(ackn.ConsumerID, sci)
	if err != nil {
		return "", errWrap(errCodeDataUsage, "fetch consumer pools failed", err)
	}

	balance, err := pools.tokenPollBalance(ackn, sci)
	if err != nil {
		return "", errWrap(errCodeDataUsage, "fetch token pool balance failed", err)
	}

	amount := billing.Amount()
	if balance <= amount {
		if _, err = m.tokenPollSpend(ackn, amount, sci); err != nil {
			return "", errWrap(errCodeDataUsage, "stake token pool failed", err)
		}
		if _, err = m.providerTermsDecrease(ackn.ProviderID, sci); err != nil {
			return "", errWrap(errCodeSessionStop, "update provider terms failed", err)
		}
	}

	return string(billing.Encode()), nil
}

// providerRegister allows registering Provider in blockchain and creates Provider
// with Provider.ID (equals to transaction client GetID), adds it to all Nodes list
// and saves results in provided state.StateContextI.
func (m *MagmaSmartContract) providerRegister(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	list, err := extractProviders(sci)
	if err != nil {
		return "", errWrap(errCodeProviderReg, "extract providers list from state failed", err)
	}

	provider := Provider{}
	if err = provider.Decode(blob); err != nil {
		return "", errWrap(errCodeProviderReg, "decode provider data failed", err)
	}

	provider.ID = txn.ClientID
	if list.contains(m.ID, &provider, sci) {
		return "", errWrap(errCodeProviderReg, "provider id: "+provider.ID, errProviderAlreadyExists)

	}

	// save the all providers
	list.Nodes.add(&provider)
	if _, err = sci.InsertTrieNode(AllProvidersKey, list); err != nil {
		return "", errWrap(errCodeProviderReg, "save providers list to state failed", err)
	}
	// save the new provider
	uid := nodeUID(m.ID, provider.ID, providerType)
	if _, err = sci.InsertTrieNode(uid, &provider); err != nil {
		return "", errWrap(errCodeProviderReg, "save provider to state failed", err)
	}

	return string(provider.Encode()), nil
}

// providerTerms represents MagmaSmartContract handler.
// providerTerms looks for Provider with id, passed in params url.Values,
// in provided state.StateContextI and returns Provider.Terms.
func (m *MagmaSmartContract) providerTerms(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	providerID := vals.Get("provider_id")

	provider, err := extractProvider(m.ID, providerID, sci)
	if err != nil {
		return nil, errWrap(errCodeFetchData, "extract provider from state failed", err)
	}

	return provider.Terms, nil
}

// providerTermsDecrease decrease and updates the current provider terms.
func (m *MagmaSmartContract) providerTermsDecrease(id datastore.Key, sci chain.StateContextI) (string, error) {
	provider, err := extractProvider(m.ID, id, sci)
	if err != nil {
		return "", errWrap(errCodeUpdateData, "extract provider terms failed", err)
	}
	if err = m.providerUpdate(provider.termsDecrease(), sci); err != nil {
		return "", errWrap(errCodeUpdateData, "provider decrease terms failed", err)
	}

	return string(provider.Encode()), nil
}

// providerTermsDecrease increase and updates the current provider terms.
func (m *MagmaSmartContract) providerTermsIncrease(id datastore.Key, sci chain.StateContextI) (string, error) {
	provider, err := extractProvider(m.ID, id, sci)
	if err != nil {
		return "", errWrap(errCodeUpdateData, "extract provider terms failed", err)
	}
	if err = m.providerUpdate(provider.termsIncrease(), sci); err != nil {
		return "", errWrap(errCodeUpdateData, "provider increase terms failed", err)
	}

	return string(provider.Encode()), nil
}

// providerTermsUpdate updates the current provider terms.
func (m *MagmaSmartContract) providerTermsUpdate(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	provider, err := extractProvider(m.ID, txn.ClientID, sci)
	if err != nil {
		return "", errWrap(errCodeUpdateData, "extract provider from state failed", err)
	}
	if err = provider.Terms.Decode(blob); err != nil {
		return "", errWrap(errCodeUpdateData, "decode provider terms failed", err)
	}
	if err = provider.Terms.validate(); err != nil || provider.Terms.expired() {
		return "", errWrap(errCodeUpdateData, "validate provider terms failed", err)
	}
	// update provider data
	if err = m.providerUpdate(provider, sci); err != nil {
		return "", errWrap(errCodeUpdateData, "save providers list to state failed", err)
	}

	return string(provider.Encode()), nil
}

// providerUpdate updates given provider in list and update trie nodes.
func (m *MagmaSmartContract) providerUpdate(provider *Provider, sci chain.StateContextI) error {
	list, err := extractProviders(sci)
	if err != nil {
		return errWrap(errCodeProviderUpdate, "extract providers list from state failed", err)
	}
	if !list.Nodes.update(provider) {
		return errWrap(errCodeProviderUpdate, "update provider in list failed", err)
	}
	// save the all providers
	if _, err = sci.InsertTrieNode(AllProvidersKey, list); err != nil {
		return errWrap(errCodeProviderUpdate, "save providers list to state failed", err)
	}
	// save the provider
	uid := nodeUID(m.ID, provider.ID, providerType)
	if _, err = sci.InsertTrieNode(uid, provider); err != nil {
		return errWrap(errCodeProviderUpdate, "save provider to state failed", err)
	}

	return nil
}

// tokenPoolCreate creates token pool and appends it to token polls list.
func (m *MagmaSmartContract) tokenPoolCreate(id datastore.Key, ackn *Acknowledgment, sci chain.StateContextI) (string, error) {
	pools, err := m.consumerPoolsFetch(ackn.ConsumerID, sci)
	if err != nil {
		return "", errWrap(errCodeTokenPoolCreate, "fetch consumer pools failed", err)
	}

	resp, err := pools.tokenPollCreate(id, ackn, sci)
	if err != nil {
		return "", errWrap(errCodeTokenPoolCreate, "create token pool failed", err)
	}
	if _, err = sci.InsertTrieNode(pools.UID, pools); err != nil {
		return "", errWrap(errCodeTokenPoolCreate, "update consumer pools failed", err)
	}

	return resp, nil
}

// tokenPoolRefund removes token pool.
func (m *MagmaSmartContract) tokenPoolRefund(ackn *Acknowledgment, sci chain.StateContextI) (string, error) {
	pools, err := m.consumerPoolsFetch(ackn.ConsumerID, sci)
	if err != nil {
		return "", errWrap(errCodeTokenPoolRefund, "fetch consumer pools failed", err)
	}

	resp, err := pools.tokenPollRefund(ackn, sci)
	if err != nil {
		return "", errWrap(errCodeTokenPoolRefund, "refund token poll failed", err)
	}
	if _, err = sci.InsertTrieNode(pools.UID, pools); err != nil {
		return "", errWrap(errCodeTokenPoolRefund, "update consumer pools failed", err)
	}

	return resp, nil
}

// tokenPollSpend spends token pool.
func (m *MagmaSmartContract) tokenPollSpend(ackn *Acknowledgment, amount state.Balance, sci chain.StateContextI) (string, error) {
	pools, err := m.consumerPoolsFetch(ackn.ConsumerID, sci)
	if err != nil {
		return "", errWrap(errCodeTokenPoolSpend, "fetch consumer pools failed", err)
	}

	resp, err := pools.tokenPollSpend(ackn, amount, sci)
	if err != nil {
		return "", errWrap(errCodeTokenPoolSpend, "spend token poll failed", err)
	}
	if _, err = sci.InsertTrieNode(pools.UID, pools); err != nil {
		return "", errWrap(errCodeTokenPoolSpend, "update consumer pools failed", err)
	}

	return resp, nil
}

// mtRegisterTimer returns a metrics.Timer with specific
// name by given smart contract's id and function name.
func mtRegisterTimer(scID, fnName string) metrics.Timer {
	return metrics.GetOrRegisterTimer("sc:"+scID+":func:"+fnName, nil)
}

// nodeUID returns a uniq id for Node interacting with magma smart contract.
// scKey is an ID of magma smart contract and nodeID is and ID of Node.
// Should be used while inserting, removing or getting Node in state.StateContextI
func nodeUID(scID, nodeID, nodeType string) datastore.Key {
	return "sc:" + scID + colon + nodeType + colon + nodeID
}
