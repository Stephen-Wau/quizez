import { Component, HostListener, OnDestroy, OnInit } from '@angular/core';
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

interface StoredPublicSession {
  email: string;
  started: boolean;
  started_at: string | null;
  access_code: string | null;
  access_granted: boolean;
  welcome_seen: boolean;
  current_question_index: number;
  review_mode: boolean;
  question_order: number[];
  answer_orders: Array<{
    question_id: number;
    option_ids: number[];
  }>;
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
  accessVerified = false;
  welcomeAcknowledged = false;
  isReviewStep = false;
  currentQuestionIndex = 0;
  progressRestored = false;
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
      access_code: [''],
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
    const stored = this.readStoredSession();
    this.loadForm(token, stored?.access_code ?? null);
  }

  ngOnDestroy(): void {
    this.clearCountdown();
    this.formSubscription.unsubscribe();
  }

  @HostListener('window:beforeunload', ['$event'])
  handleBeforeUnload(event: BeforeUnloadEvent): void {
    if (!this.shouldProtectRefresh()) return;
    this.persistFormSession();
    event.preventDefault();
    event.returnValue = '';
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

  get quizPassed(): boolean | null {
    return this.submitResult?.passed ?? null;
  }

  get totalQuestions(): number {
    return this.detail?.questions.length ?? 0;
  }

  get activeQuestion(): PublicQuestion | null {
    return this.detail?.questions[this.currentQuestionIndex] ?? null;
  }

  get activeAnswerGroup(): QuestionFormGroup | null {
    return this.answersArray.at(this.currentQuestionIndex) ?? null;
  }

  get isLastQuestion(): boolean {
    return this.currentQuestionIndex >= Math.max(this.totalQuestions - 1, 0);
  }

  get currentQuestionNumber(): number {
    return this.currentQuestionIndex + 1;
  }

  get progressPercent(): number {
    if (this.totalQuestions <= 0) return 0;
    return Math.round((this.currentQuestionNumber / this.totalQuestions) * 100);
  }

  get answeredCount(): number {
    return this.answersArray.controls.filter((control) => this.isAnsweredGroup(control)).length;
  }

  get reviewItems(): Array<{
    question: PublicQuestion;
    index: number;
    answered: boolean;
    answerPreview: string;
  }> {
    if (!this.detail) return [];

    return this.detail.questions.map((question, index) => {
      const group = this.answersArray.at(index);
      return {
        question,
        index,
        answered: this.isAnsweredGroup(group),
        answerPreview: this.answerPreview(index, question),
      };
    });
  }

  // loadForm ambil detail publik, lalu kalau kode akses valid form akan disiapkan penuh beserta restore state.
  loadForm(token: string, accessCode: string | null = null): void {
    this.loading = true;
    this.loadError = '';
    this.publicFormService.getByToken(token, accessCode).subscribe({
      next: (detail) => {
        this.loading = false;
        this.submitResult = null;
        this.detail = this.applyStoredPresentation(detail);
        this.autoSubmitTriggered = false;
        this.accessForm.patchValue(
          { access_code: accessCode ?? this.accessForm.get('access_code')?.value ?? '' },
          { emitEvent: false },
        );

        this.accessVerified = !detail.access_code_required || detail.access_granted;
        if (!this.accessVerified) {
          this.resetRuntimeState();
          this.answersArray.clear();
          return;
        }

        this.buildAnswerControls(this.detail.questions);
        this.restoreFormSession();

        if (detail.type === 'quiz' && detail.state !== 'active') {
          this.clearFormSession();
        }
        if (detail.type === 'survey' && detail.state === 'active') {
          this.persistFormSession();
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

  submitAccessCode(): void {
    const token = this.route.snapshot.paramMap.get('token') ?? '';
    const accessCode = this.readAccessCode();
    if (!accessCode) {
      this.accessForm.get('access_code')?.markAsTouched();
      this.toast.error('PIN akses wajib diisi.');
      return;
    }
    this.loadForm(token, accessCode);
  }

  acknowledgeWelcome(): void {
    this.welcomeAcknowledged = true;
    this.persistFormSession();
  }

  // startQuiz ngecek email dulu, lalu menandai quiz benar-benar dimulai setelah user melihat instruksi.
  startQuiz(): void {
    if (!this.isQuiz || !this.detail) return;
    if (!this.welcomeAcknowledged) {
      this.toast.info('Baca instruksi dulu sebelum mulai quiz.');
      return;
    }
    if (this.accessForm.get('email')?.invalid) {
      this.accessForm.get('email')?.markAsTouched();
      return;
    }
    if (!this.accessForm.get('started_at')) {
      this.accessForm.addControl('started_at', this.fb.control(new Date().toISOString()));
    } else if (!this.readStartedAt()) {
      this.accessForm.get('started_at')?.setValue(new Date().toISOString());
    }
    this.hasStartedQuiz = true;
    this.currentQuestionIndex = 0;
    this.isReviewStep = false;
    this.persistFormSession();
    this.startCountdown();
  }

  nextQuestion(): void {
    if (!this.validateCurrentQuestion()) return;
    this.currentQuestionIndex = Math.min(this.currentQuestionIndex + 1, Math.max(this.totalQuestions - 1, 0));
    this.persistFormSession();
  }

  previousQuestion(): void {
    this.currentQuestionIndex = Math.max(this.currentQuestionIndex - 1, 0);
    this.persistFormSession();
  }

  jumpToQuestion(index: number): void {
    this.isReviewStep = false;
    this.currentQuestionIndex = index;
    this.persistFormSession();
  }

  openReview(): void {
    if (!this.validateCurrentQuestion()) return;
    this.isReviewStep = true;
    this.persistFormSession();
  }

  backToQuestions(): void {
    this.isReviewStep = false;
    this.persistFormSession();
  }

  // submit dipakai tombol final submit atau auto-submit saat timer quiz habis.
  submit(autoSubmit = false): void {
    if (!this.detail) return;
    if (!autoSubmit && !this.canSubmitAllAnswers()) {
      this.toast.error('Masih ada jawaban yang belum lengkap.');
      return;
    }

    const payload = this.buildSubmitPayload();
    this.isSubmitting = true;
    this.publicFormService.submit(this.detail.token ?? '', payload).subscribe({
      next: (result) => {
        this.isSubmitting = false;
        this.submitResult = result;
        this.clearCountdown();
        this.clearFormSession();
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

  refillSurvey(): void {
    if (!this.detail) return;
    this.submitResult = null;
    this.autoSubmitTriggered = false;
    this.currentQuestionIndex = 0;
    this.isReviewStep = false;
    this.progressRestored = false;
    this.buildAnswerControls(this.detail.questions);
    this.clearFormSession();
    this.persistFormSession();
  }

  isSelected(index: number, optionId: number): boolean {
    return Number(this.answersArray.at(index).get('question_answer_id')?.value) === optionId;
  }

  chooseOption(index: number, optionId: number): void {
    this.answersArray.at(index).get('question_answer_id')?.setValue(optionId);
    this.answersArray.at(index).get('question_answer_id')?.markAsTouched();
    this.persistFormSession();
  }

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

  answerPreview(index: number, question: PublicQuestion): string {
    const group = this.answersArray.at(index);
    if (!group) return 'Belum dijawab';
    if (question.type_answer === 'free_text') {
      return group.get('answer_text')?.value?.trim() || 'Belum dijawab';
    }

    const optionId = Number(group.get('question_answer_id')?.value ?? 0);
    const selected = question.answers.find((option) => option.id === optionId);
    return selected?.label || 'Belum dijawab';
  }

  resultAnswerPreview(answer: PublicFormSubmitResult['answer_details'][number]): string {
    if (answer.selected_answer_text) return answer.selected_answer_text;
    if (answer.selected_answer_label) return answer.selected_answer_label;
    return 'Tidak dijawab';
  }

  resultCorrectAnswer(answer: PublicFormSubmitResult['answer_details'][number]): string {
    return answer.correct_answers.length > 0 ? answer.correct_answers.join(', ') : '-';
  }

  formatScorePercent(value: number | null): string {
    if (value === null || value === undefined) return '0';
    return value.toFixed(2).replace(/\.00$/, '');
  }

  trackByQuestionId(_: number, question: PublicQuestion): number {
    return question.id;
  }

  trackByOptionId(_: number, option: { id: number }): number {
    return option.id;
  }

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
        if (this.accessVerified) {
          this.persistFormSession();
        }
      }),
    );
    this.formSubscription.add(
      this.questionForm.valueChanges.subscribe(() => {
        if (this.accessVerified) {
          this.persistFormSession();
        }
      }),
    );
  }

  private buildSubmitPayload(): PublicFormSubmitPayload {
    return {
      email: this.isQuiz ? (this.accessForm.get('email')?.value?.trim() || null) : null,
      started_at: this.isQuiz ? this.readStartedAt() : null,
      access_code: this.readAccessCode(),
      answers: this.answersArray.getRawValue().map((answer) => ({
        question_id: Number(answer['question_id']),
        question_answer_id: answer['question_answer_id'] ? Number(answer['question_answer_id']) : null,
        answer_text: answer['answer_text']?.trim() || null,
      })),
    };
  }

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

  private sessionKey(): string | null {
    const token = this.detail?.token ?? this.route.snapshot.paramMap.get('token');
    return token ? `public-form-session:${token}` : null;
  }

  private persistFormSession(): void {
    if (!this.detail || !this.accessVerified || !this.canFillNow || this.submitResult) return;
    const key = this.sessionKey();
    if (!key) return;

    const session: StoredPublicSession = {
      email: this.accessForm.get('email')?.value?.trim() || '',
      started: this.hasStartedQuiz,
      started_at: this.readStartedAt(),
      access_code: this.readAccessCode(),
      access_granted: this.accessVerified,
      welcome_seen: this.welcomeAcknowledged,
      current_question_index: this.currentQuestionIndex,
      review_mode: this.isReviewStep,
      question_order: (this.detail.questions ?? []).map((question) => question.id),
      answer_orders: (this.detail.questions ?? []).map((question) => ({
        question_id: question.id,
        option_ids: question.answers.map((option) => option.id),
      })),
      answers: this.answersArray.getRawValue().map((answer) => ({
        question_id: Number(answer['question_id']),
        question_answer_id: answer['question_answer_id'] ? Number(answer['question_answer_id']) : null,
        answer_text: answer['answer_text']?.trim() || null,
      })),
    };

    localStorage.setItem(key, JSON.stringify(session));
  }

  private restoreFormSession(): void {
    if (!this.detail || !this.canFillNow || !this.accessVerified) return;
    const session = this.readStoredSession();
    if (!session) {
      this.persistFormSession();
      return;
    }

    this.progressRestored = false;
    if (session.email) {
      this.accessForm.patchValue({ email: session.email }, { emitEvent: false });
    }
    if (session.access_code) {
      this.accessForm.patchValue({ access_code: session.access_code }, { emitEvent: false });
    }
    if (session.started_at) {
      if (!this.accessForm.get('started_at')) {
        this.accessForm.addControl('started_at', this.fb.control(session.started_at));
      } else {
        this.accessForm.get('started_at')?.setValue(session.started_at, { emitEvent: false });
      }
    }
    this.welcomeAcknowledged = !!session.welcome_seen;

    if (Array.isArray(session.answers)) {
      const answersByQuestionId = new Map(session.answers.map((answer) => [Number(answer.question_id), answer]));
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
      this.progressRestored = session.answers.some(
        (answer) => answer.question_answer_id !== null || !!answer.answer_text?.trim(),
      );
    }

    this.currentQuestionIndex = Math.min(
      Math.max(session.current_question_index ?? 0, 0),
      Math.max(this.totalQuestions - 1, 0),
    );
    this.isReviewStep = !!session.review_mode;

    if (session.started && this.isQuiz) {
      this.hasStartedQuiz = true;
      this.startCountdown();
    }

    this.persistFormSession();
  }

  private readStoredSession(): StoredPublicSession | null {
    const key = this.sessionKey();
    if (!key) return null;

    const raw = localStorage.getItem(key);
    if (!raw) return null;

    try {
      return JSON.parse(raw) as StoredPublicSession;
    } catch {
      localStorage.removeItem(key);
      return null;
    }
  }

  private clearFormSession(): void {
    const key = this.sessionKey();
    if (key) {
      localStorage.removeItem(key);
    }
  }

  private applyStoredPresentation(detail: PublicFormDetail): PublicFormDetail {
    const session = this.readStoredSession();
    if (!session || !Array.isArray(session.question_order) || session.question_order.length === 0) {
      return detail;
    }

    const questionMap = new Map(detail.questions.map((question) => [question.id, question]));
    const answerOrderMap = new Map(session.answer_orders.map((item) => [item.question_id, item.option_ids]));
    const orderedQuestions: PublicQuestion[] = [];

    for (const questionID of session.question_order) {
      const question = questionMap.get(questionID);
      if (!question) continue;
      questionMap.delete(questionID);
      orderedQuestions.push(this.applyStoredAnswerOrder(question, answerOrderMap.get(questionID) ?? []));
    }

    for (const question of questionMap.values()) {
      orderedQuestions.push(this.applyStoredAnswerOrder(question, answerOrderMap.get(question.id) ?? []));
    }

    return {
      ...detail,
      questions: orderedQuestions,
    };
  }

  private applyStoredAnswerOrder(question: PublicQuestion, optionOrder: number[]): PublicQuestion {
    if (!Array.isArray(optionOrder) || optionOrder.length === 0) {
      return question;
    }

    const optionMap = new Map(question.answers.map((option) => [option.id, option]));
    const orderedAnswers = [];
    for (const optionID of optionOrder) {
      const option = optionMap.get(optionID);
      if (!option) continue;
      optionMap.delete(optionID);
      orderedAnswers.push(option);
    }
    for (const option of optionMap.values()) {
      orderedAnswers.push(option);
    }

    return {
      ...question,
      answers: orderedAnswers,
    };
  }

  private validateCurrentQuestion(): boolean {
    const control = this.activeAnswerGroup;
    if (!control) return true;
    if (control.valid) return true;
    control.markAllAsTouched();
    this.toast.error('Jawaban untuk question ini wajib diisi sebelum lanjut.');
    return false;
  }

  private canSubmitAllAnswers(): boolean {
    this.questionForm.markAllAsTouched();
    return this.questionForm.valid;
  }

  private isAnsweredGroup(control: QuestionFormGroup | null): boolean {
    if (!control) return false;
    const answerID = control.get('question_answer_id')?.value;
    const answerText = control.get('answer_text')?.value?.trim();
    return !!answerID || !!answerText;
  }

  private shouldProtectRefresh(): boolean {
    return !!this.detail && this.canFillNow && !this.submitResult && this.accessVerified && (this.isSurvey || this.hasStartedQuiz);
  }

  private readStartedAt(): string | null {
    return this.accessForm.get('started_at')?.value || null;
  }

  private readAccessCode(): string | null {
    return this.accessForm.get('access_code')?.value?.trim() || null;
  }

  private resetRuntimeState(): void {
    this.clearCountdown();
    this.hasStartedQuiz = false;
    this.welcomeAcknowledged = false;
    this.isReviewStep = false;
    this.currentQuestionIndex = 0;
    this.progressRestored = false;
  }
}
