package models

import (
	"database/sql"
	"fmt"
	"strings"

	"quizez/backend/internal/listquery"
)

// whitelist FE `sort_by` (col.prop) -> kolom SQL asli. WAJIB whitelist, jangan interpolate raw.
var questionBankSortColumns = map[string]string{
	"question":    "question",
	"type_answer": "type_answer",
}

// QuestionBankItem 1 soal reusable di bank, independen dari quiz manapun. Answers ikut struct
// QuestionAnswer yang sama dipakai question per-quiz biar konsisten dan gampang di-copy.
type QuestionBankItem struct {
	ID         int64   `json:"id"`
	Question   *string `json:"question"`
	TypeAnswer *string `json:"type_answer"`
	Point      *int    `json:"point"`
	// Tags freeform, disimpan gabungan dipisah ";" di DB tapi diexpose sebagai slice ke FE.
	Tags    []string         `json:"tags"`
	Answers []QuestionAnswer `json:"answers"`
}

// ListQuestionBank ambil daftar soal bank buat DataTable FE: search di teks soal, filter tag
// (substring match di kolom tags), sort kolom whitelist, dan pagination.
func ListQuestionBank(db *sql.DB, params listquery.Params, tagFilter string) ([]QuestionBankItem, int, error) {
	whereClauses := []string{}
	args := []interface{}{}

	if params.SearchWord != "" {
		whereClauses = append(whereClauses, "question LIKE ?")
		args = append(args, "%"+params.SearchWord+"%")
	}
	// Filter tag dicocokkan sebagai substring diapit ";" biar gak ke-match tag lain yang cuma
	// kebetulan mengandung kata yang sama (ex: tag "hr" gak boleh ke-match "hrd").
	if tagFilter != "" {
		whereClauses = append(whereClauses, "CONCAT(';', tags, ';') LIKE ?")
		args = append(args, "%;"+tagFilter+";%")
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM question_bank"+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	sortCol := params.SortColumn(questionBankSortColumns, "id")
	query := "SELECT id, question, type_answer, point, tags FROM question_bank" + whereSQL +
		fmt.Sprintf(" ORDER BY %s %s LIMIT ? OFFSET ?", sortCol, params.SortDirSQL())
	args = append(args, params.PerPage, params.Offset())

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []QuestionBankItem{}
	for rows.Next() {
		item, err := scanQuestionBankItem(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	for i := range items {
		answers, err := listQuestionBankAnswers(db, items[i].ID)
		if err != nil {
			return nil, 0, err
		}
		items[i].Answers = answers
	}
	return items, total, nil
}

// ListQuestionBankTags ambil semua tag unik yang pernah dipakai, buat dropdown/autocomplete filter FE.
func ListQuestionBankTags(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT tags FROM question_bank WHERE tags IS NOT NULL AND tags <> ''")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := map[string]bool{}
	tags := []string{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		for _, tag := range decodeTags(raw) {
			if !seen[tag] {
				seen[tag] = true
				tags = append(tags, tag)
			}
		}
	}
	return tags, rows.Err()
}

// GetQuestionBankByID ambil 1 soal bank lengkap dengan answer-nya, dipakai form edit & copy-to-quiz.
func GetQuestionBankByID(db *sql.DB, id int64) (QuestionBankItem, error) {
	row := db.QueryRow("SELECT id, question, type_answer, point, tags FROM question_bank WHERE id = ? LIMIT 1", id)
	item, err := scanQuestionBankItem(row)
	if err != nil {
		return QuestionBankItem{}, err
	}
	answers, err := listQuestionBankAnswers(db, item.ID)
	if err != nil {
		return QuestionBankItem{}, err
	}
	item.Answers = answers
	return item, nil
}

// GetQuestionBankByIDs ambil banyak soal bank sekaligus (dipakai copy-to-quiz), urutan hasil
// ngikutin urutan ids yang diminta biar FE bisa nampilin sesuai pilihan checkbox-nya.
func GetQuestionBankByIDs(db *sql.DB, ids []int64) ([]QuestionBankItem, error) {
	items := make([]QuestionBankItem, 0, len(ids))
	for _, id := range ids {
		item, err := GetQuestionBankByID(db, id)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// scanQuestionBankItem scan 1 baris question_bank ke struct, kolom nullable di-scan ke sql.Null*
// dulu baru dikonversi ke pointer biar JSON-nya bisa null.
func scanQuestionBankItem(row questionRowScanner) (QuestionBankItem, error) {
	var (
		item            QuestionBankItem
		question, qType sql.NullString
		point           sql.NullInt64
		tags            sql.NullString
	)
	if err := row.Scan(&item.ID, &question, &qType, &point, &tags); err != nil {
		return QuestionBankItem{}, err
	}
	item.Question = nullableString(question)
	item.TypeAnswer = nullableString(qType)
	if point.Valid {
		v := int(point.Int64)
		item.Point = &v
	}
	item.Tags = decodeTags(tags.String)
	return item, nil
}

// listQuestionBankAnswers ambil semua opsi jawaban milik 1 soal bank, urut sesuai insert.
func listQuestionBankAnswers(db *sql.DB, questionBankID int64) ([]QuestionAnswer, error) {
	rows, err := db.Query(
		"SELECT id, question_bank_id, label, value FROM question_bank_answers WHERE question_bank_id = ? ORDER BY id ASC",
		questionBankID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	answers := []QuestionAnswer{}
	for rows.Next() {
		var (
			answer       QuestionAnswer
			label, value sql.NullString
			bankID       sql.NullInt64
		)
		if err := rows.Scan(&answer.ID, &bankID, &label, &value); err != nil {
			return nil, err
		}
		answer.Label = nullableString(label)
		answer.Value = nullableString(value)
		answers = append(answers, answer)
	}
	return answers, rows.Err()
}

// CreateQuestionBankItem insert 1 soal bank baru beserta answer-nya dalam 1 transaction.
func CreateQuestionBankItem(db *sql.DB, item QuestionBankItem) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		"INSERT INTO question_bank (question, type_answer, point, tags) VALUES (?, ?, ?, ?)",
		strPtrValue(item.Question), strPtrValue(item.TypeAnswer), intPtrValue(item.Point), encodeTags(item.Tags),
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := insertQuestionBankAnswers(tx, id, item.Answers); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

// UpdateQuestionBankItem overwrite data soal bank + rebuild semua answer-nya.
func UpdateQuestionBankItem(db *sql.DB, item QuestionBankItem) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		"UPDATE question_bank SET question = ?, type_answer = ?, point = ?, tags = ? WHERE id = ?",
		strPtrValue(item.Question), strPtrValue(item.TypeAnswer), intPtrValue(item.Point), encodeTags(item.Tags), item.ID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM question_bank_answers WHERE question_bank_id = ?", item.ID); err != nil {
		return err
	}
	if err := insertQuestionBankAnswers(tx, item.ID, item.Answers); err != nil {
		return err
	}
	return tx.Commit()
}

// DeleteQuestionBankItem hapus 1 soal bank. Answer ikut kehapus via FK cascade; question yang
// pernah di-copy dari sini TETAP ada di quiz masing-masing (question_bank_id-nya jadi NULL).
func DeleteQuestionBankItem(db *sql.DB, id int64) error {
	_, err := db.Exec("DELETE FROM question_bank WHERE id = ?", id)
	return err
}

// insertQuestionBankAnswers bulk insert daftar answer milik 1 soal bank dalam transaction aktif.
func insertQuestionBankAnswers(tx *sql.Tx, questionBankID int64, answers []QuestionAnswer) error {
	if len(answers) == 0 {
		return nil
	}
	stmt, err := tx.Prepare("INSERT INTO question_bank_answers (question_bank_id, label, value) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, answer := range answers {
		if _, err := stmt.Exec(questionBankID, strPtrValue(answer.Label), strPtrValue(answer.Value)); err != nil {
			return err
		}
	}
	return nil
}

// CopyQuestionBankItemsToQuiz duplicate sejumlah soal bank jadi question baru milik quiz tujuan
// (copy, bukan live-link) -- edit question hasil copy gak akan ngaruh balik ke bank atau quiz lain
// yang pernah copy soal yang sama. question_bank_id ditandain buat pelacakan asal soal saja.
func CopyQuestionBankItemsToQuiz(db *sql.DB, quizID int64, items []QuestionBankItem) ([]Question, error) {
	created := make([]Question, 0, len(items))
	for _, item := range items {
		bankID := item.ID
		q := Question{
			QuizID:     &quizID,
			Question:   item.Question,
			TypeAnswer: item.TypeAnswer,
			Point:      item.Point,
			Answers:    item.Answers,
		}
		id, err := createQuestionFromBank(db, q, bankID)
		if err != nil {
			return nil, err
		}
		q.ID = id
		created = append(created, q)
	}
	return created, nil
}

// createQuestionFromBank sama seperti CreateQuestion tapi ikut nyimpen question_bank_id asal-nya.
func createQuestionFromBank(db *sql.DB, q Question, bankID int64) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		"INSERT INTO questions (quiz_id, question_bank_id, question, type_answer, point) VALUES (?, ?, ?, ?, ?)",
		int64PtrValue(q.QuizID), bankID, strPtrValue(q.Question), strPtrValue(q.TypeAnswer), intPtrValue(q.Point),
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
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return questionID, nil
}

// encodeTags gabungin slice tag jadi 1 string dipisah ";", trim tiap tag & buang yang kosong.
func encodeTags(tags []string) interface{} {
	cleaned := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	if len(cleaned) == 0 {
		return nil
	}
	return strings.Join(cleaned, ";")
}

// decodeTags parse balik string tags dari DB jadi slice, string kosong/invalid jadi slice kosong.
func decodeTags(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []string{}
	}
	parts := strings.Split(trimmed, ";")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p != "" {
			tags = append(tags, p)
		}
	}
	return tags
}
