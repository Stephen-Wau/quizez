import { Component, Input } from '@angular/core';

export type BadgeVariant = 'default' | 'indigo' | 'gray' | 'emerald' | 'amber' | 'rose' | 'danger';

// AppBadgeComponent label kecil buat nilai kategorikal di table/detail (ex: Type, Status, Role) --
// kotak dengan sudut dikit rounded (bukan pill) + gradient tipis, biar konsisten di seluruh app
// dan gak tiap halaman bikin style badge sendiri-sendiri.
@Component({
  selector: 'app-badge',
  standalone: true,
  templateUrl: './badge.component.html',
  styleUrl: './badge.component.scss',
  host: {
    class: 'app-badge',
    '[class.app-badge--indigo]': "variant === 'indigo'",
    '[class.app-badge--gray]': "variant === 'gray'",
    '[class.app-badge--emerald]': "variant === 'emerald'",
    '[class.app-badge--amber]': "variant === 'amber'",
    '[class.app-badge--rose]': "variant === 'rose'",
    '[class.app-badge--danger]': "variant === 'danger'",
  },
})
export class BadgeComponent {
  @Input() variant: BadgeVariant = 'default';
}
