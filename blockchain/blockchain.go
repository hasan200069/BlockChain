package blockchain

import (
	"awesomeProject/util"
	"fmt"
	"time"
)

type Blockchain struct {
	Blocks []*Block
}

func (bc *Blockchain) getPreviousHash() string {
	var previousHash string
	if len(bc.Blocks) > 0 {
		previousHash = bc.Blocks[len(bc.Blocks)-1].getPreviousHash()
	}

	return previousHash

}

func (bc *Blockchain) CreateNewBlock(transactions []Transaction, nonce int) *Block {

	var previousHash = bc.getPreviousHash()
	newBlock := &Block{
		PreviousHash: previousHash,
		Timestamp:    time.Now(),
		Nonce:        nonce,
		Data:         Data{Transactions: transactions},
	}
	newBlock.Hash = util.GenerateHash(newBlock)
	bc.Blocks = append(bc.Blocks, newBlock)
	return newBlock
}

func (bc *Blockchain) GetLastBlock() *Block {
	if len(bc.Blocks) == 0 {
		return nil
	}
	return bc.Blocks[len(bc.Blocks)-1]
}

func (bc *Blockchain) PrintChain() {
	if len(bc.Blocks) == 0 {
		fmt.Println("Blockchain is empty.")
		return
	}

	fmt.Println("Blockchain:")
	for i, block := range bc.Blocks {
		fmt.Printf("\nBlock %d:\n", i+1)
		fmt.Printf("Previous Hash: %s\n", block.PreviousHash)
		fmt.Printf("Timestamp: %s\n", block.Timestamp)
		fmt.Printf("Nonce: %d\n", block.Nonce)
		fmt.Printf("Hash: %s\n", block.Hash)
		fmt.Println("Transactions:")
		for _, transaction := range block.Data.Transactions {
			fmt.Printf("  - %s\n", transaction)
		}
	}
}
