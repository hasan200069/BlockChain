package Peer2Peer

import (
    "awesomeProject/blockchain"
    "awesomeProject/Consensus"
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
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

    fmt.Printf("[PROPAGATOR] Propagating block number %d to peers\n", message.BlockNumber)

    var wg sync.WaitGroup
    peers := bp.peerManager.GetPeers()
    
    for _, peer := range peers {
        wg.Add(1)
        go func(peerAddr string) {
            defer wg.Done()
            if err := bp.sendBlockToPeer(peerAddr, messageJSON); err != nil {
                fmt.Printf("[PROPAGATOR] Failed to send block to %s: %v\n", peerAddr, err)
            } else {
                fmt.Printf("[PROPAGATOR] Successfully sent block to %s\n", peerAddr)
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

    bp.mu.Lock()
    if bp.processedMessages[message.MessageID] {
        bp.mu.Unlock()
        return nil
    }
    bp.processedMessages[message.MessageID] = true
    bp.mu.Unlock()

    fmt.Printf("[PROPAGATOR] Received new block %d from node %s\n", message.BlockNumber, message.NodeID)

    // Validate the block
    if err := bp.validateBlock(message.Block); err != nil {
        return fmt.Errorf("block validation failed: %v", err)
    }

    // Add block to blockchain
    if err := bp.blockchain.AddBlock(message.Block); err != nil {
        return fmt.Errorf("failed to add block: %v", err)
    }

    fmt.Printf("[PROPAGATOR] Successfully added block %d to blockchain\n", message.BlockNumber)

    // Propagate to other peers
    if message.NodeID != bp.nodeID {
        go bp.propagateToOthers(blockData, message.NodeID)
    }

    return nil
}

func (bp *BlockPropagator) validateBlock(block *blockchain.Block) error {
    if len(bp.blockchain.Blocks) > 0 {
        lastBlock := bp.blockchain.Blocks[len(bp.blockchain.Blocks)-1]
        if block.PreviousHash != lastBlock.Hash {
            return fmt.Errorf("invalid previous hash")
        }
    }

    // Validate proof of work
    prefix := bytes.Repeat([]byte("0"), 4)
    if !bytes.HasPrefix([]byte(block.Hash), prefix) {
        return fmt.Errorf("invalid proof of work")
    }

    // Validate transactions
    for _, tx := range block.Data.Transactions {
        if err := tx.Verify(); err != nil {
            return fmt.Errorf("invalid transaction: %v", err)
        }
    }

    return nil
}

func (bp *BlockPropagator) propagateToOthers(blockData []byte, senderID string) {
    peers := bp.peerManager.GetPeers()
    var wg sync.WaitGroup

    for _, peer := range peers {
        if peer == senderID {
            continue
        }

        wg.Add(1)
        go func(peerAddr string) {
            defer wg.Done()
            if err := bp.sendBlockToPeer(peerAddr, blockData); err != nil {
                fmt.Printf("[PROPAGATOR] Failed to propagate block to %s: %v\n", peerAddr, err)
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
