package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"quizez/backend/internal/models"
	"quizez/backend/internal/response"
)

type publicSubmissionRequest struct {
	Email *string `json:"email"`
	// Name nama respondent, wajib untuk quiz (dipakai cetak sertifikat & leaderboard).
	Name       *string `json:"name"`
	StartedAt  *string `json:"started_at"`
	AccessCode *string `json:"access_code"`
	// AttemptSeed dipakai untuk recompute subset random_question_count yang sama persis dengan
	// yang ditampilkan ke responden saat GET (lihat models.selectRandomQuestionSubset).
	AttemptSeed *string `json:"attempt_seed"`
	// DeviceFingerprint & ViolationCount dipakai fitur anti-cheat (lock_mode): dedup device per
	// quiz, dan jumlah pelanggaran tab-switch/keluar-fullscreen yang direkam FE selama sesi.
	DeviceFingerprint *string                         `json:"device_fingerprint"`
	ViolationCount    *int                            `json:"violation_count"`
	Answers           []publicSubmissionAnswerRequest `json:"answers"`
}

type publicSubmissionAnswerRequest struct {
	QuestionID       *int64  `json:"question_id"`
	QuestionAnswerID *int64  `json:"question_answer_id"`
	AnswerText       *string `json:"answer_text"`
	// SelectedAnswerIDs dipakai buat question tipe checkbox (bisa pilih lebih dari 1 opsi).
	SelectedAnswerIDs []int64 `json:"selected_answer_ids"`
	// MatrixAnswers dipakai buat question tipe matrix, 1 entri per baris pernyataan.
	MatrixAnswers []publicMatrixAnswerRequest `json:"matrix_answers"`
}

type publicMatrixAnswerRequest struct {
	RowID            int64 `json:"row_id"`
	QuestionAnswerID int64 `json:"question_answer_id"`
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
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid id")
			return
		}

		quiz, err := models.GetQuizByID(db, id)
		if errors.Is(err, sql.ErrNoRows) {
			response.Error(w, http.StatusNotFound, "Quiz tidak ditemukan.")
			return
		}
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to generate share link")
			return
		}
		if !strings.EqualFold(strings.TrimSpace(stringValue(quiz.Status)), "active") {
			response.Error(w, http.StatusConflict, "Quiz inactive tidak bisa generate link share.")
			return
		}

		share, err := models.GetOrCreateQuizShare(db, quiz.ID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to generate share link")
			return
		}
		writeAuditLog(r, db, "quiz.generate_share_link", "quiz", &quiz.ID, "Generate atau mengambil link share quiz.")

		response.JSON(w, http.StatusOK, quizShareResponse{
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
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		token := strings.TrimSpace(r.PathValue("token"))
		if token == "" {
			response.Error(w, http.StatusBadRequest, "invalid token")
			return
		}

		accessCode := normalizeStrPtr(r.URL.Query().Get("code"))
		attemptSeed := r.URL.Query().Get("attempt")
		quiz, err := models.GetPublicQuizByToken(db, token, accessCode, time.Now(), attemptSeed)
		if errors.Is(err, sql.ErrNoRows) {
			response.Error(w, http.StatusNotFound, "Link tidak ditemukan.")
			return
		}
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to load public quiz")
			return
		}

		response.JSON(w, http.StatusOK, quiz)
	}
}

