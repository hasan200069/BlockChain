package main

import (
    "awesomeProject/Consensus"
    "awesomeProject/Peer2Peer"
    "awesomeProject/blockchain"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "time"
)

func main() {
    port := flag.Int("port", 8000, "Port to listen on")
    bootstrapNode := flag.String("bootstrap", "", "Bootstrap node address")
    nodeID := flag.String("id", "", "Node ID")
    flag.Parse()
    
    if *nodeID == "" {
        *nodeID = fmt.Sprintf("node-%d", *port)
    }

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
    if *bootstrapNode != "" {
        req := fmt.Sprintf("http://%s/register-peer", *bootstrapNode)
        fmt.Printf("Connecting to bootstrap node at: %s\n", req)
        
        // Get initial blockchain from network
        blockchainJSON, err := bc.ToJSON()
        if err != nil {
            log.Printf("Error converting blockchain to JSON: %v", err)
        } else {
            // Validate blockchain through consensus before accepting
            err = consensusValidator.HandleBlockchainSync(*nodeID, blockchainJSON)
            if err != nil {
                log.Printf("Initial consensus validation failed: %v", err)
            }
        }
        
        client := &http.Client{Timeout: 10 * time.Second}
        registerReq, err := http.NewRequest("POST", req, nil)
        if err != nil {
            log.Printf("Error creating register request: %v", err)
        } else {
            registerReq.Header.Set("X-Peer-Address", fmt.Sprintf("localhost:%d", *port))
            _, err = client.Do(registerReq)
            if err != nil {
                log.Printf("Error registering with bootstrap node: %v", err)
            }
        }
    }
    
    // Create new blocks periodically
    go func() {
        for {
            time.Sleep(30 * time.Second)
            
            // Get transactions from pool instead of creating new ones
            transactions := txHandler.GetTransactionsForBlock(10) // Max 10 tx per block
            
            if len(transactions) > 0 {
                // Create new block with selected transactions
                newBlock := bc.CreateNewBlock(transactions)
                
                // Convert block to JSON for consensus validation
                blockchainJSON, err := bc.ToJSON()
                if err != nil {
                    log.Printf("Error converting blockchain to JSON: %v", err)
                    continue
                }

                // Validate through consensus before propagating
                err = consensusValidator.HandleBlockchainSync(*nodeID, blockchainJSON)
                if err != nil {
                    log.Printf("Consensus validation failed: %v", err)
                    continue
                }
                
                // If consensus is achieved, propagate the block
                err = network.PropagateNewBlock(newBlock)
                if err != nil {
                    log.Printf("Error propagating new block: %v", err)
                    continue
                }
                
                fmt.Printf("\n[NODE] Created and propagated new block with %d transactions. Chain length: %d\n",
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

        var tx blockchain.Transaction
        if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
            http.Error(w, fmt.Sprintf("Invalid transaction data: %v", err), http.StatusBadRequest)
            return
        }

        status, err := txHandler.SubmitTransaction(&tx)
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

    fmt.Printf("Node %s running on port %d\n", *nodeID, *port)
    if *bootstrapNode != "" {
        fmt.Printf("Connected to bootstrap node: %s\n", *bootstrapNode)
    } else {
        fmt.Println("Running as bootstrap node")
    }

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt)
    <-sigChan
    fmt.Println("\nShutting down node...")
}
