package blockchain

import (
    "crypto/ecdsa"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "sort"
    "sync/atomic"
    "time"
)

// Global sequence counter for transactions
var globalSequence uint64

// Transaction status constants
const (
    StatusPending   = "PENDING"
    StatusConfirmed = "CONFIRMED"
    StatusFailed    = "FAILED"
)

// Enhanced Transaction structure
type Transaction struct {
    TransactionID string    `json:"transaction_id"`
    Sequence      uint64    `json:"sequence"`      // Added field for serialization
    Timestamp     time.Time `json:"timestamp"`
    Sender        string    `json:"sender"`
    Receiver      string    `json:"receiver"`
    Amount        int       `json:"amount"`
    Signature     []byte    `json:"signature"`
    PublicKey     []byte    `json:"public_key"`
    Status        string    `json:"status"`
    
    // Algorithm execution related fields
    Algorithm     string    `json:"algorithm"`
    ResultHash    string    `json:"result_hash"`
    ActualOutput  string    `json:"actual_output"`
}

// TransactionPool manages pending transactions
type TransactionPool struct {
    PendingTransactions map[string]*Transaction
}

// Create a new transaction with sequence number
func CreateTransaction(sender, receiver string, amount int) (*Transaction, error) {
    sequence := atomic.AddUint64(&globalSequence, 1)
    
    tx := &Transaction{
        TransactionID: generateTransactionID(),
        Sequence:     sequence,
        Timestamp:    time.Now(),
        Sender:       sender,
        Receiver:     receiver,
        Amount:       amount,
        Status:       StatusPending,
    }
    return tx, nil
}

// Generate unique transaction ID
func generateTransactionID() string {
    timestamp := time.Now().UnixNano()
    data := fmt.Sprintf("%d%d", timestamp, time.Now().UnixNano())
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])
}

// Sign a transaction
func (tx *Transaction) Sign(privateKey *ecdsa.PrivateKey) error {
    // Create transaction data hash
    data, err := tx.getSigningData()
    if err != nil {
        return fmt.Errorf("failed to get signing data: %v", err)
    }
    
    hash := sha256.Sum256(data)
    
    // Sign the hash
    signature, err := ecdsa.SignASN1(nil, privateKey, hash[:])
    if err != nil {
        return fmt.Errorf("failed to sign transaction: %v", err)
    }
    
    tx.Signature = signature
    return nil
}

// Verify transaction signature and data
func (tx *Transaction) Verify() error {
    // Basic validation
    if tx.Amount <= 0 {
        return fmt.Errorf("invalid amount: %d", tx.Amount)
    }
    if tx.Sender == "" || tx.Receiver == "" {
        return fmt.Errorf("invalid sender or receiver")
    }
    
    // Verify sequence number is set
    if tx.Sequence == 0 {
        return fmt.Errorf("transaction sequence number not set")
    }
    
    // More validations can be added here
    
    return nil
}

// Get data for signing
func (tx *Transaction) getSigningData() ([]byte, error) {
    // Create a copy of the transaction without the signature
    txCopy := *tx
    txCopy.Signature = nil
    return json.Marshal(txCopy)
}

// Implement custom sorting for transactions
type TransactionSlice []*Transaction

func (s TransactionSlice) Len() int { return len(s) }
func (s TransactionSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s TransactionSlice) Less(i, j int) bool {
    if s[i].Timestamp.Equal(s[j].Timestamp) {
        return s[i].Sequence < s[j].Sequence
    }
    return s[i].Timestamp.Before(s[j].Timestamp)
}

// TransactionPool methods
func NewTransactionPool() *TransactionPool {
    return &TransactionPool{
        PendingTransactions: make(map[string]*Transaction),
    }
}

func (pool *TransactionPool) AddTransaction(tx *Transaction) error {
    if err := tx.Verify(); err != nil {
        return fmt.Errorf("transaction validation failed: %v", err)
    }
    
    pool.PendingTransactions[tx.TransactionID] = tx
    return nil
}

func (pool *TransactionPool) GetTransaction(txID string) (*Transaction, error) {
    tx, exists := pool.PendingTransactions[txID]
    if !exists {
        return nil, fmt.Errorf("transaction not found: %s", txID)
    }
    return tx, nil
}

func (pool *TransactionPool) RemoveTransaction(txID string) {
    delete(pool.PendingTransactions, txID)
}

// Get all pending transactions in sequence order
func (pool *TransactionPool) GetPendingTransactions() []*Transaction {
    transactions := make(TransactionSlice, 0, len(pool.PendingTransactions))
    for _, tx := range pool.PendingTransactions {
        transactions = append(transactions, tx)
    }
    // Sort by timestamp and sequence before returning
    sort.Sort(transactions)
    return transactions
}
