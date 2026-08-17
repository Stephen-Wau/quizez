import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

import { environment } from '../../../environments/environment';

export type PublicFormState = 'active' | 'upcoming' | 'expired' | 'inactive';
export type PublicQuestionType = 'multiple_choice' | 'dropdown' | 'checkbox' | 'matrix' | 'rating' | 'free_text';
export type PublicFormType = 'quiz' | 'survey';

export interface PublicQuestionOption {
  id: number;
  label: string | null;
}

export interface PublicMatrixRow {
  id: number;
  row_label: string | null;
}

export interface PublicQuestion {
  id: number;
  question: string | null;
  type_answer: PublicQuestionType | null;
  answers: PublicQuestionOption[];
  // matrix_rows cuma keisi buat type_answer="matrix"; answers berperan sebagai kolom skala.
  matrix_rows: PublicMatrixRow[];
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
  // lock_mode: anti-cheat (khusus quiz) -- wajib fullscreen, keluar tab/fullscreen dihitung pelanggaran.
  lock_mode: boolean;
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
  // name nama respondent, wajib untuk quiz (dipakai cetak sertifikat & leaderboard admin).
  name: string | null;
  started_at: string | null;
  access_code: string | null;
  // attempt_seed dipakai backend untuk recompute subset random_question_count yang sama persis
  // dengan yang ditampilkan pas GET, biar scoring/validasi konsisten sama soal yang benar dilihat responden.
  attempt_seed: string | null;
  // device_fingerprint & violation_count dipakai anti-cheat (lock_mode): dedup device per quiz
  // dan jumlah pelanggaran tab-switch/keluar-fullscreen yang direkam selama sesi.
  device_fingerprint: string | null;
  violation_count: number;
  answers: Array<{
    question_id: number;
    question_answer_id: number | null;
    answer_text: string | null;
    // selected_answer_ids dipakai buat type_answer="checkbox".
    selected_answer_ids?: number[];
    // matrix_answers dipakai buat type_answer="matrix", 1 entri per baris pernyataan.
    matrix_answers?: Array<{ row_id: number; question_answer_id: number }>;
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
  // badge_tier tier gamifikasi (gold/silver/bronze) dari score_percentage, null kalau quiz gak punya scoring.
  badge_tier: 'gold' | 'silver' | 'bronze' | null;
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
  correct_answers: string[] | null;
}

@Injectable({ providedIn: 'root' })
export class PublicFormService {
  private baseUrl = `${environment.apiUrl}/api/public/quizzes`;

  constructor(private http: HttpClient) {}

  // Ambil detail form publik berdasarkan token share. attemptSeed dikirim biar backend bisa
  // pilih subset random_question_count yang stabil (sama) tiap kali sesi ini reload/refresh.
  getByToken(token: string, accessCode: string | null = null, attemptSeed: string | null = null): Observable<PublicFormDetail> {
    const params: string[] = [];
    if (accessCode) params.push(`code=${encodeURIComponent(accessCode)}`);
    if (attemptSeed) params.push(`attempt=${encodeURIComponent(attemptSeed)}`);
    const query = params.length ? `?${params.join('&')}` : '';
    return this.http.get<PublicFormDetail>(`${this.baseUrl}/${token}${query}`);
  }

  // Submit jawaban publik untuk quiz/survey aktif.
  submit(token: string, payload: PublicFormSubmitPayload): Observable<PublicFormSubmitResult> {
    return this.http.post<PublicFormSubmitResult>(`${this.baseUrl}/${token}/submit`, payload);
  }

  // URL download sertifikat PDF -- dibuka langsung lewat <a href> (bukan HttpClient) karena
  // response-nya file binary, bukan JSON.
  certificateUrl(token: string, submissionId: number): string {
    return `${this.baseUrl}/${token}/submissions/${submissionId}/certificate`;
  }
}
