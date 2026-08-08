package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"quizez/backend/internal/models"
)

const (
	questionTypeMultipleChoice = "multiple_choice"
	questionTypeRating         = "rating"
	questionTypeFreeText       = "free_text"
)

type questionRequest struct {
	QuizID     *int64                  `json:"quiz_id"`
	Question   *string                 `json:"question"`
	TypeAnswer *string                 `json:"type_answer"`
	Point      *int                    `json:"point"`
	Answers    []questionAnswerRequest `json:"answers"`
}

type questionAnswerRequest struct {
	Label *string `json:"label"`
	Value *string `json:"value"`
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
			deleteQuestion(w, db, id)
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
	case questionTypeMultipleChoice, questionTypeRating, questionTypeFreeText:
	default:
		return "Type answer harus pilihan ganda, rating, atau free text."
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
	case questionTypeMultipleChoice:
		// Pilihan ganda wajib punya minimal 2 opsi, teks opsi gak boleh kosong, dan cuma boleh ada
		// satu jawaban benar (true).
		if len(req.Answers) < 2 {
			return "Pilihan ganda minimal harus punya 2 jawaban."
		}
		trueCount := 0
		for _, answer := range req.Answers {
			if answer.Label == nil || strings.TrimSpace(*answer.Label) == "" {
				return "Teks jawaban pilihan ganda wajib diisi."
			}
			if answer.Value == nil || strings.TrimSpace(*answer.Value) == "" {
				return "Semua jawaban pilihan ganda wajib diisi."
			}
			value := strings.ToLower(strings.TrimSpace(*answer.Value))
			if value != "true" && value != "false" {
				return "Jawaban pilihan ganda harus true atau false."
			}
			if value == "true" {
				trueCount++
			}
		}
		if trueCount == 0 {
			return "Pilihan ganda harus punya satu jawaban true."
		}
		if trueCount > 1 {
			return "Pilihan ganda hanya boleh punya satu jawaban true."
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(q)
}

// deleteQuestion handle DELETE /api/questions/{id}. Answer ikut terhapus lewat FK cascade.
func deleteQuestion(w http.ResponseWriter, db *sql.DB, id int64) {
	if err := models.DeleteQuestion(db, id); err != nil {
		http.Error(w, "failed to delete question", http.StatusInternalServerError)
		return
	}
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
	return models.Question{
		QuizID:     req.QuizID,
		Question:   normalizeStr(req.Question),
		TypeAnswer: normalizeStr(req.TypeAnswer),
		Point:      req.Point,
		Answers:    models.NormalizeQuestionAnswers(answers),
	}
}
