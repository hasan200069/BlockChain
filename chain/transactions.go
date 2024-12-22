package chain

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "sort"
    "sync"
    "sync/atomic"
    "time"
)

var globalSequence uint64

const (
    StatusPending   = "PENDING"
    StatusConfirmed = "CONFIRMED"
    StatusFailed    = "FAILED"
    StatusReplaced  = "REPLACED"
)

type Transaction struct {
    TransactionID string    `json:"transaction_id"`
    Sequence      uint64    `json:"sequence"`
    Timestamp     time.Time `json:"timestamp"`
    Sender        string    `json:"sender"`
    Receiver      string    `json:"receiver"`
    Amount        int       `json:"amount"`
    Fee           int       `json:"fee"`
    Nonce         uint64    `json:"nonce"`
    Status        string    `json:"status"`
    Algorithm     string    `json:"algorithm"`
    ResultHash    string    `json:"result_hash"`
    ActualOutput  string    `json:"actual_output"`
    InputRefs     []string  `json:"input_refs"`
    InputAmounts  []int     `json:"input_amounts"`
    LastSeen      time.Time `json:"last_seen"`
    SpentOutputs  []string  `json:"spent_outputs"`
}

type TransactionSlice []*Transaction

func (s TransactionSlice) Len() int { return len(s) }
func (s TransactionSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s TransactionSlice) Less(i, j int) bool {
    if s[i].Fee != s[j].Fee {
        return s[i].Fee > s[j].Fee
    }
    return s[i].Timestamp.Before(s[j].Timestamp)
}

type TransactionPool struct {
    PendingTransactions map[string]*Transaction
    txByAddress         map[string][]*Transaction
    spentOutputs        map[string]string
    feeThreshold        int
    maxPoolSize         int
    totalSize           int
    mu                  sync.RWMutex
}

func CreateTransaction(sender, receiver string, amount, fee int, nonce uint64, inputRefs []string, inputAmounts []int) (*Transaction, error) {
    if fee < 0 || amount <= 0 {
        return nil, fmt.Errorf("invalid fee or amount")
    }
    if len(inputRefs) != len(inputAmounts) {
        return nil, fmt.Errorf("input references and amounts must match")
    }
    totalInput := 0
    for _, amount := range inputAmounts {
        totalInput += amount
    }
    if totalInput < amount+fee {
        return nil, fmt.Errorf("insufficient input amount: got %d, need %d", totalInput, amount+fee)
    }
    sequence := atomic.AddUint64(&globalSequence, 1)
    tx := &Transaction{
        TransactionID: generateTransactionID(sender, sequence),
        Sequence:      sequence,
        Timestamp:     time.Now(),
        Sender:        sender,
        Receiver:      receiver,
        Amount:        amount,
        Fee:           fee,
        Nonce:         nonce,
        Status:        StatusPending,
        LastSeen:      time.Now(),
        InputRefs:     inputRefs,
        InputAmounts:  inputAmounts,
        SpentOutputs:  make([]string, 0),
    }
    return tx, nil
}

func NewTransactionPool(maxSize, minFee int) *TransactionPool {
    return &TransactionPool{
        PendingTransactions: make(map[string]*Transaction),
        txByAddress:         make(map[string][]*Transaction),
        spentOutputs:        make(map[string]string),
        feeThreshold:        minFee,
        maxPoolSize:         maxSize,
        totalSize:           0,
    }
}

func (pool *TransactionPool) isDoubleSpend(tx *Transaction) bool {
    pool.mu.RLock()
    defer pool.mu.RUnlock()
    for i, inputRef := range tx.InputRefs {
        if spendingTx, exists := pool.spentOutputs[inputRef]; exists {
            fmt.Printf("[DOUBLE-SPEND] Input %s already spent in transaction %s\n", inputRef, spendingTx)
            return true
        }
        if tx.InputAmounts[i] <= 0 {
            fmt.Printf("[INVALID-INPUT] Input amount must be positive: got %d for input %s\n", tx.InputAmounts[i], inputRef)
            return true
        }
    }
    return false
}

func (tx *Transaction) Verify() error {
    if tx.Amount <= 0 {
        return fmt.Errorf("invalid amount: %d", tx.Amount)
    }
    if tx.Fee < 0 {
        return fmt.Errorf("invalid fee: %d", tx.Fee)
    }
    if tx.Sender == "" || tx.Receiver == "" {
        return fmt.Errorf("invalid sender or receiver")
    }
    if tx.Sequence == 0 {
        return fmt.Errorf("transaction sequence number not set")
    }
    if len(tx.InputRefs) == 0 {
        return fmt.Errorf("no input references provided")
    }
    if len(tx.InputRefs) != len(tx.InputAmounts) {
        return fmt.Errorf("input references and amounts must match")
    }
    totalInput := 0
    for _, amount := range tx.InputAmounts {
        if amount <= 0 {
            return fmt.Errorf("invalid input amount: %d", amount)
        }
        totalInput += amount
    }
    if totalInput < tx.Amount+tx.Fee {
        return fmt.Errorf("insufficient input amount: got %d, need %d", totalInput, tx.Amount+tx.Fee)
    }
    if tx.Timestamp.After(time.Now()) {
        return fmt.Errorf("invalid timestamp: future time not allowed")
    }
    return nil
}

func (pool *TransactionPool) AddTransaction(tx *Transaction) error {
    pool.mu.Lock()
    defer pool.mu.Unlock()
    if tx.Fee < pool.feeThreshold {
        return fmt.Errorf("fee below minimum threshold: got %d, need %d", tx.Fee, pool.feeThreshold)
    }
    if err := tx.Verify(); err != nil {
        return fmt.Errorf("transaction verification failed: %v", err)
    }
    if pool.isDoubleSpend(tx) {
        return fmt.Errorf("double-spend detected")
    }
    if existing := pool.PendingTransactions[tx.TransactionID]; existing != nil {
        if !pool.canReplace(existing, tx) {
            return fmt.Errorf("cannot replace existing transaction")
        }
        pool.removeTransaction(existing.TransactionID)
    }
    for _, inputRef := range tx.InputRefs {
        pool.spentOutputs[inputRef] = tx.TransactionID
    }
    pool.PendingTransactions[tx.TransactionID] = tx
    pool.txByAddress[tx.Sender] = append(pool.txByAddress[tx.Sender], tx)
    pool.totalSize++
    fmt.Printf("[TRANSACTION] Added transaction %s to pool (fee: %d, inputs: %v)\n", tx.TransactionID, tx.Fee, tx.InputRefs)
    return nil
}

func (pool *TransactionPool) removeTransaction(txID string) {
    if tx, exists := pool.PendingTransactions[txID]; exists {
        for _, inputRef := range tx.InputRefs {
            delete(pool.spentOutputs, inputRef)
        }
        senderTxs := pool.txByAddress[tx.Sender]
        for i, t := range senderTxs {
            if t.TransactionID == txID {
                pool.txByAddress[tx.Sender] = append(senderTxs[:i], senderTxs[i+1:]...)
                break
            }
        }
        delete(pool.PendingTransactions, txID)
        pool.totalSize--
    }
}
