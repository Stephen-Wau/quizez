-- Tambah nama respondent (dipakai buat cetak sertifikat & tampilan leaderboard) ke submission quiz.
ALTER TABLE quiz_submissions
    ADD COLUMN respondent_name VARCHAR(191) NULL AFTER respondent_email;
