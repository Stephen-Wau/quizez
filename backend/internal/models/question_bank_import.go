package models

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// questionBankImportMaxOptions batas jumlah kolom opsi jawaban yang didukung file import
// (option_1..option_6), cukup buat semua tipe soal yang didukung (checkbox/rating paling banyak opsi).
const questionBankImportMaxOptions = 6

// Tipe soal yang didukung bank soal & import (matrix sengaja gak didukung format flat file ini).
const (
	questionTypeMultipleChoice = "multiple_choice"
	questionTypeDropdown       = "dropdown"
	questionTypeCheckbox       = "checkbox"
	questionTypeRating         = "rating"
	questionTypeFreeText       = "free_text"
)

// QuestionBankImportHeader urutan kolom baku file import/template CSV & XLSX.
var QuestionBankImportHeader = buildQuestionBankImportHeader()

func buildQuestionBankImportHeader() []string {
	header := []string{"question", "type_answer", "point", "tags"}
	for i := 1; i <= questionBankImportMaxOptions; i++ {
		header = append(header, fmt.Sprintf("option_%d_label", i), fmt.Sprintf("option_%d_value", i))
	}
	return header
}

// QuestionBankImportRowError 1 baris import yang gagal divalidasi, dilaporkan balik ke admin
// biar tau baris mana yang perlu dibetulin (import tetap lanjut buat baris lain yang valid).
type QuestionBankImportRowError struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// ParseQuestionBankImportCSV parse file CSV import, baris pertama dianggap header (dilewati).
func ParseQuestionBankImportCSV(data []byte) ([]QuestionBankItem, []QuestionBankImportRowError, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	// Kolom opsi (option_5/6) boleh gak ada di semua baris, jadi jumlah kolom per baris gak dipaksa sama.
	reader.FieldsPerRecord = -1

	var rows [][]string
	first := true
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		if first {
			first = false
			continue
		}
		rows = append(rows, record)
	}
	items, errs := parseQuestionBankImportRows(rows)
	return items, errs, nil
}

// ParseQuestionBankImportXLSX parse file XLSX import (sheet pertama), baris pertama dianggap header.
func ParseQuestionBankImportXLSX(data []byte) ([]QuestionBankItem, []QuestionBankImportRowError, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	sheet := f.GetSheetName(0)
	allRows, err := f.GetRows(sheet)
	if err != nil {
		return nil, nil, err
	}
	if len(allRows) <= 1 {
		return []QuestionBankItem{}, nil, nil
	}
	items, errs := parseQuestionBankImportRows(allRows[1:])
	return items, errs, nil
}

// parseQuestionBankImportRows validasi tiap baris satu-satu; baris invalid dicatat sebagai error
// (nomor baris = index+2, karena baris ke-1 file adalah header) tapi gak menggagalkan baris lain.
func parseQuestionBankImportRows(rows [][]string) ([]QuestionBankItem, []QuestionBankImportRowError) {
	items := []QuestionBankItem{}
	errs := []QuestionBankImportRowError{}
	for i, row := range rows {
		if isImportRowEmpty(row) {
			continue
		}
		item, msg := rowToQuestionBankItem(row)
		if msg != "" {
			errs = append(errs, QuestionBankImportRowError{Row: i + 2, Message: msg})
			continue
		}
		items = append(items, item)
	}
	return items, errs
}

