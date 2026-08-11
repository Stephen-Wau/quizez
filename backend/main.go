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

// main nyiapin config, koneksi DB, dan daftar semua route sebelum server jalan di port aplikasi.
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
	mux.HandleFunc("/api/quizzes/{id}/share-link", withCORS(cfg.FrontendOrigin, auth.RequireAuth(cfg.JWTSecret, handlers.QuizShareHandler(conn))))
	mux.HandleFunc("/api/quizzes/{id}/summary", withCORS(cfg.FrontendOrigin, auth.RequireAuth(cfg.JWTSecret, handlers.QuizSummaryHandler(conn))))
	mux.HandleFunc("/api/quizzes/{id}/analytics", withCORS(cfg.FrontendOrigin, auth.RequireAuth(cfg.JWTSecret, handlers.QuizAnalyticsHandler(conn))))
	mux.HandleFunc("/api/quizzes/{id}/analytics/export", withCORS(cfg.FrontendOrigin, auth.RequireAuth(cfg.JWTSecret, handlers.QuizAnalyticsExportHandler(conn))))
	mux.HandleFunc("/api/quizzes/{id}/submissions/{submissionId}", withCORS(cfg.FrontendOrigin, auth.RequireAuth(cfg.JWTSecret, handlers.QuizSubmissionDetailHandler(conn))))
	mux.HandleFunc("/api/questions", withCORS(cfg.FrontendOrigin, auth.RequireAuth(cfg.JWTSecret, handlers.QuestionsHandler(conn))))
	mux.HandleFunc("/api/questions/{id}", withCORS(cfg.FrontendOrigin, auth.RequireAuth(cfg.JWTSecret, handlers.QuestionHandler(conn))))
	mux.HandleFunc("/api/public/quizzes/{token}", withCORS(cfg.FrontendOrigin, handlers.PublicQuizHandler(conn)))
	mux.HandleFunc("/api/public/quizzes/{token}/submit", withCORS(cfg.FrontendOrigin, handlers.PublicQuizSubmitHandler(conn)))

	addr := "127.0.0.1:" + cfg.AppPort
	log.Printf("quizez backend listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
