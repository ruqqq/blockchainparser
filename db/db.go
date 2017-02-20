package db

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/ruqqq/blockchainparser"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"time"
)

const (
	//! Unused.
	BLOCK_VALID_UNKNOWN = 0

	//! Parsed, version ok, hash satisfies claimed PoW, 1 <= vtx count <= max, timestamp not in future
	BLOCK_VALID_HEADER = 1

	//! All parent headers found, difficulty matches, timestamp >= median previous, checkpoint. Implies all parents
	//! are also at least TREE.
	BLOCK_VALID_TREE = 2

	/**
	 * Only first tx is coinbase, 2 <= coinbase input script length <= 100, transactions valid, no duplicate txids,
	 * sigops, size, merkle root. Implies all parents are at least TREE but not necessarily TRANSACTIONS. When all
	 * parent blocks also have TRANSACTIONS, CBlockIndex::nChainTx will be set.
	 */
	BLOCK_VALID_TRANSACTIONS = 3

	//! Outputs do not overspend inputs, no double spends, coinbase output ok, no immature coinbase spends, BIP30.
	//! Implies all parents are also at least CHAIN.
	BLOCK_VALID_CHAIN = 4

	//! Scripts & signatures ok. Implies all parents are also at least SCRIPTS.
	BLOCK_VALID_SCRIPTS = 5

	//! All validity bits.
	BLOCK_VALID_MASK = BLOCK_VALID_HEADER | BLOCK_VALID_TREE | BLOCK_VALID_TRANSACTIONS |
		BLOCK_VALID_CHAIN | BLOCK_VALID_SCRIPTS

	BLOCK_HAVE_DATA = 8  //!< full block available in blk*.dat
	BLOCK_HAVE_UNDO = 16 //!< undo data available in rev*.dat
	BLOCK_HAVE_MASK = BLOCK_HAVE_DATA | BLOCK_HAVE_UNDO

	BLOCK_FAILED_VALID = 32 //!< stage after last reached validness failed
	BLOCK_FAILED_CHILD = 64 //!< descends from failed block
	BLOCK_FAILED_MASK  = BLOCK_FAILED_VALID | BLOCK_FAILED_CHILD

	BLOCK_OPT_WITNESS = 128 //!< block data in blk*.data was received with a witness-enforcing client
)

type BlockIndexRecord struct {
	Version        int32
	Height         int32
	Status         uint32
	NTx            uint32
	NFile          int32
	NDataPos       uint32
	NUndoPos       uint32
	HashPrev       blockchainparser.Hash256
	HashMerkleRoot blockchainparser.Hash256
	NTime          time.Time
	NBits          uint32
	NNonce         uint32
}

type FileInfoRecord struct {
	NumOfBlocks uint32
	Size        uint32
	UndoSize    uint32
	HeightFirst uint32
	HeightLast  uint32
	TimeFirst   time.Time
	TimeLast    time.Time
}

type TxIndexRecord struct {
	NFile     int32
	NDataPos  uint32
	NTxOffset uint32
}

type IndexDb struct {
	*leveldb.DB
}

type ChainstateDb struct {
	*leveldb.DB
}

func OpenIndexDb(blockchainDataDir string) (*IndexDb, error) {
	db, err := leveldb.OpenFile(blockchainDataDir+"/blocks/index/", &opt.Options{
		ReadOnly: true,
	})
	if err != nil {
		return nil, err
	}

	return &IndexDb{db}, nil
}

func OpenChainstateDb(blockchainDataDir string) (*ChainstateDb, error) {
	db, err := leveldb.OpenFile(blockchainDataDir+"/chainstate/", &opt.Options{
		ReadOnly: true,
	})
	if err != nil {
		return nil, err
	}

	return &ChainstateDb{db}, nil
}

func GetBlockIndexRecordByBigEndianHex(indexDb *IndexDb, blockHash string) (*BlockIndexRecord, error) {
	blockHashInBytes, err := hex.DecodeString(blockHash)
	if err != nil {
		return nil, err
	}
	// Reverse hex to get the LittleEndian order
	blockHashInBytes = blockchainparser.ReverseHex(blockHashInBytes)

	return GetBlockIndexRecord(indexDb, blockHashInBytes)
}

func GetBlockIndexRecordByHex(indexDb *IndexDb, blockHash string) (*BlockIndexRecord, error) {
	blockHashInBytes, err := hex.DecodeString(blockHash)
	if err != nil {
		return nil, err
	}
	return GetBlockIndexRecord(indexDb, blockHashInBytes)
}

func GetBlockIndexRecord(indexDb *IndexDb, blockHash []byte) (*BlockIndexRecord, error) {
	fmt.Printf("blockHash: %v, %d bytes\n", blockHash, len(blockHash))

	// Get data
	data, err := indexDb.Get(append([]byte("b"), blockHash...), nil)
	if err != nil {
		return nil, err
	}
	fmt.Printf("rawBlockIndexRecord: %v\n", data)

	// Parse the raw bytes
	blockIndexRecord := NewBlockIndexRecordFromBytes(data)

	return blockIndexRecord, nil
}

func GetFileInfoRecord(indexDb *IndexDb, number uint32) (*FileInfoRecord, error) {
	fileNumber := make([]byte, 4)
	// the key is stored in LittleEndian in LevelDB
	binary.LittleEndian.PutUint32(fileNumber, number)
	fmt.Printf("fileNumber: %v\n", fileNumber)

	// Get data
	data, err := indexDb.Get(append([]byte("f"), fileNumber...), nil)
	if err != nil {
		return nil, err
	}

	// Parse the raw bytes
	fileInfoRecord := NewFileInfoRecordFromBytes(data)

	return fileInfoRecord, nil
}

