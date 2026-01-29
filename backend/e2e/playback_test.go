package e2e

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/zoff-music/vibes/vibe"
)

func TestAutoPlayOnJoin(t *testing.T) {
	env := setupEnv(t)
	defer env.Teardown()

	// 1. Create Room
	roomID := createTestRoom(t, env, "AutoPlay Room")

	// 2. Add a song
	addTestSong(t, env, roomID, "Song 1")

	// 3. Connect to SSE
	resp, err := http.Get(env.ServerURL + "/api/v1/rooms/" + roomID + "/events")
	if err != nil {
		t.Fatalf("failed to connect to SSE: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	// 4. Read first few lines to find playback update
	scanner := bufio.NewScanner(resp.Body)
	foundPlaybackUpdate := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: playback_update") {
			if scanner.Scan() {
				dataLine := scanner.Text()
				if strings.HasPrefix(dataLine, "data: ") {
					data := strings.TrimPrefix(dataLine, "data: ")
					var state vibe.PlaybackState
					if err := json.Unmarshal([]byte(data), &state); err != nil {
						t.Fatalf("failed to unmarshal playback state: %v", err)
					}
					if state.IsPlaying {
						foundPlaybackUpdate = true
						break
					}
				}
			}
		}
		// Don't wait forever
	}

	if !foundPlaybackUpdate {
		t.Errorf("expected auto-play to start on join, but isPlaying is false or no update received")
	}
}

func TestLoopQueueAndRemoveOnPlay(t *testing.T) {
	env := setupEnv(t)
	defer env.Teardown()

	// 1. Create Room with LoopQueue=true, RemoveOnPlay=false
	roomID := createTestRoom(t, env, "Loop Room")
	updateRoomSettings(t, env, roomID, &vibe.RoomSettings{
		LoopQueue:      true,
		RemoveOnPlay:   false,
		SkipAllowed:    true,
		EnabledSources: []string{"youtube", "spotify", "soundcloud"},
	})

	// 2. Add two songs
	song1 := addTestSong(t, env, roomID, "Song 1")
	song2 := addTestSong(t, env, roomID, "Song 2")

	// 3. Start playback (joining will start it)
	_, _ = http.Get(env.ServerURL + "/api/v1/rooms/" + roomID + "/events")
	time.Sleep(100 * time.Millisecond)

	// Verify Song 1 is playing
	state := getPlaybackState(t, env, roomID)
	if state.CurrentSong == nil || state.CurrentSong.ID != song1.ID {
		t.Errorf("expected song 1 to be playing, got %v", getSongIDStr(state))
	}

	// 4. Skip to Song 2
	skipTrack(t, env, roomID)
	state = getPlaybackState(t, env, roomID)
	if state.CurrentSong == nil || state.CurrentSong.ID != song2.ID {
		t.Errorf("expected song 2 to be playing after skip, got %v", getSongIDStr(state))
	}

	// 5. Skip again - should loop back to Song 1
	skipTrack(t, env, roomID)
	state = getPlaybackState(t, env, roomID)
	if state.CurrentSong == nil || state.CurrentSong.ID != song1.ID {
		t.Errorf("expected song 1 to be playing after loop skip, got %v", getSongIDStr(state))
	}

	// 6. Enable RemoveOnPlay, Disable LoopQueue
	updateRoomSettings(t, env, roomID, &vibe.RoomSettings{
		LoopQueue:      false,
		RemoveOnPlay:   true,
		SkipAllowed:    true,
		EnabledSources: []string{"youtube", "spotify", "soundcloud"},
	})

	// 7. Skip Song 1 - it should be removed, Song 2 should play
	skipTrack(t, env, roomID)
	state = getPlaybackState(t, env, roomID)
	if state.CurrentSong == nil || state.CurrentSong.ID != song2.ID {
		t.Errorf("expected song 2 to be playing after removal skip, got %v", getSongIDStr(state))
	}

	// Verify Song 1 is gone from DB
	var count int
	env.DB.QueryRow("SELECT COUNT(*) FROM songs WHERE id = ?", song1.ID).Scan(&count)
	if count != 0 {
		t.Errorf("expected song 1 to be removed from DB, but it still exists")
	}

	// 8. Skip Song 2 - it should be removed, queue should be empty
	skipTrack(t, env, roomID)
	state = getPlaybackState(t, env, roomID)
	if state.CurrentSong != nil {
		t.Errorf("expected queue to be empty, but song %v is playing", state.CurrentSong.ID)
	}
	if state.IsPlaying {
		t.Errorf("expected playback to stop when queue is empty")
	}

	// Verify Song 2 is gone from DB
	env.DB.QueryRow("SELECT COUNT(*) FROM songs WHERE id = ?", song2.ID).Scan(&count)
	if count != 0 {
		t.Errorf("expected song 2 to be removed from DB, but it still exists")
	}
}

