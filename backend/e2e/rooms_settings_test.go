package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/zoff-music/vibes/vibe"
)

func TestCreateRoomWithSettings(t *testing.T) {
	env := setupEnv(t)
	defer env.Teardown()

	// Create Room with custom settings
	settings := vibe.DefaultRoomSettings()
	settings.RemoveOnPlay = false
	settings.LoopQueue = true
	settings.SkipAllowed = false

	createPayload := map[string]interface{}{
		"name":     "Settings Test Room",
		"settings": settings,
	}
	body, _ := json.Marshal(createPayload)
	resp, err := http.Post(env.ServerURL+"/api/v1/rooms", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create room: %v", err)
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

	// Verify in response
	if result.Settings.RemoveOnPlay != false {
		t.Error("expected RemoveOnPlay to be false")
	}
	if result.Settings.LoopQueue != true {
		t.Error("expected LoopQueue to be true")
	}
	if result.Settings.SkipAllowed != false {
		t.Error("expected SkipAllowed to be false")
	}

	// Double check by GETting the room
	resp, err = http.Get(env.ServerURL + "/api/v1/rooms/" + result.ID)
	if err != nil {
		t.Fatalf("failed to get room: %v", err)
	}
	defer resp.Body.Close()

	var getResult vibe.Room
	err = json.NewDecoder(resp.Body).Decode(&getResult)
	if err != nil {
		t.Fatalf("failed to decode GET response: %v", err)
	}

	if getResult.Settings.RemoveOnPlay != false {
		t.Error("GET: expected RemoveOnPlay to be false")
	}
	if getResult.Settings.LoopQueue != true {
		t.Error("GET: expected LoopQueue to be true")
	}
}
