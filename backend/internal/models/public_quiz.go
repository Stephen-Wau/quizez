package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	mathrand "math/rand"
	"strings"
	"time"
)

const (
	PublicQuizStateActive   = "active"
	PublicQuizStateUpcoming = "upcoming"
	PublicQuizStateExpired  = "expired"
	PublicQuizStateInactive = "inactive"
)

type QuizShare struct {
	ID     int64   `json:"id"`
	QuizID *int64  `json:"quiz_id"`
	Token  *string `json:"token"`
}

type PublicQuestion struct {
	ID         int64                  `json:"id"`
	Question   *string                `json:"question"`
	TypeAnswer *string                `json:"type_answer"`
	Answers    []PublicQuestionAnswer `json:"answers"`
}

type PublicQuestionAnswer struct {
	ID    int64   `json:"id"`
	Label *string `json:"label"`
}

type PublicQuiz struct {
	ID            int64            `json:"id"`
	Token         *string          `json:"token"`
	Title         *string          `json:"title"`
	Type          *string          `json:"type"`
	Description   *string          `json:"description"`
	StartTime     *string          `json:"start_time"`
	EndTime       *string          `json:"end_time"`
	MaxPoint      *int             `json:"max_point"`
	PassingGrade  *int             `json:"passing_grade"`
	TotalQuestion int              `json:"total_question"`
	Status        *string          `json:"status"`
	State         string           `json:"state"`
	ServerTime    string           `json:"server_time"`
	Questions     []PublicQuestion `json:"questions"`
}

type PublicSubmissionAnswerInput struct {
	QuestionID       int64
	QuestionAnswerID *int64
	AnswerText       *string
}

type PublicSubmissionResult struct {
	SubmissionID     int64                          `json:"submission_id"`
	Title            *string                        `json:"title"`
	Type             *string                        `json:"type"`
	Score            *int                           `json:"score"`
	MaxPoint         *int                           `json:"max_point"`
	PassingGrade     *int                           `json:"passing_grade"`
	ScorePercentage  *float64                       `json:"score_percentage"`
	Passed           *bool                          `json:"passed"`
	CorrectAnswers   int                            `json:"correct_answers"`
	AnsweredQuestions int                           `json:"answered_questions"`
	TotalQuestions   int                            `json:"total_questions"`
	SubmittedAt      string                         `json:"submitted_at"`
	Message          string                         `json:"message"`
	AnswerDetails    []PublicSubmissionAnswerResult `json:"answer_details"`
}

type PublicSubmissionAnswerResult struct {
	QuestionID          int64    `json:"question_id"`
	Question            *string  `json:"question"`
	TypeAnswer          *string  `json:"type_answer"`
	Point               *int     `json:"point"`
	SelectedAnswerLabel *string  `json:"selected_answer_label"`
	SelectedAnswerText  *string  `json:"selected_answer_text"`
	IsCorrect           *bool    `json:"is_correct"`
	CorrectAnswers      []string `json:"correct_answers"`
}

// GetOrCreateQuizShare ambil token share quiz yang sudah ada, atau bikin token acak baru kalau
// quiz ini belum pernah dibagikan sebelumnya.
func GetOrCreateQuizShare(db *sql.DB, quizID int64) (QuizShare, error) {
	var share QuizShare
	var token sql.NullString
	var storedQuizID sql.NullInt64

	err := db.QueryRow("SELECT id, quiz_id, token FROM quiz_shares WHERE quiz_id = ? LIMIT 1", quizID).
		Scan(&share.ID, &storedQuizID, &token)
	if err == nil {
		if storedQuizID.Valid {
			v := storedQuizID.Int64
			share.QuizID = &v
		}
		share.Token = nullableString(token)
		return share, nil
	}
	if err != sql.ErrNoRows {
		return QuizShare{}, err
	}

	for i := 0; i < 5; i++ {
		nextToken, err := generateShareToken()
		if err != nil {
			return QuizShare{}, err
		}

		res, err := db.Exec("INSERT INTO quiz_shares (quiz_id, token) VALUES (?, ?)", quizID, nextToken)
		if err != nil {
			// Token bentrok itu sangat jarang, tapi kalau kejadian cukup generate ulang.
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				continue
			}
			return QuizShare{}, err
		}

		shareID, err := res.LastInsertId()
		if err != nil {
			return QuizShare{}, err
		}
		share.ID = shareID
		share.QuizID = &quizID
		share.Token = &nextToken
		return share, nil
	}

	return QuizShare{}, fmt.Errorf("failed to generate unique share token")
}

