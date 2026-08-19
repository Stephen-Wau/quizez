package handlers

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"quizez/backend/internal/listquery"
	"quizez/backend/internal/models"
	"quizez/backend/internal/response"
)

type questionBankRequest struct {
	Question   *string                 `json:"question"`
	TypeAnswer *string                 `json:"type_answer"`
	Point      *int                    `json:"point"`
	Tags       []string                `json:"tags"`
	Answers    []questionAnswerRequest `json:"answers"`
}

type questionBankImportRequest struct {
	FileName string `json:"file_name"`
	// FileData base64 data URI ("data:<mime>;base64,....") dari app-files-upload, konsisten sama
	// pola upload lain di codebase ini (semua embed base64 di body JSON, gak ada endpoint multipart).
	FileData string `json:"file_data"`
}

type questionBankFromBankRequest struct {
	BankIDs []int64 `json:"bank_ids"`
}

// GET (list) & POST (create) at /api/question-bank
func QuestionBankHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listQuestionBank(w, r, db)
		case http.MethodPost:
			createQuestionBankItem(w, r, db)
		default:
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

// PUT & DELETE at /api/question-bank/{id}
func QuestionBankItemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid id")
			return
		}
		switch r.Method {
		case http.MethodPut:
			updateQuestionBankItem(w, r, db, id)
		case http.MethodDelete:
			deleteQuestionBankItem(w, r, db, id)
		default:
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

// GET /api/question-bank/tags -> daftar tag unik buat filter/autocomplete FE.
func QuestionBankTagsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		tags, err := models.ListQuestionBankTags(db)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to load tags")
			return
		}
		response.JSON(w, http.StatusOK, tags)
	}
}

// GET /api/question-bank/import-template?format=csv|xlsx -> download contoh file import.
func QuestionBankImportTemplateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
		if format == "xlsx" {
			data, err := models.BuildQuestionBankTemplateXLSX()
			if err != nil {
				response.Error(w, http.StatusInternalServerError, "failed to build template")
				return
			}
			w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
			w.Header().Set("Content-Disposition", "attachment; filename=question-bank-template.xlsx")
			w.Write(data)
			return
		}
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=question-bank-template.csv")
		w.Write(models.BuildQuestionBankTemplateCSV())
	}
}

// POST /api/question-bank/import -> bulk import soal dari file CSV/XLSX (base64 dari FE).
func QuestionBankImportHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req questionBankImportRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}

		data, err := decodeDataURI(req.FileData)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "File tidak valid.")
			return
		}

		var items []models.QuestionBankItem
		var rowErrors []models.QuestionBankImportRowError
		lowerName := strings.ToLower(req.FileName)
		switch {
		case strings.HasSuffix(lowerName, ".xlsx"):
			items, rowErrors, err = models.ParseQuestionBankImportXLSX(data)
		case strings.HasSuffix(lowerName, ".csv"):
			items, rowErrors, err = models.ParseQuestionBankImportCSV(data)
		default:
			response.Error(w, http.StatusBadRequest, "Format file harus .csv atau .xlsx.")
			return
		}
		if err != nil {
			response.Error(w, http.StatusBadRequest, "Gagal membaca file, pastikan formatnya sesuai template.")
			return
		}

		created := 0
		for _, item := range items {
			if _, err := models.CreateQuestionBankItem(db, item); err != nil {
				rowErrors = append(rowErrors, models.QuestionBankImportRowError{Message: "Gagal simpan soal \"" + stringValue(item.Question) + "\": " + err.Error()})
				continue
			}
			created++
		}
		writeAuditLog(r, db, "question_bank.import", "question_bank", nil, "Import bulk bank soal dari file.")

		response.JSON(w, http.StatusOK, map[string]interface{}{
			"created": created,
			"errors":  rowErrors,
		})
	}
}

// POST /api/quizzes/{id}/questions/from-bank -> copy sejumlah soal bank jadi question baru milik quiz ini.
func QuizQuestionsFromBankHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		quizID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid id")
			return
		}

		var req questionBankFromBankRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if len(req.BankIDs) == 0 {
			response.Error(w, http.StatusBadRequest, "Pilih minimal 1 soal dari bank.")
			return
		}

		exists, err := models.QuizExists(db, quizID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to add questions")
			return
		}
		if !exists {
			response.Error(w, http.StatusNotFound, "Quiz tidak ditemukan.")
			return
		}
		if msg, status := validateQuizNotLocked(db, quizID); msg != "" {
			response.Error(w, status, msg)
			return
		}

		items, err := models.GetQuestionBankByIDs(db, req.BankIDs)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "Ada soal bank yang tidak ditemukan.")
			return
		}

		// Total point soal yang mau di-copy gak boleh bikin total point quiz kelebihan dari max_point,
		// sama seperti aturan tambah question manual satu-satu.
		totalNewPoint := 0
		for _, item := range items {
			if item.Point != nil {
				totalNewPoint += *item.Point
			}
		}
		maxPoint, usedPoint, err := models.QuizPointBudget(db, quizID, nil)
		if err == nil && maxPoint != nil && usedPoint+totalNewPoint > *maxPoint {
			response.Error(w, http.StatusBadRequest, "Total point soal yang ditambahkan melebihi max point quiz.")
			return
		}

		created, err := models.CopyQuestionBankItemsToQuiz(db, quizID, items)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "failed to add questions")
			return
		}
		writeAuditLog(r, db, "question_bank.copy_to_quiz", "quiz", &quizID, "Menyalin soal dari bank soal ke quiz.")

		response.JSON(w, http.StatusCreated, created)
	}
}

