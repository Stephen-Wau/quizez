import { CommonModule } from '@angular/common';
import { Component, OnInit } from '@angular/core';
import { NavigationEnd, Router, RouterOutlet } from '@angular/router';
import { LucideAngularModule } from 'lucide-angular';

import { AuthService, Me } from '../../../core/auth/auth.service';
import { ToastService } from '../../ui/toast/toast.service';
import { SidebarComponent } from '../sidebar/sidebar.component';

// Key localStorage buat inget preferensi collapse sidebar antar sesi (biar gak collapse ulang tiap reload).
const SIDEBAR_COLLAPSED_KEY = 'quizez_sidebar_collapsed';

// Layout parent semua halaman /admin-cms: validasi sesi sekali di sini, render sidebar + konten anak route.
@Component({
  selector: 'app-cms-layout',
  standalone: true,
  imports: [CommonModule, RouterOutlet, SidebarComponent, LucideAngularModule],
  templateUrl: './cms-layout.component.html',
  styleUrl: './cms-layout.component.scss',
})
export class CmsLayoutComponent implements OnInit {
  user: Me | null = null;
  // Cuma relevan di layar sempit (<=880px, lihat sidebar-nya jadi drawer) — di desktop sidebar
  // selalu keliatan terlepas dari state ini (CSS di sidebar.component.scss yang nentuin).
  isSidebarOpen = false;
  // Collapse/expand sidebar di desktop (icon-only vs full width) — beda konsep sama isSidebarOpen
  // yang buat drawer mobile. Persist ke localStorage biar preferensi user gak reset tiap reload.
  isSidebarCollapsed = false;

  constructor(
    private auth: AuthService,
    private router: Router,
    private toast: ToastService,
  ) {}

  // Ambil data user yang lagi login buat ditampilin di layout (sidebar), sekaligus validasi sesi
  // masih valid pas CMS dibuka.
  ngOnInit(): void {
    this.isSidebarCollapsed = localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === '1';

    this.auth.me().subscribe({
      next: (me) => (this.user = me),
      error: () => {
        this.auth.logout();
        this.toast.warning('Sesi kamu berakhir, silakan login lagi.');
        this.router.navigate(['/admin-cms/login']);
      },
    });

    // Tutup drawer mobile otomatis abis pindah halaman, biar user gak perlu nutup manual
    // tiap habis klik menu (klik menu item juga udah nutup lewat (closed), ini jaga-jaga
    // buat navigasi lain, ex: klik tombol back browser).
    this.router.events.subscribe((event) => {
      if (event instanceof NavigationEnd) {
        this.isSidebarOpen = false;
      }
    });
  }

  toggleSidebar(): void {
    this.isSidebarOpen = !this.isSidebarOpen;
  }

  closeSidebar(): void {
    this.isSidebarOpen = false;
  }

  // Toggle collapse sidebar desktop (icon-only) + persist ke localStorage biar kepake lagi pas reload.
  toggleSidebarCollapse(): void {
    this.isSidebarCollapsed = !this.isSidebarCollapsed;
    localStorage.setItem(SIDEBAR_COLLAPSED_KEY, this.isSidebarCollapsed ? '1' : '0');
  }
}
