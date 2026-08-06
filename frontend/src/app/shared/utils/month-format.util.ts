const MONTH_LABELS = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];

export function formatMonth(yyyymm: string): string {
  const [year, month] = yyyymm.split('-').map(Number);
  return `${MONTH_LABELS[month - 1]} ${year}`;
}

export function formatPeriod(startDate: string, endDate: string | null): string {
  const start = formatMonth(startDate);
  const end = endDate ? formatMonth(endDate) : 'Sekarang';
  return `${start} - ${end}`;
}
