package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/acme/autocert"

	"github.com/vega-trello/trello-back/internal/auth"
	"github.com/vega-trello/trello-back/internal/handler"
	"github.com/vega-trello/trello-back/internal/repository"
	"github.com/vega-trello/trello-back/internal/router"
	"github.com/vega-trello/trello-back/internal/service"
)

func main() {
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
		vegaSSOURL = "https://vegastage.ru/authservice.php"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	tlsCertFile := os.Getenv("TLS_CERT_FILE")
	tlsKeyFile := os.Getenv("TLS_KEY_FILE")
	autocertDomain := os.Getenv("TLS_AUTOCERT_DOMAIN")

	useTLS := tlsCertFile != "" && tlsKeyFile != ""
	useAutocert := autocertDomain != ""

	if useTLS && useAutocert {
		log.Fatal("Cannot use both TLS_CERT_FILE and TLS_AUTOCERT_DOMAIN. Choose one.")
	}

	protocol := "HTTP"
	if useTLS || useAutocert {
		protocol = "HTTPS"
	}
	log.Printf("Starting server on %s :%s", protocol, port)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
	jwtManager := auth.NewJWTManager(jwtSecret, 24*time.Hour)
	userHandler := handler.NewUserHandler(userService, jwtManager, vegaSSOURL)

	projectRepo := repository.NewProjectRepository(pool)
	projectService := service.NewProjectService(projectRepo)
	projectHandler := handler.NewProjectHandler(projectService)

	columnRepo := repository.NewColumnRepository(pool)
	columnService := service.NewColumnService(columnRepo)
	columnHandler := handler.NewColumnHandler(columnService)

	taskRepo := repository.NewTaskRepository(pool)
	taskService := service.NewTaskService(taskRepo)
	taskHandler := handler.NewTaskHandler(taskService)

	memberRepo := repository.NewMemberRepository(pool)
	memberService := service.NewMemberService(memberRepo)
	memberHandler := handler.NewMemberHandler(memberService)

	r := router.SetupRouter(userHandler, projectHandler, columnHandler, taskHandler, memberHandler, jwtManager)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	if useAutocert {
		log.Printf("Using Let's Encrypt autocert for domain: %s", autocertDomain)

		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(autocertDomain),
			Cache:      autocert.DirCache("./certs"),
		}

		srv.TLSConfig = &tls.Config{
			GetCertificate: m.GetCertificate,
			MinVersion:     tls.VersionTLS12,
		}

		go func() {
			log.Printf("Starting HTTP redirect server on :80")
			if err := http.ListenAndServe(":80", m.HTTPHandler(nil)); err != nil {
				log.Printf("⚠ HTTP redirect server error: %v", err)
			}
		}()

		srv.Addr = ":443"
	}

	go func() {
		var serveErr error

		switch {
		case useAutocert:
			serveErr = srv.ListenAndServeTLS("", "")
		case useTLS:
			log.Printf("Using TLS certificates: %s, %s", tlsCertFile, tlsKeyFile)
			serveErr = srv.ListenAndServeTLS(tlsCertFile, tlsKeyFile)
		default:
			serveErr = srv.ListenAndServe()
		}

		if serveErr != nil && serveErr != http.ErrServerClosed {
			log.Fatalf("Server crashed: %v", serveErr)
		}
	}()

	log.Printf("Listening on %s://localhost:%s",
		map[bool]string{true: "https", false: "http"}[useTLS || useAutocert],
		map[bool]string{true: "443", false: port}[useAutocert])

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("⚠ Shutting down server gracefully...")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited successfully")
}
