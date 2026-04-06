package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	ListenAddr    string
	DatabaseURL   string
	JWTSigningKey []byte
	AESMasterKey  []byte
}

type secretsFile struct {
	DatabaseURL   string `json:"database_url"`
	JWTSigningKey string `json:"jwt_signing_key"`
	AESMasterKey  string `json:"aes_master_key"`
}

func Load() (*Config, error) {
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	keyfile := os.Getenv("SECRETS_KEYFILE")
	if keyfile != "" {
		return loadFromKeyfile(listenAddr, keyfile)
	}

	// Plaintext env vars are only allowed when APP_ENV is explicitly set
	// to "development". All other values (including unset) require an
	// encrypted keyfile to prevent accidental secret exposure.
	appEnv := os.Getenv("APP_ENV")
	if appEnv != "development" {
		return nil, fmt.Errorf("SECRETS_KEYFILE is required when APP_ENV=%q; plaintext env vars are only allowed when APP_ENV is explicitly set to 'development'", appEnv)
	}

	return loadFromEnv(listenAddr)
}

func loadFromEnv(listenAddr string) (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	jwtKey := os.Getenv("JWT_SIGNING_KEY")
	if jwtKey == "" {
		return nil, fmt.Errorf("JWT_SIGNING_KEY is required")
	}

	aesKey, err := decodeAESKey(os.Getenv("AES_MASTER_KEY"))
	if err != nil {
		return nil, err
	}

	return &Config{
		ListenAddr:    listenAddr,
		DatabaseURL:   dbURL,
		JWTSigningKey: []byte(jwtKey),
		AESMasterKey:  aesKey,
	}, nil
}

func loadFromKeyfile(listenAddr, path string) (*Config, error) {
	passphrase := os.Getenv("SECRETS_PASSPHRASE")
	if passphrase == "" {
		return nil, fmt.Errorf("SECRETS_PASSPHRASE is required when using SECRETS_KEYFILE")
	}

	// Validate keyfile permissions: reject world-readable or group-readable files.
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat keyfile %s: %w", path, err)
	}
	if mode := info.Mode().Perm(); mode&0o077 != 0 {
		return nil, fmt.Errorf("keyfile %s has insecure permissions %04o; must be owner-only (0600)", path, mode)
	}

	ciphertext, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read keyfile %s: %w", path, err)
	}

	plaintext, err := decryptKeyfile(ciphertext, passphrase)
	if err != nil {
		return nil, fmt.Errorf("decrypt keyfile: %w", err)
	}

	var secrets secretsFile
	if err := json.Unmarshal(plaintext, &secrets); err != nil {
		return nil, fmt.Errorf("parse keyfile JSON: %w", err)
	}

	if secrets.DatabaseURL == "" || secrets.JWTSigningKey == "" || secrets.AESMasterKey == "" {
		return nil, fmt.Errorf("keyfile must contain database_url, jwt_signing_key, and aes_master_key")
	}

	aesKey, err := decodeAESKey(secrets.AESMasterKey)
	if err != nil {
		return nil, err
	}

	return &Config{
		ListenAddr:    listenAddr,
		DatabaseURL:   secrets.DatabaseURL,
		JWTSigningKey: []byte(secrets.JWTSigningKey),
		AESMasterKey:  aesKey,
	}, nil
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

func decodeAESKey(hexStr string) ([]byte, error) {
	if hexStr == "" {
		return nil, fmt.Errorf("AES_MASTER_KEY is required")
	}
	aesKey, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("AES_MASTER_KEY must be hex-encoded: %w", err)
	}
	if len(aesKey) != 32 {
		return nil, fmt.Errorf("AES_MASTER_KEY must be 32 bytes (64 hex chars), got %d bytes", len(aesKey))
	}
	return aesKey, nil
}
