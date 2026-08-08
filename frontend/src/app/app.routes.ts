import { Routes } from '@angular/router';

import { authGuard } from './core/auth/auth.guard';

// Daftar route aplikasi, semua komponen di-lazy load pake loadComponent biar bundle awal kecil.
export const routes: Routes = [
  { path: '', pathMatch: 'full', redirectTo: 'admin-cms' },
  {
    path: 'admin-cms/login',
    // Lazy load halaman login, ga perlu authGuard karena ini justru buat login.
    loadComponent: () =>
      import('./features/admin-cms/login/login.component').then((m) => m.LoginComponent),
  },
  {
    path: 'admin-cms',
    // Semua route di bawah admin-cms wajib login dulu, dicek lewat authGuard.
    canActivate: [authGuard],
    loadComponent: () =>
      import('./shared/layout/cms-layout/cms-layout.component').then((m) => m.CmsLayoutComponent),
    children: [
      {
        path: '',
        // Default child route, tampil pas buka /admin-cms tanpa path tambahan.
        loadComponent: () =>
          import('./features/admin-cms/dashboard/dashboard.component').then(
            (m) => m.DashboardComponent
          ),
      },
      {
        path: 'quiz',
        loadComponent: () =>
          import('./features/admin-cms/quiz/quiz.component').then((m) => m.QuizComponent),
      },
      {
        path: 'question-answer',
        loadComponent: () =>
          import('./features/admin-cms/question-answer/question-answer.component').then(
            (m) => m.QuestionAnswerComponent
          ),
      },
    ],
  },
  // Fallback buat path yang ga dikenal, lempar balik ke admin-cms.
  { path: '**', redirectTo: 'admin-cms' },
];
