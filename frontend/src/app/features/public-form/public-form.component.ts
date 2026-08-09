import { Component, OnDestroy, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  FormArray,
  FormBuilder,
  FormGroup,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import { Subscription } from 'rxjs';

import {
  PublicFormDetail,
  PublicQuestion,
  PublicFormService,
  PublicFormSubmitPayload,
  PublicFormSubmitResult,
} from './public-form.service';
import { InputComponent } from '../../shared/ui/input/input.component';
import { ButtonComponent } from '../../shared/ui/button/button.component';
import { ToastService } from '../../shared/ui/toast/toast.service';

type QuestionFormGroup = FormGroup;

interface StoredQuizSession {
  email: string;
  started: boolean;
  answers: Array<{
    question_id: number;
    question_answer_id: number | null;
    answer_text: string | null;
  }>;
}

@Component({
  selector: 'app-public-form',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, InputComponent, ButtonComponent],
  templateUrl: './public-form.component.html',
  styleUrl: './public-form.component.scss',
})
export class PublicFormComponent implements OnInit, OnDestroy {
  loading = true;
  loadError = '';
  isSubmitting = false;
  hasStartedQuiz = false;
  countdownLabel = '';
  submitResult: PublicFormSubmitResult | null = null;
  detail: PublicFormDetail | null = null;
  accessForm: ReturnType<FormBuilder['group']>;
  questionForm: ReturnType<FormBuilder['group']>;
  private autoSubmitTriggered = false;
  private formSubscription = new Subscription();
  private countdownTimer: ReturnType<typeof setInterval> | null = null;

  constructor(
    private fb: FormBuilder,
    private route: ActivatedRoute,
    private publicFormService: PublicFormService,
    private toast: ToastService,
  ) {
    this.accessForm = this.fb.group({
      email: ['', [Validators.required, Validators.email]],
    });

    this.questionForm = this.fb.group({
      answers: this.fb.array<QuestionFormGroup>([]),
    });
  }

  ngOnInit(): void {
    const token = this.route.snapshot.paramMap.get('token') ?? '';
    if (!token) {
      this.loading = false;
      this.loadError = 'Link publik tidak valid.';
      return;
    }
    this.loadForm(token);
  }

  ngOnDestroy(): void {
    this.clearCountdown();
    this.formSubscription.unsubscribe();
  }

  get answersArray(): FormArray<QuestionFormGroup> {
    return this.questionForm.get('answers') as FormArray<QuestionFormGroup>;
  }

  get isQuiz(): boolean {
    return this.detail?.type === 'quiz';
  }

  get isSurvey(): boolean {
    return this.detail?.type === 'survey';
  }

  get canFillNow(): boolean {
    return this.detail?.state === 'active';
  }

  // loadForm ambil detail link publik lalu siapkan form answer sesuai question yang diterima.
  loadForm(token: string): void {
    this.loading = true;
    this.loadError = '';
    this.publicFormService.getByToken(token).subscribe({
      next: (detail) => {
        this.loading = false;
        this.detail = detail;
        this.submitResult = null;
        this.hasStartedQuiz = false;
        this.autoSubmitTriggered = false;
        this.buildAnswerControls(detail.questions);
        this.restoreQuizSession();

        if (detail.type === 'quiz' && detail.state !== 'active') {
          this.clearQuizSession();
        }
        if (detail.type === 'survey' && detail.state === 'active') {
          return;
        }
        if (detail.type !== 'quiz' || this.hasStartedQuiz) {
          this.updateCountdown();
        }
      },
      error: (err) => {
        this.loading = false;
        this.detail = null;
        this.loadError = typeof err?.error === 'string' && err.error ? err.error : 'Gagal memuat form publik.';
      },
    });
  }

  // startQuiz ngecek email dulu, baru nyalain timer visual dan munculkan daftar question quiz.
  startQuiz(): void {
    if (!this.isQuiz) return;
    if (this.accessForm.invalid) {
      this.accessForm.markAllAsTouched();
      return;
    }
    this.hasStartedQuiz = true;
    this.persistQuizSession();
    this.startCountdown();
  }

  // submit dipakai tombol kirim manual maupun auto-submit saat timer quiz habis.
  submit(autoSubmit = false): void {
    if (!this.detail) return;
    if (!autoSubmit && this.questionForm.invalid) {
      this.questionForm.markAllAsTouched();
      this.toast.error('Semua jawaban wajib diisi.');
      return;
    }

    const payload = this.buildSubmitPayload();
    this.isSubmitting = true;
    this.publicFormService.submit(this.detail.token ?? '', payload).subscribe({
      next: (result) => {
        this.isSubmitting = false;
        this.submitResult = result;
        this.clearCountdown();
        this.clearQuizSession();
        if (this.isSurvey) {
          this.toast.success('Survey berhasil dikirim.');
        }
      },
      error: (err) => {
        this.isSubmitting = false;
        const message = typeof err?.error === 'string' && err.error ? err.error : 'Gagal mengirim jawaban.';
        this.toast.error(message);
      },
    });
  }

