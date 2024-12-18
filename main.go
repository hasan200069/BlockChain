package main

import (
    "awesomeProject/Consensus"
    "awesomeProject/Peer2Peer"
    "awesomeProject/blockchain"
    "flag"
    "fmt"
    "log"
    "os"
    "os/signal"
    "time"
)

func main() {
    port := flag.Int("port", 8000, "Port to listen on")
    bootstrapNode := flag.String("bootstrap", "", "Bootstrap node address")
    nodeID := flag.String("id", "", "Node ID")
    flag.Parse()
    
    // If no node ID provided, generate one
    if *nodeID == "" {
        *nodeID = fmt.Sprintf("node-%d", *port)
    }

    // Initialize blockchain
    bc := &blockchain.Blockchain{}
    
    // Initialize consensus validator
    consensusValidator := Consensus.NewNetworkBlockchainValidator(4) // difficulty of 4
    
    // Convert blockchain to JSON for network transmission
    blockchainJSON, err := bc.ToJSON()
    if err != nil {
        log.Fatalf("Error converting blockchain to JSON: %v", err)
    }

    // Initialize P2P network with consensus
    network := Peer2Peer.NewBlockchainNetwork(*port, blockchainJSON)
    
    // Start the P2P server
    go func() {
        listenAddr := fmt.Sprintf(":%d", *port)
        if err := network.StartServer(listenAddr); err != nil {
            log.Fatalf("Failed to start P2P server: %v", err)
        }
    }()

    // If this is not the bootstrap node, connect to the network
    if *bootstrapNode != "" {
        req := fmt.Sprintf("http://%s/register-peer", *bootstrapNode)
        fmt.Printf("Connecting to bootstrap node at: %s\n", req)
        
        // Validate and sync blockchain
        err = consensusValidator.HandleBlockchainSync(*nodeID, blockchainJSON)
        if err != nil {
            log.Printf("Initial consensus validation failed: %v", err)
        }
        
        network.FloodBlockchain()
    }
    
    // Periodically create new transactions and test consensus
    go func() {
        for {
            time.Sleep(30 * time.Second) // Create new block every 30 seconds
            
            tx := blockchain.CreateTransaction(
                fmt.Sprintf("tx-%s-%d", *nodeID, len(bc.Blocks)),
                *nodeID,
                "receiver",
                "SHA-256",
                "result-hash",
                []string{"sig1"},
                "Successful transaction",
                100,
            )
            
            // Add block
            bc.CreateNewBlock([]blockchain.Transaction{*tx})
            
            // Convert updated blockchain to JSON
            newBlockchainJSON, err := bc.ToJSON()
            if err != nil {
                log.Printf("Error converting updated blockchain to JSON: %v", err)
                continue
            }
            
            // Validate the new blockchain
            err = consensusValidator.HandleBlockchainSync(*nodeID, newBlockchainJSON)
            if err != nil {
                log.Printf("Consensus validation failed for new block: %v", err)
                continue
            }
            
            // Broadcast if consensus is achieved
            network.FloodBlockchain()
            fmt.Printf("\n[NODE] Created and broadcasted new block. Chain length: %d\n", len(bc.Blocks))
        }
    }()

    fmt.Printf("Node %s running on port %d\n", *nodeID, *port)
    if *bootstrapNode != "" {
        fmt.Printf("Connected to bootstrap node: %s\n", *bootstrapNode)
    } else {
        fmt.Println("Running as bootstrap node")
    }

    // Keep the program running
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt)
    <-sigChan
    fmt.Println("\nShutting down node...")
}
