import { Component, OnInit, TemplateRef, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  FormArray,
  FormBuilder,
  FormControl,
  FormGroup,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import { LucideAngularModule } from 'lucide-angular';

import { Quiz, QuizService } from '../quiz/quiz.service';
import {
  Question,
  QuestionAnswerService,
  QuestionPayload,
  QuestionType,
} from './question-answer.service';
import {
  DataTableColumn,
  DataTableComponent,
  DataTableQuery,
} from '../../../shared/ui/data-table/data-table.component';
import { ButtonComponent } from '../../../shared/ui/button/button.component';
import { ModalComponent } from '../../../shared/ui/modal/modal.component';
import { InputComponent } from '../../../shared/ui/input/input.component';
import { ToastService } from '../../../shared/ui/toast/toast.service';
import { fieldError } from '../../../shared/utils/form-error.util';
import { confirmAndDelete } from '../../../shared/utils/confirm-delete.util';
import { loadPagedList } from '../../../shared/utils/load-paged-list.util';

@Component({
  selector: 'app-question-answer',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    LucideAngularModule,
    DataTableComponent,
    ButtonComponent,
    ModalComponent,
    InputComponent,
  ],
  templateUrl: './question-answer.component.html',
  styleUrl: './question-answer.component.scss',
})
export class QuestionAnswerComponent implements OnInit {
  @ViewChild('typeTpl', { static: true }) typeTpl!: TemplateRef<unknown>;
  @ViewChild('periodTpl', { static: true }) periodTpl!: TemplateRef<unknown>;
  @ViewChild('actionTpl', { static: true }) actionTpl!: TemplateRef<unknown>;

  quizzes: Quiz[] = [];
  columns: DataTableColumn[] = [];
  totalCount = 0;
  pageSize = 10;
  isModalOpen = false;
  isQuestionSaving = false;
  questionsLoading = false;
  questions: Question[] = [];
  selectedQuiz: Quiz | null = null;
  editingQuestionId: number | null = null;
  isFormVisible = false;
  answerSectionError = '';
  pointLimitError = '';
  private currentQuery: DataTableQuery = {};

  form: ReturnType<FormBuilder['group']>;

  constructor(
    private fb: FormBuilder,
    private quizService: QuizService,
    private questionService: QuestionAnswerService,
    private toast: ToastService,
  ) {
    this.form = this.fb.group({
      question: ['', Validators.required],
      type_answer: ['multiple_choice', Validators.required],
      have_point: ['no', Validators.required],
      point: [null as number | null],
      rating_max: [5 as number | null],
      answers: this.fb.array<FormGroup>([]),
      // matrix_rows cuma dipakai buat type_answer="matrix" (daftar baris pernyataan).
      matrix_rows: this.fb.array<FormGroup>([]),
    });

    this.form.get('type_answer')!.valueChanges.subscribe((type) => this.applyAnswerMode(type as QuestionType));
    this.form.get('have_point')!.valueChanges.subscribe((value) => {
      this.pointLimitError = '';
      this.applyPointValidator(value === 'yes');
    });
    this.form.get('point')!.valueChanges.subscribe(() => {
      this.pointLimitError = '';
    });
    this.applyPointValidator(false);
    this.applyAnswerMode('multiple_choice');
  }

  // Setup kolom tabel quiz dan langsung load data quiz yang bisa dipilih untuk manajemen question.
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

  get answersArray(): FormArray<FormGroup> {
    return this.form.get('answers') as FormArray<FormGroup>;
  }

  get matrixRowsArray(): FormArray<FormGroup> {
    return this.form.get('matrix_rows') as FormArray<FormGroup>;
  }

  // Ambil pesan error form control umum (required/min/dll) biar template tetap ringkas.
  fieldError(name: string): string {
    return fieldError(this.form, name);
  }

  // Field point punya 2 sumber error: validator biasa dan rule total point quiz.
  pointError(): string {
    return this.fieldError('point') || this.pointLimitError;
  }

  get isMultipleChoice(): boolean {
    return this.form.get('type_answer')?.value === 'multiple_choice';
  }

