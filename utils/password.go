package utils

import (
	"fmt"
	"math/rand"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func StringPtr(s string) *string {
	return &s
}

func PtrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func GenerateRandomPin() string {
	pin := rand.Intn(999999) + 100000
	return fmt.Sprintf("%06d", pin)
}
