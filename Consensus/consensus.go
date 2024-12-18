package Consensus

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
)

type ConsensusManager struct {
	mu          sync.RWMutex
	blockchains map[string]*Blockchain
	difficulty  int
}

func NewConsensusManager(difficulty int) *ConsensusManager {
	return &ConsensusManager{
		blockchains: make(map[string]*Blockchain),
		difficulty:  difficulty,
	}
}

func (cm *ConsensusManager) ValidateAndSyncBlockchain(nodeID string, newBlockchain []byte) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	receivedBlockchain := &Blockchain{}
	err := receivedBlockchain.FromNetworkBytes(newBlockchain)
	if err != nil {
		return fmt.Errorf("invalid blockchain format: %v", err)
	}

	if err := cm.validateBlockchainIntegrity(receivedBlockchain); err != nil {
		return fmt.Errorf("blockchain integrity check failed: %v", err)
	}

	return cm.resolveConflict(nodeID, receivedBlockchain)
}

func (cm *ConsensusManager) validateBlockchainIntegrity(bc *Blockchain) error {
	if len(bc.Blocks) == 0 {
		return errors.New("empty blockchain")
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

func (cm *ConsensusManager) verifyProofOfWork(block *Block) bool {
	prefix := bytes.Repeat([]byte("0"), cm.difficulty)
	hash := block.Hash
	return bytes.HasPrefix([]byte(hash), prefix)
}

func (cm *ConsensusManager) resolveConflict(nodeID string, newBlockchain *Blockchain) error {
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

	return errors.New("blockchain not accepted")
}

func (cm *ConsensusManager) compareBlockchainHashes(bc1, bc2 *Blockchain) bool {
	hash1 := cm.calculateBlockchainHash(bc1)
	hash2 := cm.calculateBlockchainHash(bc2)
	return hash1 > hash2
}

func (cm *ConsensusManager) calculateBlockchainHash(bc *Blockchain) string {
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

	return blockchain.ToNetworkBytes()
}

func (cm *ConsensusManager) GetNodeBlockchain(nodeID string) (*Blockchain, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	blockchain, exists := cm.blockchains[nodeID]
	if !exists {
		return nil, fmt.Errorf("no blockchain found for node %s", nodeID)
	}

	return blockchain, nil
}

type NetworkBlockchainValidator struct {
	consensusManager *ConsensusManager
}

func NewNetworkBlockchainValidator(difficulty int) *NetworkBlockchainValidator {
	return &NetworkBlockchainValidator{
		consensusManager: NewConsensusManager(difficulty),
	}
}

func (nbv *NetworkBlockchainValidator) ValidateIncomingBlockchain(nodeID string, blockchainData []byte) error {
	return nbv.consensusManager.ValidateAndSyncBlockchain(nodeID, blockchainData)
}

func (nbv *NetworkBlockchainValidator) HandleBlockchainSync(nodeID string, receivedBlockchain []byte) error {
	err := nbv.ValidateIncomingBlockchain(nodeID, receivedBlockchain)
	if err != nil {
		return fmt.Errorf("blockchain sync failed: %v", err)
	}

	return nil
}
