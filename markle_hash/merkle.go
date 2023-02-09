package merkle

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"time"
)

type Block struct {
	Index     int
	Timestamp string
	Content   string
	Hash      string
	PrevHash  string
}

func GenerateBlock(oldBlock Block, sendData string) Block {
	var newBlock Block
	t := time.Now()

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.Content = sendData
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Hash = CalculateHash(newBlock)

	return newBlock
}

func IsBlockValid(oldBlock Block, newBlock Block) bool {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}
	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}
	if CalculateHash(newBlock) != newBlock.Hash {
		return false
	}
	return true
}

func CalculateHash(block Block) string {
	rec := strconv.Itoa(block.Index) + block.Timestamp + block.Content + block.PrevHash
	h := sha256.New()
	h.Write([]byte(rec))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}
