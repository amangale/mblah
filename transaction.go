package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"
)

const subsidy = 10

// TXOutput - output of a transaction
type TXOutput struct {
	Value      int
	PubKeyHash []byte
	//ScriptPubKey string
}

// TXInput - input of a transaction
type TXInput struct {
	Txid      []byte
	Vout      int
	Signature []byte
	PubKey    []byte
	//ScriptSig string
}

// UsesKey - checks whether the address initiated this transaction
func (in *TXInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := HashPubKey(in.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}

// CanUnlockOutputWith - used to unlock input transactions
//func (tin *TXInput) CanUnlockOutputWith(unlockingData string) bool {
//	return tin.ScriptSig == unlockingData
//}

// Lock - used to set the pubkeyhash
func (out *TXOutput) Lock(address []byte) {
	pubKeyHash := Base58Decode(address)
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-addressChecksumLen]

	out.PubKeyHash = pubKeyHash
}

// IsLockedWithKey - used to verify the signature/lock
func (out *TXOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PubKeyHash, pubKeyHash) == 0
}

// NewTXOutput - creates a new TXOutput
func NewTXOutput(value int, address string) *TXOutput {
	txo := &TXOutput{
		Value:      value,
		PubKeyHash: nil,
	}
	txo.Lock([]byte(address))
	return txo
}

// TXOutputs - collection of outputs
type TXOutputs struct {
	Outputs []TXOutput
}

// Serialize - serialize the outputs
func (outs *TXOutputs) Serialize() []byte {
	var buff bytes.Buffer

	encoder := gob.NewEncoder(&buff)
	err := encoder.Encode(outs)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

// DeSerializeOutputs - deserialize the TXOutputs
func DeSerializeOutputs(data []byte) TXOutputs {
	var outputs TXOutputs

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&outputs)
	if err != nil {
		log.Panic(err)
	}

	return outputs
}

// CanBeUnlockedWith - used to unlock output transactions
//func (tout *TXOutput) CanBeUnlockedWith(unlockingData string) bool {
//	return tout.ScriptPubKey == unlockingData
//}

// Transaction - makes the life of a block worthwhile
type Transaction struct {
	ID   []byte
	Vin  []TXInput
	Vout []TXOutput
}

// IsCoinbase - identify the coinbase transaction
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vout) == 0 && tx.Vin[0].Vout == -1
}

// Serialize - serialize a transaction
func (tx *Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	encoder := gob.NewEncoder(&encoded)
	err := encoder.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()
}

// Hash - return the hash of the transaction
func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

// String - the to string function for transaction
func (tx *Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("Transaction %x", tx.ID))

	for i, vin := range tx.Vin {
		lines = append(lines, fmt.Sprintf("    Input   %d: ", i))
		lines = append(lines, fmt.Sprintf("      TXID      %x: ", vin.Txid))
		lines = append(lines, fmt.Sprintf("      Out       %d: ", vin.Vout))
		lines = append(lines, fmt.Sprintf("      Signature %x: ", vin.Signature))
		lines = append(lines, fmt.Sprintf("      PubKey    %x: ", vin.PubKey))
	}

	for i, vout := range tx.Vout {
		lines = append(lines, fmt.Sprintf("    Output  %d: ", i))
		lines = append(lines, fmt.Sprintf("      Value  %d: ", vout.Value))
		lines = append(lines, fmt.Sprintf("      Script %x: ", vout.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}

// TrimmedCopy - returns a trimmed copy of the transaction
func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	for _, vin := range tx.Vin {
		inputs = append(inputs, TXInput{
			Txid:      vin.Txid,
			Vout:      vin.Vout,
			Signature: nil,
			PubKey:    nil,
		})
	}

	for _, vout := range tx.Vout {
		outputs = append(outputs, TXOutput{
			Value:      vout.Value,
			PubKeyHash: vout.PubKeyHash,
		})
	}

	txCopy := Transaction{
		ID:   tx.ID,
		Vin:  inputs,
		Vout: outputs,
	}

	return txCopy
}

// SetID - set the tx id
func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	var hash [32]byte

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)

	if err != nil {
		log.Panic(err)
	}

	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]

}

