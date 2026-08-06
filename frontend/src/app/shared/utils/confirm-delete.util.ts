import { Observable } from 'rxjs';
import { ToastService } from '../ui/toast/toast.service';

// Pola hapus standar semua menu CRUD CMS: konfirmasi native browser dulu (cukup buat aksi
// destruktif sederhana, gak perlu component confirm dialog terpisah), baru panggil API delete,
// lalu toast + callback sukses (biasanya reload list) atau toast error kalau gagal.
export function confirmAndDelete(
  confirmMessage: string,
  deleteRequest: () => Observable<unknown>,
  toast: ToastService,
  successMessage: string,
  errorMessage: string,
  onSuccess: () => void,
): void {
  if (!window.confirm(confirmMessage)) return;

  deleteRequest().subscribe({
    next: () => {
      toast.success(successMessage);
      onSuccess();
    },
    error: () => toast.error(errorMessage),
  });
}
