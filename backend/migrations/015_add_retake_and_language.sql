-- Retake policy per quiz: max_attempts NULL/1 = behavior lama (1x doang), angka >1 = boleh diulang
-- sampai segitu kali. retake_score_policy cuma relevan kalau max_attempts > 1 ('best'/'latest').
ALTER TABLE quizzes ADD COLUMN max_attempts INT NULL AFTER lock_mode;
ALTER TABLE quizzes ADD COLUMN retake_score_policy VARCHAR(20) NULL AFTER max_attempts;

-- Bahasa form publik per-quiz ('id'/'en'), NULL dianggap 'id' di level aplikasi.
ALTER TABLE quizzes ADD COLUMN language VARCHAR(10) NULL AFTER retake_score_policy;

-- Retake butuh banyak baris per email/device_fingerprint yang sama untuk 1 quiz, jadi unique
-- constraint lama (cuma izinin 1x) diganti index biasa -- masih perlu buat performa WHERE
-- quiz_id+email / quiz_id+fingerprint, tapi limit jumlah attempt-nya sekarang dicek di level
-- aplikasi (bandingin COUNT(*) vs quiz.max_attempts), bukan lewat DB constraint lagi.
ALTER TABLE quiz_submissions DROP INDEX uq_quiz_submissions_quiz_email;
ALTER TABLE quiz_submissions ADD INDEX idx_quiz_submissions_quiz_email (quiz_id, respondent_email);
ALTER TABLE quiz_submissions DROP INDEX uq_quiz_submissions_quiz_fingerprint;
ALTER TABLE quiz_submissions ADD INDEX idx_quiz_submissions_quiz_fingerprint (quiz_id, device_fingerprint);
