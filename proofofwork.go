package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
)

const targetBits = 24

var maxNonce = math.MaxInt64

// ProofofWork - used to calculate PoW
type ProofofWork struct {
	block  *Block
	target *big.Int
}

// NewProofOfWork - get the PoW for the block
func NewProofOfWork(b *Block) *ProofofWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))

	pow := &ProofofWork{
		block:  b,
		target: target,
	}

	return pow
}

func (pow *ProofofWork) prepareData(nonce int) []byte {
	data := bytes.Join([][]byte{
		pow.block.PrevBlockHash,
		pow.block.HashTransactions(),
		IntToHex(pow.block.Timestamp),
		IntToHex(int64(targetBits)),
		IntToHex(int64(nonce)),
	}, []byte{})

	return data
}

// Run - this is where the PoW computes the nonce
func (pow *ProofofWork) Run() (int, []byte) {
	var hashInt big.Int
	var hash [32]byte
	nonce := 0

	//fmt.Printf("Mining block with data : %s\n", pow.block.Data)
	for nonce < maxNonce {
		data := pow.prepareData(nonce)
		hash = sha256.Sum256(data)
		//fmt.Printf("Current hash: %x\n", hash)
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(pow.target) == -1 {
			break
		} else {
			nonce++
		}

	}

	fmt.Print("\nDone\n")

	return nonce, hash[:]
}

// Validate - ensure that the nonce is correct
func (pow *ProofofWork) Validate() bool {
	var hashInt big.Int

	data := pow.prepareData(pow.block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])

	isValid := hashInt.Cmp(pow.target) == -1

	return isValid
}
