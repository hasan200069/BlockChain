package blockchain

import (
    "encoding/json"
    "fmt"
    "sort"
    "sync"
    "time"
)

type TransactionProcessor struct {
    pool           *TransactionPool
    mu             sync.RWMutex
    maxTxSize      int
    minFee         int
    maxFeePerByte  int
    metrics        ProcessorMetrics
}

type ProcessorMetrics struct {
    ProcessedCount      int64
    RejectedCount      int64
    TotalFeesCollected int64
    LastProcessingTime time.Time
    AverageFee        float64
}

func NewTransactionProcessor(maxTxSize, minFee, maxFeePerByte int) *TransactionProcessor {
    return &TransactionProcessor{
        pool:          NewTransactionPool(maxTxSize, minFee),
        maxTxSize:     maxTxSize,
        minFee:        minFee,
        maxFeePerByte: maxFeePerByte,
        metrics:       ProcessorMetrics{},
    }
}

func (tp *TransactionProcessor) ProcessTransaction(tx *Transaction) error {
    tp.mu.Lock()
    defer tp.mu.Unlock()

    startTime := time.Now()

    // Validate transaction
    if err := tx.Verify(); err != nil {
        tp.metrics.RejectedCount++
        tx.Status = StatusFailed
        return fmt.Errorf("transaction verification failed: %v", err)
    }

    // Validate fee
    if tx.Fee < tp.minFee {
        tp.metrics.RejectedCount++
        return fmt.Errorf("fee too low, minimum required: %d", tp.minFee)
    }

    // Check fee per byte (anti-spam measure)
    txSize := estimateTransactionSize(tx)
    if (tx.Fee / txSize) > tp.maxFeePerByte {
        tp.metrics.RejectedCount++
        return fmt.Errorf("fee per byte too high, maximum allowed: %d", tp.maxFeePerByte)
    }

    // Add to pool
    if err := tp.pool.AddTransaction(tx); err != nil {
        tp.metrics.RejectedCount++
        return fmt.Errorf("failed to add transaction to pool: %v", err)
    }

    // Update metrics
    tp.metrics.ProcessedCount++
    tp.metrics.TotalFeesCollected += int64(tx.Fee)
    tp.metrics.LastProcessingTime = time.Now()
    tp.metrics.AverageFee = float64(tp.metrics.TotalFeesCollected) / float64(tp.metrics.ProcessedCount)

    processingDuration := time.Since(startTime)
    fmt.Printf("[PROCESSOR] Processed transaction %s in %v\n", tx.TransactionID, processingDuration)

    tx.Status = StatusPending
    return nil
}

func (tp *TransactionProcessor) GetTransactionsForBlock() ([]*Transaction, error) {
    tp.mu.Lock()
    defer tp.mu.Unlock()

    pendingTx := tp.pool.GetPendingTransactions() // This now returns fee-prioritized transactions
    if len(pendingTx) == 0 {
        return nil, nil
    }

    // Select transactions for the next block, prioritizing by fee
    var selectedTx []*Transaction
    var totalSize int
    var totalFees int
    blockSizeLimit := tp.maxTxSize * 1000 // Convert to bytes

    for _, tx := range pendingTx {
        txSize := estimateTransactionSize(tx)
        
        // Check if adding this transaction would exceed block size limit
        if totalSize+txSize > blockSizeLimit {
            continue
        }

        // Validate transaction again before inclusion
        if err := tx.Verify(); err != nil {
            fmt.Printf("[PROCESSOR] Skipping invalid transaction %s: %v\n", tx.TransactionID, err)
            continue
        }

        selectedTx = append(selectedTx, tx)
        totalSize += txSize
        totalFees += tx.Fee

        // Check if we've reached the transaction count limit
        if len(selectedTx) >= tp.maxTxSize {
            break
        }
    }

    // Update status and remove selected transactions from pool
    for _, tx := range selectedTx {
        tx.Status = StatusConfirmed
        tp.pool.removeTransaction(tx.TransactionID)
    }

    fmt.Printf("[PROCESSOR] Selected %d transactions with total fees: %d satoshis, total size: %d bytes\n", 
        len(selectedTx), totalFees, totalSize)

    return selectedTx, nil
}

func (tp *TransactionProcessor) GetMetrics() ProcessorMetrics {
    tp.mu.RLock()
    defer tp.mu.RUnlock()
    return tp.metrics
}

func (tp *TransactionProcessor) GetPoolSize() int {
    return tp.pool.totalSize
}

func (tp *TransactionProcessor) CleanupOldTransactions(maxAge time.Duration) int {
    removed := tp.pool.CleanupOldTransactions(maxAge)
    fmt.Printf("[PROCESSOR] Cleaned up %d old transactions\n", removed)
    return removed
}

// Helper function to estimate transaction size in bytes
func estimateTransactionSize(tx *Transaction) int {
    // Convert transaction to JSON to get approximate size
    txJSON, err := json.Marshal(tx)
    if err != nil {
        // Return a conservative estimate if marshaling fails
        return 250
    }
    
    // Add overhead for network protocol and storage
    return len(txJSON) + 50
}

// New method to handle transaction replacement
func (tp *TransactionProcessor) ReplaceTransaction(oldTxID string, newTx *Transaction) error {
    tp.mu.Lock()
    defer tp.mu.Unlock()

    // Get the old transaction
    oldTx, err := tp.pool.GetTransaction(oldTxID)
    if err != nil {
        return fmt.Errorf("old transaction not found: %v", err)
    }

    // Verify replacement criteria
    if oldTx.Sender != newTx.Sender {
        return fmt.Errorf("replacement transaction must be from same sender")
    }

    if newTx.Fee <= oldTx.Fee {
        return fmt.Errorf("replacement transaction must have higher fee")
    }

    // Remove old transaction and add new one
    tp.pool.removeTransaction(oldTxID)
    if err := tp.ProcessTransaction(newTx); err != nil {
        // Restore old transaction if new one fails
        tp.pool.AddTransaction(oldTx)
        return fmt.Errorf("failed to process replacement transaction: %v", err)
    }

    oldTx.Status = StatusReplaced
    return nil
}

// New method to get current fee statistics
func (tp *TransactionProcessor) GetFeeStats() map[string]interface{} {
    tp.mu.RLock()
    defer tp.mu.RUnlock()

    // Calculate fee percentiles from pending transactions
    pendingTx := tp.pool.GetPendingTransactions()
    var fees []int
    for _, tx := range pendingTx {
        fees = append(fees, tx.Fee)
    }

    // Sort fees for percentile calculation
    sort.Ints(fees)
    
    var lowFee, medianFee, highFee int
    if len(fees) > 0 {
        lowFee = fees[len(fees)/4]                    // 25th percentile
        medianFee = fees[len(fees)/2]                 // 50th percentile
        highFee = fees[len(fees)-len(fees)/4]         // 75th percentile
    }

    return map[string]interface{}{
        "min_fee_required": tp.minFee,
        "current_low_fee": lowFee,
        "current_median_fee": medianFee,
        "current_high_fee": highFee,
        "average_fee": tp.metrics.AverageFee,
        "max_fee_per_byte": tp.maxFeePerByte,
    }
}
