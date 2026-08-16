package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"quizez/backend/internal/auth"
	"quizez/backend/internal/listquery"
	"quizez/backend/internal/models"
)

type adminUserRequest struct {
	Email    *string `json:"email"`
	Name     *string `json:"name"`
	Role     *string `json:"role"`
	Password *string `json:"password"`
}

// AdminUsersHandler handle list & create admin CMS, khusus super admin.
func AdminUsersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !requireSuperAdmin(w, r, db) {
			return
		}
		switch r.Method {
		case http.MethodGet:
			listAdminUsers(w, r, db)
		case http.MethodPost:
			createAdminUser(w, r, db)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// AdminUserHandler handle update & delete admin CMS, khusus super admin.
func AdminUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !requireSuperAdmin(w, r, db) {
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodPut:
			updateAdminUser(w, r, db, id)
		case http.MethodDelete:
			deleteAdminUser(w, r, db, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// AuditLogsHandler balikin daftar audit trail admin untuk investigasi aktivitas CMS.
func AuditLogsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		params := listquery.Parse(r)
		items, total, err := models.ListAuditLogs(db, params)
		if err != nil {
			http.Error(w, "failed to load audit logs", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listquery.ListResponse[models.AuditLog]{
			Data: items,
			Meta: listquery.BuildMeta(params, total),
		})
	}
}

// validateAdminUserRequest cek payload admin management supaya email/role/password tetap konsisten.
func validateAdminUserRequest(req adminUserRequest, isCreate bool) string {
	if req.Name == nil || strings.TrimSpace(*req.Name) == "" {
		return "Nama admin wajib diisi."
	}
	if req.Email == nil || strings.TrimSpace(*req.Email) == "" {
		return "Email admin wajib diisi."
	}
	if req.Role == nil || !models.IsValidUserRole(strings.TrimSpace(*req.Role)) {
		return "Role admin harus super_admin atau editor."
	}
	if isCreate && (req.Password == nil || strings.TrimSpace(*req.Password) == "") {
		return "Password admin wajib diisi."
	}
	if req.Password != nil && strings.TrimSpace(*req.Password) != "" && len(strings.TrimSpace(*req.Password)) < 8 {
		return "Password admin minimal 8 karakter."
	}
	return ""
}

// listAdminUsers handle GET /api/admin-users dengan pola pagination/search/sort standar DataTable.
func listAdminUsers(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	params := listquery.Parse(r)
	items, total, err := models.ListUsers(db, params)
	if err != nil {
		http.Error(w, "failed to load admin users", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(listquery.ListResponse[models.UserListItem]{
		Data: items,
		Meta: listquery.BuildMeta(params, total),
	})
}

// createAdminUser handle POST /api/admin-users untuk menambah admin CMS baru.
func createAdminUser(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req adminUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if msg := validateAdminUserRequest(req, true); msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	id, err := models.CreateUser(db, models.UserCreateInput{
		Email:    strings.TrimSpace(*req.Email),
		Name:     strings.TrimSpace(*req.Name),
		Role:     strings.TrimSpace(*req.Role),
		Password: strings.TrimSpace(*req.Password),
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			http.Error(w, "Email admin sudah dipakai.", http.StatusConflict)
			return
		}
		http.Error(w, "failed to create admin user", http.StatusInternalServerError)
		return
	}

	writeAuditLog(r, db, "admin_user.create", "admin_user", &id, "Menambah admin CMS baru.")

	user, err := models.GetUserByID(db, id)
	if err != nil {
		http.Error(w, "failed to load admin user", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
		"role":  user.Role,
	})
}

// updateAdminUser handle PUT /api/admin-users/{id} termasuk opsi ganti password bila diisi.
func updateAdminUser(w http.ResponseWriter, r *http.Request, db *sql.DB, id int64) {
	var req adminUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if msg := validateAdminUserRequest(req, false); msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	nextRole := strings.TrimSpace(*req.Role)
	if err := models.EnsureSuperAdminStillExists(db, id, &nextRole, false); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	var password *string
	if req.Password != nil && strings.TrimSpace(*req.Password) != "" {
		trimmed := strings.TrimSpace(*req.Password)
		password = &trimmed
	}
	if err := models.UpdateUser(db, models.UserUpdateInput{
		ID:       id,
		Email:    strings.TrimSpace(*req.Email),
		Name:     strings.TrimSpace(*req.Name),
		Role:     nextRole,
		Password: password,
	}); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			http.Error(w, "Email admin sudah dipakai.", http.StatusConflict)
			return
		}
		http.Error(w, "failed to update admin user", http.StatusInternalServerError)
		return
	}

	writeAuditLog(r, db, "admin_user.update", "admin_user", &id, "Memperbarui data admin CMS.")

	user, err := models.GetUserByID(db, id)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Admin tidak ditemukan.", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to load admin user", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
		"role":  user.Role,
	})
}

// deleteAdminUser handle DELETE /api/admin-users/{id} dengan guard self-delete & last super admin.
func deleteAdminUser(w http.ResponseWriter, r *http.Request, db *sql.DB, id int64) {
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.UserID == id {
		http.Error(w, "Akun sendiri tidak bisa dihapus.", http.StatusConflict)
		return
	}
	if err := models.EnsureSuperAdminStillExists(db, id, nil, true); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err := models.DeleteUser(db, id); err != nil {
		http.Error(w, "failed to delete admin user", http.StatusInternalServerError)
		return
	}
	writeAuditLog(r, db, "admin_user.delete", "admin_user", &id, "Menghapus admin CMS.")
	w.WriteHeader(http.StatusNoContent)
}
