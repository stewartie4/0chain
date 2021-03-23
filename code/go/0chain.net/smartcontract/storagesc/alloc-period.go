package storagesc

import (
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
)

// allocation period used to calculate weighted average prices
type allocPeriod struct {
	read   state.Balance    // read price
	write  state.Balance    // write price
	period common.Timestamp // period (duration)
	size   int64            // size for period
}

func (ap *allocPeriod) weight() float64 {
	return float64(ap.period) * float64(ap.size)
}

// returns weighted average read and write prices
func (ap *allocPeriod) join(np *allocPeriod) (avgRead, avgWrite state.Balance) {
	var (
		apw, npw = ap.weight(), np.weight() // weights
		ws       = apw + npw                // weights sum
		rp, wp   float64                    // read sum, write sum (weighted)
	)
	rp = (float64(ap.read) * apw) + (float64(np.read) * npw)
	wp = (float64(ap.write) * apw) + (float64(np.write) * npw)
	avgRead = state.Balance(rp / ws)
	avgWrite = state.Balance(wp / ws)
	return
}
