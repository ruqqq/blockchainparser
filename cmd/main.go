package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"github.com/ruqqq/blockchainparser"
	"github.com/ruqqq/blockchainparser/db"
	"log"
	"os"
	"strconv"
)

func main() {
	var help bool
	var testnet bool
	var datadir string
	flag.BoolVar(&testnet, "testnet", testnet, "Use testnet")
	flag.StringVar(&datadir, "datadir", blockchainparser.BitcoinDir(), "Bitcoin data path")
	flag.BoolVar(&help, "help", help, "Show help")
	flag.Parse()

	fmt.Printf("testnet: %v\n", testnet)
	fmt.Printf("datapath: %s\n", datadir)
	args := flag.Args()

	magicId := blockchainparser.BLOCK_MAGIC_ID_BITCOIN
	if testnet {
		datadir += "/testnet3"
		magicId = blockchainparser.BLOCK_MAGIC_ID_TESTNET
	}

	showHelp := func() {
		fmt.Fprint(os.Stderr, "blockchainparser\n(c)2017 Faruq Rasid\n\n"+
			"Commands:\n"+
			"  GetBlock <hash>\n"+
			"  GetBlockIndexRecord <hash>\n"+
			"  GetBlockFromFile <fileNum> <blockStartPos>\n"+
			"  GetTx <hash>\n"+
			"  GetTxIndexRecord <hash>\n"+
			"  GetTxFromFile <fileNum> <blockStartPos> <txPos>\n"+
			"  GetFileInfoRecord <fileNum>\n"+
			"  GetLastBlockFileNumberUsed\n"+
			"  GetFlag <name>\n"+
			"  GetReindexing\n"+
			"\n"+
			"Options:\n")
		flag.PrintDefaults()
	}

	if len(args) == 0 || help {
		showHelp()
		return
	}

	// open index db as READONLY
	indexDb, err := db.OpenIndexDb(datadir)
	if err != nil {
		log.Fatal(err)
	}
	defer indexDb.Close()

	if len(args) == 2 && args[0] == "GetBlock" {
		failIfReindexing(indexDb)
		result, err := db.GetBlockIndexRecordByBigEndianHex(indexDb, args[1])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%+v\n", result)

		block, err := blockchainparser.NewBlockFromFile(datadir, magicId, uint32(result.NFile), result.NDataPos)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("%+v\n", block)
		fmt.Printf("First Txid: %s\n", hex.EncodeToString(blockchainparser.ReverseHex(block.Transactions[0].Txid())))
	} else if len(args) == 2 && args[0] == "GetBlockIndexRecord" {
		failIfReindexing(indexDb)
		result, err := db.GetBlockIndexRecordByBigEndianHex(indexDb, args[1])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%+v\n", result)
	} else if len(args) == 3 && args[0] == "GetBlockFromFile" {
		num, err := strconv.ParseUint(args[1], 10, 32)
		if err != nil {
			log.Fatal(err)
		}
		pos, err := strconv.ParseUint(args[2], 10, 32)
		if err != nil {
			log.Fatal(err)
		}

		block, err := blockchainparser.NewBlockFromFile(datadir, magicId, uint32(num), uint32(pos))
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("%+v\n", block)
	} else if len(args) == 2 && args[0] == "GetTx" {
		failIfReindexing(indexDb)
		f, _ := db.GetFlag(indexDb, []byte("txindex"))
		if !f {
			log.Fatal(errors.New("txindex is not enabled for your bitcoind"))
		}
		result, err := db.GetTxIndexRecordByBigEndianHex(indexDb, args[1])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%+v\n", result)

		tx, err := blockchainparser.NewTxFromFile(datadir, magicId, uint32(result.NFile), result.NDataPos, result.NTxOffset)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("%+v\n", tx)
	} else if len(args) == 2 && args[0] == "GetTxIndexRecord" {
		failIfReindexing(indexDb)
		f, _ := db.GetFlag(indexDb, []byte("txindex"))
		if !f {
			log.Fatal(errors.New("txindex is not enabled for your bitcoind"))
		}
		result, err := db.GetTxIndexRecordByBigEndianHex(indexDb, args[1])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%+v\n", result)
	} else if len(args) == 4 && args[0] == "GetTxFromFile" {
		num, err := strconv.ParseUint(args[1], 10, 32)
		if err != nil {
			log.Fatal(err)
		}
		pos, err := strconv.ParseUint(args[2], 10, 32)
		if err != nil {
			log.Fatal(err)
		}
		txPos, err := strconv.ParseUint(args[3], 10, 32)
		if err != nil {
			log.Fatal(err)
		}

		tx, err := blockchainparser.NewTxFromFile(datadir, magicId, uint32(num), uint32(pos), uint32(txPos))
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("%+v\n", tx)
	} else if len(args) == 2 && args[0] == "GetFileInfoRecord" {
		failIfReindexing(indexDb)
		num, err := strconv.ParseUint(args[1], 10, 32)
		if err != nil {
			log.Fatal(err)
		}
		result, err := db.GetFileInfoRecord(indexDb, uint32(num))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%+v\n", result)
	} else if args[0] == "GetLastBlockFileNumberUsed" {
		result, err := db.GetLastBlockFileNumberUsed(indexDb)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%+v\n", result)
	} else if len(args) == 2 && args[0] == "GetFlag" {
		f, _ := db.GetFlag(indexDb, []byte(args[1]))
		fmt.Printf("flag %s = %+v\n", args[1], f)
	} else if args[0] == "GetReindexing" {
		result, err := db.GetReindexing(indexDb)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%+v\n", result)
	} else {
		showHelp()
		return
	}
}

func failIfReindexing(indexDb *db.IndexDb) {
	result, err := db.GetReindexing(indexDb)
	if err != nil {
		log.Fatal(err)
	}
	if result {
		log.Fatal(errors.New("bitcoind is reindexing"))
	}
}
