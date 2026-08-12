import { Component, OnInit, TemplateRef, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormArray, FormBuilder, FormGroup, FormsModule, ReactiveFormsModule, Validators } from '@angular/forms';
import { LucideAngularModule } from 'lucide-angular';

import {
  QuestionBankImportResult,
  QuestionBankItem,
  QuestionBankPayload,
  QuestionBankService,
  QuestionBankType,
} from './question-bank.service';
import {
  DataTableColumn,
  DataTableComponent,
  DataTableQuery,
} from '../../../shared/ui/data-table/data-table.component';
import { ButtonComponent } from '../../../shared/ui/button/button.component';
import { ModalComponent } from '../../../shared/ui/modal/modal.component';
import { InputComponent } from '../../../shared/ui/input/input.component';
import { FilesUploadComponent, UploadedFile } from '../../../shared/ui/files-upload/files-upload.component';
import { ToastService } from '../../../shared/ui/toast/toast.service';
import { fieldError } from '../../../shared/utils/form-error.util';
import { confirmAndDelete } from '../../../shared/utils/confirm-delete.util';
import { loadPagedList } from '../../../shared/utils/load-paged-list.util';

@Component({
  selector: 'app-question-bank',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    LucideAngularModule,
    DataTableComponent,
    ButtonComponent,
    ModalComponent,
    InputComponent,
    FilesUploadComponent,
  ],
  templateUrl: './question-bank.component.html',
  styleUrl: './question-bank.component.scss',
})
export class QuestionBankComponent implements OnInit {
  @ViewChild('typeTpl', { static: true }) typeTpl!: TemplateRef<unknown>;
  @ViewChild('tagsTpl', { static: true }) tagsTpl!: TemplateRef<unknown>;
  @ViewChild('actionTpl', { static: true }) actionTpl!: TemplateRef<unknown>;

  items: QuestionBankItem[] = [];
  columns: DataTableColumn[] = [];
  totalCount = 0;
  pageSize = 10;
  isFormModalOpen = false;
  isImportModalOpen = false;
  isSaving = false;
  isImporting = false;
  editingId: number | null = null; // null = create
  answerSectionError = '';

  allTags: string[] = [];
  activeTag = '';

  importFiles: UploadedFile[] = [];
  importResult: QuestionBankImportResult | null = null;

  private currentQuery: DataTableQuery = {};

  form: ReturnType<FormBuilder['group']>;

  constructor(
    private fb: FormBuilder,
    private questionBankService: QuestionBankService,
    private toast: ToastService,
  ) {
    this.form = this.fb.group({
      question: ['', Validators.required],
      type_answer: ['multiple_choice', Validators.required],
      have_point: ['no', Validators.required],
      point: [null as number | null],
      rating_max: [5 as number | null],
      tags: [''], // input teks bebas, dipisah koma/semicolon
      answers: this.fb.array<FormGroup>([]),
    });

    this.form.get('type_answer')!.valueChanges.subscribe((type) => this.applyAnswerMode(type as QuestionBankType));
    this.form.get('have_point')!.valueChanges.subscribe((value) => this.applyPointValidator(value === 'yes'));
    this.applyPointValidator(false);
    this.applyAnswerMode('multiple_choice');
  }

  ngOnInit(): void {
    this.columns = [
      { name: 'Question', prop: 'question' },
      { name: 'Type', prop: 'type_answer', cellTemplate: this.typeTpl },
      { name: 'Point', prop: 'point' },
      { name: 'Tags', prop: 'tags', cellTemplate: this.tagsTpl },
      {
        name: 'Action',
        sortable: false,
        cellTemplate: this.actionTpl,
        headerClass: 'app-data-table__cell--actions',
        cellClass: 'app-data-table__cell--actions',
      },
    ];
    this.loadItems();
    this.loadTags();
  }

  get answersArray(): FormArray<FormGroup> {
    return this.form.get('answers') as FormArray<FormGroup>;
  }

  get isOptionBased(): boolean {
    const type = this.form.get('type_answer')?.value;
    return type === 'multiple_choice' || type === 'dropdown' || type === 'checkbox';
  }

  get isCheckbox(): boolean {
    return this.form.get('type_answer')?.value === 'checkbox';
  }

  get isRating(): boolean {
    return this.form.get('type_answer')?.value === 'rating';
  }

  fieldError(name: string): string {
    return fieldError(this.form, name);
  }

  // Ambil daftar soal bank sesuai query DataTable (search/sort/page) + filter tag yang lagi aktif.
  loadItems(): void {
    loadPagedList(
      this.questionBankService.list(this.currentQuery, this.activeTag || null),
      this.toast,
      'Gagal memuat data bank soal.',
      (data, totalCount, pageSize) => {
        this.items = data;
        this.totalCount = totalCount;
        this.pageSize = pageSize;
      },
    );
  }

