package blockchain

type Block struct {
	TransactionHash string
	PreviousHash    string
	PreviousBlock   *Block
	Data            string
}
