package export

import (
	"strings"

	"github.com/go-pdf/fpdf"
)

// SanitizePDFText font Arial bawaan fpdf cuma support Latin-1, jadi karakter di luar itu (emoji,
// dsb) dibuang biar Output() gak gagal/ngerender kotak aneh.
func SanitizePDFText(text string) string {
	var b strings.Builder
	for _, r := range text {
		if r <= 255 {
			b.WriteRune(r)
		} else {
			b.WriteRune('?')
		}
	}
	return b.String()
}

// WritePDFKeyValueRows render daftar pasangan label-value sebagai baris teks sederhana (ex:
// ringkasan KPI 1 kolom label + 1 kolom value), dipakai buat section yang bukan tabel beneran.
func WritePDFKeyValueRows(pdf *fpdf.Fpdf, rows [][2]string) {
	for _, row := range rows {
		pdf.CellFormat(60, 6, SanitizePDFText(row[0]), "", 0, "L", false, 0, "")
		pdf.CellFormat(0, 6, SanitizePDFText(row[1]), "", 1, "L", false, 0, "")
	}
}

// WritePDFTable render tabel bergaris sederhana: header bold + baris data, sesuai definisi
// columns. colWidths (satuan mm) panjangnya harus sama persis kayak columns.
func WritePDFTable[T any](pdf *fpdf.Fpdf, columns []Column[T], colWidths []float64, rows []T) {
	pdf.SetFont("Arial", "B", 10)
	for i, column := range columns {
		pdf.CellFormat(colWidths[i], 7, SanitizePDFText(column.Header), "1", 0, "L", false, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Arial", "", 10)
	for _, row := range rows {
		for i, column := range columns {
			pdf.CellFormat(colWidths[i], 6, SanitizePDFText(column.Value(row)), "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}
}
