package main

import (
	"encoding/hex"
	"log"

	"github.com/boltdb/bolt"
)

const (
	utxoBucket = "chainstate"
)

// UTXOSet - set of UTXOs
type UTXOSet struct {
	Blockchain *Blockchain
}

// FindSpendableOutputs - collects unspent outputs to reference in input
func (u *UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)

	accumulated := 0
	db := u.Blockchain.db

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			txID := hex.EncodeToString(k)
			outs := DeSerializeOutputs(v)

			for outIdx, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
					accumulated = accumulated + out.Value
					unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return accumulated, unspentOutputs
}

// FindUTXO - find the UTXO for a public key hash
func (u *UTXOSet) FindUTXO(pubKeyHash []byte) []TXOutput {
	var UTXOs []TXOutput
	db := u.Blockchain.db

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			outs := DeSerializeOutputs(v)

			for _, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) {
					UTXOs = append(UTXOs, out)
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return UTXOs
}

// CountTransactions - return the no of transactions in the utxo set
func (u *UTXOSet) CountTransactions() int {
	db := u.Blockchain.db
	count := 0

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		c := b.Cursor()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			count++
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return count
}

// Reindex - rebuilds the utxo set
func (u *UTXOSet) Reindex() {
	db := u.Blockchain.db
	bucketName := []byte(blocksBucket)

	err := db.View(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket(bucketName)
		if err != nil && err != bolt.ErrBucketNotFound {
			log.Panic(err)
		}

		_, err = tx.CreateBucket(bucketName)
		if err != nil {
			log.Panic(err)
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	UTXO := u.Blockchain.FindUTXO()

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)

		for txId, outs := range UTXO {
			key, err := hex.DecodeString(txId)
			if err != nil {
				log.Panic(err)
			}

			err = b.Put(key, outs.Serialize())
			if err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}

// Update - updates the UTXOset with transactions from the specified block
// the block is considered to be the tip of the blockchain
func (u *UTXOSet) Update(block *Block) {
	db := u.Blockchain.db

	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))

		for _, txn := range block.Transactions {
			if !txn.IsCoinbase() {
				for _, vin := range txn.Vin {
					updatedOuts := TXOutputs{}
					outsBytes := b.Get(vin.Txid)
					outs := DeSerializeOutputs(outsBytes)

					for outIdx, out := range outs.Outputs {
						if outIdx != vin.Vout {
							updatedOuts.Outputs = append(updatedOuts.Outputs, out)
						}
					}

					if len(updatedOuts.Outputs) == 0 {
						err := b.Delete(vin.Txid)
						if err != nil {
							log.Panic(err)
						}

					} else {
						err := b.Put(vin.Txid, updatedOuts.Serialize())
						if err != nil {
							log.Panic(err)
						}
					}
				}
			}

			newOutputs := TXOutputs{}
			for _, out := range txn.Vout {
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}

			err := b.Put(txn.ID, newOutputs.Serialize())
			if err != nil {
				log.Panic(err)
			}
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}
