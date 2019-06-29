package node

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"runtime/pprof"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/logging"
	. "0chain.net/core/logging"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var done = make(chan bool, 1)

func (np *Pool) CancelDKGMonitor() {
	Logger.Info("Canceling DKG Monitor")
	done <- true
}

/*OneTimeStatusMonitor - checks the status of nodes only once*/
func (np *Pool) OneTimeStatusMonitor(ctx context.Context) {
	Logger.Info("Triggereing oneTimeStatusMonitor")
	GNodeStatusMonitor(ctx)
}

/*DownloadNodeData - downloads the node definition data for the given pool type from the given node */
func (np *Pool) DownloadNodeData(node *Node) bool {
	url := fmt.Sprintf("%v/_nh/list/%v", node.GetN2NURLBase(), node.GetNodeType())
	client := &http.Client{Timeout: TimeoutLargeMessage}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	dnp := NewPool(NodeTypeMiner)
	ReadNodes(resp.Body, dnp, dnp, dnp)
	var changed = false
	for _, node := range dnp.Nodes {
		if _, ok := np.NodesMap[node.GetKey()]; !ok {
			node.Status = NodeStatusActive
			np.AddNode(node)
			changed = true
		}
	}
	if changed {
		np.ComputeProperties()
	}
	return true
}

// MemoryUsage Log memory usage for a node
func (n *GNode) MemoryUsage() {
	ticker := time.NewTicker(5 * time.Minute)
	for true {
		select {
		case <-ticker.C:
			common.LogRuntime(logging.MemUsage, zap.Any(n.Description, n.ShortName))

			// Average time duration to add go routine logs to 0chain.log file => 618.184µs
			// Average increase in file size for each update => 10 kB
			if viper.GetBool("logging.memlog") {
				buf := new(bytes.Buffer)
				pprof.Lookup("goroutine").WriteTo(buf, 1)
				logging.Logger.Info("runtime", zap.String("Go routine output", buf.String()))
			}
		}
	}
}
