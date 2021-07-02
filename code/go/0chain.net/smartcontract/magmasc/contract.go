package magmasc

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/rcrowley/go-metrics"

	chain "0chain.net/chaincore/chain/state"
	tx "0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

// acknowledgment tries to extract Acknowledgment with given id param.
func (m *MagmaSmartContract) acknowledgment(id datastore.Key, sci chain.StateContextI) (*Acknowledgment, error) {
	ackn := Acknowledgment{SessionID: id}

	data, err := sci.GetTrieNode(ackn.uid(m.ID))
	if err != nil {
		return nil, errWrap(errCodeFetchData, "fetch acknowledgment failed", err)
	}
	if err = json.Unmarshal(data.Encode(), &ackn); err != nil {
		return nil, errWrap(errCodeFetchData, "decode acknowledgment failed", err)
	}
	if err = ackn.validate(); err != nil {
		return nil, errWrap(errCodeFetchData, "validate acknowledgment failed", err)
	}

	return &ackn, nil
}

// acknowledgmentAccepted tries to extract Acknowledgment with given id param.
func (m *MagmaSmartContract) acknowledgmentAccepted(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	ackn, err := m.acknowledgment(vals.Get("id"), sci)
	if err != nil {
		return nil, err
	}

	return ackn, nil
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

// allConsumers represents MagmaSmartContract handler.
// Returns all registered Consumer's nodes stores in
// provided state.StateContextI with AllConsumersKey.
func (m *MagmaSmartContract) allConsumers(_ context.Context, _ url.Values, sci chain.StateContextI) (interface{}, error) {
	consumers, err := extractConsumers(sci)
	if err != nil {
		return nil, errWrap(errCodeFetchData, "fetch consumers list failed", err)
	}

	return consumers.Nodes.Sorted, nil
}

// allProviders represents MagmaSmartContract handler.
// Returns all registered Provider's nodes stores in
// provided state.StateContextI with AllProvidersKey.
func (m *MagmaSmartContract) allProviders(_ context.Context, _ url.Values, sci chain.StateContextI) (interface{}, error) {
	providers, err := extractProviders(sci)
	if err != nil {
		return nil, errWrap(errCodeFetchData, "fetch providers list failed", err)
	}

	return providers.Nodes.Sorted, nil
}

// billing tries to extract Billing data with given id param.
func (m *MagmaSmartContract) billing(id datastore.Key, sci chain.StateContextI) (*Billing, error) {
	billing := Billing{SessionID: id}

	data, err := sci.GetTrieNode(billing.uid(m.ID))
	if err != nil {
		return nil, errWrap(errCodeFetchData, "fetch billing data failed", err)
	}
	if data != nil { // decode saved data
		if err = billing.Decode(data.Encode()); err != nil {
			return nil, errWrap(errCodeFetchData, "decode billing data failed", err)
		}
	}

	return &billing, nil
}

// billingData tries to extract Billing data with given id param.
func (m *MagmaSmartContract) billingData(usage *DataUsage, sci chain.StateContextI) (*Billing, error) {
	ackn, err := m.acknowledgment(usage.SessionID, sci)
	if err != nil {
		return nil, errWrap(errCodeDataUsage, "fetch acknowledgment failed", err)
	}

	billing, err := m.billing(usage.SessionID, sci)
	if err != nil && !errIs(err, util.ErrValueNotPresent) {
		return nil, errWrap(errCodeDataUsage, "fetch billing data failed", err)
	}

	volume := usage.DownloadBytes + usage.UploadBytes
	usage.Amount = volume * ackn.ProviderTerms.Price
	billing.DataUsage = append(billing.DataUsage, usage)

	if _, err = sci.InsertTrieNode(billing.uid(m.ID), billing); err != nil { // update billing data
		return nil, errWrap(errCodeDataUsage, "insert consumer failed", err)
	}

	return billing, nil
}

// billingFetch tries to fetch Billing data with given id param.
func (m *MagmaSmartContract) billingFetch(_ context.Context, vals url.Values, sci chain.StateContextI) (interface{}, error) {
	return m.billing(vals.Get("id"), sci)
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
		return "", errWrap(errCodeAcceptTerms, "fetch provider terms failed", err)
	}
	if provider.Terms.expired() {
		return "", errNew(errCodeAcceptTerms, "provider terms is expired")
	}

	ackn.ConsumerID = txn.ClientID
	ackn.ProviderTerms = provider.Terms

	var pool tokenPool
	if _, err = pool.create(txn, &ackn, sci); err != nil {
		return "", errWrap(errCodeAcceptTerms, "create token pool failed", err)
	}
	if _, err = sci.InsertTrieNode(pool.uid(consumerUID(m.ID, ackn.ConsumerID)), &pool); err != nil {
		return "", errWrap(errCodeAcceptTerms, "insert token pool failed", err)
	}
	if _, err = sci.InsertTrieNode(ackn.uid(m.ID), &ackn); err != nil {
		return "", errWrap(errCodeAcceptTerms, "insert acknowledgment failed", err)
	}

	if err = m.providerUpdate(provider.termsIncrease(), sci); err != nil {
		return "", errWrap(errCodeAcceptTerms, "provider increase terms failed", err)
	}

	return string(ackn.Encode()), nil
}

