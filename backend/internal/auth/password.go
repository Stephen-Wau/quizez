package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword hash password plaintext pakai bcrypt sebelum disimpan ke DB, jangan pernah
// simpan password mentah-mentah.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword bandingin password plaintext dari form login sama hash yang tersimpan di DB.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
