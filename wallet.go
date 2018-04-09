package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"log"

	"golang.org/x/crypto/ripemd160"
)

const (
	addressChecksumLen = 4
	version            = byte(0x00)
	walletFile         = "wallet_%s.dat"
)

// Wallet - represents a wallet
type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

// NewWallet - get a new Wallet
func NewWallet() *Wallet {
	private, public := newKeyPair()
	wallet := Wallet{
		PrivateKey: private,
		PublicKey:  public,
	}
	return &wallet
}

func newKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}
	pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	return *private, pubKey
}

// GetAddress - fetch address from a wallet
func (w *Wallet) GetAddress() []byte {
	pubKeyHash := HashPubKey(w.PublicKey)

	versionedPayload := append([]byte{version}, pubKeyHash...)
	checksum := checksum(versionedPayload)

	fullPayload := append(versionedPayload, checksum...)
	address := Base58Encode(fullPayload)

	return address
}

// ValidateAddress - used to validate address
func ValidateAddress(address string) bool {
	pubKeyHash := Base58Decode([]byte(address))

	actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:]
	version := pubKeyHash[0]

	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-addressChecksumLen]
	targetChecksum := checksum(append([]byte{version}, pubKeyHash...))

	return bytes.Compare(actualChecksum, targetChecksum) == 0
}

// HashPubKey - hash the public key
func HashPubKey(pubKey []byte) []byte {
	publicSHA256 := sha256.Sum256(pubKey)

	RIPEMD160Hasher := ripemd160.New()
	_, err := RIPEMD160Hasher.Write(publicSHA256[:])
	if err != nil {
		log.Panic(err)
	}
	publicRIPEMD160 := RIPEMD160Hasher.Sum(nil)

	return publicRIPEMD160
}

func checksum(payload []byte) []byte {
	first := sha256.Sum256(payload)
	second := sha256.Sum256(first[:])

	return second[:addressChecksumLen]
}
