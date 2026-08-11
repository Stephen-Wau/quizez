package models

import (
	"database/sql"
	"fmt"
	"net/http"
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

// QuizAnalytics gabungan data buat halaman Analytics & Reporting: dasar dari QuizSummary (stats,
// distribusi skor, ringkasan question, tabel submission) — semua sudah kena filter (period/respondent/skor).
type QuizAnalytics struct {
	Quiz                Quiz                `json:"quiz"`
	Stats               QuizSummaryStats    `json:"stats"`
	ScoreDistribution   []SummaryBucket     `json:"score_distribution"`
	QuestionSummaries   []QuestionSummary   `json:"question_summaries"`
	SubmissionSummaries []SubmissionSummary `json:"submission_summaries"`
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
		Quiz:                quiz,
		Stats:               stats,
		ScoreDistribution:   buildScoreDistribution(quiz, submissions),
		QuestionSummaries:   questionSummaries,
		SubmissionSummaries: submissions,
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
			qs.respondent_email,
			qs.started_at,
			qs.submitted_at
		FROM quiz_submission_answers qsa
		INNER JOIN questions q ON q.id = qsa.question_id
		INNER JOIN quiz_submissions qs ON qs.id = qsa.submission_id
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
			row         summaryAnswerRow
			question    sql.NullString
			typeAnswer  sql.NullString
			answerLabel sql.NullString
			answerValue sql.NullString
			answerText  sql.NullString
			respondent  sql.NullString
			startedAt   sql.NullTime
			submittedAt sql.NullTime
			point       sql.NullInt64
			isCorrect   sql.NullBool
		)
		if err := rows.Scan(
			&row.SubmissionID, &row.QuestionID, &question, &typeAnswer, &point,
			&answerLabel, &answerValue, &answerText, &isCorrect, &respondent, &startedAt, &submittedAt,
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
		answerRows = append(answerRows, row)
	}
	return answerRows, rows.Err()
}