  // isOptionBased true buat 3 tipe yang sama-sama pakai daftar opsi label + status benar/salah:
  // pilihan ganda (pilih 1), dropdown (pilih 1 dari select), checkbox (boleh pilih lebih dari 1).
  get isOptionBased(): boolean {
    const type = this.form.get('type_answer')?.value;
    return type === 'multiple_choice' || type === 'dropdown' || type === 'checkbox';
  }

  get isCheckbox(): boolean {
    return this.form.get('type_answer')?.value === 'checkbox';
  }

  get isMatrix(): boolean {
    return this.form.get('type_answer')?.value === 'matrix';
  }

  get isRating(): boolean {
    return this.form.get('type_answer')?.value === 'rating';
  }

  get isFreeText(): boolean {
    return this.form.get('type_answer')?.value === 'free_text';
  }

  // Ambil daftar quiz dari API untuk ditampilkan di tabel utama menu Question & Answer.
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

  // Format kolom period: quiz tampil jam, survey tampil tanggal yang lebih ramah dibaca
  // (contoh: 8 May 2026 16:00).
  formatPeriod(row: Quiz): string {
    if (!row.start_time && !row.end_time) return '-';
    if (row.type === 'quiz') {
      const start = row.start_time ? row.start_time.slice(11, 16) : '?';
      const end = row.end_time ? row.end_time.slice(11, 16) : '?';
      return `${start} - ${end}`;
    }
    const fmt = (v: string | null) => (v ? this.formatSurveyDateTime(v) : '?');
    return `${fmt(row.start_time)} - ${fmt(row.end_time)}`;
  }

  // Buka modal manajemen question untuk quiz tertentu, lalu load semua question miliknya.
  openManageQuestions(quiz: Quiz): void {
    this.selectedQuiz = quiz;
    this.isModalOpen = true;
    this.cancelQuestionForm();
    this.loadQuestions();
  }

  closeModal(): void {
    this.isModalOpen = false;
    this.selectedQuiz = null;
    this.questions = [];
    this.cancelQuestionForm();
  }

  // Load semua question existing buat quiz yang sedang aktif di modal.
  loadQuestions(): void {
    if (!this.selectedQuiz) return;
    this.questionsLoading = true;
    this.questionService.listByQuiz(this.selectedQuiz.id).subscribe({
      next: (questions) => {
        this.questionsLoading = false;
        this.questions = questions;
      },
      error: (err) => {
        this.questionsLoading = false;
        const message = typeof err?.error === 'string' && err.error ? err.error : 'Gagal memuat question.';
        this.toast.error(message);
      },
    });
  }

  // Siapkan form kosong untuk bikin question baru.
  startCreateQuestion(): void {
    this.editingQuestionId = null;
    this.isFormVisible = true;
    this.answerSectionError = '';
    this.pointLimitError = '';
    this.form.reset({
      question: '',
      type_answer: 'multiple_choice',
      have_point: 'no',
      point: null,
      rating_max: 5,
    });
    this.form.markAsPristine();
    this.form.markAsUntouched();
    this.applyPointValidator(false);
    this.applyAnswerMode('multiple_choice');
  }

  // Isi form dari data question existing saat user klik edit.
  editQuestion(question: Question): void {
    this.editingQuestionId = question.id;
    this.isFormVisible = true;
    this.answerSectionError = '';
    this.pointLimitError = '';
    const type = (question.type_answer ?? 'multiple_choice') as QuestionType;
    const values = question.answers.map((answer) => ({
      label: answer.label ?? '',
      value: answer.value ?? 'false',
    }));
    this.form.reset({
      question: question.question ?? '',
      type_answer: type,
      have_point: question.point !== null ? 'yes' : 'no',
      point: question.point,
      rating_max: this.deriveRatingMax(question),
    });
    this.applyPointValidator(question.point !== null);
    this.setAnswerControls(type, values);
    // Cuma isi matrix_rows kalau tipenya emang matrix -- kalau dipaksa isi buat tipe lain,
    // setMatrixRowControls([]) fallback ke 1 baris kosong ber-validator required dan bikin form invalid diam-diam.
    if (type === 'matrix') {
      this.setMatrixRowControls(question.matrix_rows.map((row) => row.row_label ?? ''));
    } else {
      this.matrixRowsArray.clear();
    }
  }

