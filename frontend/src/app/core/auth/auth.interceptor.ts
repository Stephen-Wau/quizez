import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';

import { AuthService } from './auth.service';

// Interceptor global: nempelin Bearer token ke tiap request kalau ada, dan auto logout pas dapet 401.
export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const isPublicApiRequest = req.url.includes('/api/public/');

  const token = auth.getToken();
  // Cuma clone request buat nambahin header kalau emang ada token, biar request tanpa token tetep jalan normal.
  const authedReq = token && !isPublicApiRequest
    ? req.clone({ setHeaders: { Authorization: `Bearer ${token}` } })
    : req;

  return next(authedReq).pipe(
    catchError((err) => {
      // Request publik harus tetap berdiri sendiri; jangan ikut dipaksa logout/redirect walau
      // backend balikin 401/403, karena halaman share link memang bisa dibuka tanpa sesi CMS.
      if (err.status === 401 && !isPublicApiRequest) {
        auth.logout();
        router.navigate(['/admin-cms/login']);
      }
      return throwError(() => err);
    })
  );
};
