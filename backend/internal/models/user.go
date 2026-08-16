package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"quizez/backend/internal/auth"
	"quizez/backend/internal/listquery"
)

const (
	UserRoleSuperAdmin = "super_admin"
	UserRoleEditor     = "editor"
)

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	Name         string
	Role         string
}

type UserListItem struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

type UserCreateInput struct {
	Email    string
	Password string
	Name     string
	Role     string
}

type UserUpdateInput struct {
	ID       int64
	Email    string
	Name     string
	Role     string
	Password *string
}

// GetUserByEmail dipakai pas proses login buat cari akun berdasarkan email yang diinput di form.
func GetUserByEmail(db *sql.DB, email string) (*User, error) {
	row := db.QueryRow(`SELECT id, email, password_hash, name, role FROM users WHERE email = ?`, email)

	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role); err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByID dipakai buat ambil data user dari user_id yang ada di JWT claims (endpoint /me).
func GetUserByID(db *sql.DB, id int64) (*User, error) {
	row := db.QueryRow(`SELECT id, email, password_hash, name, role FROM users WHERE id = ?`, id)

	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role); err != nil {
		return nil, err
	}
	return &u, nil
}

// IsValidUserRole batasi role user hanya ke daftar yang memang dipakai permission layer sekarang.
func IsValidUserRole(role string) bool {
	switch strings.TrimSpace(role) {
	case UserRoleSuperAdmin, UserRoleEditor:
		return true
	default:
		return false
	}
}

// ListUsers ambil daftar admin CMS lengkap dengan pagination/search/sort untuk menu kolaborasi.
func ListUsers(db *sql.DB, params listquery.Params) ([]UserListItem, int, error) {
	search := "%" + strings.TrimSpace(params.SearchWord) + "%"
	whereSQL := ""
	args := []any{}
	if strings.TrimSpace(params.SearchWord) != "" {
		// Search di-list admin cukup mencakup nama, email, dan role supaya pencarian operasional ringkas.
		whereSQL = ` WHERE name LIKE ? OR email LIKE ? OR role LIKE ?`
		args = append(args, search, search, search)
	}

	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	sortCol := params.SortColumn(map[string]string{
		"name":       "name",
		"email":      "email",
		"role":       "role",
		"created_at": "created_at",
	}, "created_at")

	query := fmt.Sprintf(`
		SELECT id, email, name, role, DATE_FORMAT(created_at, '%%Y-%%m-%%d %%H:%%i:%%s')
		FROM users
		%s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, whereSQL, sortCol, params.SortDirSQL())

	rows, err := db.Query(query, append(append([]any{}, args...), params.PerPage, params.Offset())...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := []UserListItem{}
	for rows.Next() {
		var item UserListItem
		if err := rows.Scan(&item.ID, &item.Email, &item.Name, &item.Role, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// CreateUser bikin akun admin baru lengkap dengan hash password & role yang dipilih super admin.
func CreateUser(db *sql.DB, input UserCreateInput) (int64, error) {
	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		return 0, err
	}
	result, err := db.Exec(`
		INSERT INTO users (email, password_hash, name, role)
		VALUES (?, ?, ?, ?)
	`, strings.TrimSpace(input.Email), hash, strings.TrimSpace(input.Name), strings.TrimSpace(input.Role))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpdateUser ubah profil admin, dan hanya ganti password kalau field password ikut dikirim.
func UpdateUser(db *sql.DB, input UserUpdateInput) error {
	if input.Password == nil {
		_, err := db.Exec(`
			UPDATE users
			SET email = ?, name = ?, role = ?
			WHERE id = ?
		`, strings.TrimSpace(input.Email), strings.TrimSpace(input.Name), strings.TrimSpace(input.Role), input.ID)
		return err
	}

	hash, err := auth.HashPassword(strings.TrimSpace(*input.Password))
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		UPDATE users
		SET email = ?, name = ?, role = ?, password_hash = ?
		WHERE id = ?
	`, strings.TrimSpace(input.Email), strings.TrimSpace(input.Name), strings.TrimSpace(input.Role), hash, input.ID)
	return err
}

// DeleteUser hapus akun admin target setelah semua guard bisnis (terakhir super admin, self-delete) lolos.
func DeleteUser(db *sql.DB, id int64) error {
	_, err := db.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

// CountSuperAdmins dipakai guard agar selalu tersisa minimal 1 akun super admin aktif.
func CountSuperAdmins(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE role = ?`, UserRoleSuperAdmin).Scan(&count)
	return count, err
}

// GetUserRoleByID ambil role terkini langsung dari DB biar perubahan permission efektif tanpa relogin.
func GetUserRoleByID(db *sql.DB, id int64) (string, error) {
	var role string
	err := db.QueryRow(`SELECT role FROM users WHERE id = ?`, id).Scan(&role)
	return role, err
}

// EnsureSuperAdminStillExists nolak operasi yang akan menghilangkan super admin terakhir di sistem.
func EnsureSuperAdminStillExists(db *sql.DB, affectedUserID int64, nextRole *string, deleting bool) error {
	user, err := GetUserByID(db, affectedUserID)
	if err != nil {
		return err
	}
	if user.Role != UserRoleSuperAdmin {
		return nil
	}
	if !deleting && nextRole != nil && strings.TrimSpace(*nextRole) == UserRoleSuperAdmin {
		return nil
	}
	count, err := CountSuperAdmins(db)
	if err != nil {
		return err
	}
	if count <= 1 {
		return errors.New("minimal harus ada 1 super admin")
	}
	return nil
}
