import { CommonModule } from '@angular/common';
import { Component, Input } from '@angular/core';
import { Router, RouterLink, RouterLinkActive } from '@angular/router';

import { AuthService, Me } from '../../../core/auth/auth.service';
import { CMS_MENU_ITEMS } from '../cms-menu.config';

@Component({
  selector: 'app-sidebar',
  standalone: true,
  imports: [CommonModule, RouterLink, RouterLinkActive],
  templateUrl: './sidebar.component.html',
  styleUrl: './sidebar.component.scss',
})
export class SidebarComponent {
  @Input() user: Me | null = null;

  menuItems = CMS_MENU_ITEMS;

  constructor(private auth: AuthService, private router: Router) {}

  // Dipanggil pas user klik tombol logout di sidebar: bersihin sesi terus lempar balik ke halaman login.
  logout(): void {
    this.auth.logout();
    this.router.navigate(['/admin-cms/login']);
  }
}
