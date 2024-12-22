package blockchain

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "sort"
    "sync"
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
    StatusReplaced  = "REPLACED"
)

// Enhanced Transaction structure with fee support
type Transaction struct {
    TransactionID string    `json:"transaction_id"`
    Sequence      uint64    `json:"sequence"`
    Timestamp     time.Time `json:"timestamp"`
    Sender        string    `json:"sender"`
    Receiver      string    `json:"receiver"`
    Amount        int       `json:"amount"`
    Fee           int       `json:"fee"`            // Transaction fee
    Nonce        uint64    `json:"nonce"`          // For transaction replacement
    Status        string    `json:"status"`
    
    // Additional fields
    Algorithm     string    `json:"algorithm"`
    ResultHash    string    `json:"result_hash"`
    ActualOutput  string    `json:"actual_output"`
    
    // Double-spend prevention
    InputRefs    []string  `json:"input_refs"`     // References to previous transactions
    LastSeen     time.Time `json:"last_seen"`      // Last time transaction was seen
}

// TransactionSlice for sorting
type TransactionSlice []*Transaction

func (s TransactionSlice) Len() int { return len(s) }
func (s TransactionSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s TransactionSlice) Less(i, j int) bool {
    if s[i].Fee != s[j].Fee {
        return s[i].Fee > s[j].Fee // Higher fee first
    }
    return s[i].Timestamp.Before(s[j].Timestamp)
}

// Enhanced TransactionPool with fee-based prioritization
type TransactionPool struct {
    PendingTransactions map[string]*Transaction
    txByAddress         map[string][]*Transaction  // Transactions grouped by sender
    feeThreshold        int                        // Minimum fee required
    maxPoolSize         int                        // Maximum number of transactions
    totalSize          int                        // Current pool size
    mu                 sync.RWMutex
}

// Create a new transaction with fee
func CreateTransaction(sender, receiver string, amount, fee int, nonce uint64) (*Transaction, error) {
    if fee < 0 || amount <= 0 {
        return nil, fmt.Errorf("invalid fee or amount")
    }
    
    sequence := atomic.AddUint64(&globalSequence, 1)
    
    tx := &Transaction{
        TransactionID: generateTransactionID(sender, sequence),
        Sequence:     sequence,
        Timestamp:    time.Now(),
        Sender:       sender,
        Receiver:     receiver,
        Amount:       amount,
        Fee:         fee,
        Nonce:       nonce,
        Status:       StatusPending,
        LastSeen:    time.Now(),
        InputRefs:    make([]string, 0),
    }
    return tx, nil
}

// Generate unique transaction ID
func generateTransactionID(sender string, sequence uint64) string {
    data := fmt.Sprintf("%s-%d-%d", sender, sequence, time.Now().UnixNano())
    hasher := sha256.New()
    hasher.Write([]byte(data))
    return hex.EncodeToString(hasher.Sum(nil))
}

// New constructor with configuration
func NewTransactionPool(maxSize, minFee int) *TransactionPool {
    return &TransactionPool{
        PendingTransactions: make(map[string]*Transaction),
        txByAddress:         make(map[string][]*Transaction),
        feeThreshold:        minFee,
        maxPoolSize:         maxSize,
        totalSize:           0,
    }
}

// Add transaction with enhanced checks
func (pool *TransactionPool) AddTransaction(tx *Transaction) error {
    pool.mu.Lock()
    defer pool.mu.Unlock()

    // Check fee threshold
    if tx.Fee < pool.feeThreshold {
        return fmt.Errorf("fee below minimum threshold: got %d, need %d", tx.Fee, pool.feeThreshold)
    }

    // Check for double-spend before anything else
    if pool.isDoubleSpend(tx) {
        return fmt.Errorf("double-spend detected: input reference already used")
    }

    // Handle transaction replacement
    if existing := pool.PendingTransactions[tx.TransactionID]; existing != nil {
        if !pool.canReplace(existing, tx) {
            return fmt.Errorf("cannot replace existing transaction")
        }
        pool.removeTransaction(existing.TransactionID)
    }

    // Check pool size and make room if needed
    if pool.totalSize >= pool.maxPoolSize {
        if !pool.makeRoom(tx) {
            return fmt.Errorf("pool full and new transaction fee too low")
        }
    }

    // Add the transaction
    pool.PendingTransactions[tx.TransactionID] = tx
    pool.txByAddress[tx.Sender] = append(pool.txByAddress[tx.Sender], tx)
    pool.totalSize++

    fmt.Printf("[TRANSACTION] Added transaction %s to pool (fee: %d, inputs: %v)\n", 
        tx.TransactionID, tx.Fee, tx.InputRefs)

    return nil
}

