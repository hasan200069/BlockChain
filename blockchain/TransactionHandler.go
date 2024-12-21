package blockchain

import (
    "fmt"
    "sync"
    "time"
)

type TransactionHandler struct {
    pool            map[string]*Transaction  // Transaction pool using txID as key
    mu              sync.RWMutex
    maxPoolSize     int
    blockchain      *Blockchain
    pendingTxCount  int
}

type TransactionStatus struct {
    TransactionID string    `json:"transaction_id"`  // Added field
    Status        string    `json:"status"`
    Message       string    `json:"message"`
    Timestamp     time.Time `json:"timestamp"`
    BlockHash     string    `json:"block_hash,omitempty"`
    Sender        string    `json:"sender,omitempty"`      // Added field
    Receiver      string    `json:"receiver,omitempty"`    // Added field
    Amount        int       `json:"amount,omitempty"`      // Added field
}

func NewTransactionHandler(blockchain *Blockchain, maxPoolSize int) *TransactionHandler {
    return &TransactionHandler{
        pool:         make(map[string]*Transaction),
        blockchain:   blockchain,
        maxPoolSize:  maxPoolSize,
    }
}

// Submit a new transaction to the pool
func (th *TransactionHandler) SubmitTransaction(tx *Transaction) (*TransactionStatus, error) {
    th.mu.Lock()
    defer th.mu.Unlock()

    // Check if pool is full
    if len(th.pool) >= th.maxPoolSize {
        return nil, fmt.Errorf("transaction pool is full")
    }

    // Generate transaction ID if not present
    if tx.TransactionID == "" {
        tx.TransactionID = fmt.Sprintf("%s-%d", tx.Sender, time.Now().UnixNano())
    }

    // Validate transaction
    if err := tx.Verify(); err != nil {
        return nil, fmt.Errorf("invalid transaction: %v", err)
    }

    // Check if transaction already exists
    if _, exists := th.pool[tx.TransactionID]; exists {
        return nil, fmt.Errorf("transaction already exists")
    }

    // Add to pool
    th.pool[tx.TransactionID] = tx
    fmt.Printf("[TRANSACTION] Added new transaction %s to pool. Pool size: %d\n", 
        tx.TransactionID, len(th.pool))

    return &TransactionStatus{
        TransactionID: tx.TransactionID,
        Status:       StatusPending,
        Message:      "Transaction added to pool",
        Timestamp:    time.Now(),
        Sender:       tx.Sender,
        Receiver:     tx.Receiver,
        Amount:       tx.Amount,
    }, nil
}

// Get transactions ready for a new block
func (th *TransactionHandler) GetTransactionsForBlock(maxTx int) []*Transaction {
    th.mu.Lock()
    defer th.mu.Unlock()

    selectedTx := make([]*Transaction, 0)
    count := 0

    // Select transactions for the block
    for _, tx := range th.pool {
        if count >= maxTx {
            break
        }
        selectedTx = append(selectedTx, tx)
        delete(th.pool, tx.TransactionID)
        count++
    }

    fmt.Printf("[TRANSACTION] Selected %d transactions for new block\n", len(selectedTx))
    return selectedTx
}

// Get transaction status with enhanced details
func (th *TransactionHandler) GetTransactionStatus(txID string) (*TransactionStatus, error) {
    th.mu.RLock()
    defer th.mu.RUnlock()

    // Check if transaction is in pool
    if tx, exists := th.pool[txID]; exists {
        return &TransactionStatus{
            TransactionID: tx.TransactionID,
            Status:       StatusPending,
            Message:      "Transaction in pool",
            Timestamp:    tx.Timestamp,
            Sender:       tx.Sender,
            Receiver:     tx.Receiver,
            Amount:       tx.Amount,
        }, nil
    }

    // Check if transaction is in blockchain
    for _, block := range th.blockchain.Blocks {
        for _, tx := range block.Data.Transactions {
            if tx.TransactionID == txID {
                return &TransactionStatus{
                    TransactionID: tx.TransactionID,
                    Status:       StatusConfirmed,
                    Message:      "Transaction confirmed in blockchain",
                    Timestamp:    tx.Timestamp,
                    BlockHash:    block.Hash,
                    Sender:       tx.Sender,
                    Receiver:     tx.Receiver,
                    Amount:       tx.Amount,
                }, nil
            }
        }
    }

    return nil, fmt.Errorf("transaction not found")
}

// Clean up old transactions
func (th *TransactionHandler) cleanupOldTransactions(maxAge time.Duration) {
    th.mu.Lock()
    defer th.mu.Unlock()

    now := time.Now()
    for id, tx := range th.pool {
        if now.Sub(tx.Timestamp) > maxAge {
            delete(th.pool, id)
            fmt.Printf("[TRANSACTION] Removed expired transaction %s from pool\n", id)
        }
    }
}

// Start cleanup routine
func (th *TransactionHandler) StartCleanup(cleanupInterval, maxAge time.Duration) {
    go func() {
        ticker := time.NewTicker(cleanupInterval)
        for range ticker.C {
            th.cleanupOldTransactions(maxAge)
        }
    }()
}

// Get current pool size
func (th *TransactionHandler) GetPoolSize() int {
    th.mu.RLock()
    defer th.mu.RUnlock()
    return len(th.pool)
}
