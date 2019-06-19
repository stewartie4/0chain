package node

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/core/build"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	. "0chain.net/core/logging"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var gnodes = make(map[string]*GNode)

/*RegisterNode - register a node to a global registery
* We need to keep track of a global register of nodes. This is required to ensure we can verify a signed request
* coming from a node
 */
func RegisterNode(gnode *GNode) {
	gnodes[gnode.GetKey()] = gnode
}

/*DeregisterNode - deregisters a node */
func DeregisterNode(nodeID string) {
	delete(gnodes, nodeID)
}

func GetNodes() map[string]*GNode {
	return gnodes
}

/*GetNode - get the node from the registery */
func GetNode(nodeID string) *GNode {
	return gnodes[nodeID]
}

var (
	NodeStatusInactive = 0
	NodeStatusActive   = 1
)

var (
	NodeTypeMiner   int8 = 0
	NodeTypeSharder int8 = 1
	NodeTypeBlobber int8 = 2
)

var NodeTypeNames = common.CreateLookups("m", "Miner", "s", "Sharder", "b", "Blobber")

/*GNode - a struct holding the global information of a node*/
type GNode struct {
	client.Client  `json:"client"`
	N2NHost        string    `json:"n2n_host"`
	Host           string    `json:"host"`
	Port           int       `json:"port"`
	Type           int8      `json:"node_type"`
	ShortName      string    `json:"shortname"`
	Description    string    `json:"description"`
	Status         int       `json:"-"`
	LastActiveTime time.Time `json:"-"`
	ErrorCount     int       `json:"-"`
	CommChannel    chan bool `json:"-"`
	//These are approximiate as we are not going to lock to update
	Sent       int64 `json:"-"` // messages sent to this node
	SendErrors int64 `json:"-"` // failed message sent to this node
	Received   int64 `json:"-"` // messages received from this node

	TimersByURI map[string]metrics.Timer     `json:"-"`
	SizeByURI   map[string]metrics.Histogram `json:"-"`

	LargeMessageSendTime float64 `json:"-"`
	SmallMessageSendTime float64 `json:"-"`

	LargeMessagePullServeTime float64 `json:"-"`
	SmallMessagePullServeTime float64 `json:"-"`

	mutex *sync.Mutex

	ProtocolStats interface{} `json:"-"`

	idBytes []byte

	Info Info `json:"info"`
}

// Node Node that will be used in NodePools
type Node struct {
	*GNode   `json:"gnode"`
	SetIndex int `json:"-"`
}

/*Provider - create a node object */
func Provider() *GNode {
	gnode := &GNode{}
	// queue up at most these many messages to a node
	// because of this, we don't want the status monitoring to use this communication layer
	gnode.CommChannel = make(chan bool, 5)
	for i := 0; i < cap(gnode.CommChannel); i++ {
		gnode.CommChannel <- true
	}
	gnode.mutex = &sync.Mutex{}
	gnode.TimersByURI = make(map[string]metrics.Timer, 10)
	gnode.SizeByURI = make(map[string]metrics.Histogram, 10)
	return gnode
}

/*Equals - if two gnodes are equal. Only check by id, we don't accept configuration from anyone */
func (n *GNode) Equals(n2 *GNode) bool {
	if datastore.IsEqual(n.GetKey(), n2.GetKey()) {
		return true
	}
	if n.Port == n2.Port && n.Host == n2.Host {
		return true
	}
	return false
}

func (n *GNode) IsGNodeRegistered() bool {
	if n.GetKey() == "" {
		Logger.Error("gnode key is empty")
		return false
	}
	if gnodes[n.GetKey()] == nil {
		Logger.Error("gnode key does not exist", zap.String("key", n.GetKey()))
		return false
	}
	return true
}

/*Print - print gnode's info that is consumable by Read */
func (n *GNode) Print(w io.Writer) {
	fmt.Fprintf(w, "%v,%v,%v,%v,%v\n", n.GetNodeType(), n.Host, n.Port, n.GetKey(), n.PublicKey)
}

