//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")
	connString := os.Getenv("DB_URL")
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	var version int
	var dirty bool
	err = conn.QueryRow(ctx, "SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	if err != nil {
		fmt.Printf("schema_migrations table error/missing: %v\n", err)
	} else {
		fmt.Printf("Current schema_migrations: version=%d, dirty=%v\n", version, dirty)
	}

	// Try to fix dirty state by setting dirty to false and version to 7 if user_profile exists?
	// First let's just see what tables exist
	rows, err := conn.Query(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'")
	if err != nil {
		fmt.Printf("Error querying tables: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Println("Tables in DB:")
	for rows.Next() {
		var name string
		rows.Scan(&name)
		fmt.Println("- " + name)
	}

	// Check if user_profile has goals
	var goalsExists bool
	err = conn.QueryRow(ctx, "SELECT TRUE FROM information_schema.columns WHERE table_name='user_profile' AND column_name='goals'").Scan(&goalsExists)
	if err == nil && goalsExists {
		fmt.Println("user_profile already has 'goals' column")
	} else {
		fmt.Println("user_profile does NOT have 'goals' column")
	}
}
