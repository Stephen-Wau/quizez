ALTER TABLE quiz_shares
ADD COLUMN access_code VARCHAR(32) NULL AFTER token;

UPDATE quiz_shares
SET access_code = LPAD(CAST(id AS CHAR), 6, '0')
WHERE access_code IS NULL OR access_code = '';

ALTER TABLE quiz_shares
ADD UNIQUE KEY uq_quiz_shares_access_code (access_code);
