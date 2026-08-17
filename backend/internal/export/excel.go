package export

import (
	"github.com/xuri/excelize/v2"
)

// WriteXLSXSheet isi 1 sheet Excel dari rows sesuai definisi columns -- baris header (baris 1)
// otomatis dibikin bold, lebar kolom disesuaikan kasar dari panjang nama header-nya.
func WriteXLSXSheet[T any](f *excelize.File, sheet string, columns []Column[T], rows []T) error {
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	if err != nil {
		return err
	}

	for col, column := range columns {
		cell, err := excelize.CoordinatesToCellName(col+1, 1)
		if err != nil {
			return err
		}
		if err := f.SetCellValue(sheet, cell, column.Header); err != nil {
			return err
		}
		if err := f.SetCellStyle(sheet, cell, cell, headerStyle); err != nil {
			return err
		}

		colName, err := excelize.ColumnNumberToName(col + 1)
		if err != nil {
			return err
		}
		// Lebar kasar dari panjang header + padding, minimal 12 biar kolom angka pendek gak kepotong.
		width := float64(len(column.Header)) + 6
		if width < 12 {
			width = 12
		}
		if err := f.SetColWidth(sheet, colName, colName, width); err != nil {
			return err
		}
	}

	for rowIndex, row := range rows {
		for col, column := range columns {
			cell, err := excelize.CoordinatesToCellName(col+1, rowIndex+2)
			if err != nil {
				return err
			}
			if err := f.SetCellValue(sheet, cell, column.Value(row)); err != nil {
				return err
			}
		}
	}
	return nil
}

// NewXLSX bikin file Excel baru berisi 1 sheet data tabular. Buat file multi-sheet, bikin
// sheet tambahan manual (f.NewSheet) lalu panggil WriteXLSXSheet ke tiap sheet-nya.
func NewXLSX[T any](sheetName string, columns []Column[T], rows []T) (*excelize.File, error) {
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", sheetName)
	if err := WriteXLSXSheet(f, sheetName, columns, rows); err != nil {
		return nil, err
	}
	return f, nil
}