func importCell(row []string, idx int) string {
	if idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func isImportRowEmpty(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// rowToQuestionBankItem validasi 1 baris import sesuai rule yang sama dipakai form manual
// (minus matrix, yang sengaja gak didukung format flat file ini).
func rowToQuestionBankItem(row []string) (QuestionBankItem, string) {
	question := importCell(row, 0)
	typeAnswer := strings.ToLower(importCell(row, 1))
	pointRaw := importCell(row, 2)
	tagsRaw := importCell(row, 3)

	if question == "" {
		return QuestionBankItem{}, "Question wajib diisi."
	}
	switch typeAnswer {
	case questionTypeMultipleChoice, questionTypeDropdown, questionTypeCheckbox, questionTypeRating, questionTypeFreeText:
	default:
		return QuestionBankItem{}, "Type answer harus multiple_choice, dropdown, checkbox, rating, atau free_text."
	}

	var point *int
	if pointRaw != "" {
		n, err := strconv.Atoi(pointRaw)
		if err != nil || n <= 0 {
			return QuestionBankItem{}, "Point harus angka lebih dari 0 (atau dikosongkan)."
		}
		point = &n
	}

	tags := []string{}
	if tagsRaw != "" {
		for _, t := range strings.Split(tagsRaw, ";") {
			if t = strings.TrimSpace(t); t != "" {
				tags = append(tags, t)
			}
		}
	}

	answers := []QuestionAnswer{}
	for i := 0; i < questionBankImportMaxOptions; i++ {
		label := importCell(row, 4+i*2)
		value := importCell(row, 5+i*2)
		if label == "" && value == "" {
			continue
		}
		if label == "" || value == "" {
			return QuestionBankItem{}, fmt.Sprintf("Opsi ke-%d: label dan value harus diisi berpasangan.", i+1)
		}
		l, v := label, value
		answers = append(answers, QuestionAnswer{Label: &l, Value: &v})
	}

	if msg := validateQuestionBankImportAnswers(typeAnswer, answers); msg != "" {
		return QuestionBankItem{}, msg
	}

	q, ta := question, typeAnswer
	return QuestionBankItem{Question: &q, TypeAnswer: &ta, Point: point, Tags: tags, Answers: answers}, ""
}

// validateQuestionBankImportAnswers cek bentuk opsi jawaban sesuai tipe soal, sama seperti
// validasi form manual di question_handler.go tapi versi baris flat file.
func validateQuestionBankImportAnswers(typeAnswer string, answers []QuestionAnswer) string {
	switch typeAnswer {
	case questionTypeFreeText:
		if len(answers) > 0 {
			return "Free text tidak boleh punya opsi jawaban."
		}
	case questionTypeMultipleChoice, questionTypeDropdown, questionTypeCheckbox:
		if len(answers) < 2 {
			return "Minimal harus punya 2 opsi jawaban."
		}
		trueCount := 0
		for _, a := range answers {
			v := strings.ToLower(strings.TrimSpace(*a.Value))
			if v != "true" && v != "false" {
				return "Value opsi harus true atau false."
			}
			if v == "true" {
				trueCount++
			}
		}
		if trueCount == 0 {
			return "Minimal harus punya 1 opsi dengan value true."
		}
	case questionTypeRating:
		if len(answers) < 2 {
			return "Rating minimal harus punya rentang 1 sampai 2."
		}
		seen := map[int]bool{}
		maxRating := 0
		for _, a := range answers {
			n, err := strconv.Atoi(strings.TrimSpace(*a.Value))
			if err != nil || n < 1 || n > 10 {
				return "Value rating harus angka 1 sampai 10."
			}
			if seen[n] {
				return "Value rating tidak boleh duplikat."
			}
			seen[n] = true
			if n > maxRating {
				maxRating = n
			}
		}
		for i := 1; i <= maxRating; i++ {
			if !seen[i] {
				return "Rentang rating harus berurutan mulai dari 1."
			}
		}
	}
	return ""
}

// BuildQuestionBankTemplateCSV bikin file contoh CSV yang bisa didownload admin buat panduan
// format import (header + contoh soal realistis meliputi tiap tipe soal yang didukung).
func BuildQuestionBankTemplateCSV() []byte {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	writer.Write(QuestionBankImportHeader)
	for _, row := range questionBankTemplateRows() {
		writer.Write(row)
	}
	writer.Flush()
	return buf.Bytes()
}

// BuildQuestionBankTemplateXLSX bikin file contoh XLSX senada dengan versi CSV.
func BuildQuestionBankTemplateXLSX() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)

	for col, header := range QuestionBankImportHeader {
		cellRef, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cellRef, header)
	}
	for r, row := range questionBankTemplateRows() {
		for c, value := range row {
			cellRef, _ := excelize.CoordinatesToCellName(c+1, r+2)
			f.SetCellValue(sheet, cellRef, value)
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// questionBankTemplateRows contoh data realistis buat tiap tipe soal yang didukung import (bukan
// data dummy asal-asalan) biar admin langsung paham polanya pas buka file contoh.
func questionBankTemplateRows() [][]string {
	return [][]string{
		{"Apa ibu kota Indonesia?", "multiple_choice", "10", "geografi;pengetahuan-umum",
			"Jakarta", "true", "Surabaya", "false", "Bandung", "false", "", "", "", "", "", ""},
		{"Manakah yang termasuk bahasa pemrograman? (bisa pilih lebih dari 1)", "checkbox", "10", "teknologi;pemrograman",
			"Python", "true", "HTML", "false", "JavaScript", "true", "CSS", "false", "", "", "", ""},
		{"Search engine mana yang paling sering kamu pakai?", "dropdown", "", "kebiasaan-digital",
			"Google", "true", "Bing", "false", "DuckDuckGo", "false", "", "", "", "", "", ""},
		{"Seberapa puas kamu dengan layanan customer service kami?", "rating", "", "kepuasan-pelanggan",
			"1", "1", "2", "2", "3", "3", "4", "4", "5", "5", "", ""},
		{"Apa saran kamu untuk peningkatan layanan kami ke depannya?", "free_text", "", "feedback",
			"", "", "", "", "", "", "", "", "", "", "", ""},
	}
}
