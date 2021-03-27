package encryption

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/0chain/gosdk/bls"
	"github.com/0chain/gosdk/miracl"
)

var GenG2 *bls.G2

func init() {
	err := bls.Init()
	if err != nil {
		panic(err)
	}
	GenG2 = BN254.ECP2_generator()
}

//BLS0ChainScheme - a signature scheme for BLS0Chain Signature
type BLS0ChainScheme struct {
	privateKey string
	publicKey  string
}

//NewBLS0ChainScheme - create a BLS0ChainScheme object
func NewBLS0ChainScheme() *BLS0ChainScheme {
	return &BLS0ChainScheme{}
}

//GenerateKeys - implement interface
func (b0 *BLS0ChainScheme) GenerateKeys() error {
	var skey bls.SecretKey
	skey.SetByCSPRNG()
	b0.privateKey = skey.SerializeToHexStr()
	b0.publicKey = skey.GetPublicKey().SerializeToHexStr()
	return nil
}

//ReadKeys - implement interface
func (b0 *BLS0ChainScheme) ReadKeys(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	result := scanner.Scan()
	if result == false {
		return ErrKeyRead
	}
	publicKey := scanner.Text()
	b0.SetPublicKey(publicKey)
	result = scanner.Scan()
	if result == false {
		return ErrKeyRead
	}
	b0.privateKey = scanner.Text()
	return nil
}

//WriteKeys - implement interface
func (b0 *BLS0ChainScheme) WriteKeys(writer io.Writer) error {
	_, err := fmt.Fprintf(writer, "%v\n%v\n", b0.publicKey, b0.privateKey)
	return err
}

//SetPublicKey - implement interface
func (b0 *BLS0ChainScheme) SetPublicKey(publicKey string) error {
	if len(b0.privateKey) > 0 {
		return errors.New("cannot set public key when there is a private key")
	}
	b0.publicKey = publicKey
	return nil
}

//GetPublicKey - implement interface
func (b0 *BLS0ChainScheme) GetPublicKey() string {
	return b0.publicKey
}

//Sign - implement interface
func (b0 *BLS0ChainScheme) Sign(hash interface{}) (string, error) {
	var sk bls.SecretKey
	sk.DeserializeHexStr(b0.privateKey)
	rawHash, err := GetRawHash(hash)
	if err != nil {
		return "", err
	}
	sig := sk.Sign(rawHash)
	return sig.SerializeToHexStr(), nil
}

//Verify - implement interface
func (b0 *BLS0ChainScheme) Verify(signature string, hash string) (bool, error) {
	pk, err := b0.getPublicKey()
	if err != nil {
		return false, err
	}
	sign, err := b0.GetSignature(signature)
	if err != nil {
		return false, err
	}
	rawHash, err := hex.DecodeString(hash)
	if err != nil {
		return false, err
	}
	return sign.Verify(pk, rawHash), nil
}

//GetSignature - given a string return the signature object
func (b0 *BLS0ChainScheme) GetSignature(signature string) (*bls.Sign, error) {
	if signature == "" {
		return nil, errors.New("empty signature")
	}
	var sign bls.Sign
	err := sign.DeserializeHexStr(signature)
	if err != nil {
		return nil, err
	}
	return &sign, nil
}

func (b0 *BLS0ChainScheme) getPublicKey() (*bls.PublicKey, error) {
	var pk = &bls.PublicKey{}
	err := pk.DeserializeHexStr(b0.publicKey)
	if err != nil {
		return nil, err
	}
	return pk, nil
}

//PairMessageHash - Pair a given message hash
func (b0 *BLS0ChainScheme) PairMessageHash(hash string) (*bls.GT, error) {
	var g2 = &bls.PublicKey{}
	err := g2.DeserializeHexStr(b0.publicKey)
	if err != nil {
		return nil, err
	}

	rawHash, err := hex.DecodeString(hash)
	g1 := bls.HashAndMapTo(rawHash)
	gt := bls.Pairing(g2.GetECP2(), g1)
	return gt, nil
}

//GenerateSplitKeys - implement interface
func (b0 *BLS0ChainScheme) GenerateSplitKeys(numSplits int) ([]SignatureScheme, error) {
	var primarySk bls.SecretKey
	primarySk.DeserializeHexStr(b0.privateKey)

	splitKeys := make([]SignatureScheme, numSplits)
	sk := bls.NewSecretKey()

	//Generate all but one split keys and add the secret keys
	for i := 0; i < numSplits-1; i++ {
		key := NewBLS0ChainScheme()
		key.GenerateKeys()
		splitKeys[i] = key
		var sk2 bls.SecretKey
		sk2.DeserializeHexStr(key.privateKey)
		sk.Add(&sk2)
	}

	var aggregateSk bls.SecretKey
	aggregateSk.DeserializeHexStr(sk.SerializeToHexStr())

	lastSk := primarySk.GetFP()
	lastSk.Sub(aggregateSk.GetFP())

	// Last key
	lastKey := NewBLS0ChainScheme()
	lastSecretKey := bls.SecretKey_fromFP(lastSk)
	lastKey.privateKey = lastSecretKey.SerializeToHexStr()
	lastKey.publicKey = lastSecretKey.GetPublicKey().SerializeToHexStr()
	splitKeys[numSplits-1] = lastKey
	return splitKeys, nil
}

//AggregateSignatures - implement interface
func (b0 *BLS0ChainScheme) AggregateSignatures(signatures []string) (string, error) {
	var aggSign bls.Sign
	for _, signature := range signatures {
		var sign bls.Sign
		sign.DeserializeHexStr(signature)
		aggSign.Add(&sign)
	}
	return aggSign.SerializeToHexStr(), nil
}
