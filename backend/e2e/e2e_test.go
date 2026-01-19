package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestHealthz(t *testing.T) {
	env := setupEnv(t)
	defer env.Teardown()

	resp, err := http.Get(env.ServerURL + "/_healthz")
	if err != nil {
		t.Fatalf("failed to get healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestCreateAndGetRoom(t *testing.T) {
	env := setupEnv(t)
	defer env.Teardown()

	// Create Room
	createPayload := map[string]string{"name": "E2E Test Room"}
	body, _ := json.Marshal(createPayload)
	resp, err := http.Post(env.ServerURL+"/api/v1/rooms", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create room: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Errorf("expected 201 Created or 200 OK, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	roomID, ok := result["id"].(string)
	if !ok {
		t.Fatalf("response does not contain room id. got: %v", result)
	}

	// Verify in DB directly
	var dbName string
	err = env.DB.QueryRow("SELECT name FROM rooms WHERE id = ?", roomID).Scan(&dbName)
	if err != nil {
		t.Fatalf("failed to find room in db: %v", err)
	}
	if dbName != "E2E Test Room" {
		t.Errorf("db room name mismatch: expected 'E2E Test Room', got '%s'", dbName)
	}

	// Get Room via API
	resp, err = http.Get(env.ServerURL + "/api/v1/rooms/" + roomID)
	if err != nil {
		t.Fatalf("failed to get room: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	var roomResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&roomResult); err != nil {
		t.Fatalf("failed to decode room response: %v", err)
	}

	if roomResult["name"] != "E2E Test Room" {
		t.Errorf("expected room name 'E2E Test Room', got %v", roomResult["name"])
	}
}