// PublicQuizSubmitHandler terima jawaban publik, validasi period + email quiz, lalu simpan submit
// beserta score (kalau tipe quiz) ke database.
func PublicQuizSubmitHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		token := strings.TrimSpace(r.PathValue("token"))
		if token == "" {
			response.Error(w, http.StatusBadRequest, "invalid token")
			return
		}

		var req publicSubmissionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}

		attemptSeed := stringValue(req.AttemptSeed)
		now := time.Now()
		publicQuiz, err := models.GetPublicQuizByToken(db, token, req.AccessCode, now, attemptSeed)
		if errors.Is(err, sql.ErrNoRows) {
			response.Error(w, http.StatusNotFound, "Link tidak ditemukan.")
			return
		}
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to submit quiz")
			return
		}

		if msg, status := validatePublicQuizAvailability(publicQuiz); msg != "" {
			response.Error(w, status, msg)
			return
		}
		if publicQuiz.AccessCodeRequired && !publicQuiz.AccessGranted {
			response.Error(w, http.StatusForbidden, "PIN akses form tidak valid.")
			return
		}

		email, msg := validatePublicSubmissionEmail(publicQuiz, req.Email)
		if msg != "" {
			response.Error(w, http.StatusBadRequest, msg)
			return
		}
		name, msg := validatePublicSubmissionName(publicQuiz, req.Name)
		if msg != "" {
			response.Error(w, http.StatusBadRequest, msg)
			return
		}
		startedAt, msg := validatePublicSubmissionStartedAt(req.StartedAt)
		if msg != "" {
			response.Error(w, http.StatusBadRequest, msg)
			return
		}

		isQuizType := publicQuiz.Type != nil && *publicQuiz.Type == "quiz"
		// max_attempts nil/<=0 berarti behavior lama: cuma boleh 1x.
		maxAttempts := 1
		if publicQuiz.MaxAttempts != nil && *publicQuiz.MaxAttempts > 0 {
			maxAttempts = *publicQuiz.MaxAttempts
		}
		if isQuizType && email != nil {
			count, err := models.CountSubmissionsByEmail(db, publicQuiz.ID, *email)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, "failed to submit quiz")
				return
			}
			if count >= maxAttempts {
				response.Error(w, http.StatusConflict, retakeLimitMessage("Email ini", maxAttempts))
				return
			}
		}
		// Dedup device fingerprint (anti-cheat) cuma buat quiz -- limit-nya sama kayak email, biar 1
		// device gak bisa nge-farm attempt lebih banyak dari max_attempts pakai email yang beda-beda.
		fingerprint := stringValue(req.DeviceFingerprint)
		if isQuizType && fingerprint != "" {
			count, err := models.CountSubmissionsByFingerprint(db, publicQuiz.ID, fingerprint)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, "failed to submit quiz")
				return
			}
			if count >= maxAttempts {
				response.Error(w, http.StatusConflict, retakeLimitMessage("Device ini", maxAttempts))
				return
			}
		}

		inputs := make([]models.PublicSubmissionAnswerInput, 0, len(req.Answers))
		for _, answer := range req.Answers {
			if answer.QuestionID == nil {
				response.Error(w, http.StatusBadRequest, "Question wajib dipilih.")
				return
			}
			matrixAnswers := make([]models.PublicMatrixAnswerInput, 0, len(answer.MatrixAnswers))
			for _, matrixAnswer := range answer.MatrixAnswers {
				matrixAnswers = append(matrixAnswers, models.PublicMatrixAnswerInput{
					RowID:            matrixAnswer.RowID,
					QuestionAnswerID: matrixAnswer.QuestionAnswerID,
				})
			}
			inputs = append(inputs, models.PublicSubmissionAnswerInput{
				QuestionID:        *answer.QuestionID,
				QuestionAnswerID:  answer.QuestionAnswerID,
				AnswerText:        normalizeStr(answer.AnswerText),
				SelectedAnswerIDs: answer.SelectedAnswerIDs,
				MatrixAnswers:     matrixAnswers,
			})
		}

		quiz := models.Quiz{
			ID:                  publicQuiz.ID,
			Title:               publicQuiz.Title,
			Type:                publicQuiz.Type,
			StartTime:           publicQuiz.StartTime,
			EndTime:             publicQuiz.EndTime,
			Description:         publicQuiz.Description,
			MaxPoint:            publicQuiz.MaxPoint,
			PassingGrade:        publicQuiz.PassingGrade,
			RandomQuestionCount: publicQuiz.RandomQuestionCount,
			Status:              publicQuiz.Status,
		}

		violationCount := 0
		if req.ViolationCount != nil && *req.ViolationCount > 0 {
			violationCount = *req.ViolationCount
		}
		result, err := models.SavePublicSubmission(db, quiz, email, name, inputs, startedAt, now, attemptSeed, req.DeviceFingerprint, violationCount)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				response.Error(w, http.StatusConflict, "Email atau device ini sudah pernah mengirim quiz ini.")
				return
			}
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}

		response.JSON(w, http.StatusCreated, result)
	}
}

// PublicSubmissionCertificateHandler generate PDF sertifikat 1 submission quiz publik. Divalidasi
// lewat token share (bukan cuma submission id) biar cuma submission milik quiz yang sama yang bisa diakses.
func PublicSubmissionCertificateHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		token := strings.TrimSpace(r.PathValue("token"))
		submissionID, err := strconv.ParseInt(r.PathValue("submissionId"), 10, 64)
		if token == "" || err != nil {
			response.Error(w, http.StatusBadRequest, "invalid request")
			return
		}

		share, err := models.GetQuizShareByToken(db, token)
		if errors.Is(err, sql.ErrNoRows) || share.QuizID == nil {
			response.Error(w, http.StatusNotFound, "Link tidak ditemukan.")
			return
		}
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to load certificate")
			return
		}

		data, err := models.GetCertificateData(db, *share.QuizID, submissionID)
		if errors.Is(err, sql.ErrNoRows) {
			response.Error(w, http.StatusNotFound, "Sertifikat tidak tersedia untuk submission ini.")
			return
		}
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to load certificate")
			return
		}

		pdfBytes, err := models.BuildCertificatePDF(data)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to build certificate")
			return
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="sertifikat-submission-%d.pdf"`, submissionID))
		w.Write(pdfBytes)
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

// validatePublicSubmissionName mewajibkan nama untuk tipe quiz (dipakai cetak sertifikat &
// leaderboard), sementara survey boleh kosong karena gak ada sertifikat/leaderboard buat survey.
func validatePublicSubmissionName(quiz models.PublicQuiz, name *string) (*string, string) {
	if quiz.Type == nil || *quiz.Type != "quiz" {
		return nil, ""
	}
	if name == nil || strings.TrimSpace(*name) == "" {
		return nil, "Nama wajib diisi sebelum mulai quiz."
	}
	trimmed := strings.TrimSpace(*name)
	return &trimmed, ""
}

// retakeLimitMessage pesan error 409 saat batas percobaan (max_attempts) tercapai, beda kalimat
// buat maxAttempts=1 (behavior lama, "sudah pernah") vs >1 (retake, sebut angka batasnya).
func retakeLimitMessage(subject string, maxAttempts int) string {
	if maxAttempts <= 1 {
		return fmt.Sprintf("%s sudah pernah mengirim quiz ini.", subject)
	}
	return fmt.Sprintf("%s sudah mencapai batas maksimal %d kali percobaan quiz ini.", subject, maxAttempts)
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
