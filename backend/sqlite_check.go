package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Test UUID blob generation and hex conversion
	var uuid string
	err = db.QueryRow("SELECT lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))").Scan(&uuid)
	if err != nil {
		log.Fatal("Failed to generate UUID:", err)
	}
	fmt.Println("Generated UUID:", uuid)

	// Test RETURNING clause
	_, err = db.Exec("CREATE TABLE test (id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))), val TEXT)")
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

	var id string
	err = db.QueryRow("INSERT INTO test (val) VALUES ('foo') RETURNING id").Scan(&id)
	if err != nil {
		log.Fatal("Failed to use RETURNING:", err)
	}
	fmt.Println("Returned ID:", id)
}
