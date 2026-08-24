import { CommonModule } from '@angular/common';
import { Component, EventEmitter, Input, Output } from '@angular/core';
import { Router, RouterLink, RouterLinkActive } from '@angular/router';
import { LucideAngularModule } from 'lucide-angular';

import { AuthService, Me } from '../../../core/auth/auth.service';
import { ThemeService } from '../../../core/theme/theme.service';
import { ButtonComponent } from '../../ui/button/button.component';
import { CMS_MENU_ITEMS } from '../cms-menu.config';

// Sidebar kiri CMS: section profile+logout di atas, section menu navigasi (filter per role) di bawah.
@Component({
  selector: 'app-sidebar',
  standalone: true,
  imports: [CommonModule, RouterLink, RouterLinkActive, ButtonComponent, LucideAngularModule],
  templateUrl: './sidebar.component.html',
  styleUrl: './sidebar.component.scss',
})
export class SidebarComponent {
  @Input() user: Me | null = null;
  // Kontrol drawer di mobile (<=880px, lihat sidebar.component.scss) — di desktop diabaikan
  // karena sidebar-nya emang selalu keliatan (position: static).
  @Input() isOpen = false;
  // Di-emit tiap kali sidebar mestinya ditutup dari dalam (klik menu item, atau logout),
  // parent (CmsLayoutComponent) yang pegang source-of-truth isOpen-nya.
  @Output() closed = new EventEmitter<void>();
  // Collapse/expand sidebar desktop (icon-only vs full width) — beda dari isOpen (drawer mobile).
  // Source-of-truth-nya juga di parent (CmsLayoutComponent), biar kepersist ke localStorage di sana.
  @Input() collapsed = false;
  @Output() toggleCollapse = new EventEmitter<void>();

  constructor(
    private auth: AuthService,
    private router: Router,
    // Public biar kepakai langsung di template (toggle & baca state isDark), gak perlu Input/Output
    // tambahan cuma buat nyambung ke ThemeService yang udah root-provided & shared sama CmsLayoutComponent.
    public theme: ThemeService,
  ) {}

  // Menu difilter per role: item tanpa `roles` keliatan buat semua, item dengan `roles` cuma
  // keliatan kalau role user login termasuk di daftarnya (ex: menu Kolaborasi & Permission khusus super_admin).
  get menuItems(): typeof CMS_MENU_ITEMS {
    const role = this.user?.role;
    return CMS_MENU_ITEMS.filter((item) => !item.roles || (role ? item.roles.includes(role) : true));
  }

  get roleLabel(): string {
    if (this.user?.role === 'super_admin') return 'Super Admin';
    if (this.user?.role === 'editor') return 'Editor';
    return '';
  }

  // Dipanggil pas user klik tombol logout di sidebar: bersihin sesi terus lempar balik ke halaman login.
  logout(): void {
    this.auth.logout();
    this.router.navigate(['/admin-cms/login']);
    this.closed.emit();
  }

  // Dipanggil pas klik menu item — di mobile, drawer harus nutup otomatis abis pindah halaman.
  // Di desktop ini gak ngefek apa-apa (isOpen diabaikan lewat CSS), jadi aman dipanggil selalu.
  onMenuItemClick(): void {
    this.closed.emit();
  }
}