  // Reset state form question tanpa menutup modal daftar question.
  cancelQuestionForm(): void {
    this.isFormVisible = false;
    this.editingQuestionId = null;
    this.answerSectionError = '';
    this.pointLimitError = '';
    this.form.reset({
      question: '',
      type_answer: 'multiple_choice',
      have_point: 'no',
      point: null,
      rating_max: 5,
    });
    this.answersArray.clear();
    this.matrixRowsArray.clear();
  }

  // Tambah 1 baris opsi jawaban baru untuk mode pilihan ganda/dropdown/checkbox.
  addMultipleChoiceAnswer(): void {
    this.answersArray.push(this.createOptionAnswerGroup('', 'false'));
  }

  // Hapus 1 opsi jawaban, tapi sisakan minimal 2 opsi sesuai rule pilihan ganda/dropdown/checkbox.
  removeMultipleChoiceAnswer(index: number): void {
    if (this.answersArray.length <= 2) return;
    this.answersArray.removeAt(index);
  }

  // Tambah 1 baris pernyataan baru untuk mode matrix.
  addMatrixRow(): void {
    this.matrixRowsArray.push(this.createMatrixRowGroup(''));
  }

  // Hapus 1 baris pernyataan matrix, sisakan minimal 1 baris.
  removeMatrixRow(index: number): void {
    if (this.matrixRowsArray.length <= 1) return;
    this.matrixRowsArray.removeAt(index);
  }

  // Submit form create/update question: validasi FE dulu, baru kirim ke API.
  saveQuestion(): void {
    this.answerSectionError = '';
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      this.toast.error('Ada field yang belum valid, cek lagi form-nya.');
      return;
    }

    const raw = this.form.getRawValue();
    const type = raw.type_answer as QuestionType;
    const payload: QuestionPayload = {
      quiz_id: this.selectedQuiz?.id ?? null,
      question: raw.question || null,
      type_answer: type,
      point: raw.have_point === 'yes' && raw.point !== null ? Number(raw.point) : null,
      answers: this.buildAnswersPayload(type),
      matrix_rows:
        type === 'matrix'
          ? this.matrixRowsArray.getRawValue().map((row) => ({ row_label: row['row_label'] || null }))
          : [],
    };

    const answerError = this.validateAnswerPayload(payload);
    if (answerError) {
      this.answerSectionError = answerError;
      this.toast.error(answerError);
      return;
    }
    const pointError = this.validatePointLimit(payload.point);
    if (pointError) {
      this.pointLimitError = pointError;
      this.toast.error(pointError);
      return;
    }

    this.isQuestionSaving = true;
    const request = this.editingQuestionId
      ? this.questionService.update(this.editingQuestionId, payload)
      : this.questionService.create(payload);

