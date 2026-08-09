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
  total_question: number;
  status: string | null;
  state: PublicFormState;
  server_time: string;
  questions: PublicQuestion[];
}

export interface PublicFormSubmitPayload {
  email: string | null;
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
  submitted_at: string;
  message: string;
}

@Injectable({ providedIn: 'root' })
export class PublicFormService {
  private baseUrl = `${environment.apiUrl}/api/public/quizzes`;

  constructor(private http: HttpClient) {}

  // Ambil detail form publik berdasarkan token share.
  getByToken(token: string): Observable<PublicFormDetail> {
    return this.http.get<PublicFormDetail>(`${this.baseUrl}/${token}`);
  }

  // Submit jawaban publik untuk quiz/survey aktif.
  submit(token: string, payload: PublicFormSubmitPayload): Observable<PublicFormSubmitResult> {
    return this.http.post<PublicFormSubmitResult>(`${this.baseUrl}/${token}/submit`, payload);
  }
}
