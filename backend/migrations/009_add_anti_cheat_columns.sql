ALTER TABLE quizzes ADD COLUMN lock_mode TINYINT(1) NOT NULL DEFAULT 0 AFTER random_question_count;

ALTER TABLE quiz_submissions ADD COLUMN device_fingerprint VARCHAR(191) NULL AFTER respondent_email;
ALTER TABLE quiz_submissions ADD COLUMN violation_count INT NOT NULL DEFAULT 0 AFTER score;
-- NULL dianggap distinct sama MySQL, jadi submission survey (device_fingerprint selalu NULL) tetap
-- bisa berkali-kali; dedup device cuma efektif buat quiz yang beneran ngisi device_fingerprint.
ALTER TABLE quiz_submissions ADD UNIQUE KEY uq_quiz_submissions_quiz_fingerprint (quiz_id, device_fingerprint);
