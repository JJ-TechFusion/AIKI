//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file:", err)
	}

	connString := os.Getenv("DB_URL")
	fmt.Println("DB_URL from .env:", connString)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		fmt.Println("ParseConfig error:", err)
		return
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		fmt.Println("NewWithConfig error:", err)
		return
	}
	defer pool.Close()

	startTime := time.Now()
	fmt.Println("Attempting to Ping database...")
	if err := pool.Ping(ctx); err != nil {
		fmt.Printf("Ping error after %v: %v\n", time.Since(startTime), err)
		return
	}
	fmt.Printf("Connected successfully in %v!\n", time.Since(startTime))
}
