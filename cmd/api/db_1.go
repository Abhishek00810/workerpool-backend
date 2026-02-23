package main

import (
	"database/sql"
	"fmt"
	"log"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "password" // Matches the Docker command
	dbname   = "postgres" // Default database created by Docker
)

func Pg_Conn() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)

	if err != nil {
		log.Fatalf("failed to open the DB")
	}

	defer db.Close()

	// 3. Ping the DB to prove the connection is alive
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping DB (Is Docker running?):", err)
	}

	fmt.Println("Successfully connected to Postgres!")
}
