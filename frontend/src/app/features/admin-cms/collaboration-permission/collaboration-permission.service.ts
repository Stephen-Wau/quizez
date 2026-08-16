import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import { environment } from '../../../../environments/environment';
import { DataTableQuery, PagedResult } from '../../../shared/ui/data-table/data-table.component';

export type AdminRole = 'super_admin' | 'editor';

export interface AdminUser {
  id: number;
  email: string;
  name: string;
  role: AdminRole;
  created_at?: string;
}

export interface AdminUserPayload {
  email: string;
  name: string;
  role: AdminRole;
  password?: string;
}

export interface AuditLog {
  id: number;
  actor_user_id: number | null;
  actor_name: string;
  actor_email: string;
  action_key: string;
  entity_type: string;
  entity_id: number | null;
  description: string;
  ip_address: string;
  user_agent: string;
  created_at: string;
}

@Injectable({ providedIn: 'root' })
export class CollaborationPermissionService {
  private adminUsersUrl = `${environment.apiUrl}/api/admin-users`;
  private auditLogsUrl = `${environment.apiUrl}/api/audit-logs`;

  constructor(private http: HttpClient) {}

  // listAdminUsers ambil daftar admin CMS buat DataTable menu kolaborasi.
  listAdminUsers(query: DataTableQuery = {}): Observable<PagedResult<AdminUser>> {
    return this.http.get<PagedResult<AdminUser>>(this.adminUsersUrl, {
      params: buildDataTableParams(query),
    });
  }

  // createAdminUser bikin akun admin CMS baru dari modal form super admin.
  createAdminUser(payload: AdminUserPayload): Observable<AdminUser> {
    return this.http.post<AdminUser>(this.adminUsersUrl, payload);
  }

  // updateAdminUser perbarui profil/role admin, opsional ikut ganti password bila diisi.
  updateAdminUser(id: number, payload: AdminUserPayload): Observable<AdminUser> {
    return this.http.put<AdminUser>(`${this.adminUsersUrl}/${id}`, payload);
  }

  // deleteAdminUser hapus akun admin CMS yang dipilih dari tabel admin.
  deleteAdminUser(id: number): Observable<void> {
    return this.http.delete<void>(`${this.adminUsersUrl}/${id}`);
  }

  // listAuditLogs ambil jejak aksi admin untuk tab audit log.
  listAuditLogs(query: DataTableQuery = {}): Observable<PagedResult<AuditLog>> {
    return this.http.get<PagedResult<AuditLog>>(this.auditLogsUrl, {
      params: buildDataTableParams(query),
    });
  }
}

// buildDataTableParams seragamkan query string list server-side supaya semua tabel admin konsisten.
function buildDataTableParams(query: DataTableQuery): HttpParams {
  let params = new HttpParams();
  if (query.searchword) params = params.set('searchword', query.searchword);
  if (query.sort_by) params = params.set('sort_by', query.sort_by);
  if (query.sort_dir) params = params.set('sort_dir', query.sort_dir);
  if (query.page) params = params.set('page', query.page);
  if (query.per_page) params = params.set('per_page', query.per_page);
  return params;
}
