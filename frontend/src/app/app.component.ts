import { Component, HostBinding } from '@angular/core';
import { NavigationEnd, Router, RouterOutlet } from '@angular/router';
import { ThemeService } from './core/theme/theme.service';
import { ToastContainerComponent } from './shared/ui/toast/toast-container.component';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, ToastContainerComponent],
  templateUrl: './app.component.html',
  styleUrl: './app.component.scss'
})
export class AppComponent {
  title = 'frontend';
  // Toast di-render di root (bukan di dalam CmsLayoutComponent) biar bisa muncul juga di halaman
  // login/public-form/landing -- jadi dark mode-nya perlu di-cek manual di sini (bukan otomatis
  // ke-cover sama :host(.theme-dark) punya cms-layout) supaya toast ikut gelap pas lagi di area CMS,
  // tapi tetep terang di halaman publik walau preferensi dark mode-nya nyala.
  private isCmsRoute = window.location.pathname.startsWith('/admin-cms') && window.location.pathname !== '/admin-cms/login';

  @HostBinding('class.theme-dark') get isDarkTheme(): boolean {
    return this.isCmsRoute && this.theme.isDark();
  }

  constructor(private theme: ThemeService, router: Router) {
    router.events.subscribe((event) => {
      if (event instanceof NavigationEnd) {
        this.isCmsRoute = event.urlAfterRedirects.startsWith('/admin-cms') && event.urlAfterRedirects !== '/admin-cms/login';
      }
    });
  }
}
