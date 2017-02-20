package rpc

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

func Check(options *RpcOptions) (bool, error) {
	_, err := Cmd("getinfo", options)
	if err != nil {
		return false, err
	}

	return true, nil
}

func ListUnspent(options *RpcOptions) (UnspentTxs, error) {
	body, err := Cmd("listunspent", options)
	if err != nil {
		return nil, err
	}

	var result UnspentTxRPCResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}
	if result.Error.Message != "" {
		return nil, errors.New(result.Error.Message)
	}

	return result.Result, nil
}

func GetRawMempool(options *RpcOptions) ([]string, error) {
	body, err := Cmd("getrawmempool", options)
	if err != nil {
		return nil, err
	}

	var result MempoolRPCResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}
	if result.Error.Message != "" {
		return nil, errors.New(result.Error.Message)
	}

	return result.Result, nil
}

func CreateRawTransaction(inputs UnspentTxs, outputs map[string]float32, options *RpcOptions) (RawTxn, error) {
	body, err := Cmd("createrawtransaction", options, inputs, outputs)
	if err != nil {
		return RawTxn{}, err
	}

	result := make(map[string]interface{})
	err = json.Unmarshal(body, &result)
	if err != nil {
		return RawTxn{}, err
	}
	if errMap, ok := result["error"]; ok && errMap != nil {
		return RawTxn{}, errors.New(errMap.(map[string]interface{})["message"].(string))
	}

	if _, ok := result["result"]; !ok {
		return RawTxn{}, errors.New("Can't find result field")
	}

	// hex to []byte
	rawTxnHex, err := hex.DecodeString(result["result"].(string))
	if err != nil {
		return RawTxn{}, err
	}

	// unpack []byte to RawTxn
	rawTxn, err := unpack_txn(rawTxnHex)
	if err != nil {
		return RawTxn{}, err
	}

	return rawTxn, nil
}

func SignRawTransaction(txn RawTxn, prevTxns UnspentTxs, options *RpcOptions) (SignedTx, error) {
	rawTxn, err := pack_txn(txn)
	if err != nil {
		return SignedTx{}, err
	}

	fmt.Printf("rawTxn: %s\n", hex.EncodeToString(rawTxn))
	var body []byte

	if prevTxns != nil && len(prevTxns) != 0 {
		prevTxnsJson, err := json.Marshal(prevTxns)
		if err != nil {
			return SignedTx{}, err
		}
		fmt.Printf("prevTxnsJson: %s\n", prevTxnsJson)
		body, err = Cmd("signrawtransaction", options, hex.EncodeToString(rawTxn), prevTxns)
		if err != nil {
			return SignedTx{}, err
		}
	} else {
		body, err = Cmd("signrawtransaction", options, hex.EncodeToString(rawTxn))
		if err != nil {
			return SignedTx{}, err
		}
	}

	var result SignedTxRPCResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		return SignedTx{}, err
	}
	if len(result.Result.Errors) > 0 {
		return SignedTx{}, errors.New(result.Result.Errors[0].Error)
	} else if result.Error.Code > 0 {
		return SignedTx{}, errors.New("Code " + strconv.Itoa(result.Error.Code) + ": " + result.Error.Message)
	} else if !result.Result.Complete {
		return SignedTx{}, errors.New("Unknown error occured: " + string(body))
	}

	return result.Result, nil
}

func SendRawTransaction(rawTxn string, options *RpcOptions) (string, error) {
	sendTxid, err := CmdAsSingleResult("sendrawtransaction", options, rawTxn)
	if err != nil {
		return "", err
	}
	if len(sendTxid.(string)) != 64 {
		return "", errors.New("Can't send transaction")
	}

	return sendTxid.(string), nil
}

func CmdAsSingleResult(command string, options *RpcOptions, args ...interface{}) (interface{}, error) {
	body, err := Cmd(command, options, args...)
	if err != nil {
		return "", err
	}

	result := make(map[string]interface{})
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}
	if errMap, ok := result["error"]; ok && errMap != nil {
		return "", errors.New(errMap.(map[string]interface{})["message"].(string))
	}

	if res, ok := result["result"]; ok {
		return res, nil
	} else {
		return "", errors.New("Can't find result field")
	}
}

