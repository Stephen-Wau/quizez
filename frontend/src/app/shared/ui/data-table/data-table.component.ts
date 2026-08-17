import { Component, EventEmitter, Input, OnChanges, Output, SimpleChanges, TemplateRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { NgxDatatableModule, SortEvent } from '@swimlane/ngx-datatable';
import { LucideAngularModule } from 'lucide-angular';

import { BadgeComponent, BadgeVariant } from '../badge/badge.component';

// Definisi 1 kolom: `prop` buat nampilin value langsung, atau `cellTemplate` buat cell custom
// (ex: format tanggal, tombol aksi). `prop` tetap wajib diisi biar sort jalan walau kolomnya
// pakai cellTemplate custom (ngx-datatable sort berdasarkan `prop`, bukan hasil render template).
export interface DataTableColumn {
  name: string;
  prop?: string;
  sortable?: boolean;
  cellTemplate?: TemplateRef<unknown>;
  headerClass?: string;
  cellClass?: string;
}

// Kontrak query param standar buat semua API list yang FE-nya pakai <app-data-table serverSide>.
// Dikirim apa adanya sebagai query string BE: ?searchword=...&sort_by=...&sort_dir=asc|desc.
export interface DataTableQuery {
  searchword?: string;
  sort_by?: string;
  sort_dir?: 'asc' | 'desc';
  page?: number;
  per_page?: number;
}

// Bentuk meta pagination standar yang dibalikin semua API list BE (lihat listquery.Meta di Go).
export interface DataTableMeta {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

// Bentuk response standar semua API list: {"data": [...], "meta": {...}}.
export interface PagedResult<T> {
  data: T[];
  meta: DataTableMeta;
}

const SEARCH_DEBOUNCE_MS = 300;

// Tabel global reusable, dipakai semua menu CMS yang butuh list data lewat <app-data-table>.
// Bungkus ngx-datatable + styling standar + search box.
//
// Dua mode:
// - serverSide=false (default): search & sort dikerjakan di client, cocok buat tabel kecil.
// - serverSide=true: search & sort di-emit lewat (search)/(sortChange), parent yang manggil ulang
//   API pakai DataTableQuery — dipakai kalau BE sudah support ?searchword/&sort_by/&sort_dir.
@Component({
  selector: 'app-data-table',
  standalone: true,
  imports: [CommonModule, FormsModule, NgxDatatableModule, LucideAngularModule, BadgeComponent],
  templateUrl: './data-table.component.html',
  styleUrl: './data-table.component.scss',
})
export class DataTableComponent implements OnChanges {
  @Input() rows: unknown[] = [];
  @Input() columns: DataTableColumn[] = [];
  @Input() emptyMessage = 'Tidak ada data.';
  // Set false buat sembunyiin search box (ex: tabel kecil yang gak butuh filter).
  @Input() searchable = true;
  @Input() searchPlaceholder = 'Search...';
  // Field yang dicari (mode client-side); kosong = cari di semua kolom yang punya `prop`.
  @Input() searchKeys: string[] = [];
  // true = search/sort di-emit ke parent (panggil API), bukan difilter/sort di client.
  @Input() serverSide = false;
  // Total baris di server (dari meta.total) — cuma dipakai mode serverSide buat hitung pager.
  @Input() totalCount = 0;
  // Jumlah baris per halaman — harus sinkron sama per_page yang diminta ke BE.
  @Input() pageSize = 10;

  @Output() search = new EventEmitter<DataTableQuery>();
  // Dipancing tombol refresh di sebelah search box — parent tinggal panggil ulang load list-nya
  // pakai query/halaman yang lagi aktif (gak perlu reset search/sort/page).
  @Output() refresh = new EventEmitter<void>();

  searchTerm = '';
  currentPage = 1; // 1-indexed, dipakai buat kirim `page` ke BE & offset ke ngx-datatable
  // Property biasa (bukan getter!) — sengaja, karena ngx-datatable ngecek reference [rows] tiap
  // change-detection cycle. Getter yang manggil .filter() bakal selalu balikin array reference
  // baru tiap cycle, bikin ngx-datatable ngira data berubah terus-menerus → infinite render loop.
  filteredRows: unknown[] = [];

  private searchDebounceHandle?: ReturnType<typeof setTimeout>;
  private lastSort: { prop: string; dir: 'asc' | 'desc' } | null = null;

  // Re-apply filter tiap kali @Input rows/columns berubah dari parent (ex: hasil load API baru).
  ngOnChanges(changes: SimpleChanges): void {
    // Cuma perlu filter ulang kalau data atau daftar kolom yang berubah, bukan input lain (ex: pageSize).
    if (changes['rows'] || changes['columns']) {
      this.applyFilter();
    }
  }

  // Dipanggil dari (ngModelChange) search box. Mode client: filter langsung. Mode server:
  // debounce dulu (biar gak nge-hit API tiap ketikan huruf) baru emit ke parent.
  onSearchChange(): void {
    if (!this.serverSide) {
      this.applyFilter();
      return;
    }
    if (this.searchDebounceHandle) clearTimeout(this.searchDebounceHandle);
    this.searchDebounceHandle = setTimeout(() => {
      this.currentPage = 1; // search baru selalu balik ke halaman 1
      this.emitQuery();
    }, SEARCH_DEBOUNCE_MS);
  }

  // Dipanggil dari (sort) ngx-datatable. Mode server: emit query baru ke parent (BE yang sort).
  // Mode client: biarin ngx-datatable sort sendiri secara internal (gak perlu handle apa-apa).
  onSort(event: SortEvent): void {
    if (!this.serverSide) return;
    const sort = event.sorts?.[0];
    this.lastSort = sort ? { prop: String(sort.prop), dir: sort.dir as 'asc' | 'desc' } : null;
    this.currentPage = 1; // sort baru selalu balik ke halaman 1
    this.emitQuery();
  }

  // Dipanggil dari (page) ngx-datatable (klik nomor halaman/prev/next di footer pager).
  onPage(event: { offset: number }): void {
    if (!this.serverSide) return;
    this.currentPage = event.offset + 1; // ngx-datatable offset 0-indexed
    this.emitQuery();
  }

  // Dipanggil dari tombol refresh — biarin parent yang panggil ulang API-nya sendiri.
  onRefreshClick(): void {
    this.refresh.emit();
  }

  // Semua kolom aksi diseragamkan rata kanan dari komponen tabel reusable, jadi parent
  // tidak perlu set class manual lagi untuk tiap halaman admin.
  isActionColumn(column: DataTableColumn): boolean {
    return column.name.trim().toLowerCase() === 'action';
  }

  // Kolom status yang tidak pakai template custom tetap dibungkus badge konsisten:
  // active = hijau terang, inactive = abu secondary, selain itu pakai warna netral.
  isStatusColumn(column: DataTableColumn): boolean {
    return (column.prop ?? '').trim().toLowerCase() === 'status' && !column.cellTemplate;
  }

  statusBadgeVariant(value: unknown): BadgeVariant {
    const normalized = String(value ?? '').trim().toLowerCase();
    if (normalized === 'active') return 'emerald';
    if (normalized === 'inactive') return 'gray';
    return 'default';
  }

  // Kumpulin state search/sort/page jadi satu DataTableQuery & emit ke parent (mode serverSide).
  private emitQuery(): void {
    this.search.emit({
      searchword: this.searchTerm.trim() || undefined,
      sort_by: this.lastSort?.prop,
      sort_dir: this.lastSort?.dir,
      page: this.currentPage,
      per_page: this.pageSize,
    });
  }

  // Filter rows di client-side berdasarkan searchTerm — cuma dipakai kalau serverSide=false.
  private applyFilter(): void {
    if (this.serverSide) {
      this.filteredRows = this.rows;
      return;
    }

    const term = this.searchTerm.trim().toLowerCase();
    if (!term) {
      this.filteredRows = this.rows;
      return;
    }

    const keys =
      this.searchKeys.length > 0
        ? this.searchKeys
        : this.columns.filter((c) => c.prop).map((c) => c.prop!);

    this.filteredRows = this.rows.filter((row) =>
      keys.some((key) =>
        String((row as Record<string, unknown>)[key] ?? '')
          .toLowerCase()
          .includes(term),
      ),
    );
  }
}
