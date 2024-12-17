package main

import (
	"awesomeProject/IPFS"
	"awesomeProject/blockchain"
	"database/sql"
	"fmt"
	"log"
)

func main() {
	bc := &blockchain.Blockchain{}
	transactionID := "tx12345"
	sender := "Alice"
	receiver := "Bob"
	algorithm := "SHA-256"
	resultHash := "abc123def456"
	signatures := []string{"sig1", "sig2", "sig3"}
	actualOutput := "Successful transaction"
	amount := 100
	tx := blockchain.CreateTransaction(transactionID, sender, receiver, algorithm, resultHash, signatures, actualOutput, amount)
	transactions := []blockchain.Transaction{*tx}
	bc.CreateNewBlock(transactions)
	bc.CreateNewBlock(transactions)
	bc.PrintChain()
	db, err := IPFS.CreateConnection()
	if err != nil {
		log.Fatalf("Error creating connection: %v", err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatalf("Error closing db: %v", err)
		}
	}(db)

	newJSON, _ := bc.ToJSON()
	fmt.Println(string(newJSON))
}
