import { CommonModule } from '@angular/common';
import { Component, OnInit, TemplateRef, ViewChild } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';

import { AuthService, Me } from '../../../core/auth/auth.service';
import { ButtonComponent } from '../../../shared/ui/button/button.component';
import {
  DataTableColumn,
  DataTableComponent,
  DataTableQuery,
} from '../../../shared/ui/data-table/data-table.component';
import { InputComponent } from '../../../shared/ui/input/input.component';
import { ModalComponent } from '../../../shared/ui/modal/modal.component';
import { ToastService } from '../../../shared/ui/toast/toast.service';
import { confirmAndDelete } from '../../../shared/utils/confirm-delete.util';
import { fieldError } from '../../../shared/utils/form-error.util';
import { loadPagedList } from '../../../shared/utils/load-paged-list.util';
import {
  AdminRole,
  AdminUser,
  AdminUserPayload,
  AuditLog,
  CollaborationPermissionService,
} from './collaboration-permission.service';

type CollaborationTab = 'admins' | 'audit';

@Component({
  selector: 'app-collaboration-permission',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    ButtonComponent,
    DataTableComponent,
    InputComponent,
    ModalComponent,
  ],
  templateUrl: './collaboration-permission.component.html',
  styleUrl: './collaboration-permission.component.scss',
})
export class CollaborationPermissionComponent implements OnInit {
  @ViewChild('adminRoleTpl', { static: true }) adminRoleTpl!: TemplateRef<unknown>;
  @ViewChild('adminActionTpl', { static: true }) adminActionTpl!: TemplateRef<unknown>;
  @ViewChild('auditActorTpl', { static: true }) auditActorTpl!: TemplateRef<unknown>;
  @ViewChild('auditActionTpl', { static: true }) auditActionTpl!: TemplateRef<unknown>;
  @ViewChild('auditEntityTpl', { static: true }) auditEntityTpl!: TemplateRef<unknown>;

  activeTab: CollaborationTab = 'audit';
  me: Me | null = null;

  adminUsers: AdminUser[] = [];
  adminColumns: DataTableColumn[] = [];
  adminTotalCount = 0;
  adminPageSize = 10;
  adminQuery: DataTableQuery = {};

  auditLogs: AuditLog[] = [];
  auditColumns: DataTableColumn[] = [];
  auditTotalCount = 0;
  auditPageSize = 10;
  auditQuery: DataTableQuery = {};

  isFormModalOpen = false;
  isSaving = false;
  editingAdminId: number | null = null;

  form;

  constructor(
    private fb: FormBuilder,
    private auth: AuthService,
    private collaborationPermissionService: CollaborationPermissionService,
    private toast: ToastService,
  ) {
    this.form = this.fb.group({
      name: ['', Validators.required],
      email: ['', [Validators.required, Validators.email]],
      role: ['editor' as AdminRole, Validators.required],
      password: [''],
    });
  }

  ngOnInit(): void {
    this.setupColumns();

    const currentUser = this.auth.currentUser();
    if (currentUser) {
      this.applyCurrentUser(currentUser);
      return;
    }

    this.auth.me().subscribe({
      next: (me) => this.applyCurrentUser(me),
      error: () => this.toast.error('Gagal memuat sesi admin.'),
    });
  }

  get isSuperAdmin(): boolean {
    return this.me?.role === 'super_admin';
  }

  get passwordLabel(): string {
    return this.editingAdminId ? 'Password Baru (opsional)' : 'Password';
  }

  // fieldError bungkus helper validasi supaya template tetap ringkas.
  fieldError(name: string): string {
    return fieldError(this.form, name, (control) => {
      if (control.hasError('email')) return 'Format email tidak valid.';
      return undefined;
    });
  }

  // switchTab ganti panel aktif, dan saat masuk tab tertentu langsung load datanya bila perlu.
  switchTab(tab: CollaborationTab): void {
    if (tab === 'admins' && !this.isSuperAdmin) return;
    this.activeTab = tab;
    if (tab === 'admins') {
      this.loadAdminUsers();
      return;
    }
    this.loadAuditLogs();
  }

  // loadAdminUsers ambil daftar admin CMS sesuai query DataTable server-side.
  loadAdminUsers(): void {
    if (!this.isSuperAdmin) return;
    loadPagedList(
      this.collaborationPermissionService.listAdminUsers(this.adminQuery),
      this.toast,
      'Gagal memuat daftar admin.',
      (data, totalCount, pageSize) => {
        this.adminUsers = data;
        this.adminTotalCount = totalCount;
        this.adminPageSize = pageSize;
      },
    );
  }

  // loadAuditLogs ambil jejak audit aktivitas admin untuk investigasi operasional.
  loadAuditLogs(): void {
    loadPagedList(
      this.collaborationPermissionService.listAuditLogs(this.auditQuery),
      this.toast,
      'Gagal memuat audit log.',
      (data, totalCount, pageSize) => {
        this.auditLogs = data;
        this.auditTotalCount = totalCount;
        this.auditPageSize = pageSize;
      },
    );
  }

  onAdminQueryChange(query: DataTableQuery): void {
    this.adminQuery = query;
    this.loadAdminUsers();
  }

  onAuditQueryChange(query: DataTableQuery): void {
    this.auditQuery = query;
    this.loadAuditLogs();
  }

  // openCreateModal siapkan form kosong untuk menambah admin CMS baru.
  openCreateModal(): void {
    this.editingAdminId = null;
    this.form.reset({
      name: '',
      email: '',
      role: 'editor',
      password: '',
    });
    this.isFormModalOpen = true;
  }

