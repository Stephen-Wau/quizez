import { CommonModule } from '@angular/common';
import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';

import { ButtonComponent } from '../../../shared/ui/button/button.component';
import { ToastService } from '../../../shared/ui/toast/toast.service';
import {
  QuestionOptionSummary,
  QuestionSummary,
  QuizSummaryResponse,
  SubmissionAnswerSummary,
  SubmissionSummary,
  SummaryBucket,
  SummaryService,
} from '../summary/summary.service';

@Component({
  selector: 'app-summary-detail',
  standalone: true,
  imports: [CommonModule, ButtonComponent],
  templateUrl: './summary-detail.component.html',
  styleUrl: './summary-detail.component.scss',
})
export class SummaryDetailComponent implements OnInit {
  loading = true;
  loadError = '';
  summary: QuizSummaryResponse | null = null;
  readonly chartColors = ['#4f46e5', '#7c3aed', '#0ea5e9', '#14b8a6', '#f59e0b', '#ef4444'];
  collapsedSections = {
    overview: false,
    respondent: false,
    analytics: false,
    submissions: false,
  };

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private summaryService: SummaryService,
    private toast: ToastService,
  ) {}

  // Ambil analytics berdasarkan id quiz dari URL detail summary.
  ngOnInit(): void {
    const id = Number(this.route.snapshot.paramMap.get('id') ?? 0);
    if (!Number.isInteger(id) || id <= 0) {
      this.loading = false;
      this.loadError = 'Quiz tidak valid.';
      return;
    }
    this.loadSummary(id);
  }

  get isQuiz(): boolean {
    return this.summary?.quiz.type === 'quiz';
  }

  get hasSubmissions(): boolean {
    return (this.summary?.stats.total_submissions ?? 0) > 0;
  }

  // Ringkasan type question dipakai untuk donut chart komposisi question.
  get questionTypeBreakdown(): Array<{ label: string; count: number; color: string }> {
    if (!this.summary) return [];

    const grouped = new Map<string, number>();
    for (const question of this.summary.question_summaries) {
      const key = question.type_answer || 'unknown';
      grouped.set(key, (grouped.get(key) ?? 0) + 1);
    }

    return Array.from(grouped.entries()).map(([label, count], index) => ({
      label: this.humanizeType(label),
      count,
      color: this.chartColors[index % this.chartColors.length],
    }));
  }

  // Submission performance chart dipakai untuk membandingkan skor/completion antar user.
  get submissionPerformance(): Array<{
    id: number;
    respondent: string;
    score: number;
    completion: number;
    submittedAt: string | null;
  }> {
    if (!this.summary) return [];

    return this.summary.submission_summaries.map((submission) => ({
      id: submission.id,
      respondent: submission.respondent_email || 'Anonymous respondent',
      score: submission.score ?? 0,
      completion: submission.completion_percentage,
      submittedAt: submission.submitted_at,
    }));
  }

  // cards untuk header dashboard: disusun sebagai getter supaya template tetap ringkas.
  get summaryCards(): Array<{ label: string; value: string; accent?: 'indigo' | 'emerald' | 'amber' | 'rose' }> {
    if (!this.summary) return [];
    const stats = this.summary.stats;

    return [
      { label: 'Total Submission', value: String(stats.total_submissions), accent: 'indigo' },
      { label: 'Total Respondent', value: String(stats.unique_respondents), accent: 'emerald' },
      { label: 'Avg Completion', value: `${this.formatPercent(stats.average_completion)}%`, accent: 'amber' },
      {
        label: this.isQuiz ? 'Average Score' : 'Latest Response',
        value: this.isQuiz ? this.formatScore(stats.average_score) : this.formatDateTime(stats.latest_submission_at),
        accent: 'rose',
      },
      {
        label: this.isQuiz ? 'Passing Rate' : 'Total Question',
        value: this.isQuiz ? `${this.formatPercent(stats.passing_rate)}%` : String(this.summary.quiz.total_question),
      },
      {
        label: this.isQuiz ? 'Highest Score' : 'Current Status',
        value: this.isQuiz ? this.formatScore(stats.highest_score) : (this.summary.quiz.status ?? '-'),
      },
    ];
  }

  // Buka kembali halaman daftar summary.
  backToList(): void {
    this.router.navigate(['/admin-cms/summary']);
  }

  // Toggle dipisah per section biar nanti mudah ditambah collapse panel baru tanpa ubah banyak template.
  toggleSection(section: keyof typeof this.collapsedSections): void {
    this.collapsedSections[section] = !this.collapsedSections[section];
  }

  // Format period detail: quiz jam saja, survey tanggal+jam.
  formatPeriod(): string {
    if (!this.summary?.quiz.start_time && !this.summary?.quiz.end_time) return '-';
    if (this.summary?.quiz.type === 'quiz') {
      const start = this.summary.quiz.start_time ? this.summary.quiz.start_time.slice(11, 16) : '?';
      const end = this.summary.quiz.end_time ? this.summary.quiz.end_time.slice(11, 16) : '?';
      return `${start} - ${end}`;
    }
    const fmt = (value: string | null) => (value ? this.formatDateTime(value) : '?');
    return `${fmt(this.summary?.quiz.start_time ?? null)} - ${fmt(this.summary?.quiz.end_time ?? null)}`;
  }

  // Lebar batang chart diskalakan terhadap bucket paling besar supaya visual tetap proporsional.
  bucketWidth(bucket: SummaryBucket): number {
    const max = Math.max(...(this.summary?.score_distribution.map((item) => item.count) ?? [0]), 1);
    return (bucket.count / max) * 100;
  }

  // Donut score distribution dibangun dari bucket sebaran skor supaya lebih visual dari progress bar biasa.
  scoreDistributionChartStyle(): string {
    const buckets = this.summary?.score_distribution ?? [];
    const total = buckets.reduce((sum, bucket) => sum + bucket.count, 0);
    if (total <= 0) {
      return 'conic-gradient(#e2e8f0 0deg 360deg)';
    }

    let start = 0;
    const parts = buckets.map((bucket, index) => {
      const degrees = (bucket.count / total) * 360;
      const end = start + degrees;
      const color = this.chartColors[index % this.chartColors.length];
      const segment = `${color} ${start.toFixed(2)}deg ${end.toFixed(2)}deg`;
      start = end;
      return segment;
    });

    return `conic-gradient(${parts.join(', ')})`;
  }

  questionTypeChartStyle(): string {
    const types = this.questionTypeBreakdown;
    const total = types.reduce((sum, type) => sum + type.count, 0);
    if (total <= 0) {
      return 'conic-gradient(#e2e8f0 0deg 360deg)';
    }

    let start = 0;
    const parts = types.map((type) => {
      const degrees = (type.count / total) * 360;
      const end = start + degrees;
      const segment = `${type.color} ${start.toFixed(2)}deg ${end.toFixed(2)}deg`;
      start = end;
      return segment;
    });

    return `conic-gradient(${parts.join(', ')})`;
  }

  optionWidth(option: QuestionOptionSummary, question: QuestionSummary): number {
    if (question.total_responses <= 0) return 0;
    return option.percentage;
  }

  correctRate(question: QuestionSummary): number {
    if (question.total_responses <= 0) return 0;
    return (question.correct_count / question.total_responses) * 100;
  }

  answerPreview(answer: SubmissionAnswerSummary): string {
    if (answer.answer_text) return answer.answer_text;
    if (answer.answer_label) return answer.answer_label;
    return '-';
  }

  trackByQuestionId(_: number, item: QuestionSummary): number {
    return item.question_id;
  }

  trackBySubmissionId(_: number, item: SubmissionSummary): number {
    return item.id;
  }

  // Drill-down ke halaman submission detail per respondent untuk membaca jawaban lengkap.
  openSubmissionDetail(submission: SubmissionSummary): void {
    if (!this.summary) return;
    this.router.navigate(['/admin-cms/summary', this.summary.quiz.id, 'submission', submission.id]);
  }

  trackByOptionLabel(_: number, item: QuestionOptionSummary): string {
    return item.label;
  }

  private loadSummary(quizId: number): void {
    this.loading = true;
    this.loadError = '';
    this.summaryService.getSummary(quizId).subscribe({
      next: (summary) => {
        this.loading = false;
        this.summary = summary;
      },
      error: (err) => {
        this.loading = false;
        this.summary = null;
        this.loadError =
          typeof err?.error === 'string' && err.error ? err.error : 'Gagal memuat dashboard summary.';
        this.toast.error(this.loadError);
      },
    });
  }

  formatDateTime(value: string | null): string {
    if (!value) return '-';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return `${value.slice(0, 10)} ${value.slice(11, 16)}`;
    }

    const datePart = new Intl.DateTimeFormat('en-GB', {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
    }).format(date);
    const timePart = new Intl.DateTimeFormat('en-GB', {
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    }).format(date);
    return `${datePart} ${timePart}`;
  }

  formatScore(value: number | null): string {
    if (value === null || value === undefined) return '-';
    return `${value}`;
  }

  formatPercent(value: number | null): string {
    if (value === null || value === undefined) return '0';
    return value.toFixed(2).replace(/\.00$/, '');
  }

  private humanizeType(value: string): string {
    return value
      .split('_')
      .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
      .join(' ');
  }
}
