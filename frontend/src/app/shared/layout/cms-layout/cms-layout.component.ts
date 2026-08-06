import { CommonModule } from '@angular/common';
import { Component, OnInit } from '@angular/core';
import { RouterOutlet } from '@angular/router';

import { AuthService, Me } from '../../../core/auth/auth.service';
import { SidebarComponent } from '../sidebar/sidebar.component';

@Component({
  selector: 'app-cms-layout',
  standalone: true,
  imports: [CommonModule, RouterOutlet, SidebarComponent],
  templateUrl: './cms-layout.component.html',
  styleUrl: './cms-layout.component.scss',
})
export class CmsLayoutComponent implements OnInit {
  user: Me | null = null;

  constructor(private auth: AuthService) {}

  // Ambil data user yang lagi login buat ditampilin di layout (sidebar/header), dipanggil sekali pas layout dibuka.
  ngOnInit(): void {
    this.auth.me().subscribe({
      next: (me) => (this.user = me),
    });
  }
}
