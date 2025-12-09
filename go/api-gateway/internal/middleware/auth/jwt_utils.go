package auth

import (
	"crypto/rsa"
	"fmt"
	"os"
)

// LoadPublicKeysFromFiles はファイルから公開鍵を読み込む
func LoadPublicKeysFromFiles(keyFiles map[string]string) (map[string]*rsa.PublicKey, error) {
	publicKeys := make(map[string]*rsa.PublicKey)

	for kid, filePath := range keyFiles {
		pemData, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read public key file for kid=%s: %w", kid, err)
		}

		publicKey, err := parsePublicKeyFromPEM(string(pemData))
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key for kid=%s: %w", kid, err)
		}

		publicKeys[kid] = publicKey
	}

	return publicKeys, nil
}

// LoadPublicKeysFromPEMs はPEM文字列から公開鍵を読み込む
func LoadPublicKeysFromPEMs(publicKeyPEMs map[string]string) (map[string]*rsa.PublicKey, error) {
	publicKeys := make(map[string]*rsa.PublicKey)

	for kid, pemStr := range publicKeyPEMs {
		publicKey, err := parsePublicKeyFromPEM(pemStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key for kid=%s: %w", kid, err)
		}
		publicKeys[kid] = publicKey
	}

	return publicKeys, nil
}
