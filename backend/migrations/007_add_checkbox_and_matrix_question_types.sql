-- Kolom baru buat nyimpen jawaban checkbox (multi-select): array JSON id opsi yang dipilih
-- respondent. question_answer_id/answer_label/answer_value tetap dipakai apa adanya untuk
-- tipe single-select (multiple_choice/dropdown/rating).
ALTER TABLE quiz_submission_answers
    ADD COLUMN selected_answer_ids TEXT NULL AFTER question_answer_id;

-- Baris pernyataan untuk question tipe matrix (ex: "Kecepatan", "Kualitas", "Harga").
-- Kolom/skala penilaiannya reuse questions_answers (sama seperti opsi multiple_choice/rating).
CREATE TABLE IF NOT EXISTS question_matrix_rows (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    question_id BIGINT NULL,
    row_label VARCHAR(191) NULL,
    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_question_matrix_rows_question
        FOREIGN KEY (question_id) REFERENCES questions(id)
        ON DELETE CASCADE
);

-- Tiap baris matrix butuh 1 jawaban submission sendiri (1 question bisa punya banyak baris),
-- jadi quiz_submission_answers butuh penanda baris mana yang sedang dijawab.
ALTER TABLE quiz_submission_answers
    ADD COLUMN matrix_row_id BIGINT NULL AFTER selected_answer_ids;

ALTER TABLE quiz_submission_answers
    ADD CONSTRAINT fk_quiz_submission_answers_matrix_row
        FOREIGN KEY (matrix_row_id) REFERENCES question_matrix_rows(id)
        ON DELETE CASCADE;
