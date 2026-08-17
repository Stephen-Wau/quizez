import { CommonModule } from '@angular/common';
import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';

import { BadgeComponent } from '../../../shared/ui/badge/badge.component';
import { ButtonComponent } from '../../../shared/ui/button/button.component';
import { ToastService } from '../../../shared/ui/toast/toast.service';
import {
  QuizSubmissionDetailResponse,
  SubmissionAnswerSummary,
  SummaryService,
} from '../summary/summary.service';

@Component({
  selector: 'app-summary-submission-detail',
  standalone: true,
  imports: [CommonModule, ButtonComponent, BadgeComponent],
  templateUrl: './summary-submission-detail.component.html',
  styleUrl: './summary-submission-detail.component.scss',
})
export class SummarySubmissionDetailComponent implements OnInit {
  loading = true;
  loadError = '';
  detail: QuizSubmissionDetailResponse | null = null;

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private summaryService: SummaryService,
    private toast: ToastService,
  ) {}

  ngOnInit(): void {
    const quizId = Number(this.route.snapshot.paramMap.get('quizId') ?? 0);
    const submissionId = Number(this.route.snapshot.paramMap.get('submissionId') ?? 0);
    if (!Number.isInteger(quizId) || quizId <= 0 || !Number.isInteger(submissionId) || submissionId <= 0) {
      this.loading = false;
      this.loadError = 'Submission detail tidak valid.';
      return;
    }
    this.loadDetail(quizId, submissionId);
  }

  get isQuiz(): boolean {
    return this.detail?.quiz.type === 'quiz';
  }

  respondentName(): string {
    return this.detail?.submission.respondent_name || this.detail?.submission.respondent_email || 'Anonymous respondent';
  }

  // Kembali ke dashboard summary quiz yang sama agar admin tidak kehilangan konteks analytic.
  backToSummary(): void {
    const quizId = Number(this.route.snapshot.paramMap.get('quizId') ?? 0);
    this.router.navigate(['/admin-cms/summary', quizId]);
  }

  answerPreview(answer: SubmissionAnswerSummary): string {
    if (answer.answer_text) return answer.answer_text;
    if (answer.answer_label) return answer.answer_label;
    return 'Tidak dijawab';
  }

  correctAnswerPreview(answer: SubmissionAnswerSummary): string {
    return answer.correct_answers.length > 0 ? answer.correct_answers.join(', ') : '-';
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

  formatPercent(value: number | null): string {
    if (value === null || value === undefined) return '0';
    return value.toFixed(2).replace(/\.00$/, '');
  }

  durationLabel(): string {
    const startedAt = this.detail?.submission.started_at;
    const submittedAt = this.detail?.submission.submitted_at;
    if (!startedAt || !submittedAt) return '-';

    const start = new Date(startedAt);
    const end = new Date(submittedAt);
    if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime()) || end < start) return '-';

    const totalSeconds = Math.floor((end.getTime() - start.getTime()) / 1000);
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;
    return `${minutes}m ${String(seconds).padStart(2, '0')}s`;
  }

  private loadDetail(quizId: number, submissionId: number): void {
    this.loading = true;
    this.loadError = '';
    this.summaryService.getSubmissionDetail(quizId, submissionId).subscribe({
      next: (detail) => {
        this.loading = false;
        this.detail = detail;
      },
      error: (err) => {
        this.loading = false;
        this.detail = null;
        this.loadError =
          typeof err?.error === 'string' && err.error ? err.error : 'Gagal memuat detail submission.';
        this.toast.error(this.loadError);
      },
    });
  }
}