  loadTags(): void {
    this.questionBankService.listTags().subscribe({
      next: (tags) => (this.allTags = tags),
      error: () => this.toast.error('Gagal memuat daftar tag.'),
    });
  }

  onTableQueryChange(query: DataTableQuery): void {
    this.currentQuery = query;
    this.loadItems();
  }

  // Ganti filter tag aktif dari dropdown, lalu reload dari halaman pertama.
  onTagFilterChange(tag: string): void {
    this.activeTag = tag;
    this.loadItems();
  }

  questionTypeLabel(value: QuestionBankType | string | null): string {
    switch (value) {
      case 'multiple_choice':
        return 'Pilihan Ganda';
      case 'dropdown':
        return 'Dropdown';
      case 'checkbox':
        return 'Checkbox';
      case 'rating':
        return 'Rating';
      case 'free_text':
        return 'Free Text';
      default:
        return '-';
    }
  }

  // Buka form kosong buat bikin soal bank baru.
  openCreateModal(): void {
    this.editingId = null;
    this.answerSectionError = '';
    this.form.reset({
      question: '', type_answer: 'multiple_choice', have_point: 'no', point: null, rating_max: 5, tags: '',
    });
    this.applyPointValidator(false);
    this.applyAnswerMode('multiple_choice');
    this.isFormModalOpen = true;
  }

  // Buka form edit, isi dari data soal bank yang dipilih.
  openEditModal(item: QuestionBankItem): void {
    this.editingId = item.id;
    this.answerSectionError = '';
    const type = (item.type_answer ?? 'multiple_choice') as QuestionBankType;
    const values = item.answers.map((answer) => ({ label: answer.label ?? '', value: answer.value ?? 'false' }));
    this.form.reset({
      question: item.question ?? '',
      type_answer: type,
      have_point: item.point !== null ? 'yes' : 'no',
      point: item.point,
      rating_max: this.deriveRatingMax(item),
      tags: item.tags.join(', '),
    });
    this.applyPointValidator(item.point !== null);
    this.setAnswerControls(type, values);
    this.isFormModalOpen = true;
  }

  closeFormModal(): void {
    this.isFormModalOpen = false;
  }

  addAnswer(): void {
    this.answersArray.push(this.createOptionAnswerGroup('', 'false'));
  }

  removeAnswer(index: number): void {
    if (this.answersArray.length <= 2) return;
    this.answersArray.removeAt(index);
  }

  // Submit form create/edit soal bank.
  save(): void {
    this.answerSectionError = '';
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      this.toast.error('Ada field yang belum valid, cek lagi form-nya.');
      return;
    }

    const raw = this.form.getRawValue();
    const type = raw.type_answer as QuestionBankType;
    const payload: QuestionBankPayload = {
      question: raw.question || null,
      type_answer: type,
      point: raw.have_point === 'yes' && raw.point !== null ? Number(raw.point) : null,
      tags: this.parseTagsInput(raw.tags ?? ''),
      answers: this.buildAnswersPayload(type),
    };

    const answerError = this.validateAnswerPayload(payload);
    if (answerError) {
      this.answerSectionError = answerError;
      this.toast.error(answerError);
      return;
    }

    this.isSaving = true;
    const request = this.editingId
      ? this.questionBankService.update(this.editingId, payload)
      : this.questionBankService.create(payload);

