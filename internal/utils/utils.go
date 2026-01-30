package utils

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func ParseBigInt(value string) (*big.Int, error) {
	if value == "" {
		return big.NewInt(0), nil
	}
	clean := strings.TrimSpace(value)
	base := 10
	if strings.HasPrefix(clean, "0x") || strings.HasPrefix(clean, "0X") {
		base = 0
	}
	v, ok := new(big.Int).SetString(clean, base)
	if !ok {
		return nil, fmt.Errorf("invalid integer: %s", value)
	}
	return v, nil
}

func LeftPad32(b []byte) []byte {
	if len(b) >= 32 {
		return b[len(b)-32:]
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

func DecodeHex(data string) ([]byte, error) {
	if data == "" {
		return []byte{}, nil
	}
	if strings.HasPrefix(data, "0x") || strings.HasPrefix(data, "0X") {
		return hexutil.Decode(data)
	}
	decoded, err := hex.DecodeString(data)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func SplitAndPackSig(sig []byte) (string, error) {
	if len(sig) != 65 {
		return "", fmt.Errorf("invalid signature length: expected 65 bytes, got %d", len(sig))
	}
	r := sig[0:32]
	s := sig[32:64]
	v := sig[64]
	switch v {
	case 0, 1:
		v += 31
	case 27, 28:
		v += 4
	default:
		return "", fmt.Errorf("invalid signature v: %d", v)
	}

	packed := make([]byte, 0, 65)
	packed = append(packed, LeftPad32(r)...)
	packed = append(packed, LeftPad32(s)...)
	packed = append(packed, v)
	return hexutil.Encode(packed), nil
}

func Sleep(duration time.Duration) {
	time.Sleep(duration)
}