package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"quizez/backend/internal/auth"
	"quizez/backend/internal/config"
	"quizez/backend/internal/db"
	"quizez/backend/internal/handlers"
)

// withCORS bungkus handler biar semua response ada header CORS-nya, sekalian short-circuit
// preflight request (OPTIONS) dari browser sebelum sampai ke handler asli.
func withCORS(frontendOrigin string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", frontendOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Browser kirim OPTIONS dulu buat preflight check, gak perlu diteruskan ke handler asli.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

// main nyiapin config, koneksi DB, dan daftar semua route sebelum server jalan di :8080.
func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	conn, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("/health", withCORS(cfg.FrontendOrigin, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	mux.HandleFunc("/api/auth/login", withCORS(cfg.FrontendOrigin, handlers.LoginHandler(conn, cfg.JWTSecret, cfg.JWTExpiryHours)))
	mux.HandleFunc("/api/auth/me", withCORS(cfg.FrontendOrigin, auth.RequireAuth(cfg.JWTSecret, handlers.MeHandler(conn))))

	mux.HandleFunc("/api/quizzes", withCORS(cfg.FrontendOrigin, auth.RequireAuth(cfg.JWTSecret, handlers.QuizzesHandler(conn))))
	mux.HandleFunc("/api/quizzes/{id}", withCORS(cfg.FrontendOrigin, auth.RequireAuth(cfg.JWTSecret, handlers.QuizHandler(conn))))
	mux.HandleFunc("/api/questions", withCORS(cfg.FrontendOrigin, auth.RequireAuth(cfg.JWTSecret, handlers.QuestionsHandler(conn))))
	mux.HandleFunc("/api/questions/{id}", withCORS(cfg.FrontendOrigin, auth.RequireAuth(cfg.JWTSecret, handlers.QuestionHandler(conn))))

	log.Println("quizez backend listening on localhost:8080")
	log.Fatal(http.ListenAndServe("localhost:8080", mux))
}
