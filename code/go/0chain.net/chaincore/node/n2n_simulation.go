// +build !n2n_delays

package node

//InduceDelay - induces network delay - it's a noop for production deployment
func (nd *Node) InduceDelay(toNode *Node) {
}

//ReadNetworkDelays - read the network delay configuration - it's a noop for production ndeployment
func ReadNetworkDelays(file string) {

}
