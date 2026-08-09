package models

import (
	"database/sql"
	"math"
	"sort"
	"strconv"
	"strings"
)

type QuizSummary struct {
	Quiz                Quiz                 `json:"quiz"`
	Stats               QuizSummaryStats     `json:"stats"`
	ScoreDistribution   []SummaryBucket      `json:"score_distribution"`
	QuestionSummaries   []QuestionSummary    `json:"question_summaries"`
	SubmissionSummaries []SubmissionSummary  `json:"submission_summaries"`
}

type QuizSubmissionDetail struct {
	Quiz       Quiz              `json:"quiz"`
	Submission SubmissionSummary `json:"submission"`
}

type QuizSummaryStats struct {
	TotalSubmissions   int      `json:"total_submissions"`
	UniqueRespondents  int      `json:"unique_respondents"`
	AverageScore       *float64 `json:"average_score"`
	HighestScore       *int     `json:"highest_score"`
	LowestScore        *int     `json:"lowest_score"`
	AverageCompletion  float64  `json:"average_completion"`
	LatestSubmissionAt *string  `json:"latest_submission_at"`
	PassingCount       int      `json:"passing_count"`
	FailingCount       int      `json:"failing_count"`
	PassingRate        *float64 `json:"passing_rate"`
}

type SummaryBucket struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type QuestionSummary struct {
	QuestionID      int64                  `json:"question_id"`
	Question        *string                `json:"question"`
	TypeAnswer      *string                `json:"type_answer"`
	Point           *int                   `json:"point"`
	TotalResponses  int                    `json:"total_responses"`
	CorrectCount    int                    `json:"correct_count"`
	IncorrectCount  int                    `json:"incorrect_count"`
	AverageRating   *float64               `json:"average_rating"`
	OptionSummaries []QuestionOptionSummary `json:"option_summaries"`
	TextResponses   []QuestionTextResponse `json:"text_responses"`
}

