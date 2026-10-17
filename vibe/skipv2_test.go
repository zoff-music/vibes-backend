package vibe

import "testing"

func TestRequiresSkipVote(t *testing.T) {
	tests := []struct {
		name        string
		admin       bool
		democratic  bool
		skipAllowed bool
		hostMode    bool
		hostID      string
		want        bool
	}{
		{name: "admin votes in democratic room", admin: true, democratic: true, skipAllowed: true, want: true},
		{name: "listener votes in democratic room", democratic: true, skipAllowed: true, want: true},
		{name: "admin skips in admin-only room", admin: true, democratic: true},
		{name: "admin skips when voting disabled", admin: true, skipAllowed: true},
		{name: "admin skips with both bypass settings", admin: true},
		{name: "listener skips when voting disabled", skipAllowed: true},
		{name: "admin-only setting does not grant listener bypass", democratic: true, want: true},
		{name: "listener permission checked separately"},
		{name: "host keeps playback control", hostMode: true, hostID: "user", democratic: true, skipAllowed: true},
		{name: "admin host keeps playback control", admin: true, hostMode: true, hostID: "user", democratic: true, skipAllowed: true},
		{name: "non-host admin votes", admin: true, hostMode: true, hostID: "host", democratic: true, skipAllowed: true, want: true},
		{name: "stale host ID in server room grants no bypass", hostID: "user", democratic: true, skipAllowed: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			room := RoomV2{
				Mode:    RoomModeServer,
				HostID:  tt.hostID,
				IsAdmin: tt.admin,
				Settings: RoomSettingsV2{
					DemocraticSkip: tt.democratic,
					SkipAllowed:    tt.skipAllowed,
				},
			}
			if tt.hostMode {
				room.Mode = RoomModeHost
			}

			got := room.RequiresSkipVote("user")
			if got != tt.want {
				t.Fatalf("RequiresSkipVote = %t, want %t", got, tt.want)
			}
		})
	}
}
