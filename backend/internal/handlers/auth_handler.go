package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"quizez/backend/internal/auth"
	"quizez/backend/internal/models"
	"quizez/backend/internal/response"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
	Role      string `json:"role"`
}

// LoginHandler cek kredensial email/password, kalau cocok terbitkan JWT buat dipakai FE
// di request-request selanjutnya.
func LoginHandler(db *sql.DB, jwtSecret string, jwtExpiryHours int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}

		// Sengaja gabungin 2 kondisi ini (user gak ketemu ATAU password salah) jadi 1 pesan error
		// generik, biar gak bocorin ke attacker mana yang salah (email atau password).
		user, err := models.GetUserByEmail(db, req.Email)
		if err != nil || !auth.CheckPassword(user.PasswordHash, req.Password) {
			response.Error(w, http.StatusUnauthorized, "invalid email or password")
			return
		}

		token, expiresAt, err := auth.GenerateToken(jwtSecret, jwtExpiryHours, user.ID, user.Email)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to generate token")
			return
		}
		writeAuditLogForUser(r, db, user.ID, "auth.login", "auth", nil, "Login CMS berhasil.")

		response.JSON(w, http.StatusOK, loginResponse{
			Token:     token,
			ExpiresAt: expiresAt.Format("2006-01-02T15:04:05Z07:00"),
			Role:      user.Role,
		})
	}
}

// MeHandler return data user yang lagi login (dari claims JWT), dipakai FE buat nampilin
// profil/nama user setelah login.
func MeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := auth.ClaimsFromContext(r.Context())
		if claims == nil {
			response.Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		user, err := models.GetUserByID(db, claims.UserID)
		if err != nil {
			response.Error(w, http.StatusNotFound, "user not found")
			return
		}

		response.JSON(w, http.StatusOK, map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		})
	}
}