// GetPublicQuizByToken cari quiz dari token publik, hitung state period-nya, lalu sanitasi semua
// question/answer supaya FE publik tidak menerima flag benar-salah asli dari database.
func GetPublicQuizByToken(db *sql.DB, token string, now time.Time) (PublicQuiz, error) {
	row := db.QueryRow(
		"SELECT q.id, q.title, q.type, q.start_time, q.end_time, q.description, q.max_point, q.passing_grade, "+
			"(SELECT COUNT(*) FROM questions WHERE quiz_id = q.id) AS total_question, q.status "+
			"FROM quizzes q INNER JOIN quiz_shares qs ON qs.quiz_id = q.id WHERE qs.token = ? LIMIT 1",
		token,
	)

	quiz, err := scanQuiz(row)
	if err != nil {
		return PublicQuiz{}, err
	}

	questions, err := ListQuestionsByQuiz(db, quiz.ID)
	if err != nil {
		return PublicQuiz{}, err
	}

	publicQuestions := make([]PublicQuestion, 0, len(questions))
	for _, question := range questions {
		answers := make([]PublicQuestionAnswer, 0, len(question.Answers))
		for _, answer := range question.Answers {
			answers = append(answers, PublicQuestionAnswer{
				ID:    answer.ID,
				Label: answer.Label,
			})
		}
		publicQuestions = append(publicQuestions, PublicQuestion{
			ID:         question.ID,
			Question:   question.Question,
			TypeAnswer: question.TypeAnswer,
			Answers:    answers,
		})
	}
	if stringPtrValue(quiz.Type) == "quiz" {
		shufflePublicQuestions(publicQuestions)
	}

	tokenCopy := token
	return PublicQuiz{
		ID:            quiz.ID,
		Token:         &tokenCopy,
		Title:         quiz.Title,
		Type:          quiz.Type,
		Description:   quiz.Description,
		StartTime:     quiz.StartTime,
		EndTime:       quiz.EndTime,
		MaxPoint:      quiz.MaxPoint,
		PassingGrade:  quiz.PassingGrade,
		TotalQuestion: quiz.TotalQuestion,
		Status:        quiz.Status,
		State:         ResolvePublicQuizState(quiz, now),
		ServerTime:    now.Format(dateTimeLayout),
		Questions:     publicQuestions,
	}, nil
}

