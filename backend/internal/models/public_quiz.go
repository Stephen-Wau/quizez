package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	mathrand "math/rand"
	"sort"
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
	ID         int64   `json:"id"`
	QuizID     *int64  `json:"quiz_id"`
	Token      *string `json:"token"`
	AccessCode *string `json:"access_code"`
}

type PublicQuestion struct {
	ID         int64                  `json:"id"`
	Question   *string                `json:"question"`
	TypeAnswer *string                `json:"type_answer"`
	Answers    []PublicQuestionAnswer `json:"answers"`
	// MatrixRows cuma keisi buat type_answer="matrix", Answers berperan sebagai kolom skala yang sama.
	MatrixRows []PublicMatrixRow `json:"matrix_rows"`
}

type PublicQuestionAnswer struct {
	ID    int64   `json:"id"`
	Label *string `json:"label"`
}

// PublicMatrixRow versi publik dari QuestionMatrixRow (tanpa metadata internal).
type PublicMatrixRow struct {
	ID       int64   `json:"id"`
	RowLabel *string `json:"row_label"`
}

type PublicQuiz struct {
	ID           int64   `json:"id"`
	Token        *string `json:"token"`
	Title        *string `json:"title"`
	Type         *string `json:"type"`
	Description  *string `json:"description"`
	StartTime    *string `json:"start_time"`
	EndTime      *string `json:"end_time"`
	MaxPoint     *int    `json:"max_point"`
	PassingGrade *int    `json:"passing_grade"`
	// RandomQuestionCount diteruskan biar handler submit bisa recompute subset yang sama tanpa query ulang ke DB.
	RandomQuestionCount *int             `json:"-"`
	LockMode            bool             `json:"lock_mode"`
	TotalQuestion       int              `json:"total_question"`
	Status              *string          `json:"status"`
	State               string           `json:"state"`
	ServerTime          string           `json:"server_time"`
	AccessCodeRequired  bool             `json:"access_code_required"`
	AccessGranted       bool             `json:"access_granted"`
	AccessMessage       *string          `json:"access_message"`
	Questions           []PublicQuestion `json:"questions"`
}

type PublicSubmissionAnswerInput struct {
	QuestionID       int64
	QuestionAnswerID *int64
	AnswerText       *string
	// SelectedAnswerIDs dipakai buat type_answer="checkbox" (bisa pilih lebih dari 1 opsi sekaligus).
	SelectedAnswerIDs []int64
	// MatrixAnswers dipakai buat type_answer="matrix": 1 entri per baris pernyataan yang dijawab.
	MatrixAnswers []PublicMatrixAnswerInput
}

// PublicMatrixAnswerInput jawaban 1 baris pernyataan pada question matrix: baris mana, pilih
// kolom/skala yang mana.
type PublicMatrixAnswerInput struct {
	RowID            int64
	QuestionAnswerID int64
}

type PublicSubmissionResult struct {
	SubmissionID      int64                          `json:"submission_id"`
	Title             *string                        `json:"title"`
	Type              *string                        `json:"type"`
	Score             *int                           `json:"score"`
	MaxPoint          *int                           `json:"max_point"`
	PassingGrade      *int                           `json:"passing_grade"`
	ScorePercentage   *float64                       `json:"score_percentage"`
	Passed            *bool                          `json:"passed"`
	CorrectAnswers    int                            `json:"correct_answers"`
	AnsweredQuestions int                            `json:"answered_questions"`
	TotalQuestions    int                            `json:"total_questions"`
	SubmittedAt       string                         `json:"submitted_at"`
	Message           string                         `json:"message"`
	AnswerDetails     []PublicSubmissionAnswerResult `json:"answer_details"`
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
	var accessCode sql.NullString
	var storedQuizID sql.NullInt64

	err := db.QueryRow("SELECT id, quiz_id, token, access_code FROM quiz_shares WHERE quiz_id = ? LIMIT 1", quizID).
		Scan(&share.ID, &storedQuizID, &token, &accessCode)
	if err == nil {
		if storedQuizID.Valid {
			v := storedQuizID.Int64
			share.QuizID = &v
		}
		share.Token = nullableString(token)
		share.AccessCode = nullableString(accessCode)
		if share.AccessCode == nil || strings.TrimSpace(stringPtrValue(share.AccessCode)) == "" {
			code, err := generateAccessCode(db)
			if err != nil {
				return QuizShare{}, err
			}
			if _, err := db.Exec("UPDATE quiz_shares SET access_code = ? WHERE id = ?", code, share.ID); err != nil {
				return QuizShare{}, err
			}
			share.AccessCode = &code
		}
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

		accessCode, err := generateAccessCode(db)
		if err != nil {
			return QuizShare{}, err
		}

		res, err := db.Exec("INSERT INTO quiz_shares (quiz_id, token, access_code) VALUES (?, ?, ?)", quizID, nextToken, accessCode)
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
		share.AccessCode = &accessCode
		return share, nil
	}

	return QuizShare{}, fmt.Errorf("failed to generate unique share token")
}

