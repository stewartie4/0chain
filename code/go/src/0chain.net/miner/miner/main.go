package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"0chain.net/miner"
	"0chain.net/threshold/bls"

	_ "net/http/pprof"

	"0chain.net/block"
	"0chain.net/chain"
	"0chain.net/client"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/diagnostics"
	"0chain.net/encryption"
	"0chain.net/logging"
	. "0chain.net/logging"
	"0chain.net/memorystore"
	"0chain.net/node"
	"0chain.net/round"
	"0chain.net/state"
	"0chain.net/transaction"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const REGISTER_CLIENT = "v1/client/put"
const MAX_TXN_RETRIES = 5

const SLEEP_BETWEEN_RETRIES = 5
const SLEEP_FOR_TXN_CONFIRMATION = 5

func main() {
	deploymentMode := flag.Int("deployment_mode", 2, "deployment_mode")
	nodesFile := flag.String("nodes_file", "config/single_node.txt", "nodes_file")         
	keysFile := flag.String("keys_file", "config/single_node_miner_keys.txt", "keys_file") 
	maxDelay := flag.Int("max_delay", 0, "max_delay")
	nongenesis := flag.Bool("non_genesis", false, "non_genesis")
	flag.Parse()

	config.Configuration.DeploymentMode = byte(*deploymentMode)
	config.SetupDefaultConfig()
	config.SetupConfig()

	if config.Development() {
		logging.InitLogging("development")
	} else {
		logging.InitLogging("production")
	}
	config.Configuration.ChainID = viper.GetString("server_chain.id")
	config.Configuration.MaxDelay = *maxDelay
	transaction.SetTxnTimeout(int64(viper.GetInt("server_chain.transaction.timeout")))

	reader, err := os.Open(*keysFile)
	if err != nil {
		panic(err)
	}
	signatureScheme := encryption.NewED25519Scheme()
	err = signatureScheme.ReadKeys(reader)
	if err != nil {
		Logger.Panic("Error reading keys file")
	}
	node.Self.SetSignatureScheme(signatureScheme)
	reader.Close()

	// set the chain this server is responsible for processing
	config.SetServerChainID(config.Configuration.ChainID)
	common.SetupRootContext(node.GetNodeContext())
	ctx := common.GetRootContext()
	initEntities()
	serverChain := chain.NewChainFromConfig()
	miner.SetupMinerChain(serverChain)
	mc := miner.GetMinerChain()
	mc.DiscoverClients = viper.GetBool("server_chain.client.discover")
	mc.SetGenerationTimeout(viper.GetInt("server_chain.block.generation.timeout"))
	mc.SetRetryWaitTime(viper.GetInt("server_chain.block.generation.retry_wait_time"))
	chain.SetServerChain(serverChain)

	miner.SetNetworkRelayTime(viper.GetDuration("network.relay_time") * time.Millisecond)
	node.ReadConfig()
	if *nodesFile == "" {
		panic("Please specify --nodes_file file.txt option with a file.txt containing nodes including self")
	}
	if strings.HasSuffix(*nodesFile, "txt") {
		reader, err := os.Open(*nodesFile)
		if err != nil {
			log.Fatalf("%v", err)
		}
		node.ReadNodes(reader, serverChain.Miners, serverChain.Sharders, serverChain.Blobbers)
		reader.Close()
	} else {
		mc.ReadNodePools(*nodesFile)
	}
	if node.Self.ID == "" {
		Logger.Panic("node definition for self node doesn't exist")
	}

	if state.Debug() {
		chain.SetupStateLogger("/tmp/state.txt")
	}
	if *nongenesis {
		//////////// NON-GENESIS Miner ///////////////////////////
		// node.Host , node.Port, node.SetID, node.Self.PublicKey
		reader, err := os.Open(*keysFile)
		if err != nil {
			panic(err)
		}
		scanner := bufio.NewScanner(reader)
		scanner.Scan()
		node.Self.PublicKey = scanner.Text()
		scanner.Scan()
		// privateKey = scanner.Text()
		scanner.Scan()

		node.Self.Host = scanner.Text()
		scanner.Scan()
		port, _ := strconv.ParseInt(scanner.Text(), 10, 32)
		node.Self.Port = int(port)
		reader.Close()
		// node.Self.signatureScheme
	}

	mode := "main net"
	if config.Development() {
		mode = "development"
	} else if config.TestNet() {
		mode = "test net"
	}
	address := fmt.Sprintf(":%v", node.Self.Port)

	if *nongenesis {
		Logger.Info("non-genesis : ", zap.Bool("non-genesis", *nongenesis))
		RegisterMiner(ctx, serverChain)
	}

	Logger.Info("Starting miner", zap.String("go_version", runtime.Version()), zap.Int("available_cpus", runtime.NumCPU()), zap.String("port", address))
	Logger.Info("Chain info", zap.String("chain_id", config.GetServerChainID()), zap.String("mode", mode))
	Logger.Info("Self identity", zap.Any("set_index", node.Self.Node.SetIndex), zap.Any("id", node.Self.Node.GetKey()))

	//TODO - get stake of miner from biding (currently hard coded)
	//serverChain.updateMiningStake(node.Self.Node.GetKey(), 100)  we do not want to expose this feature at this point.
	var server *http.Server

	if config.Development() {
		// No WriteTimeout setup to enable pprof
		server = &http.Server{
			Addr:           address,
			ReadTimeout:    30 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
	} else {
		server = &http.Server{
			Addr:           address,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
	}
	common.HandleShutdown(server)
	memorystore.GetInfo()
	initWorkers(ctx)

	mc.SetupGenesisBlock(viper.GetString("server_chain.genesis_block.id"))

	initN2NHandlers()

	initServer()
	initHandlers()

	chain.StartTime = time.Now().UTC()
	go func() {
		miner.StartDKG(ctx)
		if config.Development() {
			go TransactionGenerator(mc.BlockSize) // wallet code
		}
	}()

	Logger.Info("Ready to listen to the requests")
	log.Fatal(server.ListenAndServe())
}

func initServer() {
	/* TODO: when a new server is brought up, it needs to first download
	all the state before it can start accepting requests
	*/
	time.Sleep(time.Second)
}

func initEntities() {
	memoryStorage := memorystore.GetStorageProvider()

	chain.SetupEntity(memoryStorage)
	round.SetupEntity(memoryStorage)
	round.SetupVRFShareEntity(memoryStorage)
	block.SetupEntity(memoryStorage)
	block.SetupBlockSummaryEntity(memoryStorage)
	block.SetupStateChange(memoryStorage)

	client.SetupEntity(memoryStorage)

	transaction.SetupTransactionDB()
	transaction.SetupEntity(memoryStorage)

	miner.SetupNotarizationEntity()

	bls.SetupDKGEntity()
	bls.SetupBLSEntity()
}

func initHandlers() {
	SetupHandlers()
	config.SetupHandlers()
	node.SetupHandlers()
	chain.SetupHandlers()
	client.SetupHandlers()
	transaction.SetupHandlers()
	block.SetupHandlers()
	miner.SetupHandlers()
	diagnostics.SetupHandlers()
	chain.SetupStateHandlers()

	serverChain := chain.GetServerChain()
	serverChain.SetupNodeHandlers()
}

func initN2NHandlers() {
	node.SetupN2NHandlers()
	miner.SetupM2MReceivers()
	miner.SetupM2MSenders()
	miner.SetupM2SSenders()
	miner.SetupM2SRequestors()

	miner.SetupX2MResponders()
	chain.SetupX2MRequestors()
}

func initWorkers(ctx context.Context) {
	serverChain := chain.GetServerChain()
	serverChain.SetupWorkers(ctx)
	miner.SetupWorkers(ctx)
	transaction.SetupWorkers(ctx)
}

func SendPostRequestSync(relativeURL string, data []byte, chain *chain.Chain) {
	wg := sync.WaitGroup{}
	wg.Add(chain.Miners.Size())
	// Get miners
	miners := chain.Miners.GetRandomNodes(chain.Miners.Size())
	for _, miner := range miners {
		url := fmt.Sprintf("%v/%v", miner.GetURLBase(), relativeURL)
		Logger.Info("Ready to send new request to ", zap.String("url", url))
		go sendPostRequest(url, data, &wg)
	}
	wg.Wait()
}

func sendPostRequest(url string, data []byte, wg *sync.WaitGroup) ([]byte, error) {
	if wg != nil {
		defer wg.Done()
	}
	req, ctx, cncl, err := NewHTTPRequest(http.MethodPost, url, data)
	defer cncl()
	var resp *http.Response
	for i := 0; i < MAX_TXN_RETRIES; i++ {
		resp, err = http.DefaultClient.Do(req.WithContext(ctx))
		if err == nil {
			break
		}
		//TODO: Handle ctx cncl
		Logger.Error("SendPostRequest Error", zap.String("error", err.Error()), zap.String("URL", url))
		time.Sleep(SLEEP_BETWEEN_RETRIES * time.Second)
	}
	if err != nil {
		Logger.Error("Failed after multiple retries", zap.Int("retried", MAX_TXN_RETRIES))
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	Logger.Info("SendPostRequest success", zap.String("url", url))
	return body, nil
}

func NewHTTPRequest(method string, url string, data []byte) (*http.Request, context.Context, context.CancelFunc, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Access-Control-Allow-Origin", "*")
	ctx, cncl := context.WithTimeout(context.Background(), time.Second*10)
	return req, ctx, cncl, err
}

// func RegisterMiner(ctx context.Context) (string, error) {
func RegisterMiner(ctx context.Context, chain *chain.Chain) {
	nodeBytes, _ := json.Marshal(node.Self.Node)
	SendPostRequestSync(REGISTER_CLIENT, nodeBytes, chain)
	// time.Sleep(transaction.SLEEP_FOR_TXN_CONFIRMATION * time.Second)

	// txn := transaction.NewTransactionEntity()
	//txn := "yes"

	// sn := &transaction.StorageNode{}
	// sn.ID = node.Self.GetKey()
	// sn.BaseURL = node.Self.GetURLBase()

	// scData := &transaction.SmartContractTxnData{}
	// scData.Name = transaction.ADD_BLOBBER_SC_NAME
	// scData.InputArgs = sn

	// txn.ToClientID = transaction.STORAGE_CONTRACT_ADDRESS
	// txn.Value = 0
	// txn.TransactionType = transaction.TxnTypeSmartContract
	// txnBytes, err := json.Marshal(scData)
	// if err != nil {
	// 	return "", err
	// }
	// txn.TransactionData = string(txnBytes)

	// err = txn.ComputeHashAndSign()
	// if err != nil {
	// 	Logger.Info("Signing Failed during registering blobber to the mining network", zap.String("err:", err.Error()))
	// 	return "", err
	// }
	// Logger.Info("Adding blobber to the blockchain.", zap.String("txn", txn.Hash))
	// transaction.SendTransaction(txn, sp.ServerChain)
	// return txn.Hash, nil
	//return txn, nil
}
