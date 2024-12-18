package consensus

import (
	"awesomeProject/chain"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
)

type ConsensusManager struct {
	mu          sync.RWMutex
	blockchains map[string]*chain.Blockchain
	difficulty  int
}

func NewConsensusManager(difficulty int) *ConsensusManager {
	return &ConsensusManager{
		blockchains: make(map[string]*chain.Blockchain),
		difficulty:  difficulty,
	}
}

func (cm *ConsensusManager) ValidateAndSyncBlockchain(nodeID string, newBlockchain []byte) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	receivedBlockchain := &chain.Blockchain{}
	err := receivedBlockchain.FromNetworkBytes(newBlockchain)
	if err != nil {
		return fmt.Errorf("invalid chain format: %v", err)
	}

	if err := cm.validateBlockchainIntegrity(receivedBlockchain); err != nil {
		return fmt.Errorf("chain integrity check failed: %v", err)
	}

	return cm.resolveConflict(nodeID, receivedBlockchain)
}

func (cm *ConsensusManager) validateBlockchainIntegrity(bc *chain.Blockchain) error {
	if len(bc.Blocks) == 0 {
		return errors.New("empty chain")
	}

	for i := 1; i < len(bc.Blocks); i++ {
		currentBlock := bc.Blocks[i]
		prevBlock := bc.Blocks[i-1]

		if currentBlock.PreviousHash != prevBlock.Hash {
			return fmt.Errorf("invalid previous hash at block %d", i)
		}

		if !cm.verifyProofOfWork(currentBlock) {
			return fmt.Errorf("invalid proof of work at block %d", i)
		}
	}

	return nil
}

func (cm *ConsensusManager) verifyProofOfWork(block *chain.Block) bool {
	prefix := bytes.Repeat([]byte("0"), cm.difficulty)
	hash := block.Hash
	return bytes.HasPrefix([]byte(hash), prefix)
}

func (cm *ConsensusManager) resolveConflict(nodeID string, newBlockchain *chain.Blockchain) error {
	currentBlockchain, exists := cm.blockchains[nodeID]

	if !exists || len(newBlockchain.Blocks) > len(currentBlockchain.Blocks) {
		cm.blockchains[nodeID] = newBlockchain
		return nil
	}

	if len(newBlockchain.Blocks) == len(currentBlockchain.Blocks) {
		if cm.compareBlockchainHashes(newBlockchain, currentBlockchain) {
			cm.blockchains[nodeID] = newBlockchain
			return nil
		}
	}

	return errors.New("chain not accepted")
}
func (cm *ConsensusManager) calculateBlockchainHash(bc *chain.Blockchain) string {
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

func (cm *ConsensusManager) compareBlockchainHashes(bc1, bc2 *chain.Blockchain) bool {
	hash1 := cm.calculateBlockchainHash(bc1)
	hash2 := cm.calculateBlockchainHash(bc2)
	return hash1 > hash2
}

func (cm *ConsensusManager) BroadcastBlockchain(nodeID string) ([]byte, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	blockchain, exists := cm.blockchains[nodeID]
	if !exists {
		return nil, errors.New("no chain found for node")
	}

	return blockchain.ToNetworkBytes()
}

func (cm *ConsensusManager) GetNodeBlockchain(nodeID string) (*chain.Blockchain, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	blockchain, exists := cm.blockchains[nodeID]
	if !exists {
		return nil, fmt.Errorf("no chain found for node %s", nodeID)
	}

	return blockchain, nil
}
