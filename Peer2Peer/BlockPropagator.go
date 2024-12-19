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

    // Validate through consensus before propagating
    err = bp.consensusValidator.HandleBlockchainSync(bp.nodeID, messageJSON)
    if err != nil {
        return fmt.Errorf("consensus validation failed: %v", err)
    }

    var wg sync.WaitGroup
    peers := bp.peerManager.GetPeers()
    
    for _, peer := range peers {
        wg.Add(1)
        go func(peerAddr string) {
            defer wg.Done()
            bp.sendBlockToPeer(peerAddr, messageJSON)
        }(peer)
    }

    wg.Wait()
    return nil
}

func (bp *BlockPropagator) sendBlockToPeer(peerAddr string, blockData []byte) {
    url := fmt.Sprintf("http://%s/receive-block", peerAddr)
    
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(blockData))
    if err != nil {
        fmt.Printf("Error creating request for peer %s: %v\n", peerAddr, err)
        return
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        fmt.Printf("Error sending block to peer %s: %v\n", peerAddr, err)
        bp.peerManager.RemovePeer(peerAddr)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := ioutil.ReadAll(resp.Body)
        fmt.Printf("Peer %s rejected block: %s\n", peerAddr, string(body))
    }
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

    // Validate through consensus
    if err := bp.consensusValidator.HandleBlockchainSync(bp.nodeID, blockData); err != nil {
        return fmt.Errorf("consensus validation failed: %v", err)
    }

    // Additional block validation
    if err := bp.validateBlock(message.Block); err != nil {
        return fmt.Errorf("block validation failed: %v", err)
    }

    // Add the block to blockchain
    bp.blockchain.Blocks = append(bp.blockchain.Blocks, message.Block)

    // Propagate to other peers
    go bp.propagateToOthers(blockData, message.NodeID)

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
            bp.sendBlockToPeer(peerAddr, blockData)
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
