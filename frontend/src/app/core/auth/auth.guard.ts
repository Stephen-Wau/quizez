import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';

import { AuthService } from './auth.service';

// Guard buat route admin-cms, blokir akses kalau token ga ada/expired dan redirect ke halaman login.
export const authGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);

  // Token masih valid, boleh lanjut ke route yang dituju.
  if (auth.hasValidToken()) {
    return true;
  }

  return router.createUrlTree(['/admin-cms/login']);
};
