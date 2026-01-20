package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

// KeyPair represents an SSH key pair with both private and public keys
type KeyPair struct {
	PrivateKey string // PEM-encoded private key
	PublicKey  string // OpenSSH format public key
}

// GenerateED25519KeyPair generates a new ED25519 SSH key pair
// Returns the private key in PEM format and public key in OpenSSH format
func GenerateED25519KeyPair(comment string) (*KeyPair, error) {
	// Generate ED25519 key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ED25519 key: %w", err)
	}

	// Convert private key to PEM format
	privateKeyPEM, err := encodePrivateKeyToPEM(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encode private key: %w", err)
	}

	// Convert public key to OpenSSH format
	publicKeySSH, err := encodePublicKeyToOpenSSH(publicKey, comment)
	if err != nil {
		return nil, fmt.Errorf("failed to encode public key: %w", err)
	}

	return &KeyPair{
		PrivateKey: privateKeyPEM,
		PublicKey:  publicKeySSH,
	}, nil
}

// encodePrivateKeyToPEM encodes an ED25519 private key to OpenSSH PEM format
func encodePrivateKeyToPEM(privateKey ed25519.PrivateKey) (string, error) {
	// Use MarshalPrivateKey to create the OpenSSH private key format
	pemBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		return "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	return string(pem.EncodeToMemory(pemBlock)), nil
}

// encodePublicKeyToOpenSSH encodes an ED25519 public key to OpenSSH authorized_keys format
func encodePublicKeyToOpenSSH(publicKey ed25519.PublicKey, comment string) (string, error) {
	sshPublicKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to create SSH public key: %w", err)
	}

	// Get the authorized_keys format
	authorizedKey := ssh.MarshalAuthorizedKey(sshPublicKey)

	// Trim the trailing newline and add comment if provided
	result := string(authorizedKey)
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}

	if comment != "" {
		result = result + " " + comment
	}

	return result, nil
}
