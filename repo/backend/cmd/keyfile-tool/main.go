package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type secretsFile struct {
	DatabaseURL   string `json:"database_url"`
	JWTSigningKey string `json:"jwt_signing_key"`
	AESMasterKey  string `json:"aes_master_key"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  keyfile-tool encrypt <secrets.json> <output.enc> <passphrase>\n")
		fmt.Fprintf(os.Stderr, "  keyfile-tool decrypt <input.enc> <passphrase>\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "encrypt":
		if len(os.Args) != 5 {
			fmt.Fprintf(os.Stderr, "Usage: keyfile-tool encrypt <secrets.json> <output.enc> <passphrase>\n")
			os.Exit(1)
		}
		if err := encryptCmd(os.Args[2], os.Args[3], os.Args[4]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "decrypt":
		if len(os.Args) != 4 {
			fmt.Fprintf(os.Stderr, "Usage: keyfile-tool decrypt <input.enc> <passphrase>\n")
			os.Exit(1)
		}
		if err := decryptCmd(os.Args[2], os.Args[3]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func encryptCmd(inputPath, outputPath, passphrase string) error {
	plaintext, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", inputPath, err)
	}

	// Validate JSON structure
	var secrets secretsFile
	if err := json.Unmarshal(plaintext, &secrets); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if secrets.DatabaseURL == "" || secrets.JWTSigningKey == "" || secrets.AESMasterKey == "" {
		return fmt.Errorf("secrets file must contain database_url, jwt_signing_key, and aes_master_key")
	}

	ciphertext, err := encryptKeyfile(plaintext, passphrase)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	if err := os.WriteFile(outputPath, ciphertext, 0600); err != nil {
		return fmt.Errorf("write %s: %w", outputPath, err)
	}

	fmt.Printf("Encrypted keyfile written to %s (mode 0600)\n", outputPath)
	return nil
}

func decryptCmd(inputPath, passphrase string) error {
	ciphertext, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", inputPath, err)
	}

	plaintext, err := decryptKeyfile(ciphertext, passphrase)
	if err != nil {
		return fmt.Errorf("decrypt: %w", err)
	}

	// Pretty-print the JSON
	var raw json.RawMessage
	if err := json.Unmarshal(plaintext, &raw); err != nil {
		os.Stdout.Write(plaintext)
		return nil
	}
	pretty, _ := json.MarshalIndent(raw, "", "  ")
	os.Stdout.Write(pretty)
	fmt.Println()
	return nil
}

func encryptKeyfile(plaintext []byte, passphrase string) ([]byte, error) {
	key := sha256.Sum256([]byte(passphrase))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decryptKeyfile(ciphertext []byte, passphrase string) ([]byte, error) {
	key := sha256.Sum256([]byte(passphrase))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("keyfile too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}
