package models

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/xuri/excelize/v2"

	"quizez/backend/internal/export"
)

// rankedIncorrectQuestion bungkus QuestionIncorrectRank + nomor urut, dipakai kolom "#" di
// tabel PDF "Top Incorrect Questions" (Column.Value gak nerima index, jadi rank digabung di sini).
type rankedIncorrectQuestion struct {
	Rank  int
	Inner QuestionIncorrectRank
}

// submissionExportColumns definisi kolom export submission (dipakai bareng CSV & sheet Excel) --
// mau ubah nama/urutan/tambah kolom baru, cukup edit di sini, otomatis kepakai di kedua format.
func submissionExportColumns() []export.Column[SubmissionSummary] {
	return []export.Column[SubmissionSummary]{
		{Header: "ID", Value: func(s SubmissionSummary) string { return strconv.FormatInt(s.ID, 10) }},
		{Header: "Respondent", Value: func(s SubmissionSummary) string { return stringPtrValue(s.RespondentEmail) }},
		{Header: "Score", Value: func(s SubmissionSummary) string { return intPtrToStr(s.Score) }},
		{Header: "Passing Grade", Value: func(s SubmissionSummary) string { return intPtrToStr(s.PassingGrade) }},
		{Header: "Passed", Value: func(s SubmissionSummary) string { return boolPtrToStr(s.Passed) }},
		{Header: "Score %", Value: func(s SubmissionSummary) string { return floatPtrToStr(s.ScorePercentage) }},
		{Header: "Correct", Value: func(s SubmissionSummary) string { return strconv.Itoa(s.CorrectAnswers) }},
		{Header: "Incorrect", Value: func(s SubmissionSummary) string { return strconv.Itoa(s.IncorrectAnswers) }},
		{Header: "Total Questions", Value: func(s SubmissionSummary) string { return strconv.Itoa(s.TotalQuestions) }},
		{Header: "Started At", Value: func(s SubmissionSummary) string { return stringPtrValue(s.StartedAt) }},
		{Header: "Submitted At", Value: func(s SubmissionSummary) string { return stringPtrValue(s.SubmittedAt) }},
		{Header: "Completion %", Value: func(s SubmissionSummary) string { return fmt.Sprintf("%.2f", s.CompletionPercentage) }},
	}
}

// WriteSubmissionsCSV tulis raw submission (hasil filter analytics) ke writer dalam format CSV,
// dipakai handler export "format=csv".
func WriteSubmissionsCSV(w io.Writer, analytics QuizAnalytics) error {
	return export.WriteCSV(w, submissionExportColumns(), analytics.SubmissionSummaries)
}

// BuildSubmissionsXLSX bikin file Excel 2 sheet: "Summary" (KPI ringkas) dan "Submissions" (raw
// data per baris, header bold) — dipakai handler export "format=xlsx". Caller wajib Close() file.
func BuildSubmissionsXLSX(analytics QuizAnalytics) (*excelize.File, error) {
	f := excelize.NewFile()

	summarySheet := "Summary"
	f.SetSheetName("Sheet1", summarySheet)
	if err := writeSummarySheet(f, summarySheet, analytics); err != nil {
		return nil, err
	}

	submissionsSheet := "Submissions"
	if _, err := f.NewSheet(submissionsSheet); err != nil {
		return nil, err
	}
	if err := export.WriteXLSXSheet(f, submissionsSheet, submissionExportColumns(), analytics.SubmissionSummaries); err != nil {
		return nil, err
	}

	f.SetActiveSheet(0)
	return f, nil
}

// summaryMetric 1 baris sheet "Summary": pasangan label KPI + value-nya (bukan raw submission,
// jadi dimodelin terpisah dari submissionExportColumns yang per-baris-per-submission).
type summaryMetric struct {
	Label string
	Value string
}

func summaryMetricColumns() []export.Column[summaryMetric] {
	return []export.Column[summaryMetric]{
		{Header: "Metric", Value: func(m summaryMetric) string { return m.Label }},
		{Header: "Value", Value: func(m summaryMetric) string { return m.Value }},
	}
}

// writeSummarySheet isi sheet "Summary": judul quiz + KPI utama dalam bentuk pasangan label-value.
func writeSummarySheet(f *excelize.File, sheet string, analytics QuizAnalytics) error {
	metrics := []summaryMetric{
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
	if err := export.WriteXLSXSheet(f, sheet, summaryMetricColumns(), metrics); err != nil {
		return err
	}
	f.SetColWidth(sheet, "A", "A", 22)
	f.SetColWidth(sheet, "B", "B", 20)
	return nil
}

// BuildSummaryPDF susun laporan ringkas 1 halaman (KPI, top incorrect question, trend) dalam
// format PDF — dipakai handler export "format=pdf". Fokus ke ringkasan (bukan raw data mentah)
// karena PDF kurang cocok buat tabel data besar dibanding CSV/Excel.
func BuildSummaryPDF(analytics QuizAnalytics) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 10, export.SanitizePDFText(fmt.Sprintf("Analytics Report - %s", stringPtrValue(analytics.Quiz.Title))), "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 6, export.SanitizePDFText(fmt.Sprintf("Type: %s", stringPtrValue(analytics.Quiz.Type))), "", 1, "L", false, 0, "")
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
	export.WritePDFKeyValueRows(pdf, statRows)
	pdf.Ln(4)

	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 8, "Top Incorrect Questions", "", 1, "L", false, 0, "")
	if len(analytics.TopIncorrectQuestions) == 0 {
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(0, 6, "Tidak ada data.", "", 1, "L", false, 0, "")
	} else {
		ranked := make([]rankedIncorrectQuestion, 0, len(analytics.TopIncorrectQuestions))
		for i, q := range analytics.TopIncorrectQuestions {
			ranked = append(ranked, rankedIncorrectQuestion{Rank: i + 1, Inner: q})
		}
		columns := []export.Column[rankedIncorrectQuestion]{
			{Header: "#", Value: func(r rankedIncorrectQuestion) string { return strconv.Itoa(r.Rank) }},
			{Header: "Question", Value: func(r rankedIncorrectQuestion) string { return stringPtrValue(r.Inner.Question) }},
			{Header: "Incorrect %", Value: func(r rankedIncorrectQuestion) string { return fmt.Sprintf("%.0f%%", r.Inner.IncorrectRate) }},
			{Header: "Incorrect / Total", Value: func(r rankedIncorrectQuestion) string {
				return fmt.Sprintf("%d / %d", r.Inner.IncorrectCount, r.Inner.TotalResponses)
			}},
		}
		export.WritePDFTable(pdf, columns, []float64{10, 120, 25, 35}, ranked)
	}
	pdf.Ln(4)

	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 8, fmt.Sprintf("Trend Submission (per %s)", trendGroupLabel(analytics)), "", 1, "L", false, 0, "")
	if len(analytics.Trend) == 0 {
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(0, 6, "Tidak ada data.", "", 1, "L", false, 0, "")
	} else {
		columns := []export.Column[TrendPoint]{
			{Header: "Periode", Value: func(p TrendPoint) string { return p.Label }},
			{Header: "Submission", Value: func(p TrendPoint) string { return strconv.Itoa(p.Count) }},
		}
		export.WritePDFTable(pdf, columns, []float64{80, 40}, analytics.Trend)
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