// listQuestionBankItem handle GET /api/question-bank: search/tag filter/sort/pagination standar.
func listQuestionBank(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	params := listquery.Parse(r)
	tagFilter := strings.TrimSpace(r.URL.Query().Get("tag"))
	items, total, err := models.ListQuestionBank(db, params, tagFilter)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to load question bank")
		return
	}
	response.Paginated(w, items, listquery.BuildMeta(params, total))
}

// validateQuestionBankRequest cek business rule form bank soal: mirip question per-quiz tapi
// tanpa matrix (gak didukung bank) dan tanpa limit point (bank gak terikat max_point quiz manapun).
func validateQuestionBankRequest(req questionBankRequest) string {
	if req.Question == nil || strings.TrimSpace(*req.Question) == "" {
		return "Question wajib diisi."
	}
	if req.TypeAnswer == nil {
		return "Type answer wajib dipilih."
	}
	switch *req.TypeAnswer {
	case questionTypeMultipleChoice, questionTypeDropdown, questionTypeCheckbox, questionTypeRating, questionTypeFreeText:
	default:
		return "Type answer harus pilihan ganda, dropdown, checkbox, rating, atau free text."
	}
	if req.Point != nil && *req.Point <= 0 {
		return "Point harus lebih dari 0."
	}

	switch *req.TypeAnswer {
	case questionTypeFreeText:
		if len(req.Answers) > 0 {
			return "Free text tidak boleh punya jawaban pilihan."
		}
	case questionTypeMultipleChoice, questionTypeDropdown, questionTypeCheckbox:
		if msg := validateOptionAnswers(req.Answers); msg != "" {
			return msg
		}
	case questionTypeRating:
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

// createQuestionBankItem handle POST /api/question-bank.
func createQuestionBankItem(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req questionBankRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if msg := validateQuestionBankRequest(req); msg != "" {
		response.Error(w, http.StatusBadRequest, msg)
		return
	}

	item := mapQuestionBankRequestToModel(req)
	id, err := models.CreateQuestionBankItem(db, item)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to save question bank item")
		return
	}
	item.ID = id
	writeAuditLog(r, db, "question_bank.create", "question_bank", &item.ID, "Membuat item bank soal baru.")
	response.JSON(w, http.StatusCreated, item)
}

// updateQuestionBankItem handle PUT /api/question-bank/{id}.
func updateQuestionBankItem(w http.ResponseWriter, r *http.Request, db *sql.DB, id int64) {
	var req questionBankRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if msg := validateQuestionBankRequest(req); msg != "" {
		response.Error(w, http.StatusBadRequest, msg)
		return
	}

	item := mapQuestionBankRequestToModel(req)
	item.ID = id
	if err := models.UpdateQuestionBankItem(db, item); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to update question bank item")
		return
	}
	writeAuditLog(r, db, "question_bank.update", "question_bank", &item.ID, "Memperbarui item bank soal.")
	response.JSON(w, http.StatusOK, item)
}

// deleteQuestionBankItem handle DELETE /api/question-bank/{id}.
func deleteQuestionBankItem(w http.ResponseWriter, r *http.Request, db *sql.DB, id int64) {
	if err := models.DeleteQuestionBankItem(db, id); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to delete question bank item")
		return
	}
	writeAuditLog(r, db, "question_bank.delete", "question_bank", &id, "Menghapus item bank soal.")
	w.WriteHeader(http.StatusNoContent)
}

// mapQuestionBankRequestToModel normalisasi string kosong -> nil, ubah payload API jadi model DB.
func mapQuestionBankRequestToModel(req questionBankRequest) models.QuestionBankItem {
	answers := make([]models.QuestionAnswer, 0, len(req.Answers))
	for _, answer := range req.Answers {
		answers = append(answers, models.QuestionAnswer{
			Label: normalizeStr(answer.Label),
			Value: normalizeStr(answer.Value),
		})
	}
	return models.QuestionBankItem{
		Question:   normalizeStr(req.Question),
		TypeAnswer: normalizeStr(req.TypeAnswer),
		Point:      req.Point,
		Tags:       req.Tags,
		Answers:    models.NormalizeQuestionAnswers(answers),
	}
}

// decodeDataURI decode base64 data URI ("data:<mime>;base64,....") jadi raw bytes, dipakai
// buat baca file CSV/XLSX yang dikirim FE lewat app-files-upload (JSON body, bukan multipart).
func decodeDataURI(dataURI string) ([]byte, error) {
	idx := strings.Index(dataURI, ",")
	if idx == -1 {
		return base64.StdEncoding.DecodeString(dataURI)
	}
	return base64.StdEncoding.DecodeString(dataURI[idx+1:])
}
