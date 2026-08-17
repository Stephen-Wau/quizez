import { Component, Input } from '@angular/core';

export type CardVariant = 'default' | 'indigo' | 'emerald' | 'amber' | 'rose' | 'danger';

// AppCardComponent wadah visual buat tile info/stat kecil (ex: "Periode", "Total Question", KPI
// analytics) -- elevated + gradient tipis biar kerasa timbul. BUKAN buat card besar pembungkus
// section (pakai class ".xxx__card" biasa buat itu, tetap flat).
@Component({
  selector: 'app-card',
  standalone: true,
  templateUrl: './card.component.html',
  styleUrl: './card.component.scss',
  host: {
    class: 'app-card',
    '[class.app-card--indigo]': "variant === 'indigo'",
    '[class.app-card--emerald]': "variant === 'emerald'",
    '[class.app-card--amber]': "variant === 'amber'",
    '[class.app-card--rose]': "variant === 'rose'",
    '[class.app-card--danger]': "variant === 'danger'",
  },
})
export class CardComponent {
  // variant nentuin warna gradient aksen -- default cuma netral, sisanya nuansa warna soft.
  @Input() variant: CardVariant = 'default';
}