func Cmd(command string, options *RpcOptions, args ...interface{}) ([]byte, error) {
	// For cmd line
	//args_compiled := args
	//if Testnet {
	//	args_compiled = append([]string{"-Testnet"}, args_compiled...)
	//}
	//args_compiled = append(args_compiled, command)
	//body, err = exec.Command(BITCOIN_PATH, args_compiled...).Output()
	//if err != nil {
	//	return nil, err
	//}

	now := strconv.FormatInt(time.Now().Unix(), 10)
	random := strconv.FormatInt(rand.Int63n(999999)+100000, 10)

	requestBody := struct {
		Id     string        `json:"id"`
		Method string        `json:"method"`
		Params []interface{} `json:"params"`
	}{now + "-" + random, command, args}

	client := &http.Client{}
	port := options.Port
	if options.Testnet {
		port = "1" + port
	}
	jsonBody, _ := json.Marshal(requestBody)
	//fmt.Printf("jsonBody: %s\n", jsonBody)
	req, err := http.NewRequest("POST", "http://"+options.Host+":"+port, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(options.User, options.Pass)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

//func bitcoin_cmd_as_map(command string) (map[string]interface{}, error) {
//	use_cmd := false // TODO: Get from args
//
//	body, err := bitcoin_cmd(command)
//	if err != nil {
//		return nil, err
//	}
//
//	result := make(map[string]interface{})
//	err = json.Unmarshal(body, &result)
//	if err != nil {
//		return nil, err
//	}
//
//	if !use_cmd {
//		if res, ok := result["result"]; ok {
//			return res.(map[string]interface{}), nil
//		} else {
//			return nil, errors.New("Can't find result field")
//		}
//	} else {
//		return result, nil
//	}
//}
//
//func bitcoin_cmd_as_map_array(command string) ([]map[string]interface{}, error) {
//	use_cmd := false // TODO: Get from args
//
//	body, err := bitcoin_cmd(command)
//	if err != nil {
//		return nil, err
//	}
//
//	result := make(map[string]interface{}, 0)
//	err = json.Unmarshal(body, &result)
//	if err != nil {
//		return nil, err
//	}
//
//	if !use_cmd {
//		if res, ok := result["result"]; ok {
//			return res.([]map[string]interface{}), nil
//		} else {
//			return nil, errors.New("Can't find result field")
//		}
//	} else {
//		return result, nil
//	}
//}

func unpack_txn(b []byte) (RawTxn, error) {
	var txn RawTxn
	txnBuf := TxnBuf{b, 0}
	txn.Version = txnBuf.shift_32bit()

	for inputs := txnBuf.shift_varint(); inputs > 0; inputs -= 1 {
		var input RawTxIn
		txid := txnBuf.shift_bits(32)
		// reverse to get txid
		for i := len(txid)/2 - 1; i >= 0; i-- {
			opp := len(txid) - 1 - i
			txid[i], txid[opp] = txid[opp], txid[i]
		}
		input.Txid = hex.EncodeToString(txid)
		input.Vout = txnBuf.shift_32bit()
		scriptSig := txnBuf.shift_bits(txnBuf.shift_varint())
		input.ScriptSig = hex.EncodeToString(scriptSig)
		input.Sequence = txnBuf.shift_32bit()

		txn.Vin = append(txn.Vin, input)
	}

	for outputs := txnBuf.shift_varint(); outputs > 0; outputs -= 1 {
		var output RawTxOut
		output.Value = float32(txnBuf.shift_64bit()) / 100000000.0
		output.ScriptPubKey = hex.EncodeToString(txnBuf.shift_bits(txnBuf.shift_varint()))

		txn.Vout = append(txn.Vout, output)
	}

	txn.Locktime = txnBuf.shift_32bit()

	return txn, nil
}

func pack_txn(txn RawTxn) ([]byte, error) {
	b := make([]byte, 0)
	version := make([]byte, 4)
	binary.LittleEndian.PutUint32(version, txn.Version)
	b = append(b, version...)

	b = append(b, pack_varint(uint64(len(txn.Vin)))...)
	for _, input := range txn.Vin {
		txid, err := hex.DecodeString(input.Txid)
		if err != nil {
			return nil, err
		}
		// reverse to store txid
		for i := len(txid)/2 - 1; i >= 0; i-- {
			opp := len(txid) - 1 - i
			txid[i], txid[opp] = txid[opp], txid[i]
		}
		b = append(b, txid...)

		vout := make([]byte, 4)
		binary.LittleEndian.PutUint32(vout, input.Vout)
		b = append(b, vout...)

		scriptSig, err := hex.DecodeString(input.ScriptSig)
		if err != nil {
			return nil, err
		}
		b = append(b, pack_varint(uint64(len(scriptSig)))...)
		b = append(b, scriptSig...)

		sequence := make([]byte, 4)
		binary.LittleEndian.PutUint32(sequence, input.Sequence)
		b = append(b, sequence...)
	}

	b = append(b, pack_varint(uint64(len(txn.Vout)))...)
	for _, output := range txn.Vout {
		value := make([]byte, 8)
		binary.LittleEndian.PutUint64(value, uint64(math.Ceil(float64(output.Value)*100000000)))
		b = append(b, value...)

		scriptPubKey, err := hex.DecodeString(output.ScriptPubKey)
		if err != nil {
			return nil, err
		}
		b = append(b, pack_varint(uint64(len(scriptPubKey)))...)
		b = append(b, scriptPubKey...)
	}

	locktime := make([]byte, 4)
	binary.LittleEndian.PutUint32(locktime, txn.Locktime)
	b = append(b, locktime...)

	return b, nil
}

func pack_varint(val uint64) []byte {
	b := make([]byte, 0)
	if val > 0xFFFFFFFF {
		l := make([]byte, 8)
		binary.LittleEndian.PutUint64(l, val)
		b = append(b, 0xFF)
		b = append(b, l...)
	} else if val > 0xFFFF {
		l := make([]byte, 4)
		binary.LittleEndian.PutUint32(l, uint32(val))
		b = append(b, 0xFE)
		b = append(b, l...)
	} else if val > 0xFC {
		l := make([]byte, 2)
		binary.LittleEndian.PutUint16(l, uint16(val))
		b = append(b, 0xFD)
		b = append(b, l...)
	} else {
		b = append(b, uint8(val))
	}
	return b
}
