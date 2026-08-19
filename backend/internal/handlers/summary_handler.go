package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"quizez/backend/internal/models"
	"quizez/backend/internal/response"
)

// QuizSummaryHandler balikin analytics summary 1 quiz/survey lengkap buat halaman dashboard admin.
func QuizSummaryHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid id")
			return
		}

		summary, err := models.GetQuizSummary(db, id)
		if errors.Is(err, sql.ErrNoRows) {
			response.Error(w, http.StatusNotFound, "Quiz tidak ditemukan.")
			return
		}
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to load quiz summary")
			return
		}

		response.JSON(w, http.StatusOK, summary)
	}
}

// QuizLeaderboardHandler balikin ranking submission 1 quiz (score tertinggi, tie-break durasi
// pengerjaan tercepat) buat panel gamifikasi di menu Summary admin.
func QuizLeaderboardHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid id")
			return
		}

		leaderboard, err := models.GetQuizLeaderboard(db, id)
		if errors.Is(err, sql.ErrNoRows) {
			response.Error(w, http.StatusNotFound, "Quiz tidak ditemukan.")
			return
		}
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to load leaderboard")
			return
		}

		response.JSON(w, http.StatusOK, leaderboard)
	}
}

// QuizSubmissionDetailHandler balikin detail lengkap satu submission untuk drill-down per respondent.
func QuizSubmissionDetailHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		quizID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid quiz id")
			return
		}
		submissionID, err := strconv.ParseInt(r.PathValue("submissionId"), 10, 64)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid submission id")
			return
		}

		detail, err := models.GetQuizSubmissionDetail(db, quizID, submissionID)
		if errors.Is(err, sql.ErrNoRows) {
			response.Error(w, http.StatusNotFound, "Submission tidak ditemukan.")
			return
		}
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to load submission detail")
			return
		}

		response.JSON(w, http.StatusOK, detail)
	}
}

// QuizSubmissionCertificateHandler generate PDF sertifikat 1 submission dari sisi admin (Summary),
// buat kasus respondent lupa/minta ulang download. Beda dari versi publik (public_quiz_handler.go)
// yang divalidasi lewat token share, admin udah punya akses quiz_id langsung dari URL CMS.
func QuizSubmissionCertificateHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		quizID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid quiz id")
			return
		}
		submissionID, err := strconv.ParseInt(r.PathValue("submissionId"), 10, 64)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid submission id")
			return
		}

		data, err := models.GetCertificateData(db, quizID, submissionID)
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
