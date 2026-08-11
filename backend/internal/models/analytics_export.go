package models

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/xuri/excelize/v2"
)

// submissionExportRow satu baris data submission mentah yang dipakai bareng oleh export CSV & Excel,
// biar kolom dan urutannya selalu konsisten di kedua format.
type submissionExportRow struct {
	ID           int64
	Respondent   string
	Score        string
	PassingGrade string
	Passed       string
	ScorePercent string
	Correct      string
	Incorrect    string
	Total        string
	StartedAt    string
	SubmittedAt  string
	Completion   string
}

// buildSubmissionExportRows ubah SubmissionSummary (format internal analytics) jadi baris flat
// siap tulis ke CSV/Excel — semua nilai nullable diubah ke string kosong/"-" biar file gampang dibaca.
func buildSubmissionExportRows(analytics QuizAnalytics) []submissionExportRow {
	rows := make([]submissionExportRow, 0, len(analytics.SubmissionSummaries))
	for _, s := range analytics.SubmissionSummaries {
		rows = append(rows, submissionExportRow{
			ID:           s.ID,
			Respondent:   stringPtrValue(s.RespondentEmail),
			Score:        intPtrToStr(s.Score),
			PassingGrade: intPtrToStr(s.PassingGrade),
			Passed:       boolPtrToStr(s.Passed),
			ScorePercent: floatPtrToStr(s.ScorePercentage),
			Correct:      strconv.Itoa(s.CorrectAnswers),
			Incorrect:    strconv.Itoa(s.IncorrectAnswers),
			Total:        strconv.Itoa(s.TotalQuestions),
			StartedAt:    stringPtrValue(s.StartedAt),
			SubmittedAt:  stringPtrValue(s.SubmittedAt),
			Completion:   fmt.Sprintf("%.2f", s.CompletionPercentage),
		})
	}
	return rows
}

var submissionExportHeaders = []string{
	"ID", "Respondent", "Score", "Passing Grade", "Passed", "Score %",
	"Correct", "Incorrect", "Total Questions", "Started At", "Submitted At", "Completion %",
}

// WriteSubmissionsCSV tulis raw submission (hasil filter analytics) ke writer dalam format CSV,
// dipakai handler export "format=csv".
func WriteSubmissionsCSV(w io.Writer, analytics QuizAnalytics) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(submissionExportHeaders); err != nil {
		return err
	}
	for _, row := range buildSubmissionExportRows(analytics) {
		record := []string{
			strconv.FormatInt(row.ID, 10), row.Respondent, row.Score, row.PassingGrade, row.Passed,
			row.ScorePercent, row.Correct, row.Incorrect, row.Total, row.StartedAt, row.SubmittedAt, row.Completion,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return nil
}

// BuildSubmissionsXLSX bikin file Excel 2 sheet: "Summary" (KPI ringkas) dan "Submissions" (raw
// data per baris) — dipakai handler export "format=xlsx". Caller wajib Close() file yang dibalikin.
func BuildSubmissionsXLSX(analytics QuizAnalytics) (*excelize.File, error) {
	f := excelize.NewFile()

	summarySheet := "Summary"
	f.SetSheetName("Sheet1", summarySheet)
	writeSummarySheet(f, summarySheet, analytics)

	submissionsSheet := "Submissions"
	if _, err := f.NewSheet(submissionsSheet); err != nil {
		return nil, err
	}
	if err := writeSubmissionsSheet(f, submissionsSheet, analytics); err != nil {
		return nil, err
	}

	f.SetActiveSheet(0)
	return f, nil
}

// writeSummarySheet isi sheet "Summary": judul quiz + KPI utama dalam bentuk pasangan label-value.
func writeSummarySheet(f *excelize.File, sheet string, analytics QuizAnalytics) {
	rows := [][2]string{
		{"Quiz", stringPtrValue(analytics.Quiz.Title)},
		{"Type", stringPtrValue(analytics.Quiz.Type)},
		{"Total Submissions", strconv.Itoa(analytics.Stats.TotalSubmissions)},
		{"Unique Respondents", strconv.Itoa(analytics.Stats.UniqueRespondents)},
		{"Average Score", floatPtrToStr(analytics.Stats.AverageScore)},
		{"Highest Score", intPtrToStr(analytics.Stats.HighestScore)},
		{"Lowest Score", intPtrToStr(analytics.Stats.LowestScore)},
		{"Average Completion %", fmt.Sprintf("%.2f", analytics.Stats.AverageCompletion)},
		{"Passing Count", strconv.Itoa(analytics.Stats.PassingCount)},
		{"Failing Count", strconv.Itoa(analytics.Stats.FailingCount)},
		{"Passing Rate %", floatPtrToStr(analytics.Stats.PassingRate)},
	}
	f.SetCellValue(sheet, "A1", "Metric")
	f.SetCellValue(sheet, "B1", "Value")
	for i, row := range rows {
		rowNum := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowNum), row[0])
		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowNum), row[1])
	}
	f.SetColWidth(sheet, "A", "A", 22)
	f.SetColWidth(sheet, "B", "B", 20)
}

