package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// AnalyticsFilter nampung semua parameter filter yang didukung halaman Analytics & Reporting:
// rentang tanggal submit, pencarian respondent (email), dan rentang skor.
type AnalyticsFilter struct {
	StartDate  *time.Time
	EndDate    *time.Time
	Respondent string
	MinScore   *int
	MaxScore   *int
	GroupBy    string // "day" atau "hour", default "day"
}

// ParseAnalyticsFilter baca query param dari request (?start_date=&end_date=&respondent=&min_score=&max_score=&group_by=)
// jadi AnalyticsFilter siap pakai. Semua param opsional; format tanggal "YYYY-MM-DD".
func ParseAnalyticsFilter(r *http.Request) AnalyticsFilter {
	q := r.URL.Query()
	filter := AnalyticsFilter{
		Respondent: strings.TrimSpace(q.Get("respondent")),
		GroupBy:    strings.ToLower(strings.TrimSpace(q.Get("group_by"))),
	}
	if filter.GroupBy != "hour" {
		filter.GroupBy = "day"
	}

	if v := strings.TrimSpace(q.Get("start_date")); v != "" {
		if t, err := time.ParseInLocation("2006-01-02", v, time.Local); err == nil {
			filter.StartDate = &t
		}
	}
	if v := strings.TrimSpace(q.Get("end_date")); v != "" {
		if t, err := time.ParseInLocation("2006-01-02", v, time.Local); err == nil {
			// end_date inklusif sampai akhir hari itu, bukan jam 00:00, biar submission di hari
			// yang sama ikut kehitung.
			endOfDay := t.Add(24*time.Hour - time.Second)
			filter.EndDate = &endOfDay
		}
	}
	if v := strings.TrimSpace(q.Get("min_score")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.MinScore = &n
		}
	}
	if v := strings.TrimSpace(q.Get("max_score")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.MaxScore = &n
		}
	}
	return filter
}

// TrendPoint 1 titik data trend chart: label tanggal/jam beserta jumlah submission di periode itu.
type TrendPoint struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// QuestionIncorrectRank rangking question yang paling sering dijawab salah, dipakai widget
// "Top incorrect questions".
type QuestionIncorrectRank struct {
	QuestionID     int64   `json:"question_id"`
	Question       *string `json:"question"`
	TotalResponses int     `json:"total_responses"`
	IncorrectCount int     `json:"incorrect_count"`
	IncorrectRate  float64 `json:"incorrect_rate"`
}

// KeywordCount 1 entri kata kunci hasil grouping sederhana dari jawaban free text, beserta
// berapa kali kata itu muncul.
type KeywordCount struct {
	Keyword string `json:"keyword"`
	Count   int    `json:"count"`
}

// QuestionSentimentSummary ringkasan sentimen jawaban free text untuk 1 question survey:
// jumlah positif/netral/negatif (heuristik keyword) + kata kunci yang paling sering muncul.
type QuestionSentimentSummary struct {
	QuestionID  int64          `json:"question_id"`
	Question    *string        `json:"question"`
	Positive    int            `json:"positive"`
	Neutral     int            `json:"neutral"`
	Negative    int            `json:"negative"`
	TopKeywords []KeywordCount `json:"top_keywords"`
}

// QuizAnalytics gabungan data buat halaman Analytics & Reporting: dasar dari QuizSummary (stats,
// distribusi skor, ringkasan question, tabel submission) ditambah trend submission, ranking top
// incorrect question, dan ringkasan sentimen — semua sudah kena filter (period/respondent/skor) yang sama.
type QuizAnalytics struct {
	Quiz                  Quiz                       `json:"quiz"`
	Stats                 QuizSummaryStats           `json:"stats"`
	ScoreDistribution     []SummaryBucket            `json:"score_distribution"`
	QuestionSummaries     []QuestionSummary          `json:"question_summaries"`
	SubmissionSummaries   []SubmissionSummary        `json:"submission_summaries"`
	Trend                 []TrendPoint               `json:"trend"`
	TopIncorrectQuestions []QuestionIncorrectRank    `json:"top_incorrect_questions"`
	SentimentSummaries    []QuestionSentimentSummary `json:"sentiment_summaries"`
}

