import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../../environments/environment';
import { DataTableQuery, PagedResult } from '../../../shared/ui/data-table/data-table.component';

export type QuizType = 'quiz' | 'survey';

export interface Quiz {
  id: number;
  title: string | null;
  type: QuizType | null;
  start_time: string | null; // "YYYY-MM-DDTHH:mm:ss"
  end_time: string | null;
  description: string | null;
  max_point: number | null;
  passing_grade: number | null;
  total_question: number;
  status: string | null;
}

export interface QuizShareResponse {
  quiz_id: number;
  token: string | null;
}

export type QuizPayload = Omit<Quiz, 'id' | 'total_question'>;

@Injectable({ providedIn: 'root' })
export class QuizService {
  private baseUrl = `${environment.apiUrl}/api/quizzes`;

  constructor(private http: HttpClient) {}

  // Ambil daftar quiz (paginated), cuma nempelin query param yang emang keisi biar url bersih.
  list(query: DataTableQuery = {}): Observable<PagedResult<Quiz>> {
    let params = new HttpParams();
    if (query.searchword) params = params.set('searchword', query.searchword);
    if (query.sort_by) params = params.set('sort_by', query.sort_by);
    if (query.sort_dir) params = params.set('sort_dir', query.sort_dir);
    if (query.page) params = params.set('page', query.page);
    if (query.per_page) params = params.set('per_page', query.per_page);

    return this.http.get<PagedResult<Quiz>>(this.baseUrl, { params });
  }

  // Bikin quiz baru.
  create(payload: QuizPayload): Observable<Quiz> {
    return this.http.post<Quiz>(this.baseUrl, payload);
  }

  // Update quiz yang udah ada berdasarkan id.
  update(id: number, payload: QuizPayload): Observable<Quiz> {
    return this.http.put<Quiz>(`${this.baseUrl}/${id}`, payload);
  }

  // Hapus quiz berdasarkan id.
  delete(id: number): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${id}`);
  }

  // Generate atau ambil token share publik milik quiz tertentu.
  shareLink(id: number): Observable<QuizShareResponse> {
    return this.http.post<QuizShareResponse>(`${this.baseUrl}/${id}/share-link`, {});
  }
}
