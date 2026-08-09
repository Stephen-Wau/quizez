CREATE TABLE IF NOT EXISTS quiz_shares (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    quiz_id BIGINT NULL,
    token VARCHAR(191) NULL,
    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_quiz_shares_quiz_id (quiz_id),
    UNIQUE KEY uq_quiz_shares_token (token),
    CONSTRAINT fk_quiz_shares_quiz
        FOREIGN KEY (quiz_id) REFERENCES quizzes(id)
        ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS quiz_submissions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    quiz_id BIGINT NULL,
    respondent_email VARCHAR(191) NULL,
    score INT NULL,
    started_at DATETIME NULL,
    submitted_at DATETIME NULL,
    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_quiz_submissions_quiz_id (quiz_id),
    KEY idx_quiz_submissions_email (respondent_email),
    UNIQUE KEY uq_quiz_submissions_quiz_email (quiz_id, respondent_email),
    CONSTRAINT fk_quiz_submissions_quiz
        FOREIGN KEY (quiz_id) REFERENCES quizzes(id)
        ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS quiz_submission_answers (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    submission_id BIGINT NULL,
    question_id BIGINT NULL,
    question_answer_id BIGINT NULL,
    answer_label TEXT NULL,
    answer_value TEXT NULL,
    answer_text TEXT NULL,
    is_correct TINYINT(1) NULL,
    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_quiz_submission_answers_submission_id (submission_id),
    KEY idx_quiz_submission_answers_question_id (question_id),
    CONSTRAINT fk_quiz_submission_answers_submission
        FOREIGN KEY (submission_id) REFERENCES quiz_submissions(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_quiz_submission_answers_question
        FOREIGN KEY (question_id) REFERENCES questions(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_quiz_submission_answers_question_answer
        FOREIGN KEY (question_answer_id) REFERENCES questions_answers(id)
        ON DELETE SET NULL
);
