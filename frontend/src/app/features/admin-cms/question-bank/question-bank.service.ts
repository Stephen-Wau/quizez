import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../../environments/environment';
import { DataTableQuery, PagedResult } from '../../../shared/ui/data-table/data-table.component';

// Tipe soal yang didukung bank soal & import CSV/XLSX (matrix sengaja gak didukung, formatnya
// gak cocok buat flat file dan baris pernyataannya variatif per soal).
export type QuestionBankType = 'multiple_choice' | 'dropdown' | 'checkbox' | 'rating' | 'free_text';

export interface QuestionBankAnswer {
  id: number;
  label: string | null;
  value: string | null;
}

export interface QuestionBankItem {
  id: number;
  question: string | null;
  type_answer: QuestionBankType | null;
  point: number | null;
  tags: string[];
  answers: QuestionBankAnswer[];
}

export interface QuestionBankPayload {
  question: string | null;
  type_answer: QuestionBankType | null;
  point: number | null;
  tags: string[];
  answers: Array<{ label: string | null; value: string | null }>;
}

export interface QuestionBankImportRowError {
  row?: number;
  message: string;
}

export interface QuestionBankImportResult {
  created: number;
  errors: QuestionBankImportRowError[];
}

@Injectable({ providedIn: 'root' })
export class QuestionBankService {
  private baseUrl = `${environment.apiUrl}/api/question-bank`;

  constructor(private http: HttpClient) {}

  // Ambil daftar soal bank (paginated), opsional filter tag di luar search box DataTable.
  list(query: DataTableQuery = {}, tag: string | null = null): Observable<PagedResult<QuestionBankItem>> {
    let params = new HttpParams();
    if (query.searchword) params = params.set('searchword', query.searchword);
    if (query.sort_by) params = params.set('sort_by', query.sort_by);
    if (query.sort_dir) params = params.set('sort_dir', query.sort_dir);
    if (query.page) params = params.set('page', query.page);
    if (query.per_page) params = params.set('per_page', query.per_page);
    if (tag) params = params.set('tag', tag);

    return this.http.get<PagedResult<QuestionBankItem>>(this.baseUrl, { params });
  }

  // Ambil semua tag unik yang pernah dipakai, buat dropdown filter.
  listTags(): Observable<string[]> {
    return this.http.get<string[]>(`${this.baseUrl}/tags`);
  }

  create(payload: QuestionBankPayload): Observable<QuestionBankItem> {
    return this.http.post<QuestionBankItem>(this.baseUrl, payload);
  }

  update(id: number, payload: QuestionBankPayload): Observable<QuestionBankItem> {
    return this.http.put<QuestionBankItem>(`${this.baseUrl}/${id}`, payload);
  }

  delete(id: number): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${id}`);
  }

  // Import bulk dari file CSV/XLSX. fileData base64 data URI dari FileReader (app-files-upload).
  import(fileName: string, fileData: string): Observable<QuestionBankImportResult> {
    return this.http.post<QuestionBankImportResult>(`${this.baseUrl}/import`, {
      file_name: fileName,
      file_data: fileData,
    });
  }

  // Download contoh file import (CSV/XLSX) buat panduan format ke admin.
  downloadTemplate(format: 'csv' | 'xlsx'): Observable<Blob> {
    return this.http.get(`${this.baseUrl}/import-template`, {
      params: { format },
      responseType: 'blob',
    });
  }
}