  // openEditModal isi form dari akun admin yang dipilih; password sengaja dikosongkan.
  openEditModal(item: AdminUser): void {
    this.editingAdminId = item.id;
    this.form.reset({
      name: item.name,
      email: item.email,
      role: item.role,
      password: '',
    });
    this.isFormModalOpen = true;
  }

  closeFormModal(): void {
    this.isFormModalOpen = false;
  }

  // saveAdminUser kirim create/update admin, lalu reload tabel kalau backend sukses.
  saveAdminUser(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      this.toast.error('Ada field admin yang belum valid.');
      return;
    }

    const raw = this.form.getRawValue();
    const payload: AdminUserPayload = {
      name: raw.name ?? '',
      email: raw.email ?? '',
      role: (raw.role ?? 'editor') as AdminRole,
    };
    const password = (raw.password ?? '').trim();
    if (!this.editingAdminId || password) {
      payload.password = password;
    }

    this.isSaving = true;
    const request = this.editingAdminId
      ? this.collaborationPermissionService.updateAdminUser(this.editingAdminId, payload)
      : this.collaborationPermissionService.createAdminUser(payload);

    request.subscribe({
      next: () => {
        this.isSaving = false;
        this.toast.success(this.editingAdminId ? 'Admin berhasil diperbarui.' : 'Admin berhasil ditambahkan.');
        this.closeFormModal();
        this.loadAdminUsers();
      },
      error: (err) => {
        this.isSaving = false;
        const message = typeof err?.error === 'string' && err.error ? err.error : 'Gagal menyimpan admin.';
        this.toast.error(message);
      },
    });
  }

  removeAdmin(item: AdminUser): void {
    confirmAndDelete(
      `Hapus admin "${item.name}"?`,
      () => this.collaborationPermissionService.deleteAdminUser(item.id),
      this.toast,
      'Admin berhasil dihapus.',
      'Gagal menghapus admin.',
      () => this.loadAdminUsers(),
    );
  }

  // roleLabel bikin role backend lebih enak dibaca di badge tabel dan header profil.
  roleLabel(role: AdminRole | string | null): string {
    return role === 'super_admin' ? 'Super Admin' : role === 'editor' ? 'Editor' : '-';
  }

  // formatAuditDate bikin timestamp audit lebih mudah dipindai saat investigasi.
  formatAuditDate(value: string): string {
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return new Intl.DateTimeFormat('id-ID', {
      dateStyle: 'medium',
      timeStyle: 'short',
    }).format(date);
  }

  // formatActionKey ubah action_key teknis jadi label yang lebih ramah dibaca di tab audit.
  formatActionKey(value: string): string {
    const dictionary: Record<string, string> = {
      'auth.login': 'Login CMS',
      'quiz.create': 'Buat Quiz',
      'quiz.update': 'Edit Quiz',
      'quiz.delete': 'Hapus Quiz',
      'quiz.duplicate': 'Duplicate Quiz',
      'quiz.generate_share_link': 'Generate Share Link',
      'question.create': 'Buat Soal',
      'question.update': 'Edit Soal',
      'question.delete': 'Hapus Soal',
      'question_bank.create': 'Buat Bank Soal',
      'question_bank.update': 'Edit Bank Soal',
      'question_bank.delete': 'Hapus Bank Soal',
      'question_bank.import': 'Import Bank Soal',
      'question_bank.copy_to_quiz': 'Salin Bank Soal ke Quiz',
      'analytics.export': 'Export Analytics',
      'admin_user.create': 'Tambah Admin',
      'admin_user.update': 'Edit Admin',
      'admin_user.delete': 'Hapus Admin',
    };
    return dictionary[value] ?? value;
  }

  // formatEntity ringkas entity_type + entity_id supaya investigasi tidak perlu membaca JSON mentah.
  formatEntity(item: AuditLog): string {
    const typeLabel = item.entity_type.replaceAll('_', ' ');
    return item.entity_id ? `${typeLabel} #${item.entity_id}` : typeLabel;
  }

  private applyCurrentUser(me: Me): void {
    this.me = me;
    this.activeTab = me.role === 'super_admin' ? 'admins' : 'audit';
    if (me.role === 'super_admin') {
      this.loadAdminUsers();
      return;
    }
    this.loadAuditLogs();
  }

  // setupColumns siapkan definisi DataTable sekali saja supaya template tab tetap rapi.
  private setupColumns(): void {
    this.adminColumns = [
      { name: 'Nama', prop: 'name' },
      { name: 'Email', prop: 'email' },
      { name: 'Role', prop: 'role', cellTemplate: this.adminRoleTpl },
      { name: 'Dibuat', prop: 'created_at' },
      {
        name: 'Action',
        sortable: false,
        cellTemplate: this.adminActionTpl,
        headerClass: 'app-data-table__cell--actions',
        cellClass: 'app-data-table__cell--actions',
      },
    ];

    this.auditColumns = [
      { name: 'Waktu', prop: 'created_at' },
      { name: 'Aktor', prop: 'actor_name', cellTemplate: this.auditActorTpl },
      { name: 'Aksi', prop: 'action_key', cellTemplate: this.auditActionTpl },
      { name: 'Entitas', prop: 'entity_type', cellTemplate: this.auditEntityTpl },
      { name: 'Deskripsi', prop: 'description' },
    ];
  }
}
