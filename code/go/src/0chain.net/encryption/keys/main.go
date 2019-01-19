package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"0chain.net/common"
	"0chain.net/encryption"
)

func main() {
	clientSigScheme := flag.String("signature_scheme", "", "ed25519 or bls0chain")
	keysFile := flag.String("keys_file", "keys.txt", "keys_file")
	data := flag.String("data", "", "data")
	timestamp := flag.Bool("timestamp", true, "timestamp")
	generateKeys := flag.Bool("generate_keys", false, "generate_keys")
	flag.Parse()
	fmt.Printf("clientSigScheme: %v\n", *clientSigScheme)
	fmt.Printf("keys file: %v\n", *keysFile)
	fmt.Printf("data: %v\n", *data)
	fmt.Printf("timestamp: %v\n", *timestamp)
	fmt.Printf("generateKeys: %v\n", *generateKeys)

	var sigScheme = encryption.GetSignatureScheme(*clientSigScheme)
	if *generateKeys {
		err := sigScheme.GenerateKeys()
		if err != nil {
			panic(err)
		}
		if len(*keysFile) > 0 {
			writer, err := os.OpenFile(*keysFile, os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				panic(err)
			}
			defer writer.Close()
			sigScheme.WriteKeys(writer)
		} else {
			sigScheme.WriteKeys(os.Stdout)
		}
	} else {
		fmt.Printf("Did not generate Keys")
	}
	if len(*keysFile) == 0 {
		return
	}
	reader, err := os.Open(*keysFile)
	if err != nil {
		panic(err)
	}
	_, publicKey, privateKey := encryption.ReadKeys(reader)
	pubKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		panic(err)
	}
	clientID := encryption.Hash(pubKeyBytes)
	reader.Close()
	time := common.Now()
	fmt.Printf("data: %v\n", *data)
	fmt.Printf("keys file: %v\n", *keysFile)
	fmt.Printf("public_key: %v\n", publicKey)
	fmt.Printf("timestamp: %v\n", time)
	fmt.Printf("client_id: %v\n", clientID)
	var hashdata string
	if *timestamp {
		hashdata = fmt.Sprintf("%v:%v:%v\n", clientID, time, *data)
	} else {
		hashdata = fmt.Sprintf("%v:%v\n", clientID, *data)
	}
	fmt.Printf("privateKey: %v\n", privateKey)
	fmt.Printf("hashdata: %v", hashdata)
	hash := encryption.Hash(hashdata)
	fmt.Printf("hash: %v\n", hash)
	sign, err := encryption.Sign(privateKey, hash)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	} else {
		fmt.Printf("signature:%v\n", sign)
	}
}
