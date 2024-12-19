package Peer2Peer

import (
    "awesomeProject/Consensus"
    "awesomeProject/blockchain"
    "context"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "sync"
)

type BlockchainNetwork struct {
    peerManager         *PeerManager
    blockchain          *blockchain.Blockchain
    mu                  sync.Mutex
    consensusValidator  *Consensus.NetworkBlockchainValidator
    nodeID             string
    blockPropagator    *BlockPropagator
}

func NewBlockchainNetwork(localPort int, blockchain *blockchain.Blockchain, nodeID string) *BlockchainNetwork {
    consensusValidator := Consensus.NewNetworkBlockchainValidator(4)
    
    bn := &BlockchainNetwork{
        peerManager:        NewPeerManager(localPort),
        blockchain:         blockchain,
        consensusValidator: consensusValidator,
        nodeID:            nodeID,
    }
    
    // Initialize block propagator with consensus validator
    bn.blockPropagator = NewBlockPropagator(
        bn.peerManager,
        consensusValidator,
        nodeID,
        blockchain,
    )
    
    return bn
}

func (bn *BlockchainNetwork) StartServer(listenAddress string) error {
    http.HandleFunc("/receive-block", bn.handleReceiveBlock)
    http.HandleFunc("/register-peer", bn.registerPeerHandler)
    http.HandleFunc("/get-blockchain", bn.getBlockchainHandler)

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    bn.peerManager.DiscoverPeers(ctx)

    log.Printf("Starting blockchain network server on %s", listenAddress)
    return http.ListenAndServe(listenAddress, nil)
}

func (bn *BlockchainNetwork) handleReceiveBlock(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Error reading request body", http.StatusBadRequest)
        return
    }

    // First validate through consensus
    err = bn.consensusValidator.HandleBlockchainSync(bn.nodeID, body)
    if err != nil {
        http.Error(w, fmt.Sprintf("Consensus validation failed: %v", err), http.StatusBadRequest)
        return
    }

    // If consensus is achieved, process the block
    err = bn.blockPropagator.HandleIncomingBlock(body)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error processing block: %v", err), http.StatusBadRequest)
        return
    }

    w.WriteHeader(http.StatusOK)
}

func (bn *BlockchainNetwork) registerPeerHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    peerAddress := r.Header.Get("X-Peer-Address")
    if peerAddress == "" {
        http.Error(w, "Peer address required", http.StatusBadRequest)
        return
    }

    bn.peerManager.AddPeer(peerAddress)
    w.WriteHeader(http.StatusOK)
}

func (bn *BlockchainNetwork) getBlockchainHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    bn.mu.Lock()
    blockchainJSON, err := bn.blockchain.ToJSON()
    bn.mu.Unlock()

    if err != nil {
        http.Error(w, "Error serializing blockchain", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write(blockchainJSON)
}

func (bn *BlockchainNetwork) PropagateNewBlock(block *blockchain.Block) error {
    // First validate the block through consensus
    blockchainJSON, err := bn.blockchain.ToJSON()
    if err != nil {
        return fmt.Errorf("error converting blockchain to JSON: %v", err)
    }

    err = bn.consensusValidator.HandleBlockchainSync(bn.nodeID, blockchainJSON)
    if err != nil {
        return fmt.Errorf("consensus validation failed: %v", err)
    }

    // If consensus is achieved, propagate the block
    return bn.blockPropagator.PropagateNewBlock(block)
}

func (bn *BlockchainNetwork) GetPeerManager() *PeerManager {
    return bn.peerManager
}

func (bn *BlockchainNetwork) GetConsensusValidator() *Consensus.NetworkBlockchainValidator {
    return bn.consensusValidator
}
