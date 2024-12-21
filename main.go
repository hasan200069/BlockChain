package main

import (
    "awesomeProject/Consensus"
    "awesomeProject/Peer2Peer"
    "awesomeProject/blockchain"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "net"
    "net/http"
    "os"
    "os/signal"
    "time"
)

// Helper function to get outbound IP
func getOutboundIP() string {
    conn, err := net.Dial("udp", "8.8.8.8:80")
    if err != nil {
        return "127.0.0.1"
    }
    defer conn.Close()

    localAddr := conn.LocalAddr().(*net.UDPAddr)
    return localAddr.IP.String()
}

func main() {
    port := flag.Int("port", 8000, "Port to listen on")
    bootstrapNode := flag.String("bootstrap", "", "Bootstrap node address")
    nodeID := flag.String("id", "", "Node ID")
    flag.Parse()
    
    if *nodeID == "" {
        *nodeID = fmt.Sprintf("node-%d", *port)
    }

    log.Printf("[STARTUP] Initializing node %s on port %d", *nodeID, *port)

    // Initialize blockchain
    bc := &blockchain.Blockchain{}
    
    // Initialize transaction handler
    txHandler := blockchain.NewTransactionHandler(bc, 1000) // Pool size of 1000
    txHandler.StartCleanup(5*time.Minute, 1*time.Hour)
    
    // Initialize consensus validator explicitly
    consensusValidator := Consensus.NewNetworkBlockchainValidator(4)
    
    // Initialize network with consensus
    network := Peer2Peer.NewBlockchainNetwork(*port, bc, *nodeID)
    
    // Start the P2P server
    go func() {
        listenAddr := fmt.Sprintf(":%d", *port)
        if err := network.StartServer(listenAddr); err != nil {
            log.Fatalf("Failed to start P2P server: %v", err)
        }
    }()

    // Connect to bootstrap node if specified
    // Connect to bootstrap node if specified
	if *bootstrapNode != "" {
	    req := fmt.Sprintf("http://%s/register-peer", *bootstrapNode)
	    log.Printf("[NETWORK] Connecting to bootstrap node at: %s", req)
	    
	    client := &http.Client{Timeout: 10 * time.Second}
	    registerReq, err := http.NewRequest("POST", req, nil)
	    if err != nil {
		log.Printf("[ERROR] Error creating register request: %v", err)
	    } else {
		// Use actual IP address instead of localhost
		myAddr := fmt.Sprintf("%s:%d", getOutboundIP(), *port)
		registerReq.Header.Set("X-Peer-Address", myAddr)
		
		resp, err := client.Do(registerReq)
		if err != nil {
		    log.Printf("[ERROR] Error registering with bootstrap node: %v", err)
		} else {
		    defer resp.Body.Close()
		    var result map[string]interface{}
		    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		        log.Printf("[ERROR] Error reading response: %v", err)
		    } else {
		        log.Printf("[SUCCESS] Connected to network. Network size: %v", result["network_size"])
		        
		        // Add bootstrap node to our peer list
		        network.GetPeerManager().AddPeer(*bootstrapNode)
		    }
		}
	    }

	    // Get initial blockchain state
	    blockchainJSON, err := bc.ToJSON()
	    if err != nil {
		log.Printf("[ERROR] Error converting blockchain to JSON: %v", err)
	    } else {
		if err := consensusValidator.HandleBlockchainSync(*nodeID, blockchainJSON); err != nil {
		    log.Printf("[ERROR] Initial consensus validation failed: %v", err)
		}
	    }
	}
    
    // Create new blocks periodically
    go func() {
        for {
            time.Sleep(30 * time.Second)
            
            transactions := txHandler.GetTransactionsForBlock(10) // Max 10 tx per block
            
            if len(transactions) > 0 {
                // Create new block with selected transactions
                newBlock := bc.CreateNewBlock(transactions)
                if newBlock == nil {
                    log.Printf("[ERROR] Failed to create new block")
                    continue
                }
                
                // Convert block to JSON for consensus validation
                blockchainJSON, err := bc.ToJSON()
                if err != nil {
                    log.Printf("[ERROR] Error converting blockchain to JSON: %v", err)
                    continue
                }

                // Validate through consensus before propagating
                err = consensusValidator.HandleBlockchainSync(*nodeID, blockchainJSON)
                if err != nil {
                    log.Printf("[ERROR] Consensus validation failed: %v", err)
                    continue
                }
                
                // If consensus is achieved, propagate the block
                err = network.PropagateNewBlock(newBlock)
                if err != nil {
                    log.Printf("[ERROR] Error propagating new block: %v", err)
                    continue
                }
                
                log.Printf("[SUCCESS] Created and propagated new block with %d transactions. Chain length: %d",
                    len(transactions), len(bc.Blocks))
            }
        }
    }()

    // Add HTTP endpoints for transaction handling
	http.HandleFunc("/submit-transaction", func(w http.ResponseWriter, r *http.Request) {
	    if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	    }

	    // Create a temporary struct to hold the input data
	    var input struct {
		Sender   string `json:"sender"`
		Receiver string `json:"receiver"`
		Amount   int    `json:"amount"`
	    }

	    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, fmt.Sprintf("Invalid transaction data: %v", err), http.StatusBadRequest)
		return
	    }

	    // Create a new transaction using the CreateTransaction function
	    tx, err := blockchain.CreateTransaction(input.Sender, input.Receiver, input.Amount)
	    if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create transaction: %v", err), http.StatusBadRequest)
		return
	    }

	    status, err := txHandler.SubmitTransaction(tx)
	    if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	    }

	    w.Header().Set("Content-Type", "application/json")
	    json.NewEncoder(w).Encode(status)
	})

    http.HandleFunc("/transaction-status", func(w http.ResponseWriter, r *http.Request) {
        txID := r.URL.Query().Get("id")
        if txID == "" {
            http.Error(w, "Transaction ID required", http.StatusBadRequest)
            return
        }

        status, err := txHandler.GetTransactionStatus(txID)
        if err != nil {
            http.Error(w, err.Error(), http.StatusNotFound)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(status)
    })

    // Add node status endpoint
    http.HandleFunc("/node-status", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }

        status := map[string]interface{}{
            "node_id": *nodeID,
            "port": *port,
            "chain_length": len(bc.Blocks),
            "pending_transactions": txHandler.GetPoolSize(),
            "is_bootstrap": *bootstrapNode == "",
            "connected_peers": len(network.GetPeerManager().GetPeers()),
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(status)
    })

    log.Printf("Node %s running on port %d", *nodeID, *port)
    if *bootstrapNode != "" {
        log.Printf("Connected to bootstrap node: %s", *bootstrapNode)
    } else {
        log.Println("Running as bootstrap node")
    }

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt)
    <-sigChan
    log.Println("\nShutting down node...")
}
