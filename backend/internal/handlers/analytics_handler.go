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

// QuizAnalyticsHandler balikin data analytics (stats, distribusi skor, ringkasan question) untuk
// 1 quiz, sudah kena filter query param (period/respondent/skor).
func QuizAnalyticsHandler(db *sql.DB) http.HandlerFunc {
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

		filter := models.ParseAnalyticsFilter(r)
		analytics, err := models.GetQuizAnalytics(db, id, filter)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Quiz tidak ditemukan.", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "failed to load analytics", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(analytics)
	}
}

// QuizAnalyticsExportHandler unduh hasil analytics (summary + raw submission) dalam format
// csv/xlsx/pdf sesuai query param ?format=..., dengan filter yang sama seperti halaman analytics.
func QuizAnalyticsExportHandler(db *sql.DB) http.HandlerFunc {
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

		format := r.URL.Query().Get("format")
		switch format {
		case "csv", "xlsx", "pdf":
		default:
			http.Error(w, "format harus csv, xlsx, atau pdf.", http.StatusBadRequest)
			return
		}

		filter := models.ParseAnalyticsFilter(r)
		analytics, err := models.GetQuizAnalytics(db, id, filter)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Quiz tidak ditemukan.", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "failed to load analytics", http.StatusInternalServerError)
			return
		}

		filenameBase := fmt.Sprintf("analytics-quiz-%d", id)
		switch format {
		case "csv":
			w.Header().Set("Content-Type", "text/csv")
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, filenameBase))
			if err := models.WriteSubmissionsCSV(w, analytics); err != nil {
				http.Error(w, "failed to build csv export", http.StatusInternalServerError)
				return
			}
		case "xlsx":
			file, err := models.BuildSubmissionsXLSX(analytics)
			if err != nil {
				http.Error(w, "failed to build excel export", http.StatusInternalServerError)
				return
			}
			defer file.Close()
			w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.xlsx"`, filenameBase))
			if err := file.Write(w); err != nil {
				http.Error(w, "failed to write excel export", http.StatusInternalServerError)
				return
			}
		case "pdf":
			pdfBytes, err := models.BuildSummaryPDF(analytics)
			if err != nil {
				http.Error(w, "failed to build pdf export", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.pdf"`, filenameBase))
			w.Write(pdfBytes)
		}
	}
}