  // refillSurvey reset jawaban supaya link survey yang sama bisa dipakai isi ulang lagi.
  refillSurvey(): void {
    if (!this.detail) return;
    this.submitResult = null;
    this.autoSubmitTriggered = false;
    this.buildAnswerControls(this.detail.questions);
  }

  // isSelected dipakai template radio supaya pilihan tersinkron dengan FormControl dynamic.
  isSelected(index: number, optionId: number): boolean {
    return Number(this.answersArray.at(index).get('question_answer_id')?.value) === optionId;
  }

  // chooseOption set nilai radio manual agar tetap konsisten walau option dirender custom card.
  chooseOption(index: number, optionId: number): void {
    this.answersArray.at(index).get('question_answer_id')?.setValue(optionId);
    this.answersArray.at(index).get('question_answer_id')?.markAsTouched();
  }

  // formatPeriod tampilkan period publik yang lebih mudah dibaca.
  formatPeriod(): string {
    if (!this.detail?.start_time && !this.detail?.end_time) return '-';
    if (this.isQuiz) {
      const start = this.detail.start_time ? this.detail.start_time.slice(11, 16) : '?';
      const end = this.detail.end_time ? this.detail.end_time.slice(11, 16) : '?';
      return `${start} - ${end}`;
    }
    const fmt = (value: string | null) => (value ? this.formatDateTime(value) : '?');
    return `${fmt(this.detail?.start_time ?? null)} - ${fmt(this.detail?.end_time ?? null)}`;
  }

  // statusTitle kasih judul pendek untuk state non-aktif seperti expired atau belum mulai.
  statusTitle(): string {
    switch (this.detail?.state) {
      case 'upcoming':
        return 'Form Belum Dibuka';
      case 'expired':
        return 'Form Expired';
      case 'inactive':
        return 'Form Tidak Aktif';
      default:
        return 'Form Tidak Tersedia';
    }
  }

  // statusDescription kasih penjelasan beda antara quiz dan survey saat link tidak bisa diisi.
  statusDescription(): string {
    if (!this.detail) return '';
    const quizTimeOnly = this.isQuiz
      ? `${this.detail.start_time?.slice(11, 16) ?? '?'}`
      : this.formatDateTime(this.detail.start_time ?? null);

    if (this.detail.state === 'upcoming') {
      return this.isQuiz
        ? `Quiz ini baru bisa dimulai mulai pukul ${quizTimeOnly}.`
        : `Form ini baru bisa diisi mulai ${quizTimeOnly}.`;
    }
    if (this.detail.state === 'expired') {
      return this.isSurvey
        ? 'Periode survey sudah selesai, jadi link ini tidak menerima jawaban baru.'
        : 'Waktu quiz hari ini sudah habis, jadi link ini tidak bisa dipakai lagi.';
    }
    return 'Admin sedang menonaktifkan form ini untuk sementara.';
  }

  trackByQuestionId(_: number, question: PublicQuestion): number {
    return question.id;
  }

  trackByOptionId(_: number, option: { id: number }): number {
    return option.id;
  }

  // buildAnswerControls bikin 1 FormGroup per question agar validasi dan payload submit konsisten.
  private buildAnswerControls(questions: PublicQuestion[]): void {
    this.formSubscription.unsubscribe();
    this.formSubscription = new Subscription();
    this.answersArray.clear();
    this.questionForm.reset();

    questions.forEach((question) => {
      const isText = question.type_answer === 'free_text';
      this.answersArray.push(
        this.fb.group({
          question_id: [question.id],
          question_answer_id: [null, isText ? [] : [Validators.required]],
          answer_text: ['', isText ? [Validators.required] : []],
        }),
      );
    });

    this.formSubscription.add(
      this.accessForm.valueChanges.subscribe(() => {
        if (this.isQuiz && this.hasStartedQuiz) {
          this.persistQuizSession();
        }
      }),
    );
    this.formSubscription.add(
      this.questionForm.valueChanges.subscribe(() => {
        if (this.isQuiz && this.hasStartedQuiz) {
          this.persistQuizSession();
        }
      }),
    );
  }

  // buildSubmitPayload ubah FormArray dynamic menjadi payload API publik yang simpel.
  private buildSubmitPayload(): PublicFormSubmitPayload {
    return {
      email: this.isQuiz ? (this.accessForm.get('email')?.value?.trim() || null) : null,
      answers: this.answersArray.getRawValue().map((answer) => ({
        question_id: Number(answer['question_id']),
        question_answer_id: answer['question_answer_id'] ? Number(answer['question_answer_id']) : null,
        answer_text: answer['answer_text']?.trim() || null,
      })),
    };
  }