/*Read - read a node config line and create the node */
func Read(line string) (*GNode, error) {
	gnode := Provider()
	fields := strings.Split(line, ",")
	if len(fields) != 5 {
		return nil, common.NewError("invalid_num_fields", fmt.Sprintf("invalid number of fields [%v]", line))
	}
	switch fields[0] {
	case "m":
		gnode.Type = NodeTypeMiner
	case "s":
		gnode.Type = NodeTypeSharder
	case "b":
		gnode.Type = NodeTypeBlobber
	default:
		return nil, common.NewError("unknown_node_type", fmt.Sprintf("Unkown node type %v", fields[0]))
	}
	gnode.Host = fields[1]
	if gnode.Host == "" {
		if gnode.Port != config.Configuration.Port {
			gnode.Host = config.Configuration.Host
		} else {
			panic(fmt.Sprintf("invalid node setup for %v\n", gnode.GetKey()))
		}
	}

	port, err := strconv.ParseInt(fields[2], 10, 32)
	if err != nil {
		return nil, err
	}
	gnode.Port = int(port)
	gnode.SetID(fields[3])
	gnode.PublicKey = fields[4]
	gnode.Client.SetPublicKey(gnode.PublicKey)
	hash := encryption.Hash(gnode.PublicKeyBytes)
	if gnode.ID != hash {
		return nil, common.NewError("invalid_client_id", fmt.Sprintf("public key: %v, client_id: %v, hash: %v\n", gnode.PublicKey, gnode.ID, hash))
	}
	gnode.ComputeProperties()
	if Self.PublicKey == gnode.PublicKey {
		setSelfNode(gnode)
	}
	return gnode, nil
}

func CreateGNode(nType int8, port int, host, n2nHost, ID, pkey, desc, shortName string) (*GNode, error) {
	toN := Provider()
	toN.Type = nType
	toN.Host = host
	toN.N2NHost = n2nHost
	toN.Port = port
	toN.SetID(ID)
	toN.PublicKey = pkey
	toN.Description = desc
	toN.ShortName = shortName
	toN.Client.SetPublicKey(pkey)
	hash := encryption.Hash(toN.PublicKeyBytes)
	if toN.ID != hash {
		return nil, common.NewError("invalid_client_id", fmt.Sprintf("public key: %v, client_id: %v, hash: %v\n", toN.PublicKey, toN.ID, hash))
	}
	toN.ComputeProperties()
	if Self.PublicKey == toN.PublicKey {
		setSelfNode(toN)
	}
	return toN, nil
}

// CopyGNode copy and initialize the node.
func CopyGNode(fromN *GNode) (*GNode, error) {
	toN := Provider()
	toN.Type = fromN.Type
	toN.Host = fromN.Host
	toN.N2NHost = fromN.N2NHost
	toN.Port = fromN.Port
	toN.SetID(fromN.ID)
	toN.PublicKey = fromN.PublicKey
	toN.Description = fromN.Description
	toN.Client.SetPublicKey(fromN.PublicKey)
	hash := encryption.Hash(toN.PublicKeyBytes)
	if toN.ID != hash {
		return nil, common.NewError("invalid_client_id", fmt.Sprintf("public key: %v, client_id: %v, hash: %v\n", toN.PublicKey, toN.ID, hash))
	}
	toN.ComputeProperties()
	if Self.PublicKey == toN.PublicKey {
		setSelfNode(toN)
	}
	return toN, nil
}

/*NewNode - read a node config line and create the node */
func NewNode(nc map[interface{}]interface{}) (*GNode, error) {
	gnode := Provider()
	gnode.Type = nc["type"].(int8)
	gnode.Host = nc["public_ip"].(string)
	gnode.N2NHost = nc["n2n_ip"].(string)
	gnode.Port = nc["port"].(int)
	gnode.SetID(nc["id"].(string))
	gnode.PublicKey = nc["public_key"].(string)

	if description, ok := nc["description"]; ok {
		gnode.Description = description.(string)
	} else {
		gnode.Description = gnode.GetNodeType() + gnode.GetKey()[:6]
	}

	gnode.Client.SetPublicKey(gnode.PublicKey)
	hash := encryption.Hash(gnode.PublicKeyBytes)
	if gnode.ID != hash {
		return nil, common.NewError("invalid_client_id", fmt.Sprintf("public key: %v, client_id: %v, hash: %v\n", gnode.PublicKey, gnode.ID, hash))
	}
	if shortName, ok := nc["short_name"]; ok {
		gnode.ShortName = shortName.(string)
	} else {
		gnode.ShortName = gnode.PublicKey
	}
	gnode.ComputeProperties()
	if Self.PublicKey == gnode.PublicKey {
		setSelfNode(gnode)
	}
	RegisterNode(gnode)
	return gnode, nil
}

func setSelfNode(n *GNode) {
	Self.GNode = n
	Self.GNode.Info.StateMissingNodes = -1
	Self.GNode.Info.BuildTag = build.BuildTag
	Self.GNode.Status = NodeStatusActive
}

