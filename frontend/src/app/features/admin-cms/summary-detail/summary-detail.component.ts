import { CommonModule } from '@angular/common';
import { Component } from '@angular/core';
import { Router } from '@angular/router';

import { ButtonComponent } from '../../../shared/ui/button/button.component';

@Component({
  selector: 'app-summary-detail',
  standalone: true,
  imports: [CommonModule, ButtonComponent],
  templateUrl: './summary-detail.component.html',
  styleUrl: './summary-detail.component.scss',
})
export class SummaryDetailComponent {
  constructor(private router: Router) {}

  // Sementara detail summary baru menyiapkan shell halaman, jadi tombol ini balikin user ke daftar summary.
  backToList(): void {
    this.router.navigate(['/admin-cms/summary']);
  }
}
