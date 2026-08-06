package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateToken bikin JWT baru buat user yang habis login, dipakai FE buat autentikasi request
// selanjutnya. Return juga waktu expired-nya biar bisa ditampilin/disimpan di FE.
func GenerateToken(secret string, expiryHours int, userID int64, email string) (string, time.Time, error) {
	expiresAt := time.Now().Add(time.Duration(expiryHours) * time.Hour)

	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiresAt, nil
}

// ParseToken validasi signature & expiry token dari header Authorization, dipakai middleware
// RequireAuth buat cek request yang masuk beneran punya token yang sah.
func ParseToken(secret, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	// token.Valid ikut dicek karena ParseWithClaims bisa return err nil tapi token invalid (misal expired).
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}
