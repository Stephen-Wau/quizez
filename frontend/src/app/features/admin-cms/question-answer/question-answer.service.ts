import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../../environments/environment';

export type QuestionType = 'multiple_choice' | 'dropdown' | 'checkbox' | 'matrix' | 'rating' | 'free_text';

export interface QuestionAnswer {
  id: number;
  question_id: number | null;
  label: string | null;
  value: string | null;
}

export interface QuestionMatrixRow {
  id: number;
  question_id: number | null;
  row_label: string | null;
}

export interface Question {
  id: number;
  quiz_id: number | null;
  question: string | null;
  type_answer: QuestionType | null;
  point: number | null;
  answers: QuestionAnswer[];
  // matrix_rows cuma keisi buat type_answer="matrix".
  matrix_rows: QuestionMatrixRow[];
}

export interface QuestionPayload {
  quiz_id: number | null;
  question: string | null;
  type_answer: QuestionType | null;
  point: number | null;
  answers: Array<{ label: string | null; value: string | null }>;
  matrix_rows: Array<{ row_label: string | null }>;
}

@Injectable({ providedIn: 'root' })
export class QuestionAnswerService {
  private baseUrl = `${environment.apiUrl}/api/questions`;

  constructor(private http: HttpClient) {}

  // Ambil semua question milik 1 quiz. Untuk menu ini belum dipaging karena item ditampilkan
  // di dalam modal per-quiz, jadi cukup load semua sekaligus.
  listByQuiz(quizId: number): Observable<Question[]> {
    return this.http.get<Question[]>(this.baseUrl, {
      params: { quiz_id: quizId },
    });
  }

  // Simpan question baru beserta daftar answer-nya.
  create(payload: QuestionPayload): Observable<Question> {
    return this.http.post<Question>(this.baseUrl, payload);
  }

  // Update 1 question existing berdasarkan id.
  update(id: number, payload: QuestionPayload): Observable<Question> {
    return this.http.put<Question>(`${this.baseUrl}/${id}`, payload);
  }

  // Hapus 1 question; answer ikut terhapus via cascade di backend.
  delete(id: number): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${id}`);
  }
}
