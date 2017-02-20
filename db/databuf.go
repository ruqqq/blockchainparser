package db

import "encoding/binary"

// TODO: Improve this to use Buffer interface
type DataBuf struct {
	b   []byte
	pos uint64
}

func NewDataBuf(b []byte) *DataBuf {
	return &DataBuf{b, 0}
}

func (buf *DataBuf) Reset() {
	buf.pos = 0
}

func (buf *DataBuf) Seek(pos uint64) {
	buf.pos = pos
}

func (buf *DataBuf) ShiftByte() byte {
	val := buf.b[buf.pos : buf.pos+1]
	buf.pos += 1
	return val[0]
}

func (buf *DataBuf) ShiftBytes(length uint64) []byte {
	val := buf.b[buf.pos : buf.pos+length]
	buf.pos += length
	return val
}

func (buf *DataBuf) Shift16bit() uint16 {
	val := binary.LittleEndian.Uint16(buf.b[buf.pos : buf.pos+2])
	buf.pos += 2
	return val
}

func (buf *DataBuf) ShiftU64bit() uint64 {
	val := binary.LittleEndian.Uint64(buf.b[buf.pos : buf.pos+8])
	buf.pos += 8
	return val
}

func (buf *DataBuf) Shift64bit() int64 {
	val := binary.LittleEndian.Uint64(buf.b[buf.pos : buf.pos+8])
	buf.pos += 8
	return int64(val)
}

func (buf *DataBuf) ShiftU32bit() uint32 {
	val := binary.LittleEndian.Uint32(buf.b[buf.pos : buf.pos+4])
	buf.pos += 4
	return val
}

func (buf *DataBuf) Shift32bit() int32 {
	val := binary.LittleEndian.Uint32(buf.b[buf.pos : buf.pos+4])
	buf.pos += 4
	return int32(val)
}

func (buf *DataBuf) ShiftVarint() uint64 {
	var n uint64
	for true {
		b := buf.b[buf.pos : buf.pos+1][0]
		buf.pos += 1
		n = (n << uint64(7)) | uint64(b&uint8(0x7F))
		if b&uint8(0x80) > 0 {
			n++
		} else {
			return n
		}
	}

	return n
}
