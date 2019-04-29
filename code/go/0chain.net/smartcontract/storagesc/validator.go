package storagesc

import (
	"encoding/json"
	"sort"

	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
)

func (sc *StorageSmartContract) getValidatorsList() (*ValidatorNodes, error) {
	allValidatorsList := &ValidatorNodes{}
	allValidatorsBytes, err := sc.GetNode(ALL_VALIDATORS_KEY)
	if allValidatorsBytes == nil {
		return allValidatorsList, nil
	}
	err = json.Unmarshal(allValidatorsBytes.Encode(), allValidatorsList)
	if err != nil {
		return nil, common.NewError("getValidatorsList_failed", "Failed to retrieve existing validators list")
	}
	sort.SliceStable(allValidatorsList.Nodes, func(i, j int) bool {
		return allValidatorsList.Nodes[i].ID < allValidatorsList.Nodes[j].ID
	})
	return allValidatorsList, nil
}

func (sc *StorageSmartContract) addValidator(t *transaction.Transaction, input []byte) (string, error) {
	allValidatorsList, err := sc.getValidatorsList()
	if err != nil {
		return "", common.NewError("add_validator_failed", "Failed to get validator list."+err.Error())
	}
	newValidator := &ValidationNode{}
	err = newValidator.Decode(input) //json.Unmarshal(input, &newBlobber)
	if err != nil {
		return "", err
	}
	newValidator.ID = t.ClientID
	newValidator.PublicKey = t.PublicKey
	blobberBytes, _ := sc.GetNode(newValidator.GetKey())
	if blobberBytes == nil {
		allValidatorsList.Nodes = append(allValidatorsList.Nodes, newValidator)
		// allValidatorsBytes, _ := json.Marshal(allValidatorsList)
		sc.InsertNode(ALL_VALIDATORS_KEY, allValidatorsList)
		sc.InsertNode(newValidator.GetKey(), newValidator)
	}

	buff := newValidator.Encode()
	return string(buff), nil
}
