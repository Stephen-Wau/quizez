package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"quizez/backend/internal/models"
)

// QuizSummaryHandler balikin analytics summary 1 quiz/survey lengkap buat halaman dashboard admin.
func QuizSummaryHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		summary, err := models.GetQuizSummary(db, id)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Quiz tidak ditemukan.", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "failed to load quiz summary", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(summary)
	}
}

// QuizLeaderboardHandler balikin ranking submission 1 quiz (score tertinggi, tie-break durasi
// pengerjaan tercepat) buat panel gamifikasi di menu Summary admin.
func QuizLeaderboardHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		leaderboard, err := models.GetQuizLeaderboard(db, id)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Quiz tidak ditemukan.", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "failed to load leaderboard", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(leaderboard)
	}
}

// QuizSubmissionDetailHandler balikin detail lengkap satu submission untuk drill-down per respondent.
func QuizSubmissionDetailHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		quizID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid quiz id", http.StatusBadRequest)
			return
		}
		submissionID, err := strconv.ParseInt(r.PathValue("submissionId"), 10, 64)
		if err != nil {
			http.Error(w, "invalid submission id", http.StatusBadRequest)
			return
		}

		detail, err := models.GetQuizSubmissionDetail(db, quizID, submissionID)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Submission tidak ditemukan.", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "failed to load submission detail", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detail)
	}
}

// QuizSubmissionCertificateHandler generate PDF sertifikat 1 submission dari sisi admin (Summary),
// buat kasus respondent lupa/minta ulang download. Beda dari versi publik (public_quiz_handler.go)
// yang divalidasi lewat token share, admin udah punya akses quiz_id langsung dari URL CMS.
func QuizSubmissionCertificateHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		quizID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid quiz id", http.StatusBadRequest)
			return
		}
		submissionID, err := strconv.ParseInt(r.PathValue("submissionId"), 10, 64)
		if err != nil {
			http.Error(w, "invalid submission id", http.StatusBadRequest)
			return
		}

		data, err := models.GetCertificateData(db, quizID, submissionID)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Sertifikat tidak tersedia untuk submission ini.", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "failed to load certificate", http.StatusInternalServerError)
			return
		}

		pdfBytes, err := models.BuildCertificatePDF(data)
		if err != nil {
			http.Error(w, "failed to build certificate", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="sertifikat-submission-%d.pdf"`, submissionID))
		w.Write(pdfBytes)
	}
}
