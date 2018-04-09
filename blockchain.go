package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/boltdb/bolt"
)

const (
	dbFile              = "blockchain_%s.db"
	blocksBucket        = "blocks"
	genesisCoinbaseData = "The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"
)

// Blockchain - the star of the show
type Blockchain struct {
	tip []byte
	db  *bolt.DB
}

// BlockchainIterator - used to iterate over the blocks
type BlockchainIterator struct {
	currentHash []byte
	db          *bolt.DB
}

// Iterator - return an iterator for a blockchain
func (bc *Blockchain) Iterator() *BlockchainIterator {
	bci := &BlockchainIterator{
		currentHash: bc.tip,
		db:          bc.db,
	}
	return bci
}

// Next - the blockchain iterator returns the next block
func (bci *BlockchainIterator) Next() *Block {
	var block *Block

	err := bci.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(bci.currentHash)
		block = DeserializeBlock(encodedBlock)

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	bci.currentHash = block.PrevBlockHash

	return block
}

// AddBlock - add a block to the blockchain
func (bc *Blockchain) AddBlock(block *Block) {

	err := bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		blockInDb := b.Get(block.Hash)

		if blockInDb == nil {
			return nil
		}

		blockData := block.Serialize()
		err := b.Put(block.Hash, blockData)

		if err != nil {
			log.Panic(err)
		}

		lastHash := b.Get([]byte("1"))
		lastBlockData := b.Get(lastHash)
		lastBlock := DeserializeBlock(lastBlockData)

		if block.Height > lastBlock.Height {
			err = b.Put([]byte("1"), block.Hash)
			if err != nil {
				log.Panic(err)
			}
			bc.tip = block.Hash
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}

// NewBlockchain - start a new blockchain
func NewBlockchain(nodeID string) *Blockchain {
	//return &Blockchain{[]*Block{NewGenesisBlock()}}
	dbFile := fmt.Sprintf(dbFile, nodeID)
	if !dbExists(dbFile) {
		fmt.Println("no existing blockchain found. Please create one.")
		os.Exit(1)
	}

	var tip []byte

	db, err := bolt.Open(dbFile, 0600, nil)

	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		tip = b.Get([]byte("1"))

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	bc := Blockchain{
		tip: tip,
		db:  db,
	}

	return &bc
}

func dbExists(fName string) bool {
	if _, err := os.Stat(fName); os.IsNotExist(err) {
		return false
	}
	return true
}

// CreateBlockchain - creates a new blockchain db
func CreateBlockchain(address, nodeID string) *Blockchain {
	dbFile := fmt.Sprintf(dbFile, nodeID)
	if dbExists(dbFile) {
		fmt.Println("DB exists.")
		os.Exit(1)
	}

	var tip []byte

	cbtx := NewCoinbaseTX(address, genesisCoinbaseData)
	genesis := NewGenesisBlock(cbtx)

	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {

		b, err := tx.CreateBucket([]byte(blocksBucket))
		if err != nil {
			log.Panic(err)
		}

		err = b.Put(genesis.Hash, genesis.Serialize())
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte("1"), genesis.Hash)
		if err != nil {
			log.Panic(err)
		}

		tip = genesis.Hash

		return nil
	})

	bc := Blockchain{
		tip: tip,
		db:  db,
	}

	return &bc
}

// GetBestHeight - returns the height of the latest block
func (bc *Blockchain) GetBestHeight() int {
	var lastBlock Block

	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash := b.Get([]byte("1"))
		lastBlockData := b.Get(lastHash)
		lastBlock = *DeserializeBlock(lastBlockData)

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return lastBlock.Height
}

// GetBlock - returns the block for the specified hash
func (bc *Blockchain) GetBlock(blockHash []byte) (Block, error) {
	var block Block

	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		blockData := b.Get(blockHash)
		block = *DeserializeBlock(blockData)

		return nil
	})
	if err != nil {
		return block, err
	}

	return block, nil
}

// GetBlockHashes - return the hashes of the blocks
func (bc *Blockchain) GetBlockHashes() [][]byte {
	var hashes [][]byte

	bci := bc.Iterator()
	for {
		block := bci.Next()
		hashes = append(hashes, block.Hash)
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return hashes
}

// FindUnspentTransactions - get the unspent transactions for specific address
func (bc *Blockchain) FindUnspentTransactions(pubKeyHash []byte) []Transaction {
	var unspentTXs []Transaction

	spentTXOs := make(map[string][]int)
	bci := bc.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Vout {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}
				if out.IsLockedWithKey(pubKeyHash) {
					unspentTXs = append(unspentTXs, *tx)
				}
			}

			if !tx.IsCoinbase() {
				for _, tin := range tx.Vin {
					if tin.UsesKey(pubKeyHash) {
						inTXID := hex.EncodeToString(tin.Txid)
						spentTXOs[inTXID] = append(spentTXOs[inTXID], tin.Vout)
					}
				}
			}
		}
		if len(block.PrevBlockHash) == 0 {
			break
		}

	}
	return unspentTXs
}

// FindUTXO - finds all unspent output txns and returns with spent txns removed
func (bc *Blockchain) FindUTXO() map[string]TXOutputs {
	UTXO := make(map[string]TXOutputs)
	spentTXOs := make(map[string][]int)
	bci := bc.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Vout {
				if spentTXOs[txID] != nil {
					for _, spentOutIdx := range spentTXOs[txID] {
						if spentOutIdx == outIdx {
							continue Outputs
						}
					}
				}

				outs := UTXO[txID]
				outs.Outputs = append(outs.Outputs, out)
				UTXO[txID] = outs
			}

			if !tx.IsCoinbase() {
				for _, in := range tx.Vin {
					inTxID := hex.EncodeToString(in.Txid)
					spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout)
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return UTXO
}

// FindSpendableOutputs - find spendable outputs for an address
func (bc *Blockchain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTXs := bc.FindUnspentTransactions(pubKeyHash)
	accumulated := 0

Work:
	for _, tx := range unspentTXs {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Vout {
			if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
				accumulated = accumulated + out.Value
				unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)

				if accumulated >= amount {
					break Work
				}

			}
		}
	}

	return accumulated, unspentOutputs
}

// FindTransaction - get the transaction with the ID specified
func (bc *Blockchain) FindTransaction(ID []byte) (Transaction, error) {
	bci := bc.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			if bytes.Compare(ID, tx.ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("transaction not found")
}

// MineBlock - mine a block for the blockchain
func (bc *Blockchain) MineBlock(transactions []*Transaction) *Block {
	var lastHash []byte
	var lastHeight int

	for _, tx := range transactions {
		if !bc.VerifyTransaction(tx) {
			log.Panic("err : invalid transaction")
		}
	}

	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		lastHash = b.Get([]byte("1"))
		lastBlockData := b.Get(lastHash)
		lastBlock := DeserializeBlock(lastBlockData)

		lastHeight = lastBlock.Height

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	newBlock := NewBlock(transactions, lastHash, lastHeight+1)

	err = bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		err := b.Put(newBlock.Hash, newBlock.Serialize())
		if err != nil {
			log.Panic(err)
		}
		err = b.Put([]byte("1"), newBlock.Hash)
		if err != nil {
			log.Panic(err)
		}
		bc.tip = newBlock.Hash
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	return newBlock
}

// SignTransaction - signs input of a transaction
func (bc *Blockchain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransaction(vin.Txid)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	tx.Sign(privKey, prevTXs)
}

// VerifyTransaction - verify the input signatures of a transaction
func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool {
	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransaction(vin.Txid)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs)
}
