package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "data/vibes.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT sql FROM sqlite_master WHERE type='table' AND name='room_users'")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	if rows.Next() {
		var sqlStmt string
		rows.Scan(&sqlStmt)
		fmt.Println("Schema for room_users:")
		fmt.Println(sqlStmt)
	} else {
		fmt.Println("Table room_users not found")
	}
}
