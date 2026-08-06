package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"quizez/backend/internal/auth"
	"quizez/backend/internal/models"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

// LoginHandler cek kredensial email/password, kalau cocok terbitkan JWT buat dipakai FE
// di request-request selanjutnya.
func LoginHandler(db *sql.DB, jwtSecret string, jwtExpiryHours int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Sengaja gabungin 2 kondisi ini (user gak ketemu ATAU password salah) jadi 1 pesan error
		// generik, biar gak bocorin ke attacker mana yang salah (email atau password).
		user, err := models.GetUserByEmail(db, req.Email)
		if err != nil || !auth.CheckPassword(user.PasswordHash, req.Password) {
			http.Error(w, "invalid email or password", http.StatusUnauthorized)
			return
		}

		token, expiresAt, err := auth.GenerateToken(jwtSecret, jwtExpiryHours, user.ID, user.Email)
		if err != nil {
			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(loginResponse{
			Token:     token,
			ExpiresAt: expiresAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
}

// MeHandler return data user yang lagi login (dari claims JWT), dipakai FE buat nampilin
// profil/nama user setelah login.
func MeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := auth.ClaimsFromContext(r.Context())
		if claims == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := models.GetUserByID(db, claims.UserID)
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		})
	}
}