// NewCoinbaseTX - generate a new coinbase tx
func NewCoinbaseTX(to, data string) *Transaction {
	if data == "" {
		randData := make([]byte, 20)
		_, err := rand.Read(randData)
		if err != nil {
			log.Panic(err)
		}
		data = fmt.Sprintf("%x", randData)
	}

	txin := TXInput{
		Txid:      []byte{},
		Vout:      -1,
		Signature: nil,
		PubKey:    []byte(data),
	}

	txout := NewTXOutput(subsidy, to)

	tx := Transaction{
		ID:   nil,
		Vin:  []TXInput{txin},
		Vout: []TXOutput{*txout},
	}

	//tx.SetID()
	tx.ID = tx.Hash()

	return &tx
}

// NewUTXOTransaction - create a new UTXO
func NewUTXOTransaction(wallet *Wallet, to string, amount int, UTXOSet *UTXOSet) *Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	pubKeyHash := HashPubKey(wallet.PublicKey)

	acc, validOutputs := UTXOSet.FindSpendableOutputs(pubKeyHash, amount)

	if acc < amount {
		log.Panic("ERROR: Not enough funds")
	}

	// build the inputs
	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		if err != nil {
			log.Panic(err)
		}

		for _, out := range outs {
			input := TXInput{
				Txid:      txID,
				Vout:      out,
				Signature: nil,
				PubKey:    wallet.PublicKey,
			}
			inputs = append(inputs, input)
		}

	}

	from := fmt.Sprintf("%s", wallet.GetAddress())
	// build the outputs
	outputs = append(outputs, *NewTXOutput(amount, to))
	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc-amount, from))
	}

	tx := Transaction{
		ID:   nil,
		Vin:  inputs,
		Vout: outputs,
	}
	//tx.SetID()
	tx.ID = tx.Hash()
	UTXOSet.Blockchain.SignTransaction(&tx, wallet.PrivateKey)

	return &tx
}

// Sign - sign each input of the specified transaction
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}

	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("Err : Previous transaction incorrect")
		}
	}

	txCopy := tx.TrimmedCopy()

	for inID, vin := range txCopy.Vin {
		prevTX := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTX.Vout[vin.Vout].PubKeyHash
		//txCopy.ID = txCopy.Hash()
		//txCopy.Vin[inID].PubKey = nil

		//r, s, err := ecdsa.Sign(rand.Reader, &privKey, txCopy.ID)
		dataToSign := fmt.Sprintf("%x\n", txCopy)
		r, s, err := ecdsa.Sign(rand.Reader, &privKey, []byte(dataToSign))
		if err != nil {
			log.Panic(err)
		}

		signature := append(r.Bytes(), s.Bytes()...)

		tx.Vin[inID].Signature = signature
		txCopy.Vin[inID].PubKey = nil
	}

}

// Verify - verify the signatures of transaction inputs
func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("Err : Previous transaction is incorrect.")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inID, vin := range tx.Vin {
		prevTX := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTX.Vout[vin.Vout].PubKeyHash
		//txCopy.ID = txCopy.Hash()
		//txCopy.Vin[inID].PubKey = nil

		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen / 2)])
		s.SetBytes(vin.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubKey)
		x.SetBytes(vin.PubKey[:(keyLen / 2)])
		y.SetBytes(vin.PubKey[(keyLen / 2):])

		dataToVerify := fmt.Sprintf("%x\n", txCopy)

		rawPubKey := ecdsa.PublicKey{
			Curve: curve,
			X:     &x,
			Y:     &y,
		}
		if ecdsa.Verify(&rawPubKey, []byte(dataToVerify), &r, &s) == false {
			return false
		}

	}

	return true
}

// DeserializeTransaction - deserialize transaction
func DeserializeTransaction(data []byte) Transaction {
	var transaction Transaction

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&transaction)
	if err != nil {
		log.Panic(err)
	}

	return transaction
}
