package models

import "database/sql"

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	Name         string
}

// GetUserByEmail dipakai pas proses login buat cari akun berdasarkan email yang diinput di form.
func GetUserByEmail(db *sql.DB, email string) (*User, error) {
	row := db.QueryRow(`SELECT id, email, password_hash, name FROM users WHERE email = ?`, email)

	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name); err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByID dipakai buat ambil data user dari user_id yang ada di JWT claims (endpoint /me).
func GetUserByID(db *sql.DB, id int64) (*User, error) {
	row := db.QueryRow(`SELECT id, email, password_hash, name FROM users WHERE id = ?`, id)

	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name); err != nil {
		return nil, err
	}
	return &u, nil
}
