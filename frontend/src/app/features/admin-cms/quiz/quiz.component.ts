import { Component, OnInit, TemplateRef, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { LucideAngularModule } from 'lucide-angular';

import { Quiz, QuizPayload, QuizService } from './quiz.service';
import { InputComponent } from '../../../shared/ui/input/input.component';
import { DatetimePickerComponent } from '../../../shared/ui/datetime-picker/datetime-picker.component';
import { ButtonComponent } from '../../../shared/ui/button/button.component';
import { ModalComponent } from '../../../shared/ui/modal/modal.component';
import {
  DataTableColumn,
  DataTableComponent,
  DataTableQuery,
} from '../../../shared/ui/data-table/data-table.component';
import { ToastService } from '../../../shared/ui/toast/toast.service';
import { fieldError } from '../../../shared/utils/form-error.util';
import { confirmAndDelete } from '../../../shared/utils/confirm-delete.util';
import { loadPagedList } from '../../../shared/utils/load-paged-list.util';

@Component({
  selector: 'app-quiz',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    InputComponent,
    DatetimePickerComponent,
    ButtonComponent,
    ModalComponent,
    LucideAngularModule,
    DataTableComponent,
  ],
  templateUrl: './quiz.component.html',
  styleUrl: './quiz.component.scss',
})
export class QuizComponent implements OnInit {
  @ViewChild('typeTpl', { static: true }) typeTpl!: TemplateRef<unknown>;
  @ViewChild('periodTpl', { static: true }) periodTpl!: TemplateRef<unknown>;
  @ViewChild('statusTpl', { static: true }) statusTpl!: TemplateRef<unknown>;
  @ViewChild('aksiTpl', { static: true }) aksiTpl!: TemplateRef<unknown>;

  quizzes: Quiz[] = [];
  columns: DataTableColumn[] = [];
  totalCount = 0;
  pageSize = 10;
  isModalOpen = false;
  isSaving = false;
  sharingQuizId: number | null = null;
  editingId: number | null = null; // null = create
  private currentQuery: DataTableQuery = {};

  form: ReturnType<FormBuilder['group']>;

  constructor(
    private fb: FormBuilder,
    private quizService: QuizService,
    private toast: ToastService,
  ) {
    this.form = this.fb.group({
      title: ['', Validators.required],
      type: ['quiz', Validators.required],
      start_input: ['', Validators.required],
      end_input: ['', Validators.required],
      description: [''],
      max_point: [null],
      passing_grade: [null],
      status: ['active', Validators.required],
    });

    // Ganti tipe -> format start/end beda (jam vs tanggal+jam) & max_point cuma relevan buat quiz
    // (wajib diisi), jadi kosongin dulu biar gak ada value nyangkut dari format tipe sebelumnya.
    this.form.get('type')!.valueChanges.subscribe((type) => {
      this.form.patchValue({ start_input: '', end_input: '', max_point: null, passing_grade: null });
      this.applyMaxPointValidator(type);
    });
    this.applyMaxPointValidator(this.form.get('type')!.value);
  }

  // max_point dan passing_grade cuma wajib untuk quiz, sedangkan survey tidak memakai scoring lulus.
  private applyMaxPointValidator(type: string): void {
    const maxPointControl = this.form.get('max_point')!;
    const passingGradeControl = this.form.get('passing_grade')!;
    if (type === 'quiz') {
      maxPointControl.setValidators([Validators.required, Validators.min(0)]);
      passingGradeControl.setValidators([Validators.required, Validators.min(0)]);
    } else {
      maxPointControl.clearValidators();
      passingGradeControl.clearValidators();
      maxPointControl.setValue(null);
      passingGradeControl.setValue(null);
    }
    maxPointControl.updateValueAndValidity();
    passingGradeControl.updateValueAndValidity();
  }

