package discovery

import (
	"fmt"
	"github.com/asaskevich/govalidator"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	. "0chain.net/core/logging"
	"net/url"
	"time"
)

func SetDefaultConfig() {
	viper.SetDefault("discovery.enabled", false)
	viper.SetDefault("discovery.endpoint", "")
	viper.SetDefault("discovery.nodes_file", "nodes.yml")

}

//type Discovery interface {
//	SetConfig()
//	Valid() bool
//	NodeLocation() string
//	EndPointURL() *url.URL
//	PostMagicBlock() *url.URL
//}

type Discovery struct {
	isEnabled bool
	isValid bool

	PropagateMagicBlockSecs int `json:update_magic_block_secs`
	PropagateMagicBlock time.Duration

	MonitorViewChangeSecs int `json:check_view_change_secs`
	MonitorViewChange time.Duration

	EndPoint string
	Bucket   string
	Chain    string


	nodesFile string
	nodeLocation string

	EndPointURL url.URL

	MagicBlockURL url.URL
}

var Control *Discovery

func init() {
	Control = &Discovery{isEnabled:false, isValid:false, nodesFile: "nodes.yml"}
}

func (dc *Discovery)ShowConfig() {

	Logger.Info("DS-Config",
		zap.Bool("enabled", dc.isEnabled),
		zap.Bool("valid", dc.isValid),
		zap.Any("Endpoint", dc.EndPointURL),
		zap.Any("MagicBlock", dc.MagicBlockURL),
		zap.String("nodes_file", dc.nodesFile),
		zap.String("node_url", dc.nodeLocation))

	Logger.Info("DS-Config",
		zap.Int("monitor (secs)", dc.MonitorViewChangeSecs),
		zap.Int("propagate (secs)", dc.PropagateMagicBlockSecs))

}
func (dc *Discovery) SetConfig () {
	dc.isEnabled = viper.GetBool("discovery.enabled")

	// Update the endpoint, bucket and chain.
	dc.EndPoint = viper.GetString("discovery.endpoint")
	dc.Bucket = viper.GetString("discovery.bucket")
	dc.Chain = viper.GetString("discovery.chain")

	// Configure magic block update times.
	dc.PropagateMagicBlockSecs = viper.GetInt("discovery.propagate_magicblock_secs")
	dc.PropagateMagicBlock = time.Duration(dc.PropagateMagicBlockSecs) * time.Second

	// Frequent - Check for viewchanges
	dc.MonitorViewChangeSecs = viper.GetInt("discovery.monitor_viewchange_secs")
	dc.MonitorViewChange = time.Duration(dc.MonitorViewChangeSecs) * time.Second

	dc.nodesFile = viper.GetString("discovery.nodes_file")

	if govalidator.IsURL(dc.EndPoint) {
		// Create magic block url.
		magicBlockURL, err := url.Parse(dc.EndPoint)
		if err == nil {
			dc.MagicBlockURL = *magicBlockURL
			if dc.MagicBlockURL.Scheme == "" {
				dc.MagicBlockURL.Scheme = "http"
			}
		}
		dc.nodeLocation = fmt.Sprintf("http://%v/%v", dc.EndPoint, dc.nodesFile)
		dc.isValid = true
	}

	// Show the discovery configuration
	dc.ShowConfig()

}

func(dc *Discovery) NodeLocation() string {
	return dc.nodeLocation
}

//func (dc *Discovery) EndPointURL() *url.URL {
//	return dc.endPointURL
//}

//func (dc *Discovery) PostMagicBlock() *url.URL {
//	return dc.postMagicBlock
//}

func(dc *Discovery) Valid() bool {
	return dc.isEnabled && dc.isValid
}

func(dc *Discovery) Download() {

}
//func (dc *discovery) EnabledStatus() bool {
//	return dc.Enabled
//}

