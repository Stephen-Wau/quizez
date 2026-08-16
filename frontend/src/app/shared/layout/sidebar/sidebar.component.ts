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

  constructor(private auth: AuthService, private router: Router) {}

  get menuItems(): typeof CMS_MENU_ITEMS {
    const role = this.user?.role;
    return CMS_MENU_ITEMS.filter((item) => !item.roles || (role ? item.roles.includes(role) : true));
  }

  // Dipanggil pas user klik tombol logout di sidebar: bersihin sesi terus lempar balik ke halaman login.
  logout(): void {
    this.auth.logout();
    this.router.navigate(['/admin-cms/login']);
  }
}
