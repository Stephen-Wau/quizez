package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"quizez/backend/internal/listquery"
	"quizez/backend/internal/models"
	"quizez/backend/internal/response"
)

type quizRequest struct {
	Title               *string `json:"title"`
	Type                *string `json:"type"`
	StartTime           *string `json:"start_time"`
	EndTime             *string `json:"end_time"`
	Description         *string `json:"description"`
	MaxPoint            *int    `json:"max_point"`
	PassingGrade        *int    `json:"passing_grade"`
	RandomQuestionCount *int    `json:"random_question_count"`
	LockMode            bool    `json:"lock_mode"`
	MaxAttempts         *int    `json:"max_attempts"`
	RetakeScorePolicy   *string `json:"retake_score_policy"`
	Language            *string `json:"language"`
	Status              *string `json:"status"`
}

// GET (list) & POST (create) at /api/quizzes
func QuizzesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listQuizzes(w, r, db)
		case http.MethodPost:
			createQuiz(w, r, db)
		default:
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

// PUT (update) & DELETE at /api/quizzes/{id}
func QuizHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid id")
			return
		}
		switch r.Method {
		case http.MethodPut:
			updateQuiz(w, r, db, id)
		case http.MethodDelete:
			deleteQuiz(w, r, db, id)
		default:
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

// listQuizzes handle GET /api/quizzes: parse query param search/sort/pagination lalu balikin
// list quiz dalam format standar listquery (data + meta).
func listQuizzes(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	params := listquery.Parse(r)
	quizzes, total, err := models.ListQuizzes(db, params)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to load quizzes")
		return
	}
	response.Paginated(w, quizzes, listquery.BuildMeta(params, total))
}

// Kolom DB tetap nullable, tapi API mewajibkan field-field ini diisi buat quiz yang valid dipakai.
// validateQuizRequest cek business rule form quiz sebelum disimpan, return pesan error pertama
// yang ketemu (kosong berarti valid semua). Dipakai bareng sama createQuiz & updateQuiz.
func validateQuizRequest(req quizRequest) string {
	if req.Title == nil || strings.TrimSpace(*req.Title) == "" {
		return "Title wajib diisi."
	}
	// Tipe quiz dibatasi cuma 2 pilihan valid, selain itu ditolak.
	if req.Type == nil || (*req.Type != "quiz" && *req.Type != "survey") {
		return "Tipe harus quiz atau survey."
	}
	if req.StartTime == nil || strings.TrimSpace(*req.StartTime) == "" {
		return "Waktu mulai wajib diisi."
	}
	if req.EndTime == nil || strings.TrimSpace(*req.EndTime) == "" {
		return "Waktu selesai wajib diisi."
	}
	// Perbandingan string di sini valid karena format datetime-nya udah ISO 8601 (lexicographic
	// order sama dengan urutan waktu asli).
	if *req.StartTime > *req.EndTime {
		return "Waktu selesai tidak boleh sebelum waktu mulai."
	}
	if req.Status == nil || (*req.Status != "active" && *req.Status != "inactive") {
		return "Status harus active atau inactive."
	}
	// Max point cuma wajib buat tipe quiz, survey gak butuh nilai maksimal.
	if *req.Type == "quiz" && req.MaxPoint == nil {
		return "Max point wajib diisi untuk tipe quiz."
	}
	// Passing grade opsional untuk survey, tapi untuk quiz dipakai sebagai threshold lulus/tidak lulus
	// sehingga harus konsisten dan tidak boleh lebih besar dari max point.
	if *req.Type == "quiz" && req.PassingGrade == nil {
		return "Passing grade wajib diisi untuk tipe quiz."
	}
	if req.MaxPoint != nil && *req.MaxPoint < 0 {
		return "Max point tidak boleh negatif."
	}
	if req.PassingGrade != nil && *req.PassingGrade < 0 {
		return "Passing grade tidak boleh negatif."
	}
	if *req.Type == "quiz" && req.MaxPoint != nil && req.PassingGrade != nil && *req.PassingGrade > *req.MaxPoint {
		return "Passing grade tidak boleh melebihi max point."
	}
	// Random question count opsional, tapi kalau diisi harus angka positif (0/negatif gak masuk akal).
	if req.RandomQuestionCount != nil && *req.RandomQuestionCount <= 0 {
		return "Jumlah soal random harus lebih dari 0."
	}
	// Retake cuma masuk akal buat quiz (survey emang udah bebas diisi berkali-kali dari awal).
	if req.MaxAttempts != nil {
		if *req.MaxAttempts < 1 {
			return "Max attempts minimal 1."
		}
		if req.Type != nil && *req.Type != "quiz" {
			return "Max attempts cuma berlaku untuk tipe quiz."
		}
	}
	// Retake score policy wajib jelas ("best"/"latest") begitu retake diaktifkan (max_attempts > 1),
	// biar leaderboard/hasil akhir tau attempt mana yang dipakai.
	if req.MaxAttempts != nil && *req.MaxAttempts > 1 {
		if req.RetakeScorePolicy == nil || (*req.RetakeScorePolicy != "best" && *req.RetakeScorePolicy != "latest") {
			return "Pilih skor yang dipakai (Terbaik/Terakhir) untuk quiz yang bisa diulang."
		}
	}
	if req.Language != nil && *req.Language != "id" && *req.Language != "en" {
		return "Bahasa form harus Indonesia atau English."
	}
	return ""
}