// Check for double-spend attempts
func (pool *TransactionPool) isDoubleSpend(tx *Transaction) bool {
    pool.mu.RLock()
    defer pool.mu.RUnlock()

    fmt.Printf("[DEBUG] Checking double-spend for tx %s with inputs %v\n", 
        tx.TransactionID, tx.InputRefs)

    existingTxs := pool.txByAddress[tx.Sender]
    fmt.Printf("[DEBUG] Found %d existing transactions for sender %s\n", 
        len(existingTxs), tx.Sender)

    for _, existingTx := range existingTxs {
        // Skip if it's the same transaction
        if existingTx.TransactionID == tx.TransactionID {
            continue
        }
        

        fmt.Printf("[DEBUG] Comparing with existing tx %s with inputs %v\n", 
            existingTx.TransactionID, existingTx.InputRefs)

        for _, newRef := range tx.InputRefs {
            for _, existingRef := range existingTx.InputRefs {
                if newRef == existingRef {
                    fmt.Printf("[DOUBLE-SPEND] Detected: Transaction %s tries to spend input %s already used in transaction %s\n",
                        tx.TransactionID, newRef, existingTx.TransactionID)
                    return true
                }
            }
        }
    }

    fmt.Printf("[DEBUG] No double-spend detected for tx %s\n", tx.TransactionID)
    return false
}

// Transaction replacement rules
func (pool *TransactionPool) canReplace(old, new *Transaction) bool {
    return new.Nonce == old.Nonce && new.Fee >= (old.Fee + (old.Fee / 10))
}

// Make room in the pool for new transactions
func (pool *TransactionPool) makeRoom(newTx *Transaction) bool {
    if pool.totalSize < pool.maxPoolSize {
        return true
    }

    var lowestFeeTx *Transaction
    lowestFee := newTx.Fee

    for _, tx := range pool.PendingTransactions {
        if tx.Fee < lowestFee {
            lowestFeeTx = tx
            lowestFee = tx.Fee
        }
    }

    if lowestFeeTx != nil {
        pool.removeTransaction(lowestFeeTx.TransactionID)
        return true
    }

    return false
}

// Remove transaction from pool
func (pool *TransactionPool) removeTransaction(txID string) {
    if tx, exists := pool.PendingTransactions[txID]; exists {
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

// Get pending transactions sorted by fee
func (pool *TransactionPool) GetPendingTransactions() []*Transaction {
    pool.mu.RLock()
    defer pool.mu.RUnlock()

    transactions := make([]*Transaction, 0, len(pool.PendingTransactions))
    for _, tx := range pool.PendingTransactions {
        transactions = append(transactions, tx)
    }

    // Sort by fee (highest first) and then by timestamp
    sort.Slice(transactions, func(i, j int) bool {
        if transactions[i].Fee != transactions[j].Fee {
            return transactions[i].Fee > transactions[j].Fee
        }
        return transactions[i].Timestamp.Before(transactions[j].Timestamp)
    })

    return transactions
}

// Get transaction by ID
func (pool *TransactionPool) GetTransaction(txID string) (*Transaction, error) {
    pool.mu.RLock()
    defer pool.mu.RUnlock()

    tx, exists := pool.PendingTransactions[txID]
    if !exists {
        return nil, fmt.Errorf("transaction not found: %s", txID)
    }
    return tx, nil
}

func (tx *Transaction) Verify() error {
    // Basic validation
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
    
    // Timestamp validation
    if tx.Timestamp.After(time.Now()) {
        return fmt.Errorf("invalid timestamp: future time not allowed")
    }
    
    // Nonce validation
    if tx.Nonce == 0 {
        return fmt.Errorf("transaction nonce not set")
    }
    
    return nil
}

// Clean up old transactions
func (pool *TransactionPool) CleanupOldTransactions(maxAge time.Duration) int {
    pool.mu.Lock()
    defer pool.mu.Unlock()

    now := time.Now()
    removed := 0

    for txID, tx := range pool.PendingTransactions {
        if now.Sub(tx.LastSeen) > maxAge {
            pool.removeTransaction(txID)
            removed++
        }
    }

    return removed
}
