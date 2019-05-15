package chain

import (
	"context"
	"encoding/hex"
	"net/http"

	"0chain.net/chaincore/node"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"go.uber.org/zap"
)

/*SetupNodeHandlers - setup the handlers for the chain */
func (c *Chain) SetupNodeHandlers() {
	http.HandleFunc("/_nh/list/m", c.GetMinersHandler)
	http.HandleFunc("/_nh/list/s", c.GetShardersHandler)
	http.HandleFunc("/_nh/list/b", c.GetBlobbersHandler)
}

/*MinerNotarizedBlockRequestor - reuqest a notarized block from a node*/
var MinerNotarizedBlockRequestor node.EntityRequestor

//BlockStateChangeRequestor - request state changes for the block
var BlockStateChangeRequestor node.EntityRequestor

//PartialStateRequestor - request partial state from a given root
var PartialStateRequestor node.EntityRequestor

//StateNodesRequestor - request a set of state nodes given their keys
var StateNodesRequestor node.EntityRequestor

//SCStateNodesRequestor - request a set of smart contract state nodes given their keys
var SCStateNodesRequestor node.EntityRequestor

/*SetupX2MRequestors - setup requestors */
func SetupX2MRequestors() {
	options := &node.SendOptions{Timeout: node.TimeoutLargeMessage, CODEC: node.CODEC_MSGPACK, Compress: true}

	blockEntityMetadata := datastore.GetEntityMetadata("block")
	MinerNotarizedBlockRequestor = node.RequestEntityHandler("/v1/_x2m/block/notarized_block/get", options, blockEntityMetadata)

	options = &node.SendOptions{Timeout: node.TimeoutLargeMessage, CODEC: node.CODEC_JSON, Compress: true}
	blockStateChangeEntityMetadata := datastore.GetEntityMetadata("block_state_change")
	BlockStateChangeRequestor = node.RequestEntityHandler("/v1/_x2m/block/state_change/get", options, blockStateChangeEntityMetadata)

	partialStateEntityMetadata := datastore.GetEntityMetadata("partial_state")
	PartialStateRequestor = node.RequestEntityHandler("/v1/_x2m/state/get", options, partialStateEntityMetadata)

	stateNodesEntityMetadata := datastore.GetEntityMetadata("state_nodes")
	StateNodesRequestor = node.RequestEntityHandler("/v1/_x2x/state/get_nodes", options, stateNodesEntityMetadata)

	SCStateNodesRequestor = node.RequestEntityHandler("/v1/_x2x/scstate/get_nodes", options, stateNodesEntityMetadata)
}

func SetupX2XResponders() {
	http.HandleFunc("/v1/_x2x/state/get_nodes", common.N2NRateLimit(node.ToN2NSendEntityHandler(StateNodesHandler)))
	http.HandleFunc("/v1/_x2x/scstate/get_nodes", common.N2NRateLimit(node.ToN2NSendEntityHandler(SCStateNodesHandler)))
}

//StateNodesHandler - return a list of state nodes
func StateNodesHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	r.ParseForm() // this is needed as we get multiple values for the same key
	nodes := r.Form["nodes"]
	c := GetServerChain()
	keys := make([]util.Key, len(nodes))
	for idx, nd := range nodes {
		key, err := hex.DecodeString(nd)
		if err != nil {
			return nil, err
		}
		keys[idx] = key
	}
	ns, err := c.GetStateNodesFrom(ctx, keys)
	if err != nil {
		if ns != nil {
			Logger.Error("state nodes handler", zap.Int("keys", len(nodes)), zap.Int("found_keys", len(ns.Nodes)), zap.Error(err))
			return ns, nil
		}
		Logger.Error("state nodes handler", zap.Int("keys", len(nodes)), zap.Error(err))
		return nil, err
	}
	Logger.Info("state nodes handler", zap.Int("keys", len(keys)), zap.Int("nodes", len(ns.Nodes)))
	return ns, nil
}

//SCStateNodesHandler - return a list of state nodes for smart contract
func SCStateNodesHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	r.ParseForm() // this is needed as we get multiple values for the same key
	nodes := r.Form["nodes"]
	scAddress := r.Form["sc_address"][0]
	c := GetServerChain()
	keys := make([]util.Key, len(nodes))
	for idx, nd := range nodes {
		key, err := hex.DecodeString(nd)
		if err != nil {
			return nil, err
		}
		keys[idx] = key
	}
	ns, err := c.GetSCStateNodesFrom(ctx, scAddress, keys)
	if err != nil {
		if ns != nil {
			Logger.Error("smart contract state nodes handler", zap.Int("keys", len(nodes)), zap.Int("found_keys", len(ns.Nodes)), zap.Error(err))
			return ns, nil
		}
		Logger.Error("smart contract state nodes handler", zap.Int("keys", len(nodes)), zap.Error(err))
		return nil, err
	}
	Logger.Info("smart contract state nodes handler", zap.Int("keys", len(keys)), zap.Int("nodes", len(ns.Nodes)))
	return ns, nil
}
