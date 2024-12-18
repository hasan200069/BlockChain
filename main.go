/*package main

import (
	"awesomeProject/IPFS"
	"awesomeProject/blockchain"
	"database/sql"
	"fmt"
	"log"
)

func main() {
	bc := &blockchain.Blockchain{}
	transactionID := "tx12345"
	sender := "Alice"
	receiver := "Bob"
	algorithm := "SHA-256"
	resultHash := "abc123def456"
	signatures := []string{"sig1", "sig2", "sig3"}
	actualOutput := "Successful transaction"
	amount := 100
	tx := blockchain.CreateTransaction(transactionID, sender, receiver, algorithm, resultHash, signatures, actualOutput, amount)
	transactions := []blockchain.Transaction{*tx}
	bc.CreateNewBlock(transactions)
	bc.CreateNewBlock(transactions)
	bc.PrintChain()
	db, err := IPFS.CreateConnection()
	if err != nil {
		log.Fatalf("Error creating connection: %v", err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatalf("Error closing db: %v", err)
		}
	}(db)

	newJSON, _ := bc.ToJSON()
	fmt.Println(string(newJSON))
}*/
package main

import (
    //"awesomeProject/IPFS"
    //"encoding/json"
    "awesomeProject/Consensus"
    "awesomeProject/Peer2Peer"
    "awesomeProject/blockchain"
    "flag"
    "fmt"
    "log"
    "os"
    "os/signal"
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
    
    // Convert blockchain to JSON for network transmission
    blockchainJSON, err := bc.ToJSON()
    if err != nil {
        log.Fatalf("Error converting blockchain to JSON: %v", err)
    }

    // Initialize P2P network
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
        // Use proper method to register peer
        req := fmt.Sprintf("http://%s/register-peer", *bootstrapNode)
        fmt.Printf("Connecting to bootstrap node at: %s\n", req)
        network.FloodBlockchain()
    }
    
    //exmaple transaction
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
    
    // Add block and broadcast
    bc.CreateNewBlock([]blockchain.Transaction{*tx})
    network.FloodBlockchain()

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
