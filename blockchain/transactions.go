package blockchain

import (
	"time"
)

type Transaction struct {
	transactionID string
	timestamp     time.Time
	sender        string
	receiver      string
	algorithm     string
	resultHash    string
	signatures    []string
	amount        int
	actualoutput  string
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
func (t *Transaction) GetActualOutput() string {
	return t.actualoutput
}

func CreateTransaction(transactionID, sender, receiver, algorithm, resultHash string, signatures []string, actual string, amount1 int) *Transaction {
	return &Transaction{
		transactionID: transactionID,
		timestamp:     time.Now(),
		sender:        sender,
		receiver:      receiver,
		algorithm:     algorithm,
		resultHash:    resultHash,
		signatures:    signatures,
		amount:        amount1,
		actualoutput:  actual,
	}
}
