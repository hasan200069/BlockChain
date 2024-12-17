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
	table := IPFS.IPFSTable{
		AlgorithmName:    "Example Algorithm",
		AlgorithmFileCid: "QmExampleCid1",
		DatasetCid:       "QmExampleCid2",
	}
	err = IPFS.InsertIPFSTable(db, table)
	if err != nil {
		log.Fatalf("Error inserting data into IPFS table: %v", err)
	} else {
		fmt.Println("Record inserted successfully!")
	}
	algorithmName := "Example Algorithm"
	result, err := IPFS.GetIPFSTableByName(db, algorithmName)
	if err != nil {
		log.Fatalf("Error retrieving data: %v", err)
	} else {
		fmt.Printf("Retrieved Record: %+v\n", result)
	}
}
