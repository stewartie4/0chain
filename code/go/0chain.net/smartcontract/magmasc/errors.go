package magmasc

import (
	"errors"
)

const (
	errDelim = ": "

	errCodeAcceptTerms    = "accept_terms"
	errCodeBadRequest     = "bad_request"
	errCodeCheckCondition = "check_condition"
	errCodeConsumerReg    = "consumer_reg"
	errCodeDataUsage      = "data_usage"
	errCodeDecode         = "decode_error"
	errCodeFetchData      = "fetch_data"
	errCodeInsertData     = "insert_data"
	errCodeInternal       = "internal_error"
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

type (
	// wrapper implements Wrapper interface.
	errWrapper struct {
		code string
		text string
		wrap error
	}
)

var (
	// errAcknowledgmentInvalid represents an error
	// that an acknowledgment was invalidated.
	errAcknowledgmentInvalid = errNew(errCodeInternal, errTextAcknowledgmentInvalid)

	// errDataUsageInvalid represents an error
	// that a data usage was invalidated.
	errDataUsageInvalid = errNew(errCodeInternal, "data usage invalid")

	// errDecodeData represents an error
	// that decode data was failed.
	errDecodeData = errNew(errCodeDecode, errTextDecode)

	// errConsumerAlreadyExists represents an error that can occur while
	// Consumer is creating and saving in blockchain state.
	errConsumerAlreadyExists = errNew(errCodeInternal, "consumer already exists")

	// errProviderAlreadyExists represents an error that can occur while
	// Provider is creating and saving in blockchain state.
	errProviderAlreadyExists = errNew(errCodeInternal, "provider already exists")

	// errProviderTermsInvalid represents an error
	// that provider terms was invalidated.
	errProviderTermsInvalid = errNew(errCodeInternal, "provider terms invalid")
)

// Error implements error interface.
func (m *errWrapper) Error() string {
	return m.code + errDelim + m.text
}

// Unwrap implements error unwrap interface.
func (m *errWrapper) Unwrap() error {
	return m.wrap
}

// WrapErr implements error wrapper interface.
func (m *errWrapper) WrapErr(err error) *errWrapper {
	if err != nil {
		m.wrap = err
		m.text += errDelim + err.Error()
	}

	return m
}

// errIs wraps function errors.Is from stdlib to avoid import it
// in other places of the magma smart contract (magmasc) package.
func errIs(err, target error) bool {
	return errors.Is(err, target)
}

// errNew returns constructed error wrapper interface.
func errNew(code, text string) *errWrapper {
	return &errWrapper{code: code, text: text}
}

// errWrap wraps given error into a new error with format.
func errWrap(code, text string, err error) *errWrapper {
	var wrap string
	if err != nil {
		wrap = errDelim + err.Error()
	}

	return &errWrapper{code: code, text: text + wrap, wrap: err}
}
