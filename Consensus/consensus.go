package Consensus

import (
    "awesomeProject/blockchain"
    "bytes"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "errors"
    "fmt"
    "sort"
    "sync"
)

type ConsensusManager struct {
    mu          sync.RWMutex
    blockchains map[string]*blockchain.Blockchain
    difficulty  int
}

// Helper functions for network bytes conversion
func blockchainToBytes(bc *blockchain.Blockchain) ([]byte, error) {
    return json.Marshal(bc)
}

func blockchainFromBytes(data []byte) (*blockchain.Blockchain, error) {
    var bc blockchain.Blockchain
    err := json.Unmarshal(data, &bc)
    return &bc, err
}

func NewConsensusManager(difficulty int) *ConsensusManager {
    return &ConsensusManager{
        blockchains: make(map[string]*blockchain.Blockchain),
        difficulty:  difficulty,
    }
}

func (cm *ConsensusManager) ValidateAndSyncBlockchain(nodeID string, newBlockchainBytes []byte) error {
    cm.mu.Lock()
    defer cm.mu.Unlock()

    fmt.Printf("\n[CONSENSUS] Node %s validating blockchain\n", nodeID)
    
    var receivedBlockchain blockchain.Blockchain
    if err := json.Unmarshal(newBlockchainBytes, &receivedBlockchain); err != nil {
        return fmt.Errorf("invalid blockchain format: %v", err)
    }

    // For new nodes or initial blocks
    if len(receivedBlockchain.Blocks) == 0 {
        if _, exists := cm.blockchains[nodeID]; !exists {
            cm.blockchains[nodeID] = &receivedBlockchain
        }
        return nil
    }

    // Validate blockchain integrity
    if err := cm.validateBlockchainIntegrity(&receivedBlockchain); err != nil {
        return fmt.Errorf("blockchain integrity check failed: %v", err)
    }

    currentBC, exists := cm.blockchains[nodeID]
    if !exists || len(receivedBlockchain.Blocks) > len(currentBC.Blocks) {
        cm.blockchains[nodeID] = &receivedBlockchain
        fmt.Printf("[CONSENSUS] Blockchain updated for node %s, length: %d\n", 
            nodeID, len(receivedBlockchain.Blocks))
        return nil
    }

    return nil
}

func (cm *ConsensusManager) validateBlockchainIntegrity(bc *blockchain.Blockchain) error {
    // It's valid to have just one block (genesis block)
    if len(bc.Blocks) == 0 {
        return nil
    }

    // Validate chain of blocks
    for i := 1; i < len(bc.Blocks); i++ {
        currentBlock := bc.Blocks[i]
        previousBlock := bc.Blocks[i-1]

        // Verify previous hash
        if currentBlock.PreviousHash != previousBlock.Hash {
            return fmt.Errorf("invalid previous hash at block %d", i)
        }

        // Verify proof of work
        if !cm.verifyProofOfWork(currentBlock) {
            return fmt.Errorf("invalid proof of work at block %d", i)
        }
    }
    return nil
}

func (cm *ConsensusManager) verifyProofOfWork(block *blockchain.Block) bool {
    prefix := bytes.Repeat([]byte("0"), cm.difficulty)
    return bytes.HasPrefix([]byte(block.Hash), prefix)
}

func (cm *ConsensusManager) resolveConflict(nodeID string, newBlockchain *blockchain.Blockchain) error {
    currentBlockchain, exists := cm.blockchains[nodeID]
    if !exists || len(newBlockchain.Blocks) > len(currentBlockchain.Blocks) {
        cm.blockchains[nodeID] = newBlockchain
        fmt.Printf("[CONSENSUS] Accepting new blockchain from node %s\n", nodeID)
        return nil
    }

    if len(newBlockchain.Blocks) == len(currentBlockchain.Blocks) {
        if cm.compareBlockchainHashes(newBlockchain, currentBlockchain) {
            cm.blockchains[nodeID] = newBlockchain
            fmt.Printf("[CONSENSUS] Accepting new blockchain from node %s (same length, higher hash)\n", nodeID)
            return nil
        }
    }
    
    fmt.Printf("[CONSENSUS] Rejecting blockchain from node %s (shorter or lower hash)\n", nodeID)
    return errors.New("blockchain not accepted")
}

func (cm *ConsensusManager) compareBlockchainHashes(bc1, bc2 *blockchain.Blockchain) bool {
    hash1 := cm.calculateBlockchainHash(bc1)
    hash2 := cm.calculateBlockchainHash(bc2)
    return hash1 > hash2
}

func (cm *ConsensusManager) calculateBlockchainHash(bc *blockchain.Blockchain) string {
    hasher := sha256.New()
    hashes := make([]string, len(bc.Blocks))
    for i, block := range bc.Blocks {
        hashes[i] = block.Hash
    }
    sort.Strings(hashes)
    for _, hash := range hashes {
        hasher.Write([]byte(hash))
    }
    return hex.EncodeToString(hasher.Sum(nil))
}

func (cm *ConsensusManager) BroadcastBlockchain(nodeID string) ([]byte, error) {
    cm.mu.RLock()
    defer cm.mu.RUnlock()
    
    blockchain, exists := cm.blockchains[nodeID]
    if !exists {
        return nil, errors.New("no blockchain found for node")
    }
    return blockchainToBytes(blockchain)
}

func (cm *ConsensusManager) GetNodeBlockchain(nodeID string) (*blockchain.Blockchain, error) {
    cm.mu.RLock()
    defer cm.mu.RUnlock()
    
    blockchain, exists := cm.blockchains[nodeID]
    if !exists {
        return nil, fmt.Errorf("no blockchain found for node %s", nodeID)
    }
    return blockchain, nil
}


