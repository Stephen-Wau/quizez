import { Component } from '@angular/core';
import { RouterLink } from '@angular/router';
import { LucideAngularModule } from 'lucide-angular';
import { ModalComponent } from '../../shared/ui/modal/modal.component';

type Accent = 'indigo' | 'emerald' | 'amber' | 'rose' | 'sky' | 'violet' | 'teal' | 'pink';

interface FeatureCard {
  icon: string;
  accent: Accent;
  title: string;
  desc: string;
  // Poin struktur/langkah fitur ini, ditampilin pas card di-klik (buka modal detail).
  steps: string[];
}

interface StepCard {
  number: string;
  title: string;
  desc: string;
}

interface StatItem {
  icon: string;
  value: string;
  label: string;
}

// Halaman profil aplikasi (landing page publik di "/"), murni presentational tanpa data dari backend.
@Component({
  selector: 'app-landing',
  standalone: true,
  imports: [RouterLink, LucideAngularModule, ModalComponent],
  templateUrl: './landing.component.html',
  styleUrl: './landing.component.scss',
})
export class LandingComponent {
  readonly currentYear = new Date().getFullYear();

  // Fitur yang lagi dibuka detailnya di modal, null = modal tertutup.
  activeFeature: FeatureCard | null = null;

  // Highlight angka singkat buat social proof di hero.
  readonly stats: StatItem[] = [
    { icon: 'clipboard-list', value: '6', label: 'Tipe Soal' },
    { icon: 'file-down', value: '3', label: 'Format Export' },
    { icon: 'shield-check', value: '0', label: 'Login Buat Responden' },
    { icon: 'trophy', value: '3', label: 'Tier Badge' },
  ];

