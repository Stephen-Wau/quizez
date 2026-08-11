import { ApplicationConfig, importProvidersFrom, provideZoneChangeDetection } from '@angular/core';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { provideRouter } from '@angular/router';
import { LucideAngularModule, X, RefreshCw, File, Paperclip, ImagePlus, Trash2, LogIn, Plus, Pencil, Save, Filter } from 'lucide-angular';

import { routes } from './app.routes';
import { authInterceptor } from './core/auth/auth.interceptor';

// Konfigurasi provider global aplikasi (router, http client + interceptor auth, icon set lucide).
export const appConfig: ApplicationConfig = {
  providers: [
    provideZoneChangeDetection({ eventCoalescing: true }),
    provideRouter(routes),
    provideHttpClient(withInterceptors([authInterceptor])),
    importProvidersFrom(
      LucideAngularModule.pick({ X, RefreshCw, File, Paperclip, ImagePlus, Trash2, LogIn, Plus, Pencil, Save, Filter })
    ),
  ]
};
