import { CommonModule } from '@angular/common';
import { Component, OnInit } from '@angular/core';
import { FormBuilder, FormsModule, ReactiveFormsModule } from '@angular/forms';
import { LucideAngularModule } from 'lucide-angular';

import { Quiz, QuizService } from '../quiz/quiz.service';
import {
  AnalyticsExportFormat,
  AnalyticsFilter,
  AnalyticsService,
  QuizAnalyticsResponse,
} from './analytics.service';
import { ButtonComponent } from '../../../shared/ui/button/button.component';
import { InputComponent } from '../../../shared/ui/input/input.component';
import { DatetimePickerComponent } from '../../../shared/ui/datetime-picker/datetime-picker.component';
import { ToastService } from '../../../shared/ui/toast/toast.service';

// Palet warna chart konsisten sama summary-detail (donut CSS conic-gradient) biar visual selaras
// di seluruh menu analytics/summary.
const CHART_COLORS = ['#4f46e5', '#7c3aed', '#0ea5e9', '#14b8a6', '#f59e0b', '#ef4444'];

@Component({
  selector: 'app-analytics',
  standalone: true,
  imports: [CommonModule, FormsModule, ReactiveFormsModule, ButtonComponent, InputComponent, DatetimePickerComponent, LucideAngularModule],
  templateUrl: './analytics.component.html',
  styleUrl: './analytics.component.scss',
})
export class AnalyticsComponent implements OnInit {
  quizzes: Quiz[] = [];
  selectedQuizId: number | null = null;
  analytics: QuizAnalyticsResponse | null = null;
  isLoading = false;
  isExporting: AnalyticsExportFormat | null = null;

  filterForm: ReturnType<FormBuilder['group']>;
  chartColors = CHART_COLORS;

  constructor(
    private fb: FormBuilder,
    private quizService: QuizService,
    private analyticsService: AnalyticsService,
    private toast: ToastService,
  ) {
    this.filterForm = this.fb.group({
      start_date: [''],
      end_date: [''],
      respondent: [''],
      min_score: [null as number | null],
      max_score: [null as number | null],
      group_by: ['day' as 'day' | 'hour'],
    });
  }

  // Muat daftar quiz buat dropdown pemilih, lalu otomatis pilih quiz pertama biar halaman gak kosong.
  ngOnInit(): void {
    this.quizService.list({ per_page: 100, sort_by: 'title', sort_dir: 'asc' }).subscribe({
      next: (result) => {
        this.quizzes = result.data;
        if (this.quizzes.length > 0) {
          this.selectedQuizId = this.quizzes[0].id;
          this.loadAnalytics();
        }
      },
      error: () => this.toast.error('Gagal memuat daftar quiz.'),
    });
  }

  // Dipanggil pas user ganti pilihan quiz di dropdown; muat ulang analytics buat quiz yang baru dipilih.
  onQuizChange(): void {
    this.loadAnalytics();
  }

  // Dipanggil dari tombol "Terapkan" filter; muat ulang analytics pakai nilai filter form saat ini.
  applyFilter(): void {
    this.loadAnalytics();
  }

  // Kosongkan semua field filter balik ke default, lalu muat ulang analytics tanpa filter (semua data).
  resetFilter(): void {
    this.filterForm.reset({ start_date: '', end_date: '', respondent: '', min_score: null, max_score: null, group_by: 'day' });
    this.loadAnalytics();
  }

  // Baca nilai form filter jadi AnalyticsFilter siap kirim ke API; string/angka kosong diubah
  // ke undefined biar service gak nempelin query param yang gak perlu.
  private currentFilter(): AnalyticsFilter {
    const raw = this.filterForm.getRawValue();
    return {
      start_date: raw.start_date || undefined,
      end_date: raw.end_date || undefined,
      respondent: raw.respondent || undefined,
      min_score: raw.min_score ?? undefined,
      max_score: raw.max_score ?? undefined,
      group_by: raw.group_by ?? 'day',
    };
  }

  // Hit API analytics buat quiz yang lagi dipilih dengan filter aktif, dipanggil pas pertama load,
  // ganti quiz, submit filter, atau reset filter.
  loadAnalytics(): void {
    if (this.selectedQuizId === null) return;

    this.isLoading = true;
    this.analyticsService.getAnalytics(this.selectedQuizId, this.currentFilter()).subscribe({
      next: (data) => {
        this.isLoading = false;
        this.analytics = data;
      },
      error: () => {
        this.isLoading = false;
        this.toast.error('Gagal memuat data analytics.');
      },
    });
  }

  // Tinggi bar trend chart dalam persen, dinormalisasi ke titik dengan jumlah submission terbanyak
  // biar bar tertinggi selalu penuh (100%) dan yang lain proporsional.
  trendBarHeight(count: number): number {
    if (!this.analytics || this.analytics.trend.length === 0) return 0;
    const max = Math.max(...this.analytics.trend.map((t) => t.count), 1);
    return Math.round((count / max) * 100);
  }

  // Warna chart per-index, diulang (modulo) kalau jumlah opsi/kategori lebih banyak dari palet.
  colorFor(index: number): string {
    return this.chartColors[index % this.chartColors.length];
  }

  // Ekspor hasil analytics (summary + raw submission) sesuai filter aktif, lalu trigger download
  // langsung di browser lewat Blob URL sementara (di-revoke setelah klik biar gak nyangkut di memory).
  export(format: AnalyticsExportFormat): void {
    if (this.selectedQuizId === null) return;

    this.isExporting = format;
    this.analyticsService.exportAnalytics(this.selectedQuizId, format, this.currentFilter()).subscribe({
      next: (blob) => {
        this.isExporting = null;
        const url = window.URL.createObjectURL(blob);
        const anchor = document.createElement('a');
        anchor.href = url;
        anchor.download = `analytics-quiz-${this.selectedQuizId}.${format}`;
        anchor.click();
        window.URL.revokeObjectURL(url);
        this.toast.success('Export berhasil diunduh.');
      },
      error: () => {
        this.isExporting = null;
        this.toast.error('Gagal export data.');
      },
    });
  }
}
