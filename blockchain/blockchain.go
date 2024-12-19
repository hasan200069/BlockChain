package blockchain

import (
	"awesomeProject/util"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Blockchain struct {
    Blocks []*Block
}

const Difficulty = 4

func (b *Block) MineBlock(difficulty int) {
    prefix := bytes.Repeat([]byte("0"), difficulty)
    fmt.Printf("[MINING] Starting mining process for new block...\n")
    startTime := time.Now()
    
    for {
        b.Hash = util.GenerateHash(b)
        if bytes.HasPrefix([]byte(b.Hash), prefix) {
            break
        }
        b.Nonce++
    }
    
    duration := time.Since(startTime)
    fmt.Printf("[MINING] Block mined with hash %s in %v\n", b.Hash, duration)
}

func (bc *Blockchain) getPreviousHash() string {
    if len(bc.Blocks) > 0 {
        return bc.Blocks[len(bc.Blocks)-1].Hash // Changed from getPreviousHash() to directly access Hash
    }
    return ""
}

func (bc *Blockchain) CreateNewBlock(transactions []*Transaction) *Block {
    fmt.Printf("[BLOCKCHAIN] Creating new block with %d transactions\n", len(transactions))
    
    previousHash := bc.getPreviousHash()
    newBlock := &Block{
        PreviousHash: previousHash,
        Timestamp:    time.Now(),
        Data:         Data{Transactions: transactions},
        Nonce:        0,
    }
    
    // Mine the block
    newBlock.MineBlock(Difficulty)
    
    // Validate block before adding
    if err := bc.ValidateBlock(newBlock); err != nil {
        fmt.Printf("[BLOCKCHAIN] Block validation failed: %v\n", err)
        return nil
    }
    
    // Add to blockchain
    bc.Blocks = append(bc.Blocks, newBlock)
    fmt.Printf("[BLOCKCHAIN] New block added to chain. Length: %d\n", len(bc.Blocks))
    
    return newBlock
}

func (bc *Blockchain) ValidateBlock(block *Block) error {
    // For genesis block
    if len(bc.Blocks) == 0 {
        return nil
    }

    lastBlock := bc.Blocks[len(bc.Blocks)-1]
    
    // Validate previous hash
    if block.PreviousHash != lastBlock.Hash {
        return fmt.Errorf("invalid previous hash")
    }

    // Validate timestamp
    if block.Timestamp.Before(lastBlock.Timestamp) {
        return fmt.Errorf("invalid timestamp")
    }

    // Validate proof of work
    prefix := bytes.Repeat([]byte("0"), Difficulty)
    if !bytes.HasPrefix([]byte(block.Hash), prefix) {
        return fmt.Errorf("invalid proof of work")
    }

    // Validate transactions
    if len(block.Data.Transactions) == 0 {
        return fmt.Errorf("block must contain at least one transaction")
    }

    for _, tx := range block.Data.Transactions {
        if err := tx.Verify(); err != nil {
            return fmt.Errorf("invalid transaction: %v", err)
        }
    }

    return nil
}

func (bc *Blockchain) AddBlock(block *Block) error {
    if err := bc.ValidateBlock(block); err != nil {
        return fmt.Errorf("block validation failed: %v", err)
    }

    bc.Blocks = append(bc.Blocks, block)
    fmt.Printf("[BLOCKCHAIN] Block added to chain. New length: %d\n", len(bc.Blocks))
    return nil
}

func (bc *Blockchain) GetLastBlock() *Block {
    if len(bc.Blocks) == 0 {
        return nil
    }
    return bc.Blocks[len(bc.Blocks)-1]
}

func (bc *Blockchain) PrintChain() {
    if len(bc.Blocks) == 0 {
        fmt.Println("[BLOCKCHAIN] Blockchain is empty.")
        return
    }

    fmt.Println("\n[BLOCKCHAIN] Current State:")
    for i, block := range bc.Blocks {
        fmt.Printf("\nBlock %d:\n", i+1)
        fmt.Printf("Previous Hash: %s\n", block.PreviousHash)
        fmt.Printf("Timestamp: %s\n", block.Timestamp)
        fmt.Printf("Nonce: %d\n", block.Nonce)
        fmt.Printf("Hash: %s\n", block.Hash)
        fmt.Printf("Transactions (%d):\n", len(block.Data.Transactions))
        for j, transaction := range block.Data.Transactions {
            fmt.Printf("  %d. %v\n", j+1, transaction)
        }
    }
    fmt.Println()
}

func (bc *Blockchain) ToJSON() ([]byte, error) {
    data, err := json.MarshalIndent(bc, "", "  ")
    if err != nil {
        return nil, fmt.Errorf("failed to marshal blockchain: %v", err)
    }
    return data, nil
}

func (bc *Blockchain) SaveToFile(filename string) error {
    data, err := bc.ToJSON()
    if err != nil {
        return fmt.Errorf("failed to convert blockchain to JSON: %v", err)
    }

    if err := os.WriteFile(filename, data, 0644); err != nil {
        return fmt.Errorf("failed to write blockchain to file: %v", err)
    }

    fmt.Printf("[BLOCKCHAIN] Successfully saved to %s\n", filename)
    return nil
}

func (bc *Blockchain) LoadFromJSON(data []byte) error {
    if err := json.Unmarshal(data, bc); err != nil {
        return fmt.Errorf("failed to unmarshal blockchain: %v", err)
    }
    
    fmt.Printf("[BLOCKCHAIN] Successfully loaded blockchain with %d blocks\n", len(bc.Blocks))
    return nil
}