  // Setup kolom tabel dan langsung load data pertama kali komponen dibuka.
  ngOnInit(): void {
    this.columns = [
      { name: 'Title', prop: 'title' },
      { name: 'Type', prop: 'type', cellTemplate: this.typeTpl },
      { name: 'Period', prop: 'start_time', cellTemplate: this.periodTpl },
      { name: 'Status', prop: 'status', cellTemplate: this.statusTpl },
      {
        name: 'Action',
        sortable: false,
        cellTemplate: this.aksiTpl,
        headerClass: 'app-data-table__cell--actions',
        cellClass: 'app-data-table__cell--actions',
      },
    ];
    this.loadQuizzes();
  }

  // Dipakai template buat toggle tampilan field yang cuma relevan buat type quiz (misal max_point).
  get isQuizType(): boolean {
    return this.form.get('type')?.value === 'quiz';
  }

  // Ambil data quiz dari service pake query (search/sort/page) yang lagi aktif, terus isi state tabel.
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

  // Dipanggil data-table pas user ganti search/sort/page, simpan query terbaru terus reload data.
  onTableQueryChange(query: DataTableQuery): void {
    this.currentQuery = query;
    this.loadQuizzes();
  }

  // Format kolom "Period" di tabel: quiz cuma tampil jam, survey tampil tanggal+jam lengkap.
  formatPeriod(row: Quiz): string {
    // Gak ada start/end time sama sekali, tampilin strip aja.
    if (!row.start_time && !row.end_time) return '-';
    if (row.type === 'quiz') {
      const start = row.start_time ? row.start_time.slice(11, 16) : '?';
      const end = row.end_time ? row.end_time.slice(11, 16) : '?';
      return `${start} - ${end}`;
    }
    const fmt = (v: string | null) => (v ? `${v.slice(0, 10)} ${v.slice(11, 16)}` : '?');
    return `${fmt(row.start_time)} - ${fmt(row.end_time)}`;
  }

  // Ambil pesan error field form, dipakai template buat nampilin error di bawah input.
  fieldError(name: string): string {
    return fieldError(this.form, name);
  }

  // Buka modal buat bikin quiz baru, reset form ke default dan pastiin validator max_point sesuai type quiz.
  openCreateModal(): void {
    this.editingId = null;
    this.form.reset({
      title: '', type: 'quiz', start_input: '', end_input: '', description: '', max_point: null, passing_grade: null, status: 'active',
    });
    this.applyMaxPointValidator('quiz');
    this.isModalOpen = true;
  }

  // Buka modal edit, isi form dari data quiz yang dipilih (termasuk convert format waktu BE -> input).
  openEditModal(quiz: Quiz): void {
    this.editingId = quiz.id;
    const isQuiz = quiz.type === 'quiz';
    this.form.reset({
      title: quiz.title ?? '',
      type: quiz.type ?? 'quiz',
      start_input: this.extractInputValue(quiz.start_time, isQuiz),
      end_input: this.extractInputValue(quiz.end_time, isQuiz),
      description: quiz.description ?? '',
      max_point: quiz.max_point,
      passing_grade: quiz.passing_grade,
      status: quiz.status ?? 'active',
    });
    this.applyMaxPointValidator(quiz.type ?? 'quiz');
    this.isModalOpen = true;
  }

  // Tutup modal create/edit tanpa nyimpen apa-apa.
  closeModal(): void {
    this.isModalOpen = false;
  }