// GetPublicQuizByToken cari quiz dari token publik, hitung state period-nya, lalu sanitasi semua
// question/answer supaya FE publik tidak menerima flag benar-salah asli dari database.
// attemptSeed dipakai untuk memilih subset random_question_count secara deterministic per sesi
// responden (browser generate sekali, dikirim ulang tiap GET/submit) supaya subset soal yang
// ditampilkan gak berubah-ubah tiap kali form di-reload.
func GetPublicQuizByToken(db *sql.DB, token string, accessCode *string, now time.Time, attemptSeed string) (PublicQuiz, error) {
	row := db.QueryRow(
		"SELECT q.id, q.title, q.type, q.start_time, q.end_time, q.description, q.max_point, q.passing_grade, q.random_question_count, q.lock_mode, "+
			"(SELECT COUNT(*) FROM questions WHERE quiz_id = q.id) AS total_question, q.status, q.duplicated_from_id "+
			"FROM quizzes q INNER JOIN quiz_shares qs ON qs.quiz_id = q.id WHERE qs.token = ? LIMIT 1",
		token,
	)

	quiz, err := scanQuiz(row)
	if err != nil {
		return PublicQuiz{}, err
	}
	// Auto-close survey yang udah lewat end_time begitu diakses publik, biar status "closed" langsung
	// konsisten tanpa perlu admin buka CMS dulu.
	quiz, err = AutoCloseSurveyIfExpired(db, quiz, now)
	if err != nil {
		return PublicQuiz{}, err
	}

	share, err := getQuizShareByToken(db, token)
	if err != nil {
		return PublicQuiz{}, err
	}

	accessRequired := share.AccessCode != nil && strings.TrimSpace(stringPtrValue(share.AccessCode)) != ""
	accessGranted := !accessRequired
	if accessRequired {
		accessGranted = strings.EqualFold(
			strings.TrimSpace(stringPtrValue(share.AccessCode)),
			strings.TrimSpace(stringPtrValue(accessCode)),
		)
	}

	publicQuestions := []PublicQuestion{}
	totalQuestionForResponse := quiz.TotalQuestion
	if accessGranted {
		questions, err := ListQuestionsByQuiz(db, quiz.ID)
		if err != nil {
			return PublicQuiz{}, err
		}
		// Random subset cuma diterapkan kalau admin set random_question_count dan nilainya lebih
		// kecil dari total pool -- kalau >= total, sama aja tampilkan semua jadi diabaikan.
		questions = selectRandomQuestionSubset(questions, quiz.RandomQuestionCount, attemptSeed)
		totalQuestionForResponse = len(questions)

		publicQuestions = make([]PublicQuestion, 0, len(questions))
		for _, question := range questions {
			answers := make([]PublicQuestionAnswer, 0, len(question.Answers))
			for _, answer := range question.Answers {
				answers = append(answers, PublicQuestionAnswer{
					ID:    answer.ID,
					Label: answer.Label,
				})
			}
			matrixRows := make([]PublicMatrixRow, 0, len(question.MatrixRows))
			for _, row := range question.MatrixRows {
				matrixRows = append(matrixRows, PublicMatrixRow{ID: row.ID, RowLabel: row.RowLabel})
			}

			publicQuestions = append(publicQuestions, PublicQuestion{
				ID:         question.ID,
				Question:   question.Question,
				TypeAnswer: question.TypeAnswer,
				Answers:    answers,
				MatrixRows: matrixRows,
			})
		}
		if stringPtrValue(quiz.Type) == "quiz" {
			shufflePublicQuestions(publicQuestions)
		}
	}

	tokenCopy := token
	accessMessage := (*string)(nil)
	if accessRequired && !accessGranted {
		msg := "Masukkan PIN akses yang benar untuk membuka form ini."
		accessMessage = &msg
	}
	return PublicQuiz{
		ID:                  quiz.ID,
		Token:               &tokenCopy,
		Title:               quiz.Title,
		Type:                quiz.Type,
		Description:         quiz.Description,
		StartTime:           quiz.StartTime,
		EndTime:             quiz.EndTime,
		MaxPoint:            quiz.MaxPoint,
		PassingGrade:        quiz.PassingGrade,
		RandomQuestionCount: quiz.RandomQuestionCount,
		LockMode:            quiz.LockMode,
		TotalQuestion:       totalQuestionForResponse,
		Status:              quiz.Status,
		State:               ResolvePublicQuizState(quiz, now),
		ServerTime:          now.Format(dateTimeLayout),
		AccessCodeRequired:  accessRequired,
		AccessGranted:       accessGranted,
		AccessMessage:       accessMessage,
		Questions:           publicQuestions,
	}, nil
}

