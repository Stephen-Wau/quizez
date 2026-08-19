package handlers

import (
	"database/sql"
	"errors"
	"net"
	"net/http"
	"strings"

	"quizez/backend/internal/auth"
	"quizez/backend/internal/models"
	"quizez/backend/internal/response"
)

// requireSuperAdmin pastikan aksi sensitif cuma bisa dijalankan akun dengan role super_admin.
func requireSuperAdmin(w http.ResponseWriter, r *http.Request, db *sql.DB) bool {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return false
	}

	role, err := models.GetUserRoleByID(db, claims.UserID)
	if errors.Is(err, sql.ErrNoRows) {
		response.Error(w, http.StatusUnauthorized, "user not found")
		return false
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to validate permission")
		return false
	}
	if role != models.UserRoleSuperAdmin {
		response.Error(w, http.StatusForbidden, "forbidden")
		return false
	}
	return true
}

// writeAuditLog catat aksi admin penting secara best-effort tanpa mengganggu request utama saat log gagal.
func writeAuditLog(r *http.Request, db *sql.DB, actionKey, entityType string, entityID *int64, description string) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		return
	}
	actorName, actorEmail := currentActorSnapshot(db, claims.UserID)
	_ = models.CreateAuditLog(db, models.AuditLogInput{
		ActorUserID:        claims.UserID,
		ActorNameSnapshot:  actorName,
		ActorEmailSnapshot: actorEmail,
		ActionKey:          actionKey,
		EntityType:         entityType,
		EntityID:           entityID,
		Description:        description,
		IPAddress:          requestIPAddress(r),
		UserAgent:          strings.TrimSpace(r.UserAgent()),
	})
}

// writeAuditLogForUser dipakai saat login sukses karena request itu belum lewat middleware auth.
func writeAuditLogForUser(r *http.Request, db *sql.DB, actorUserID int64, actionKey, entityType string, entityID *int64, description string) {
	actorName, actorEmail := currentActorSnapshot(db, actorUserID)
	_ = models.CreateAuditLog(db, models.AuditLogInput{
		ActorUserID:        actorUserID,
		ActorNameSnapshot:  actorName,
		ActorEmailSnapshot: actorEmail,
		ActionKey:          actionKey,
		EntityType:         entityType,
		EntityID:           entityID,
		Description:        description,
		IPAddress:          requestIPAddress(r),
		UserAgent:          strings.TrimSpace(r.UserAgent()),
	})
}

// requestIPAddress ambil IP client efektif untuk audit trail tanpa ikut membawa port koneksi.
func requestIPAddress(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

// currentActorSnapshot simpan nama/email aktor saat itu juga supaya audit log tetap terbaca walau user dihapus.
func currentActorSnapshot(db *sql.DB, userID int64) (string, string) {
	user, err := models.GetUserByID(db, userID)
	if err != nil {
		return "", ""
	}
	return user.Name, user.Email
}
