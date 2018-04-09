package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
)

// CLI - cmd line interface
type CLI struct {
	//bc *Blockchain
}

func (cli *CLI) printUsage() {
	fmt.Println("Usage: ")
	//fmt.Println(" addblock -data BLOCK_DATA - add a block to the blockchain")
	fmt.Println("createwallet - generates a new key pair and stores them in a wallet")
	fmt.Println("listaddresses - lists all addresses from the wallet")
	fmt.Println(" createblockchain -address ADDRESS create a blockchain and send the genesis reward to ADDRESS")
	fmt.Println(" getbalance -address ADDRESS get the balance for ADDRESS")
	fmt.Println(" printchain - print all the blocks of the blockchain")
	fmt.Println(" send -from SENDER -to RECEIVER -amount AMOUNT -mine  send AMAOUNT from SENDER to RECEIVER and mine if mine is set")
	fmt.Println("startnode -miner ADDRESS  - Start a node with the specified ID in the env var. miner enables mining")
}

func (cli *CLI) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
}

func (cli *CLI) createBlockchain(address, nodeID string) {
	if !ValidateAddress(address) {
		log.Panic("Err : address invalid")
	}

	bc := CreateBlockchain(address, nodeID)

	bc.db.Close()
	fmt.Println("Done!")

}

/*
func (cli *CLI) addBlock(transactions []*Transaction) {
	cli.bc.AddBlock(transactions)
}
*/

func (cli *CLI) getBalance(address, nodeID string) {
	if !ValidateAddress(address) {
		log.Panic("Err : Invalid Address")
	}

	bc := NewBlockchain(nodeID)
	UTXOSet := UTXOSet{
		Blockchain: bc,
	}
	defer bc.db.Close()

	balance := 0
	pubKeyHash := Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]

	UTXOs := UTXOSet.FindUTXO(pubKeyHash)

	for _, out := range UTXOs {
		balance = balance + out.Value
	}

	fmt.Printf("Balance of %s is %d\n", address, balance)
}

func (cli *CLI) send(from, to, nodeID string, amount int, mineNow bool) {
	if !ValidateAddress(from) {
		log.Panic("err : sender address invalid")
	}

	if !ValidateAddress(to) {
		log.Panic("err : recipient address invalid")
	}

	bc := NewBlockchain(nodeID)
	UTXOSet := UTXOSet{
		Blockchain: bc,
	}
	defer bc.db.Close()

	wallets, err := NewWallets(nodeID)
	if err != nil {
		log.Panic(err)
	}

	wallet := wallets.GetWallet(from)

	tx := NewUTXOTransaction(&wallet, to, amount, &UTXOSet)

	if mineNow {
		cbTx := NewCoinbaseTX(from, "")
		txs := []*Transaction{cbTx, tx}

		newBlock := bc.MineBlock(txs)
		UTXOSet.Update(newBlock)
	} else {
		sendTx(knownNodes[0], tx)
	}

	fmt.Println("success!")

}

func (cli *CLI) printChain(nodeID string) {

	bc := NewBlockchain(nodeID)
	bci := bc.Iterator()

	for {
		block := bci.Next()

		pow := NewProofOfWork(block)
		//fmt.Printf("Data       : %s\n", block.Data)
		fmt.Printf("Prev Hash  : %x\n", block.PrevBlockHash)
		fmt.Printf("Hash       : %x\n", block.Hash)
		fmt.Printf("Nonce      : %d\n", block.Nonce)
		fmt.Printf("PoW        : %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Printf("Height     : %d\n", block.Height)
		fmt.Println("-------------------------------------------------------")

		if len(block.PrevBlockHash) == 0 {
			break
		}

	}

}

func (cli *CLI) createWallet(nodeID string) {
	wallets, _ := NewWallets(nodeID)
	address := wallets.CreateWallet()
	wallets.SaveToFile(nodeID)

	fmt.Printf("Your new address : %s\n", address)
}

func (cli *CLI) listAddresses(nodeID string) {
	wallets, err := NewWallets(nodeID)
	if err != nil {
		log.Panic(err)
	}

	addresses := wallets.GetAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}

}

