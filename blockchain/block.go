package blockchain


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