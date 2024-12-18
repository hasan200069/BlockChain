package blockchain

import (
    "fmt"
    "sync"
)

type TransactionProcessor struct {
    pool      *TransactionPool
    mu        sync.RWMutex
    maxTxSize int
}

func NewTransactionProcessor(maxTxSize int) *TransactionProcessor {
    return &TransactionProcessor{
        pool:      NewTransactionPool(),
        maxTxSize: maxTxSize,
    }
}

func (tp *TransactionProcessor) ProcessTransaction(tx *Transaction) error {
    tp.mu.Lock()
    defer tp.mu.Unlock()

    // Verify transaction
    if err := tx.Verify(); err != nil {
        tx.Status = StatusFailed
        return fmt.Errorf("transaction verification failed: %v", err)
    }

    // Add to pool
    if err := tp.pool.AddTransaction(tx); err != nil {
        return fmt.Errorf("failed to add transaction to pool: %v", err)
    }

    tx.Status = StatusPending
    return nil
}

func (tp *TransactionProcessor) GetTransactionsForBlock() ([]*Transaction, error) {
    tp.mu.Lock()
    defer tp.mu.Unlock()

    pendingTx := tp.pool.GetPendingTransactions()
    if len(pendingTx) == 0 {
        return nil, nil
    }

    // selecting transactions for the next block
    var selectedTx []*Transaction
    for _, tx := range pendingTx {
        if len(selectedTx) >= tp.maxTxSize {
            break
        }
        selectedTx = append(selectedTx, tx)
        tx.Status = StatusConfirmed
        tp.pool.RemoveTransaction(tx.TransactionID)
    }

    return selectedTx, nil
}
