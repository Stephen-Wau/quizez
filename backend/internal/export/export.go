// Package export nampung helper generik buat download/export data tabular (CSV/Excel/PDF).
// Dipakai lintas menu (bukan cuma Analytics) -- resource lain yang butuh export tinggal
// definisiin Column-nya sendiri, gak perlu nulis ulang logic tulis file.
package export

// Column definisi 1 kolom export: Header nama kolom yang ditampilin (bebas custom, dinamis
// sesuai kebutuhan caller), Value cara ambil isi kolom itu dari 1 baris data T.
type Column[T any] struct {
	Header string
	Value  func(row T) string
}
