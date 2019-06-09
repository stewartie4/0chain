package minersc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"github.com/asaskevich/govalidator"
	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

const (
	//ADDRESS address of minersc
	ADDRESS   = "CF9C03CD22C9C7B116EED04E4A909F95ABEC17E98FE631D6AC94D5D8420C5B20"
	bufRounds = 5000 //ToDo: make it configurable
	cfdBuffer = 10
	name      = "miner"
)

//MinerSmartContract Smartcontract that takes care of all miner related requests
type MinerSmartContract struct {
	*smartcontractinterface.SmartContract
	bcContext smartcontractinterface.BCContextI
}

func (msc *MinerSmartContract) GetName() string {
	return name
}

func (msc *MinerSmartContract) GetAddress() string {
	return ADDRESS
}

func (msc *MinerSmartContract) GetRestPoints() map[string]smartcontractinterface.SmartContractRestHandler {
	return msc.RestHandlers
}

//SetSC setting up smartcontract. implementing the interface
func (msc *MinerSmartContract) SetSC(sc *smartcontractinterface.SmartContract, bcContext smartcontractinterface.BCContextI) {
	msc.SmartContract = sc
	msc.SmartContract.RestHandlers["/getNodepool"] = msc.GetNodepoolHandler
	msc.bcContext = bcContext
	msc.SmartContractExecutionStats["add_miner"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "add_miner"), nil)
	msc.SmartContractExecutionStats["viewchange_req"] = metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", msc.ID, "viewchange_req"), nil)
}

//Execute implemetning the interface
func (msc *MinerSmartContract) Execute(t *transaction.Transaction, funcName string, input []byte, balances c_state.StateContextI) (string, error) {

	switch funcName {

	case "add_miner":
		resp, err := msc.AddMiner(t, input, balances)
		if err != nil {
			return "", err
		}
		return resp, nil

	case "viewchange_req":
		resp, err := msc.RequestViewchange(t, input, balances)
		if err != nil {
			return "", err
		}
		return resp, nil

	default:
		return common.NewError("failed execution", "no function with that name").Error(), nil

	}
}

//REST API Handlers

//GetNodepoolHandler API to provide nodepool information for registered miners
func (msc *MinerSmartContract) GetNodepoolHandler(ctx context.Context, params url.Values, statectx c_state.StateContextI) (interface{}, error) {

	var regMiner MinerNode
	err := regMiner.decodeFromValues(params)
	if err != nil {
		Logger.Info("Returing error from GetNodePoolHandler", zap.Error(err))
		return nil, err
	}
	if !msc.doesMinerExist(regMiner.getKey(msc.ID), statectx) {
		return "", errors.New("unknown_miner" + err.Error())
	}
	npi := msc.bcContext.GetNodepoolInfo()

	return npi, nil
}

func (msc *MinerSmartContract) doesMinerExist(pkey datastore.Key, statectx c_state.StateContextI) bool {
	mbits, _ := statectx.GetTrieNode(pkey)
	if mbits != nil {
		return true
	}
	return false
}

//AddMiner Function to handle miner register
func (msc *MinerSmartContract) AddMiner(t *transaction.Transaction, input []byte, statectx c_state.StateContextI) (string, error) {

	allMinersList, err := msc.getMinersList(statectx)
	if err != nil {
		Logger.Error("Error in getting list from the DB", zap.Error(err))
		return "", errors.New("add_miner_failed - Failed to get miner list" + err.Error())
	}
	msc.verifyMinerState(statectx, "Checking allminerslist in the beginning")

	newMiner := &MinerNode{}
	err = newMiner.Decode(input)
	if err != nil {
		Logger.Error("Error in decoding the input", zap.Error(err))

		return "", err
	}
	Logger.Info("The new miner info", zap.String("base URL", newMiner.BaseURL), zap.String("ID", newMiner.ID), zap.String("pkey", newMiner.PublicKey), zap.Any("mscID", msc.ID))

	if newMiner.PublicKey == "" || newMiner.ID == "" {
		Logger.Error("public key or ID is empty")
		return "", errors.New("PublicKey or the ID is empty. Cannot proceed")
	}
	//ToDo: Add validation that ID is hash of PublicKey

	hostName, port, err := getHostnameAndPort(newMiner.BaseURL)
	if err != nil {
		return "", err
	}

	for _, miner := range allMinersList.Nodes {
		Logger.Info("checking if miner exists", zap.String("ID", miner.ID), zap.String("newMinerID", newMiner.ID))
		if miner.ID == newMiner.ID {
			Logger.Error("Miner received already exists", zap.String("ID", miner.ID), zap.String("baseURL", miner.BaseURL))
			buff := newMiner.Encode()
			return string(buff), nil
		}
	}
	mn := msc.getStoredMiner(statectx, newMiner)
	if mn != nil {
		Logger.Error("Miner received already exists", zap.String("ID", mn.ID), zap.String("baseURL", mn.BaseURL))
		buff := newMiner.Encode()
		return string(buff), nil
	}

	minerBytes, _ := statectx.GetTrieNode(newMiner.getKey(msc.ID))
	if minerBytes == nil {
		allMinersList.Nodes = append(allMinersList.Nodes, newMiner)
		_, err := statectx.InsertTrieNode(allMinersKey, allMinersList)

		if err != nil {
			Logger.Error("newMinerInsert failed", zap.Error(err))
			return "", err
		}
		statectx.InsertTrieNode(newMiner.getKey(msc.ID), newMiner)
		msc.verifyMinerState(statectx, "Checking allminerslist afterInsert")
		statectx.GetBlock().AddARegisteredMiner(newMiner.PublicKey, newMiner.ID, hostName, port)
	} else {
		Logger.Info("Miner received already exist", zap.String("url", newMiner.BaseURL))
	}

	buff := newMiner.Encode()
	return string(buff), nil
}

