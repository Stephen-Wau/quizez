// Package response menyeragamkan cara handler nulis balikan HTTP (sukses, error, list
// berpaginasi) biar format & status code konsisten di seluruh API tanpa ngulang
// json.NewEncoder/http.Error manual di tiap handler.
package response

import (
	"encoding/json"
	"net/http"

	"quizez/backend/internal/listquery"
)

// Pesan default per status code dipakai kalau handler manggil Error tanpa pesan custom,
// biar semua endpoint tetap konsisten walau lupa isi pesan spesifik.
var defaultMessages = map[int]string{
	http.StatusBadRequest:          "Permintaan tidak valid.",
	http.StatusUnauthorized:        "Silakan login terlebih dahulu.",
	http.StatusForbidden:           "Anda tidak punya akses untuk aksi ini.",
	http.StatusNotFound:            "Data tidak ditemukan.",
	http.StatusConflict:            "Data bentrok dengan yang sudah ada.",
	http.StatusMethodNotAllowed:    "Method tidak diizinkan.",
	http.StatusInternalServerError: "Terjadi kesalahan pada server.",
}

// JSON nulis data apapun sebagai body JSON dengan status code yang diminta.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Error nulis error sebagai plain text (bukan JSON) - sengaja disamain sama http.Error
// karena FE baca error message lewat `typeof err.error === 'string'` (HttpErrorResponse
// body apa adanya). Kalau message dikosongin, pakai pesan default sesuai status code.
func Error(w http.ResponseWriter, status int, message string) {
	if message == "" {
		if def, ok := defaultMessages[status]; ok {
			message = def
		} else {
			message = "Terjadi kesalahan."
		}
	}
	http.Error(w, message, status)
}

// Paginated bungkus response list yang butuh meta pagination (dipakai bareng
// internal/listquery), biar handler list gak perlu susun struct literal ListResponse manual.
func Paginated[T any](w http.ResponseWriter, items []T, meta listquery.Meta) {
	JSON(w, http.StatusOK, listquery.ListResponse[T]{Data: items, Meta: meta})
}