  // Submit form create/edit quiz: validasi, susun payload sesuai type, baru hit API create/update.
  submit(): void {
    // Form belum valid, tandain semua field touched biar error langsung kelihatan, terus stop.
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const isQuiz = raw.type === 'quiz';

    const payload: QuizPayload = {
      title: raw.title || null,
      type: raw.type as QuizPayload['type'],
      start_time: this.combineDateTime(raw.start_input!, isQuiz),
      end_time: this.combineDateTime(raw.end_input!, isQuiz),
      description: raw.description || null,
      max_point: isQuiz && raw.max_point !== null && raw.max_point !== '' ? Number(raw.max_point) : null,
      passing_grade: isQuiz && raw.passing_grade !== null && raw.passing_grade !== '' ? Number(raw.passing_grade) : null,
      status: raw.status || null,
    };

    // Validasi cross-field: end_time gak boleh lebih awal dari start_time.
    if (payload.start_time && payload.end_time && payload.start_time > payload.end_time) {
      this.toast.error('Waktu selesai tidak boleh sebelum waktu mulai.');
      return;
    }
    if (
      isQuiz &&
      payload.max_point !== null &&
      payload.passing_grade !== null &&
      payload.passing_grade > payload.max_point
    ) {
      this.toast.error('Passing grade tidak boleh melebihi max point.');
      return;
    }

    this.isSaving = true;
    // editingId null berarti mode create, kalau ada isinya berarti mode update.
    const request = this.editingId
      ? this.quizService.update(this.editingId, payload)
      : this.quizService.create(payload);

    request.subscribe({
      next: () => {
        this.isSaving = false;
        this.toast.success('Quiz berhasil disimpan.');
        this.closeModal();
        this.loadQuizzes();
      },
      error: (err) => {
        this.isSaving = false;
        const message = typeof err?.error === 'string' && err.error ? err.error : 'Gagal menyimpan quiz.';
        this.toast.error(message);
      },
    });
  }

  // Hapus quiz, minta konfirmasi dulu sebelum beneran manggil API delete.
  remove(quiz: Quiz): void {
    confirmAndDelete(
      `Hapus quiz "${quiz.title ?? ''}"?`,
      () => this.quizService.delete(quiz.id),
      this.toast,
      'Quiz dihapus.',
      'Gagal menghapus quiz.',
      () => this.loadQuizzes(),
    );
  }

  // Generate token share publik lalu copy URL-nya ke clipboard supaya admin bisa langsung bagikan.
  share(quiz: Quiz): void {
    if (!this.canShareQuiz(quiz)) {
      this.toast.error('Link share hanya bisa dibuat untuk quiz yang statusnya active.');
      return;
    }

    this.sharingQuizId = quiz.id;
    this.quizService.shareLink(quiz.id).subscribe({
      next: async (response) => {
        this.sharingQuizId = null;
        if (!response.token) {
          this.toast.error('Token share tidak valid.');
          return;
        }

        const shareUrl = `${window.location.origin}/public-form/${response.token}`;
        try {
          await navigator.clipboard.writeText(shareUrl);
          this.toast.success('Link share berhasil dicopy.');
        } catch {
          this.toast.info(`Link siap dibagikan: ${shareUrl}`);
        }
      },
      error: (err) => {
        this.sharingQuizId = null;
        const message = typeof err?.error === 'string' && err.error ? err.error : 'Gagal membuat link share.';
        this.toast.error(message);
      },
    });
  }

  // Tombol share cuma relevan untuk quiz yang aktif; status inactive disembunyikan supaya admin
  // tidak membagikan form yang memang sudah dinonaktifkan.
  canShareQuiz(quiz: Quiz): boolean {
    return (quiz.status ?? '').toLowerCase() === 'active';
  }

  // "YYYY-MM-DDTHH:mm:ss" (BE) -> "HH:mm" (input time) kalau quiz, atau "YYYY-MM-DDTHH:mm" (input
  // datetime-local) kalau survey.
  private extractInputValue(stored: string | null, isQuiz: boolean): string {
    if (!stored) return '';
    return isQuiz ? stored.slice(11, 16) : stored.slice(0, 16);
  }

  // Input FE -> "YYYY-MM-DDTHH:mm:ss" (BE). Quiz cuma nginput jam, jadi digabung sama tanggal
  // hari ini pas disimpan; survey udah full tanggal+jam dari <input type="datetime-local">.
  private combineDateTime(raw: string, isQuiz: boolean): string | null {
    if (!raw) return null;
    if (isQuiz) {
      const now = new Date();
      const y = now.getFullYear();
      const m = String(now.getMonth() + 1).padStart(2, '0');
      const d = String(now.getDate()).padStart(2, '0');
      return `${y}-${m}-${d}T${raw}:00`;
    }
    return `${raw}:00`;
  }
}
