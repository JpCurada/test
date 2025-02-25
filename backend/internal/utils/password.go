package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword generates a bcrypt hash from a password
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// CheckPassword compares a bcrypt hashed password with its plaintext version
func CheckPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// GenerateRandomToken creates a secure random token for email verification or password reset
func GenerateRandomToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GenerateOTP generates a numeric OTP of the specified length
func GenerateOTP(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("length must be positive")
	}

	// Define the possible digits
	const digits = "0123456789"
	
	// Create a byte slice to store the OTP
	result := make([]byte, length)
	
	// Fill each position with a random digit
	for i := 0; i < length; i++ {
		// Generate a random index within the digits string
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		
		// Set the digit at the current position
		result[i] = digits[num.Int64()]
	}
	
	return string(result), nil
}