type QuestionOptionSummary struct {
	Label      string  `json:"label"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type QuestionTextResponse struct {
	RespondentEmail *string `json:"respondent_email"`
	AnswerText      string  `json:"answer_text"`
	SubmittedAt     *string `json:"submitted_at"`
}

type SubmissionSummary struct {
	ID                   int64                    `json:"id"`
	RespondentEmail      *string                  `json:"respondent_email"`
	Score                *int                     `json:"score"`
	PassingGrade         *int                     `json:"passing_grade"`
	Passed               *bool                    `json:"passed"`
	ScorePercentage      *float64                 `json:"score_percentage"`
	CorrectAnswers       int                      `json:"correct_answers"`
	IncorrectAnswers     int                      `json:"incorrect_answers"`
	TotalQuestions       int                      `json:"total_questions"`
	StartedAt            *string                  `json:"started_at"`
	SubmittedAt          *string                  `json:"submitted_at"`
	CompletionPercentage float64                  `json:"completion_percentage"`
	Answers              []SubmissionAnswerSummary `json:"answers"`
}

type SubmissionAnswerSummary struct {
	QuestionID      int64    `json:"question_id"`
	Question        *string  `json:"question"`
	TypeAnswer      *string  `json:"type_answer"`
	Point           *int     `json:"point"`
	AnswerLabel     *string  `json:"answer_label"`
	AnswerText      *string  `json:"answer_text"`
	IsCorrect       *bool    `json:"is_correct"`
	CorrectAnswers  []string `json:"correct_answers"`
}

type summaryAnswerRow struct {
	SubmissionID    int64
	QuestionID      int64
	Question        *string
	TypeAnswer      *string
	Point           *int
	AnswerLabel     *string
	AnswerValue     *string
	AnswerText      *string
	IsCorrect       *bool
	RespondentEmail *string
	StartedAt       *string
	SubmittedAt     *string
}

// GetQuizSummary rangkum analytics quiz/survey dari submission yang sudah terkumpul: KPI umum,
// distribusi skor, ringkasan per-question, dan tabel jawaban per submission.
func GetQuizSummary(db *sql.DB, quizID int64) (QuizSummary, error) {
	quiz, err := GetQuizByID(db, quizID)
	if err != nil {
		return QuizSummary{}, err
	}

	questions, err := ListQuestionsByQuiz(db, quizID)
	if err != nil {
		return QuizSummary{}, err
	}

	submissions, err := listSubmissionSummaries(db, quizID)
	if err != nil {
		return QuizSummary{}, err
	}

	answerRows, err := listSubmissionAnswerRows(db, quizID)
	if err != nil {
		return QuizSummary{}, err
	}

	attachAnswersToSubmissions(quiz, questions, submissions, answerRows, len(questions))
	stats := buildQuizSummaryStats(quiz, submissions)
	questionSummaries := buildQuestionSummaries(questions, answerRows)

	return QuizSummary{
		Quiz:                quiz,
		Stats:               stats,
		ScoreDistribution:   buildScoreDistribution(quiz, submissions),
		QuestionSummaries:   questionSummaries,
		SubmissionSummaries: submissions,
	}, nil
}

// GetQuizSubmissionDetail ambil 1 submission spesifik milik quiz beserta seluruh jawaban detailnya.
// Untuk ukuran data saat ini, detail diturunkan dari summary builder yang sama supaya rule analytic
// dan perhitungan skor/correctness tetap satu sumber.
func GetQuizSubmissionDetail(db *sql.DB, quizID int64, submissionID int64) (QuizSubmissionDetail, error) {
	summary, err := GetQuizSummary(db, quizID)
	if err != nil {
		return QuizSubmissionDetail{}, err
	}

	for _, submission := range summary.SubmissionSummaries {
		if submission.ID == submissionID {
			return QuizSubmissionDetail{
				Quiz:       summary.Quiz,
				Submission: submission,
			}, nil
		}
	}

	return QuizSubmissionDetail{}, sql.ErrNoRows
}

// listSubmissionSummaries ambil daftar submission inti untuk 1 quiz, diurutkan terbaru dulu supaya
// tabel summary FE langsung enak dipindai tanpa sort tambahan.
func listSubmissionSummaries(db *sql.DB, quizID int64) ([]SubmissionSummary, error) {
	rows, err := db.Query(
		"SELECT id, respondent_email, score, started_at, submitted_at FROM quiz_submissions WHERE quiz_id = ? ORDER BY submitted_at DESC, id DESC",
		quizID,
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

// listSubmissionAnswerRows ambil semua jawaban submission lengkap dengan metadata question dan
// submission supaya analytic per-question dan per-user bisa dibangun dari satu sumber data yang sama.
func listSubmissionAnswerRows(db *sql.DB, quizID int64) ([]summaryAnswerRow, error) {
	rows, err := db.Query(
		`SELECT
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
		WHERE qs.quiz_id = ?
		ORDER BY qs.submitted_at DESC, qsa.submission_id DESC, qsa.question_id ASC`,
		quizID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	answerRows := []summaryAnswerRow{}
	for rows.Next() {
		var (
			row            summaryAnswerRow
			question       sql.NullString
			typeAnswer     sql.NullString
			answerLabel    sql.NullString
			answerValue    sql.NullString
			answerText     sql.NullString
			respondent     sql.NullString
			startedAt      sql.NullTime
			submittedAt    sql.NullTime
			point          sql.NullInt64
			isCorrect      sql.NullBool
		)
		if err := rows.Scan(
			&row.SubmissionID,
			&row.QuestionID,
			&question,
			&typeAnswer,
			&point,
			&answerLabel,
			&answerValue,
			&answerText,
			&isCorrect,
			&respondent,
			&startedAt,
			&submittedAt,
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

// buildQuizSummaryStats hitung KPI level atas seperti total submission, unique respondent, dan
// statistik skor/completion supaya FE bisa render cards tanpa hitung ulang.
func buildQuizSummaryStats(quiz Quiz, submissions []SubmissionSummary) QuizSummaryStats {
	stats := QuizSummaryStats{
		TotalSubmissions: len(submissions),
	}
	if len(submissions) == 0 {
		return stats
	}

	uniqueRespondents := map[string]bool{}
	scoreTotal := 0
	scoreCount := 0
	completionTotal := 0.0

	for index, submission := range submissions {
		completionTotal += submission.CompletionPercentage

		if submission.RespondentEmail != nil && strings.TrimSpace(*submission.RespondentEmail) != "" {
			uniqueRespondents[strings.ToLower(strings.TrimSpace(*submission.RespondentEmail))] = true
		}
		if submission.Score != nil {
			scoreTotal += *submission.Score
			scoreCount++

			if stats.HighestScore == nil || *submission.Score > *stats.HighestScore {
				v := *submission.Score
				stats.HighestScore = &v
			}
			if stats.LowestScore == nil || *submission.Score < *stats.LowestScore {
				v := *submission.Score
				stats.LowestScore = &v
			}
		}
		if index == 0 {
			stats.LatestSubmissionAt = submission.SubmittedAt
		}
	}

	// Quiz biasanya punya email unik, survey bisa kosong; kalau semuanya kosong, total submission
	// lebih representatif untuk kartu "respondent".
	if len(uniqueRespondents) > 0 {
		stats.UniqueRespondents = len(uniqueRespondents)
	} else {
		stats.UniqueRespondents = len(submissions)
	}

	if scoreCount > 0 && quiz.Type != nil && *quiz.Type == "quiz" {
		avg := roundFloat(float64(scoreTotal) / float64(scoreCount))
		stats.AverageScore = &avg
	}
	stats.AverageCompletion = roundFloat(completionTotal / float64(len(submissions)))

	if quiz.Type != nil && *quiz.Type == "quiz" {
		passingCount := 0
		failingCount := 0
		for _, submission := range submissions {
			if submission.Passed == nil {
				continue
			}
			if *submission.Passed {
				passingCount++
			} else {
				failingCount++
			}
		}
		stats.PassingCount = passingCount
		stats.FailingCount = failingCount
		if passingCount+failingCount > 0 {
			rate := roundFloat((float64(passingCount) / float64(passingCount+failingCount)) * 100)
			stats.PassingRate = &rate
		}
	}
	return stats
}

// buildScoreDistribution kelompokkan skor quiz ke 5 bucket persentase yang gampang dibaca.
func buildScoreDistribution(quiz Quiz, submissions []SubmissionSummary) []SummaryBucket {
	if quiz.Type == nil || *quiz.Type != "quiz" || quiz.MaxPoint == nil || *quiz.MaxPoint <= 0 {
		return []SummaryBucket{}
	}

	buckets := []SummaryBucket{
		{Label: "0-20%", Count: 0},
		{Label: "21-40%", Count: 0},
		{Label: "41-60%", Count: 0},
		{Label: "61-80%", Count: 0},
		{Label: "81-100%", Count: 0},
	}

	for _, submission := range submissions {
		if submission.Score == nil {
			continue
		}
		percentage := (float64(*submission.Score) / float64(*quiz.MaxPoint)) * 100
		switch {
		case percentage <= 20:
			buckets[0].Count++
		case percentage <= 40:
			buckets[1].Count++
		case percentage <= 60:
			buckets[2].Count++
		case percentage <= 80:
			buckets[3].Count++
		default:
			buckets[4].Count++
		}
	}

	return buckets
}

// buildQuestionSummaries rangkum analytics per-question: distribusi opsi, rating rata-rata,
// jumlah benar/salah, dan kumpulan jawaban free text.
func buildQuestionSummaries(questions []Question, answerRows []summaryAnswerRow) []QuestionSummary {
	summaries := make([]QuestionSummary, 0, len(questions))
	summaryMap := map[int64]*QuestionSummary{}
	optionCounters := map[int64]map[string]int{}
	ratingTotals := map[int64]int{}
	ratingCounts := map[int64]int{}

	for _, question := range questions {
		summary := QuestionSummary{
			QuestionID:      question.ID,
			Question:        question.Question,
			TypeAnswer:      question.TypeAnswer,
			Point:           question.Point,
			OptionSummaries: []QuestionOptionSummary{},
			TextResponses:   []QuestionTextResponse{},
		}
		summaries = append(summaries, summary)
		summaryMap[question.ID] = &summaries[len(summaries)-1]
		optionCounters[question.ID] = map[string]int{}
	}

	for _, row := range answerRows {
		summary := summaryMap[row.QuestionID]
		if summary == nil {
			continue
		}

		if isAnsweredRow(row) {
			summary.TotalResponses++
		}
		if row.IsCorrect != nil {
			if *row.IsCorrect {
				summary.CorrectCount++
			} else {
				summary.IncorrectCount++
			}
		}

		switch stringPtrValue(row.TypeAnswer) {
		case "multiple_choice", "rating":
			label := strings.TrimSpace(stringPtrValue(row.AnswerLabel))
			if label == "" {
				label = strings.TrimSpace(stringPtrValue(row.AnswerValue))
			}
			if label != "" {
				optionCounters[row.QuestionID][label]++
			}
			if stringPtrValue(row.TypeAnswer) == "rating" {
				if n, err := strconv.Atoi(strings.TrimSpace(stringPtrValue(row.AnswerValue))); err == nil {
					ratingTotals[row.QuestionID] += n
					ratingCounts[row.QuestionID]++
				}
			}
		case "free_text":
			text := strings.TrimSpace(stringPtrValue(row.AnswerText))
			if text != "" {
				summary.TextResponses = append(summary.TextResponses, QuestionTextResponse{
					RespondentEmail: row.RespondentEmail,
					AnswerText:      text,
					SubmittedAt:     row.SubmittedAt,
				})
			}
		}
	}

	for i := range summaries {
		summary := &summaries[i]
		counter := optionCounters[summary.QuestionID]
		labels := make([]string, 0, len(counter))
		for label := range counter {
			labels = append(labels, label)
		}
		sort.Strings(labels)

		for _, label := range labels {
			count := counter[label]
			percentage := 0.0
			if summary.TotalResponses > 0 {
				percentage = roundFloat((float64(count) / float64(summary.TotalResponses)) * 100)
			}
			summary.OptionSummaries = append(summary.OptionSummaries, QuestionOptionSummary{
				Label:      label,
				Count:      count,
				Percentage: percentage,
			})
		}
		if ratingCounts[summary.QuestionID] > 0 {
			avg := roundFloat(float64(ratingTotals[summary.QuestionID]) / float64(ratingCounts[summary.QuestionID]))
			summary.AverageRating = &avg
		}
	}

	return summaries
}

// attachAnswersToSubmissions tempelkan jawaban ke masing-masing submission sekaligus hitung
// completion percentage per-user berdasarkan berapa question yang benar-benar terisi.
func attachAnswersToSubmissions(quiz Quiz, questions []Question, submissions []SubmissionSummary, answerRows []summaryAnswerRow, totalQuestions int) {
	submissionMap := map[int64]*SubmissionSummary{}
	answeredCounters := map[int64]int{}
	correctAnswersByQuestion := map[int64][]string{}
	for _, question := range questions {
		correctAnswersByQuestion[question.ID] = collectCorrectAnswerLabels(question)
	}
	for i := range submissions {
		submissionMap[submissions[i].ID] = &submissions[i]
		submissions[i].Answers = []SubmissionAnswerSummary{}
		submissions[i].TotalQuestions = totalQuestions
		submissions[i].PassingGrade = quiz.PassingGrade
	}

	for _, row := range answerRows {
		submission := submissionMap[row.SubmissionID]
		if submission == nil {
			continue
		}
		submission.Answers = append(submission.Answers, SubmissionAnswerSummary{
			QuestionID:     row.QuestionID,
			Question:       row.Question,
			TypeAnswer:     row.TypeAnswer,
			Point:          row.Point,
			AnswerLabel:    row.AnswerLabel,
			AnswerText:     row.AnswerText,
			IsCorrect:      row.IsCorrect,
			CorrectAnswers: correctAnswersByQuestion[row.QuestionID],
		})
		if isAnsweredRow(row) {
			answeredCounters[row.SubmissionID]++
		}
		if row.IsCorrect != nil {
			if *row.IsCorrect {
				submission.CorrectAnswers++
			} else {
				submission.IncorrectAnswers++
			}
		}
	}

	for i := range submissions {
		if totalQuestions <= 0 {
			submissions[i].CompletionPercentage = 0
		} else {
			submissions[i].CompletionPercentage = roundFloat((float64(answeredCounters[submissions[i].ID]) / float64(totalQuestions)) * 100)
		}
		if quiz.MaxPoint != nil && *quiz.MaxPoint > 0 && submissions[i].Score != nil {
			percentage := roundFloat((float64(*submissions[i].Score) / float64(*quiz.MaxPoint)) * 100)
			submissions[i].ScorePercentage = &percentage
		}
		if quiz.PassingGrade != nil && submissions[i].Score != nil {
			passed := *submissions[i].Score >= *quiz.PassingGrade
			submissions[i].Passed = &passed
		}
	}
}

// isAnsweredRow bantu bedain jawaban kosong hasil auto-submit vs jawaban yang benar-benar diisi user.
func isAnsweredRow(row summaryAnswerRow) bool {
	return strings.TrimSpace(stringPtrValue(row.AnswerLabel)) != "" ||
		strings.TrimSpace(stringPtrValue(row.AnswerText)) != ""
}

// roundFloat rapikan angka desimal analytic ke 2 digit supaya card/chart FE lebih rapi.
func roundFloat(value float64) float64 {
	return math.Round(value*100) / 100
}
