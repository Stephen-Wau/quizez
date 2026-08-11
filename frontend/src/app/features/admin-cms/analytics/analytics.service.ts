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

export interface QuizAnalyticsResponse {
  quiz: Quiz;
  stats: QuizSummaryStats;
  score_distribution: SummaryBucket[];
  question_summaries: QuestionSummary[];
  submission_summaries: SubmissionSummary[];
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