func (cli *CLI) reindexUTXO(nodeID string) {
	bc := NewBlockchain(nodeID)
	UTXOSet := UTXOSet{
		Blockchain: bc,
	}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	fmt.Printf("Done! There are %d transactions in the UTXO set\n", count)
}

func (cli *CLI) startNode(nodeID, minerAddress string) {
	fmt.Printf("Starting node %s]n", nodeID)
	if len(minerAddress) > 0 {
		if ValidateAddress(minerAddress) {
			fmt.Println("Mining started. Rewards sent to: ", minerAddress)
		} else {
			log.Panic("wrong miner address")
		}
	}
	StartServer(nodeID, minerAddress)
}

// Run - execute the cli
func (cli *CLI) Run() {
	cli.validateArgs()

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Println("No node ID set.")
		os.Exit(1)
	}

	//addBlockCmd := flag.NewFlagSet("addblock", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	reindexCmd := flag.NewFlagSet("reindex", flag.ExitOnError)
	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)

	//addBlockData := addBlockCmd.String("data", "", "block data")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "coinbase address")
	getBalanceAddress := getBalanceCmd.String("address", "", "get the balance for this address")
	senderAddress := sendCmd.String("from", "", " specify the sender address")
	receiverAddress := sendCmd.String("to", "", " specify the receiver address")
	amountInt := sendCmd.Int("amount", 0, " specify the amount to be transferred")
	sendMine := sendCmd.Bool("mine", false, "mine on the same node")
	startNodeMiner := startNodeCmd.String("miner", "", "enable mining and send reward to ADDRESS")

	switch os.Args[1] {
	case "createblockchain":
		{
			err := createBlockchainCmd.Parse(os.Args[2:])
			if err != nil {
				cli.printUsage()
				os.Exit(1)
			}
		}
	case "printchain":
		{
			err := printChainCmd.Parse(os.Args[2:])
			if err != nil {
				cli.printUsage()
				os.Exit(1)
			}
		}
	case "getbalance":
		{
			err := getBalanceCmd.Parse(os.Args[2:])
			if err != nil {
				cli.printUsage()
				os.Exit(1)
			}
		}
	case "send":
		{
			err := sendCmd.Parse(os.Args[2:])
			if err != nil {
				cli.printUsage()
				os.Exit(1)
			}
		}
	case "createwallet":
		{
			err := createWalletCmd.Parse(os.Args[2:])
			if err != nil {
				cli.printUsage()
				os.Exit(1)
			}
		}
	case "listaddresses":
		{
			err := listAddressesCmd.Parse(os.Args[2:])
			if err != nil {
				cli.printUsage()
				os.Exit(1)
			}
		}
	case "reindex":
		{
			err := reindexCmd.Parse(os.Args[2:])
			if err != nil {
				cli.printUsage()
				os.Exit(1)
			}
		}
	case "startnode":
		{
			err := startNodeCmd.Parse(os.Args[2:])
			if err != nil {
				cli.printChain(nodeID)
				os.Exit(1)
			}
		}
	default:
		{
			cli.printUsage()
			os.Exit(1)
		}

	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			os.Exit(1)
		}
		cli.createBlockchain(*createBlockchainAddress, nodeID)
	}

	if printChainCmd.Parsed() {
		cli.printChain(nodeID)
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			os.Exit(1)
		}
		cli.getBalance(*getBalanceAddress, nodeID)
	}

	if sendCmd.Parsed() {
		if *senderAddress == "" || *receiverAddress == "" || *amountInt <= 0 {
			sendCmd.Usage()
			os.Exit(1)
		}
		cli.send(*senderAddress, *receiverAddress, nodeID, *amountInt, *sendMine)
	}

	if startNodeCmd.Parsed() {
		nodeID = os.Getenv("NODE_ID")
		if nodeID == "" {
			startNodeCmd.Usage()
			os.Exit(1)
		}
		cli.startNode(nodeID, *startNodeMiner)
	}

	if createWalletCmd.Parsed() {
		cli.createWallet(nodeID)
	}

	if listAddressesCmd.Parsed() {
		cli.listAddresses(nodeID)
	}

	if reindexCmd.Parsed() {
		cli.reindexUTXO(nodeID)
	}

}
