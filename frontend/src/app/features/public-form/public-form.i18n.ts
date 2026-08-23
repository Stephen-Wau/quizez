export type PublicFormLanguage = 'id' | 'en';

// Kamus teks UI statis form publik (label/tombol/badge/instruksi) -- soal & jawaban TETAP bahasa
// yang ditulis admin (gak ditranslate otomatis), cuma "chrome" form-nya yang ikut quiz.language.
// {token} di dalam string di-replace lewat PublicFormComponent.t(key, params).
export const PUBLIC_FORM_I18N: Record<string, Record<PublicFormLanguage, string>> = {
  loading_title: { id: 'Memuat Form', en: 'Loading Form' },
  loading_body: {
    id: 'Mohon tunggu, kami sedang menyiapkan quiz atau survey-nya.',
    en: 'Please wait, we are preparing the quiz or survey.',
  },
  link_unavailable_title: { id: 'Link Tidak Tersedia', en: 'Link Unavailable' },

  quiz_done_eyebrow: { id: 'Quiz Selesai', en: 'Quiz Completed' },
  quiz_fallback: { id: 'Quiz', en: 'Quiz' },
  survey_fallback: { id: 'Survey', en: 'Survey' },
  passed: { id: 'Lulus', en: 'Passed' },
  not_passed: { id: 'Tidak Lulus', en: 'Not Passed' },
  completed: { id: 'Selesai', en: 'Completed' },
  score_label: { id: 'Skor', en: 'Score' },
  passing_grade_label: { id: 'Passing Grade', en: 'Passing Grade' },
  correct_label: { id: 'Benar', en: 'Correct' },
  answered_label: { id: 'Terjawab', en: 'Answered' },
  badge_prefix: { id: 'Badge', en: 'Badge' },
  download_certificate: { id: 'Download Sertifikat', en: 'Download Certificate' },
  submitted_at_prefix: { id: 'Terkirim pada', en: 'Submitted at' },
  question_fallback: { id: 'Pertanyaan', en: 'Question' },
  your_answer_prefix: { id: 'Jawaban kamu', en: 'Your answer' },
  correct_answer_prefix: { id: 'Jawaban benar', en: 'Correct answer' },
  correct: { id: 'Benar', en: 'Correct' },
  incorrect: { id: 'Salah', en: 'Incorrect' },
  not_answered: { id: 'Belum dijawab', en: 'Not answered yet' },
  not_answered_result: { id: 'Tidak dijawab', en: 'Not answered' },

  thanks_eyebrow: { id: 'Terima Kasih', en: 'Thank You' },
  refill_survey: { id: 'Isi Kembali Survey', en: 'Refill Survey' },

  quiz_public: { id: 'Quiz Publik', en: 'Public Quiz' },
  survey_public: { id: 'Survey Publik', en: 'Public Survey' },
  access_with_pin: { id: 'Akses Dengan PIN', en: 'Access With PIN' },
  access_message_default: {
    id: 'Masukkan PIN unik untuk membuka form ini.',
    en: 'Enter the unique PIN to open this form.',
  },
  access_code_label: { id: 'Access Code / PIN', en: 'Access Code / PIN' },
  pin_required: { id: 'PIN wajib diisi.', en: 'PIN is required.' },
  open_form: { id: 'Buka Form', en: 'Open Form' },
  form_public_fallback: { id: 'Form Publik', en: 'Public Form' },

  period_label: { id: 'Periode', en: 'Period' },
  total_question_label: { id: 'Total Soal', en: 'Total Questions' },
  max_point_label: { id: 'Max Point', en: 'Max Point' },
  remaining_time_label: { id: 'Sisa Waktu', en: 'Remaining Time' },
  progress_label: { id: 'Progress', en: 'Progress' },
  progress_restored: { id: 'Dipulihkan dari sesi terakhir', en: 'Restored from last session' },
  lock_mode_label: { id: 'Lock Mode', en: 'Lock Mode' },
  lock_mode_active: { id: 'Aktif — wajib fullscreen', en: 'Active — fullscreen required' },
  violation_label: { id: 'Pelanggaran', en: 'Violation' },
  max_attempts_label: { id: 'Boleh Diulang', en: 'Retakes Allowed' },

  lock_mode_active_eyebrow: { id: 'Lock Mode Aktif', en: 'Lock Mode Active' },
  enter_fullscreen_title: { id: 'Masuk Mode Fullscreen', en: 'Enter Fullscreen Mode' },
  enter_fullscreen_desc: {
    id: 'Quiz ini pakai anti-cheat: kamu wajib mengerjakan dalam mode fullscreen. Keluar tab atau fullscreen dihitung pelanggaran.',
    en: 'This quiz uses anti-cheat: you must work in fullscreen mode. Leaving the tab or fullscreen counts as a violation.',
  },
  enter_fullscreen_button: { id: 'Masuk Fullscreen & Lanjutkan', en: 'Enter Fullscreen & Continue' },

  welcome_title: { id: 'Welcome Page & Instruction', en: 'Welcome Page & Instruction' },
  welcome_desc: {
    id: 'Baca aturan quiz dulu, lalu mulai dengan email yang valid.',
    en: 'Read the quiz rules first, then start with a valid email.',
  },
  ready_to_start: { id: 'Siap Dimulai', en: 'Ready to start' },
  quiz_duration_label: { id: 'Durasi Quiz', en: 'Quiz Duration' },
  question_count_label: { id: 'Jumlah Soal', en: 'Number of Questions' },
  rule_random_order: {
    id: 'Urutan soal dan pilihan jawaban dapat berbeda untuk setiap peserta.',
    en: 'Question and answer order may differ for each participant.',
  },
  rule_autosave: {
    id: 'Jawaban akan tersimpan di browser ini selama form belum dikirim.',
    en: 'Your answers are saved in this browser until the form is submitted.',
  },
  rule_restore: {
    id: 'Jika browser direfresh, quiz akan dipulihkan dari sesi terakhir selama waktunya masih aktif.',
    en: 'If the browser is refreshed, the quiz will be restored from your last session while the time is still active.',
  },
  rule_dedup: {
    id: 'Kamu bisa mengirim quiz ini maksimal {max} kali percobaan.',
    en: 'You can submit this quiz a maximum of {max} time(s).',
  },
  rule_review: {
    id: 'Sebelum submit final, kamu bisa review semua jawaban lebih dulu.',
    en: 'Before the final submit, you can review all your answers first.',
  },
  ack_instructions: {
    id: 'Saya sudah membaca dan memahami instruksi di atas.',
    en: 'I have read and understood the instructions above.',
  },
  name_label: { id: 'Nama', en: 'Name' },
  name_required: { id: 'Nama wajib diisi.', en: 'Name is required.' },
  email_label: { id: 'Email', en: 'Email' },
  email_required: { id: 'Email wajib valid.', en: 'A valid email is required.' },
  start_quiz: { id: 'Mulai Quiz', en: 'Start Quiz' },

  question_of: { id: 'Soal {current} dari {total}', en: 'Question {current} of {total}' },
  percent_done: { id: '{percent}% selesai', en: '{percent}% done' },
  answered_count: { id: '{count} terjawab', en: '{count} answered' },
  answer_label: { id: 'Jawaban', en: 'Answer' },
  answer_required: { id: 'Jawaban untuk soal ini wajib diisi.', en: 'An answer for this question is required.' },
  choose_answer_placeholder: { id: 'Pilih jawaban...', en: 'Choose an answer...' },
  previous: { id: 'Sebelumnya', en: 'Previous' },
  next_question: { id: 'Soal Berikutnya', en: 'Next Question' },
  review_before_submit: { id: 'Review Sebelum Submit', en: 'Review Before Submit' },
  review_desc: {
    id: 'Cek ulang semua jawaban sebelum dikirim final.',
    en: 'Double-check all your answers before the final submit.',
  },
  edit_answer: { id: 'Edit Jawaban', en: 'Edit Answer' },
  back_to_questions: { id: 'Kembali ke Soal', en: 'Back to Questions' },
  submitting: { id: 'Mengirim...', en: 'Submitting...' },
  submit_quiz_final: { id: 'Submit Quiz Final', en: 'Submit Final Quiz' },
  submit_survey_final: { id: 'Submit Survey Final', en: 'Submit Final Survey' },
  answered_badge: { id: 'Terjawab', en: 'Answered' },
  missing_badge: { id: 'Belum Lengkap', en: 'Missing' },

  status_upcoming_title: { id: 'Form Belum Dibuka', en: 'Form Not Open Yet' },
  status_expired_title: { id: 'Form Expired', en: 'Form Expired' },
  status_inactive_title: { id: 'Form Tidak Aktif', en: 'Form Inactive' },
  status_unavailable_title: { id: 'Form Tidak Tersedia', en: 'Form Unavailable' },
  status_upcoming_quiz: {
    id: 'Quiz ini baru bisa dimulai mulai pukul {time}.',
    en: 'This quiz can only be started from {time}.',
  },
  status_upcoming_other: {
    id: 'Form ini baru bisa diisi mulai {time}.',
    en: 'This form can only be filled starting {time}.',
  },
  status_expired_survey: {
    id: 'Periode survey sudah selesai, jadi link ini tidak menerima jawaban baru.',
    en: 'The survey period has ended, so this link no longer accepts new answers.',
  },
  status_expired_quiz: {
    id: 'Waktu quiz hari ini sudah habis, jadi link ini tidak bisa dipakai lagi.',
    en: "Today's quiz time is over, so this link can no longer be used.",
  },
  status_inactive_body: {
    id: 'Admin sedang menonaktifkan form ini untuk sementara.',
    en: 'The admin has temporarily deactivated this form.',
  },

  toast_pin_required: { id: 'PIN akses wajib diisi.', en: 'Access PIN is required.' },
  toast_read_instructions: {
    id: 'Baca instruksi dulu sebelum mulai quiz.',
    en: 'Please read the instructions before starting the quiz.',
  },
  toast_fullscreen_unsupported: {
    id: 'Browser ini tidak mendukung mode fullscreen.',
    en: 'This browser does not support fullscreen mode.',
  },
  toast_fullscreen_failed: {
    id: 'Gagal masuk mode fullscreen, coba lagi.',
    en: 'Failed to enter fullscreen mode, please try again.',
  },
  violation_tab_switch: {
    id: 'Kamu terdeteksi berpindah tab atau aplikasi lain.',
    en: 'You were detected switching tabs or apps.',
  },
  violation_exit_fullscreen: {
    id: 'Kamu keluar dari mode fullscreen.',
    en: 'You exited fullscreen mode.',
  },
  toast_violation_final: {
    id: 'Pelanggaran ke-{count}: {reason} Quiz otomatis dikirim.',
    en: 'Violation #{count}: {reason} The quiz was submitted automatically.',
  },
  toast_violation_warning: {
    id: 'Peringatan {count}/{max}: {reason} Kembali ke fullscreen sekarang.',
    en: 'Warning {count}/{max}: {reason} Return to fullscreen now.',
  },
  toast_incomplete_answers: {
    id: 'Masih ada jawaban yang belum lengkap.',
    en: 'Some answers are still incomplete.',
  },
  toast_submit_failed: { id: 'Gagal mengirim jawaban.', en: 'Failed to submit your answers.' },
  toast_survey_submitted: { id: 'Survey berhasil dikirim.', en: 'Survey submitted successfully.' },
  toast_answer_required_next: {
    id: 'Jawaban untuk soal ini wajib diisi sebelum lanjut.',
    en: 'An answer for this question is required before continuing.',
  },
};
