package blockchain

import "time"

type Data struct {
	Transactions []Transaction
}

type Block struct {
	PreviousHash string
	Timestamp    time.Time
	Nonce        int
	Data         Data
	Hash         string
}

func (b *Block) getPreviousHash() string {
	return b.Hash
}
