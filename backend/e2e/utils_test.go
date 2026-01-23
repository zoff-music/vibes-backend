package e2e

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/zoff-music/vibes/config"
	"github.com/zoff-music/vibes/server"
)

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

type TestEnv struct {
	Server    *server.Server
	ServerURL string
	DB        *sql.DB
	DBPath    string
	Cancel    context.CancelFunc
}

func (e *TestEnv) Teardown() {
	e.Cancel()
	if e.DB != nil {
		e.DB.Close()
	}
	if e.DBPath != "" {
		os.Remove(e.DBPath)
	}
}

func setupEnv(t *testing.T) *TestEnv {
	port, err := getFreePort()
	if err != nil {
		if isBindNotPermitted(err) {
			t.Skipf("skipping e2e tests: cannot bind local port: %v", err)
		}
		t.Fatalf("failed to get free port: %v", err)
	}

	// Create temp DB file
	f, err := os.CreateTemp("", "vibes-e2e-*.db")
	if err != nil {
		t.Fatalf("failed to create temp db: %v", err)
	}
	dbPath := f.Name()
	f.Close()

	// Apply migrations
	err = applyMigrations(dbPath)
	if err != nil {
		os.Remove(dbPath)
		t.Fatalf("failed to apply migrations: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	cfg.Port = strconv.Itoa(port)
	cfg.DatabasePath = dbPath
	cfg.YouTubeAPIKey = "dummy-key" // Mock or expecting failures if used

	ctx, cancel := context.WithCancel(context.Background())

	srv := &server.Server{}
	err = srv.Create(ctx, cfg)
	if err != nil {
		cancel()
		os.Remove(dbPath)
		t.Fatalf("failed to create server: %v", err)
	}

	errc := make(chan error, 1)
	go srv.Serve(ctx, errc)

	// Wait for server to start (simple check)
	time.Sleep(100 * time.Millisecond)
	select {
	case err := <-errc:
		cancel()
		os.Remove(dbPath)
		t.Fatalf("server failed to start: %v", err)
	default:
	}

	// We also need a direct DB connection for assertions
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		cancel()
		os.Remove(dbPath)
		t.Fatalf("failed to open db verification connection: %v", err)
	}

	return &TestEnv{
		Server:    srv,
		ServerURL: fmt.Sprintf("http://localhost:%d", port),
		DB:        db,
		DBPath:    dbPath,
		Cancel:    cancel,
	}
}

func isBindNotPermitted(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, syscall.EPERM) {
		return true
	}
	if errors.Is(err, syscall.EACCES) {
		return true
	}
	if strings.Contains(err.Error(), "operation not permitted") {
		return true
	}
	if strings.Contains(err.Error(), "permission denied") {
		return true
	}
	return false
}

func applyMigrations(dbPath string) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Locate migrations directory
	// Since we run from backend/e2e, it is ../../migrator/migrations
	wd, _ := os.Getwd()
	// Check if we are running from root or package.
	// `go test ./backend/e2e` runs in backend/e2e.

	// Start looking for git root to be safe?
	// Or just relative path assuming typical go test execution.
	migrationDir := filepath.Join(wd, "../../migrator/migrations")

	// Fallback if running from root?
	if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
		// Maybe we are in root?
		migrationDir = "migrator/migrations"
	}

	if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory not found at %s", migrationDir)
	}

	files, err := os.ReadDir(migrationDir)
	if err != nil {
		return err
	}

	var upFiles []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".up.sql") {
			upFiles = append(upFiles, filepath.Join(migrationDir, f.Name()))
		}
	}
	sort.Strings(upFiles)

	for _, file := range upFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		_, err = db.Exec(string(content))
		if err != nil {
			return fmt.Errorf("failed to exec migration %s: %w", file, err)
		}
	}
	return nil
}