/*ComputeProperties - implement entity interface */
func (n *GNode) ComputeProperties() {
	n.Client.ComputeProperties()
	if n.Host == "" {
		n.Host = "localhost"
	}
	if n.N2NHost == "" {
		n.N2NHost = n.Host
	}
	if n.ShortName == "" {
		n.ShortName = n.Host
	}
}

/*GetURLBase - get the end point base */
func (n *GNode) GetURLBase() string {
	return fmt.Sprintf("http://%v:%v", n.Host, n.Port)
}

/*GetN2NURLBase - get the end point base for n2n communication */
func (n *GNode) GetN2NURLBase() string {
	return fmt.Sprintf("http://%v:%v", n.N2NHost, n.Port)
}

/*GetStatusURL - get the end point where to ping for the status */
func (n *GNode) GetStatusURL() string {
	return fmt.Sprintf("%v/_nh/status", n.GetN2NURLBase())
}

/*GetNodeType - as a string */
func (n *GNode) GetNodeType() string {
	return NodeTypeNames[n.Type].Code
}

/*GetNodeTypeName - get the name of this node type */
func (n *GNode) GetNodeTypeName() string {
	return NodeTypeNames[n.Type].Value
}

//Grab - grab a slot to send message
func (n *GNode) Grab() {
	<-n.CommChannel
	n.Sent++
}

//Release - release a slot after sending the message
func (n *GNode) Release() {
	n.CommChannel <- true
}

//GetTimer - get the timer
func (n *GNode) GetTimer(uri string) metrics.Timer {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	timer, ok := n.TimersByURI[uri]
	if !ok {
		timerID := fmt.Sprintf("%v.%v.time", n.ID, uri)
		timer = metrics.GetOrRegisterTimer(timerID, nil)
		n.TimersByURI[uri] = timer
	}
	return timer
}

//GetSizeMetric - get the size metric
func (n *GNode) GetSizeMetric(uri string) metrics.Histogram {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	metric, ok := n.SizeByURI[uri]
	if !ok {
		metricID := fmt.Sprintf("%v.%v.size", n.ID, uri)
		metric = metrics.NewHistogram(metrics.NewUniformSample(256))
		n.SizeByURI[uri] = metric
		metrics.Register(metricID, metric)
	}
	return metric
}

//GetLargeMessageSendTime - get the time it takes to send a large message to this node
func (n *GNode) GetLargeMessageSendTime() float64 {
	return n.LargeMessageSendTime / 1000000
}

//GetSmallMessageSendTime - get the time it takes to send a small message to this node
func (n *GNode) GetSmallMessageSendTime() float64 {
	return n.SmallMessageSendTime / 1000000
}

func (n *GNode) updateMessageTimings() {
	n.updateSendMessageTimings()
	n.updateRequestMessageTimings()
}

func (n *GNode) updateSendMessageTimings() {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	var minval = math.MaxFloat64
	var maxval float64
	var maxCount int64
	for uri, timer := range n.TimersByURI {
		if timer.Count() == 0 {
			continue
		}
		if isGetRequest(uri) {
			continue
		}
		if sizer, ok := n.SizeByURI[uri]; ok {
			tv := timer.Mean()
			sv := sizer.Mean()
			sc := sizer.Count()
			if int(sv) < LargeMessageThreshold {
				if tv < minval {
					minval = tv
				}
			} else {
				if sc > maxCount {
					maxval = tv
					maxCount = sc
				}
			}
		}
	}
	if minval > maxval {
		if minval != math.MaxFloat64 {
			maxval = minval
		} else {
			minval = maxval
		}
	}
	n.LargeMessageSendTime = maxval
	n.SmallMessageSendTime = minval
}

func (n *GNode) updateRequestMessageTimings() {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	var minval = math.MaxFloat64
	var maxval float64
	var minSize = math.MaxFloat64
	var maxSize float64
	for uri, timer := range n.TimersByURI {
		if timer.Count() == 0 {
			continue
		}
		if !isGetRequest(uri) {
			continue
		}
		v := timer.Mean()
		if sizer, ok := n.SizeByURI[uri]; ok {
			if sizer.Mean() == 0 {
				continue
			}
			if sizer.Mean() > maxSize {
				maxSize = sizer.Mean()
				if v > maxval {
					maxval = v
				}
			}
			if sizer.Mean() < minSize {
				minSize = sizer.Mean()
				if v < minval {
					minval = v
				}
			}
		}
	}
	if minval > maxval {
		if minval != math.MaxFloat64 {
			maxval = minval
		} else {
			minval = maxval
		}
	}
	n.LargeMessagePullServeTime = maxval
	n.SmallMessagePullServeTime = minval
}

