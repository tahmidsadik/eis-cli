package kubernetes

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ParseEnvTemplate parses a .env.template file and extracts variable keys
// Supports format: KEY=${VALUE_HERE}
// Ignores comments (lines starting with #) and empty lines
func ParseEnvTemplate(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open template file: %w", err)
	}
	defer file.Close()

	var keys []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE or KEY=${VALUE} format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) < 1 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		if key == "" {
			continue
		}

		// Validate key format (alphanumeric and underscores)
		if !isValidEnvKey(key) {
			fmt.Printf("Warning: Line %d has invalid key format '%s', skipping\n", lineNum, key)
			continue
		}

		keys = append(keys, key)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading template file: %w", err)
	}

	return keys, nil
}

// isValidEnvKey checks if a key is a valid environment variable name
// Valid keys contain only letters, numbers, and underscores, and don't start with a number
func isValidEnvKey(key string) bool {
	if len(key) == 0 {
		return false
	}

	// Check first character (must not be a number)
	firstChar := key[0]
	if !((firstChar >= 'A' && firstChar <= 'Z') ||
		(firstChar >= 'a' && firstChar <= 'z') ||
		firstChar == '_') {
		return false
	}

	// Check remaining characters
	for _, char := range key {
		if !((char >= 'A' && char <= 'Z') ||
			(char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '_') {
			return false
		}
	}

	return true
}
