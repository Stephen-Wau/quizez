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

// QuizAnalyticsHandler balikin data analytics (stats, distribusi skor, ringkasan question) untuk
// 1 quiz, sudah kena filter query param (period/respondent/skor).
func QuizAnalyticsHandler(db *sql.DB) http.HandlerFunc {
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

		filter := models.ParseAnalyticsFilter(r)
		analytics, err := models.GetQuizAnalytics(db, id, filter)
		if errors.Is(err, sql.ErrNoRows) {
			response.Error(w, http.StatusNotFound, "Quiz tidak ditemukan.")
			return
		}
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to load analytics")
			return
		}

		response.JSON(w, http.StatusOK, analytics)
	}
}

// QuizAnalyticsExportHandler unduh hasil analytics (summary + raw submission) dalam format
// csv/xlsx/pdf sesuai query param ?format=..., dengan filter yang sama seperti halaman analytics.
func QuizAnalyticsExportHandler(db *sql.DB) http.HandlerFunc {
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

		format := r.URL.Query().Get("format")
		switch format {
		case "csv", "xlsx", "pdf":
		default:
			response.Error(w, http.StatusBadRequest, "format harus csv, xlsx, atau pdf.")
			return
		}

		filter := models.ParseAnalyticsFilter(r)
		analytics, err := models.GetQuizAnalytics(db, id, filter)
		if errors.Is(err, sql.ErrNoRows) {
			response.Error(w, http.StatusNotFound, "Quiz tidak ditemukan.")
			return
		}
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to load analytics")
			return
		}

		filenameBase := fmt.Sprintf("analytics-quiz-%d", id)
		writeAuditLog(r, db, "analytics.export", "quiz", &id, "Export analytics quiz.")
		switch format {
		case "csv":
			w.Header().Set("Content-Type", "text/csv")
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, filenameBase))
			if err := models.WriteSubmissionsCSV(w, analytics); err != nil {
				response.Error(w, http.StatusInternalServerError, "failed to build csv export")
				return
			}
		case "xlsx":
			file, err := models.BuildSubmissionsXLSX(analytics)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, "failed to build excel export")
				return
			}
			defer file.Close()
			w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.xlsx"`, filenameBase))
			if err := file.Write(w); err != nil {
				response.Error(w, http.StatusInternalServerError, "failed to write excel export")
				return
			}
		case "pdf":
			pdfBytes, err := models.BuildSummaryPDF(analytics)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, "failed to build pdf export")
				return
			}
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.pdf"`, filenameBase))
			w.Write(pdfBytes)
		}
	}
}
