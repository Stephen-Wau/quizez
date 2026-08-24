import { Injectable, signal } from '@angular/core';

const THEME_KEY = 'quizez_cms_theme';

// Kelola preferensi tema (light/dark) khusus area CMS, persist ke localStorage biar gak reset tiap reload.
@Injectable({ providedIn: 'root' })
export class ThemeService {
  readonly isDark = signal(localStorage.getItem(THEME_KEY) === 'dark');

  // Balik tema aktif & simpen preferensinya, dipanggil dari tombol switch di sidebar.
  toggle(): void {
    this.isDark.update((value) => !value);
    localStorage.setItem(THEME_KEY, this.isDark() ? 'dark' : 'light');
  }
}
