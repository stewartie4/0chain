package config

import (
	"testing"
	"time"
)

func TestAll(t *testing.T) {
	SetupDefaultConfig()
	SetupConfig()
	SetupSmartContractConfig()
	SetupDefaultSmartContractConfig()

	Configuration.DeploymentMode = DeploymentTestNet
	if !TestNet() {
		t.Error("Expected true, but got false")
	}

	Configuration.DeploymentMode = DeploymentDevelopment
	if !Development() {
		t.Error("Expected true, but got false")
	}

	Configuration.DeploymentMode = DeploymentMainNet
	if !MainNet() {
		t.Error("Expected true, but got false")
	}

	got := GetMainChainID()
	if got != MAIN_CHAIN {
		t.Errorf("Expected = %v, but got = %v", MAIN_CHAIN, got)
	}

	ServerChainID = ""
	got = GetServerChainID()
	if got != MAIN_CHAIN {
		t.Errorf("Expected = %v, but got = %v", MAIN_CHAIN, got)
	}

	ServerChainID = "server chain ID"
	got = GetServerChainID()
	if got != ServerChainID {
		t.Errorf("Expected = %v, but got = %v", ServerChainID, got)
	}

	SetServerChainID("")
	if ServerChainID != MAIN_CHAIN {
		t.Errorf("Expected = %v, but got = %v", MAIN_CHAIN, ServerChainID)
	}

	sChID := "server chain id"
	SetServerChainID(sChID)
	if ServerChainID != sChID {
		t.Errorf("Expected = %v, but got = %v", sChID, ServerChainID)
	}

	// checks config values

	tc := GetThresholdCount()
	if tc != 66 {
		t.Errorf("Expected = 67, but got = %v", tc)
	}

	tt := GetReBroadcastLFBTicketTimeout()
	if tt != time.Second*15 {
		t.Errorf("Expected = %v, but got = %v", time.Second*16, tt)
	}

	lfbTicket := GetLFBTicketAhead()
	if lfbTicket != 5 {
		t.Errorf("Expected = 2, but got = %v", lfbTicket)
	}

	fbfl := GetFBFetchingLifetime()
	if fbfl != time.Second*10 {
		t.Errorf("Expected = %v, but got = %v", time.Second*10, fbfl)
	}

	abdmsFromMiners := AsyncBlocksFetchingMaxSimultaneousFromMiners()
	if abdmsFromMiners != 100 {
		t.Errorf("Expected = 100, but got = %v", abdmsFromMiners)
	}

	abdmsFromSharders := AsyncBlocksFetchingMaxSimultaneousFromSharders()
	if abdmsFromSharders != 30 {
		t.Errorf("Expected = 30, but got = %v", abdmsFromSharders)
	}
}

func TestValidChain(t *testing.T) {
	type args struct {
		chain string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			args:    args{chain: ServerChainID},
			wantErr: false,
		},
		{
			name:    "ERR",
			args:    args{chain: "unknown id"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidChain(tt.args.chain); (err != nil) != tt.wantErr {
				t.Errorf("ValidChain() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
