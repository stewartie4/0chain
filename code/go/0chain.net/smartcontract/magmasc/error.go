package magmasc

import (
	"errors"

	"0chain.net/core/common"
)

const (
	errDelim = ": "

	errCodeAcceptTerms    = "accept_terms"
	errCodeCheckCondition = "check_condition"
	errCodeConsumerReg    = "consumer_reg"
	errCodeDataUsage      = "data_usage"
	errCodeDecode         = "decode_error"
	errCodeFetchData      = "fetch_data"
	errCodeInsertData     = "insert_data"
	errCodeProviderReg    = "provider_reg"
	errCodeProviderUpdate = "provider_update"
	errCodeSessionStop    = "session_stop"
	errCodeUpdateData     = "update_data"

	errCodeTokenPoolCreate = "token_pool_create"
	errCodeTokenPoolEmpty  = "token_pool_empty"
	errCodeTokenPoolRefund = "token_pool_refund"
	errCodeTokenPoolSpend  = "token_pool_spend"

	errTextAcknowledgmentInvalid = "acknowledgment invalid"
	errTextDecode                = "decode error"
	errTextInsufficientFunds     = "insufficient funds"
	errTextUnexpected            = "unexpected error"

	errCodeInvalidFuncName = "invalid_func_name"
	errTextInvalidFuncName = "function with provided name is not supported"
)

var (
	// errAcknowledgmentInvalid represents an error
	// that an acknowledgment was invalidated.
	errAcknowledgmentInvalid = common.NewErrInternal(errTextAcknowledgmentInvalid)

	// errDataUsageInvalid represents an error
	// that a data usage was invalidated.
	errDataUsageInvalid = common.NewErrInternal("data usage invalid")

	// errConsumerAlreadyExists represents an error that can occur while
	// Consumer is creating and saving in blockchain state.
	errConsumerAlreadyExists = errors.New("consumer already exists")

	// errProviderAlreadyExists represents an error that can occur while
	// Provider is creating and saving in blockchain state.
	errProviderAlreadyExists = errors.New("provider already exists")

	// errProviderTermsInvalid represents an error
	// that provider terms was invalidated.
	errProviderTermsInvalid = common.NewErrInternal("provider terms invalid")
)

// errIs wraps function errors.Is from stdlib to avoid import it
// in other places of the magma smart contract (magmasc) package.
func errIs(err, target error) bool {
	return errors.Is(err, target)
}

// wrapError wraps given error into a new error with format.
func wrapError(code, msg string, err error) *common.Error {
	var wrap string
	if err != nil {
		wrap = errDelim + err.Error()
	}

	return &common.Error{Code: code, Msg: msg + wrap}
}