    request.subscribe({
      next: () => {
        this.isSaving = false;
        this.toast.success('Soal bank berhasil disimpan.');
        this.closeFormModal();
        this.loadItems();
        this.loadTags();
      },
      error: (err) => {
        this.isSaving = false;
        const message = typeof err?.error === 'string' && err.error ? err.error : 'Gagal menyimpan soal bank.';
        this.toast.error(message);
      },
    });
  }

  remove(item: QuestionBankItem): void {
    confirmAndDelete(
      `Hapus soal bank "${item.question ?? ''}"?`,
      () => this.questionBankService.delete(item.id),
      this.toast,
      'Soal bank dihapus.',
      'Gagal menghapus soal bank.',
      () => this.loadItems(),
    );
  }

  // === Import ===

  openImportModal(): void {
    this.importFiles = [];
    this.importResult = null;
    this.isImportModalOpen = true;
  }

  closeImportModal(): void {
    this.isImportModalOpen = false;
  }

  downloadTemplate(format: 'csv' | 'xlsx'): void {
    this.questionBankService.downloadTemplate(format).subscribe({
      next: (blob) => {
        const url = window.URL.createObjectURL(blob);
        const anchor = document.createElement('a');
        anchor.href = url;
        anchor.download = `question-bank-template.${format}`;
        anchor.click();
        window.URL.revokeObjectURL(url);
      },
      error: () => this.toast.error('Gagal download contoh file.'),
    });
  }

  runImport(): void {
    const file = this.importFiles[0];
    if (!file) {
      this.toast.error('Pilih file CSV atau XLSX dulu.');
      return;
    }

    this.isImporting = true;
    this.importResult = null;
    this.questionBankService.import(file.file_name, file.file_data).subscribe({
      next: (result) => {
        this.isImporting = false;
        this.importResult = result;
        if (result.created > 0) {
          this.toast.success(`${result.created} soal berhasil diimport.`);
          this.loadItems();
          this.loadTags();
        }
        if (result.errors.length > 0) {
          this.toast.error(`${result.errors.length} baris gagal diimport, cek detail di bawah.`);
        }
      },
      error: (err) => {
        this.isImporting = false;
        const message = typeof err?.error === 'string' && err.error ? err.error : 'Gagal import file.';
        this.toast.error(message);
      },
    });
  }

  // === Private helpers ===

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

  // Ganti mode jawaban ubah bentuk section answer: pilihan/dropdown/checkbox pakai array opsi,
  // rating pakai rentang otomatis, free text gak punya opsi preset sama sekali.
  private applyAnswerMode(type: QuestionBankType): void {
    this.answerSectionError = '';
    switch (type) {
      case 'multiple_choice':
      case 'dropdown':
      case 'checkbox': {
        const current = this.answersArray.getRawValue() as Array<{ label: string; value: string }>;
        this.setAnswerControls(type, current.length ? current : [{ label: '', value: 'true' }, { label: '', value: 'false' }]);
        this.form.get('rating_max')!.clearValidators();
        this.form.get('rating_max')!.updateValueAndValidity({ emitEvent: false });
        break;
      }
      case 'rating':
        this.answersArray.clear();
        this.form.get('rating_max')!.setValidators([Validators.required, Validators.min(2), Validators.max(10)]);
        this.form.get('rating_max')!.updateValueAndValidity({ emitEvent: false });
        break;
      case 'free_text':
        this.answersArray.clear();
        this.form.get('rating_max')!.clearValidators();
        this.form.get('rating_max')!.updateValueAndValidity({ emitEvent: false });
        break;
    }
  }

  private setAnswerControls(type: QuestionBankType, values: Array<{ label: string; value: string }>): void {
    this.answersArray.clear();
    if (type !== 'multiple_choice' && type !== 'dropdown' && type !== 'checkbox') return;
    const nextValues = values.length >= 2 ? values : [{ label: '', value: 'true' }, { label: '', value: 'false' }];
    nextValues.forEach((value) => this.answersArray.push(this.createOptionAnswerGroup(value.label || '', value.value || 'false')));
  }

  private buildAnswersPayload(type: QuestionBankType): Array<{ label: string | null; value: string | null }> {
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

  private validateAnswerPayload(payload: QuestionBankPayload): string {
    if (payload.type_answer === 'multiple_choice' || payload.type_answer === 'dropdown' || payload.type_answer === 'checkbox') {
      if (payload.answers.length < 2) return 'Minimal harus punya 2 jawaban.';
      if (payload.answers.some((answer) => !answer.label)) return 'Semua teks jawaban wajib diisi.';
      if (payload.answers.filter((answer) => answer.value === 'true').length === 0) {
        return 'Harus punya minimal satu jawaban true.';
      }
      return '';
    }
    if (payload.type_answer === 'rating') {
      const max = Number(this.form.get('rating_max')!.value ?? 0);
      if (!Number.isInteger(max) || max < 2 || max > 10) return 'Rentang rating wajib diisi antara 2 sampai 10.';
    }
    return '';
  }

  private deriveRatingMax(item: QuestionBankItem): number {
    if (item.type_answer !== 'rating') return 5;
    const values = item.answers.map((answer) => Number(answer.value ?? 0)).filter((v) => Number.isFinite(v) && v > 0);
    return values.length > 0 ? Math.max(...values) : 5;
  }

  // Tag input teks bebas dipisah koma atau semicolon, trim tiap tag & buang yang kosong/duplikat.
  private parseTagsInput(raw: string): string[] {
    const seen = new Set<string>();
    const tags: string[] = [];
    raw.split(/[,;]/).forEach((part) => {
      const tag = part.trim();
      if (tag && !seen.has(tag)) {
        seen.add(tag);
        tags.push(tag);
      }
    });
    return tags;
  }

  private createOptionAnswerGroup(label: string, value: string): FormGroup {
    return this.fb.group({
      label: [label, Validators.required],
      value: [value, Validators.required],
    });
  }

  trackByIndex(index: number, control: FormGroup): FormGroup {
    return control;
  }
}
