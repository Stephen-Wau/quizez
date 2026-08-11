import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';

import { environment } from '../../../../environments/environment';
import { Quiz } from '../quiz/quiz.service';
import {
  QuestionSummary,
  QuizSummaryStats,
  SubmissionSummary,
  SummaryBucket,
} from '../summary/summary.service';

export interface TrendPoint {
  label: string;
  count: number;
}

export interface QuestionIncorrectRank {
  question_id: number;
  question: string | null;
  total_responses: number;
  incorrect_count: number;
  incorrect_rate: number;
}

export interface KeywordCount {
  keyword: string;
  count: number;
}

export interface QuestionSentimentSummary {
  question_id: number;
  question: string | null;
  positive: number;
  neutral: number;
  negative: number;
  top_keywords: KeywordCount[];
}

export interface QuizAnalyticsResponse {
  quiz: Quiz;
  stats: QuizSummaryStats;
  score_distribution: SummaryBucket[];
  question_summaries: QuestionSummary[];
  submission_summaries: SubmissionSummary[];
  trend: TrendPoint[];
  top_incorrect_questions: QuestionIncorrectRank[];
  sentiment_summaries: QuestionSentimentSummary[];
}

// Filter halaman Analytics & Reporting: semua field opsional, kosong/undefined berarti "semua data".
export interface AnalyticsFilter {
  start_date?: string; // "YYYY-MM-DD"
  end_date?: string; // "YYYY-MM-DD"
  respondent?: string;
  min_score?: number;
  max_score?: number;
  group_by?: 'day' | 'hour';
}

export type AnalyticsExportFormat = 'csv' | 'xlsx' | 'pdf';

@Injectable({ providedIn: 'root' })
export class AnalyticsService {
  private baseUrl = `${environment.apiUrl}/api/quizzes`;

  constructor(private http: HttpClient) {}

  // Ambil data analytics untuk 1 quiz, sudah kena filter yang aktif di halaman.
  getAnalytics(quizId: number, filter: AnalyticsFilter): Observable<QuizAnalyticsResponse> {
    return this.http.get<QuizAnalyticsResponse>(`${this.baseUrl}/${quizId}/analytics`, {
      params: this.buildParams(filter),
    });
  }

  // Unduh hasil analytics (summary + raw submission) dalam format csv/xlsx/pdf sesuai filter aktif.
  // responseType 'blob' karena hasilnya file binary, bukan JSON.
  exportAnalytics(quizId: number, format: AnalyticsExportFormat, filter: AnalyticsFilter): Observable<Blob> {
    return this.http.get(`${this.baseUrl}/${quizId}/analytics/export`, {
      params: this.buildParams(filter).set('format', format),
      responseType: 'blob',
    });
  }

  // Susun HttpParams dari AnalyticsFilter, cuma nempelin field yang emang keisi biar query string bersih.
  private buildParams(filter: AnalyticsFilter): HttpParams {
    let params = new HttpParams();
    if (filter.start_date) params = params.set('start_date', filter.start_date);
    if (filter.end_date) params = params.set('end_date', filter.end_date);
    if (filter.respondent) params = params.set('respondent', filter.respondent);
    if (filter.min_score !== undefined && filter.min_score !== null) params = params.set('min_score', filter.min_score);
    if (filter.max_score !== undefined && filter.max_score !== null) params = params.set('max_score', filter.max_score);
    if (filter.group_by) params = params.set('group_by', filter.group_by);
    return params;
  }
}
