package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vega-trello/trello-back/internal/auth"
	"github.com/vega-trello/trello-back/internal/handler"
	"github.com/vega-trello/trello-back/internal/repository"
	"github.com/vega-trello/trello-back/internal/router"
	"github.com/vega-trello/trello-back/internal/service"
)

func main() {

	// В будущем вынести в отдельный пакет config/ с использованием viper/env
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("Environment variable DATABASE_URL is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("Environment variable JWT_SECRET is required")
	}

	vegaSSOURL := os.Getenv("VEGA_SSO_URL")
	if vegaSSOURL == "" {
		// Дефолтное значение для разработки
		vegaSSOURL = "https://vegastage.ru/authservice.php"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s", port)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	userRepo := repository.NewUserRepository(pool)
	userService := service.NewUserService(userRepo)

	// Токен живет 24 часа (можно вынести в конфиг)
	jwtManager := auth.NewJWTManager(jwtSecret, 24*time.Hour)

	userHandler := handler.NewUserHandler(
		userService,
		jwtManager,
		vegaSSOURL,
	)

	r := router.SetupRouter(userHandler, jwtManager)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Запускаем сервер в горутине, чтобы не блокировать основной поток
	go func() {
		log.Printf("Listening on http://localhost:%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server crashed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server gracefully...")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