// GetQuizAnalytics rangkum data Analytics & Reporting untuk 1 quiz, dengan submission yang sudah
// difilter sesuai AnalyticsFilter. Builder inti (stats, question summary, dst) dipakai bareng
// dengan summary.go supaya rule perhitungan skor/correctness tetap satu sumber.
func GetQuizAnalytics(db *sql.DB, quizID int64, filter AnalyticsFilter) (QuizAnalytics, error) {
	quiz, err := GetQuizByID(db, quizID)
	if err != nil {
		return QuizAnalytics{}, err
	}

	questions, err := ListQuestionsByQuiz(db, quizID)
	if err != nil {
		return QuizAnalytics{}, err
	}

	submissions, err := listFilteredSubmissionSummaries(db, quizID, filter)
	if err != nil {
		return QuizAnalytics{}, err
	}

	submissionIDs := make([]int64, 0, len(submissions))
	for _, s := range submissions {
		submissionIDs = append(submissionIDs, s.ID)
	}
	answerRows, err := listAnswerRowsForSubmissionIDs(db, submissionIDs)
	if err != nil {
		return QuizAnalytics{}, err
	}

	attachAnswersToSubmissions(quiz, questions, submissions, answerRows, len(questions))
	stats := buildQuizSummaryStats(quiz, submissions)
	questionSummaries := buildQuestionSummaries(questions, answerRows)

	return QuizAnalytics{
		Quiz:                  quiz,
		Stats:                 stats,
		ScoreDistribution:     buildScoreDistribution(quiz, submissions),
		QuestionSummaries:     questionSummaries,
		SubmissionSummaries:   submissions,
		Trend:                 buildTrendPoints(submissions, filter.GroupBy),
		TopIncorrectQuestions: buildTopIncorrectQuestions(questionSummaries),
		SentimentSummaries:    buildSentimentSummaries(questionSummaries),
	}, nil
}

// listFilteredSubmissionSummaries sama seperti listSubmissionSummaries (summary.go) tapi nerima
// filter tambahan: rentang tanggal submit, pencarian respondent, dan rentang skor — semua diterapkan
// langsung di WHERE clause SQL supaya query tetap efisien buat dataset besar.
func listFilteredSubmissionSummaries(db *sql.DB, quizID int64, filter AnalyticsFilter) ([]SubmissionSummary, error) {
	whereClause := "WHERE quiz_id = ?"
	args := []interface{}{quizID}

	if filter.StartDate != nil {
		whereClause += " AND submitted_at >= ?"
		args = append(args, filter.StartDate)
	}
	if filter.EndDate != nil {
		whereClause += " AND submitted_at <= ?"
		args = append(args, filter.EndDate)
	}
	if filter.Respondent != "" {
		whereClause += " AND respondent_email LIKE ?"
		args = append(args, "%"+filter.Respondent+"%")
	}
	if filter.MinScore != nil {
		whereClause += " AND score >= ?"
		args = append(args, *filter.MinScore)
	}
	if filter.MaxScore != nil {
		whereClause += " AND score <= ?"
		args = append(args, *filter.MaxScore)
	}

	rows, err := db.Query(
		"SELECT id, respondent_email, score, started_at, submitted_at FROM quiz_submissions "+whereClause+" ORDER BY submitted_at DESC, id DESC",
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summaries := []SubmissionSummary{}
	for rows.Next() {
		var (
			item        SubmissionSummary
			email       sql.NullString
			score       sql.NullInt64
			startedAt   sql.NullTime
			submittedAt sql.NullTime
		)
		if err := rows.Scan(&item.ID, &email, &score, &startedAt, &submittedAt); err != nil {
			return nil, err
		}
		item.RespondentEmail = nullableString(email)
		if score.Valid {
			v := int(score.Int64)
			item.Score = &v
		}
		item.StartedAt = nullableTime(startedAt)
		item.SubmittedAt = nullableTime(submittedAt)
		summaries = append(summaries, item)
	}
	return summaries, rows.Err()
}

// listAnswerRowsForSubmissionIDs ambil semua jawaban milik daftar submission ID tertentu — dipakai
// supaya analytics per-question (distribusi opsi, dst) hanya menghitung submission yang lolos
// filter, bukan seluruh submission quiz.
func listAnswerRowsForSubmissionIDs(db *sql.DB, submissionIDs []int64) ([]summaryAnswerRow, error) {
	if len(submissionIDs) == 0 {
		return []summaryAnswerRow{}, nil
	}

	placeholders := make([]string, len(submissionIDs))
	args := make([]interface{}, len(submissionIDs))
	for i, id := range submissionIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`SELECT
			qsa.submission_id,
			qsa.question_id,
			q.question,
			q.type_answer,
			q.point,
			qsa.answer_label,
			qsa.answer_value,
			qsa.answer_text,
			qsa.is_correct,
			qsa.selected_answer_ids,
			qsa.matrix_row_id,
			qmr.row_label,
			qs.respondent_email,
			qs.started_at,
			qs.submitted_at
		FROM quiz_submission_answers qsa
		INNER JOIN questions q ON q.id = qsa.question_id
		INNER JOIN quiz_submissions qs ON qs.id = qsa.submission_id
		LEFT JOIN question_matrix_rows qmr ON qmr.id = qsa.matrix_row_id
		WHERE qsa.submission_id IN (%s)
		ORDER BY qs.submitted_at DESC, qsa.submission_id DESC, qsa.question_id ASC`, strings.Join(placeholders, ","))

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	answerRows := []summaryAnswerRow{}
	for rows.Next() {
		var (
			row               summaryAnswerRow
			question          sql.NullString
			typeAnswer        sql.NullString
			answerLabel       sql.NullString
			answerValue       sql.NullString
			answerText        sql.NullString
			respondent        sql.NullString
			startedAt         sql.NullTime
			submittedAt       sql.NullTime
			point             sql.NullInt64
			isCorrect         sql.NullBool
			selectedAnswerIDs sql.NullString
			matrixRowID       sql.NullInt64
			matrixRowLabel    sql.NullString
		)
		if err := rows.Scan(
			&row.SubmissionID, &row.QuestionID, &question, &typeAnswer, &point,
			&answerLabel, &answerValue, &answerText, &isCorrect, &selectedAnswerIDs, &matrixRowID, &matrixRowLabel,
			&respondent, &startedAt, &submittedAt,
		); err != nil {
			return nil, err
		}
		row.Question = nullableString(question)
		row.TypeAnswer = nullableString(typeAnswer)
		if point.Valid {
			v := int(point.Int64)
			row.Point = &v
		}
		row.AnswerLabel = nullableString(answerLabel)
		row.AnswerValue = nullableString(answerValue)
		row.AnswerText = nullableString(answerText)
		row.RespondentEmail = nullableString(respondent)
		row.StartedAt = nullableTime(startedAt)
		row.SubmittedAt = nullableTime(submittedAt)
		if isCorrect.Valid {
			v := isCorrect.Bool
			row.IsCorrect = &v
		}
		if selectedAnswerIDs.Valid {
			row.SelectedAnswerIDsRaw = selectedAnswerIDs.String
		}
		if matrixRowID.Valid {
			v := matrixRowID.Int64
			row.MatrixRowID = &v
		}
		row.MatrixRowLabel = nullableString(matrixRowLabel)
		answerRows = append(answerRows, row)
	}
	return answerRows, rows.Err()
}

