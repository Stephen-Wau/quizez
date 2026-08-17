package models

import (
	"bytes"
	"database/sql"
	"fmt"

	"github.com/go-pdf/fpdf"
)

// Threshold tier badge berdasarkan persentase skor. Semua responden yang punya skor selalu dapat
// salah satu tier ini (gak ada state "tanpa badge") -- lulus/tidaknya passing grade dicek terpisah.
const (
	BadgeTierGold   = "gold"
	BadgeTierSilver = "silver"
	BadgeTierBronze = "bronze"
)

// ResolveBadgeTier tentukan tier badge dari persentase skor: >=85% gold, >=60% silver, sisanya bronze.
func ResolveBadgeTier(scorePercentage *float64) *string {
	if scorePercentage == nil {
		return nil
	}
	tier := BadgeTierBronze
	switch {
	case *scorePercentage >= 85:
		tier = BadgeTierGold
	case *scorePercentage >= 60:
		tier = BadgeTierSilver
	}
	return &tier
}

// CertificateData data siap-pakai buat cetak sertifikat PDF 1 submission.
type CertificateData struct {
	QuizTitle       string
	RespondentName  string
	Score           int
	MaxPoint        int
	ScorePercentage float64
	BadgeTier       string
	SubmittedAt     string
}

// GetCertificateData ambil data submission buat sertifikat, sekaligus validasi quiz ini punya
// scoring aktif (type=quiz & max_point>0) -- survey atau quiz tanpa point gak bisa cetak sertifikat.
func GetCertificateData(db *sql.DB, quizID int64, submissionID int64) (CertificateData, error) {
	quiz, err := GetQuizByID(db, quizID)
	if err != nil {
		return CertificateData{}, err
	}
	if quiz.Type == nil || *quiz.Type != "quiz" || quiz.MaxPoint == nil || *quiz.MaxPoint <= 0 {
		return CertificateData{}, sql.ErrNoRows
	}

	var (
		name        sql.NullString
		email       sql.NullString
		score       sql.NullInt64
		submittedAt sql.NullTime
	)
	err = db.QueryRow(
		"SELECT respondent_name, respondent_email, score, submitted_at FROM quiz_submissions WHERE id = ? AND quiz_id = ?",
		submissionID, quizID,
	).Scan(&name, &email, &score, &submittedAt)
	if err != nil {
		return CertificateData{}, err
	}
	if !score.Valid {
		return CertificateData{}, sql.ErrNoRows
	}

	respondentName := stringPtrValue(nullableString(name))
	if respondentName == "" {
		respondentName = stringPtrValue(nullableString(email))
	}

	percentage := roundFloat((float64(score.Int64) / float64(*quiz.MaxPoint)) * 100)
	tier := ResolveBadgeTier(&percentage)

	return CertificateData{
		QuizTitle:       stringPtrValue(quiz.Title),
		RespondentName:  respondentName,
		Score:           int(score.Int64),
		MaxPoint:        *quiz.MaxPoint,
		ScorePercentage: percentage,
		BadgeTier:       stringPtrValue(tier),
		SubmittedAt:     stringPtrValue(nullableTime(submittedAt)),
	}, nil
}

// badgeTierLabel label tampilan Indonesia buat tiap tier badge di sertifikat.
func badgeTierLabel(tier string) string {
	switch tier {
	case BadgeTierGold:
		return "GOLD"
	case BadgeTierSilver:
		return "SILVER"
	default:
		return "BRONZE"
	}
}

// BuildCertificatePDF susun sertifikat 1 halaman landscape (judul, nama respondent, skor, badge
// tier) -- dipakai handler download sertifikat publik setelah submit quiz.
func BuildCertificatePDF(data CertificateData) ([]byte, error) {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.AddPage()

	pageWidth, pageHeight := pdf.GetPageSize()
	pdf.SetLineWidth(1.5)
	pdf.Rect(8, 8, pageWidth-16, pageHeight-16, "D")

	pdf.SetY(30)
	pdf.SetFont("Arial", "B", 28)
	pdf.CellFormat(0, 14, "SERTIFIKAT PENGHARGAAN", "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(0, 10, "Diberikan kepada", "", 1, "C", false, 0, "")

	pdf.Ln(4)
	pdf.SetFont("Arial", "B", 24)
	pdf.CellFormat(0, 14, sanitizePDFText(data.RespondentName), "", 1, "C", false, 0, "")

	pdf.Ln(4)
	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(0, 8, sanitizePDFText(fmt.Sprintf("atas partisipasi dan hasil yang diraih dalam quiz \"%s\"", data.QuizTitle)), "", 1, "C", false, 0, "")

	pdf.Ln(6)
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 10, sanitizePDFText(fmt.Sprintf("Skor: %d / %d (%.2f%%)", data.Score, data.MaxPoint, data.ScorePercentage)), "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 10, sanitizePDFText(fmt.Sprintf("Badge: %s", badgeTierLabel(data.BadgeTier))), "", 1, "C", false, 0, "")

	pdf.Ln(10)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 6, sanitizePDFText(fmt.Sprintf("Diselesaikan pada: %s", data.SubmittedAt)), "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
