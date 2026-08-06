import { AbstractControl, FormGroup } from '@angular/forms';

// Cek error validasi 1 field: cuma tampil kalau field itu udah disentuh (touched) DAN invalid
// (biar gak langsung muncul semua pesan error pas form baru dibuka). Dipakai semua form CMS
// (Login, Profile, Work Histories, Education, dst) biar pesan error konsisten.
//
// customMessage opsional buat kasus field yang butuh pesan spesifik (ex: validasi format email
// di Profile, atau nama field custom di Login) — kalau gak dikasih/balikin undefined, fallback ke
// pesan generik "Wajib diisi.".
export function fieldError(
  form: FormGroup,
  name: string,
  customMessage?: (control: AbstractControl) => string | undefined,
): string {
  const control = form.get(name);
  if (!control?.touched || !control.invalid) return '';
  return customMessage?.(control) ?? 'Wajib diisi.';
}
