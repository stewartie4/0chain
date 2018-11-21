package miner

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"0chain.net/chain"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/diagnostics"
	"0chain.net/node"
	"0chain.net/round"
)

/*SetupHandlers - setup miner handlers */
func SetupHandlers() {
	http.HandleFunc("/_chain_stats", ChainStatsHandler)
	http.HandleFunc("/_current_round_stats", CurrentRoundStatsHandler)
	http.HandleFunc("/_round_stats", RoundStatsHandler)
}

/*ChainStatsHandler - a handler to provide block statistics */
func ChainStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	c := GetMinerChain().Chain
	chain.PrintCSS(w)
	diagnostics.WriteStatisticsCSS(w)

	self := node.Self.Node
	fmt.Fprintf(w, "<div>%v - %v</div>", self.GetPseudoName(), self.Description)

	diagnostics.WriteConfiguration(w, c)

	fmt.Fprintf(w, "<br>")
	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>Block Finalization Statistics (Steady state)</h2>")
	diagnostics.WriteTimerStatistics(w, c, chain.SteadyStateFinalizationTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td>")
	fmt.Fprintf(w, "<h2>Block Finalization Statistics (Start to Finish)</h2>")
	diagnostics.WriteTimerStatistics(w, c, chain.StartToFinalizeTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")
	fmt.Fprintf(w, "<tr><td colspan='2'>")
	fmt.Fprintf(w, "<p>Block finalization time = block generation + block verification + network time (1*large message + 2*small message)</p>")
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>Txn Finalization Statistics (Start to Finish)</h2>")
	if config.Development() {
		diagnostics.WriteTimerStatistics(w, c, chain.StartToFinalizeTxnTimer, 1000000.0)
	} else {
		fmt.Fprintf(w, "Available only in development mode")
	}
	fmt.Fprintf(w, "</td><td valign='top'>")
	fmt.Fprintf(w, "<h2>Finalization Lag Statistics</h2>")
	diagnostics.WriteHistogramStatistics(w, c, chain.FinalizationLagMetric)
	fmt.Fprintf(w, "</td></tr>")

	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>Block Generation Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, bgTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td>")
	fmt.Fprintf(w, "<h2>Block Verification Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, bvTimer, 1000000.0)
	fmt.Fprintf(w, "</td></tr>")
	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>State Save Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, chain.StateSaveTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td></td></tr>")
	fmt.Fprintf(w, "<tr><td>")
	fmt.Fprintf(w, "<h2>State Prune Update Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, chain.StatePruneUpdateTimer, 1000000.0)
	fmt.Fprintf(w, "</td><td>")
	fmt.Fprintf(w, "<h2>State Prune Delete Statistics</h2>")
	diagnostics.WriteTimerStatistics(w, c, chain.StatePruneDeleteTimer, 1000000.0)
	fmt.Fprintf(w, "</tr>")
	fmt.Fprintf(w, "</table>")
}

func RoundStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	num, _ := strconv.ParseInt(r.FormValue("round"), 10, 64)
	rnd := GetMinerChain().GetMinerRound(num)
	if rnd == nil {
		fmt.Fprintf(w, "Round %v not found", num)
		return
	}
	fmt.Fprintf(w, "<html lang='en-US'><head><title>Round Stats</title></head>\n")
	common.AddCollapsibleStyle(w)
	fmt.Fprintf(w, "<table>\n")
	fmt.Fprintf(w, "<tr><td class='sheader' colspan='2'><b>Round #%v</b></td></tr>\n", rnd.Number)
	fmt.Fprintf(w, "<tr><td>Round</td><td>%v</td></tr>\n", rnd.Number)
	fmt.Fprintf(w, "<tr><td>Random Seed</td><td>%v</td></tr>\n", rnd.RandomSeed)
	var state string
	switch rnd.GetState() {
	case round.RoundShareVRF:
		state = "Share round vrf"
	case round.RoundVRFComplete:
		state = "Round VRF Complete"
	case round.RoundGenerating:
		state = "Generating"
	case round.RoundGenerated:
		state = "Generated"
	case round.RoundCollectingBlockProposals:
		state = "Collecting block proposals"
	case round.RoundStateVerificationTimedOut:
		state = "State verification timed out"
	case round.RoundStateFinalizing:
		state = "Finalizing round state"
	case round.RoundStateFinalized:
		state = "Finzalized round stte"
	}
	fmt.Fprintf(w, "<tr><td>State</td><td>%v</td></tr>\n", state)
	if rnd.Block != nil {
		fmt.Fprintf(w, "<tr><td>Block</td><td>%v</td></tr>\n", rnd.Block.Hash)
	}
	fmt.Fprintf(w, "</table>\n")

	proposedBlocks := rnd.GetBlocksByRank(rnd.GetProposedBlocks())
	notarizedBlocks := rnd.GetBlocksByRank(rnd.GetNotarizedBlocks())
	vrfShares := rnd.GetVRFShares()
	if len(proposedBlocks) > 0 {
		fmt.Fprintf(w, "<button class='collapsible'>Proposed Blocks</button>\n")
		fmt.Fprintf(w, "<div class='content'>\n")
		for _, b := range proposedBlocks {
			b.WriteBlock(w)
		}
		fmt.Fprintf(w, "</div>\n")
	}
	if len(notarizedBlocks) > 0 {
		fmt.Fprintf(w, "<button class='collapsible'>Notarized Blocks</button>\n")
		fmt.Fprintf(w, "<div class='content'>\n")
		for _, b := range notarizedBlocks {
			b.WriteBlock(w)
		}
		fmt.Fprintf(w, "</div>\n")
	}
	if len(vrfShares) > 0 {
		fmt.Fprintf(w, "<button class='collapsible'>VRF Shares</button>\n")
		fmt.Fprintf(w, "<div class='content'>\n")
		fmt.Fprintf(w, "<table border='1' class='menu' style='border-collapse: collapse;'>\n")
		fmt.Fprintf(w, "<tr class='header'><td>Miner ID</td><td>Set Index</td><td>Share</td></tr>")
		for _, s := range vrfShares {
			n := s.GetParty()
			fmt.Fprintf(w, "<tr><td>%v</td><td>%v</td><td>%v</td></tr>\n", n.ID, n.SetIndex, s.Share)
		}
		fmt.Fprintf(w, "</table>")
		fmt.Fprintf(w, "</div>\n")
	}
	common.AddCollapsibleScript(w)
}

func CurrentRoundStatsHandler(w http.ResponseWriter, r *http.Request) {
	roundNum := GetMinerChain().GetMinerRound(GetMinerChain().CurrentRound)
	v := url.Values{}
	v.Set("round", strconv.FormatInt(roundNum.Number, 10))
	r.Form = v
	RoundStatsHandler(w, r)
}
