import {
  AfterViewInit,
  Component,
  ElementRef,
  Input,
  OnDestroy,
  ViewChild,
  forwardRef,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { ControlValueAccessor, NG_VALUE_ACCESSOR } from '@angular/forms';
import flatpickr from 'flatpickr';
import monthSelectPlugin from 'flatpickr/dist/plugins/monthSelect';
import { Instance } from 'flatpickr/dist/types/instance';

let nextId = 0;

// Month picker global, pengganti <input type="month"> bawaan browser (yang stylingnya beda-beda
// tiap browser/OS) — dipakai di semua form CMS yang punya field bulan-tahun (Work Histories,
// Education, dst). Implement ControlValueAccessor supaya langsung bisa dipasang formControlName,
// ex: <app-month-picker formControlName="start_date" label="Mulai" />. Value yang di-emit ke form
// selalu format "YYYY-MM" (konsisten sama kontrak API BE).
@Component({
  selector: 'app-month-picker',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './month-picker.component.html',
  styleUrl: './month-picker.component.scss',
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => MonthPickerComponent),
      multi: true,
    },
  ],
})
export class MonthPickerComponent implements ControlValueAccessor, AfterViewInit, OnDestroy {
  @ViewChild('inputEl') inputEl!: ElementRef<HTMLInputElement>;

  @Input() label = '';
  @Input() errorMessage = '';
  @Input() invalid = false;

  id = `app-month-picker-${nextId++}`;
  disabled = false;

  private fp?: Instance;
  private value = '';
  private onChange: (value: string) => void = () => {};
  private onTouched: () => void = () => {};

  // Dropdown tahun custom (scroll + search) — flatpickr bawaannya cuma input angka polos
  // dengan tombol atas/bawah, gak enak buat lompat jauh (ex: cari tahun 1995).
  private yearDropdownEl: HTMLElement | null = null;
  private readonly outsideClickHandler = (e: MouseEvent): void => {
    const target = e.target as Node;
    if (this.yearDropdownEl && !this.yearDropdownEl.contains(target) && target !== this.fp?.currentYearElement) {
      this.closeYearDropdown();
    }
  };

  // Init flatpickr setelah view siap (butuh elemen <input> beneran buat di-attach).
  ngAfterViewInit(): void {
    this.fp = flatpickr(this.inputEl.nativeElement, {
      plugins: [monthSelectPlugin({ shorthand: true, dateFormat: 'Y-m', altFormat: 'M Y' })],
      altInput: true,
      disableMobile: true,
      defaultDate: this.value || undefined,
      onChange: (selectedDates) => {
        const date = selectedDates[0];
        const next = date ? this.toYearMonth(date) : '';
        this.value = next;
        this.onChange(next);
      },
      onClose: () => this.onTouched(),
    });
    this.fp.set('clickOpens', !this.disabled);
    this.setupYearPicker();
  }

  ngOnDestroy(): void {
    document.removeEventListener('click', this.outsideClickHandler);
    this.fp?.destroy();
  }

  // Dipanggil Angular forms saat set value dari luar (ex: patchValue pas load data existing).
  writeValue(value: string): void {
    this.value = value ?? '';
    // fp belum ke-init kalau writeValue kepanggil sebelum ngAfterViewInit (ex: reset() di constructor).
    this.fp?.setDate(this.value || '', false);
  }

  // Daftarin callback yang dipanggil onChange flatpickr tiap kali user pilih bulan-tahun.
  registerOnChange(fn: (value: string) => void): void {
    this.onChange = fn;
  }

  // Daftarin callback yang dipanggil onClose flatpickr (nandain field udah "disentuh").
  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  // Dipanggil Angular forms saat FormControl di-disable/enable (ex: mode read-only).
  setDisabledState(isDisabled: boolean): void {
    this.disabled = isDisabled;
    this.fp?.set('clickOpens', !isDisabled);
  }

  // Format Date bawaan flatpickr jadi string "YYYY-MM" sesuai kontrak API BE.
  private toYearMonth(date: Date): string {
    const year = date.getFullYear();
    const month = `${date.getMonth() + 1}`.padStart(2, '0');
    return `${year}-${month}`;
  }

