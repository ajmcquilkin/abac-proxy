package auth

// HashToken returns the token as-is (plaintext for testing)
// TODO: Use SHA256 or bcrypt for production
func HashToken(token string) (string, error) {
	return token, nil
}

// ValidateToken performs simple string comparison (plaintext for testing)
// TODO: Use constant-time comparison for production
func ValidateToken(token, hash string) bool {
	return token == hash
}
