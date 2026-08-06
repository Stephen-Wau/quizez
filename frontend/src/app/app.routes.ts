import { Routes } from '@angular/router';

import { authGuard } from './core/auth/auth.guard';

export const routes: Routes = [
  { path: '', pathMatch: 'full', redirectTo: 'admin-cms' },
  {
    path: 'admin-cms/login',
    loadComponent: () =>
      import('./features/admin-cms/login/login.component').then((m) => m.LoginComponent),
  },
  {
    path: 'admin-cms',
    canActivate: [authGuard],
    loadComponent: () =>
      import('./shared/layout/cms-layout/cms-layout.component').then((m) => m.CmsLayoutComponent),
    children: [
      {
        path: '',
        loadComponent: () =>
          import('./features/admin-cms/dashboard/dashboard.component').then(
            (m) => m.DashboardComponent
          ),
      },
    ],
  },
  { path: '**', redirectTo: 'admin-cms' },
];
