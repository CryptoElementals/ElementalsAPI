package stress

import (
	"encoding/csv"
	"fmt"
	"os"
	"sync"

	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// BotInfoManager manages bot information in CSV file and wallet creation/loading
type BotInfoManager struct {
	csvPath string
	mu      sync.RWMutex
}

// NewBotInfoManager creates a new bot info manager
func NewBotInfoManager(csvPath string) (*BotInfoManager, error) {
	manager := &BotInfoManager{
		csvPath: csvPath,
	}

	// Create CSV file with header if it doesn't exist
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		if err := manager.writeHeader(); err != nil {
			return nil, fmt.Errorf("failed to create CSV file: %w", err)
		}
	}

	return manager, nil
}

// writeHeader writes the CSV header
func (m *BotInfoManager) writeHeader() error {
	file, err := os.Create(m.csvPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	return writer.Write([]string{"private_key", "tmp_key", "refresh_token"})
}

// GetOrCreateWallets gets existing wallets from CSV or creates new ones
// If CSV has enough records, returns wallets from CSV. Otherwise creates new wallets and saves them to CSV.
// Returns wallets for the specified number of bots
func (m *BotInfoManager) GetOrCreateWallets(numBots int) ([]*wallet.Wallet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Read existing records
	existingRecords, err := m.readAllRecords()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV records: %w", err)
	}

	wallets := make([]*wallet.Wallet, 0, numBots)

	// Load existing wallets from CSV
	existingCount := len(existingRecords)
	if existingCount > numBots {
		existingCount = numBots
	}

	for i := 0; i < existingCount && len(wallets) < numBots; i++ {
		record := existingRecords[i]
		if len(record) < 1 || record[0] == "" {
			// Skip invalid records, will create new wallet instead
			continue
		}

		w, err := m.loadWalletFromHex(record[0])
		if err != nil {
			return nil, fmt.Errorf("failed to load wallet from CSV record %d: %w", i, err)
		}
		wallets = append(wallets, w)
	}

	// Create new wallets if needed and save to CSV
	if len(wallets) < numBots {
		file, err := os.OpenFile(m.csvPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open CSV file for appending: %w", err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		for i := len(wallets); i < numBots; i++ {
			w, err := wallet.NewWallet("") // Don't save to file, we'll save to CSV
			if err != nil {
				return nil, fmt.Errorf("failed to create wallet: %w", err)
			}

			// Save new wallet to CSV (private_key, tmp_key, empty refresh_token)
			privateKey := w.GetPrivateKeyHex()
			tmpKey := w.GetPrivateKeyHex() // tmp_key is same as private_key

			record := []string{
				privateKey,
				tmpKey,
				"", // refresh_token will be updated after login
			}

			if err := writer.Write(record); err != nil {
				return nil, fmt.Errorf("failed to write CSV record: %w", err)
			}

			wallets = append(wallets, w)
		}
	}

	return wallets, nil
}

// loadWalletFromHex creates a wallet from a hex private key string
func (m *BotInfoManager) loadWalletFromHex(privateKeyHex string) (*wallet.Wallet, error) {
	if privateKeyHex == "" {
		return nil, fmt.Errorf("empty private key hex string")
	}

	// Ensure we have "0x" prefix
	hexStr := privateKeyHex
	if len(hexStr) >= 2 && hexStr[:2] != "0x" {
		hexStr = "0x" + hexStr
	}

	privateKeyBytes := common.FromHex(hexStr)
	if len(privateKeyBytes) == 0 {
		return nil, fmt.Errorf("invalid private key hex string")
	}

	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return wallet.NewWalletFromPrivKey(privateKey), nil
}

// readAllRecords reads all records from CSV (excluding header)
func (m *BotInfoManager) readAllRecords() ([][]string, error) {
	file, err := os.Open(m.csvPath)
	if err != nil {
		if os.IsNotExist(err) {
			return [][]string{}, nil
		}
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file: %w", err)
	}

	// Skip header
	if len(records) > 0 {
		return records[1:], nil
	}
	return [][]string{}, nil
}

// UpdateRefreshToken updates the refresh token for a bot at the specified index
func (m *BotInfoManager) UpdateRefreshToken(botIndex int, refreshToken string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Read all records
	file, err := os.Open(m.csvPath)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	file.Close()
	if err != nil {
		return fmt.Errorf("failed to read CSV file: %w", err)
	}

	// Check if botIndex is valid (accounting for header)
	if botIndex+1 >= len(records) {
		return fmt.Errorf("bot index %d out of range", botIndex)
	}

	// Update refresh_token (column index 2)
	record := records[botIndex+1]
	if len(record) < 3 {
		return fmt.Errorf("invalid CSV record format at index %d", botIndex)
	}

	// Skip update if token hasn't changed
	if record[2] == refreshToken {
		return nil
	}

	record[2] = refreshToken
	records[botIndex+1] = record

	// Write back to file
	file, err = os.Create(m.csvPath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, rec := range records {
		if err := writer.Write(rec); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	return nil
}
