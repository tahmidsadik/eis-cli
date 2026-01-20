package ssh

import (
	"strings"
	"testing"
)

func TestGenerateED25519KeyPair(t *testing.T) {
	tests := []struct {
		name    string
		comment string
	}{
		{
			name:    "Generate key pair with comment",
			comment: "test-service-pipeline",
		},
		{
			name:    "Generate key pair without comment",
			comment: "",
		},
		{
			name:    "Generate key pair with special characters in comment",
			comment: "my-service_v2-pipeline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyPair, err := GenerateED25519KeyPair(tt.comment)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify private key format
			if !strings.HasPrefix(keyPair.PrivateKey, "-----BEGIN OPENSSH PRIVATE KEY-----") {
				t.Errorf("private key should start with OpenSSH private key header, got: %s", keyPair.PrivateKey[:50])
			}
			if !strings.Contains(keyPair.PrivateKey, "-----END OPENSSH PRIVATE KEY-----") {
				t.Error("private key should contain OpenSSH private key footer")
			}

			// Verify public key format (ssh-ed25519)
			if !strings.HasPrefix(keyPair.PublicKey, "ssh-ed25519 ") {
				t.Errorf("public key should start with 'ssh-ed25519 ', got: %s", keyPair.PublicKey[:20])
			}

			// Verify comment is included in public key if provided
			if tt.comment != "" {
				if !strings.HasSuffix(keyPair.PublicKey, " "+tt.comment) {
					t.Errorf("public key should end with comment '%s', got: %s", tt.comment, keyPair.PublicKey)
				}
			}

			// Verify public key doesn't have trailing newline
			if strings.HasSuffix(keyPair.PublicKey, "\n") {
				t.Error("public key should not have trailing newline")
			}
		})
	}
}

func TestGenerateED25519KeyPair_UniquePairs(t *testing.T) {
	// Generate two key pairs and ensure they are different
	keyPair1, err := GenerateED25519KeyPair("test1")
	if err != nil {
		t.Fatalf("unexpected error generating first key pair: %v", err)
	}

	keyPair2, err := GenerateED25519KeyPair("test2")
	if err != nil {
		t.Fatalf("unexpected error generating second key pair: %v", err)
	}

	// Private keys should be different
	if keyPair1.PrivateKey == keyPair2.PrivateKey {
		t.Error("two generated key pairs should have different private keys")
	}

	// Public keys should be different
	// Extract just the key part (without comment) for comparison
	pubKey1Parts := strings.Fields(keyPair1.PublicKey)
	pubKey2Parts := strings.Fields(keyPair2.PublicKey)

	if len(pubKey1Parts) < 2 || len(pubKey2Parts) < 2 {
		t.Fatal("public keys should have at least 2 parts (type and key)")
	}

	if pubKey1Parts[1] == pubKey2Parts[1] {
		t.Error("two generated key pairs should have different public keys")
	}
}

func TestGenerateED25519KeyPair_PublicKeyFormat(t *testing.T) {
	keyPair, err := GenerateED25519KeyPair("test-comment")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Public key should have format: "ssh-ed25519 BASE64KEY comment"
	parts := strings.Fields(keyPair.PublicKey)
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts in public key (type, key, comment), got %d: %s", len(parts), keyPair.PublicKey)
	}

	if parts[0] != "ssh-ed25519" {
		t.Errorf("expected key type 'ssh-ed25519', got '%s'", parts[0])
	}

	// The key part should be base64 encoded (no spaces, reasonable length)
	keyPart := parts[1]
	if len(keyPart) < 60 || len(keyPart) > 100 {
		t.Errorf("unexpected key length: %d", len(keyPart))
	}

	if parts[2] != "test-comment" {
		t.Errorf("expected comment 'test-comment', got '%s'", parts[2])
	}
}
