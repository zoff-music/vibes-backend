package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/zoff-music/vibes/vibe"
)

func TestCreateRoomWithOnlyAdminAddSongs(t *testing.T) {
	env := setupEnv(t)
	defer env.Teardown()

	// 1. Try to create room with OnlyAdminAddSongs=true but NO password (should fail)
	settings := vibe.DefaultRoomSettings()
	settings.OnlyAdminAddSongs = true

	createPayload := map[string]interface{}{
		"name":     "Strict Room Fail",
		"settings": settings,
	}
	body, _ := json.Marshal(createPayload)
	resp, err := http.Post(env.ServerURL+"/api/v1/rooms", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to perform request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request when missing password, got %d", resp.StatusCode)
	}

	// 2. Try to create room with OnlyAdminAddSongs=true AND password (should success)
	createPayload["name"] = "Strict Room Success"
	createPayload["password"] = "admin123"

	body, _ = json.Marshal(createPayload)
	resp, err = http.Post(env.ServerURL+"/api/v1/rooms", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to perform request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201 Created, got %d", resp.StatusCode)
	}

	var result vibe.Room
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Settings.OnlyAdminAddSongs != true {
		t.Error("expected OnlyAdminAddSongs to be true")
	}
	if !result.HasPassword {
		t.Error("expected HasPassword to be true")
	}
}

func TestUpdateRoomSettingsOnlyAdminAddSongs(t *testing.T) {
	env := setupEnv(t)
	defer env.Teardown()

	// 1. Create a room without password
	createPayload := map[string]interface{}{
		"name": "Update Test Room",
	}
	body, _ := json.Marshal(createPayload)
	resp, err := http.Post(env.ServerURL+"/api/v1/rooms", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create room: %v", err)
	}

	var room vibe.Room
	json.NewDecoder(resp.Body).Decode(&room)
	resp.Body.Close()

	// 2. Try to enable OnlyAdminAddSongs (should fail because no password)
	settings := room.Settings
	settings.OnlyAdminAddSongs = true
	updatePayload := vibe.UpdateRoomRequest{
		Settings: &settings,
	}

	body, _ = json.Marshal(updatePayload)
	req, _ := http.NewRequest("PATCH", env.ServerURL+"/api/v1/rooms/"+room.ID+"/settings", bytes.NewReader(body))
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to perform request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request when enabling generic setting without password, got %d", resp.StatusCode)
	}
}
