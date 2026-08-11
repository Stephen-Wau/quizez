package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"quizez/backend/internal/models"
)

type publicSubmissionRequest struct {
	Email      *string                         `json:"email"`
	StartedAt  *string                         `json:"started_at"`
	AccessCode *string                         `json:"access_code"`
	Answers    []publicSubmissionAnswerRequest `json:"answers"`
}

type publicSubmissionAnswerRequest struct {
	QuestionID       *int64  `json:"question_id"`
	QuestionAnswerID *int64  `json:"question_answer_id"`
	AnswerText       *string `json:"answer_text"`
}

type quizShareResponse struct {
	QuizID     int64   `json:"quiz_id"`
	Token      *string `json:"token"`
	AccessCode *string `json:"access_code"`
}

// QuizShareHandler generate atau ambil token share publik untuk 1 quiz dari menu admin.
func QuizShareHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		quiz, err := models.GetQuizByID(db, id)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Quiz tidak ditemukan.", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "failed to generate share link", http.StatusInternalServerError)
			return
		}
		if !strings.EqualFold(strings.TrimSpace(stringValue(quiz.Status)), "active") {
			http.Error(w, "Quiz inactive tidak bisa generate link share.", http.StatusConflict)
			return
		}

		share, err := models.GetOrCreateQuizShare(db, quiz.ID)
		if err != nil {
			http.Error(w, "failed to generate share link", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(quizShareResponse{
			QuizID:     quiz.ID,
			Token:      share.Token,
			AccessCode: share.AccessCode,
		})
	}
}

// PublicQuizHandler expose detail quiz/survey publik berdasarkan token share tanpa perlu login.
func PublicQuizHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		token := strings.TrimSpace(r.PathValue("token"))
		if token == "" {
			http.Error(w, "invalid token", http.StatusBadRequest)
			return
		}

		accessCode := normalizeStrPtr(r.URL.Query().Get("code"))
		quiz, err := models.GetPublicQuizByToken(db, token, accessCode, time.Now())
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Link tidak ditemukan.", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "failed to load public quiz", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(quiz)
	}
}

// PublicQuizSubmitHandler terima jawaban publik, validasi period + email quiz, lalu simpan submit
// beserta score (kalau tipe quiz) ke database.
func PublicQuizSubmitHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		token := strings.TrimSpace(r.PathValue("token"))
		if token == "" {
			http.Error(w, "invalid token", http.StatusBadRequest)
			return
		}

		var req publicSubmissionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		now := time.Now()
		publicQuiz, err := models.GetPublicQuizByToken(db, token, req.AccessCode, now)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Link tidak ditemukan.", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "failed to submit quiz", http.StatusInternalServerError)
			return
		}

		if msg, status := validatePublicQuizAvailability(publicQuiz); msg != "" {
			http.Error(w, msg, status)
			return
		}
		if publicQuiz.AccessCodeRequired && !publicQuiz.AccessGranted {
			http.Error(w, "PIN akses form tidak valid.", http.StatusForbidden)
			return
		}

		email, msg := validatePublicSubmissionEmail(publicQuiz, req.Email)
		if msg != "" {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		startedAt, msg := validatePublicSubmissionStartedAt(req.StartedAt)
		if msg != "" {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		if publicQuiz.Type != nil && *publicQuiz.Type == "quiz" && email != nil {
			exists, err := models.HasSubmittedEmail(db, publicQuiz.ID, *email)
			if err != nil {
				http.Error(w, "failed to submit quiz", http.StatusInternalServerError)
				return
			}
			if exists {
				http.Error(w, "Email ini sudah pernah mengirim quiz ini.", http.StatusConflict)
				return
			}
		}

		inputs := make([]models.PublicSubmissionAnswerInput, 0, len(req.Answers))
		for _, answer := range req.Answers {
			if answer.QuestionID == nil {
				http.Error(w, "Question wajib dipilih.", http.StatusBadRequest)
				return
			}
			inputs = append(inputs, models.PublicSubmissionAnswerInput{
				QuestionID:       *answer.QuestionID,
				QuestionAnswerID: answer.QuestionAnswerID,
				AnswerText:       normalizeStr(answer.AnswerText),
			})
		}

		quiz := models.Quiz{
			ID:           publicQuiz.ID,
			Title:        publicQuiz.Title,
			Type:         publicQuiz.Type,
			StartTime:    publicQuiz.StartTime,
			EndTime:      publicQuiz.EndTime,
			Description:  publicQuiz.Description,
			MaxPoint:     publicQuiz.MaxPoint,
			PassingGrade: publicQuiz.PassingGrade,
			Status:       publicQuiz.Status,
		}

		result, err := models.SavePublicSubmission(db, quiz, email, inputs, startedAt, now)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				http.Error(w, "Email ini sudah pernah mengirim quiz ini.", http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(result)
	}
}

// validatePublicSubmissionStartedAt parse waktu mulai quiz dari client agar backend bisa menyimpan
// durasi/riwayat yang lebih akurat pada detail submission respondent.
func validatePublicSubmissionStartedAt(value *string) (*time.Time, string) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, ""
	}

	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*value))
	if err != nil {
		return nil, "Format started_at tidak valid."
	}
	return &parsed, ""
}

// validatePublicQuizAvailability jaga link publik cuma bisa dikerjakan saat period dan status-nya aktif.
func validatePublicQuizAvailability(quiz models.PublicQuiz) (string, int) {
	switch quiz.State {
	case models.PublicQuizStateInactive:
		return "Form ini sedang tidak aktif.", http.StatusGone
	case models.PublicQuizStateUpcoming:
		return "Form ini belum bisa dikerjakan.", http.StatusConflict
	case models.PublicQuizStateExpired:
		return "Form ini sudah expired.", http.StatusGone
	default:
		return "", 0
	}
}

// validatePublicSubmissionEmail mewajibkan email valid untuk tipe quiz, sementara survey boleh
// kosong karena user bisa submit berkali-kali tanpa identitas.
func validatePublicSubmissionEmail(quiz models.PublicQuiz, email *string) (*string, string) {
	if quiz.Type == nil || *quiz.Type != "quiz" {
		return nil, ""
	}
	if email == nil || strings.TrimSpace(*email) == "" {
		return nil, "Email wajib diisi sebelum mulai quiz."
	}

	normalized := strings.ToLower(strings.TrimSpace(*email))
	if _, err := mail.ParseAddress(normalized); err != nil {
		return nil, "Format email tidak valid."
	}
	return &normalized, ""
}

// stringValue bantu baca pointer string handler tanpa perlu ngecek nil berulang.
func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func normalizeStrPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