//ReadConfig - read configuration from the default config
func ReadConfig() {
	SetTimeoutSmallMessage(viper.GetDuration("network.timeout.small_message") * time.Millisecond)
	SetTimeoutLargeMessage(viper.GetDuration("network.timeout.large_message") * time.Millisecond)
	SetMaxConcurrentRequests(viper.GetInt("network.max_concurrent_requests"))
	SetLargeMessageThresholdSize(viper.GetInt("network.large_message_th_size"))
}

//SetID - set the id of the node
func (n *GNode) SetID(id string) error {
	n.ID = id
	bytes, err := hex.DecodeString(id)
	if err != nil {
		return err
	}
	n.idBytes = bytes
	return nil
}

//IsActive - returns if this node is active or not
func (n *GNode) IsActive() bool {
	return n.Status == NodeStatusActive
}

func serveMetricKey(uri string) string {
	return "p?" + uri
}

func isPullRequestURI(uri string) bool {
	return strings.HasPrefix(uri, "p?")
}

func isGetRequest(uri string) bool {
	if strings.HasPrefix(uri, "p?") {
		return true
	}
	return strings.HasSuffix(uri, "/get")
}

//GetPseudoName - create a pseudo name that is unique in the current active set
func (n *GNode) GetPseudoName() string {
	return fmt.Sprintf("%v-%v", n.GetNodeTypeName(), n.ShortName)
}

//GetOptimalLargeMessageSendTime - get the push or pull based optimal large message send time
func (n *GNode) GetOptimalLargeMessageSendTime() float64 {
	return n.getOptimalLargeMessageSendTime() / 1000000
}

func (n *GNode) getOptimalLargeMessageSendTime() float64 {
	p2ptime := getPushToPullTime(n)
	if p2ptime < n.LargeMessageSendTime {
		return p2ptime
	}
	if n.LargeMessageSendTime == 0 {
		return p2ptime
	}
	return n.LargeMessageSendTime
}

func (n *GNode) getTime(uri string) float64 {
	pullTimer := n.GetTimer(uri)
	return pullTimer.Mean()
}

func shuffleNodes() []*GNode {
	size := len(gnodes)
	if size == 0 {
		return nil
	}

	var array = make([]*GNode, 0, size)
	for _, v := range gnodes {
		array = append(array, v)
	}

	shuffled := make([]*GNode, size)
	perm := rand.Perm(size)
	for i, v := range perm {
		shuffled[v] = array[i]
	}
	return shuffled
}

//GNodeStatusMonitor monitors statuses of all the registered nodes
func GNodeStatusMonitor(ctx context.Context) {

	gnodesArr := shuffleNodes()
	for _, gnode := range gnodesArr {
		if gnode == Self.GNode {
			continue
		}

		if common.Within(gnode.LastActiveTime.Unix(), 10) {
			gnode.updateMessageTimings()
			if time.Since(gnode.Info.AsOf) < 60*time.Second {
				continue
			}
		}
		statusURL := gnode.GetStatusURL()

		ts := time.Now().UTC()
		data, hash, signature, err := Self.TimeStampSignature()
		if err != nil {
			panic(err)
		}
		statusURL = fmt.Sprintf("%v?id=%v&data=%v&hash=%v&signature=%v", statusURL, Self.GNode.GetKey(), data, hash, signature)

		resp, err := httpClient.Get(statusURL)
		if err != nil {
			gnode.ErrorCount++
			if gnode.IsActive() {
				if gnode.ErrorCount > 5 {
					gnode.Status = NodeStatusInactive
					N2n.Error("Node inactive", zap.String("node_type", gnode.GetNodeTypeName()), zap.String("shortName", gnode.GetPseudoName()), zap.Any("node_id", gnode.GetKey()), zap.Error(err))
				}
			}
		} else {
			if err := common.FromJSON(resp.Body, &gnode.Info); err == nil {
				gnode.Info.AsOf = time.Now()
			}
			resp.Body.Close()
			if !gnode.IsActive() {
				gnode.ErrorCount = 0
				gnode.Status = NodeStatusActive
				N2n.Info("Node active", zap.String("node_type", gnode.GetNodeTypeName()), zap.String("short_name", gnode.GetPseudoName()), zap.Any("key", gnode.GetKey()))
			}
			gnode.LastActiveTime = ts
		}
	}

}
