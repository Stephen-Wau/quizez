import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AUTO_DISMISS_MS, ToastService } from './toast.service';

// Dipasang sekali di root AppComponent, render semua toast aktif dari ToastService.
@Component({
  selector: 'app-toast-container',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './toast-container.component.html',
  styleUrl: './toast-container.component.scss',
})
export class ToastContainerComponent {
  // Dipakai template buat sinkronin durasi animasi border countdown ke waktu auto-dismiss beneran.
  dismissMs = AUTO_DISMISS_MS;

  // Inject ToastService as public biar template bisa langsung baca toastService.toasts() buat di-loop.
  constructor(public toastService: ToastService) {}
}
