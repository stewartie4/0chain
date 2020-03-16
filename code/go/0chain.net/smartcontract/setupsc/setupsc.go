package setupsc

import (
	"0chain.net/chaincore/smartcontract"
	"0chain.net/core/util"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"path"
	"sync"

	sci "0chain.net/chaincore/smartcontractinterface"
)

// Errors
var (
	ErrRegisteredTwice = errors.New("the smart contract service is registered twice")
)

var (
	mutex          sync.RWMutex
	smartContracts = make(map[string]sci.SmartContractInterface)
)

func Register(service sci.SmartContractInterface) error {
	mutex.Lock()
	defer mutex.Unlock()
	name := service.GetName()
	_, exists := smartContracts[name]
	if exists {
		return ErrRegisteredTwice
	}
	smartContracts[name] = service
	return nil
}

//SetupSmartContracts initialize smart contract addresses
func SetupSmartContracts() {
	mutex.RLock()
	defer mutex.RUnlock()
	for _, sc := range smartContracts {
		name := sc.GetName()

		useSelfState := sc.UseSelfState()
		var (
			db  *util.PNodeDB
			err error
		)
		if useSelfState {
			db, err = util.NewPNodeDB(path.Join("data", "rocksdb", "state_sc_"+name),
				path.Join("/0chain", "log", "rocksdb", "state_sc_name"+name))
			if err != nil {
				panic(err)
			}
		}

		smartContract := sci.NewSC(sc.GetAddress(), sc.GetName(), db)
		sc.SetSC(smartContract)

		if viper.GetBool(fmt.Sprintf("development.smart_contract.%v", sc.GetName())) {
			smartcontract.SetSmartContract(sc.GetAddress(),sc)
		}
	}
}

func IsUseStateSmartContract(name string) bool {
	mutex.RLock()
	defer mutex.RUnlock()
	sci, ok := smartContracts[name]
	if ok {
		return sci.UseSelfState()
	}
	return false
}

func StatesBlockInits(initiator sci.StateInitiator) {
	for _, sc := range smartContracts {
		name := sc.GetName()
		if IsUseStateSmartContract(name) {
			state := sc.InitState()
			initiator.InitStateSmartContract(name, state)
		}
	}
}
