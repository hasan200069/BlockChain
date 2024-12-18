package consensus

import "fmt"

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
		return fmt.Errorf("chain sync failed: %v", err)
	}

	return nil
}
