package chain

import (
    "fmt"
    "sync"
    "time"
    "sync/atomic"
)

const StatusPending = "pending"

var globalSequence uint64

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
    SenderBalanceBefore   int `json:"sender_balance_before"`
    SenderBalanceAfter    int `json:"sender_balance_after"`
    ReceiverBalanceBefore int `json:"receiver_balance_before"`
    ReceiverBalanceAfter  int `json:"receiver_balance_after"`
    Algorithm    string    `json:"algorithm"`
    ResultHash   string    `json:"result_hash"`
    ActualOutput string    `json:"actual_output"`
    InputRefs    []string  `json:"input_refs"`
    InputAmounts []int     `json:"input_amounts"`
    LastSeen     time.Time `json:"last_seen"`
    SpentOutputs []string  `json:"spent_outputs"`
}

type TransactionPool struct {
    mu                 sync.RWMutex
    PendingTransactions map[string]*Transaction
    txByAddress        map[string][]*Transaction
    feeThreshold       int
    maxSize            int
    totalSize          int
}

func NewTransactionPool(feeThreshold, maxSize int) *TransactionPool {
    return &TransactionPool{
        PendingTransactions: make(map[string]*Transaction),
        txByAddress:        make(map[string][]*Transaction),
        feeThreshold:       feeThreshold,
        maxSize:            maxSize,
        totalSize:          0,
    }
}

func (bc *Blockchain) GetAccountBalance(address string) int {
    lastTx := bc.GetLastTransaction(address)
    if lastTx == nil {
        return 0
    }
    
    if lastTx.Sender == address {
        return lastTx.SenderBalanceAfter
    }
    return lastTx.ReceiverBalanceAfter
}

func (bc *Blockchain) GetLastTransaction(address string) *Transaction {
    for i := len(bc.Blocks) - 1; i >= 0; i-- {
        block := bc.Blocks[i]
        for j := len(block.Data.Transactions) - 1; j >= 0; j-- {
            tx := block.Data.Transactions[j]
            if tx.Sender == address || tx.Receiver == address {
                return tx
            }
        }
    }
    return nil
}

func CreateTransaction(sender, receiver string, amount, fee int, nonce uint64, blockchain *Blockchain) (*Transaction, error) {
    senderBalance := blockchain.GetAccountBalance(sender)
    receiverBalance := blockchain.GetAccountBalance(receiver)
    
    if senderBalance < amount+fee {
        return nil, fmt.Errorf("insufficient funds: balance %d, required %d", senderBalance, amount+fee)
    }
    
    newSenderBalance := senderBalance - (amount + fee)
    newReceiverBalance := receiverBalance + amount
    
    sequence := atomic.AddUint64(&globalSequence, 1)
    
    tx := &Transaction{
        TransactionID:         generateTransactionID(sender, sequence),
        Sequence:             sequence,
        Timestamp:            time.Now(),
        Sender:               sender,
        Receiver:             receiver,
        Amount:               amount,
        Fee:                  fee,
        Nonce:               nonce,
        Status:              StatusPending,
        SenderBalanceBefore: senderBalance,
        SenderBalanceAfter:  newSenderBalance,
        ReceiverBalanceBefore: receiverBalance,
        ReceiverBalanceAfter:  newReceiverBalance,
        LastSeen:            time.Now(),
        SpentOutputs:        make([]string, 0),
    }
    
    return tx, nil
}

func (tx *Transaction) Verify() error {
    if err := tx.verifyBasics(); err != nil {
        return err
    }
    
    if tx.SenderBalanceBefore - (tx.Amount + tx.Fee) != tx.SenderBalanceAfter {
        return fmt.Errorf("invalid sender balance calculation")
    }
    
    if tx.ReceiverBalanceBefore + tx.Amount != tx.ReceiverBalanceAfter {
        return fmt.Errorf("invalid receiver balance calculation")
    }
    
    if tx.SenderBalanceAfter < 0 {
        return fmt.Errorf("transaction would result in negative balance")
    }
    
    return nil
}

func (tx *Transaction) verifyBasics() error {
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
    if tx.Timestamp.After(time.Now()) {
        return fmt.Errorf("invalid timestamp: future time not allowed")
    }
    return nil
}

func (pool *TransactionPool) AddTransaction(tx *Transaction, blockchain *Blockchain) error {
    pool.mu.Lock()
    defer pool.mu.Unlock()
    
    if pool.totalSize >= pool.maxSize {
        return fmt.Errorf("transaction pool is full")
    }
    
    if tx.Fee < pool.feeThreshold {
        return fmt.Errorf("fee below minimum threshold: got %d, need %d", tx.Fee, pool.feeThreshold)
    }
    
    if err := tx.Verify(); err != nil {
        return fmt.Errorf("transaction verification failed: %v", err)
    }
    
    currentBalance := blockchain.GetAccountBalance(tx.Sender)
    if currentBalance != tx.SenderBalanceBefore {
        return fmt.Errorf("invalid sender balance state: expected %d, got %d", 
            tx.SenderBalanceBefore, currentBalance)
    }
    
    pendingDebit := 0
    for _, pendingTx := range pool.PendingTransactions {
        if pendingTx.Sender == tx.Sender {
            pendingDebit += (pendingTx.Amount + pendingTx.Fee)
        }
    }
    
    if currentBalance - pendingDebit < tx.Amount + tx.Fee {
        return fmt.Errorf("insufficient balance including pending transactions")
    }
    
    if _, exists := pool.PendingTransactions[tx.TransactionID]; exists {
        return fmt.Errorf("transaction already in pool")
    }
    
    pool.PendingTransactions[tx.TransactionID] = tx
    pool.txByAddress[tx.Sender] = append(pool.txByAddress[tx.Sender], tx)
    pool.totalSize++
    
    return nil
}

func (pool *TransactionPool) RemoveTransaction(txID string) {
    pool.mu.Lock()
    defer pool.mu.Unlock()
    
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

func (pool *TransactionPool) GetTransactionsByAddress(address string) []*Transaction {
    pool.mu.RLock()
    defer pool.mu.RUnlock()
    
    if txs, exists := pool.txByAddress[address]; exists {
        return append([]*Transaction{}, txs...)
    }
    return nil
}

func (pool *TransactionPool) CleanExpiredTransactions(maxAge time.Duration) {
    pool.mu.Lock()
    defer pool.mu.Unlock()
    
    now := time.Now()
    for txID, tx := range pool.PendingTransactions {
        if now.Sub(tx.Timestamp) > maxAge {
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
}

func (pool *TransactionPool) GetSize() int {
    pool.mu.RLock()
    defer pool.mu.RUnlock()
    return pool.totalSize
}

func (pool *TransactionPool) GetFeeThreshold() int {
    pool.mu.RLock()
    defer pool.mu.RUnlock()
    return pool.feeThreshold
}

func (pool *TransactionPool) SetFeeThreshold(newThreshold int) {
    pool.mu.Lock()
    defer pool.mu.Unlock()
    pool.feeThreshold = newThreshold
}

func (pool *TransactionPool) Clear() {
    pool.mu.Lock()
    defer pool.mu.Unlock()
    
    pool.PendingTransactions = make(map[string]*Transaction)
    pool.txByAddress = make(map[string][]*Transaction)
    pool.totalSize = 0
}

func generateTransactionID(sender string, sequence uint64) string {
    return fmt.Sprintf("%s-%d", sender, sequence)
}
