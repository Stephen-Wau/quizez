import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

import { environment } from '../../../environments/environment';

export type PublicFormState = 'active' | 'upcoming' | 'expired' | 'inactive';
export type PublicQuestionType = 'multiple_choice' | 'rating' | 'free_text';
export type PublicFormType = 'quiz' | 'survey';

export interface PublicQuestionOption {
  id: number;
  label: string | null;
}

export interface PublicQuestion {
  id: number;
  question: string | null;
  type_answer: PublicQuestionType | null;
  answers: PublicQuestionOption[];
}

export interface PublicFormDetail {
  id: number;
  token: string | null;
  title: string | null;
  type: PublicFormType | null;
  description: string | null;
  start_time: string | null;
  end_time: string | null;
  max_point: number | null;
  passing_grade: number | null;
  total_question: number;
  status: string | null;
  state: PublicFormState;
  server_time: string;
  access_code_required: boolean;
  access_granted: boolean;
  access_message: string | null;
  questions: PublicQuestion[];
}

export interface PublicFormSubmitPayload {
  email: string | null;
  started_at: string | null;
  access_code: string | null;
  answers: Array<{
    question_id: number;
    question_answer_id: number | null;
    answer_text: string | null;
  }>;
}

export interface PublicFormSubmitResult {
  submission_id: number;
  title: string | null;
  type: PublicFormType | null;
  score: number | null;
  max_point: number | null;
  passing_grade: number | null;
  score_percentage: number | null;
  passed: boolean | null;
  correct_answers: number;
  answered_questions: number;
  total_questions: number;
  submitted_at: string;
  message: string;
  answer_details: PublicFormSubmitAnswerDetail[];
}

export interface PublicFormSubmitAnswerDetail {
  question_id: number;
  question: string | null;
  type_answer: PublicQuestionType | null;
  point: number | null;
  selected_answer_label: string | null;
  selected_answer_text: string | null;
  is_correct: boolean | null;
  correct_answers: string[];
}

@Injectable({ providedIn: 'root' })
export class PublicFormService {
  private baseUrl = `${environment.apiUrl}/api/public/quizzes`;

  constructor(private http: HttpClient) {}

  // Ambil detail form publik berdasarkan token share.
  getByToken(token: string, accessCode: string | null = null): Observable<PublicFormDetail> {
    const query = accessCode ? `?code=${encodeURIComponent(accessCode)}` : '';
    return this.http.get<PublicFormDetail>(`${this.baseUrl}/${token}${query}`);
  }

  // Submit jawaban publik untuk quiz/survey aktif.
  submit(token: string, payload: PublicFormSubmitPayload): Observable<PublicFormSubmitResult> {
    return this.http.post<PublicFormSubmitResult>(`${this.baseUrl}/${token}/submit`, payload);
  }
}
