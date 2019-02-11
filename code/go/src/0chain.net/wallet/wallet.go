package wallet

import (
	"encoding/hex"
	"os"

	"0chain.net/encryption"
)

/*Wallet - a struct representing the client's wallet */
type Wallet struct {
	SignatureScheme encryption.SignatureScheme
	PublicKeyBytes  []byte
	ClientID        string
	Balance         int64
}

/*Initialize - initialize a wallet with public/private keys */
func (w *Wallet) Initialize(clientSignatureScheme string) error {
	var sigScheme encryption.SignatureScheme = encryption.GetSignatureScheme(clientSignatureScheme)
	err := sigScheme.GenerateKeys()
	if err != nil {
		return err
	}
	return w.SetSignatureScheme(sigScheme)
}

func (w *Wallet) GetOwnerWallet(c *Cluster) {
	var keysFile string
	if c.ClientSignatureScheme == "ed25519" {
		keysFile = "config/owner_keys.txt"
	} else {
		keysFile = "config/b0owner_keys.txt"
	}
	reader, err := os.Open(keysFile)
	if err != nil {
		panic(err)
	}
	sigScheme := encryption.GetSignatureScheme(c.ClientSignatureScheme)
	err = sigScheme.ReadKeys(reader)
	if err != nil {
		panic(err)
	}
	err = w.SetSignatureScheme(sigScheme)
	if err != nil {
		panic(err)
	}
}

/*SetSignatureScheme - sets the keys for the wallet */
func (w *Wallet) SetSignatureScheme(signatureScheme encryption.SignatureScheme) error {
	w.SignatureScheme = signatureScheme
	publicKeyBytes, err := hex.DecodeString(signatureScheme.GetPublicKey())
	if err != nil {
		return err
	}
	w.ClientID = encryption.Hash(publicKeyBytes)
	return nil
}
