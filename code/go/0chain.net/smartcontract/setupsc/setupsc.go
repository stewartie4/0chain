package setupsc

import (
	"0chain.net/chaincore/block/statesc"
	"0chain.net/chaincore/smartcontract"
	"0chain.net/core/datastore"
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
		sc.InitSC()
		name := sc.GetName()
		optionsSC := []sci.OptionSmartContract{
			sci.WithNameAddress(sc.GetAddress(), sc.GetName()),
		}

		var (
			db  *util.PNodeDB
			err error
		)
		isSeparateState := sc.IsSeparateState()
		if isSeparateState {
			db, err = util.NewPNodeDB(path.Join("data", "rocksdb", "state_sc_"+name),
				path.Join("/0chain", "log", "rocksdb", "state_sc_"+name))
			if err != nil {
				panic(err)
			}
			optionsSC = append(optionsSC, sci.WithStateDB(db))
		}

		sc.SetSC(optionsSC...)
		if viper.GetBool(fmt.Sprintf("development.smart_contract.%v", sc.GetName())) {
			smartcontract.SetSmartContract(sc.GetAddress(), sc)
		}
	}
}

func IsSeparateStateSmartContract(name string) bool {
	mutex.RLock()
	defer mutex.RUnlock()
	sci, ok := smartContracts[name]
	if ok {
		return sci.IsSeparateState()
	}
	return false
}

func GetAddressContract(name string) string {
	mutex.RLock()
	defer mutex.RUnlock()
	sci, ok := smartContracts[name]
	if ok {
		return sci.GetAddress()
	}
	return ""
}

func StatesBlockInits(initiator statesc.StateSCInitiator) {
	for _, sc := range smartContracts {
		name := sc.GetName()
		if IsSeparateStateSmartContract(name) {
			state := sc.InitState(datastore.Key(sc.GetAddress()))
			initiator.InitStateSmartContract(name, state)
		}
	}
}

func GetStateDBContract(name string) util.NodeDB {
	mutex.RLock()
	defer mutex.RUnlock()
	sci, ok := smartContracts[name]
	if ok {
		return sci.GetStateDB()
	}
	return nil
}

func GetSmartContracts() []string {
	mutex.RLock()
	defer mutex.RUnlock()
	result := make([]string, 0, len(smartContracts))
	for name := range smartContracts {
		result = append(result, name)
	}
	return result
}

func init() {
	statesc.StateSCDBGetter = GetStateDBContract
	statesc.StatesSCBlockInits = StatesBlockInits
	statesc.IsSeparateStateSmartContract = IsSeparateStateSmartContract
	//statesc.StateSCGetter = GetStateContract
}
