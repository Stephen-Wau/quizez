package models

import (
	"database/sql"
	"fmt"
	"strings"
)

type Question struct {
	ID         int64               `json:"id"`
	QuizID     *int64              `json:"quiz_id"`
	Question   *string             `json:"question"`
	TypeAnswer *string             `json:"type_answer"`
	Point      *int                `json:"point"`
	Answers    []QuestionAnswer    `json:"answers"`
	// MatrixRows cuma keisi buat type_answer="matrix": daftar baris pernyataan yang masing-masing
	// dinilai pakai skala yang sama (Answers berperan sebagai kolom skala, ex: "Buruk".."Baik").
	MatrixRows []QuestionMatrixRow `json:"matrix_rows"`
}

type QuestionAnswer struct {
	ID         int64   `json:"id"`
	QuestionID *int64  `json:"question_id"`
	Label      *string `json:"label"`
	Value      *string `json:"value"`
}

// QuestionMatrixRow 1 baris pernyataan pada question tipe matrix (ex: "Kecepatan Layanan").
type QuestionMatrixRow struct {
	ID         int64   `json:"id"`
	QuestionID *int64  `json:"question_id"`
	RowLabel   *string `json:"row_label"`
}

type questionRowScanner interface {
	Scan(dest ...interface{}) error
}

// ListQuestionsByQuiz ambil semua question milik 1 quiz lengkap dengan daftar answer-nya.
func ListQuestionsByQuiz(db *sql.DB, quizID int64) ([]Question, error) {
	rows, err := db.Query(
		"SELECT id, quiz_id, question, type_answer, point FROM questions WHERE quiz_id = ? ORDER BY id DESC",
		quizID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	questions := []Question{}
	for rows.Next() {
		q, err := scanQuestion(rows)
		if err != nil {
			return nil, err
		}
		answers, err := listQuestionAnswers(db, q.ID)
		if err != nil {
			return nil, err
		}
		q.Answers = answers

		// Matrix rows cuma relevan buat question tipe matrix, tapi query-nya aman dipanggil buat
		// tipe lain juga (bakal balikin slice kosong karena gak ada row-nya).
		matrixRows, err := listQuestionMatrixRows(db, q.ID)
		if err != nil {
			return nil, err
		}
		q.MatrixRows = matrixRows

		questions = append(questions, q)
	}
	return questions, rows.Err()
}

// scanQuestion scan 1 row tabel questions ke struct Question, sambil konversi kolom nullable
// ke pointer biar response JSON bisa null kalau value memang kosong di DB.
func scanQuestion(row questionRowScanner) (Question, error) {
	var (
		q                Question
		quizID, point    sql.NullInt64
		question, qType  sql.NullString
	)
	if err := row.Scan(&q.ID, &quizID, &question, &qType, &point); err != nil {
		return Question{}, err
	}
	if quizID.Valid {
		v := quizID.Int64
		q.QuizID = &v
	}
	q.Question = nullableString(question)
	q.TypeAnswer = nullableString(qType)
	if point.Valid {
		v := int(point.Int64)
		q.Point = &v
	}
	return q, nil
}

// listQuestionAnswers ambil semua opsi/jawaban milik 1 question (pilihan ganda, rating, dll)
// dengan urutan insert asc supaya tampil stabil di FE.
func listQuestionAnswers(db *sql.DB, questionID int64) ([]QuestionAnswer, error) {
	rows, err := db.Query(
		"SELECT id, question_id, label, value FROM questions_answers WHERE question_id = ? ORDER BY id ASC",
		questionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	answers := []QuestionAnswer{}
	for rows.Next() {
		var (
			answer           QuestionAnswer
			label, value     sql.NullString
			questionIDNumber sql.NullInt64
		)
		if err := rows.Scan(&answer.ID, &questionIDNumber, &label, &value); err != nil {
			return nil, err
		}
		if questionIDNumber.Valid {
			v := questionIDNumber.Int64
			answer.QuestionID = &v
		}
		answer.Label = nullableString(label)
		answer.Value = nullableString(value)
		answers = append(answers, answer)
	}
	return answers, rows.Err()
}

// CreateQuestion insert question baru plus semua answer (dan matrix row kalau tipenya matrix)
// di transaction yang sama.
func CreateQuestion(db *sql.DB, q Question) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		"INSERT INTO questions (quiz_id, question, type_answer, point) VALUES (?, ?, ?, ?)",
		int64PtrValue(q.QuizID), strPtrValue(q.Question), strPtrValue(q.TypeAnswer), intPtrValue(q.Point),
	)
	if err != nil {
		return 0, err
	}
	questionID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	if err := insertQuestionAnswers(tx, questionID, q.Answers); err != nil {
		return 0, err
	}
	if err := insertQuestionMatrixRows(tx, questionID, q.MatrixRows); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return questionID, nil
}

// UpdateQuestion overwrite data question + rebuild semua answer dan matrix row-nya.
func UpdateQuestion(db *sql.DB, q Question) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		"UPDATE questions SET quiz_id = ?, question = ?, type_answer = ?, point = ? WHERE id = ?",
		int64PtrValue(q.QuizID), strPtrValue(q.Question), strPtrValue(q.TypeAnswer), intPtrValue(q.Point), q.ID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM questions_answers WHERE question_id = ?", q.ID); err != nil {
		return err
	}
	if err := insertQuestionAnswers(tx, q.ID, q.Answers); err != nil {
		return err
	}
	// Matrix row dihapus-insert-ulang kayak answer di atas. Baris lama otomatis kehapus dari
	// quiz_submission_answers via FK cascade (ON DELETE CASCADE) kalau memang sudah gak dipakai lagi.
	if _, err := tx.Exec("DELETE FROM question_matrix_rows WHERE question_id = ?", q.ID); err != nil {
		return err
	}
	if err := insertQuestionMatrixRows(tx, q.ID, q.MatrixRows); err != nil {
		return err
	}
	return tx.Commit()
}