    request.subscribe({
      next: () => {
        this.isQuestionSaving = false;
        this.toast.success('Question berhasil disimpan.');
        this.cancelQuestionForm();
        this.loadQuestions();
      },
      error: (err) => {
        this.isQuestionSaving = false;
        const message = typeof err?.error === 'string' && err.error ? err.error : 'Gagal menyimpan question.';
        this.toast.error(message);
      },
    });
  }

  // Hapus question dari quiz aktif setelah konfirmasi.
  removeQuestion(question: Question): void {
    confirmAndDelete(
      `Hapus question "${question.question ?? ''}"?`,
      () => this.questionService.delete(question.id),
      this.toast,
      'Question dihapus.',
      'Gagal menghapus question.',
      () => this.loadQuestions(),
    );
  }

  // Mapping value teknis type_answer -> label yang lebih enak dibaca di UI.
  questionTypeLabel(value: QuestionType | string | null): string {
    switch (value) {
      case 'multiple_choice':
        return 'Pilihan Ganda';
      case 'dropdown':
        return 'Dropdown';
      case 'checkbox':
        return 'Checkbox';
      case 'matrix':
        return 'Matrix';
      case 'rating':
        return 'Rating';
      case 'free_text':
        return 'Free Text';
      default:
        return '-';
    }
  }

  // Ringkasan answer per card: free text cukup label umum, rating jadi rentang, matrix nampilin
  // jumlah baris & kolom, pilihan ganda/dropdown/checkbox nampilin teks opsi + status benar/salahnya.
  answerSummary(question: Question): string {
    if (question.type_answer === 'free_text') return 'Jawaban bebas';
    if (question.type_answer === 'rating') {
      const max = this.deriveRatingMax(question);
      return max > 0 ? `Rentang 1 - ${max}` : '-';
    }
    if (question.type_answer === 'matrix') {
      const columns = question.answers.map((answer) => answer.label ?? '-').join(', ');
      return `${question.matrix_rows.length} baris × kolom: ${columns}`;
    }
    return question.answers
      .map((answer) => `${answer.label ?? '-'} (${answer.value === 'true' ? 'benar' : 'salah'})`)
      .join(', ');
  }

  // Track by control instance (bukan index) supaya *ngFor rebuild view saat FormArray di-clear+push ulang
  // (ex: ganti Type Answer) -- kalau track by index, formGroupName lama nempel ke control basi yang udah dibuang.
  trackByIndex(index: number, control: FormGroup): FormGroup {
    return control;
  }

  // Point baru wajib diisi hanya kalau user pilih "Have Point = Yes".
  private applyPointValidator(hasPoint: boolean): void {
    const control = this.form.get('point')!;
    if (hasPoint) {
      control.setValidators([Validators.required, Validators.min(1)]);
    } else {
      control.clearValidators();
      control.setValue(null);
    }
    control.updateValueAndValidity({ emitEvent: false });
  }

  // Ganti mode jawaban akan mengubah bentuk section answer: pilihan ganda/dropdown/checkbox pakai
  // array opsi, matrix pakai array baris + array kolom, rating pakai rentang otomatis, free text
  // gak punya opsi preset sama sekali.
  private applyAnswerMode(type: QuestionType): void {
    this.answerSectionError = '';
    switch (type) {
      case 'multiple_choice':
      case 'dropdown':
      case 'checkbox': {
        const currentAnswers = this.answersArray.getRawValue() as Array<{ label: string; value: string }>;
        this.setAnswerControls(
          type,
          this.answersArray.length
            ? currentAnswers
            : [
                { label: '', value: 'true' },
                { label: '', value: 'false' },
              ],
        );
        this.form.get('rating_max')!.clearValidators();
        this.matrixRowsArray.clear();
        break;
      }
      case 'matrix': {
        const currentColumns = this.answersArray.getRawValue() as Array<{ label: string; value: string }>;
        this.setAnswerControls(type, currentColumns.length ? currentColumns : [{ label: '', value: '' }, { label: '', value: '' }]);
        if (this.matrixRowsArray.length === 0) {
          this.setMatrixRowControls(['']);
        }
        this.form.get('rating_max')!.clearValidators();
        this.form.get('rating_max')!.updateValueAndValidity({ emitEvent: false });
        break;
      }
      case 'rating':
        this.answersArray.clear();
        this.matrixRowsArray.clear();
        this.form.get('rating_max')!.setValidators([Validators.required, Validators.min(2), Validators.max(10)]);
        this.form.get('rating_max')!.updateValueAndValidity({ emitEvent: false });
        break;
      case 'free_text':
        this.answersArray.clear();
        this.matrixRowsArray.clear();
        this.form.get('rating_max')!.clearValidators();
        this.form.get('rating_max')!.updateValueAndValidity({ emitEvent: false });
        break;
    }
  }

  // Isi ulang FormArray answer opsi (dipakai pilihan ganda/dropdown/checkbox/kolom matrix), dipakai
  // saat create default maupun edit.
  private setAnswerControls(type: QuestionType, values: Array<{ label: string; value: string }>): void {
    this.answersArray.clear();
    if (type !== 'multiple_choice' && type !== 'dropdown' && type !== 'checkbox' && type !== 'matrix') return;
    const nextValues =
      values.length >= 2
        ? values
        : [
            { label: '', value: 'true' },
            { label: '', value: 'false' },
          ];
    nextValues.forEach((value) => {
      this.answersArray.push(this.createOptionAnswerGroup(value.label || '', value.value || 'false'));
    });
    this.form.get('rating_max')!.clearValidators();
    this.form.get('rating_max')!.updateValueAndValidity({ emitEvent: false });
  }

  // Isi ulang FormArray baris pernyataan matrix, dipakai saat create default maupun edit.
  private setMatrixRowControls(rowLabels: string[]): void {
    this.matrixRowsArray.clear();
    const nextLabels = rowLabels.length > 0 ? rowLabels : [''];
    nextLabels.forEach((label) => this.matrixRowsArray.push(this.createMatrixRowGroup(label)));
  }

  // Bentuk payload answers beda per tipe: free text kosong, rating auto-generate 1..N, matrix
  // ambil dari array kolom, tipe pilihan lain ambil dari input teks + status benar/salah user.
  private buildAnswersPayload(type: QuestionType): Array<{ label: string | null; value: string | null }> {
    if (type === 'free_text') return [];
    if (type === 'rating') {
      const max = Number(this.form.get('rating_max')!.value ?? 0);
      return Array.from({ length: max }, (_, index) => {
        const value = String(index + 1);
        return { label: value, value };
      });
    }
    return this.answersArray.getRawValue().map((answer) => ({
      label: answer['label'] || null,
      value: answer['value'] || null,
    }));
  }

  // Validasi FE khusus answer yang tidak tercakup validator field biasa.
  private validateAnswerPayload(payload: QuestionPayload): string {
    if (payload.type_answer === 'multiple_choice' || payload.type_answer === 'dropdown' || payload.type_answer === 'checkbox') {
      if (payload.answers.length < 2) {
        return 'Minimal harus punya 2 jawaban.';
      }
      if (payload.answers.some((answer) => !answer.label)) {
        return 'Semua teks jawaban wajib diisi.';
      }
      const trueCount = payload.answers.filter((answer) => answer.value === 'true').length;
      if (trueCount === 0) {
        return 'Harus punya minimal satu jawaban true.';
      }
      return '';
    }
    if (payload.type_answer === 'matrix') {
      if (payload.matrix_rows.length < 1 || payload.matrix_rows.some((row) => !row.row_label)) {
        return 'Minimal 1 baris pernyataan dan teksnya wajib diisi.';
      }
      if (payload.answers.length < 2 || payload.answers.some((answer) => !answer.label)) {
        return 'Minimal 2 kolom skala dan teksnya wajib diisi.';
      }
      return '';
    }
    if (payload.type_answer === 'rating') {
      const max = Number(this.form.get('rating_max')!.value ?? 0);
      if (!Number.isInteger(max) || max < 2 || max > 10) {
        return 'Rentang rating wajib diisi antara 2 sampai 10.';
      }
    }
    return '';
  }

  // Jaga total point question yang diinput user tidak melebihi max_point quiz yang dipilih.
  private validatePointLimit(point: number | null): string {
    if (point === null || !this.selectedQuiz?.max_point) {
      return '';
    }
    const usedPoint = this.questions
      .filter((question) => question.id !== this.editingQuestionId)
      .reduce((total, question) => total + (question.point ?? 0), 0);

    if (usedPoint + point > this.selectedQuiz.max_point) {
      return `Total point question melebihi max point quiz (${this.selectedQuiz.max_point}).`;
    }
    return '';
  }

  // Dari answer rating existing, ambil angka maksimum buat ngisi field rentang saat edit.
  private deriveRatingMax(question: Question): number {
    if (question.type_answer !== 'rating') return 5;
    const values = question.answers
      .map((answer) => Number(answer.value ?? 0))
      .filter((value) => Number.isFinite(value) && value > 0);
    return values.length > 0 ? Math.max(...values) : 5;
  }

  // String datetime backend -> format UI "8 May 2026 16:00" khusus tampilan survey.
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

  // Factory 1 row opsi (pilihan ganda/dropdown/checkbox/kolom matrix): teks jawaban + flag benar/salah.
  private createOptionAnswerGroup(label: string, value: string): FormGroup {
    return this.fb.group({
      label: [label, Validators.required],
      value: [value, Validators.required],
    });
  }

  // Factory 1 baris pernyataan matrix: cuma butuh teksnya, gak ada status benar/salah.
  private createMatrixRowGroup(label: string): FormGroup {
    return this.fb.group({
      row_label: [label, Validators.required],
    });
  }
}
