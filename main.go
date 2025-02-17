package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"log"
	"math"
	"math/big"
	"strconv"
	"time"
)

type Block struct {
	Timestamp     int64
	Data          []byte
	PrevBlockHash []byte
	Hash          []byte
	Nonce         int
}

func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	err := encoder.Encode(b)
	if err != nil {
		log.Panic(err)
	}

	return result.Bytes()
}

func DeserializeBlock(b []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(b))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}

	return &block
}

func NewBlock(data string, prevBlockHash []byte) *Block {
	block := &Block{
		Timestamp:     time.Now().Unix(),
		Data:          []byte(data),
		PrevBlockHash: prevBlockHash,
		Hash:          []byte{},
	}
	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()

	block.Hash = hash
	block.Nonce = nonce

	return block
}

func NewGenesisBlock() *Block {
	return NewBlock("Genesis Block", []byte{})
}

type Blockchain struct {
	tip []byte
	db  *bolt.DB
}

func (bc *Blockchain) AddBlock(data string) {
	var lastHash []byte
	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte((blocksBucket)))
		lastHash = b.Get([]byte("l"))

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	newBlock := NewBlock(data, lastHash)

	err = bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		err := b.Put(newBlock.Hash, newBlock.Serialize())
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte("l"), newBlock.Hash)
		if err != nil {
			log.Panic(err)
		}
		bc.tip = newBlock.Hash

		return nil
	})

	if err != nil {
		log.Panic(err)
	}
}

func (b *Blockchain) Iterator() *BlockchainIterator {
	return &BlockchainIterator{currentHash: b.tip, db: b.db}
}

const dbFile = "blockchain.db"
const blocksBucket = "dupa"

func NewBlockchain() *Blockchain {
	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			genesis := NewGenesisBlock()

			b, err := tx.CreateBucket([]byte(blocksBucket))
			if err != nil {
				log.Panic(err)
			}

			err = b.Put(genesis.Hash, genesis.Serialize())
			if err != nil {
				log.Panic(err)
			}

			err = b.Put([]byte("l"), genesis.Hash)
			if err != nil {
				log.Panic(err)
			}

			tip = genesis.Hash
		} else {
			tip = b.Get([]byte("l"))
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return &Blockchain{tip: tip, db: db}
}

type BlockchainIterator struct {
	currentHash []byte
	db          *bolt.DB
}

func (bi *BlockchainIterator) Next() *Block {
	var block *Block
	err := bi.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(bi.currentHash)
		block = DeserializeBlock(encodedBlock)

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	bi.currentHash = block.PrevBlockHash

	return block
}

const targetBits = 24

func IntToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

type ProofOfWork struct {
	block  *Block
	target *big.Int
}

func (p *ProofOfWork) prepareData(nonce int) []byte {
	return bytes.Join(
		[][]byte{
			p.block.PrevBlockHash,
			p.block.Data,
			IntToHex(p.block.Timestamp),
			IntToHex(int64(targetBits)),
			IntToHex(int64(nonce)),
		},
		[]byte{},
	)
}

const maxNonce = math.MaxInt64

func (p *ProofOfWork) Run() (int, []byte) {
	var hashInt big.Int
	var hash [32]byte
	nonce := 0

	fmt.Printf("Mining the block containing \"%s\"", p.block.Data)
	for nonce < maxNonce {
		data := p.prepareData(nonce)
		hash = sha256.Sum256(data)
		fmt.Printf("\r%x", hash)
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(p.target) == -1 {
			// if smaller than target finish mining
			break
		} else {
			nonce++
		}
	}

	fmt.Printf("\n\n")
	return nonce, hash[:]
}

func (p *ProofOfWork) Validate() bool {
	var hashInt big.Int
	data := p.prepareData(p.block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])

	return hashInt.Cmp(p.target) == -1
}

func NewProofOfWork(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))
	pow := &ProofOfWork{
		block:  b,
		target: target,
	}

	return pow
}

func main() {
	bc := NewBlockchain()
	defer bc.db.Close()

	bc.AddBlock("dupa")
	bc.AddBlock("cycki")
	iter := bc.Iterator()
	for {
		block := iter.Next()
		fmt.Printf("Prev. hash: %x\n", block.PrevBlockHash)
		fmt.Printf("Data: %s\n", block.Data)
		fmt.Printf("Hash: %x\n", block.Hash)

		pow := NewProofOfWork(block)
		fmt.Printf("Pow: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	fmt.Printf("END")
}
