import { Component, Input, forwardRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ControlValueAccessor, NG_VALUE_ACCESSOR } from '@angular/forms';

let nextId = 0;

// Input global buat semua form CMS. Implement ControlValueAccessor supaya langsung bisa
// dipasang formControlName tanpa wiring tambahan, ex: <app-input formControlName="username" label="Username" />.
@Component({
  selector: 'app-input',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './input.component.html',
  styleUrl: './input.component.scss',
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => InputComponent),
      multi: true,
    },
  ],
})
export class InputComponent implements ControlValueAccessor {
  @Input() label = '';
  @Input() type: 'text' | 'password' | 'email' | 'month' | 'time' | 'datetime-local' | 'number' = 'text';
  // Nampilin tanda "*" merah di sebelah label, dipakai buat field yang wajib diisi.
  @Input() required = false;
  @Input() errorMessage = '';
  // Paksa border error tanpa teks di bawah, dipakai buat error umum (ex: kredensial salah dari API).
  @Input() invalid = false;
  // Render <textarea> dibanding <input>, dipakai buat field teks panjang (ex: about me).
  @Input() multiline = false;
  @Input() rows = 3;

  id = `app-input-${nextId++}`;
  value = '';
  disabled = false;

  private onChange: (value: string) => void = () => {};
  private onTouched: () => void = () => {};

  // Dipanggil Angular forms saat set value dari luar (ex: patchValue pas load data existing).
  writeValue(value: string): void {
    this.value = value ?? '';
  }

  // Daftarin callback yang dipanggil handleInput tiap kali user ngetik.
  registerOnChange(fn: (value: string) => void): void {
    this.onChange = fn;
  }

  // Daftarin callback yang dipanggil handleBlur (nandain field udah "disentuh").
  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  // Dipanggil Angular forms saat FormControl di-disable/enable (ex: mode read-only).
  setDisabledState(isDisabled: boolean): void {
    this.disabled = isDisabled;
  }

  // Sinkronin value lokal + kasih tau FormControl tiap kali user ngetik di input/textarea.
  handleInput(value: string): void {
    this.value = value;
    this.onChange(value);
  }

  // Tandain field "touched" begitu kehilangan fokus, biar pesan validasi baru muncul setelahnya.
  handleBlur(): void {
    this.onTouched();
  }
}
