package models

import (
	"bytes"
	"database/sql"
	"fmt"

	"github.com/go-pdf/fpdf"

	"quizez/backend/internal/export"
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

// LeaderboardEntry 1 baris ranking respondent quiz, sudah termasuk badge tier & durasi pengerjaan.
type LeaderboardEntry struct {
	Rank            int      `json:"rank"`
	SubmissionID    int64    `json:"submission_id"`
	RespondentName  *string  `json:"respondent_name"`
	RespondentEmail *string  `json:"respondent_email"`
	Score           int      `json:"score"`
	ScorePercentage *float64 `json:"score_percentage"`
	BadgeTier       *string  `json:"badge_tier"`
	DurationSeconds *int64   `json:"duration_seconds"`
	SubmittedAt     *string  `json:"submitted_at"`
}

// GetQuizLeaderboard ranking submission 1 quiz berdasarkan score tertinggi, tie-break durasi
// pengerjaan tercepat (started_at ke submitted_at). Survey gak punya score jadi selalu balikin kosong.
func GetQuizLeaderboard(db *sql.DB, quizID int64) ([]LeaderboardEntry, error) {
	quiz, err := GetQuizByID(db, quizID)
	if err != nil {
		return nil, err
	}
	if quiz.Type == nil || *quiz.Type != "quiz" {
		return []LeaderboardEntry{}, nil
	}

	rows, err := db.Query(
		`SELECT id, respondent_name, respondent_email, score, started_at, submitted_at,
			TIMESTAMPDIFF(SECOND, started_at, submitted_at) AS duration_seconds
		FROM quiz_submissions
		WHERE quiz_id = ? AND score IS NOT NULL
		ORDER BY score DESC, duration_seconds ASC, submitted_at ASC`,
		quizID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []LeaderboardEntry{}
	for rows.Next() {
		var (
			id              int64
			name            sql.NullString
			email           sql.NullString
			score           int
			startedAt       sql.NullTime
			submittedAt     sql.NullTime
			durationSeconds sql.NullInt64
		)
		if err := rows.Scan(&id, &name, &email, &score, &startedAt, &submittedAt, &durationSeconds); err != nil {
			return nil, err
		}

		entry := LeaderboardEntry{
			SubmissionID:    id,
			RespondentName:  nullableString(name),
			RespondentEmail: nullableString(email),
			Score:           score,
			SubmittedAt:     nullableTime(submittedAt),
		}
		if durationSeconds.Valid {
			v := durationSeconds.Int64
			entry.DurationSeconds = &v
		}
		if quiz.MaxPoint != nil && *quiz.MaxPoint > 0 {
			percentage := roundFloat((float64(score) / float64(*quiz.MaxPoint)) * 100)
			entry.ScorePercentage = &percentage
		}
		entry.BadgeTier = ResolveBadgeTier(entry.ScorePercentage)
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range entries {
		entries[i].Rank = i + 1
	}
	return entries, nil
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

// badgeTierColor RGB aksen warna per tier badge -- dipakai buat garis dekoratif & kotak badge
// di sertifikat, biar gold/silver/bronze keliatan beda dari sekadar teks.
func badgeTierColor(tier string) (int, int, int) {
	switch tier {
	case BadgeTierGold:
		return 180, 130, 20
	case BadgeTierSilver:
		return 100, 116, 139
	default:
		return 154, 82, 18
	}
}

// BuildCertificatePDF susun sertifikat 1 halaman landscape dengan border ganda, garis dekoratif,
// dan badge box berwarna sesuai tier -- dipakai handler download sertifikat publik setelah submit quiz.
func BuildCertificatePDF(data CertificateData) ([]byte, error) {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.AddPage()

	pageWidth, pageHeight := pdf.GetPageSize()
	accentR, accentG, accentB := 67, 56, 202 // indigo, senada sama warna aksen sidebar admin (#4338CA)
	tierR, tierG, tierB := badgeTierColor(data.BadgeTier)

	// Border ganda: garis luar tebal warna indigo, garis dalam tipis dengan jarak, biar kesan
	// "frame sertifikat" -- bukan cuma kotak polos.
	pdf.SetDrawColor(accentR, accentG, accentB)
	pdf.SetLineWidth(1.2)
	pdf.Rect(8, 8, pageWidth-16, pageHeight-16, "D")
	pdf.SetLineWidth(0.4)
	pdf.Rect(12, 12, pageWidth-24, pageHeight-24, "D")

	centerX := pageWidth / 2

	pdf.SetY(28)
	pdf.SetTextColor(accentR, accentG, accentB)
	pdf.SetFont("Arial", "B", 28)
	pdf.CellFormat(0, 14, "SERTIFIKAT PENGHARGAAN", "", 1, "C", false, 0, "")

	// Garis dekoratif pendek di bawah judul, biar gak langsung nabrak ke teks berikutnya.
	lineWidth := 60.0
	pdf.Line(centerX-lineWidth/2, pdf.GetY()+2, centerX+lineWidth/2, pdf.GetY()+2)
	pdf.Ln(10)

	pdf.SetTextColor(60, 60, 60)
	pdf.SetFont("Arial", "I", 12)
	pdf.CellFormat(0, 10, "Diberikan kepada", "", 1, "C", false, 0, "")

	pdf.Ln(2)
	pdf.SetTextColor(20, 20, 20)
	pdf.SetFont("Arial", "B", 26)
	pdf.CellFormat(0, 14, export.SanitizePDFText(data.RespondentName), "", 1, "C", false, 0, "")

	pdf.SetTextColor(60, 60, 60)
	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(0, 8, export.SanitizePDFText(fmt.Sprintf("atas partisipasi dan hasil yang diraih dalam quiz \"%s\"", data.QuizTitle)), "", 1, "C", false, 0, "")

	pdf.Ln(8)

	// Skor & badge disusun sejajar di tengah halaman (bukan ditumpuk vertikal) biar lebih rapi
	// dibaca sebagai satu "kartu hasil", bukan daftar teks.
	scoreText := export.SanitizePDFText(fmt.Sprintf("Skor: %d / %d (%.2f%%)", data.Score, data.MaxPoint, data.ScorePercentage))
	pdf.SetFont("Arial", "B", 16)
	scoreWidth := pdf.GetStringWidth(scoreText)

	badgeLabel := badgeTierLabel(data.BadgeTier)
	pdf.SetFont("Arial", "B", 12)
	badgeTextWidth := pdf.GetStringWidth(badgeLabel)
	badgeBoxWidth := badgeTextWidth + 16
	badgeBoxHeight := 9.0

	gap := 10.0
	rowY := pdf.GetY()
	rowWidth := scoreWidth + gap + badgeBoxWidth
	startX := centerX - rowWidth/2

	pdf.SetTextColor(20, 20, 20)
	pdf.SetFont("Arial", "B", 16)
	pdf.SetXY(startX, rowY+1)
	pdf.CellFormat(scoreWidth, 8, scoreText, "", 0, "L", false, 0, "")

	badgeX := startX + scoreWidth + gap
	pdf.SetFillColor(tierR, tierG, tierB)
	pdf.RoundedRect(badgeX, rowY, badgeBoxWidth, badgeBoxHeight, 2, "1234", "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetXY(badgeX, rowY)
	pdf.CellFormat(badgeBoxWidth, badgeBoxHeight, badgeLabel, "", 0, "C", false, 0, "")

	pdf.SetY(rowY + badgeBoxHeight + 14)

	// Footer 2 kolom: tanggal di kiri, "stempel" lingkaran nama platform di kanan -- pola umum
	// sertifikat (kolom tanda tangan/tanggal kiri-kanan) tapi disederhanain jadi teks + seal bulat.
	footerY := pageHeight - 34
	pdf.SetDrawColor(accentR, accentG, accentB)
	pdf.SetLineWidth(0.3)

	leftX := 40.0
	pdf.Line(leftX, footerY, leftX+70, footerY)
	pdf.SetXY(leftX, footerY+2)
	pdf.SetTextColor(60, 60, 60)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(70, 6, export.SanitizePDFText(fmt.Sprintf("Diselesaikan pada: %s", data.SubmittedAt)), "", 0, "C", false, 0, "")

	sealX := pageWidth - 55.0
	sealY := footerY - 8
	pdf.SetDrawColor(accentR, accentG, accentB)
	pdf.SetLineWidth(0.6)
	pdf.Circle(sealX, sealY, 12, "D")
	pdf.SetTextColor(accentR, accentG, accentB)
	pdf.SetFont("Arial", "B", 9)
	pdf.SetXY(sealX-12, sealY-3)
	pdf.CellFormat(24, 6, "QUIZEZ", "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 7)
	pdf.SetXY(sealX-12, sealY+1)
	pdf.CellFormat(24, 5, "VERIFIED", "", 0, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
