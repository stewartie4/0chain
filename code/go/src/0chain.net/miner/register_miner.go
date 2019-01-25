package miner

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"

	. "0chain.net/logging"
	"go.uber.org/zap"
)

type PoolMembers struct {
	Miners   []string `json:"miners"`
	Sharders []string `json:"sharders"`
}

var discoverIpPath = "/_nh/getpoolmembers"
var discoveryIps = []string{"http://198.18.0.71:7071",
	"http://198.18.0.72:7072",
	"http://198.18.0.73:7073"}

var members PoolMembers

func DiscoverPoolMembers() bool {

	for _, url := range discoveryIps {
		pm := PoolMembers{}
		MakeGetRequest(url+discoverIpPath, &pm)

		if pm.Miners != nil {
			if len(pm.Miners) == 0 {
				Logger.Info("Length of miners is 0")
			} else {
				sort.Strings(pm.Miners)
				sort.Strings(pm.Sharders)
				if len(members.Miners) == 0 {
					members = pm
					Logger.Info("First set of members from", zap.String("URL", url), zap.Any("Miners", members.Miners))

				} else {
					if !isSliceEq(pm.Miners, members.Miners) {
						Logger.Info("The members are different from", zap.String("URL", url),
							zap.Any("curset", members.Miners), zap.Any("Miners", pm.Miners))
						return false
					}
				}

			}
		} else {
			Logger.Info("Miners are nil")
			return false
		}
		Logger.Info("Discovered pool members", zap.Any("Miners", pm.Miners))
		return true
	}
	return false

}

func isSliceEq(a, b []string) bool {

	// If one is nil, the other must also be nil.
	if (a == nil) != (b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func MakeGetRequest(url string, result interface{}) {

	client := http.Client{}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalln(err)
	}

	resp, err := client.Do(request)
	if err != nil {
		Logger.Info("Failed to run get", zap.Error(err))
		return
	}

	if resp.Body != nil {
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			json.NewDecoder(resp.Body).Decode(result)
		}
	} else {
		Logger.Info("resp.Body is nil")
	}
}
