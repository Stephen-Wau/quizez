ALTER TABLE quizzes
    ADD COLUMN passing_grade INT NULL AFTER max_point;

UPDATE quizzes
SET passing_grade = CEIL(max_point * 0.7)
WHERE type = 'quiz'
  AND max_point IS NOT NULL
  AND passing_grade IS NULL;
