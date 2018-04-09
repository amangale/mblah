package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"strconv"
	"time"
)

// Block - the block
type Block struct {
	Timestamp     int64
	Nonce         int
	Transactions  []*Transaction
	PrevBlockHash []byte
	Hash          []byte
	Height        int
	//Data          []byte
}

// Serialize - used to store in BoltDB
func (b *Block) Serialize() []byte {
	var result bytes.Buffer

	encoder := gob.NewEncoder(&result)

	err := encoder.Encode(b)

	if err != nil {
		log.Panic(err)
	}

	return result.Bytes()
}

// DeserializeBlock - returns a block previously serialized
func DeserializeBlock(d []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(d))

	err := decoder.Decode(&block)

	if err != nil {
		log.Printf("passed value is not a block: %v\n", err)
		return nil
	}

	return &block
}

// HashTransactions - hash the included transactions
func (b *Block) HashTransactions() []byte {
	//var txHashes [][]byte
	//var txHash [32]byte

	var transactions [][]byte

	for _, tx := range b.Transactions {
		transactions = append(transactions, tx.Serialize())
	}

	mTree := NewMerkleTree(transactions)

	//txHash = sha256.Sum256(bytes.Join(txHashes, []byte{}))

	return mTree.RootNode.Data
}

// SetHash - set the block hash
func (b *Block) SetHash() {
	timestamp := []byte(strconv.FormatInt(b.Timestamp, 10))
	headers := bytes.Join([][]byte{b.PrevBlockHash, b.HashTransactions(), timestamp}, []byte{})
	hash := sha256.Sum256(headers)
	b.Hash = hash[:]
}

// NewBlock - return a new block
func NewBlock(transactions []*Transaction, prevBlockHash []byte, height int) *Block {
	block := &Block{
		Timestamp:     time.Now().Unix(),
		Transactions:  transactions,
		PrevBlockHash: prevBlockHash,
		Height:        height,
	}
	//block.SetHash()

	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

// NewGenesisBlock - get the one that starts it all
func NewGenesisBlock(coinbase *Transaction) *Block {
	return NewBlock([]*Transaction{coinbase}, []byte{}, 0)
}
