package utils

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestSignAndVerify(t *testing.T) {
	// Generate a test private key
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Get the expected address from the private key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("Failed to assert public key type")
	}
	expectedAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Test with various value types
	values := []any{
		big.NewInt(12345),
		uint64(67890),
		uint32(11111),
		common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb"),
		"test string",
		[]byte{0x01, 0x02, 0x03},
	}

	// Sign the values
	signature, err := Sign(values, privateKey)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	if len(signature) != 65 {
		t.Fatalf("Expected signature length 65, got %d", len(signature))
	}

	// Verify with correct address
	valid, err := Verify(values, signature, expectedAddress)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Fatal("Signature verification failed with correct address")
	}

	// Verify with incorrect address
	wrongAddress := common.HexToAddress("0x0000000000000000000000000000000000000001")
	valid, err = Verify(values, signature, wrongAddress)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if valid {
		t.Fatal("Signature verification should have failed with wrong address")
	}
}

func TestSignAndVerifyWithNumbers(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("Failed to assert public key type")
	}
	expectedAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Test with different number types
	testCases := []struct {
		name   string
		values []any
	}{
		{"big.Int", []any{big.NewInt(1000000)}},
		{"uint64", []any{uint64(999999)}},
		{"uint32", []any{uint32(12345)}},
		{"uint16", []any{uint16(1234)}},
		{"uint8", []any{uint8(123)}},
		{"int64", []any{int64(500000)}},
		{"int32", []any{int32(25000)}},
		{"int16", []any{int16(2500)}},
		{"int8", []any{int8(125)}},
		{"int", []any{int(100000)}},
		{"uint", []any{uint(200000)}},
		{"mixed numbers", []any{
			big.NewInt(1),
			uint64(2),
			uint32(3),
			uint16(4),
			uint8(5),
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			signature, err := Sign(tc.values, privateKey)
			if err != nil {
				t.Fatalf("Sign failed: %v", err)
			}

			valid, err := Verify(tc.values, signature, expectedAddress)
			if err != nil {
				t.Fatalf("Verify failed: %v", err)
			}
			if !valid {
				t.Fatal("Signature verification failed")
			}
		})
	}
}

func TestSignAndVerifyWithAddress(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("Failed to assert public key type")
	}
	expectedAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	testAddress := common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb")

	values := []any{
		big.NewInt(1),
		testAddress,
		big.NewInt(2),
	}

	signature, err := Sign(values, privateKey)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	valid, err := Verify(values, signature, expectedAddress)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Fatal("Signature verification failed")
	}
}

func TestSignAndVerifyWithBytes(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("Failed to assert public key type")
	}
	expectedAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Test with bytes32
	var bytes32 [32]byte
	copy(bytes32[:], []byte("test bytes32 value here!!"))

	// Test with regular bytes
	bytesVal := []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	testCases := []struct {
		name   string
		values []any
	}{
		{"bytes32", []any{bytes32}},
		{"bytes", []any{bytesVal}},
		{"mixed with bytes", []any{
			big.NewInt(1),
			bytes32,
			bytesVal,
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			signature, err := Sign(tc.values, privateKey)
			if err != nil {
				t.Fatalf("Sign failed: %v", err)
			}

			valid, err := Verify(tc.values, signature, expectedAddress)
			if err != nil {
				t.Fatalf("Verify failed: %v", err)
			}
			if !valid {
				t.Fatal("Signature verification failed")
			}
		})
	}
}

func TestSignAndVerifyWithString(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("Failed to assert public key type")
	}
	expectedAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	values := []any{
		big.NewInt(1),
		"test string value",
		big.NewInt(2),
	}

	signature, err := Sign(values, privateKey)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	valid, err := Verify(values, signature, expectedAddress)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Fatal("Signature verification failed")
	}
}

func TestSolidityPackedKeccak256(t *testing.T) {
	// Test with empty values
	hash, err := SolidityPackedKeccak256([]any{})
	if err != nil {
		t.Fatalf("SolidityPackedKeccak256 failed: %v", err)
	}
	if hash == (common.Hash{}) {
		t.Fatal("Hash should not be zero")
	}

	// Test with single value
	hash1, err := SolidityPackedKeccak256([]any{big.NewInt(123)})
	if err != nil {
		t.Fatalf("SolidityPackedKeccak256 failed: %v", err)
	}

	// Test with same value - should produce same hash
	hash2, err := SolidityPackedKeccak256([]any{big.NewInt(123)})
	if err != nil {
		t.Fatalf("SolidityPackedKeccak256 failed: %v", err)
	}
	if hash1 != hash2 {
		t.Fatal("Same values should produce same hash")
	}

	// Test with different values - should produce different hash
	hash3, err := SolidityPackedKeccak256([]any{big.NewInt(456)})
	if err != nil {
		t.Fatalf("SolidityPackedKeccak256 failed: %v", err)
	}
	if hash1 == hash3 {
		t.Fatal("Different values should produce different hash")
	}

	// Test with multiple values
	values := []any{
		big.NewInt(1),
		uint64(2),
		common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb"),
		"test",
	}
	hash4, err := SolidityPackedKeccak256(values)
	if err != nil {
		t.Fatalf("SolidityPackedKeccak256 failed: %v", err)
	}
	if hash4 == (common.Hash{}) {
		t.Fatal("Hash should not be zero")
	}
}

func TestSolidityPackedKeccak256Deterministic(t *testing.T) {
	// Test that the same values always produce the same hash
	values := []any{
		big.NewInt(100),
		uint64(200),
		uint32(300),
		common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb"),
		"deterministic test",
		[]byte{0x01, 0x02, 0x03},
	}

	hash1, err := SolidityPackedKeccak256(values)
	if err != nil {
		t.Fatalf("SolidityPackedKeccak256 failed: %v", err)
	}

	hash2, err := SolidityPackedKeccak256(values)
	if err != nil {
		t.Fatalf("SolidityPackedKeccak256 failed: %v", err)
	}

	if hash1 != hash2 {
		t.Fatal("Same values should always produce the same hash")
	}
}

func TestVerifyWithWrongSignature(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("Failed to assert public key type")
	}
	expectedAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	values := []any{big.NewInt(123)}

	// Create a wrong signature (all zeros)
	wrongSignature := make([]byte, 65)

	valid, err := Verify(values, wrongSignature, expectedAddress)
	if err == nil && valid {
		t.Fatal("Verification should fail with wrong signature")
	}
}

func TestVerifyWithModifiedValues(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("Failed to assert public key type")
	}
	expectedAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	values := []any{big.NewInt(123)}
	signature, err := Sign(values, privateKey)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	// Try to verify with modified values
	modifiedValues := []any{big.NewInt(456)}
	valid, err := Verify(modifiedValues, signature, expectedAddress)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if valid {
		t.Fatal("Verification should fail with modified values")
	}
}
