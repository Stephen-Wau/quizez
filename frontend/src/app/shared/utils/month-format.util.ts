const MONTH_LABELS = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];

// Format "YYYY-MM" (kontrak API BE) jadi label ringkas "Mmm YYYY" (ex: "2024-03" -> "Mar 2024").
export function formatMonth(yyyymm: string): string {
  const [year, month] = yyyymm.split('-').map(Number);
  return `${MONTH_LABELS[month - 1]} ${year}`;
}

// Format rentang tanggal mulai-selesai jadi label "Mmm YYYY - Mmm YYYY"; endDate null berarti
// masih berlangsung sampai sekarang, jadi ditampilkan "Sekarang" (ex: Work Histories/Education aktif).
export function formatPeriod(startDate: string, endDate: string | null): string {
  const start = formatMonth(startDate);
  const end = endDate ? formatMonth(endDate) : 'Sekarang';
  return `${start} - ${end}`;
}
