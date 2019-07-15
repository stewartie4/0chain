package minersc

import (
	"context"
	"errors"
	"net/url"

	c_state "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

func (msc *MinerSmartContract) GetUserPoolsHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	un, err := msc.getUserNode(params.Get("client_id"), msc.ID, balances)
	if err != nil {
		return nil, err
	}
	var totalInvested int64
	for _, p := range un.Pools {
		totalInvested += p.Balance
	}
	var response userResponse
	for key, pool := range un.Pools {
		stakePercent := float64(pool.Balance) / float64(totalInvested)
		response.Pools = append(response.Pools, &userPoolsResponse{poolInfo: pool, StakeDiversity: stakePercent, PoolID: key})

	}
	return response, nil
}

func (msc *MinerSmartContract) GetPoolStatsHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	mn, err := msc.getMinerNode(params.Get("miner_id"), msc.ID, balances)
	if err != nil {
		return nil, err
	}
	pool, ok := mn.Active[params.Get("pool_id")]
	if ok {
		return pool.PoolStats, nil
	}
	pool, ok = mn.Pending[params.Get("pool_id")]
	if ok {
		return pool.PoolStats, nil
	}
	pool, ok = mn.Deleting[params.Get("pool_id")]
	if ok {
		return pool.PoolStats, nil
	}
	return nil, common.NewError("failed to get stats", "pool doesn't exist")
}

//REST API Handlers

//GetNodepoolHandler API to provide nodepool information for registered miners
func (msc *MinerSmartContract) GetNodepoolHandler(ctx context.Context, params url.Values, statectx c_state.StateContextI) (interface{}, error) {

	regMiner := NewMinerNode()
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

func (msc *MinerSmartContract) GetMinerListHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	allMinersList, err := msc.getMinersList(balances)
	if err != nil {
		return "", nil
	}
	list := &SimpleMinerNodes{}
	for _, node := range allMinersList.Nodes {
		list.Nodes = append(list.Nodes, &SimpleMinerNode{ID: node.ID, BaseURL: node.BaseURL, PublicKey: node.PublicKey, ShortName: node.ShortName})
	}
	return list, nil
}

func (msc *MinerSmartContract) GetPhaseHandler(ctx context.Context, params url.Values, balances c_state.StateContextI) (interface{}, error) {
	pn, err := msc.getPhaseNode(balances)
	if err != nil {
		return "", err
	}
	return pn, nil
}
