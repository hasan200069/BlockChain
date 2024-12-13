package blockchain

import "awesomeProject/util"

type Blockchain struct {
	Blocks []*Block
}

func (bc *Blockchain) CreateNewBlock(transactions []Transaction, nonce int) *Block {
	var previousHash string
	if len(bc.Blocks) > 0 {
		previousHash = util.GenerateHash(bc.Blocks[len(bc.Blocks)-1])
	}
	newBlock := &Block{
		PreviousHash: previousHash,
		Timestamp:    time.Now(),
		Nonce:        nonce,
		Data:         Data{Transactions: transactions},
		Hash:         util.GenerateHash(newBlock),
	}
	bc.Blocks = append(bc.Blocks, newBlock)
	return newBlock
}

func (bc *Blockchain) GetLastBlock() *Block {
	if len(bc.Blocks) == 0 {
		return nil
	}
	return bc.Blocks[len(bc.Blocks)-1]
}