func GetQuizShareByToken(db *sql.DB, token string) (QuizShare, error) {
	return getQuizShareByToken(db, token)
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

// HasSubmittedFingerprint dipakai alur quiz publik (anti-cheat) untuk mencegah 1 device submit
// quiz yang sama dua kali walau pakai email berbeda-beda.
func HasSubmittedFingerprint(db *sql.DB, quizID int64, fingerprint string) (bool, error) {
	var exists int
	err := db.QueryRow(
		"SELECT 1 FROM quiz_submissions WHERE quiz_id = ? AND device_fingerprint = ? LIMIT 1",
		quizID,
		strings.TrimSpace(fingerprint),
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func ValidateQuizShareAccessCode(share QuizShare, accessCode *string) bool {
	if share.AccessCode == nil || strings.TrimSpace(stringPtrValue(share.AccessCode)) == "" {
		return true
	}
	return strings.EqualFold(
		strings.TrimSpace(stringPtrValue(share.AccessCode)),
		strings.TrimSpace(stringPtrValue(accessCode)),
	)
}

// SavePublicSubmission simpan satu submit publik beserta semua jawabannya dalam transaction yang
// sama, sekaligus menghitung score quiz pilihan ganda tanpa membocorkan kunci jawaban ke FE.
func SavePublicSubmission(db *sql.DB, quiz Quiz, email *string, answers []PublicSubmissionAnswerInput, startedAt *time.Time, now time.Time, attemptSeed string, deviceFingerprint *string, violationCount int) (PublicSubmissionResult, error) {
	questions, err := ListQuestionsByQuiz(db, quiz.ID)
	if err != nil {
		return PublicSubmissionResult{}, err
	}
	// Pool question dibatasi ke subset random yang sama persis dengan yang ditampilkan ke responden
	// (dihitung ulang dari attemptSeed, bukan percaya question_id yang dikirim client) supaya requirement
	// "wajib jawab semua" dan scoring cuma berlaku untuk soal yang benar-benar dia lihat.
	questions = selectRandomQuestionSubset(questions, quiz.RandomQuestionCount, attemptSeed)
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
		case "multiple_choice", "dropdown", "rating":
			if !isQuizSubmission && (answer.QuestionAnswerID == nil || *answer.QuestionAnswerID <= 0) {
				return PublicSubmissionResult{}, fmt.Errorf("Semua question pilihan wajib dijawab.")
			}
		case "checkbox":
			if !isQuizSubmission && len(answer.SelectedAnswerIDs) == 0 {
				return PublicSubmissionResult{}, fmt.Errorf("Semua question checkbox wajib dipilih minimal 1 opsi.")
			}
		case "matrix":
			// Survey wajib jawab semua baris; quiz (jarang dipakai buat matrix) boleh sebagian.
			if !isQuizSubmission && len(answer.MatrixAnswers) < len(question.MatrixRows) {
				return PublicSubmissionResult{}, fmt.Errorf("Semua baris matrix wajib diisi.")
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
		QuestionID        int64
		Question          *string
		TypeAnswer        *string
		Point             *int
		QuestionAnswerID  *int64
		SelectedAnswerIDs []int64
		MatrixRowID       *int64
		AnswerLabel       *string
		AnswerValue       *string
		AnswerText        *string
		IsCorrect         *bool
		CorrectAnswers    []string
	}

	storedAnswers := make([]storedAnswer, 0, len(questions))
	score := 0
	correctAnswers := 0
	answeredQuestions := 0
	for _, question := range questions {
		input, hasInput := inputByQuestionID[question.ID]
		if !hasInput && isQuizSubmission {
			storedAnswers = append(storedAnswers, storedAnswer{
				QuestionID:     question.ID,
				Question:       question.Question,
				TypeAnswer:     question.TypeAnswer,
				Point:          question.Point,
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
						QuestionID:     question.ID,
						Question:       question.Question,
						TypeAnswer:     question.TypeAnswer,
						Point:          question.Point,
						CorrectAnswers: collectCorrectAnswerLabels(question),
					})
					continue
				}
				return PublicSubmissionResult{}, fmt.Errorf("Jawaban free text wajib diisi.")
			}
			answerText := strings.TrimSpace(stringPtrValue(input.AnswerText))
			answeredQuestions++
			storedAnswers = append(storedAnswers, storedAnswer{
				QuestionID:     question.ID,
				Question:       question.Question,
				TypeAnswer:     question.TypeAnswer,
				Point:          question.Point,
				AnswerText:     &answerText,
				CorrectAnswers: collectCorrectAnswerLabels(question),
			})
		case "checkbox":
			if len(input.SelectedAnswerIDs) == 0 {
				if isQuizSubmission {
					storedAnswers = append(storedAnswers, storedAnswer{
						QuestionID:     question.ID,
						Question:       question.Question,
						TypeAnswer:     question.TypeAnswer,
						Point:          question.Point,
						CorrectAnswers: collectCorrectAnswerLabels(question),
					})
					continue
				}
				return PublicSubmissionResult{}, fmt.Errorf("Semua question checkbox wajib dipilih minimal 1 opsi.")
			}
			selectedOptions, err := findQuestionAnswersByIDs(question.Answers, input.SelectedAnswerIDs)
			if err != nil {
				return PublicSubmissionResult{}, err
			}
			// Benar cuma kalau himpunan yang dipilih PERSIS sama dengan himpunan opsi true —
			// gak ada partial credit, biar konsisten sama model benar/salah biner tipe lain.
			correct := selectedSetMatchesCorrectSet(question.Answers, selectedOptions)
			isCorrect := &correct
			if correct {
				correctAnswers++
				if question.Point != nil {
					score += *question.Point
				}
			}
			labels := make([]string, 0, len(selectedOptions))
			ids := make([]int64, 0, len(selectedOptions))
			for _, option := range selectedOptions {
				labels = append(labels, strings.TrimSpace(stringPtrValue(option.Label)))
				ids = append(ids, option.ID)
			}
			joinedLabel := strings.Join(labels, ", ")
			answeredQuestions++
			storedAnswers = append(storedAnswers, storedAnswer{
				QuestionID:        question.ID,
				Question:          question.Question,
				TypeAnswer:        question.TypeAnswer,
				Point:             question.Point,
				SelectedAnswerIDs: ids,
				AnswerLabel:       &joinedLabel,
				IsCorrect:         isCorrect,
				CorrectAnswers:    collectCorrectAnswerLabels(question),
			})
		case "matrix":
			if len(input.MatrixAnswers) == 0 {
				if isQuizSubmission {
					storedAnswers = append(storedAnswers, storedAnswer{
						QuestionID:     question.ID,
						Question:       question.Question,
						TypeAnswer:     question.TypeAnswer,
						Point:          question.Point,
						CorrectAnswers: collectCorrectAnswerLabels(question),
					})
					continue
				}
				return PublicSubmissionResult{}, fmt.Errorf("Semua baris matrix wajib diisi.")
			}
			// Matrix bersifat survey (ungraded, kayak rating/free_text) — 1 storedAnswer per baris
			// pernyataan yang dijawab, ditandai MatrixRowID biar gak ketuker sama jawaban baris lain.
			rowByID := make(map[int64]QuestionMatrixRow, len(question.MatrixRows))
			for _, row := range question.MatrixRows {
				rowByID[row.ID] = row
			}
			for _, matrixInput := range input.MatrixAnswers {
				row, exists := rowByID[matrixInput.RowID]
				if !exists {
					return PublicSubmissionResult{}, fmt.Errorf("Ada baris matrix yang tidak sesuai dengan question ini.")
				}
				answerID := matrixInput.QuestionAnswerID
				selected, err := findQuestionAnswerByID(question.Answers, &answerID)
				if err != nil {
					return PublicSubmissionResult{}, err
				}
				rowID := row.ID
				rowLabel := fmt.Sprintf("%s — %s", stringPtrValue(question.Question), stringPtrValue(row.RowLabel))
				storedAnswers = append(storedAnswers, storedAnswer{
					QuestionID:  question.ID,
					Question:    &rowLabel,
					TypeAnswer:  question.TypeAnswer,
					Point:       question.Point,
					MatrixRowID: &rowID,
					AnswerLabel: selected.Label,
					AnswerValue: selected.Value,
				})
			}
			// Question matrix cuma dihitung "answered" 1x buat total answered_questions, walau
			// jawabannya tersebar di banyak baris (biar gak lebih besar dari total_questions).
			answeredQuestions++
		default:
			if input.QuestionAnswerID == nil || *input.QuestionAnswerID <= 0 {
				if isQuizSubmission {
					storedAnswers = append(storedAnswers, storedAnswer{
						QuestionID:     question.ID,
						Question:       question.Question,
						TypeAnswer:     question.TypeAnswer,
						Point:          question.Point,
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
			if typeAnswer := stringPtrValue(question.TypeAnswer); typeAnswer == "multiple_choice" || typeAnswer == "dropdown" {
				correct := strings.EqualFold(stringPtrValue(selected.Value), "true")
				isCorrect = &correct
				if correct {
					correctAnswers++
				}
				// Scoring otomatis hanya berlaku untuk pilihan ganda/dropdown; tipe lain tetap tersimpan
				// tapi tidak menambah poin karena tidak ada rule benar/salah yang eksplisit.
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
	// Device fingerprint cuma disimpan buat quiz (dedup anti-cheat) -- survey sengaja dibiarkan NULL
	// biar bisa diisi berkali-kali tanpa identitas (fitur "Isi Ulang" survey yang udah ada).
	fingerprintValue := interface{}(nil)
	if isQuizSubmission && deviceFingerprint != nil && strings.TrimSpace(*deviceFingerprint) != "" {
		fingerprintValue = strings.TrimSpace(*deviceFingerprint)
	}

	res, err := tx.Exec(
		"INSERT INTO quiz_submissions (quiz_id, respondent_email, device_fingerprint, score, violation_count, started_at, submitted_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		quiz.ID, emailValue, fingerprintValue, scoreValue, violationCount, startedAtValue, now,
	)
	if err != nil {
		return PublicSubmissionResult{}, err
	}
	submissionID, err := res.LastInsertId()
	if err != nil {
		return PublicSubmissionResult{}, err
	}

	stmt, err := tx.Prepare(
		"INSERT INTO quiz_submission_answers (submission_id, question_id, question_answer_id, selected_answer_ids, matrix_row_id, answer_label, answer_value, answer_text, is_correct) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
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
			int64SlicePtrValue(answer.SelectedAnswerIDs),
			int64PtrValue(answer.MatrixRowID),
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

func generateAccessCode(db *sql.DB) (string, error) {
	for i := 0; i < 5; i++ {
		buf := make([]byte, 4)
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		value := (int(buf[0]) << 24) | (int(buf[1]) << 16) | (int(buf[2]) << 8) | int(buf[3])
		code := fmt.Sprintf("%06d", value%1000000)
		var exists int
		err := db.QueryRow("SELECT 1 FROM quiz_shares WHERE access_code = ? LIMIT 1", code).Scan(&exists)
		if err == sql.ErrNoRows {
			return code, nil
		}
		if err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("failed to generate unique access code")
}

func getQuizShareByToken(db *sql.DB, token string) (QuizShare, error) {
	var share QuizShare
	var quizID sql.NullInt64
	var storedToken sql.NullString
	var accessCode sql.NullString
	err := db.QueryRow(
		"SELECT id, quiz_id, token, access_code FROM quiz_shares WHERE token = ? LIMIT 1",
		token,
	).Scan(&share.ID, &quizID, &storedToken, &accessCode)
	if err != nil {
		return QuizShare{}, err
	}
	if quizID.Valid {
		value := quizID.Int64
		share.QuizID = &value
	}
	share.Token = nullableString(storedToken)
	share.AccessCode = nullableString(accessCode)
	return share, nil
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
// supaya user/admin bisa melihat kunci jawaban yang benar untuk soal pilihan ganda/dropdown/checkbox.
func collectCorrectAnswerLabels(question Question) []string {
	switch stringPtrValue(question.TypeAnswer) {
	case "multiple_choice", "dropdown", "checkbox":
	default:
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

// findQuestionAnswersByIDs versi jamak dari findQuestionAnswerByID, dipakai buat checkbox yang
// bisa pilih lebih dari 1 opsi sekaligus. Urutan hasil ngikutin urutan answerIDs yang dikirim.
func findQuestionAnswersByIDs(answers []QuestionAnswer, answerIDs []int64) ([]QuestionAnswer, error) {
	byID := make(map[int64]QuestionAnswer, len(answers))
	for _, answer := range answers {
		byID[answer.ID] = answer
	}

	selected := make([]QuestionAnswer, 0, len(answerIDs))
	seen := map[int64]bool{}
	for _, id := range answerIDs {
		if seen[id] {
			continue // abaikan id duplikat, biar gak dihitung dobel di exact-match check
		}
		seen[id] = true
		answer, exists := byID[id]
		if !exists {
			return nil, fmt.Errorf("Ada jawaban yang tidak valid untuk question ini.")
		}
		selected = append(selected, answer)
	}
	return selected, nil
}

// selectedSetMatchesCorrectSet cek apakah himpunan opsi yang dipilih respondent PERSIS sama
// dengan himpunan opsi yang ditandai true oleh admin (checkbox gak ada partial credit).
func selectedSetMatchesCorrectSet(allAnswers []QuestionAnswer, selected []QuestionAnswer) bool {
	correctIDs := map[int64]bool{}
	for _, answer := range allAnswers {
		if strings.EqualFold(stringPtrValue(answer.Value), "true") {
			correctIDs[answer.ID] = true
		}
	}
	if len(selected) != len(correctIDs) {
		return false
	}
	for _, answer := range selected {
		if !correctIDs[answer.ID] {
			return false
		}
	}
	return true
}

// selectRandomQuestionSubset ambil count question secara acak dari pool, deterministic berdasarkan
// attemptSeed (dikirim FE dari localStorage per sesi) supaya subset yang sama dikembalikan lagi
// tiap kali GET/submit dipanggil ulang dalam sesi yang sama (form gak reload subset baru tiap refresh).
// count nil/<=0/>=total pool berarti fitur nonaktif, balikin semua question apa adanya.
func selectRandomQuestionSubset(questions []Question, count *int, attemptSeed string) []Question {
	if count == nil || *count <= 0 || *count >= len(questions) {
		return questions
	}

	// Urutkan dulu berdasarkan ID biar hasil seeded-shuffle deterministic (urutan asli dari DB
	// gak dijamin konsisten antar query).
	pool := make([]Question, len(questions))
	copy(pool, questions)
	sort.Slice(pool, func(i, j int) bool { return pool[i].ID < pool[j].ID })

	var seed int64
	if strings.TrimSpace(attemptSeed) == "" {
		// Gak ada attempt seed dari FE (harusnya gak terjadi di alur normal) -- fallback ke random
		// murni per-request biar tetap ada subset yang masuk akal walau gak stabil antar reload.
		seed = time.Now().UnixNano()
	} else {
		hasher := fnv.New64a()
		hasher.Write([]byte(attemptSeed))
		seed = int64(hasher.Sum64())
	}

	randomizer := mathrand.New(mathrand.NewSource(seed))
	randomizer.Shuffle(len(pool), func(i, j int) {
		pool[i], pool[j] = pool[j], pool[i]
	})
	return pool[:*count]
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

// int64SlicePtrValue encode slice id opsi checkbox jadi JSON array buat kolom TEXT,
// slice kosong/nil jadi SQL NULL (bukan checkbox atau checkbox belum dijawab).
func int64SlicePtrValue(ids []int64) interface{} {
	if len(ids) == 0 {
		return nil
	}
	encoded, err := json.Marshal(ids)
	if err != nil {
		return nil
	}
	return string(encoded)
}

// decodeInt64Slice parse balik JSON array id opsi checkbox dari kolom TEXT. String kosong/invalid
// dianggap gak ada selection (dipakai analytics/summary buat baca selected_answer_ids).
func decodeInt64Slice(raw string) []int64 {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var ids []int64
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil
	}
	return ids
}
