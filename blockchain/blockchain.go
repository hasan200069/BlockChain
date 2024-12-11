package blockchain

import "awesomeProject/util"

type Blockchain struct {
	Blocks []*Block
}

func (bc *Blockchain) CreateNewBlock(data string) *Block {
	previousBlock := bc.getLastBlock()
	previousHash := ""
	if previousBlock != nil {
		previousHash = util.GenerateHash(previousBlock)
	}
	newBlock := &Block{
		PreviousHash:  previousHash,
		PreviousBlock: previousBlock,
		Data:          data,
	}
	bc.Blocks = append(bc.Blocks, newBlock)
	return newBlock
}

func (bc *Blockchain) getLastBlock() *Block {
	if len(bc.Blocks) == 0 {
		return nil
	}
	return bc.Blocks[len(bc.Blocks)-1]
}
