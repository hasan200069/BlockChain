package main

import (
	"blockchain"
	"fmt"
)

func main() {
	// Initialize the blockchain
	bc := blockchain.NewBlockchain()

	// Create transactions
	tx1 := blockchain.CreateTransaction("tx10001", "Alice", "Bob", "SHA-256", "fileCID1", "datasetCID1", "hash1", []string{"sig1"})
	tx2 := blockchain.CreateTransaction("tx10002", "Bob", "Charlie", "SHA-256", "fileCID2", "datasetCID2", "hash2", []string{"sig2"})
	tx3 := blockchain.CreateTransaction("tx10003", "Charlie", "Dave", "SHA-256", "fileCID3", "datasetCID3", "hash3", []string{"sig3"})

	// Add transactions to the blockchain
	bc.AddBlock(tx1)
	bc.AddBlock(tx2)
	bc.AddBlock(tx3)

	// Print out details of each block in the blockchain
	for i, block := range bc.Blocks {
		fmt.Printf("Block %d:\n", i+1)
		fmt.Printf("Previous Hash: %s\n", block.PreviousHash)
		fmt.Printf("Timestamp: %s\n", block.Timestamp)
		fmt.Printf("Transactions: %v\n", block.Data.Transactions)
		fmt.Printf("Hash: %s\n\n", block.Hash)
	}
}
