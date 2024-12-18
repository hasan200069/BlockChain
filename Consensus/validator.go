package Consensus

import (
    "fmt"
)

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
    fmt.Printf("[CONSENSUS] Handling blockchain sync for node %s\n", nodeID)
    err := nbv.ValidateIncomingBlockchain(nodeID, receivedBlockchain)
    if err != nil {
        fmt.Printf("[CONSENSUS ERROR] Blockchain sync failed: %v\n", err)
        return fmt.Errorf("blockchain sync failed: %v", err)
    }
    fmt.Printf("[CONSENSUS] Blockchain sync successful for node %s\n", nodeID)
    return nil
}
