package Peer2Peer

import (
    "awesomeProject/blockchain"
    "awesomeProject/Consensus"
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "sync"
    "time"
)

type BlockMessage struct {
    Block       *blockchain.Block `json:"block"`
    NodeID      string           `json:"node_id"`
    Timestamp   time.Time        `json:"timestamp"`
    MessageID   string           `json:"message_id"`
    BlockNumber int              `json:"block_number"`
}

type BlockPropagator struct {
    peerManager         *PeerManager
    consensusValidator  *Consensus.NetworkBlockchainValidator
    processedMessages   map[string]bool
    mu                  sync.RWMutex
    nodeID             string
    blockchain         *blockchain.Blockchain
}

func NewBlockPropagator(peerManager *PeerManager, validator *Consensus.NetworkBlockchainValidator, nodeID string, blockchain *blockchain.Blockchain) *BlockPropagator {
    bp := &BlockPropagator{
        peerManager:       peerManager,
        consensusValidator: validator,
        processedMessages: make(map[string]bool),
        nodeID:           nodeID,
        blockchain:       blockchain,
    }
    go bp.startCleanup()
    return bp
}

func (bp *BlockPropagator) PropagateNewBlock(block *blockchain.Block) error {
    message := BlockMessage{
        Block:       block,
        NodeID:      bp.nodeID,
        Timestamp:   time.Now(),
        MessageID:   fmt.Sprintf("%s-%d", bp.nodeID, time.Now().UnixNano()),
        BlockNumber: len(bp.blockchain.Blocks),
    }

    messageJSON, err := json.Marshal(message)
    if err != nil {
        return fmt.Errorf("failed to marshal block message: %v", err)
    }

    log.Printf("[PROPAGATOR] Propagating block number %d to peers", message.BlockNumber)

    var wg sync.WaitGroup
    peers := bp.peerManager.GetPeers()
    
    for _, peer := range peers {
        wg.Add(1)
        go func(peerAddr string) {
            defer wg.Done()
            if err := bp.sendBlockToPeer(peerAddr, messageJSON); err != nil {
                log.Printf("[PROPAGATOR] Failed to send block to %s: %v", peerAddr, err)
            } else {
                log.Printf("[PROPAGATOR] Successfully sent block to %s", peerAddr)
            }
        }(peer)
    }

    wg.Wait()
    return nil
}

func (bp *BlockPropagator) sendBlockToPeer(peerAddr string, blockData []byte) error {
    url := fmt.Sprintf("http://%s/receive-block", peerAddr)
    
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(blockData))
    if err != nil {
        return fmt.Errorf("error creating request: %v", err)
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        bp.peerManager.RemovePeer(peerAddr)
        return fmt.Errorf("error sending block: %v", err)
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("peer rejected block: %s", string(body))
    }

    return nil
}

func (bp *BlockPropagator) HandleIncomingBlock(blockData []byte) error {
    var message BlockMessage
    if err := json.Unmarshal(blockData, &message); err != nil {
        return fmt.Errorf("failed to unmarshal block message: %v", err)
    }

    // Check if we've already processed this block
    bp.mu.Lock()
    if bp.processedMessages[message.MessageID] {
        bp.mu.Unlock()
        log.Printf("[PROPAGATOR] Block already processed, skipping: %s", message.MessageID)
        return nil
    }
    bp.processedMessages[message.MessageID] = true
    bp.mu.Unlock()

    log.Printf("[PROPAGATOR] Received new block %d from node %s", message.BlockNumber, message.NodeID)

    // Don't propagate our own blocks back to us
    if message.NodeID == bp.nodeID {
        log.Printf("[PROPAGATOR] Skipping propagation of our own block")
        return nil
    }

    // Validate the block
    if err := bp.validateBlock(message.Block); err != nil {
        return fmt.Errorf("block validation failed: %v", err)
    }

    // Add block to blockchain
    if err := bp.blockchain.AddBlock(message.Block); err != nil {
        return fmt.Errorf("failed to add block: %v", err)
    }

    log.Printf("[PROPAGATOR] Successfully added block %d to blockchain", message.BlockNumber)

    // Only propagate to other peers if we're not the original sender
    if message.NodeID != bp.nodeID {
        // Propagate to other peers after a small delay to avoid network congestion
        time.Sleep(100 * time.Millisecond)
        go bp.propagateToOthers(blockData, message.NodeID)
    }

    return nil
}

func (bp *BlockPropagator) validateBlock(block *blockchain.Block) error {
    currentBlocks := len(bp.blockchain.Blocks)
    
    // Special case for the first block
    if currentBlocks == 0 {
        // For genesis block, previousHash should be empty
        if block.PreviousHash != "" {
            return fmt.Errorf("invalid genesis block: should have empty previous hash")
        }
        return nil
    }

    lastBlock := bp.blockchain.Blocks[currentBlocks-1]
    
    // Validate previous hash
    if block.PreviousHash != lastBlock.Hash {
        return fmt.Errorf("invalid previous hash")
    }

    // Validate proof of work
    prefix := bytes.Repeat([]byte("0"), 4) // difficulty level
    if !bytes.HasPrefix([]byte(block.Hash), prefix) {
        return fmt.Errorf("invalid proof of work")
    }

    return nil
}

func (bp *BlockPropagator) propagateToOthers(blockData []byte, senderID string) {
    peers := bp.peerManager.GetPeers()
    var wg sync.WaitGroup

    for _, peer := range peers {
        // Skip the original sender
        if peer == senderID {
            continue
        }

        wg.Add(1)
        go func(peerAddr string) {
            defer wg.Done()
            if err := bp.sendBlockToPeer(peerAddr, blockData); err != nil {
                log.Printf("[PROPAGATOR] Failed to propagate block to %s: %v", peerAddr, err)
            } else {
                log.Printf("[PROPAGATOR] Successfully propagated block to %s", peerAddr)
            }
        }(peer)
    }

    wg.Wait()
}

func (bp *BlockPropagator) startCleanup() {
    for {
        time.Sleep(1 * time.Hour)
        bp.mu.Lock()
        bp.processedMessages = make(map[string]bool)
        bp.mu.Unlock()
    }
}
