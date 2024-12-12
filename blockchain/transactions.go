package blockchain

import (
	"time"
)

type Transaction struct {
	transactionID    string
	timestamp        time.Time
	sender           string
	receiver         string
	algorithm        string
	algorithmFileCid string
	datasetCID       string
	resultHash       string
	signatures       []string
	amount           int
}

func (t *Transaction) GetTransactionID() string {
	return t.transactionID
}

func (t *Transaction) GetTransactionAmount() int {
	return t.amount
}

func (t *Transaction) GetTimestamp() time.Time {
	return t.timestamp
}

func (t *Transaction) GetSender() string {
	return t.sender
}

func (t *Transaction) GetReceiver() string {
	return t.receiver
}

func (t *Transaction) GetAlgorithm() string {
	return t.algorithm
}
func (t *Transaction) GetResultHash() string {
	return t.resultHash
}

func (t *Transaction) GetSignatures() []string {
	return t.signatures
}

func CreateTransaction(transactionID, sender, receiver, algorithm, algorithmFileCid, datasetCID, resultHash string, signatures []string) *Transaction {
	return &Transaction{
		transactionID: transactionID,
		timestamp:     time.Now(),
		sender:        sender,
		receiver:      receiver,
		algorithm:     algorithm,
		resultHash:    resultHash,
		signatures:    signatures,
	}
}
