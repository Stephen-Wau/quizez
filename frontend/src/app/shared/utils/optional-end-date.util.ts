import { FormGroup, Validators } from '@angular/forms';

// Pasang toggle "masih berlangsung" (masih bekerja / masih menempuh pendidikan) ke sebuah form:
// pas checkbox dicentang, field end_date di-kosongin & Validators.required-nya dilepas; pas gak
// dicentang, Validators.required dipasang lagi. Dipanggil sekali di constructor komponen (Work
// Histories, Education, dst — form mana pun yang punya pola "tanggal selesai opsional").
export function toggleOptionalEndDate(
  form: FormGroup,
  checkboxControlName: string,
  endDateControlName: string,
): void {
  form.get(checkboxControlName)!.valueChanges.subscribe((checked) => {
    const endDate = form.get(endDateControlName)!;
    if (checked) {
      endDate.clearValidators();
      endDate.setValue('');
    } else {
      endDate.setValidators(Validators.required);
    }
    endDate.updateValueAndValidity();
  });
}
