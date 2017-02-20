package rpc

import "encoding/binary"

type RpcOptions struct {
	Host    string
	Port    string
	User    string
	Pass    string
	Testnet bool
}

type RpcRequest struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Id string `json:"id"`
}

type MempoolRPCResult struct {
	RpcRequest
	Result []string `json:"result"`
}

type UnspentTx struct {
	Txid          string  `json:"txid"`
	Vout          int     `json:"vout"`
	Address       string  `json:"address,omitempty"`
	ScriptPubKey  string  `json:"scriptPubKey,omitempty"`
	Amount        float32 `json:"amount,omitempty"`
	Confirmations int64   `json:"confirmations,omitempty"`
	Spendable     bool    `json:"spendable,omitempty"`
	Solvable      bool    `json:"solvable,omitempty"`
	Priority      float32 `json:"priority,omitempty"`
}

type UnspentTxs []UnspentTx

type UnspentTxRPCResult struct {
	RpcRequest
	Result UnspentTxs `json:"result"`
}

func (slice UnspentTxs) Len() int {
	return len(slice)
}

func (slice UnspentTxs) Less(i, j int) bool {
	return slice[i].Priority > slice[j].Priority
}

func (slice UnspentTxs) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

type RawTxIn struct {
	Txid      string `json:"txid"`
	Vout      uint32 `json:"vout"`
	ScriptSig string `json:"scriptSig"`
	Sequence  uint32 `json:"sequence"`
}

type RawTxOut struct {
	Value        float32 `json:"value"`
	ScriptPubKey string  `json:"scriptPubKey"`
}

type RawTxn struct {
	Version  uint32     `json:"version"`
	Locktime uint32     `json:"locktime"`
	Vin      []RawTxIn  `json:"vin"`
	Vout     []RawTxOut `json:"vout"`
}

type TxnBuf struct {
	b   []byte
	pos uint64
}

func (txnBuf *TxnBuf) shift_byte() byte {
	val := txnBuf.b[txnBuf.pos : txnBuf.pos+1]
	txnBuf.pos += 1
	return val[0]
}

func (txnBuf *TxnBuf) shift_16bit() uint16 {
	val := binary.LittleEndian.Uint16(txnBuf.b[txnBuf.pos : txnBuf.pos+2])
	txnBuf.pos += 2
	return val
}

func (txnBuf *TxnBuf) shift_32bit() uint32 {
	val := binary.LittleEndian.Uint32(txnBuf.b[txnBuf.pos : txnBuf.pos+4])
	txnBuf.pos += 4
	return val
}

func (txnBuf *TxnBuf) shift_64bit() uint64 {
	val := binary.LittleEndian.Uint64(txnBuf.b[txnBuf.pos : txnBuf.pos+8])
	txnBuf.pos += 8
	return val
}

func (txnBuf *TxnBuf) shift_bits(length uint64) []byte {
	val := txnBuf.b[txnBuf.pos : txnBuf.pos+length]
	txnBuf.pos += length
	return val
}

func (txnBuf *TxnBuf) shift_varint() uint64 {
	intType := txnBuf.shift_byte()
	if intType == 0xFF {
		return txnBuf.shift_64bit()
	} else if intType == 0xFE {
		return uint64(txnBuf.shift_32bit())
	} else if intType == 0xFD {
		return uint64(txnBuf.shift_16bit())
	}

	return uint64(intType)
}

type SignedTx struct {
	Hex      string `json:"hex"`
	Complete bool   `json:"complete"`
	Errors   []struct {
		Txid      string `json:"txid"`
		Vout      uint32 `json:"vout"`
		ScriptSig string `json:"scriptSig"`
		Sequence  uint32 `json:"sequence"`
		Error     string `json:"error"`
	} `json:"errors"`
}

type SignedTxRPCResult struct {
	RpcRequest
	Result SignedTx `json:"result`
}
