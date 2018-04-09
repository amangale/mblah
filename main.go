package main

import (
	"fmt"
)

func main() {
	fmt.Println("MBlah is a simple go based blockchain...")

	//mBlah := NewBlockchain("Ashwin")
	//defer mBlah.db.Close()

	//mBlah.AddBlock("One piggy two piggy...")
	//mBlah.AddBlock("penny and dime...")

	/*
		for _, block := range mBlah.blocks {
			pow := NewProofOfWork(block)
			fmt.Printf("Data       : %s\n", block.Data)
			fmt.Printf("Prev Hash  : %x\n", block.PrevBlockHash)
			fmt.Printf("Hash       : %x\n", block.Hash)
			fmt.Printf("Nonce      : %d\n", block.Nonce)
			fmt.Printf("PoW        : %s\n", strconv.FormatBool(pow.Validate()))
			fmt.Println("-------------------------------------------------------")
		}
	*/

	cli := CLI{}
	cli.Run()

}
