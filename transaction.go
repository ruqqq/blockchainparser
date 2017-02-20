package blockchainparser

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
)

type Script []byte

func (script Script) String() string {
	return hex.EncodeToString(script)
}

type TxInput struct {
	Hash          Hash256
	Index         uint32
	Script        Script
	Sequence      uint32
	ScriptWitness [][]byte
}

func (in TxInput) Binary() []byte {
	bin := make([]byte, 0)
	bin = append(bin, in.Hash...)

	index := make([]byte, 4)
	binary.LittleEndian.PutUint32(index, uint32(in.Index))
	bin = append(bin, index...)

	scriptLength := Varint(uint64(len(in.Script)))
	bin = append(bin, scriptLength...)

	bin = append(bin, in.Script...)

	sequence := make([]byte, 4)
	binary.LittleEndian.PutUint32(sequence, uint32(in.Sequence))
	bin = append(bin, sequence...)

	return bin
}

func (in TxInput) ScriptWitnessBinary() []byte {
	bin := make([]byte, 0)
	witnessCount := Varint(uint64(len(in.ScriptWitness)))
	bin = append(bin, witnessCount...)
	for _, data := range in.ScriptWitness {
		length := Varint(uint64(len(data)))
		bin = append(bin, length...)
		bin = append(bin, data...)
	}

	return bin
}

type TxOutput struct {
	Value  int64
	Script Script
}

func (out TxOutput) BTC() float64 {
	return float64(out.Value) / 1000000000.0
}

func (out TxOutput) Binary() []byte {
	bin := make([]byte, 0)

	value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value, uint64(out.Value))
	bin = append(bin, value...)

	scriptLength := Varint(uint64(len(out.Script)))
	bin = append(bin, scriptLength...)

	bin = append(bin, out.Script...)

	return bin
}

type Transaction struct {
	hash     Hash256 // not actually in blockchain data; for caching
	Version  int32
	Locktime uint32
	Vin      []TxInput
	Vout     []TxOutput
	StartPos uint64 // not actually in blockchain data
}

func Varint(n uint64) []byte {
	if n > 4294967295 {
		val := make([]byte, 8)
		binary.BigEndian.PutUint64(val, n)
		return append([]byte{0xFF}, val...)
	} else if n > 65535 {
		val := make([]byte, 4)
		binary.BigEndian.PutUint32(val, uint32(n))
		return append([]byte{0xFE}, val...)
	} else if n > 255 {
		val := make([]byte, 2)
		binary.BigEndian.PutUint16(val, uint16(n))
		return append([]byte{0xFD}, val...)
	} else {
		return []byte{byte(n)}
	}
}

func (tx Transaction) HasWitness() bool {
	for _, in := range tx.Vin {
		if len(in.ScriptWitness) > 0 {
			return true
		}
	}

	return false
}

func (tx Transaction) Txid() Hash256 {
	if tx.hash != nil {
		return tx.hash
	}

	bin := make([]byte, 0)

	//hasScriptWitness := tx.HasWitness()

	version := make([]byte, 4)
	binary.LittleEndian.PutUint32(version, uint32(tx.Version))
	bin = append(bin, version...)

	//var flags byte
	//if hasScriptWitness {
	//	bin = append(bin, 0)
	//	flags |= 1
	//	bin = append(bin, flags)
	//}

	vinLength := Varint(uint64(len(tx.Vin)))
	bin = append(bin, vinLength...)
	for _, in := range tx.Vin {
		bin = append(bin, in.Binary()...)
	}

	voutLength := Varint(uint64(len(tx.Vout)))
	bin = append(bin, voutLength...)
	for _, out := range tx.Vout {
		bin = append(bin, out.Binary()...)
	}

	//if hasScriptWitness {
	//	for _, in := range tx.Vin {
	//		bin = append(bin, in.ScriptWitnessBinary()...)
	//	}
	//}

	locktime := make([]byte, 4)
	binary.LittleEndian.PutUint32(locktime, tx.Locktime)
	bin = append(bin, locktime...)

	tx.hash = DoubleSha256(bin)
	return tx.hash
}

func NewTxFromFile(blockchainDataDir string, magicHeader MagicId, num uint32, pos uint32, txPos uint32) (*Transaction, error) {
	block := &Block{}

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

	// Read and validate Magic ID
	block.MagicId = MagicId(blockFile.ReadUint32())
	if block.MagicId != magicHeader {
		return nil, errors.New("Invalid block header: Can't find Magic ID")
	}

	// Read header fields
	err = ParseBlockHeaderFromFile(blockFile, block)
	if err != nil {
		return nil, err
	}

	// Seek to the transaction pos
	_, err = blockFile.Seek(int64(txPos), 1)
	if err != nil {
		return nil, err
	}

	// Parse transaction
	tx, err := ParseBlockTransactionFromFile(blockFile)
	if err != nil {
		return nil, err
	}

	return tx, nil
}
