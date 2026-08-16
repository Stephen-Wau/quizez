package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"quizez/backend/internal/models"
)

const (
	questionTypeMultipleChoice = "multiple_choice"
	questionTypeDropdown       = "dropdown"
	questionTypeCheckbox       = "checkbox"
	questionTypeMatrix         = "matrix"
	questionTypeRating         = "rating"
	questionTypeFreeText       = "free_text"
)

type questionRequest struct {
	QuizID     *int64                  `json:"quiz_id"`
	Question   *string                 `json:"question"`
	TypeAnswer *string                 `json:"type_answer"`
	Point      *int                    `json:"point"`
	Answers    []questionAnswerRequest `json:"answers"`
	// MatrixRows cuma dipakai buat type_answer="matrix".
	MatrixRows []questionMatrixRowRequest `json:"matrix_rows"`
}

type questionAnswerRequest struct {
	Label *string `json:"label"`
	Value *string `json:"value"`
}

type questionMatrixRowRequest struct {
	RowLabel *string `json:"row_label"`
}

// GET (list) & POST (create) at /api/questions
func QuestionsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listQuestions(w, r, db)
		case http.MethodPost:
			createQuestion(w, r, db)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// PUT & DELETE at /api/questions/{id}
func QuestionHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodPut:
			updateQuestion(w, r, db, id)
		case http.MethodDelete:
			deleteQuestion(w, r, db, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// listQuestions handle GET /api/questions?quiz_id=... : validasi quiz target dulu, baru balikin
// semua question milik quiz tersebut lengkap dengan answer-nya.
func listQuestions(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	quizID, err := strconv.ParseInt(r.URL.Query().Get("quiz_id"), 10, 64)
	if err != nil || quizID <= 0 {
		http.Error(w, "quiz_id wajib diisi.", http.StatusBadRequest)
		return
	}

	exists, err := models.QuizExists(db, quizID)
	if err != nil {
		http.Error(w, "failed to load questions", http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "Quiz tidak ditemukan.", http.StatusNotFound)
		return
	}

	questions, err := models.ListQuestionsByQuiz(db, quizID)
	if err != nil {
		http.Error(w, "failed to load questions", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(questions)
}

// validateQuestionRequest cek business rule utama form question: type answer valid, aturan point,
// dan bentuk answer yang wajib beda tergantung jenis pertanyaannya.
func validateQuestionRequest(req questionRequest) string {
	if req.QuizID == nil || *req.QuizID <= 0 {
		return "Quiz wajib dipilih."
	}
	if req.Question == nil || strings.TrimSpace(*req.Question) == "" {
		return "Question wajib diisi."
	}
	if req.TypeAnswer == nil {
		return "Type answer wajib dipilih."
	}
	switch *req.TypeAnswer {
	case questionTypeMultipleChoice, questionTypeDropdown, questionTypeCheckbox, questionTypeMatrix, questionTypeRating, questionTypeFreeText:
	default:
		return "Type answer harus pilihan ganda, dropdown, checkbox, matrix, rating, atau free text."
	}

	if req.Point != nil && *req.Point <= 0 {
		return "Point harus lebih dari 0."
	}

	switch *req.TypeAnswer {
	case questionTypeFreeText:
		// Free text gak punya opsi jawaban preset; user jawab bebas saat mengerjakan quiz.
		if len(req.Answers) > 0 {
			return "Free text tidak boleh punya jawaban pilihan."
		}
	case questionTypeMultipleChoice, questionTypeDropdown, questionTypeCheckbox:
		// Pilihan ganda/dropdown/checkbox sama-sama wajib minimal 2 opsi, teks opsi gak boleh
		// kosong, dan minimal 1 jawaban true (boleh lebih dari 1 — ex: "sebutkan salah satu
		// bilangan prima" buat pilihan ganda, atau checkbox yang emang butuh banyak jawaban benar).
		if msg := validateOptionAnswers(req.Answers); msg != "" {
			return msg
		}
	case questionTypeMatrix:
		// Matrix butuh minimal 1 baris pernyataan + minimal 2 kolom skala (kolom reuse field Answers).
		if len(req.MatrixRows) < 1 {
			return "Matrix minimal harus punya 1 baris pernyataan."
		}
		for _, row := range req.MatrixRows {
			if row.RowLabel == nil || strings.TrimSpace(*row.RowLabel) == "" {
				return "Teks baris matrix wajib diisi."
			}
		}
		if len(req.Answers) < 2 {
			return "Matrix minimal harus punya 2 kolom skala."
		}
		for _, answer := range req.Answers {
			if answer.Label == nil || strings.TrimSpace(*answer.Label) == "" {
				return "Teks kolom skala matrix wajib diisi."
			}
		}
	case questionTypeRating:
		// Rating disimpan sebagai rentang angka berurutan (1..N), jadi value harus numerik,
		// unik, dan tidak boleh lompat.
		if len(req.Answers) < 2 {
			return "Rating wajib punya minimal rentang 1 sampai 2."
		}
		seen := map[int]bool{}
		maxRating := 0
		for _, answer := range req.Answers {
			if answer.Label == nil || strings.TrimSpace(*answer.Label) == "" || answer.Value == nil || strings.TrimSpace(*answer.Value) == "" {
				return "Rentang rating wajib diisi."
			}
			n, err := strconv.Atoi(strings.TrimSpace(*answer.Value))
			if err != nil || n < 1 || n > 10 {
				return "Jawaban rating harus angka 1 sampai 10."
			}
			if seen[n] {
				return "Jawaban rating tidak boleh duplikat."
			}
			seen[n] = true
			if n > maxRating {
				maxRating = n
			}
		}
		for i := 1; i <= maxRating; i++ {
			if !seen[i] {
				return "Rentang rating harus berurutan mulai dari 1."
			}
		}
	}

	return ""
}

// validateOptionAnswers cek aturan bersama buat question yang jawabannya berupa daftar opsi
// true/false: multiple_choice, dropdown, dan checkbox. Minimal 2 opsi, teks gak boleh kosong,
// value harus true/false, dan minimal 1 opsi true (boleh lebih dari 1).
func validateOptionAnswers(answers []questionAnswerRequest) string {
	if len(answers) < 2 {
		return "Minimal harus punya 2 jawaban."
	}
	trueCount := 0
	for _, answer := range answers {
		if answer.Label == nil || strings.TrimSpace(*answer.Label) == "" {
			return "Teks jawaban wajib diisi."
		}
		if answer.Value == nil || strings.TrimSpace(*answer.Value) == "" {
			return "Semua jawaban wajib diisi."
		}
		value := strings.ToLower(strings.TrimSpace(*answer.Value))
		if value != "true" && value != "false" {
			return "Jawaban harus true atau false."
		}
		if value == "true" {
			trueCount++
		}
	}
	if trueCount == 0 {
		return "Harus punya minimal satu jawaban true."
	}
	return ""
}

// validateQuizNotLocked cek lifecycle lock: quiz yang udah punya submission gak boleh diubah lagi
// soal-nya (biar data submission/analytics history gak korup) -- admin harus duplicate ke versi baru.
func validateQuizNotLocked(db *sql.DB, quizID int64) (string, int) {
	hasSubmissions, err := models.QuizHasSubmissions(db, quizID)
	if err != nil {
		return "failed to validate quiz", http.StatusInternalServerError
	}
	if hasSubmissions {
		return "Quiz ini sudah punya submission, soal tidak bisa diubah. Duplicate quiz jadi versi baru kalau mau ubah soal.", http.StatusConflict
	}
	return "", 0
}

// validateQuestionPointLimit jaga total point semua question dalam 1 quiz supaya gak melebihi
// max_point quiz induknya. Saat update, point question yang sedang diedit dikeluarin dulu dari sum.
func validateQuestionPointLimit(db *sql.DB, req questionRequest, excludeQuestionID *int64) string {
	if req.Point == nil || req.QuizID == nil {
		return ""
	}

	maxPoint, usedPoint, err := models.QuizPointBudget(db, *req.QuizID, excludeQuestionID)
	if err != nil || maxPoint == nil {
		return ""
	}
	if usedPoint+*req.Point > *maxPoint {
		return "Total point question melebihi max point quiz."
	}
	return ""
}

// createQuestion handle POST /api/questions: decode body, validasi rule form + limit point quiz,
// lalu simpan question beserta daftar answer-nya.
func createQuestion(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req questionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if msg := validateQuestionRequest(req); msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	exists, err := models.QuizExists(db, *req.QuizID)
	if err != nil {
		http.Error(w, "failed to save question", http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "Quiz tidak ditemukan.", http.StatusBadRequest)
		return
	}
	if msg, status := validateQuizNotLocked(db, *req.QuizID); msg != "" {
		http.Error(w, msg, status)
		return
	}
	if msg := validateQuestionPointLimit(db, req, nil); msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	q := mapQuestionRequestToModel(req)
	id, err := models.CreateQuestion(db, q)
	if err != nil {
		http.Error(w, "failed to save question", http.StatusInternalServerError)
		return
	}
	q.ID = id
	writeAuditLog(r, db, "question.create", "question", &q.ID, "Membuat soal quiz baru.")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(q)
}

// updateQuestion handle PUT /api/questions/{id}: validasi payload baru, lalu overwrite question
// existing beserta seluruh answer-nya.
func updateQuestion(w http.ResponseWriter, r *http.Request, db *sql.DB, id int64) {
	var req questionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if msg := validateQuestionRequest(req); msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	exists, err := models.QuizExists(db, *req.QuizID)
	if err != nil {
		http.Error(w, "failed to update question", http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "Quiz tidak ditemukan.", http.StatusBadRequest)
		return
	}
	if msg, status := validateQuizNotLocked(db, *req.QuizID); msg != "" {
		http.Error(w, msg, status)
		return
	}
	if msg := validateQuestionPointLimit(db, req, &id); msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	q := mapQuestionRequestToModel(req)
	q.ID = id
	if err := models.UpdateQuestion(db, q); err != nil {
		http.Error(w, "failed to update question", http.StatusInternalServerError)
		return
	}
	writeAuditLog(r, db, "question.update", "question", &q.ID, "Memperbarui soal quiz.")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(q)
}

// deleteQuestion handle DELETE /api/questions/{id}. Answer ikut terhapus lewat FK cascade.
func deleteQuestion(w http.ResponseWriter, r *http.Request, db *sql.DB, id int64) {
	quizID, err := models.GetQuestionQuizID(db, id)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Question tidak ditemukan.", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to delete question", http.StatusInternalServerError)
		return
	}
	if msg, status := validateQuizNotLocked(db, quizID); msg != "" {
		http.Error(w, msg, status)
		return
	}

	if err := models.DeleteQuestion(db, id); err != nil {
		http.Error(w, "failed to delete question", http.StatusInternalServerError)
		return
	}
	writeAuditLog(r, db, "question.delete", "question", &id, "Menghapus soal quiz.")
	w.WriteHeader(http.StatusNoContent)
}

// mapQuestionRequestToModel normalisasi string kosong -> nil, lalu ubah payload API jadi model
// yang siap dipakai layer DB.
func mapQuestionRequestToModel(req questionRequest) models.Question {
	answers := make([]models.QuestionAnswer, 0, len(req.Answers))
	for _, answer := range req.Answers {
		answers = append(answers, models.QuestionAnswer{
			Label: normalizeStr(answer.Label),
			Value: normalizeStr(answer.Value),
		})
	}
	matrixRows := make([]models.QuestionMatrixRow, 0, len(req.MatrixRows))
	for _, row := range req.MatrixRows {
		matrixRows = append(matrixRows, models.QuestionMatrixRow{RowLabel: normalizeStr(row.RowLabel)})
	}
	return models.Question{
		QuizID:     req.QuizID,
		Question:   normalizeStr(req.Question),
		TypeAnswer: normalizeStr(req.TypeAnswer),
		Point:      req.Point,
		Answers:    models.NormalizeQuestionAnswers(answers),
		MatrixRows: matrixRows,
	}
}
