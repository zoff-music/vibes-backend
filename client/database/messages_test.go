package database

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMessageUsage(t *testing.T) {
	url := os.Getenv("VIBES_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set VIBES_TEST_DATABASE_URL to an isolated local database")
	}
	if !strings.Contains(url, "@127.0.0.1:") {
		t.Fatal("test database must be local")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	client := &Client{DB: db}
	err = client.prepareCreateMessageUsageStmt()
	if err != nil {
		t.Fatal(err)
	}
	defer client.CreateMessageUsageStatement.Close()
	err = client.prepareListAdminMessageUsageStmt()
	if err != nil {
		t.Fatal(err)
	}
	defer client.ListAdminMessageUsageStatement.Close()
	_, err = db.Exec("TRUNCATE chat_usage")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for _, room := range []string{"electro", "ambient"} {
		for _, at := range []time.Time{now, now, now.Add(-48 * time.Hour)} {
			err = client.CreateMessageUsage(context.Background(), room, at)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	tests := []struct {
		name, room    string
		total, recent int
	}{
		{name: "overall", total: 6, recent: 4},
		{name: "per room", room: "electro", total: 3, recent: 2},
		{name: "unknown room", room: "missing", total: 0, recent: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage, err := client.ListAdminMessageUsage(context.Background(), tt.room)
			if err != nil {
				t.Fatal(err)
			}
			if usage.Total != tt.total {
				t.Fatalf("total %d, want %d", usage.Total, tt.total)
			}
			totals := map[string]int{}
			for _, point := range usage.Points {
				totals[point.Window] += point.Messages
			}
			if totals["hour"] != tt.recent || totals["day"] != tt.total || totals["month"] != tt.total {
				t.Fatalf("wrong bucket totals: %v", totals)
			}
		})
	}
}
