import { HttpClient } from '@angular/common/http';
import { Injectable, signal } from '@angular/core';
import { Observable, tap } from 'rxjs';

import { environment } from '../../../environments/environment';

const TOKEN_KEY = 'auth_token';

export interface LoginResponse {
  token: string;
  expires_at: string;
}

export interface Me {
  id: number;
  email: string;
  name: string;
}

@Injectable({ providedIn: 'root' })
export class AuthService {
  isLoggedIn = signal(this.hasValidToken());

  constructor(private http: HttpClient) {}

  // Login ke backend, kalau sukses langsung simpan token ke localStorage dan update signal isLoggedIn.
  login(email: string, password: string): Observable<LoginResponse> {
    return this.http
      .post<LoginResponse>(`${environment.apiUrl}/api/auth/login`, { email, password })
      .pipe(
        tap((res) => {
          localStorage.setItem(TOKEN_KEY, res.token);
          this.isLoggedIn.set(true);
        })
      );
  }

  // Hapus token dari localStorage dan set isLoggedIn ke false, dipanggil pas user logout atau token expired.
  logout(): void {
    localStorage.removeItem(TOKEN_KEY);
    this.isLoggedIn.set(false);
  }

  // Ambil data user yang lagi login dari backend.
  me(): Observable<Me> {
    return this.http.get<Me>(`${environment.apiUrl}/api/auth/me`);
  }

  // Ambil raw token dari localStorage, dipakai interceptor buat nempelin header Authorization.
  getToken(): string | null {
    return localStorage.getItem(TOKEN_KEY);
  }

  // Cek token ada dan belum expired (decode payload JWT manual tanpa lib tambahan).
  hasValidToken(): boolean {
    const token = this.getToken();
    if (!token) return false;

    try {
      const payload = JSON.parse(atob(token.split('.')[1]));
      // exp di JWT satuannya detik, Date.now() satuannya ms, makanya dikali 1000.
      return payload.exp * 1000 > Date.now();
    } catch {
      // Token rusak/format ga valid dianggap ga valid aja, jangan sampe throw.
      return false;
    }
  }
}
