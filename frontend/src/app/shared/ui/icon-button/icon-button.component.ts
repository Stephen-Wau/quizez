import { Component, ElementRef, HostListener, Input, OnDestroy, Renderer2 } from '@angular/core';
import { LucideAngularModule } from 'lucide-angular';

export type IconButtonVariant = 'ghost' | 'danger' | 'primary';
export type IconButtonSize = 'sm' | 'md';

// Tombol aksi icon-only reusable, dipakai di kolom action semua <app-data-table> (Quiz, Bank Soal,
// Question & Answer, Summary, Collaboration & Permission, dst) supaya konsisten & gak makan tempat
// kayak tombol teks biasa.
@Component({
  selector: 'app-icon-button',
  standalone: true,
  imports: [LucideAngularModule],
  templateUrl: './icon-button.component.html',
  styleUrl: './icon-button.component.scss',
})
export class IconButtonComponent implements OnDestroy {
  @Input() icon = '';
  // Wajib diisi -- dipakai sebagai teks tooltip & aria-label (screen reader gak punya teks tombol).
  @Input() label = '';
  @Input() variant: IconButtonVariant = 'ghost';
  @Input() size: IconButtonSize = 'sm';
  @Input() disabled = false;

  private tooltipEl: HTMLElement | null = null;
  // Bound reference biar bisa di-remove lagi di hideTooltip (removeEventListener butuh reference yang sama).
  private readonly onScroll = () => this.hideTooltip();

  constructor(private host: ElementRef<HTMLElement>, private renderer: Renderer2) {}

  // Tooltip di-append ke <body> (bukan di dalam template component), karena ngx-datatable ngasih
  // overflow:hidden + transform ke tiap row buat keperluan virtual scroll -- tooltip yang di-posisikan
  // absolute/relative di dalam DOM row bakal ke-clip atau ketutup row lain. Nempel di body + posisi
  // dihitung dari getBoundingClientRect tombolnya sendiri biar selalu keliatan penuh di atas row manapun.
  @HostListener('mouseenter')
  @HostListener('focusin')
  showTooltip(): void {
    if (!this.label || this.tooltipEl || this.disabled) return;

    const button = this.host.nativeElement.querySelector('button');
    if (!button) return;
    const rect = button.getBoundingClientRect();

    const tooltip = this.renderer.createElement('span') as HTMLElement;
    this.renderer.appendChild(tooltip, this.renderer.createText(this.label));
    this.renderer.setStyle(tooltip, 'position', 'fixed');
    this.renderer.setStyle(tooltip, 'left', `${rect.left + rect.width / 2}px`);
    this.renderer.setStyle(tooltip, 'top', `${rect.top - 8}px`);
    this.renderer.setStyle(tooltip, 'transform', 'translate(-50%, -90%)');
    this.renderer.setStyle(tooltip, 'background', '#17325c');
    this.renderer.setStyle(tooltip, 'color', '#fff');
    this.renderer.setStyle(tooltip, 'font-size', '0.72rem');
    this.renderer.setStyle(tooltip, 'font-weight', '600');
    this.renderer.setStyle(tooltip, 'line-height', '1');
    this.renderer.setStyle(tooltip, 'padding', '5px 9px');
    this.renderer.setStyle(tooltip, 'border-radius', '6px');
    this.renderer.setStyle(tooltip, 'white-space', 'nowrap');
    this.renderer.setStyle(tooltip, 'z-index', '9999');
    this.renderer.setStyle(tooltip, 'pointer-events', 'none');
    this.renderer.setStyle(tooltip, 'opacity', '0');
    this.renderer.setStyle(tooltip, 'transition', 'opacity 0.12s ease, transform 0.12s ease');
    this.renderer.appendChild(document.body, tooltip);
    this.tooltipEl = tooltip;
    // Capture phase biar ke-tangkep juga scroll di dalam container internal (ex: .app-data-table__scroll
    // yang overflow-x:auto) -- scroll event elemen non-window gak bubble ke window secara default.
    document.addEventListener('scroll', this.onScroll, true);

    // Trigger di frame berikutnya biar transition opacity/transform-nya kepakai (bukan langsung full state).
    requestAnimationFrame(() => {
      if (!this.tooltipEl) return;
      this.renderer.setStyle(this.tooltipEl, 'opacity', '1');
      this.renderer.setStyle(this.tooltipEl, 'transform', 'translate(-50%, -100%)');
    });
  }

  @HostListener('mouseleave')
  @HostListener('focusout')
  hideTooltip(): void {
    if (this.tooltipEl) {
      this.renderer.removeChild(document.body, this.tooltipEl);
      this.tooltipEl = null;
    }
    document.removeEventListener('scroll', this.onScroll, true);
  }

  ngOnDestroy(): void {
    this.hideTooltip();
  }
}