func getSongIDStr(state *vibe.PlaybackState) string {
	if state == nil || state.CurrentSong == nil {
		return "nil"
	}
	return state.CurrentSong.ID
}

// Helpers

func createTestRoom(t *testing.T, env *TestEnv, name string) string {
	payload := map[string]string{"name": name}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(env.ServerURL+"/api/v1/rooms", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create room: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result["id"].(string)
}

func addTestSong(t *testing.T, env *TestEnv, roomID, title string) *vibe.Song {
	payload := vibe.AddSongRequest{
		SourceType: vibe.SourceTypeYouTube,
		SourceID:   "vid-" + title,
		Title:      title,
		Duration:   180,
		Thumbnail:  "http://example.com/thumb.jpg",
	}
	body, _ := json.Marshal(payload)

	// We need a session to add a song
	req, _ := http.NewRequest(http.MethodPost, env.ServerURL+"/api/v1/rooms/"+roomID+"/sessions", bytes.NewReader([]byte(`{"nickname":"testuser"}`)))
	resp, _ := http.DefaultClient.Do(req)
	cookie := resp.Header.Get("Set-Cookie")
	resp.Body.Close()

	req, _ = http.NewRequest(http.MethodPost, env.ServerURL+"/api/v1/rooms/"+roomID+"/songs", bytes.NewReader(body))
	req.Header.Set("Cookie", cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to add song: %v", err)
	}
	defer resp.Body.Close()

	var song vibe.Song
	json.NewDecoder(resp.Body).Decode(&song)
	return &song
}

func updateRoomSettings(t *testing.T, env *TestEnv, roomID string, settings *vibe.RoomSettings) {
	payload := vibe.UpdateRoomRequest{
		Settings: settings,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPatch, env.ServerURL+"/api/v1/rooms/"+roomID+"/settings", bytes.NewReader(body))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to update settings: %v", err)
	}
	resp.Body.Close()
}

func getPlaybackState(t *testing.T, env *TestEnv, roomID string) *vibe.PlaybackState {
	resp, err := http.Get(env.ServerURL + "/api/v1/rooms/" + roomID + "/events")
	if err != nil {
		t.Fatalf("failed to get playback state: %v", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: playback_update") {
			if scanner.Scan() {
				data := strings.TrimPrefix(scanner.Text(), "data: ")
				var state vibe.PlaybackState
				json.Unmarshal([]byte(data), &state)
				return &state
			}
		}
	}
	return nil
}

func skipTrack(t *testing.T, env *TestEnv, roomID string) {
	// Create a session first to be host
	reqSession, _ := http.NewRequest(http.MethodPost, env.ServerURL+"/api/v1/rooms/"+roomID+"/sessions", bytes.NewReader([]byte(`{"nickname":"host"}`)))
	respSession, _ := http.DefaultClient.Do(reqSession)
	cookie := respSession.Header.Get("Set-Cookie")
	respSession.Body.Close()

	req, _ := http.NewRequest(http.MethodPost, env.ServerURL+"/api/v1/rooms/"+roomID+"/skips", nil)
	req.Header.Set("Cookie", cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to skip track: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for skip, got %d", resp.StatusCode)
	}
}