// writeSubmissionsSheet isi sheet "Submissions": 1 baris header + 1 baris per submission mentah.
func writeSubmissionsSheet(f *excelize.File, sheet string, analytics QuizAnalytics) error {
	for col, header := range submissionExportHeaders {
		cell, err := excelize.CoordinatesToCellName(col+1, 1)
		if err != nil {
			return err
		}
		f.SetCellValue(sheet, cell, header)
	}

	for rowIndex, row := range buildSubmissionExportRows(analytics) {
		values := []interface{}{
			row.ID, row.Respondent, row.Score, row.PassingGrade, row.Passed,
			row.ScorePercent, row.Correct, row.Incorrect, row.Total, row.StartedAt, row.SubmittedAt, row.Completion,
		}
		for col, value := range values {
			cell, err := excelize.CoordinatesToCellName(col+1, rowIndex+2)
			if err != nil {
				return err
			}
			f.SetCellValue(sheet, cell, value)
		}
	}
	return nil
}

// BuildSummaryPDF susun laporan ringkas 1 halaman (KPI, top incorrect question, trend) dalam
// format PDF — dipakai handler export "format=pdf". Fokus ke ringkasan (bukan raw data mentah)
// karena PDF kurang cocok buat tabel data besar dibanding CSV/Excel.
func BuildSummaryPDF(analytics QuizAnalytics) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 10, sanitizePDFText(fmt.Sprintf("Analytics Report - %s", stringPtrValue(analytics.Quiz.Title))), "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 6, sanitizePDFText(fmt.Sprintf("Type: %s", stringPtrValue(analytics.Quiz.Type))), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 8, "Ringkasan (Summary)", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	statRows := [][2]string{
		{"Total Submissions", strconv.Itoa(analytics.Stats.TotalSubmissions)},
		{"Unique Respondents", strconv.Itoa(analytics.Stats.UniqueRespondents)},
		{"Average Score", floatPtrToStr(analytics.Stats.AverageScore)},
		{"Average Completion %", fmt.Sprintf("%.2f", analytics.Stats.AverageCompletion)},
		{"Passing Rate %", floatPtrToStr(analytics.Stats.PassingRate)},
	}
	writePDFKeyValueRows(pdf, statRows)
	pdf.Ln(4)

	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 8, "Top Incorrect Questions", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	if len(analytics.TopIncorrectQuestions) == 0 {
		pdf.CellFormat(0, 6, "Tidak ada data.", "", 1, "L", false, 0, "")
	}
	for i, q := range analytics.TopIncorrectQuestions {
		line := fmt.Sprintf("%d. %s (salah %.0f%%, %d dari %d jawaban)",
			i+1, stringPtrValue(q.Question), q.IncorrectRate, q.IncorrectCount, q.TotalResponses)
		pdf.MultiCell(0, 6, sanitizePDFText(line), "", "L", false)
	}
	pdf.Ln(4)

	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 8, fmt.Sprintf("Trend Submission (per %s)", trendGroupLabel(analytics)), "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	if len(analytics.Trend) == 0 {
		pdf.CellFormat(0, 6, "Tidak ada data.", "", 1, "L", false, 0, "")
	}
	for _, point := range analytics.Trend {
		pdf.CellFormat(0, 6, sanitizePDFText(fmt.Sprintf("%s: %d submission", point.Label, point.Count)), "", 1, "L", false, 0, "")
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// trendGroupLabel cuma buat judul section PDF ("per hari"/"per jam") — trend selalu udah dihitung
// sesuai GroupBy filter sebelum sampai sini, jadi cukup tebak dari format label titik pertama.
func trendGroupLabel(analytics QuizAnalytics) string {
	if len(analytics.Trend) > 0 && strings.Contains(analytics.Trend[0].Label, ":") {
		return "jam"
	}
	return "hari"
}

// writePDFKeyValueRows helper render daftar pasangan label-value sebagai baris teks sederhana.
func writePDFKeyValueRows(pdf *fpdf.Fpdf, rows [][2]string) {
	for _, row := range rows {
		pdf.CellFormat(60, 6, sanitizePDFText(row[0]), "", 0, "L", false, 0, "")
		pdf.CellFormat(0, 6, sanitizePDFText(row[1]), "", 1, "L", false, 0, "")
	}
}

// sanitizePDFText font Arial bawaan fpdf cuma support Latin-1, jadi karakter di luar itu (emoji,
// dsb) dibuang biar Output() gak gagal/ngerender kotak aneh.
func sanitizePDFText(text string) string {
	var b strings.Builder
	for _, r := range text {
		if r <= 255 {
			b.WriteRune(r)
		} else {
			b.WriteRune('?')
		}
	}
	return b.String()
}

// intPtrToStr konversi *int ke string buat kolom export, "-" kalau nil (biar file gampang dibaca).
func intPtrToStr(v *int) string {
	if v == nil {
		return "-"
	}
	return strconv.Itoa(*v)
}

// floatPtrToStr konversi *float64 ke string 2 desimal buat kolom export, "-" kalau nil.
func floatPtrToStr(v *float64) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%.2f", *v)
}

// boolPtrToStr konversi *bool ke "Yes"/"No" buat kolom export, "-" kalau nil.
func boolPtrToStr(v *bool) string {
	if v == nil {
		return "-"
	}
	if *v {
		return "Yes"
	}
	return "No"
}
