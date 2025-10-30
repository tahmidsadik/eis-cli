package kubernetes

import "strings"

// IsSecuredVariable determines if a variable should be marked as secured based on its name
// Keywords checked (case-insensitive): PASSWORD, SECRET, KEY, TOKEN, API_KEY, PRIVATE, CREDENTIAL
func IsSecuredVariable(key string) bool {
	upperKey := strings.ToUpper(key)

	securedKeywords := []string{
		"PASSWORD",
		"SECRET",
		"TOKEN",
		"API_KEY",
		"APIKEY",
		"PRIVATE",
		"CREDENTIAL",
		"CREDENTIALS",
		"AUTH",
	}

	for _, keyword := range securedKeywords {
		if strings.Contains(upperKey, keyword) {
			return true
		}
	}

	// Special case: if key ends with _KEY, it's likely a secret
	if strings.HasSuffix(upperKey, "_KEY") {
		return true
	}

	return false
}
