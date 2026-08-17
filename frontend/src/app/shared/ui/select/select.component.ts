import { CommonModule } from '@angular/common';
import { Component, ElementRef, HostListener, Input, forwardRef } from '@angular/core';
import { ControlValueAccessor, NG_VALUE_ACCESSOR } from '@angular/forms';
import { LucideAngularModule } from 'lucide-angular';

export interface SelectOption {
  label: string;
  value: unknown;
}

let nextId = 0;

// Dropdown custom global buat semua form CMS, pengganti native <select> yang gak bisa di-styling
// bebas (browser beda-beda render dropdown-nya). Implement ControlValueAccessor supaya langsung
// bisa dipasang formControlName atau ngModel tanpa wiring tambahan.
@Component({
  selector: 'app-select',
  standalone: true,
  imports: [CommonModule, LucideAngularModule],
  templateUrl: './select.component.html',
  styleUrl: './select.component.scss',
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => SelectComponent),
      multi: true,
    },
  ],
})
export class SelectComponent implements ControlValueAccessor {
  @Input() label = '';
  @Input() options: SelectOption[] = [];
  @Input() placeholder = 'Pilih...';
  @Input() required = false;
  @Input() errorMessage = '';
  @Input() invalid = false;

  id = `app-select-${nextId++}`;
  value: unknown = null;
  disabled = false;
  isOpen = false;

  private onChange: (value: unknown) => void = () => {};
  private onTouched: () => void = () => {};

  constructor(private elementRef: ElementRef<HTMLElement>) {}

  // Label opsi yang lagi kepilih buat ditampilin di trigger; kosong kalau value belum match opsi manapun.
  get selectedLabel(): string {
    return this.options.find((o) => o.value === this.value)?.label ?? '';
  }

  writeValue(value: unknown): void {
    this.value = value;
  }

  registerOnChange(fn: (value: unknown) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  setDisabledState(isDisabled: boolean): void {
    this.disabled = isDisabled;
  }

  // Buka/tutup panel opsi, dipanggil dari klik tombol trigger.
  toggle(): void {
    if (this.disabled) return;
    this.isOpen = !this.isOpen;
    if (!this.isOpen) this.onTouched();
  }

  // Set value baru + kasih tau FormControl, dipanggil dari klik salah satu opsi di panel.
  selectOption(option: SelectOption): void {
    this.value = option.value;
    this.onChange(option.value);
    this.isOpen = false;
    this.onTouched();
  }

  isSelected(option: SelectOption): boolean {
    return option.value === this.value;
  }

  // trackBy by value (bukan index/reference) -- parent yang nyuplai [options] lewat getter/method
  // call di template bakal ngirim array+object baru tiap change-detection cycle. Tanpa trackBy ini,
  // *ngFor ngancurin & bikin ulang semua <li> tiap cycle, yang bisa bikin klik opsi "ilang" kalau
  // itu kejadian pas mousedown-mouseup lagi jalan (lihat juga fix serupa di analytics.component.ts).
  trackByValue(_: number, option: SelectOption): unknown {
    return option.value;
  }

  // Tutup panel kalau user klik di luar komponen ini (dropdown gak punya backdrop sendiri,
  // jadi listener di document yang jaga klik luar).
  @HostListener('document:click', ['$event'])
  handleDocumentClick(event: MouseEvent): void {
    if (!this.isOpen) return;
    if (!this.elementRef.nativeElement.contains(event.target as Node)) {
      this.isOpen = false;
      this.onTouched();
    }
  }

  // Tutup panel pas Escape ditekan, biar tetap keyboard-friendly walau bukan native select.
  @HostListener('document:keydown.escape')
  handleEscape(): void {
    this.isOpen = false;
  }
}
