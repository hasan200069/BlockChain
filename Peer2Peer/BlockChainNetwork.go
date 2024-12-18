package Peer2Peer

import (
	"awesomeProject/Consensus"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

type BlockchainNetwork struct {
	peerManager *PeerManager
	blockchain  []byte
	mu          sync.Mutex
	consensusValidator *Consensus.NetworkBlockchainValidator  //field added with consensus file
        nodeID      string
}

func NewBlockchainNetwork(localPort int, initialBlockchain []byte) *BlockchainNetwork {
	return &BlockchainNetwork{
		peerManager: NewPeerManager(localPort),
		blockchain:  initialBlockchain,
		consensusValidator: Consensus.NewNetworkBlockchainValidator(4),  // using difficulty 4
        nodeID: fmt.Sprintf("node-%d", localPort),
	}
}

func (bn *BlockchainNetwork) FloodBlockchain() {
	peers := bn.peerManager.GetPeers()
	var wg sync.WaitGroup

	for _, peerAddress := range peers {
		wg.Add(1)
		go func(peer string) {
			defer wg.Done()
			bn.sendBlockchainToPeer(peer)
		}(peerAddress)
	}

	wg.Wait()
}

func (bn *BlockchainNetwork) sendBlockchainToPeer(peerAddress string) {
	url := fmt.Sprintf("http://%s/receive-blockchain", peerAddress)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bn.blockchain))
	if err != nil {
		log.Printf("Error creating request to %s: %v", peerAddress, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending blockchain to %s: %v", peerAddress, err)
		bn.peerManager.RemovePeer(peerAddress)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to send blockchain to %s: Status %d", peerAddress, resp.StatusCode)
	}
}

func (bn *BlockchainNetwork) StartServer(listenAddress string) error {
	http.HandleFunc("/receive-blockchain", bn.receiveBlockchainHandler)
	http.HandleFunc("/register-peer", bn.registerPeerHandler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bn.peerManager.DiscoverPeers(ctx)

	log.Printf("Starting blockchain network server on %s", listenAddress)
	return http.ListenAndServe(listenAddress, nil)
}

func (bn *BlockchainNetwork) receiveBlockchainHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Error reading request body", http.StatusBadRequest)
        return
    }

    // Add consensus validation
    err = bn.consensusValidator.HandleBlockchainSync(bn.nodeID, body)
    if err != nil {
        http.Error(w, fmt.Sprintf("Consensus validation failed: %v", err), http.StatusBadRequest)
        return
    }

    bn.mu.Lock()
    bn.blockchain = body
    bn.mu.Unlock()

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

func (bn *BlockchainNetwork) validateBlockchain(blockchainData []byte) bool {
	var blockchain interface{}
	err := json.Unmarshal(blockchainData, &blockchain)
	return err == nil
}
