package main

import (
	"awesomeProject/blockchain"
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
	bc.CreateNewBlock(transactions, 1)
	bc.CreateNewBlock(transactions, 2)
	bc.PrintChain()
}
