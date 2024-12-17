package blockchain

import (
	"time"
)

type Transaction struct {
	TransactionID string    `json:"transaction_id"`
	Timestamp     time.Time `json:"timestamp"`
	Sender        string    `json:"sender"`
	Receiver      string    `json:"receiver"`
	Algorithm     string    `json:"algorithm"`
	ResultHash    string    `json:"result_hash"`
	Signatures    []string  `json:"signatures"`
	Amount        int       `json:"amount"`
	ActualOutput  string    `json:"actual_output"`
}

// Getter Methods
func (t *Transaction) GetTransactionID() string {
	return t.TransactionID
}

func (t *Transaction) GetTransactionAmount() int {
	return t.Amount
}

func (t *Transaction) GetTimestamp() time.Time {
	return t.Timestamp
}

func (t *Transaction) GetSender() string {
	return t.Sender
}

func (t *Transaction) GetReceiver() string {
	return t.Receiver
}

func (t *Transaction) GetAlgorithm() string {
	return t.Algorithm
}

func (t *Transaction) GetResultHash() string {
	return t.ResultHash
}

func (t *Transaction) GetSignatures() []string {
	return t.Signatures
}

func (t *Transaction) GetActualOutput() string {
	return t.ActualOutput
}

// Constructor Method
func CreateTransaction(transactionID, sender, receiver, algorithm, resultHash string, signatures []string, actual string, amount1 int) *Transaction {
	return &Transaction{
		TransactionID: transactionID,
		Timestamp:     time.Now(),
		Sender:        sender,
		Receiver:      receiver,
		Algorithm:     algorithm,
		ResultHash:    resultHash,
		Signatures:    signatures,
		Amount:        amount1,
		ActualOutput:  actual,
	}
}