  // Ganti input angka bawaan flatpickr buat tahun jadi readonly + pasang klik buat buka dropdown
  // custom kita sendiri (list bisa di-scroll/di-search), bukan spinner atas-bawah bawaan.
  private setupYearPicker(): void {
    const yearInput = this.fp!.currentYearElement;
    yearInput.readOnly = true;
    yearInput.addEventListener('click', (e) => {
      e.stopPropagation();
      this.toggleYearDropdown();
    });
  }

  // Toggle dropdown tahun custom: tutup kalau lagi kebuka, buka kalau lagi ketutup.
  private toggleYearDropdown(): void {
    if (this.yearDropdownEl) {
      this.closeYearDropdown();
    } else {
      this.openYearDropdown();
    }
  }

  // Bangun & tampilkan dropdown tahun custom (search + list scrollable) di dalam calendarContainer flatpickr.
  private openYearDropdown(): void {
    const fp = this.fp!;
    const yearInput = fp.currentYearElement;

    const dropdown = document.createElement('div');
    dropdown.className = 'month-picker__year-dropdown';

    const search = document.createElement('input');
    search.type = 'text';
    search.inputMode = 'numeric';
    search.className = 'month-picker__year-search';
    search.placeholder = 'Cari tahun...';
    search.addEventListener('click', (e) => e.stopPropagation());

    const list = document.createElement('div');
    list.className = 'month-picker__year-list';

    // Rentang tahun: 10 tahun ke depan s/d 100 tahun ke belakang dari tahun yang lagi ditampilkan
    // di kalender — cukup lebar buat kebutuhan riwayat kerja/pendidikan.
    const baseYear = fp.currentYear;
    const years: number[] = [];
    for (let y = baseYear + 10; y >= baseYear - 100; y--) years.push(y);

    // Render ulang daftar tahun sesuai teks pencarian, highlight tahun yang lagi aktif.
    const renderList = (filter: string): void => {
      list.innerHTML = '';
      years
        .filter((y) => String(y).includes(filter))
        .forEach((y) => {
          const item = document.createElement('button');
          item.type = 'button';
          item.className = 'month-picker__year-item';
          if (y === fp.currentYear) item.classList.add('selected');
          item.textContent = String(y);
          item.addEventListener('click', () => this.selectYear(y));
          list.appendChild(item);
        });
    };
    renderList('');
    search.addEventListener('input', () => renderList(search.value.trim()));

    dropdown.appendChild(search);
    dropdown.appendChild(list);

    // Ditempel langsung ke calendarContainer (bukan ke wrapper input tahun) biar gak ke-clip
    // sama overflow:hidden bawaan flatpickr di header bulan, dan biar posisinya di atas
    // (bukan di belakang) grid bulan — DOM order terakhir = paling atas kalau z-index sama-sama auto.
    // Posisi dihitung manual dari koordinat layar input tahun relatif ke calendarContainer.
    const calendarRect = fp.calendarContainer.getBoundingClientRect();
    const inputRect = yearInput.getBoundingClientRect();
    dropdown.style.position = 'absolute';
    dropdown.style.top = `${inputRect.bottom - calendarRect.top + 4}px`;
    dropdown.style.left = `${inputRect.left - calendarRect.left + inputRect.width / 2}px`;
    fp.calendarContainer.appendChild(dropdown);
    this.yearDropdownEl = dropdown;

    // Auto-scroll ke tahun yang lagi aktif biar gak perlu manual cari dulu.
    list.querySelector('.selected')?.scrollIntoView({ block: 'center' });
    search.focus();

    // setTimeout biar klik yang barusan buka dropdown ini gak langsung ke-capture sebagai "klik luar".
    setTimeout(() => document.addEventListener('click', this.outsideClickHandler));
  }

  // Lepas dropdown tahun custom dari DOM & bersihin listener klik-di-luar.
  private closeYearDropdown(): void {
    this.yearDropdownEl?.remove();
    this.yearDropdownEl = null;
    document.removeEventListener('click', this.outsideClickHandler);
  }

  // Pindah ke tahun yang dipilih. changeYear doang gak nge-refresh grid bulan punya plugin
  // monthSelect (itu cuma reachable lewat klik tombol prev/next-nya) — jadi triknya: geser ke
  // (tahun-1) dulu, lalu simulasikan klik "next" biar plugin rebuild grid dengan cara normalnya.
  private selectYear(year: number): void {
    const fp = this.fp!;
    fp.changeYear(year - 1);
    fp.nextMonthNav.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));
    this.closeYearDropdown();
  }
}