  // startCountdown jaga sisa waktu quiz tetap sinkron dan auto-submit begitu deadline tercapai.
  private startCountdown(): void {
    this.clearCountdown();
    this.autoSubmitTriggered = false;
    this.updateCountdown();
    this.countdownTimer = setInterval(() => {
      this.updateCountdown();
      if (
        this.countdownLabel === '00:00:00' &&
        this.hasStartedQuiz &&
        !this.submitResult &&
        !this.isSubmitting &&
        !this.autoSubmitTriggered
      ) {
        this.autoSubmitTriggered = true;
        this.submit(true);
      }
    }, 1000);
  }

  private clearCountdown(): void {
    if (this.countdownTimer) {
      clearInterval(this.countdownTimer);
      this.countdownTimer = null;
    }
  }

  // updateCountdown hitung selisih real-time ke end_time. Saat waktu habis, label dikunci ke nol.
  private updateCountdown(): void {
    if (!this.detail?.end_time) {
      this.countdownLabel = '--:--:--';
      return;
    }

    const endTime = this.resolveCountdownEndTime();
    const diffMs = endTime.getTime() - Date.now();
    if (Number.isNaN(endTime.getTime()) || diffMs <= 0) {
      this.countdownLabel = '00:00:00';
      return;
    }

    const totalSeconds = Math.floor(diffMs / 1000);
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = totalSeconds % 60;
    this.countdownLabel = [hours, minutes, seconds].map((value) => String(value).padStart(2, '0')).join(':');
  }

  // formatDateTime ubah string backend menjadi format UI seperti "8 Aug 2026 16:00".
  private formatDateTime(value: string | null): string {
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

  // resolveCountdownEndTime buat quiz, deadline dihitung dari jam hari ini; survey tetap pakai
  // datetime penuh dari backend karena period-nya memang berbasis tanggal+jam.
  private resolveCountdownEndTime(): Date {
    if (!this.detail?.end_time) {
      return new Date(Number.NaN);
    }

    if (!this.isQuiz) {
      return new Date(this.detail.end_time);
    }

    const source = new Date(this.detail.end_time);
    if (Number.isNaN(source.getTime())) {
      return new Date(Number.NaN);
    }

    const now = new Date();
    return new Date(
      now.getFullYear(),
      now.getMonth(),
      now.getDate(),
      source.getHours(),
      source.getMinutes(),
      source.getSeconds(),
      0,
    );
  }

  // quizSessionKey bikin key localStorage per-token supaya tiap link quiz punya sesi sendiri.
  private quizSessionKey(): string | null {
    const token = this.detail?.token ?? this.route.snapshot.paramMap.get('token');
    return token ? `public-quiz-session:${token}` : null;
  }

  // persistQuizSession simpan email, status started, dan jawaban sementara agar refresh tidak
  // melempar user balik ke layar start quiz.
  private persistQuizSession(): void {
    if (!this.isQuiz || !this.detail) return;
    const key = this.quizSessionKey();
    if (!key) return;

    const session: StoredQuizSession = {
      email: this.accessForm.get('email')?.value?.trim() || '',
      started: this.hasStartedQuiz,
      answers: this.answersArray.getRawValue().map((answer) => ({
        question_id: Number(answer['question_id']),
        question_answer_id: answer['question_answer_id'] ? Number(answer['question_answer_id']) : null,
        answer_text: answer['answer_text']?.trim() || null,
      })),
    };

    localStorage.setItem(key, JSON.stringify(session));
  }

  // restoreQuizSession balikin state quiz yang sempat mulai sebelumnya, termasuk email dan jawaban.
  private restoreQuizSession(): void {
    if (!this.isQuiz || !this.detail || this.detail.state !== 'active') return;
    const key = this.quizSessionKey();
    if (!key) return;

    const raw = localStorage.getItem(key);
    if (!raw) return;

    try {
      const session = JSON.parse(raw) as StoredQuizSession;
      if (session.email) {
        this.accessForm.patchValue({ email: session.email }, { emitEvent: false });
      }
      if (Array.isArray(session.answers)) {
        const answersByQuestionId = new Map(
          session.answers.map((answer) => [Number(answer.question_id), answer]),
        );

        this.answersArray.controls.forEach((control) => {
          const questionId = Number(control.get('question_id')?.value);
          const savedAnswer = answersByQuestionId.get(questionId);
          if (!savedAnswer) return;

          control.patchValue(
            {
              question_answer_id: savedAnswer.question_answer_id,
              answer_text: savedAnswer.answer_text ?? '',
            },
            { emitEvent: false },
          );
        });
      }

      if (session.started) {
        this.hasStartedQuiz = true;
        this.startCountdown();
      }
    } catch {
      this.clearQuizSession();
    }
  }

  // clearQuizSession hapus sesi quiz lokal setelah submit sukses atau saat link sudah tidak aktif.
  private clearQuizSession(): void {
    const key = this.quizSessionKey();
    if (key) {
      localStorage.removeItem(key);
    }
  }
}
