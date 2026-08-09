import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

import { environment } from '../../../../environments/environment';
import { Quiz } from '../quiz/quiz.service';

export interface QuizSummaryStats {
  total_submissions: number;
  unique_respondents: number;
  average_score: number | null;
  highest_score: number | null;
  lowest_score: number | null;
  average_completion: number;
  latest_submission_at: string | null;
}

export interface SummaryBucket {
  label: string;
  count: number;
}

export interface QuestionOptionSummary {
  label: string;
  count: number;
  percentage: number;
}

export interface QuestionTextResponse {
  respondent_email: string | null;
  answer_text: string;
  submitted_at: string | null;
}

export interface QuestionSummary {
  question_id: number;
  question: string | null;
  type_answer: string | null;
  point: number | null;
  total_responses: number;
  correct_count: number;
  incorrect_count: number;
  average_rating: number | null;
  option_summaries: QuestionOptionSummary[];
  text_responses: QuestionTextResponse[];
}

export interface SubmissionAnswerSummary {
  question_id: number;
  question: string | null;
  type_answer: string | null;
  answer_label: string | null;
  answer_text: string | null;
  is_correct: boolean | null;
}

export interface SubmissionSummary {
  id: number;
  respondent_email: string | null;
  score: number | null;
  submitted_at: string | null;
  completion_percentage: number;
  answers: SubmissionAnswerSummary[];
}

export interface QuizSummaryResponse {
  quiz: Quiz;
  stats: QuizSummaryStats;
  score_distribution: SummaryBucket[];
  question_summaries: QuestionSummary[];
  submission_summaries: SubmissionSummary[];
}

@Injectable({ providedIn: 'root' })
export class SummaryService {
  private baseUrl = `${environment.apiUrl}/api/quizzes`;

  constructor(private http: HttpClient) {}

  // Ambil analytics summary lengkap untuk 1 quiz/survey.
  getSummary(quizId: number): Observable<QuizSummaryResponse> {
    return this.http.get<QuizSummaryResponse>(`${this.baseUrl}/${quizId}/summary`);
  }
}
