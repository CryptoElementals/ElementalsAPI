package utils

import (
	"crypto/ecdsa"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func Sign(values []any, privateKey *ecdsa.PrivateKey) (signature []byte, err error) {
	hash, err := SolidityPackedKeccak256(values)
	if err != nil {
		return nil, err
	}
	sig, err := crypto.Sign(hash.Bytes(), privateKey)
	if err != nil {
		return nil, err
	}
	sig[crypto.RecoveryIDOffset] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	return sig[:], nil
}

func Verify(values []any, signature []byte, expectedAddress common.Address) (bool, error) {
	hash, err := SolidityPackedKeccak256(values)
	if err != nil {
		return false, err
	}
	signatureSlice := slices.Clone(signature)
	signatureSlice[crypto.RecoveryIDOffset] -= 27 // Transform V from 27/28 to 0/1 according to the yellow paper
	// Recover public key from signature using Ethereum ECDSA recovery
	recoveredPubKey, err := crypto.SigToPub(hash.Bytes(), signatureSlice)
	if err != nil {
		return false, err
	}

	// Extract address from recovered public key
	recoveredAddress := crypto.PubkeyToAddress(*recoveredPubKey)

	// Verify if the recovered address matches the expected address
	return recoveredAddress == expectedAddress, nil
}

func SolidityPackedKeccak256(values []any) (common.Hash, error) {
	var packed []byte
	for _, val := range values {
		encoded, err := encodePackedValue(val)
		if err != nil {
			return common.Hash{}, err
		}
		packed = append(packed, encoded...)
	}

	hash := crypto.Keccak256Hash(packed)
	return hash, nil
}

// encodePackedValue 根据类型编码单个值（按照 abi.encodePacked 规则）
func encodePackedValue(val interface{}) ([]byte, error) {
	switch v := val.(type) {
	case common.Address:
		// address 在 packed 编码中是 20 字节（不是 32 字节）
		return v.Bytes(), nil

	case *big.Int:
		// uint256 在 packed 编码中需要填充到 32 字节
		return common.LeftPadBytes(v.Bytes(), 32), nil

	case uint64:
		bigInt := new(big.Int).SetUint64(v)
		return common.LeftPadBytes(bigInt.Bytes(), 32), nil

	case uint32:
		bigInt := big.NewInt(int64(v))
		return common.LeftPadBytes(bigInt.Bytes(), 32), nil

	case uint16:
		bigInt := big.NewInt(int64(v))
		return common.LeftPadBytes(bigInt.Bytes(), 32), nil

	case uint8:
		bigInt := big.NewInt(int64(v))
		return common.LeftPadBytes(bigInt.Bytes(), 32), nil

	case int64:
		bigInt := big.NewInt(v)
		return common.LeftPadBytes(bigInt.Bytes(), 32), nil

	case int32:
		bigInt := big.NewInt(int64(v))
		return common.LeftPadBytes(bigInt.Bytes(), 32), nil

	case int16:
		bigInt := big.NewInt(int64(v))
		return common.LeftPadBytes(bigInt.Bytes(), 32), nil

	case int8:
		bigInt := big.NewInt(int64(v))
		return common.LeftPadBytes(bigInt.Bytes(), 32), nil

	case int:
		bigInt := big.NewInt(int64(v))
		return common.LeftPadBytes(bigInt.Bytes(), 32), nil

	case uint:
		bigInt := new(big.Int).SetUint64(uint64(v))
		return common.LeftPadBytes(bigInt.Bytes(), 32), nil

	case [32]byte:
		return v[:], nil

	case []byte:
		if len(v) == 32 {
			return v, nil
		}
		// bytes 在 packed 编码中直接拼接内容（不包含长度）
		return v, nil

	case string:
		return []byte(v), nil

	default:
		return nil, ErrUnsupportedType
	}
}

// 错误定义
var (
	ErrInvalidType     = &SolidityError{msg: "invalid value type"}
	ErrInvalidLength   = &SolidityError{msg: "invalid value length"}
	ErrUnsupportedType = &SolidityError{msg: "unsupported type"}
)

type SolidityError struct {
	msg string
}

func (e *SolidityError) Error() string {
	return e.msg
}
