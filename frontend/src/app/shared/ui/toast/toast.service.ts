import { Injectable, signal } from '@angular/core';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface Toast {
  id: number;
  type: ToastType;
  message: string;
}

let nextId = 0;
// Diexport supaya ToastContainerComponent bisa sinkronin animasi border countdown ke durasi yang sama persis.
export const AUTO_DISMISS_MS = 4000;

// Store toast global, dipanggil dari komponen manapun lewat inject(ToastService) buat nampilin notifikasi pop-up.
@Injectable({ providedIn: 'root' })
export class ToastService {
  toasts = signal<Toast[]>([]);

  // Toast hijau, dipakai buat konfirmasi aksi berhasil (ex: simpan/hapus sukses).
  success(message: string): void {
    this.show('success', message);
  }

  // Toast merah, dipakai buat error dari API atau validasi gagal.
  error(message: string): void {
    this.show('error', message);
  }

  // Toast kuning, dipakai buat peringatan non-fatal (ex: sesi mau habis).
  warning(message: string): void {
    this.show('warning', message);
  }

  // Toast biru netral, dipakai buat notifikasi informasi umum.
  info(message: string): void {
    this.show('info', message);
  }

  // Hapus satu toast dari daftar aktif, dipanggil otomatis (auto-dismiss) atau manual (tombol close).
  dismiss(id: number): void {
    this.toasts.update((list) => list.filter((t) => t.id !== id));
  }

  // Tambah toast baru ke daftar aktif & jadwalkan auto-dismiss setelah AUTO_DISMISS_MS.
  private show(type: ToastType, message: string): void {
    const id = nextId++;
    this.toasts.update((list) => [...list, { id, type, message }]);
    setTimeout(() => this.dismiss(id), AUTO_DISMISS_MS);
  }
}
