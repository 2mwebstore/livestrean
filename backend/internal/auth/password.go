package auth

import (
	"sync"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// MaxPasswordBytes is bcrypt's input limit.
const MaxPasswordBytes = 72

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(hash), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

var (
	dummyOnce sync.Once
	dummyHash string
)

// EqualizeTiming spends as long as a real password check. Call it when the
// email is unknown, so login timing doesn't reveal which emails exist.
func EqualizeTiming(password string) {
	dummyOnce.Do(func() { dummyHash, _ = HashPassword("timing-equalizer") })
	CheckPassword(dummyHash, password)
}