// consumerRegister allows registering Consumer in blockchain and creates
// Consumer with Consumer.ID (equals to transaction client ID), adds it to all Consumers list,
// creates consumerPools for new Consumer and saves results in provided state.StateContextI.
func (m *MagmaSmartContract) consumerRegister(txn *tx.Transaction, sci chain.StateContextI) (string, error) {
	list, err := extractConsumers(sci)
	if err != nil {
		return "", errWrap(errCodeConsumerReg, "fetch consumers list failed", err)
	}

	consumer := Consumer{ID: txn.ClientID}
	if list.contains(m.ID, &consumer, sci) {
		return "", errWrap(errCodeConsumerReg, "consumer id: "+consumer.ID, errConsumerAlreadyExists)

	}

	// save the all consumers
	list.Nodes.add(&consumer)
	if _, err = sci.InsertTrieNode(AllConsumersKey, list); err != nil {
		return "", errWrap(errCodeConsumerReg, "insert consumers list failed", err)
	}
	// save the new consumer
	uid := nodeUID(m.ID, consumer.ID, consumerType)
	if _, err = sci.InsertTrieNode(uid, &consumer); err != nil {
		return "", errWrap(errCodeConsumerReg, "insert consumer failed", err)
	}

	return string(consumer.Encode()), nil
}

// consumerSessionStop checks input for validity and complete the session with
// stake spent tokens and refunds remaining balance by billing data.
func (m *MagmaSmartContract) consumerSessionStop(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	ackn := Acknowledgment{ConsumerID: txn.ClientID}
	if err := ackn.Decode(blob); err != nil {
		return "", errWrap(errCodeSessionStop, "decode acknowledgment failed", err)
	}
	if err := ackn.validate(); err != nil {
		return "", errWrap(errCodeSessionStop, "provided invalid acknowledgment", err)
	}

	billing, err := m.billing(ackn.SessionID, sci)
	if err != nil {
		return "", errNew(errCodeSessionStop, err.Error())
	}

	pool, err := m.tokenPollFetch(&ackn, sci)
	if err != nil {
		return "", errNew(errCodeSessionStop, err.Error())
	}

	amount := billing.Amount()
	if _, err = pool.spend(amount, sci); err != nil {
		return "", errNew(errCodeSessionStop, err.Error())
	}
	if pool.Balance > amount { // refund remaining balance of token pool
		if _, err = pool.transfer(pool.DelegateID, pool.ClientID, sci); err != nil {
			return "", errNew(errCodeSessionStop, err.Error())
		}
	}

	if _, err = sci.DeleteTrieNode(pool.uid(consumerUID(m.ID, ackn.ConsumerID))); err != nil {
		return "", errWrap(errCodeSessionStop, "delete token pool failed", err)
	}
	if _, err = sci.DeleteTrieNode(billing.uid(m.ID)); err != nil {
		return "", errWrap(errCodeSessionStop, "delete billing data failed", err)
	}
	if _, err = sci.DeleteTrieNode(ackn.uid(m.ID)); err != nil {
		return "", errWrap(errCodeSessionStop, "delete acknowledgment failed", err)
	}

	if _, err = m.providerTermsDecrease(ackn.ProviderID, sci); err != nil {
		return "", errWrap(errCodeSessionStop, "update provider terms failed", err)
	}

	return string(billing.Encode()), nil
}

// providerDataUsage updates the Provider billing session.
func (m *MagmaSmartContract) providerDataUsage(_ *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	var dataUsage DataUsage

	if err := dataUsage.Decode(blob); err != nil {
		return "", errWrap(errCodeDataUsage, "decode data usage failed", err)
	}
	if err := dataUsage.validate(); err != nil {
		return "", errWrap(errCodeDataUsage, "validate data usage failed", err)
	}

	billing, err := m.billingData(&dataUsage, sci)
	if err != nil {
		return "", errWrap(errCodeDataUsage, "append data usage failed", err)
	}

	return string(billing.Encode()), nil
}

