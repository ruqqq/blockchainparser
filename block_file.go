package blockchainparser

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

const (
	//! Magic numbers to identify start of block
	BLOCK_MAGIC_ID_BITCOIN MagicId = 0xd9b4bef9
	BLOCK_MAGIC_ID_TESTNET MagicId = 0x0709110b
)

type BlockFile struct {
	file    *os.File
	FileNum uint32
}

func NewBlockFile(blockchainDataDir string, fileNum uint32) (*BlockFile, error) {
	filepath := fmt.Sprintf(blockchainDataDir+"/blocks/blk%05d.dat", fileNum)
	//fmt.Printf("Opening file %s...\n", filepath)

	file, err := os.OpenFile(filepath, os.O_RDONLY, 0666)
	if err != nil {
		return nil, err
	}

	return &BlockFile{file: file, FileNum: fileNum}, nil
}

func (blockFile *BlockFile) Close() {
	blockFile.file.Close()
}

func (blockFile *BlockFile) Seek(offset int64, whence int) (int64, error) {
	return blockFile.file.Seek(offset, whence)
}

func (blockFile *BlockFile) Size() (int64, error) {
	fileInfo, err := blockFile.file.Stat()
	if err != nil {
		return 0, err
	}
	return fileInfo.Size(), err
}

func (blockFile *BlockFile) Peek(length int) ([]byte, error) {
	pos, err := blockFile.file.Seek(0, 1)
	if err != nil {
		return nil, err
	}
	val := make([]byte, length)
	blockFile.file.Read(val)
	_, err = blockFile.file.Seek(pos, 0)
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (blockFile *BlockFile) ReadByte() byte {
	val := make([]byte, 1)
	blockFile.file.Read(val)
	return val[0]
}

func (blockFile *BlockFile) ReadBytes(length uint64) []byte {
	val := make([]byte, length)
	blockFile.file.Read(val)
	return val
}

func (blockFile *BlockFile) ReadUint16() uint16 {
	val := make([]byte, 2)
	blockFile.file.Read(val)
	return binary.LittleEndian.Uint16(val)
}

func (blockFile *BlockFile) ReadInt32() int32 {
	raw := make([]byte, 4)
	blockFile.file.Read(raw)
	var val int32
	binary.Read(bytes.NewReader(raw), binary.LittleEndian, &val)
	return val
}

func (blockFile *BlockFile) ReadUint32() uint32 {
	val := make([]byte, 4)
	blockFile.file.Read(val)
	return binary.LittleEndian.Uint32(val)
}

func (blockFile *BlockFile) ReadInt64() int64 {
	raw := make([]byte, 8)
	blockFile.file.Read(raw)
	var val int64
	binary.Read(bytes.NewReader(raw), binary.LittleEndian, &val)
	return val
}

func (blockFile *BlockFile) ReadUint64() uint64 {
	val := make([]byte, 8)
	blockFile.file.Read(val)
	return binary.LittleEndian.Uint64(val)
}

//func (blockFile *BlockFile) ReadVarint() uint64 {
//	chSize := blockFile.ReadByte()
//	if chSize < 253 {
//		return uint64(chSize)
//	} else if chSize == 253 {
//		return uint64(blockFile.ReadUint16())
//	} else if chSize == 254 {
//		return uint64(blockFile.ReadUint32())
//	} else {
//		return blockFile.ReadUint64()
//	}
//}

func (blockFile *BlockFile) ReadVarint() uint64 {
	intType := blockFile.ReadByte()
	if intType == 0xFF {
		return blockFile.ReadUint64()
	} else if intType == 0xFE {
		return uint64(blockFile.ReadUint32())
	} else if intType == 0xFD {
		return uint64(blockFile.ReadUint16())
	}

	return uint64(intType)
}