  // Daftar fitur utama yang ditampilkan di grid, disesuaikan dengan progress fitur di CLAUDE.md.
  readonly features: FeatureCard[] = [
    {
      icon: 'clipboard-list',
      accent: 'indigo',
      title: 'Quiz & Survey Builder',
      desc: 'Bikin quiz atau survey dengan 6 tipe soal, random subset soal, dan bahasa form publik per-quiz.',
      steps: [
        'Pilih jenis form: Quiz (ada skor) atau Survey (tanpa skor).',
        'Susun soal dari 6 tipe: pilihan ganda, rating, isian bebas, dropdown, checkbox, matrix.',
        'Atur random subset soal biar tiap responden dapet soal acak yang stabil per sesi.',
        'Tentuin retake policy dan bahasa teks UI form publik (Indonesia/English).',
      ],
    },
    {
      icon: 'book-open',
      accent: 'emerald',
      title: 'Bank Soal',
      desc: 'Simpan soal reusable, import massal via CSV/XLSX, tinggal pakai ulang ke quiz mana pun.',
      steps: [
        'Simpan soal reusable terpisah dari quiz manapun, lengkap sama tag freeform.',
        'Import massal dari file CSV/XLSX, ada template contoh & validasi per baris.',
        'Cari soal lama lewat tag biar gampang dipakai ulang.',
        'Reuse ke quiz = copy independen, ubah di quiz gak bakal ganggu soal asli di bank.',
      ],
    },
    {
      icon: 'list-checks',
      accent: 'amber',
      title: 'Scoring & Retake Policy',
      desc: 'Skor dihitung otomatis, atur maksimal percobaan ulang dan skor mana yang dipakai.',
      steps: [
        'Jawaban responden langsung dicocokkan & dinilai otomatis pas submit.',
        'Atur max_attempts: default 1x, bisa dinaikkan biar boleh diulang beberapa kali.',
        'Pilih kebijakan skor: pakai yang terbaik (best) atau yang terakhir (latest).',
        'Skor final ini yang dipakai buat leaderboard dan sertifikat kelulusan.',
      ],
    },
    {
      icon: 'lock',
      accent: 'rose',
      title: 'Anti-Cheat Lock Mode',
      desc: 'Wajib fullscreen, deteksi tab-switch, auto-submit setelah 3x pelanggaran.',
      steps: [
        'Responden wajib masuk mode fullscreen sebelum mulai ngerjain quiz.',
        'Sistem otomatis mendeteksi kalau keluar fullscreen atau pindah tab browser.',
        'Setelah 3x pelanggaran terdeteksi, jawaban otomatis ke-submit paksa.',
        'Submission di-dedup per device_fingerprint biar 1 device gak bisa isi dobel.',
      ],
    },
    {
      icon: 'bar-chart-3',
      accent: 'sky',
      title: 'Analytics & Reporting',
      desc: 'Export CSV/Excel/PDF, trend chart, top incorrect questions, sentimen jawaban bebas.',
      steps: [
        'Filter data hasil quiz/survey berdasarkan periode, respondent, atau rentang skor.',
        'Lihat trend chart dan daftar soal yang paling banyak dijawab salah.',
        'Analisis ringkasan sentimen otomatis buat jawaban tipe isian bebas.',
        'Export laporan lengkap kapan aja dalam format CSV, Excel, atau PDF.',
      ],
    },
    {
      icon: 'trophy',
      accent: 'violet',
      title: 'Gamifikasi',
      desc: 'Leaderboard per-quiz, badge tier gold/silver/bronze, sertifikat PDF otomatis.',
      steps: [
        'Responden quiz wajib isi nama dulu sebelum mulai ngerjain.',
        'Badge tier gold/silver/bronze otomatis kekasih dari persentase skor akhir.',
        'Sertifikat PDF bisa didownload responden (khusus quiz yang punya poin).',
        'Leaderboard per-quiz ranking berdasar skor, tie-break dari durasi pengerjaan tercepat.',
      ],
    },
    {
      icon: 'users',
      accent: 'teal',
      title: 'Kolaborasi Multi-Admin',
      desc: 'Role super admin & editor, audit log aktivitas penting, visibility menu berbasis role.',
      steps: [
        'Akun admin dibagi 2 role: super admin dan editor.',
        'Menu admin management cuma keliatan & bisa diakses super admin.',
        'Setiap aktivitas admin yang penting otomatis kecatat di audit log.',
        'Tab dan menu CMS otomatis nyesuain sama role yang lagi login.',
      ],
    },
    {
      icon: 'share-2',
      accent: 'pink',
      title: 'Share Link + PIN',
      desc: 'Responden isi form publik tanpa login, cukup lewat link dan PIN akses.',
      steps: [
        'Tiap quiz/survey punya link + PIN akses unik yang di-generate otomatis.',
        'Responden isi lewat link itu langsung tanpa perlu bikin akun sama sekali.',
        'Urutan soal diacak per sesi biar antar responden gak gampang saling contek.',
        'Progress pengisian ke-save otomatis di browser kalau halaman ke-refresh.',
      ],
    },
  ];

  // Alur singkat pemakaian aplikasi buat ditampilin di section "Cara Kerja".
  readonly steps: StepCard[] = [
    { number: '01', title: 'Buat Quiz/Survey', desc: 'Susun soal, atur scoring, retake policy, dan bahasa form.' },
    { number: '02', title: 'Bagikan Link + PIN', desc: 'Kirim link akses ke responden, tanpa perlu bikin akun.' },
    { number: '03', title: 'Pantau Hasilnya', desc: 'Lihat leaderboard, analytics, dan export laporan kapan aja.' },
  ];

  // Peta warna aksen ke hex asli, dipakai buat header modal (var CSS gak bisa dibaca balik dari nama accent).
  private readonly accentHex: Record<Accent, string> = {
    indigo: '#4f46e5',
    emerald: '#059669',
    amber: '#d97706',
    rose: '#e11d48',
    sky: '#0284c7',
    violet: '#7c3aed',
    teal: '#0d9488',
    pink: '#db2777',
  };

  // Warna aksen hex fitur yang lagi aktif, dipakai buat header modal.
  get activeAccentHex(): string {
    return this.activeFeature ? this.accentHex[this.activeFeature.accent] : '';
  }

  // Buka modal detail struktur fitur pas card-nya diklik.
  openFeature(feature: FeatureCard): void {
    this.activeFeature = feature;
  }

  // Tutup modal detail fitur, dipanggil dari (closed) app-modal.
  closeFeature(): void {
    this.activeFeature = null;
  }
}