// providerRegister allows registering Provider in blockchain and creates Provider
// with Provider.ID (equals to transaction client GetID), adds it to all Nodes list
// and saves results in provided state.StateContextI.
func (m *MagmaSmartContract) providerRegister(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	list, err := extractProviders(sci)
	if err != nil {
		return "", errWrap(errCodeProviderReg, "fetch providers list failed", err)
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
		return "", errWrap(errCodeProviderReg, "insert providers list failed", err)
	}
	// save the new provider
	uid := nodeUID(m.ID, provider.ID, providerType)
	if _, err = sci.InsertTrieNode(uid, &provider); err != nil {
		return "", errWrap(errCodeProviderReg, "fetch provider failed", err)
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
		return nil, errWrap(errCodeFetchData, "fetch provider terms failed", err)
	}

	return provider.Terms, nil
}

// providerTermsDecrease decrease and updates the current provider terms.
func (m *MagmaSmartContract) providerTermsDecrease(id datastore.Key, sci chain.StateContextI) (string, error) {
	provider, err := extractProvider(m.ID, id, sci)
	if err != nil {
		return "", errWrap(errCodeUpdateData, "fetch provider terms failed", err)
	}
	if err = m.providerUpdate(provider.termsDecrease(), sci); err != nil {
		return "", errWrap(errCodeUpdateData, "decrease provider terms failed", err)
	}

	return string(provider.Encode()), nil
}

// providerTermsDecrease increase and updates the current provider terms.
func (m *MagmaSmartContract) providerTermsIncrease(id datastore.Key, sci chain.StateContextI) (string, error) {
	provider, err := extractProvider(m.ID, id, sci)
	if err != nil {
		return "", errWrap(errCodeUpdateData, "fetch provider terms failed", err)
	}
	if err = m.providerUpdate(provider.termsIncrease(), sci); err != nil {
		return "", errWrap(errCodeUpdateData, "increase provider terms failed", err)
	}

	return string(provider.Encode()), nil
}

// providerTermsUpdate updates the current provider terms.
func (m *MagmaSmartContract) providerTermsUpdate(txn *tx.Transaction, blob []byte, sci chain.StateContextI) (string, error) {
	provider, err := extractProvider(m.ID, txn.ClientID, sci)
	if err != nil {
		return "", errWrap(errCodeUpdateData, "fetch provider failed", err)
	}
	if err = provider.Terms.Decode(blob); err != nil {
		return "", errWrap(errCodeUpdateData, "decode provider terms failed", err)
	}
	if err = provider.Terms.validate(); err != nil || provider.Terms.expired() {
		return "", errWrap(errCodeUpdateData, "validate provider terms failed", err)
	}
	// update provider data
	if err = m.providerUpdate(provider, sci); err != nil {
		return "", errWrap(errCodeUpdateData, "update provider failed", err)
	}

	return string(provider.Encode()), nil
}

// providerUpdate updates given provider in list and update trie nodes.
func (m *MagmaSmartContract) providerUpdate(provider *Provider, sci chain.StateContextI) error {
	list, err := extractProviders(sci)
	if err != nil {
		return errWrap(errCodeProviderUpdate, "fetch providers list failed", err)
	}
	if !list.Nodes.update(provider) {
		return errWrap(errCodeProviderUpdate, "update providers list failed", err)
	}
	// save the all providers
	if _, err = sci.InsertTrieNode(AllProvidersKey, list); err != nil {
		return errWrap(errCodeProviderUpdate, "insert providers list failed", err)
	}
	// save the provider
	uid := nodeUID(m.ID, provider.ID, providerType)
	if _, err = sci.InsertTrieNode(uid, provider); err != nil {
		return errWrap(errCodeProviderUpdate, "insert provider failed", err)
	}

	return nil
}

// tokenPollFetch fetches token pool form provided state.StateContextI.
func (m *MagmaSmartContract) tokenPollFetch(ackn *Acknowledgment, sci chain.StateContextI) (*tokenPool, error) {
	var pool tokenPool

	pool.ID = ackn.SessionID
	data, err := sci.GetTrieNode(pool.uid(consumerUID(m.ID, ackn.ConsumerID)))
	if err != nil || data == nil {
		return nil, errWrap(errCodeFetchData, "fetch token pool failed", err)
	}
	if err = json.Unmarshal(data.Encode(), &pool); err != nil {
		return nil, errWrap(errCodeFetchData, "decode token pool failed", err)
	}

	if pool.ID != ackn.SessionID {
		return nil, errNew(errCodeFetchData, "cannot fetch malformed token pool: "+ackn.SessionID)
	}
	if pool.ClientID != ackn.ConsumerID {
		return nil, errNew(errCodeFetchData, "cannot fetch not owned token pool: "+ackn.ConsumerID)
	}
	if pool.DelegateID != ackn.ProviderID {
		return nil, errNew(errCodeFetchData, "cannot fetch not delegated token pool: "+ackn.ProviderID)
	}

	return &pool, nil
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
