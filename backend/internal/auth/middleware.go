package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const claimsContextKey contextKey = "claims"

// RequireAuth bungkus handler yang butuh login: cek header Authorization Bearer, validasi
// tokennya, terus taruh claims di context biar handler di dalamnya bisa tau siapa yang request.
func RequireAuth(secret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(header, "Bearer ")
		claims, err := ParseToken(secret, tokenString)
		if err != nil || claims == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), claimsContextKey, claims)
		next(w, r.WithContext(ctx))
	}
}

// ClaimsFromContext ambil claims user yang udah ditaruh RequireAuth, dipakai handler buat tau
// user_id/email yang lagi login. Return nil kalau context-nya gak ada claims (belum lewat RequireAuth).
func ClaimsFromContext(ctx context.Context) *Claims {
	claims, ok := ctx.Value(claimsContextKey).(*Claims)
	if !ok {
		return nil
	}
	return claims
}
