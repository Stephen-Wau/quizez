CREATE TABLE IF NOT EXISTS question_bank (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    question TEXT NULL,
    type_answer VARCHAR(30) NULL,
    point INT NULL,
    -- tags disimpan gabungan dipisah ";" (freeform, bukan master data terpisah).
    tags VARCHAR(255) NULL,
    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS question_bank_answers (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    question_bank_id BIGINT NULL,
    label VARCHAR(191) NULL,
    value VARCHAR(191) NULL,
    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_question_bank_answers_bank
        FOREIGN KEY (question_bank_id) REFERENCES question_bank(id)
        ON DELETE CASCADE
);

-- question_bank_id ditandain di question hasil "reuse" dari bank (copy, bukan live-link) supaya
-- masih bisa dilacak asalnya, tapi edit di quiz atau di bank gak saling mempengaruhi.
ALTER TABLE questions ADD COLUMN question_bank_id BIGINT NULL AFTER quiz_id;
ALTER TABLE questions ADD CONSTRAINT fk_questions_question_bank
    FOREIGN KEY (question_bank_id) REFERENCES question_bank(id)
    ON DELETE SET NULL;
