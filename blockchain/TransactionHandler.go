package blockchain

import (
    "bytes"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "sort"
    "sync"
    "time"
)

type TransactionHandler struct {
    pool            map[string]*Transaction
    mu              sync.RWMutex
    maxPoolSize     int
    blockchain      *Blockchain
    pendingTxCount  int
    nodeID          string
    processedMsgs   map[string]bool
    msgMutex        sync.RWMutex
}

type TransactionStatus struct {
    TransactionID string    `json:"transaction_id"`
    Status        string    `json:"status"`
    Message       string    `json:"message"`
    Timestamp     time.Time `json:"timestamp"`
    BlockHash     string    `json:"block_hash,omitempty"`
    Sender        string    `json:"sender,omitempty"`
    Receiver      string    `json:"receiver,omitempty"`
    Amount        int       `json:"amount,omitempty"`
    Sequence      uint64    `json:"sequence,omitempty"`
}

func NewTransactionHandler(blockchain *Blockchain, maxPoolSize int) *TransactionHandler {
    th := &TransactionHandler{
        pool:          make(map[string]*Transaction),
        blockchain:    blockchain,
        maxPoolSize:   maxPoolSize,
        processedMsgs: make(map[string]bool),
        nodeID:        fmt.Sprintf("node-%d", time.Now().UnixNano()),
    }
    
    // Start cleanup routines
    go th.startMessageCleanup()
    
    return th
}

func (th *TransactionHandler) SubmitTransaction(tx *Transaction) (*TransactionStatus, error) {
    th.mu.Lock()
    defer th.mu.Unlock()

    // Validate pool size
    if len(th.pool) >= th.maxPoolSize {
        return nil, fmt.Errorf("transaction pool is full")
    }

    // Generate transaction ID if not present
    if tx.TransactionID == "" {
        tx.TransactionID = fmt.Sprintf("%s-%d", tx.Sender, time.Now().UnixNano())
    }

    // Validate transaction
    if err := tx.Verify(); err != nil {
        log.Printf("[TRANSACTION] Validation failed for tx %s: %v", tx.TransactionID, err)
        return nil, fmt.Errorf("invalid transaction: %v", err)
    }

    // Check if transaction already exists
    if _, exists := th.pool[tx.TransactionID]; exists {
        return nil, fmt.Errorf("transaction already exists")
    }

    // Add to pool
    th.pool[tx.TransactionID] = tx
    log.Printf("[TRANSACTION] Added new transaction %s to pool. Pool size: %d", 
        tx.TransactionID, len(th.pool))

    // Propagate transaction
    go th.propagateTransaction(tx)

    return &TransactionStatus{
        TransactionID: tx.TransactionID,
        Status:       StatusPending,
        Message:      "Transaction added to pool",
        Timestamp:    tx.Timestamp,
        Sender:       tx.Sender,
        Receiver:     tx.Receiver,
        Amount:       tx.Amount,
        Sequence:     tx.Sequence,
    }, nil
}

func (th *TransactionHandler) GetTransactionsForBlock(maxTx int) []*Transaction {
    th.mu.Lock()
    defer th.mu.Unlock()

    // Convert map to slice for sorting
    transactions := make(TransactionSlice, 0, len(th.pool))
    for _, tx := range th.pool {
        transactions = append(transactions, tx)
    }

    // Sort transactions by timestamp and sequence
    sort.Sort(transactions)

    // Select transactions for the block
    selectedTx := make([]*Transaction, 0)
    count := 0

    for _, tx := range transactions {
        if count >= maxTx {
            break
        }
        selectedTx = append(selectedTx, tx)
        delete(th.pool, tx.TransactionID)
        tx.Status = StatusConfirmed
        count++
    }

    log.Printf("[TRANSACTION] Selected %d transactions for new block (sorted by sequence)", len(selectedTx))
    return selectedTx
}

