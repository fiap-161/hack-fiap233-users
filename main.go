package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/hack-fiap233/users/internal/handler"
	"github.com/hack-fiap233/users/internal/repository"
	"github.com/hack-fiap233/users/internal/service"
	_ "github.com/lib/pq"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	db := initDB()
	migrateDB(db)

	repo := repository.New().WithDB(db).Build()
	svc := service.New().WithRepository(repo).WithJWTSecret(jwtSecret).Build()
	h := handler.New().WithService(svc).Build()

	http.HandleFunc("/users/health", h.Health)
	http.HandleFunc("/users/register", h.Register)
	http.HandleFunc("/users/login", h.Login)
	http.HandleFunc("/users/", h.List)

	log.Printf("Users service listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func initDB() *sql.DB {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USERNAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to PostgreSQL")
	return db
}

func migrateDB(db *sql.DB) {
	query := `CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL
	)`
	if _, err := db.Exec(query); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
}