// HasSubmittedEmail dipakai alur quiz publik untuk mencegah email yang sama submit dua kali.
func HasSubmittedEmail(db *sql.DB, quizID int64, email string) (bool, error) {
	var exists int
	err := db.QueryRow(
		"SELECT 1 FROM quiz_submissions WHERE quiz_id = ? AND LOWER(respondent_email) = ? LIMIT 1",
		quizID,
		strings.ToLower(strings.TrimSpace(email)),
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// SavePublicSubmission simpan satu submit publik beserta semua jawabannya dalam transaction yang
// sama, sekaligus menghitung score quiz pilihan ganda tanpa membocorkan kunci jawaban ke FE.
func SavePublicSubmission(db *sql.DB, quiz Quiz, email *string, answers []PublicSubmissionAnswerInput, startedAt *time.Time, now time.Time) (PublicSubmissionResult, error) {
	questions, err := ListQuestionsByQuiz(db, quiz.ID)
	if err != nil {
		return PublicSubmissionResult{}, err
	}
	isQuizSubmission := stringPtrValue(quiz.Type) == "quiz"

	questionByID := make(map[int64]Question, len(questions))
	for _, question := range questions {
		questionByID[question.ID] = question
	}

	if len(questionByID) == 0 {
		return PublicSubmissionResult{}, fmt.Errorf("Quiz ini belum punya question.")
	}

	inputByQuestionID := make(map[int64]PublicSubmissionAnswerInput, len(answers))
	for _, answer := range answers {
		if answer.QuestionID <= 0 {
			return PublicSubmissionResult{}, fmt.Errorf("Question wajib dipilih.")
		}
		if _, exists := inputByQuestionID[answer.QuestionID]; exists {
			return PublicSubmissionResult{}, fmt.Errorf("Setiap question hanya boleh diisi satu kali.")
		}
		question, exists := questionByID[answer.QuestionID]
		if !exists {
			return PublicSubmissionResult{}, fmt.Errorf("Ada jawaban yang tidak sesuai dengan question quiz ini.")
		}

		switch stringPtrValue(question.TypeAnswer) {
		case "multiple_choice", "rating":
			if !isQuizSubmission && (answer.QuestionAnswerID == nil || *answer.QuestionAnswerID <= 0) {
				return PublicSubmissionResult{}, fmt.Errorf("Semua question pilihan wajib dijawab.")
			}
		case "free_text":
			if !isQuizSubmission && (answer.AnswerText == nil || strings.TrimSpace(*answer.AnswerText) == "") {
				return PublicSubmissionResult{}, fmt.Errorf("Jawaban free text wajib diisi.")
			}
		}
		inputByQuestionID[answer.QuestionID] = answer
	}

	if !isQuizSubmission && len(inputByQuestionID) != len(questionByID) {
		return PublicSubmissionResult{}, fmt.Errorf("Semua question wajib dijawab.")
	}

	type storedAnswer struct {
		QuestionID       int64
		Question         *string
		TypeAnswer       *string
		Point            *int
		QuestionAnswerID *int64
		AnswerLabel      *string
		AnswerValue      *string
		AnswerText       *string
		IsCorrect        *bool
		CorrectAnswers   []string
	}

	storedAnswers := make([]storedAnswer, 0, len(questions))
	score := 0
	correctAnswers := 0
	answeredQuestions := 0
	for _, question := range questions {
		input, hasInput := inputByQuestionID[question.ID]
		if !hasInput && isQuizSubmission {
			storedAnswers = append(storedAnswers, storedAnswer{
				QuestionID: question.ID,
				Question:   question.Question,
				TypeAnswer: question.TypeAnswer,
				Point:      question.Point,
				CorrectAnswers: collectCorrectAnswerLabels(question),
			})
			continue
		}
		if !hasInput {
			return PublicSubmissionResult{}, fmt.Errorf("Semua question wajib dijawab.")
		}

		switch stringPtrValue(question.TypeAnswer) {
		case "free_text":
			if input.AnswerText == nil || strings.TrimSpace(*input.AnswerText) == "" {
				if isQuizSubmission {
					storedAnswers = append(storedAnswers, storedAnswer{
						QuestionID: question.ID,
						Question:   question.Question,
						TypeAnswer: question.TypeAnswer,
						Point:      question.Point,
						CorrectAnswers: collectCorrectAnswerLabels(question),
					})
					continue
				}
				return PublicSubmissionResult{}, fmt.Errorf("Jawaban free text wajib diisi.")
			}
			answerText := strings.TrimSpace(stringPtrValue(input.AnswerText))
			answeredQuestions++
			storedAnswers = append(storedAnswers, storedAnswer{
				QuestionID: question.ID,
				Question:   question.Question,
				TypeAnswer: question.TypeAnswer,
				Point:      question.Point,
				AnswerText: &answerText,
				CorrectAnswers: collectCorrectAnswerLabels(question),
			})
		default:
			if input.QuestionAnswerID == nil || *input.QuestionAnswerID <= 0 {
				if isQuizSubmission {
					storedAnswers = append(storedAnswers, storedAnswer{
						QuestionID: question.ID,
						Question:   question.Question,
						TypeAnswer: question.TypeAnswer,
						Point:      question.Point,
						CorrectAnswers: collectCorrectAnswerLabels(question),
					})
					continue
				}
				return PublicSubmissionResult{}, fmt.Errorf("Semua question pilihan wajib dijawab.")
			}
			selected, err := findQuestionAnswerByID(question.Answers, input.QuestionAnswerID)
			if err != nil {
				return PublicSubmissionResult{}, err
			}
			var isCorrect *bool
			if stringPtrValue(question.TypeAnswer) == "multiple_choice" {
				correct := strings.EqualFold(stringPtrValue(selected.Value), "true")
				isCorrect = &correct
				if correct {
					correctAnswers++
				}
				// Scoring otomatis hanya berlaku untuk pilihan ganda; tipe lain tetap tersimpan tapi
				// tidak menambah poin karena tidak ada rule benar/salah yang eksplisit.
				if correct && question.Point != nil {
					score += *question.Point
				}
			}
			answeredQuestions++
			storedAnswers = append(storedAnswers, storedAnswer{
				QuestionID:       question.ID,
				Question:         question.Question,
				TypeAnswer:       question.TypeAnswer,
				Point:            question.Point,
				QuestionAnswerID: &selected.ID,
				AnswerLabel:      selected.Label,
				AnswerValue:      selected.Value,
				IsCorrect:        isCorrect,
				CorrectAnswers:   collectCorrectAnswerLabels(question),
			})
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return PublicSubmissionResult{}, err
	}
	defer tx.Rollback()

	scoreValue := interface{}(nil)
	if isQuizSubmission {
		scoreValue = score
	}
	emailValue := interface{}(nil)
	if email != nil && strings.TrimSpace(*email) != "" {
		emailValue = strings.ToLower(strings.TrimSpace(*email))
	}
	startedAtValue := now
	if startedAt != nil && !startedAt.IsZero() && !startedAt.After(now) {
		startedAtValue = *startedAt
	}

	res, err := tx.Exec(
		"INSERT INTO quiz_submissions (quiz_id, respondent_email, score, started_at, submitted_at) VALUES (?, ?, ?, ?, ?)",
		quiz.ID, emailValue, scoreValue, startedAtValue, now,
	)
	if err != nil {
		return PublicSubmissionResult{}, err
	}
	submissionID, err := res.LastInsertId()
	if err != nil {
		return PublicSubmissionResult{}, err
	}

	stmt, err := tx.Prepare(
		"INSERT INTO quiz_submission_answers (submission_id, question_id, question_answer_id, answer_label, answer_value, answer_text, is_correct) VALUES (?, ?, ?, ?, ?, ?, ?)",
	)
	if err != nil {
		return PublicSubmissionResult{}, err
	}
	defer stmt.Close()

	for _, answer := range storedAnswers {
		if _, err := stmt.Exec(
			submissionID,
			answer.QuestionID,
			int64PtrValue(answer.QuestionAnswerID),
			strPtrValue(answer.AnswerLabel),
			strPtrValue(answer.AnswerValue),
			strPtrValue(answer.AnswerText),
			boolPtrValue(answer.IsCorrect),
		); err != nil {
			return PublicSubmissionResult{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return PublicSubmissionResult{}, err
	}

	var resultScore *int
	var passed *bool
	var scorePercentage *float64
	message := "Terima kasih, jawaban berhasil dikirim."
	if isQuizSubmission {
		resultScore = &score
		message = "Quiz selesai. Nilai kamu sudah dihitung."
		if quiz.MaxPoint != nil && *quiz.MaxPoint > 0 {
			percentage := roundFloat((float64(score) / float64(*quiz.MaxPoint)) * 100)
			scorePercentage = &percentage
		}
		if quiz.PassingGrade != nil {
			isPassed := score >= *quiz.PassingGrade
			passed = &isPassed
			if isPassed {
				message = "Selamat, kamu lulus passing grade quiz ini."
			} else {
				message = "Quiz selesai. Kamu belum mencapai passing grade."
			}
		}
	}

	answerDetails := make([]PublicSubmissionAnswerResult, 0, len(storedAnswers))
	for _, answer := range storedAnswers {
		answerDetails = append(answerDetails, PublicSubmissionAnswerResult{
			QuestionID:          answer.QuestionID,
			Question:            answer.Question,
			TypeAnswer:          answer.TypeAnswer,
			Point:               answer.Point,
			SelectedAnswerLabel: answer.AnswerLabel,
			SelectedAnswerText:  answer.AnswerText,
			IsCorrect:           answer.IsCorrect,
			CorrectAnswers:      answer.CorrectAnswers,
		})
	}

	return PublicSubmissionResult{
		SubmissionID:      submissionID,
		Title:             quiz.Title,
		Type:              quiz.Type,
		Score:             resultScore,
		MaxPoint:          quiz.MaxPoint,
		PassingGrade:      quiz.PassingGrade,
		ScorePercentage:   scorePercentage,
		Passed:            passed,
		CorrectAnswers:    correctAnswers,
		AnsweredQuestions: answeredQuestions,
		TotalQuestions:    len(questions),
		SubmittedAt:       now.Format(dateTimeLayout),
		Message:           message,
		AnswerDetails:     answerDetails,
	}, nil
}

// ResolvePublicQuizState ubah period quiz/survey menjadi status publik yang gampang dipakai FE:
// inactive, upcoming, active, atau expired.
func ResolvePublicQuizState(quiz Quiz, now time.Time) string {
	if !strings.EqualFold(stringPtrValue(quiz.Status), "active") {
		return PublicQuizStateInactive
	}

	if strings.EqualFold(stringPtrValue(quiz.Type), "quiz") {
		startTime, endTime, err := resolveQuizDailyWindow(quiz, now)
		if err != nil {
			return PublicQuizStateInactive
		}
		if now.Before(startTime) {
			return PublicQuizStateUpcoming
		}
		if now.After(endTime) {
			return PublicQuizStateExpired
		}
		return PublicQuizStateActive
	}

	startTime, err := parseQuizDateTime(quiz.StartTime)
	if err == nil && now.Before(startTime) {
		return PublicQuizStateUpcoming
	}

	endTime, err := parseQuizDateTime(quiz.EndTime)
	if err == nil && now.After(endTime) {
		return PublicQuizStateExpired
	}

	return PublicQuizStateActive
}

// resolveQuizDailyWindow ubah start/end datetime quiz menjadi window waktu di tanggal "hari ini",
// karena quiz admin memang hanya memakai jam tanpa konteks tanggal tetap.
func resolveQuizDailyWindow(quiz Quiz, now time.Time) (time.Time, time.Time, error) {
	startSource, err := parseQuizDateTime(quiz.StartTime)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	endSource, err := parseQuizDateTime(quiz.EndTime)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	startTime := time.Date(
		now.Year(), now.Month(), now.Day(),
		startSource.Hour(), startSource.Minute(), startSource.Second(), 0,
		now.Location(),
	)
	endTime := time.Date(
		now.Year(), now.Month(), now.Day(),
		endSource.Hour(), endSource.Minute(), endSource.Second(), 0,
		now.Location(),
	)
	return startTime, endTime, nil
}

// parseQuizDateTime parse datetime string DB ke time.Time. Nil/empty dianggap tidak ada constraint.
func parseQuizDateTime(value *string) (time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return time.Time{}, fmt.Errorf("empty datetime")
	}
	return time.ParseInLocation(dateTimeLayout, *value, time.Local)
}

// generateShareToken bikin token acak yang cukup panjang buat dipakai sebagai URL publik.
func generateShareToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// findQuestionAnswerByID memastikan opsi jawaban yang dipilih publik memang milik question terkait.
func findQuestionAnswerByID(answers []QuestionAnswer, answerID *int64) (QuestionAnswer, error) {
	if answerID == nil {
		return QuestionAnswer{}, fmt.Errorf("Jawaban pilihan wajib diisi.")
	}
	for _, answer := range answers {
		if answer.ID == *answerID {
			return answer, nil
		}
	}
	return QuestionAnswer{}, fmt.Errorf("Ada jawaban yang tidak valid untuk question ini.")
}

// collectCorrectAnswerLabels disiapkan untuk result page publik dan detail submission admin
// supaya user/admin bisa melihat kunci jawaban yang benar untuk soal pilihan ganda.
func collectCorrectAnswerLabels(question Question) []string {
	if stringPtrValue(question.TypeAnswer) != "multiple_choice" {
		return []string{}
	}

	labels := []string{}
	for _, answer := range question.Answers {
		if strings.EqualFold(stringPtrValue(answer.Value), "true") {
			label := strings.TrimSpace(stringPtrValue(answer.Label))
			if label != "" {
				labels = append(labels, label)
			}
		}
	}
	return labels
}

// shufflePublicQuestions hanya berlaku untuk quiz publik agar user tidak selalu melihat urutan
// soal/opsi yang sama. Survey dibiarkan stabil sesuai susunan admin.
func shufflePublicQuestions(questions []PublicQuestion) {
	randomizer := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	for i := range questions {
		randomizer.Shuffle(len(questions[i].Answers), func(a, b int) {
			questions[i].Answers[a], questions[i].Answers[b] = questions[i].Answers[b], questions[i].Answers[a]
		})
	}
	randomizer.Shuffle(len(questions), func(i, j int) {
		questions[i], questions[j] = questions[j], questions[i]
	})
}

// boolPtrValue konversi *bool ke argumen SQL; nil pointer jadi SQL NULL.
func boolPtrValue(v *bool) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

// stringPtrValue baca *string dengan aman; nil pointer jadi string kosong.
func stringPtrValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
