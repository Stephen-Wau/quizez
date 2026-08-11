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
import { Instance } from 'flatpickr/dist/types/instance';

let nextId = 0;

// Picker jam/tanggal/tanggal-jam custom (pengganti <input type="time">/<input type="date">/
// <input type="datetime-local"> bawaan browser yang popup-nya beda-beda tiap OS dan defaultnya
// 12 jam AM/PM). Dipakai di form Quiz: mode="time" buat field jam-only (quiz), mode="datetime"
// buat field tanggal+jam (survey); mode="date" dipakai filter rentang tanggal (ex: Analytics).
// Selalu 24 jam, bisa diketik langsung atau dipilih dari popup. Implement ControlValueAccessor
// biar langsung bisa dipasang formControlName. Value yang di-emit ke form:
// - mode="time": "HH:mm"
// - mode="date": "YYYY-MM-DD"
// - mode="datetime": "YYYY-MM-DDTHH:mm"
@Component({
  selector: 'app-datetime-picker',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './datetime-picker.component.html',
  styleUrl: './datetime-picker.component.scss',
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => DatetimePickerComponent),
      multi: true,
    },
  ],
})
export class DatetimePickerComponent implements ControlValueAccessor, AfterViewInit, OnDestroy {
  @ViewChild('inputEl') inputEl!: ElementRef<HTMLInputElement>;

  @Input() label = '';
  @Input() mode: 'time' | 'date' | 'datetime' = 'datetime';
  // Nampilin tanda "*" merah di sebelah label, dipakai buat field yang wajib diisi.
  @Input() required = false;
  @Input() errorMessage = '';
  @Input() invalid = false;

  id = `app-datetime-picker-${nextId++}`;
  disabled = false;

  private fp?: Instance;
  private value = '';
  private onChange: (value: string) => void = () => {};
  private onTouched: () => void = () => {};

  // Init flatpickr setelah view siap (butuh elemen <input> beneran buat di-attach), dengan opsi
  // beda tergantung mode: time-only (noCalendar), date-only (tanpa time), atau tanggal+jam sekaligus.
  ngAfterViewInit(): void {
    const mode = this.mode;

    this.fp = flatpickr(this.inputEl.nativeElement, {
      enableTime: mode !== 'date',
      noCalendar: mode === 'time',
      time_24hr: true,
      // dateFormat cuma dipakai internal flatpickr buat parsing input yang diketik manual —
      // konversi ke format kontrak form ("HH:mm"/"YYYY-MM-DD"/"YYYY-MM-DDTHH:mm") dikerjakan
      // sendiri di onChange lewat objek Date, jadi gak gantung ke token format flatpickr.
      dateFormat: mode === 'time' ? 'H:i' : mode === 'date' ? 'Y-m-d' : 'Y-m-d H:i',
      allowInput: true,
      disableMobile: true,
      defaultDate: this.value ? this.toDate(this.value, mode) : undefined,
      onChange: (selectedDates) => {
        const date = selectedDates[0];
        const next = date ? this.formatValue(date, mode) : '';
        this.value = next;
        this.onChange(next);
      },
      onClose: () => this.onTouched(),
    });
    this.fp.set('clickOpens', !this.disabled);
  }

  ngOnDestroy(): void {
    this.fp?.destroy();
  }

  // Dipanggil Angular forms saat set value dari luar (ex: patchValue pas load data existing).
  writeValue(value: string): void {
    this.value = value ?? '';
    // fp belum ke-init kalau writeValue kepanggil sebelum ngAfterViewInit (ex: reset() di constructor).
    this.fp?.setDate(this.value ? this.toDate(this.value, this.mode) : '', false);
  }

  registerOnChange(fn: (value: string) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  // Dipanggil Angular forms saat FormControl di-disable/enable (ex: mode read-only).
  setDisabledState(isDisabled: boolean): void {
    this.disabled = isDisabled;
    this.fp?.set('clickOpens', !isDisabled);
  }

  // "HH:mm"/"YYYY-MM-DD"/"YYYY-MM-DDTHH:mm" (kontrak form) -> Date object buat di-load ke flatpickr.
  // Mode time-only dikasih tanggal hari ini sekadar placeholder (gak dipakai, cuma jam yang dibaca).
  private toDate(value: string, mode: 'time' | 'date' | 'datetime'): Date {
    if (mode === 'time') {
      const [h, m] = value.split(':').map(Number);
      const d = new Date();
      d.setHours(h, m, 0, 0);
      return d;
    }
    return new Date(value);
  }

  // Date object (dari flatpickr) -> string sesuai kontrak form: "HH:mm" (time), "YYYY-MM-DD"
  // (date), atau "YYYY-MM-DDTHH:mm" (datetime).
  private formatValue(date: Date, mode: 'time' | 'date' | 'datetime'): string {
    const pad = (n: number) => String(n).padStart(2, '0');
    const hh = pad(date.getHours());
    const mm = pad(date.getMinutes());
    if (mode === 'time') {
      return `${hh}:${mm}`;
    }
    const y = date.getFullYear();
    const m = pad(date.getMonth() + 1);
    const d = pad(date.getDate());
    if (mode === 'date') {
      return `${y}-${m}-${d}`;
    }
    return `${y}-${m}-${d}T${hh}:${mm}`;
  }
}