//RequestViewchange Function to handle miner viewchange request
func (msc *MinerSmartContract) RequestViewchange(t *transaction.Transaction, input []byte, statectx c_state.StateContextI) (string, error) {

	var regMiner MinerNode
	err := regMiner.Decode(input)
	if err != nil {
		Logger.Error("Error in decoding the input", zap.Error(err))

		return "", err
	}
	Logger.Info("The new view change request from", zap.String("base URL", regMiner.BaseURL))
	regMiner.ID = t.ClientID
	regMiner.PublicKey = t.PublicKey

	if !msc.doesMinerExist(regMiner.getKey(msc.ID), statectx) {
		Logger.Info("Miner received does not exist", zap.String("url", regMiner.BaseURL))
		return "", errors.New(regMiner.BaseURL + " Miner rdoes not exist")
	}

	curRound := statectx.GetBlock().Round
	vcRound := (((int64)((curRound + bufRounds) / 1000)) + 1) * 1000
	vcRoundInfo := &ViewchangeInfo{}

	vcRoundInfo.ViewchangeRound = vcRound
	vcRoundInfo.ViewchangeCFDRound = vcRound - cfdBuffer

	Logger.Info("RequestViewChange", zap.Int64("cur_round", curRound),
		zap.Int64("vc_round", vcRoundInfo.ViewchangeRound), zap.Int64("dkg_round", vcRoundInfo.ViewchangeCFDRound))

	buff := vcRoundInfo.encode()
	return string(buff), nil

}

//------------- local functions ---------------------
func (msc *MinerSmartContract) verifyMinerState(statectx c_state.StateContextI, msg string) {
	allMinersList, err := msc.getMinersList(statectx)
	if err != nil {
		Logger.Info(msg + " getMinersList_failed - Failed to retrieve existing miners list")
		return
	}
	if allMinersList == nil || len(allMinersList.Nodes) == 0 {
		Logger.Info(msg + " allminerslist is empty")
		return
	}

	Logger.Info(msg)
	for _, miner := range allMinersList.Nodes {
		Logger.Info("allminerslist", zap.String("url", miner.BaseURL), zap.String("ID", miner.ID))
	}

}

func (msc *MinerSmartContract) getStoredMiner(statectx c_state.StateContextI, miner *MinerNode) *MinerNode {
	mn := &MinerNode{}
	mn.ID = miner.ID
	minerBytes, err := statectx.GetTrieNode(mn.getKey(msc.ID))
	if err == nil {
		err := mn.Decode(minerBytes.Encode())
		if err == nil {
			return mn
		}
	} else {
		Logger.Info("error while looking for miner in trie", zap.String("ID", miner.ID), zap.Error(err))
	}
	return nil
}
func (msc *MinerSmartContract) getMinersList(statectx c_state.StateContextI) (*MinerNodes, error) {
	allMinersList := &MinerNodes{}
	allMinersBytes, err := statectx.GetTrieNode(allMinersKey)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, errors.New("getMinersList_failed - Failed to retrieve existing miners list")
	}
	if allMinersBytes == nil {
		return allMinersList, nil
	}
	allMinersList.Decode(allMinersBytes.Encode())
	return allMinersList, nil
}

func getHostnameAndPort(burl string) (string, int, error) {
	hostName := ""
	port := 0

	//ToDo: does rudimentary checks. Add more checks
	u, err := url.Parse(burl)
	if err != nil {
		return hostName, port, errors.New(burl + " is not a valid url. " + err.Error())
	}

	if u.Scheme != "http" { //|| u.scheme == "https"  we don't support
		return hostName, port, errors.New(burl + " is not a valid url. It does not have scheme http")
	}

	sp := u.Port()
	if sp == "" {
		return hostName, port, errors.New(burl + " is not a valid url. It does not have port number")
	}

	p, err := strconv.Atoi(sp)
	if err != nil {
		return hostName, port, errors.New(burl + " is not a valid url. " + err.Error())
	}

	hostName = u.Hostname()

	if govalidator.IsDNSName(hostName) || govalidator.IsIPv4(hostName) {
		return hostName, p, nil
	}

	Logger.Info("Both IsDNSName and IsIPV4 returned false for " + hostName)
	return "", 0, errors.New(burl + " is not a valid url. It not a valid IP or valid DNS name")

}
