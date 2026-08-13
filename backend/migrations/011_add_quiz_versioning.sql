-- duplicated_from_id nandain quiz ini hasil "Duplicate jadi versi baru" dari quiz mana (self-reference),
-- dipakai buat lifecycle: quiz yang udah ada submission dikunci editnya, admin duplicate ke versi baru.
ALTER TABLE quizzes ADD COLUMN duplicated_from_id BIGINT NULL AFTER status;
ALTER TABLE quizzes ADD CONSTRAINT fk_quizzes_duplicated_from
    FOREIGN KEY (duplicated_from_id) REFERENCES quizzes(id)
    ON DELETE SET NULL;
