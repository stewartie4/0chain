package smartcontract

import (
	sci "0chain.net/chaincore/smartcontractinterface"
)

func (s *SmartContract) GetContract(key string) (sc sci.SmartContractInterface, ok bool) {
	sc, ok = s.contract[key]
	return
}

func (s *SmartContract) SetContract(key string, sc sci.SmartContractInterface) {
	s.contract[key] = sc
}

func (s *SmartContract) LenContract() int {
	return len(s.contract)
}

func (s *SmartContract) GetContracts() map[string]sci.SmartContractInterface {
	return s.contract
}
