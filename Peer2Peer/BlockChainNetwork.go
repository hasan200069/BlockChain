package Peer2Peer

import (
    "awesomeProject/Consensus"
    "awesomeProject/blockchain"
    "context"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net"
    "net/http"
    "sync"
    "time"
)

type BlockchainNetwork struct {
    peerManager         *PeerManager
    blockchain          *blockchain.Blockchain
    mu                  sync.Mutex
    consensusValidator  *Consensus.NetworkBlockchainValidator
    nodeID             string
    blockPropagator    *BlockPropagator
    isConnected        bool
    bootstrapAddress   string
}

func NewBlockchainNetwork(localPort int, blockchain *blockchain.Blockchain, nodeID string) *BlockchainNetwork {
    consensusValidator := Consensus.NewNetworkBlockchainValidator(4)
    
    bn := &BlockchainNetwork{
        peerManager:        NewPeerManager(localPort),
        blockchain:         blockchain,
        consensusValidator: consensusValidator,
        nodeID:            nodeID,
    }
    
    bn.blockPropagator = NewBlockPropagator(
        bn.peerManager,
        consensusValidator,
        nodeID,
        blockchain,
    )
    
    // Start network monitoring
    go bn.monitorNetwork()
    
    return bn
}

func (bn *BlockchainNetwork) StartServer(listenAddress string) error {
    http.HandleFunc("/receive-block", bn.handleReceiveBlock)
    http.HandleFunc("/register-peer", bn.registerPeerHandler)
    http.HandleFunc("/get-blockchain", bn.getBlockchainHandler)
    http.HandleFunc("/peer-health", bn.handlePeerHealth)
    http.HandleFunc("/network-info", bn.handleNetworkInfo)

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    bn.peerManager.DiscoverPeers(ctx)

    log.Printf("[NETWORK] Starting blockchain network server on %s", listenAddress)
    return http.ListenAndServe(listenAddress, nil)
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

    // Check if peer is already registered to avoid loops
    bn.mu.Lock()
    defer bn.mu.Unlock()

    if bn.peerManager.IsPeerRegistered(peerAddress) {
        response := map[string]interface{}{
            "status": "already_connected",
            "message": "Peer already registered",
            "my_address": fmt.Sprintf("%s:%d", getOutboundIP(), bn.peerManager.localPort),
            "network_size": len(bn.peerManager.GetPeers()),
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
        return
    }

    // Add the peer
    bn.peerManager.AddPeer(peerAddress)
    currentPeerCount := len(bn.peerManager.GetPeers())

    // Log new peer registration
    log.Printf("[NETWORK] New peer registered: %s. Network size: %d", 
        peerAddress, currentPeerCount)

    response := map[string]interface{}{
        "status": "connected",
        "your_address": peerAddress,
        "my_address": fmt.Sprintf("%s:%d", getOutboundIP(), bn.peerManager.localPort),
        "network_size": currentPeerCount,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
// Add this new method to BlockchainNetwork
func (bn *BlockchainNetwork) registerWithPeer(peerAddr, myAddr string) error {
    client := &http.Client{Timeout: 5 * time.Second}
    registerURL := fmt.Sprintf("http://%s/register-peer", peerAddr)
    
    req, err := http.NewRequest("POST", registerURL, nil)
    if err != nil {
        return fmt.Errorf("failed to create register request: %v", err)
    }
    
    req.Header.Set("X-Peer-Address", myAddr)
    req.Header.Set("X-Node-ID", bn.nodeID)
    
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("failed to register with peer: %v", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("peer returned status: %d", resp.StatusCode)
    }
    
    return nil
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

    log.Printf("[NETWORK] Received new block from peer")

    // Validate through consensus
    err = bn.consensusValidator.HandleBlockchainSync(bn.nodeID, body)
    if err != nil {
        log.Printf("[NETWORK] Consensus validation failed: %v", err)
        http.Error(w, fmt.Sprintf("Consensus validation failed: %v", err), http.StatusBadRequest)
        return
    }

    // Process the block
    err = bn.blockPropagator.HandleIncomingBlock(body)
    if err != nil {
        log.Printf("[NETWORK] Block processing failed: %v", err)
        http.Error(w, fmt.Sprintf("Error processing block: %v", err), http.StatusBadRequest)
        return
    }

    log.Printf("[NETWORK] Successfully processed new block")
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

func (bn *BlockchainNetwork) handlePeerHealth(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    health := map[string]interface{}{
        "status": "healthy",
        "timestamp": time.Now(),
        "node_id": bn.nodeID,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(health)
}

func (bn *BlockchainNetwork) handleNetworkInfo(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    info := map[string]interface{}{
        "node_id": bn.nodeID,
        "peer_count": len(bn.peerManager.GetPeers()),
        "blockchain_height": len(bn.blockchain.Blocks),
        "is_connected": bn.isConnected,
        "bootstrap_address": bn.bootstrapAddress,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(info)
}

func (bn *BlockchainNetwork) PropagateNewBlock(block *blockchain.Block) error {
    blockchainJSON, err := bn.blockchain.ToJSON()
    if err != nil {
        return fmt.Errorf("error converting blockchain to JSON: %v", err)
    }

    err = bn.consensusValidator.HandleBlockchainSync(bn.nodeID, blockchainJSON)
    if err != nil {
        return fmt.Errorf("consensus validation failed: %v", err)
    }

    log.Printf("[NETWORK] Propagating new block to peers")
    return bn.blockPropagator.PropagateNewBlock(block)
}

func (bn *BlockchainNetwork) monitorNetwork() {
    ticker := time.NewTicker(30 * time.Second)
    for range ticker.C {
        peers := bn.peerManager.GetPeers()
        bn.isConnected = len(peers) > 0

        log.Printf("[NETWORK] Network Status:")
        log.Printf("- Connected Peers: %d", len(peers))
        log.Printf("- Blockchain Height: %d", len(bn.blockchain.Blocks))
        log.Printf("- Node ID: %s", bn.nodeID)
        log.Printf("- Connection Status: %v", bn.isConnected)

        // Only check peer health if we're not connected
        if !bn.isConnected {
            for _, peer := range peers {
                if err := bn.checkPeerHealth(peer); err != nil {
                    log.Printf("[NETWORK] Peer %s health check failed: %v", peer, err)
                    bn.peerManager.RemovePeer(peer)
                }
            }
        }
    }
}

func (bn *BlockchainNetwork) checkPeerHealth(peerAddr string) error {
    url := fmt.Sprintf("http://%s/peer-health", peerAddr)
    client := &http.Client{Timeout: 5 * time.Second}
    
    resp, err := client.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("peer returned status: %d", resp.StatusCode)
    }
    
    return nil
}

func (bn *BlockchainNetwork) GetPeerManager() *PeerManager {
    return bn.peerManager
}

func (bn *BlockchainNetwork) GetConsensusValidator() *Consensus.NetworkBlockchainValidator {
    return bn.consensusValidator
}

// Helper function to get outbound IP
func getOutboundIP() net.IP {
    conn, err := net.Dial("udp", "8.8.8.8:80")
    if err != nil {
        return net.ParseIP("127.0.0.1")
    }
    defer conn.Close()

    localAddr := conn.LocalAddr().(*net.UDPAddr)
    return localAddr.IP
}
