package models

import (
	"database/sql"
	"fmt"
	"time"

	"quizez/backend/internal/listquery"
)

// whitelist FE `sort_by` (col.prop) -> kolom SQL asli. WAJIB whitelist, jangan interpolate raw.
var quizSortColumns = map[string]string{
	"title":      "title",
	"type":       "type",
	"start_time": "start_time",
	"status":     "status",
}

const dateTimeLayout = "2006-01-02T15:04:05"

type Quiz struct {
	ID            int64   `json:"id"`
	Title         *string `json:"title"`
	Type          *string `json:"type"`
	StartTime     *string `json:"start_time"`
	EndTime       *string `json:"end_time"`
	Description   *string `json:"description"`
	MaxPoint      *int    `json:"max_point"`
	TotalQuestion int     `json:"total_question"`
	Status        *string `json:"status"`
}

type quizRowScanner interface {
	Scan(dest ...interface{}) error
}

// ListQuizzes ambil daftar quiz buat DataTable FE: support search title, sort kolom whitelist,
// dan pagination. Total dihitung terpisah (COUNT query) biar meta pagination-nya akurat.
func ListQuizzes(db *sql.DB, params listquery.Params) ([]Quiz, int, error) {
	whereClause := ""
	args := []interface{}{}

	// Search title cuma ditambahin ke WHERE kalau user emang ngisi search box di FE.
	if params.SearchWord != "" {
		whereClause = " WHERE title LIKE ?"
		args = append(args, "%"+params.SearchWord+"%")
	}

	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM quizzes"+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	sortCol := params.SortColumn(quizSortColumns, "id")
	query := "SELECT q.id, q.title, q.type, q.start_time, q.end_time, q.description, q.max_point, " +
		"(SELECT COUNT(*) FROM questions WHERE quiz_id = q.id) AS total_question, q.status FROM quizzes q" + whereClause +
		fmt.Sprintf(" ORDER BY %s %s LIMIT ? OFFSET ?", sortCol, params.SortDirSQL())
	args = append(args, params.PerPage, params.Offset())

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	quizzes := []Quiz{}
	for rows.Next() {
		q, err := scanQuiz(rows)
		if err != nil {
			return nil, 0, err
		}
		quizzes = append(quizzes, q)
	}
	return quizzes, total, rows.Err()
}

// scanQuiz scan 1 baris hasil query quizzes ke struct Quiz, kolom nullable di DB (title, type,
// dst) di-scan ke sql.NullString/NullTime dulu baru dikonversi ke pointer biar JSON-nya bisa null.
func scanQuiz(row quizRowScanner) (Quiz, error) {
	var (
		q                    Quiz
		title, qType, status sql.NullString
		description          sql.NullString
		startTime, endTime   sql.NullTime
		maxPoint             sql.NullInt64
	)
	if err := row.Scan(&q.ID, &title, &qType, &startTime, &endTime, &description, &maxPoint, &q.TotalQuestion, &status); err != nil {
		return Quiz{}, err
	}
	q.Title = nullableString(title)
	q.Type = nullableString(qType)
	q.Status = nullableString(status)
	q.Description = nullableString(description)
	q.StartTime = nullableTime(startTime)
	q.EndTime = nullableTime(endTime)
	if maxPoint.Valid {
		v := int(maxPoint.Int64)
		q.MaxPoint = &v
	}
	return q, nil
}

// CreateQuiz insert quiz baru, dipakai handler POST /api/quizzes. Return last insert id-nya
// biar bisa langsung dipasang ke response.
func CreateQuiz(db *sql.DB, q Quiz) (int64, error) {
	startTime, err := timePtrValue(q.StartTime)
	if err != nil {
		return 0, err
	}
	endTime, err := timePtrValue(q.EndTime)
	if err != nil {
		return 0, err
	}

	res, err := db.Exec(
		"INSERT INTO quizzes (title, type, start_time, end_time, description, max_point, status) VALUES (?, ?, ?, ?, ?, ?, ?)",
		strPtrValue(q.Title), strPtrValue(q.Type), startTime, endTime, strPtrValue(q.Description), intPtrValue(q.MaxPoint), strPtrValue(q.Status),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateQuiz overwrite semua kolom quiz berdasarkan ID, dipakai handler PUT /api/quizzes/{id}.
func UpdateQuiz(db *sql.DB, q Quiz) error {
	startTime, err := timePtrValue(q.StartTime)
	if err != nil {
		return err
	}
	endTime, err := timePtrValue(q.EndTime)
	if err != nil {
		return err
	}

	_, err = db.Exec(
		"UPDATE quizzes SET title = ?, type = ?, start_time = ?, end_time = ?, description = ?, max_point = ?, status = ? WHERE id = ?",
		strPtrValue(q.Title), strPtrValue(q.Type), startTime, endTime, strPtrValue(q.Description), intPtrValue(q.MaxPoint), strPtrValue(q.Status), q.ID,
	)
	return err
}

// DeleteQuiz hapus quiz permanen berdasarkan ID, dipakai handler DELETE /api/quizzes/{id}.
func DeleteQuiz(db *sql.DB, id int64) error {
	_, err := db.Exec("DELETE FROM quizzes WHERE id = ?", id)
	return err
}

// nullableString konversi sql.NullString ke *string, dipakai biar kolom NULL di DB jadi null di JSON.
func nullableString(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

// nullableTime konversi sql.NullTime ke *string dengan format dateTimeLayout, dipakai biar FE
// terima string datetime yang konsisten (atau null kalau kolomnya emang kosong).
func nullableTime(v sql.NullTime) *string {
	if !v.Valid {
		return nil
	}
	s := v.Time.Format(dateTimeLayout)
	return &s
}

// strPtrValue konversi *string ke interface{} buat argumen query SQL, nil pointer jadi SQL NULL.
func strPtrValue(v *string) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

// intPtrValue sama kayak strPtrValue tapi buat kolom angka (max_point).
func intPtrValue(v *int) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

// timePtrValue parse string datetime dari FE (format "YYYY-MM-DDTHH:mm" dari <input type="datetime-local">
// atau "YYYY-MM-DDTHH:mm:ss") jadi time.Time buat kolom DATETIME. nil/"" -> SQL NULL.
func timePtrValue(v *string) (interface{}, error) {
	if v == nil || *v == "" {
		return nil, nil
	}
	// Coba beberapa format layout karena FE bisa kirim format yang beda-beda tergantung sumbernya
	// (input datetime-local browser vs format yang udah lengkap detiknya).
	layouts := []string{dateTimeLayout, "2006-01-02T15:04", "2006-01-02 15:04:05"}
	var lastErr error
	for _, layout := range layouts {
		if t, err := time.Parse(layout, *v); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}
	return nil, lastErr
}