// DeleteQuestion hapus 1 question. Answer ikut kehapus via FK cascade.
func DeleteQuestion(db *sql.DB, id int64) error {
	_, err := db.Exec("DELETE FROM questions WHERE id = ?", id)
	return err
}

// insertQuestionAnswers bulk insert daftar answer milik 1 question dalam transaction aktif.
func insertQuestionAnswers(tx *sql.Tx, questionID int64, answers []QuestionAnswer) error {
	if len(answers) == 0 {
		return nil
	}
	stmt, err := tx.Prepare("INSERT INTO questions_answers (question_id, label, value) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, answer := range answers {
		if _, err := stmt.Exec(questionID, strPtrValue(answer.Label), strPtrValue(answer.Value)); err != nil {
			return err
		}
	}
	return nil
}

// listQuestionMatrixRows ambil semua baris pernyataan milik 1 question matrix, urut sesuai insert.
func listQuestionMatrixRows(db *sql.DB, questionID int64) ([]QuestionMatrixRow, error) {
	rows, err := db.Query(
		"SELECT id, question_id, row_label FROM question_matrix_rows WHERE question_id = ? ORDER BY id ASC",
		questionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matrixRows := []QuestionMatrixRow{}
	for rows.Next() {
		var (
			row              QuestionMatrixRow
			rowLabel         sql.NullString
			questionIDNumber sql.NullInt64
		)
		if err := rows.Scan(&row.ID, &questionIDNumber, &rowLabel); err != nil {
			return nil, err
		}
		if questionIDNumber.Valid {
			v := questionIDNumber.Int64
			row.QuestionID = &v
		}
		row.RowLabel = nullableString(rowLabel)
		matrixRows = append(matrixRows, row)
	}
	return matrixRows, rows.Err()
}

// insertQuestionMatrixRows bulk insert daftar baris pernyataan milik 1 question matrix dalam
// transaction aktif. No-op buat question non-matrix (MatrixRows kosong).
func insertQuestionMatrixRows(tx *sql.Tx, questionID int64, matrixRows []QuestionMatrixRow) error {
	if len(matrixRows) == 0 {
		return nil
	}
	stmt, err := tx.Prepare("INSERT INTO question_matrix_rows (question_id, row_label) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, row := range matrixRows {
		if _, err := stmt.Exec(questionID, strPtrValue(row.RowLabel)); err != nil {
			return err
		}
	}
	return nil
}

// int64PtrValue konversi *int64 ke argumen query SQL; nil pointer jadi SQL NULL.
func int64PtrValue(v *int64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

// QuizExists dipakai validasi handler biar request question ke quiz yang gak ada ditolak lebih rapi.
func QuizExists(db *sql.DB, id int64) (bool, error) {
	var exists int
	err := db.QueryRow("SELECT 1 FROM quizzes WHERE id = ? LIMIT 1", id).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// QuizPointBudget ambil max_point quiz + total point question yang sudah terpakai.
func QuizPointBudget(db *sql.DB, quizID int64, excludeQuestionID *int64) (*int, int, error) {
	var maxPoint sql.NullInt64
	if err := db.QueryRow("SELECT max_point FROM quizzes WHERE id = ?", quizID).Scan(&maxPoint); err != nil {
		return nil, 0, err
	}

	query := "SELECT COALESCE(SUM(point), 0) FROM questions WHERE quiz_id = ?"
	args := []interface{}{quizID}
	if excludeQuestionID != nil {
		query += " AND id <> ?"
		args = append(args, *excludeQuestionID)
	}

	var usedPoint int
	if err := db.QueryRow(query, args...).Scan(&usedPoint); err != nil {
		return nil, 0, err
	}

	if !maxPoint.Valid {
		return nil, usedPoint, nil
	}
	v := int(maxPoint.Int64)
	return &v, usedPoint, nil
}

// NormalizeQuestionAnswers trim semua value jawaban sebelum disimpan/di-validate lebih lanjut.
func NormalizeQuestionAnswers(values []QuestionAnswer) []QuestionAnswer {
	normalized := make([]QuestionAnswer, 0, len(values))
	for _, answer := range values {
		var trimmedLabel *string
		if answer.Label != nil {
			v := strings.TrimSpace(*answer.Label)
			trimmedLabel = &v
		}
		var trimmedValue *string
		if answer.Value != nil {
			v := strings.TrimSpace(*answer.Value)
			trimmedValue = &v
		}
		normalized = append(normalized, QuestionAnswer{
			ID:         answer.ID,
			QuestionID: answer.QuestionID,
			Label:      trimmedLabel,
			Value:      trimmedValue,
		})
	}
	return normalized
}

// FormatRatingAnswers bangun jawaban rating 1..max dalam format string.
func FormatRatingAnswers(max int) []QuestionAnswer {
	answers := make([]QuestionAnswer, 0, max)
	for i := 1; i <= max; i++ {
		v := fmt.Sprintf("%d", i)
		answers = append(answers, QuestionAnswer{Label: &v, Value: &v})
	}
	return answers
}