func GetTxIndexRecordByBigEndianHex(indexDb *IndexDb, txHash string) (*TxIndexRecord, error) {
	txHashInBytes, err := hex.DecodeString(txHash)
	if err != nil {
		return nil, err
	}
	// reverse hex to get the LittleEndian order
	txHashInBytes = blockchainparser.ReverseHex(txHashInBytes)

	return GetTxIndexRecord(indexDb, txHashInBytes)
}

func GetTxIndexRecordByHex(indexDb *IndexDb, txHash string) (*TxIndexRecord, error) {
	txHashInBytes, err := hex.DecodeString(txHash)
	if err != nil {
		return nil, err
	}

	return GetTxIndexRecord(indexDb, txHashInBytes)
}

func GetTxIndexRecord(indexDb *IndexDb, txHash []byte) (*TxIndexRecord, error) {
	fmt.Printf("tx: %v, %d bytes\n", txHash, len(txHash))

	// Get data
	data, err := indexDb.Get(append([]byte("t"), txHash...), nil)
	if err != nil {
		return nil, err
	}

	// Parse the raw bytes
	txRecord := NewTxIndexRecordFromBytes(data)

	return txRecord, nil
}

func GetLastBlockFileNumberUsed(indexDb *IndexDb) (uint32, error) {
	data, err := indexDb.Get([]byte("l"), nil)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint32(data), nil
}

func GetReindexing(indexDb *IndexDb) (bool, error) {
	return indexDb.Has([]byte("R"), nil)
}

func GetFlag(indexDb *IndexDb, name []byte) (bool, error) {
	command := append([]byte("F"), byte(len(name)))
	command = append(command, name...)
	data, err := indexDb.Get(command, nil)
	if err != nil {
		return false, err
	}

	return data[0] == []byte("1")[0], nil
}

// TODO: Implement when needed: https://github.com/bitcoin/bitcoin/blob/d4a42334d447cad48fb3996cad0fd5c945b75571/src/coins.h#L19
//func GetCoinRecord(chainstateDb *ChainstateDb, txHash []byte) (*interface{}, error) {
//	fmt.Printf("tx: %v, %d bytes\n", txHash, len(txHash))
//
//	// Get data
//	data, err := chainstateDb.Get(append([]byte("c"), txHash...), nil)
//	if err != nil {
//		return nil, err
//	}
//
//	// Parse the raw bytes
//	coinRecord := //
//
//	return coinRecord, nil
//}

func GetBestBlock(chainstateDb *ChainstateDb) ([]byte, error) {
	return chainstateDb.Get([]byte("B"), nil)
}

func NewBlockIndexRecordFromBytes(b []byte) *BlockIndexRecord {
	dataBuf := NewDataBuf(b)
	fmt.Printf("rawData: %v\n", b)
	dataHex := hex.EncodeToString(b)
	fmt.Printf("rawData: %v\n", dataHex)

	// Discard first varint
	// FIXME: Not exactly sure why need to, but if we don't do this we won't get correct values
	dataBuf.ShiftVarint()

	record := &BlockIndexRecord{}
	record.Height = int32(dataBuf.ShiftVarint())
	record.Status = uint32(dataBuf.ShiftVarint())
	record.NTx = uint32(dataBuf.ShiftVarint())
	if record.Status&(BLOCK_HAVE_DATA|BLOCK_HAVE_UNDO) > 0 {
		record.NFile = int32(dataBuf.ShiftVarint())
	}
	if record.Status&BLOCK_HAVE_DATA > 0 {
		record.NDataPos = uint32(dataBuf.ShiftVarint())
	}
	if record.Status&BLOCK_HAVE_UNDO > 0 {
		record.NUndoPos = uint32(dataBuf.ShiftVarint())
	}

	record.Version = dataBuf.Shift32bit()
	record.HashPrev = dataBuf.ShiftBytes(32)
	record.HashMerkleRoot = dataBuf.ShiftBytes(32)
	record.NTime = time.Unix(int64(dataBuf.ShiftU32bit()), 0)
	record.NBits = dataBuf.ShiftU32bit()
	record.NNonce = dataBuf.ShiftU32bit()

	return record
}

func NewFileInfoRecordFromBytes(b []byte) *FileInfoRecord {
	dataBuf := NewDataBuf(b)
	fmt.Printf("rawData: %v\n", b)
	dataHex := hex.EncodeToString(b)
	fmt.Printf("rawData: %v\n", dataHex)

	return &FileInfoRecord{
		NumOfBlocks: uint32(dataBuf.ShiftVarint()),
		Size:        uint32(dataBuf.ShiftVarint()),
		UndoSize:    uint32(dataBuf.ShiftVarint()),
		HeightFirst: uint32(dataBuf.ShiftVarint()),
		HeightLast:  uint32(dataBuf.ShiftVarint()),
		TimeFirst:   time.Unix(int64(dataBuf.ShiftVarint()), 0),
		TimeLast:    time.Unix(int64(dataBuf.ShiftVarint()), 0),
	}
}

func NewTxIndexRecordFromBytes(b []byte) *TxIndexRecord {
	dataBuf := NewDataBuf(b)
	fmt.Printf("rawData: %v\n", b)
	dataHex := hex.EncodeToString(b)
	fmt.Printf("rawData: %v\n", dataHex)

	record := &TxIndexRecord{}
	record.NFile = int32(dataBuf.ShiftVarint())
	record.NDataPos = uint32(dataBuf.ShiftVarint())
	record.NTxOffset = uint32(dataBuf.ShiftVarint())

	return record
}
