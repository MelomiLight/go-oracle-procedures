package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"oracle-golang/internal/config"
	"oracle-golang/internal/database"
	"oracle-golang/internal/handler"
	"oracle-golang/internal/repository"
	"oracle-golang/internal/service"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	_ "github.com/sijms/go-ora/v2"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	cfg := config.NewConfig()
	conn, err := database.NewOracleDatabase().Connect(cfg.OracleDatabase.DSN())
	if err != nil {
		log.Fatal(err)
	}
	defer func(conn *sql.DB) {
		err := conn.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(conn)
	log.Println("Connected to Database")

	r := setupRouter(conn)

	server := &http.Server{
		Addr:         cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Println("Starting server on:", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Server failed to start", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Println("Server shutdown encountered an error", "error", err)
		return
	}

	log.Println("Server stopped")
}

func setupRouter(conn *sql.DB) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.Heartbeat("/health"))

	r.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Route("/procedures", func(r chi.Router) {
				oracleRepository := repository.NewOracleRepository(conn)
				procedureService := service.NewProcedureService(oracleRepository)
				procedureHandler := handler.NewProcedureHandler(procedureService)
				r.Post("/call", procedureHandler.CallProcedure)
				r.Get("/info", procedureHandler.GetProcedureInfo)
			})
		})
	})

	return r
}
