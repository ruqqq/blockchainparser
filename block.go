package blockchainparser

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

type Hash256 []byte

func (hash Hash256) String() string {
	return hex.EncodeToString(ReverseHex(hash))
}

type MagicId uint32

func (magicId MagicId) String() string {
	return fmt.Sprintf("%x", uint32(magicId))
}

type BlockHeader struct {
	hash             Hash256 // not actuallyin blockchain data; for caching
	Version          int32
	HashPrev         Hash256
	HashMerkle       Hash256
	Timestamp        time.Time
	TargetDifficulty uint32 // bits
	Nonce            uint32
}

type Block struct {
	BlockHeader      // actual pos below Length field
	MagicId          MagicId
	Length           uint32
	TransactionCount uint64 // txn_count
	Transactions     []Transaction
	StartPos         uint64 // not actually in blockchain data
}

func (blockHeader *BlockHeader) Hash() Hash256 {
	if blockHeader.hash != nil {
		return blockHeader.hash
	}

	bin := make([]byte, 0)

	version := make([]byte, 4)
	binary.LittleEndian.PutUint32(version, uint32(blockHeader.Version))
	bin = append(bin, version...)

	bin = append(bin, blockHeader.HashPrev...)
	bin = append(bin, blockHeader.HashMerkle...)

	timestamp := make([]byte, 4)
	binary.LittleEndian.PutUint32(timestamp, uint32(blockHeader.Timestamp.Unix()))
	bin = append(bin, timestamp...)

	targetDifficulty := make([]byte, 4)
	binary.LittleEndian.PutUint32(targetDifficulty, blockHeader.TargetDifficulty)
	bin = append(bin, targetDifficulty...)

	nonce := make([]byte, 4)
	binary.LittleEndian.PutUint32(nonce, blockHeader.Nonce)
	bin = append(bin, nonce...)

	blockHeader.hash = DoubleSha256(bin)
	return blockHeader.hash
}

// Parse the header fields except the MagicId
// TODO: Currently won't return any error
func ParseBlockHeaderFromFile(blockFile *BlockFile, block *Block) error {
	block.Length = blockFile.ReadUint32()
	block.Version = blockFile.ReadInt32()
	block.HashPrev = blockFile.ReadBytes(32)
	block.HashMerkle = blockFile.ReadBytes(32)
	block.Timestamp = time.Unix(int64(blockFile.ReadUint32()), 0)
	block.TargetDifficulty = blockFile.ReadUint32() // TODO: Parse this as mantissa?
	block.Nonce = blockFile.ReadUint32()

	return nil
}

func ParseBlockTransactionsFromFile(blockFile *BlockFile, block *Block) error {
	// Read transaction count to know how many transactions to parse
	block.TransactionCount = blockFile.ReadVarint()
	//fmt.Printf("Total txns: %d\n", block.TransactionCount)
	for t := uint64(0); t < block.TransactionCount; t++ {
		tx, err := ParseBlockTransactionFromFile(blockFile)
		if err != nil {
			return err
		}
		block.Transactions = append(block.Transactions, *tx)
	}

	return nil
}

func ParseBlockTransactionFromFile(blockFile *BlockFile) (*Transaction, error) {
	curPos, err := blockFile.Seek(0, 1)
	if err != nil {
		return nil, err
	}

	allowWitness := true // TODO: Port code - !(s.GetVersion() & SERIALIZE_TRANSACTION_NO_WITNESS);

	tx := &Transaction{}
	tx.StartPos = uint64(curPos)
	tx.Version = blockFile.ReadInt32()

	// Check for extended transaction serialization format
	p, _ := blockFile.Peek(1)
	var txInputLength uint64
	var txFlag byte
	if p[0] == 0 {
		// We are dealing with extended transaction
		blockFile.ReadByte()          // dummy
		txFlag = blockFile.ReadByte() // flags
		txInputLength = blockFile.ReadVarint()
	} else {
		txInputLength = blockFile.ReadVarint()
	}

	for i := uint64(0); i < txInputLength; i++ {
		input := TxInput{}
		input.Hash = blockFile.ReadBytes(32)
		input.Index = blockFile.ReadUint32() // TODO: Not sure if correctly read
		scriptLength := blockFile.ReadVarint()
		input.Script = blockFile.ReadBytes(scriptLength)
		input.Sequence = blockFile.ReadUint32()
		tx.Vin = append(tx.Vin, input)
	}

	txOutputLength := blockFile.ReadVarint()
	for i := uint64(0); i < txOutputLength; i++ {
		output := TxOutput{}
		output.Value = int64(blockFile.ReadUint64())
		scriptLength := blockFile.ReadVarint()
		output.Script = blockFile.ReadBytes(scriptLength)
		tx.Vout = append(tx.Vout, output)
	}

	if (txFlag&1) == 1 && allowWitness {
		txFlag ^= 1 // Not sure what this is for
		for i := uint64(0); i < txInputLength; i++ {
			witnessCount := blockFile.ReadVarint()
			tx.Vin[i].ScriptWitness = make([][]byte, witnessCount)
			for j := uint64(0); j < witnessCount; j++ {
				length := blockFile.ReadVarint()
				tx.Vin[i].ScriptWitness[j] = blockFile.ReadBytes(length)
			}
		}
	}

	tx.Locktime = blockFile.ReadUint32()

	return tx, nil
}

func ParseBlockFromFile(blockFile *BlockFile, magicHeader MagicId) (*Block, error) {
	block := &Block{}

	curPos, err := blockFile.Seek(0, 1)
	if err != nil {
		return nil, err
	}

	// Read and validate Magic ID
	block.MagicId = MagicId(blockFile.ReadUint32())
	if block.MagicId != magicHeader {
		blockFile.Seek(curPos, 0) // Seek back to original pos before we encounter the error
		return nil, errors.New("Invalid block header: Can't find Magic ID")
	}

	// Read header fields
	err = ParseBlockHeaderFromFile(blockFile, block)
	if err != nil {
		blockFile.Seek(curPos, 0) // Seek back to original pos before we encounter the error
		return nil, err
	}

	// Parse transactions
	err = ParseBlockTransactionsFromFile(blockFile, block)
	if err != nil {
		blockFile.Seek(curPos, 0) // Seek back to original pos before we encounter the error
		return nil, err
	}

	return block, nil
}

func NewBlockFromFile(blockchainDataDir string, magicHeader MagicId, num uint32, pos uint32) (*Block, error) {
	// Open file for reading
	blockFile, err := NewBlockFile(blockchainDataDir, num)
	if err != nil {
		return nil, err
	}
	defer blockFile.Close()

	// Seek to pos - 8 to start reading from block header
	fmt.Printf("Seeking to block at %d...\n", pos)
	_, err = blockFile.Seek(int64(pos-8), 0)
	if err != nil {
		return nil, err
	}

	return ParseBlockFromFile(blockFile, magicHeader)
}
