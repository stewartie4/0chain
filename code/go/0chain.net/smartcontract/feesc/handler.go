package feesc

import (
	"context"
	"net/url"

	"0chain.net/core/common"
)

func (fsc *FeeSmartContract) globalState(ctx context.Context, params url.Values) (interface{}, error) {
	gn, err := fsc.getGlobalNode()
	if err != nil {
		return nil, common.NewError("failed to get limits", "global node does not exist")
	}
	return string(gn.Encode()), nil
}
