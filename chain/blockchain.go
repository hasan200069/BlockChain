package chain

import (
	"awesomeProject/util"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Blockchain struct {
	Blocks []*Block
}

const Difficulty = 4

func (b *Block) MineBlock(difficulty int) {
	prefix := bytes.Repeat([]byte("0"), difficulty)
	for {
		b.Hash = util.GenerateHash(b)
		if bytes.HasPrefix([]byte(b.Hash), prefix) {
			break
		}
		b.Nonce++
	}
}

func (bc *Blockchain) getPreviousHash() string {
	if len(bc.Blocks) > 0 {
		return bc.Blocks[len(bc.Blocks)-1].getPreviousHash()
	}
	return ""
}

func (bc *Blockchain) CreateNewBlock(transactions []Transaction) *Block {
	previousHash := bc.getPreviousHash()
	newBlock := &Block{
		PreviousHash: previousHash,
		Timestamp:    time.Now(),
		Data:         Data{Transactions: transactions},
	}
	newBlock.MineBlock(Difficulty)
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

func (bc *Blockchain) ToJSON() ([]byte, error) {
	data, err := json.MarshalIndent(bc, "", "  ") // Marshals the chain with indentation for readability
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (bc *Blockchain) SaveToFile(filename string) error {
	data, err := bc.ToJSON()
	if err != nil {
		return err
	}
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (bc *Blockchain) LoadFromJSON(data []byte) error {
	err := json.Unmarshal(data, bc)
	if err != nil {
		return err
	}
	return nil
}

func (bc *Blockchain) ToNetworkBytes() ([]byte, error) {
	data, err := json.Marshal(bc)
	if err != nil {
		return nil, fmt.Errorf("failed to convert blockchain to network bytes: %v", err)
	}
	return data, nil
}

func (bc *Blockchain) FromNetworkBytes(data []byte) error {
	var newBlockchain Blockchain
	err := json.Unmarshal(data, &newBlockchain)
	if err != nil {
		return fmt.Errorf("failed to convert network bytes to blockchain: %v", err)
	}
	*bc = newBlockchain
	return nil
}