// createQuiz handle POST /api/quizzes: decode body, validasi, lalu insert ke DB.
func createQuiz(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req quizRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if msg := validateQuizRequest(req); msg != "" {
		response.Error(w, http.StatusBadRequest, msg)
		return
	}

	isQuizType := req.Type != nil && *req.Type == "quiz"
	q := models.Quiz{
		Title: normalizeStr(req.Title), Type: normalizeStr(req.Type),
		StartTime: req.StartTime, EndTime: req.EndTime,
		Description: normalizeStr(req.Description), MaxPoint: req.MaxPoint, PassingGrade: req.PassingGrade,
		RandomQuestionCount: req.RandomQuestionCount, LockMode: isQuizType && req.LockMode,
		Status:   normalizeStr(req.Status),
		Language: normalizeStr(req.Language),
	}
	// Retake (max_attempts/retake_score_policy) cuma berlaku buat quiz -- survey emang udah bebas
	// diisi berkali-kali dari awal, jadi dipaksa kosong biar gak nyangkut kalau type-nya diganti.
	if isQuizType {
		q.MaxAttempts = req.MaxAttempts
		q.RetakeScorePolicy = normalizeStr(req.RetakeScorePolicy)
	}
	id, err := models.CreateQuiz(db, q)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to save quiz")
		return
	}
	q.ID = id
	writeAuditLog(r, db, "quiz.create", "quiz", &q.ID, "Membuat quiz atau survey baru.")
	response.JSON(w, http.StatusCreated, q)
}

// updateQuiz handle PUT /api/quizzes/{id}: decode body, validasi, lalu overwrite quiz existing.
func updateQuiz(w http.ResponseWriter, r *http.Request, db *sql.DB, id int64) {
	var req quizRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if msg := validateQuizRequest(req); msg != "" {
		response.Error(w, http.StatusBadRequest, msg)
		return
	}

	isQuizType := req.Type != nil && *req.Type == "quiz"
	q := models.Quiz{
		ID: id, Title: normalizeStr(req.Title), Type: normalizeStr(req.Type),
		StartTime: req.StartTime, EndTime: req.EndTime,
		Description: normalizeStr(req.Description), MaxPoint: req.MaxPoint, PassingGrade: req.PassingGrade,
		RandomQuestionCount: req.RandomQuestionCount, LockMode: isQuizType && req.LockMode,
		Status:   normalizeStr(req.Status),
		Language: normalizeStr(req.Language),
	}
	if isQuizType {
		q.MaxAttempts = req.MaxAttempts
		q.RetakeScorePolicy = normalizeStr(req.RetakeScorePolicy)
	}
	if err := models.UpdateQuiz(db, q); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to update quiz")
		return
	}
	writeAuditLog(r, db, "quiz.update", "quiz", &q.ID, "Memperbarui quiz atau survey.")
	response.JSON(w, http.StatusOK, q)
}

// deleteQuiz handle DELETE /api/quizzes/{id}.
func deleteQuiz(w http.ResponseWriter, r *http.Request, db *sql.DB, id int64) {
	if err := models.DeleteQuiz(db, id); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to delete quiz")
		return
	}
	writeAuditLog(r, db, "quiz.delete", "quiz", &id, "Menghapus quiz atau survey.")
	w.WriteHeader(http.StatusNoContent)
}

// QuizDuplicateHandler POST /api/quizzes/{id}/duplicate -> versioning: copy quiz + semua soal-nya
// jadi quiz baru (draft/inactive), dipakai admin buat ubah soal quiz yang udah dikunci karena
// sudah ada submission.
func QuizDuplicateHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid id")
			return
		}

		exists, err := models.QuizExists(db, id)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to duplicate quiz")
			return
		}
		if !exists {
			response.Error(w, http.StatusNotFound, "Quiz tidak ditemukan.")
			return
		}

		duplicated, err := models.DuplicateQuiz(db, id)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to duplicate quiz")
			return
		}
		writeAuditLog(r, db, "quiz.duplicate", "quiz", &duplicated.ID, "Menduplikasi quiz menjadi versi baru.")
		response.JSON(w, http.StatusCreated, duplicated)
	}
}

// normalizeStr ubah string kosong/whitespace jadi nil, biar tersimpan sebagai SQL NULL bukan
// string kosong di DB.
func normalizeStr(v *string) *string {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	return v
}
