package models

import (
	"database/sql"
	"fmt"
	"strings"
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

// Status "closed" cuma di-set otomatis oleh sistem (auto-close survey yang lewat end_time), bukan
// pilihan yang bisa dipilih manual lewat form admin (dropdown status FE cuma active/inactive).
const (
	QuizStatusActive   = "active"
	QuizStatusInactive = "inactive"
	QuizStatusClosed   = "closed"
)

type Quiz struct {
	ID           int64   `json:"id"`
	Title        *string `json:"title"`
	Type         *string `json:"type"`
	StartTime    *string `json:"start_time"`
	EndTime      *string `json:"end_time"`
	Description  *string `json:"description"`
	MaxPoint     *int    `json:"max_point"`
	PassingGrade *int    `json:"passing_grade"`
	// RandomQuestionCount jumlah soal yang ditampilkan secara acak per sesi responden dari total pool
	// question quiz ini. Nil/0 atau >= total question berarti fitur ini nonaktif (tampilkan semua soal).
	RandomQuestionCount *int `json:"random_question_count"`
	// LockMode aktifin anti-cheat di public form (khusus type=quiz): wajib fullscreen, keluar
	// tab/fullscreen dihitung pelanggaran & auto-submit paksa setelah 3x pelanggaran.
	LockMode      bool    `json:"lock_mode"`
	TotalQuestion int     `json:"total_question"`
	Status        *string `json:"status"`
	// DuplicatedFromID nunjuk ke quiz asal kalau quiz ini hasil "Duplicate jadi versi baru".
	DuplicatedFromID *int64 `json:"duplicated_from_id"`
	// HasSubmissions dihitung on-the-fly (bukan kolom DB) buat FE tau kapan form question harus dikunci.
	HasSubmissions bool `json:"has_submissions"`
}

type quizRowScanner interface {
	Scan(dest ...interface{}) error
}

// GetQuizByID ambil 1 quiz lengkap dengan total question. Dipakai saat butuh detail quiz per-id
// (misal generate share link atau alur publik berdasarkan token share).
func GetQuizByID(db *sql.DB, id int64) (Quiz, error) {
	row := db.QueryRow(
		"SELECT q.id, q.title, q.type, q.start_time, q.end_time, q.description, q.max_point, q.passing_grade, q.random_question_count, q.lock_mode, "+
			"(SELECT COUNT(*) FROM questions WHERE quiz_id = q.id) AS total_question, q.status, q.duplicated_from_id "+
			"FROM quizzes q WHERE q.id = ? LIMIT 1",
		id,
	)
	quiz, err := scanQuiz(row)
	if err != nil {
		return Quiz{}, err
	}
	return finalizeQuizForResponse(db, quiz, time.Now())
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
	query := "SELECT q.id, q.title, q.type, q.start_time, q.end_time, q.description, q.max_point, q.passing_grade, q.random_question_count, q.lock_mode, " +
		"(SELECT COUNT(*) FROM questions WHERE quiz_id = q.id) AS total_question, q.status, q.duplicated_from_id FROM quizzes q" + whereClause +
		fmt.Sprintf(" ORDER BY %s %s LIMIT ? OFFSET ?", sortCol, params.SortDirSQL())
	args = append(args, params.PerPage, params.Offset())

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	quizzes := []Quiz{}
	now := time.Now()
	for rows.Next() {
		q, err := scanQuiz(rows)
		if err != nil {
			return nil, 0, err
		}
		q, err = finalizeQuizForResponse(db, q, now)
		if err != nil {
			return nil, 0, err
		}
		quizzes = append(quizzes, q)
	}
	return quizzes, total, rows.Err()
}

// finalizeQuizForResponse lengkapi 1 quiz sebelum dikirim ke client: auto-close survey yang udah
// lewat end_time, lalu hitung flag HasSubmissions (dipakai FE buat kunci form question).
func finalizeQuizForResponse(db *sql.DB, q Quiz, now time.Time) (Quiz, error) {
	q, err := AutoCloseSurveyIfExpired(db, q, now)
	if err != nil {
		return Quiz{}, err
	}
	hasSubmissions, err := QuizHasSubmissions(db, q.ID)
	if err != nil {
		return Quiz{}, err
	}
	q.HasSubmissions = hasSubmissions
	return q, nil
}

// AutoCloseSurveyIfExpired tutup permanen (status="closed") survey yang statusnya masih "active"
// tapi end_time-nya udah lewat. Sengaja cuma berlaku buat type="survey" -- type="quiz" memakai
// end_time sebagai window JAM yang berulang tiap hari (lihat resolveQuizDailyWindow), jadi kalau
// ikut ditutup permanen bakal mematikan behavior recurring harian yang memang disengaja.
func AutoCloseSurveyIfExpired(db *sql.DB, q Quiz, now time.Time) (Quiz, error) {
	if stringPtrValue(q.Type) != "survey" || stringPtrValue(q.Status) != QuizStatusActive {
		return q, nil
	}
	endTime, err := parseQuizDateTime(q.EndTime)
	if err != nil || !now.After(endTime) {
		return q, nil
	}

	// Guard status='active' di WHERE biar aman kalau ada request lain yang barengan ubah status.
	if _, err := db.Exec("UPDATE quizzes SET status = ? WHERE id = ? AND status = ?", QuizStatusClosed, q.ID, QuizStatusActive); err != nil {
		return Quiz{}, err
	}
	closed := QuizStatusClosed
	q.Status = &closed
	return q, nil
}

// QuizHasSubmissions cek apakah quiz udah pernah menerima submission publik. Dipakai buat lifecycle
// lock: quiz yang udah ada submission gak boleh diubah soal-nya lagi (harus duplicate jadi versi baru).
func QuizHasSubmissions(db *sql.DB, quizID int64) (bool, error) {
	var exists int
	err := db.QueryRow("SELECT 1 FROM quiz_submissions WHERE quiz_id = ? LIMIT 1", quizID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// DuplicateQuiz bikin versi baru dari sebuah quiz: copy quiz + semua question/answer/matrix row-nya
// (independen, bukan live-link -- edit di versi baru gak ngaruh ke versi lama atau sebaliknya).
// Quiz baru selalu berstatus "inactive" (draft) biar admin sempat review sebelum di-share lagi, dan
// submission history tetap nempel di quiz lama.
func DuplicateQuiz(db *sql.DB, sourceID int64) (Quiz, error) {
	source, err := GetQuizByID(db, sourceID)
	if err != nil {
		return Quiz{}, err
	}

	newTitle := strings.TrimSpace(stringPtrValue(source.Title)) + " (Copy)"
	inactive := QuizStatusInactive
	newQuiz := Quiz{
		Title:               &newTitle,
		Type:                source.Type,
		StartTime:           source.StartTime,
		EndTime:             source.EndTime,
		Description:         source.Description,
		MaxPoint:            source.MaxPoint,
		PassingGrade:        source.PassingGrade,
		RandomQuestionCount: source.RandomQuestionCount,
		LockMode:            source.LockMode,
		Status:              &inactive,
		DuplicatedFromID:    &sourceID,
	}
	newID, err := CreateQuiz(db, newQuiz)
	if err != nil {
		return Quiz{}, err
	}

	questions, err := ListQuestionsByQuiz(db, sourceID)
	if err != nil {
		return Quiz{}, err
	}
	for _, question := range questions {
		copied := Question{
			QuizID:     &newID,
			Question:   question.Question,
			TypeAnswer: question.TypeAnswer,
			Point:      question.Point,
			Answers:    question.Answers,
			MatrixRows: question.MatrixRows,
		}
		if _, err := CreateQuestion(db, copied); err != nil {
			return Quiz{}, err
		}
	}

	return GetQuizByID(db, newID)
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
		passingGrade         sql.NullInt64
		randomQuestionCount  sql.NullInt64
		duplicatedFromID     sql.NullInt64
	)
	if err := row.Scan(&q.ID, &title, &qType, &startTime, &endTime, &description, &maxPoint, &passingGrade, &randomQuestionCount, &q.LockMode, &q.TotalQuestion, &status, &duplicatedFromID); err != nil {
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
	if passingGrade.Valid {
		v := int(passingGrade.Int64)
		q.PassingGrade = &v
	}
	if randomQuestionCount.Valid {
		v := int(randomQuestionCount.Int64)
		q.RandomQuestionCount = &v
	}
	if duplicatedFromID.Valid {
		v := duplicatedFromID.Int64
		q.DuplicatedFromID = &v
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
		"INSERT INTO quizzes (title, type, start_time, end_time, description, max_point, passing_grade, random_question_count, lock_mode, status, duplicated_from_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		strPtrValue(q.Title), strPtrValue(q.Type), startTime, endTime, strPtrValue(q.Description), intPtrValue(q.MaxPoint), intPtrValue(q.PassingGrade), intPtrValue(q.RandomQuestionCount), q.LockMode, strPtrValue(q.Status), int64PtrValue(q.DuplicatedFromID),
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
		"UPDATE quizzes SET title = ?, type = ?, start_time = ?, end_time = ?, description = ?, max_point = ?, passing_grade = ?, random_question_count = ?, lock_mode = ?, status = ? WHERE id = ?",
		strPtrValue(q.Title), strPtrValue(q.Type), startTime, endTime, strPtrValue(q.Description), intPtrValue(q.MaxPoint), intPtrValue(q.PassingGrade), intPtrValue(q.RandomQuestionCount), q.LockMode, strPtrValue(q.Status), q.ID,
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
