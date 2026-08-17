package export

import (
	"encoding/csv"
	"io"
)

// WriteCSV tulis rows ke writer dalam format CSV sesuai definisi columns -- nama & urutan
// kolom di file mengikuti persis apa yang dikasih caller lewat columns.
func WriteCSV[T any](w io.Writer, columns []Column[T], rows []T) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.Header
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	for _, row := range rows {
		record := make([]string, len(columns))
		for i, col := range columns {
			record[i] = col.Value(row)
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return nil
}