func (th *TransactionHandler) propagateTransaction(tx *Transaction) error {
    message := struct {
        Transaction *Transaction `json:"transaction"`
        SenderID    string      `json:"sender_id"`
        Timestamp   time.Time   `json:"timestamp"`
        MessageID   string      `json:"message_id"`
    }{
        Transaction: tx,
        SenderID:    th.nodeID,
        Timestamp:   time.Now(),
        MessageID:   fmt.Sprintf("%s-%d", th.nodeID, time.Now().UnixNano()),
    }

    messageJSON, err := json.Marshal(message)
    if err != nil {
        return fmt.Errorf("failed to marshal transaction message: %v", err)
    }

    // Broadcast to known peers
    peers := th.getPeerAddresses()
    
    var wg sync.WaitGroup
    for _, peer := range peers {
        wg.Add(1)
        go func(peerAddr string) {
            defer wg.Done()
            if err := th.sendTransactionToPeer(peerAddr, messageJSON); err != nil {
                log.Printf("[TRANSACTION] Failed to propagate to peer %s: %v", peerAddr, err)
            } else {
                log.Printf("[TRANSACTION] Successfully propagated to peer %s", peerAddr)
            }
        }(peer)
    }
    wg.Wait()
    return nil
}

func (th *TransactionHandler) sendTransactionToPeer(peerAddr string, txData []byte) error {
    url := fmt.Sprintf("http://%s/receive-transaction", peerAddr)
    resp, err := http.Post(url, "application/json", bytes.NewBuffer(txData))
    if err != nil {
        return fmt.Errorf("failed to send transaction: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("peer returned status: %d", resp.StatusCode)
    }
    return nil
}

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
            Sequence:     tx.Sequence,
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
                    Sequence:     tx.Sequence,
                }, nil
            }
        }
    }

    return nil, fmt.Errorf("transaction not found")
}

func (th *TransactionHandler) HandleIncomingTransaction(txData []byte) error {
    var message struct {
        Transaction *Transaction `json:"transaction"`
        SenderID    string      `json:"sender_id"`
        MessageID   string      `json:"message_id"`
    }
    if err := json.Unmarshal(txData, &message); err != nil {
        return fmt.Errorf("failed to unmarshal transaction message: %v", err)
    }

    // Check if we've already processed this message
    th.msgMutex.Lock()
    if th.processedMsgs[message.MessageID] {
        th.msgMutex.Unlock()
        return nil
    }
    th.processedMsgs[message.MessageID] = true
    th.msgMutex.Unlock()

    // Submit the transaction
    _, err := th.SubmitTransaction(message.Transaction)
    if err != nil {
        return fmt.Errorf("failed to process incoming transaction: %v", err)
    }

    return nil
}

func (th *TransactionHandler) cleanupOldTransactions(maxAge time.Duration) {
    th.mu.Lock()
    defer th.mu.Unlock()

    now := time.Now()
    for id, tx := range th.pool {
        if now.Sub(tx.Timestamp) > maxAge {
            delete(th.pool, id)
            log.Printf("[TRANSACTION] Removed expired transaction %s from pool", id)
        }
    }
}

func (th *TransactionHandler) startMessageCleanup() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        th.msgMutex.Lock()
        th.processedMsgs = make(map[string]bool)
        th.msgMutex.Unlock()
        log.Printf("[TRANSACTION] Cleared processed message cache")
    }
}

func (th *TransactionHandler) GetPoolSize() int {
    th.mu.RLock()
    defer th.mu.RUnlock()
    return len(th.pool)
}

// This method would be implemented to get peer addresses from your peer management system
func (th *TransactionHandler) getPeerAddresses() []string {
    // This is a placeholder - you'll need to implement this to integrate with your peer management
    return []string{} // Return actual peer addresses from your peer management system
}

func (th *TransactionHandler) StartCleanup(cleanupInterval, maxAge time.Duration) {
    go func() {
        ticker := time.NewTicker(cleanupInterval)
        for range ticker.C {
            th.cleanupOldTransactions(maxAge)
        }
    }()
}