// buildTrendPoints kelompokkan submission per hari (default) atau per jam ("hour") jadi rangkaian
// titik trend chart, diurutkan dari periode paling lama ke paling baru biar grafik enak dibaca kiri-kanan.
func buildTrendPoints(submissions []SubmissionSummary, groupBy string) []TrendPoint {
	counts := map[string]int{}
	for _, submission := range submissions {
		if submission.SubmittedAt == nil {
			continue
		}
		t, err := time.ParseInLocation(dateTimeLayout, *submission.SubmittedAt, time.Local)
		if err != nil {
			continue
		}
		label := t.Format("2006-01-02")
		if groupBy == "hour" {
			label = t.Format("2006-01-02 15:00")
		}
		counts[label]++
	}

	labels := make([]string, 0, len(counts))
	for label := range counts {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	points := make([]TrendPoint, 0, len(labels))
	for _, label := range labels {
		points = append(points, TrendPoint{Label: label, Count: counts[label]})
	}
	return points
}

// buildTopIncorrectQuestions rangking question pilihan ganda berdasarkan rate jawaban salah,
// dari yang paling sering salah ke paling jarang. Dibatasi 10 teratas biar widget tetap ringkas.
// Cuma question dengan minimal 1 jawaban yang dihitung — question yang belum pernah dijawab siapa
// pun tidak informatif buat ranking ini.
func buildTopIncorrectQuestions(questionSummaries []QuestionSummary) []QuestionIncorrectRank {
	ranks := make([]QuestionIncorrectRank, 0, len(questionSummaries))
	for _, q := range questionSummaries {
		totalGraded := q.CorrectCount + q.IncorrectCount
		if totalGraded == 0 {
			continue
		}
		rank := QuestionIncorrectRank{
			QuestionID:     q.QuestionID,
			Question:       q.Question,
			TotalResponses: totalGraded,
			IncorrectCount: q.IncorrectCount,
			IncorrectRate:  roundFloat((float64(q.IncorrectCount) / float64(totalGraded)) * 100),
		}
		ranks = append(ranks, rank)
	}

	sort.Slice(ranks, func(i, j int) bool {
		// Urutan utama: rate salah tertinggi dulu. Kalau seri, question dengan lebih banyak
		// responden yang dinilai dianggap lebih meyakinkan untuk ditampilkan lebih dulu.
		if ranks[i].IncorrectRate != ranks[j].IncorrectRate {
			return ranks[i].IncorrectRate > ranks[j].IncorrectRate
		}
		return ranks[i].TotalResponses > ranks[j].TotalResponses
	})

	if len(ranks) > 10 {
		ranks = ranks[:10]
	}
	return ranks
}

// Lexicon sentiment super sederhana (Indonesia + Inggris) — cukup buat heuristik kasar "minimal
// keyword grouping" sesuai kebutuhan fitur, bukan model NLP sungguhan.
var positiveWords = map[string]bool{
	"bagus": true, "baik": true, "senang": true, "suka": true, "mantap": true, "oke": true, "ok": true,
	"memuaskan": true, "puas": true, "keren": true, "hebat": true, "cepat": true, "ramah": true,
	"good": true, "great": true, "nice": true, "love": true, "excellent": true, "amazing": true, "helpful": true,
}

var negativeWords = map[string]bool{
	"buruk": true, "jelek": true, "kecewa": true, "lambat": true, "susah": true, "sulit": true,
	"lama": true, "parah": true, "gagal": true, "rusak": true, "marah": true,
	"bad": true, "poor": true, "terrible": true, "awful": true, "hate": true, "slow": true, "worst": true,
}

// stopwords kata umum yang dibuang sebelum hitung frekuensi kata kunci, biar top keyword yang muncul
// benar-benar kata yang bermakna (bukan "yang", "dan", "the", dsb).
var stopwords = map[string]bool{
	"yang": true, "dan": true, "di": true, "ini": true, "itu": true, "untuk": true, "dengan": true,
	"saya": true, "kami": true, "juga": true, "ada": true, "tidak": true, "sangat": true, "sekali": true,
	"the": true, "and": true, "is": true, "are": true, "to": true, "of": true, "a": true, "in": true,
	"it": true, "this": true, "that": true, "very": true, "for": true,
}

var wordSplitPattern = regexp.MustCompile(`[a-zA-Z0-9']+`)

// buildSentimentSummaries hitung sentiment sederhana + top keyword untuk tiap question bertipe
// free_text. Klasifikasi berbasis keyword lexicon (bukan ML): jawaban dianggap positif kalau kata
// positif lebih banyak dari negatif, negatif kalau sebaliknya, netral kalau seri/tidak ada match.
func buildSentimentSummaries(questionSummaries []QuestionSummary) []QuestionSentimentSummary {
	summaries := []QuestionSentimentSummary{}
	for _, q := range questionSummaries {
		if q.TypeAnswer == nil || *q.TypeAnswer != "free_text" || len(q.TextResponses) == 0 {
			continue
		}

		summary := QuestionSentimentSummary{QuestionID: q.QuestionID, Question: q.Question, TopKeywords: []KeywordCount{}}
		keywordCounts := map[string]int{}

		for _, response := range q.TextResponses {
			words := wordSplitPattern.FindAllString(strings.ToLower(response.AnswerText), -1)
			positiveHits := 0
			negativeHits := 0
			for _, word := range words {
				if positiveWords[word] {
					positiveHits++
				}
				if negativeWords[word] {
					negativeHits++
				}
				if !stopwords[word] && len(word) > 2 {
					keywordCounts[word]++
				}
			}

			switch {
			case positiveHits > negativeHits:
				summary.Positive++
			case negativeHits > positiveHits:
				summary.Negative++
			default:
				summary.Neutral++
			}
		}

		summary.TopKeywords = topKeywords(keywordCounts, 5)
		summaries = append(summaries, summary)
	}
	return summaries
}

// topKeywords urutkan hasil hitung frekuensi kata lalu ambil N kata paling sering muncul.
func topKeywords(counts map[string]int, limit int) []KeywordCount {
	keywords := make([]KeywordCount, 0, len(counts))
	for word, count := range counts {
		keywords = append(keywords, KeywordCount{Keyword: word, Count: count})
	}
	sort.Slice(keywords, func(i, j int) bool {
		if keywords[i].Count != keywords[j].Count {
			return keywords[i].Count > keywords[j].Count
		}
		return keywords[i].Keyword < keywords[j].Keyword
	})
	if len(keywords) > limit {
		keywords = keywords[:limit]
	}
	return keywords
}
