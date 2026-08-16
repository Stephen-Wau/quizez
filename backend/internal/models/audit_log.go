package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"quizez/backend/internal/listquery"
)

type AuditLog struct {
	ID          int64     `json:"id"`
	ActorUserID *int64    `json:"actor_user_id"`
	ActorName   string    `json:"actor_name"`
	ActorEmail  string    `json:"actor_email"`
	ActionKey   string    `json:"action_key"`
	EntityType  string    `json:"entity_type"`
	EntityID    *int64    `json:"entity_id"`
	Description string    `json:"description"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	CreatedAt   time.Time `json:"created_at"`
}

type AuditLogInput struct {
	ActorUserID        int64
	ActorNameSnapshot  string
	ActorEmailSnapshot string
	ActionKey          string
	EntityType         string
	EntityID           *int64
	Description        string
	IPAddress          string
	UserAgent          string
}

// CreateAuditLog simpan jejak aksi admin penting supaya perubahan sensitif bisa ditelusuri ulang.
func CreateAuditLog(db *sql.DB, input AuditLogInput) error {
	_, err := db.Exec(`
		INSERT INTO audit_logs (actor_user_id, actor_name_snapshot, actor_email_snapshot, action_key, entity_type, entity_id, description, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, input.ActorUserID, input.ActorNameSnapshot, input.ActorEmailSnapshot, input.ActionKey, input.EntityType, input.EntityID, input.Description, input.IPAddress, input.UserAgent)
	return err
}

// ListAuditLogs ambil daftar audit log dengan join actor biar FE bisa tampilkan pelaku aksinya.
func ListAuditLogs(db *sql.DB, params listquery.Params) ([]AuditLog, int, error) {
	search := "%" + strings.TrimSpace(params.SearchWord) + "%"
	whereParts := []string{}
	args := []any{}
	if strings.TrimSpace(params.SearchWord) != "" {
		// Search sengaja mencakup actor, action, entity, dan deskripsi biar investigasi cepat dari 1 box.
		whereParts = append(whereParts, `(u.name LIKE ? OR u.email LIKE ? OR l.action_key LIKE ? OR l.entity_type LIKE ? OR l.description LIKE ?)`)
		args = append(args, search, search, search, search, search)
	}

	whereSQL := ""
	if len(whereParts) > 0 {
		whereSQL = " WHERE " + strings.Join(whereParts, " AND ")
	}

	countQuery := `SELECT COUNT(*) FROM audit_logs l LEFT JOIN users u ON u.id = l.actor_user_id` + whereSQL
	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	sortCol := params.SortColumn(map[string]string{
		"created_at":  "l.created_at",
		"actor_name":  "u.name",
		"actor_email": "u.email",
		"action_key":  "l.action_key",
		"entity_type": "l.entity_type",
	}, "l.created_at")

	query := fmt.Sprintf(`
		SELECT l.id, l.actor_user_id, COALESCE(u.name, l.actor_name_snapshot), COALESCE(u.email, l.actor_email_snapshot), l.action_key, l.entity_type, l.entity_id, l.description, l.ip_address, l.user_agent, l.created_at
		FROM audit_logs l
		LEFT JOIN users u ON u.id = l.actor_user_id
		%s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, whereSQL, sortCol, params.SortDirSQL())

	pageArgs := append(append([]any{}, args...), params.PerPage, params.Offset())
	rows, err := db.Query(query, pageArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	logs := []AuditLog{}
	for rows.Next() {
		var item AuditLog
		var actorUserID sql.NullInt64
		var entityID sql.NullInt64
		if err := rows.Scan(
			&item.ID,
			&actorUserID,
			&item.ActorName,
			&item.ActorEmail,
			&item.ActionKey,
			&item.EntityType,
			&entityID,
			&item.Description,
			&item.IPAddress,
			&item.UserAgent,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		if actorUserID.Valid {
			item.ActorUserID = &actorUserID.Int64
		}
		if entityID.Valid {
			item.EntityID = &entityID.Int64
		}
		logs = append(logs, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
