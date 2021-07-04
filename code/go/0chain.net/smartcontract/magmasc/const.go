package magmasc

const (
	// Address is a SHA3-256 hex encoded hash of "magma" string.
	// Represents address of MagmaSmartContract.
	Address = "11f8411db41e34cea7c100f19faff32da8f3cd5a80635731cec06f32d08089be"

	// Name contents the smart contract name.
	Name = "magma"

	// colon represents values separator.
	colon = ":"

	// octetSize represents number of bits in an octet.
	octetSize = 8
)

// These constants represents SmartContractExecutionStats keys,
// used to identify smart contract functions by Consumer.
const (
	// AllConsumersKey is a concatenated Address
	// and SHA3-256 hex encoded hash of "all_consumers" string.
	AllConsumersKey = Address + "226fe0dc53026203416c348f675ce0c5ea35d87d959e41aaf6a3ca7829741710"

	// consumerType contents a value of type of Consumer's node.
	consumerType = "consumer"

	// consumerAcceptTerms represents the name of MagmaSmartContract function.
	// When function is called it means that Consumer accepted Provider terms.
	consumerAcceptTerms = "consumer_accept_terms"

	// consumerRegister represents name for Consumer's registration MagmaSmartContract function.
	consumerRegister = "consumer_register"

	// consumerSessionStop represents the name of MagmaSmartContract function.
	// When function is called it means that Consumer stops the session.
	consumerSessionStop = "consumer_session_stop"
)

// These constants represents SmartContractExecutionStats keys,
// used to identify smart contract functions by Provider.
const (
	// AllProvidersKey is a concatenated Address
	// and SHA3-256 hex encoded hash of "all_providers" string.
	AllProvidersKey = Address + "7e306c02ea1719b598aaf9dc7516eb930cd47c5360d974e22ab01e21d66a93d8"

	// providerType contents a value of type of Provider's node.
	providerType = "provider"

	// providerDataUsage represents name for
	// Provider's data usage billing MagmaSmartContract function.
	providerDataUsage = "provider_data_usage"

	// providerRegister represents name for
	// Provider's registration MagmaSmartContract function.
	providerRegister = "provider_register"

	// providerRegister represents name for
	// Provider's provider terms update MagmaSmartContract function.
	providerTermsUpdate = "provider_terms_update"

	providerDataUsageDuration    = 30          // seconds
	providerTermsExpiredDuration = 10          // seconds
	providerTermsProlongDuration = 1 * 60 * 60 // 1 hour

	providerTermsAutoUpdatePrice = 1 // coins
	providerTermsAutoUpdateQoS   = 1 // bytes per second
)
