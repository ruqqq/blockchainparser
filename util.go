package blockchainparser

import (
	"crypto/sha256"
	"os"
	"runtime"
)

func BitcoinDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("APPDATA")
		return home + "/Bitcoin"
	} else if runtime.GOOS == "osx" || runtime.GOOS == "darwin" {
		return os.Getenv("HOME") + "/Library/Application Support/Bitcoin"
	}

	return os.Getenv("HOME") + "/.bitcoin"
}

// TODO: Maybe can optimize
func ReverseHex(b []byte) []byte {
	newb := make([]byte, len(b))
	copy(newb, b)
	for i := len(newb)/2 - 1; i >= 0; i-- {
		opp := len(newb) - 1 - i
		newb[i], newb[opp] = newb[opp], newb[i]
	}

	return newb
}

func DoubleSha256(data []byte) Hash256 {
	hash := sha256.New()
	hash.Write(data)
	firstSha256 := hash.Sum(nil)
	hash.Reset()
	hash.Write(firstSha256)
	return hash.Sum(nil)
}
