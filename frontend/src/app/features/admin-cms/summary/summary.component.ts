import { CommonModule } from '@angular/common';
import { Component, OnInit, TemplateRef, ViewChild } from '@angular/core';
import { Router } from '@angular/router';

import { Quiz, QuizService } from '../quiz/quiz.service';
import {
  DataTableColumn,
  DataTableComponent,
  DataTableQuery,
} from '../../../shared/ui/data-table/data-table.component';
import { ButtonComponent } from '../../../shared/ui/button/button.component';
import { BadgeComponent } from '../../../shared/ui/badge/badge.component';
import { ToastService } from '../../../shared/ui/toast/toast.service';
import { loadPagedList } from '../../../shared/utils/load-paged-list.util';

@Component({
  selector: 'app-summary',
  standalone: true,
  imports: [CommonModule, DataTableComponent, ButtonComponent, BadgeComponent],
  templateUrl: './summary.component.html',
  styleUrl: './summary.component.scss',
})
export class SummaryComponent implements OnInit {
  @ViewChild('typeTpl', { static: true }) typeTpl!: TemplateRef<unknown>;
  @ViewChild('periodTpl', { static: true }) periodTpl!: TemplateRef<unknown>;
  @ViewChild('actionTpl', { static: true }) actionTpl!: TemplateRef<unknown>;

  quizzes: Quiz[] = [];
  columns: DataTableColumn[] = [];
  totalCount = 0;
  pageSize = 10;
  private currentQuery: DataTableQuery = {};

  constructor(
    private quizService: QuizService,
    private toast: ToastService,
    private router: Router,
  ) {}

  // Setup kolom tabel dan langsung load daftar quiz/survey yang bisa dibuka summary-nya.
  ngOnInit(): void {
    this.columns = [
      { name: 'Title', prop: 'title' },
      { name: 'Type', prop: 'type', cellTemplate: this.typeTpl },
      { name: 'Period', prop: 'start_time', cellTemplate: this.periodTpl },
      { name: 'Total Question', prop: 'total_question' },
      { name: 'Status', prop: 'status' },
      { name: 'Action', sortable: false, cellTemplate: this.actionTpl },
    ];
    this.loadQuizzes();
  }

  // Ambil daftar quiz dari API list existing biar menu summary tetap sinkron dengan data master.
  loadQuizzes(): void {
    loadPagedList(
      this.quizService.list(this.currentQuery),
      this.toast,
      'Gagal memuat data quiz.',
      (data, totalCount, pageSize) => {
        this.quizzes = data;
        this.totalCount = totalCount;
        this.pageSize = pageSize;
      },
    );
  }

  onTableQueryChange(query: DataTableQuery): void {
    this.currentQuery = query;
    this.loadQuizzes();
  }

  // Format period disamakan dengan menu question-answer: quiz tampil jam, survey tampil tanggal ramah baca.
  formatPeriod(row: Quiz): string {
    if (!row.start_time && !row.end_time) return '-';
    if (row.type === 'quiz') {
      const start = row.start_time ? row.start_time.slice(11, 16) : '?';
      const end = row.end_time ? row.end_time.slice(11, 16) : '?';
      return `${start} - ${end}`;
    }
    const fmt = (value: string | null) => (value ? this.formatSurveyDateTime(value) : '?');
    return `${fmt(row.start_time)} - ${fmt(row.end_time)}`;
  }

  // Buka halaman detail summary untuk quiz yang dipilih.
  openSummary(quiz: Quiz): void {
    this.router.navigate(['/admin-cms/summary', quiz.id]);
  }

  private formatSurveyDateTime(value: string): string {
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
}
