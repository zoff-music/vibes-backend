package vibe

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"slices"
	"time"
)

// RoomSettings holds configuration for a room
type RoomSettings struct {
	SkipAllowed       bool     `json:"skipAllowed"`
	DemocraticSkip    bool     `json:"democraticSkip"`
	SkipVoteThreshold float64  `json:"skipVoteThreshold"`
	MaxContinuousAdds int      `json:"maxContinuousAdds"`
	RemoveOnPlay      bool     `json:"removeOnPlay"`
	LoopQueue         bool     `json:"loopQueue"`
	AllowDuplicates   bool     `json:"allowDuplicates"`
	EnabledSources    []string `json:"enabledSources"`
	OnlyAdminAddSongs bool     `json:"onlyAdminAddSongs"`
}

func (r RoomSettings) IsEmpty() bool {
	return r.SkipAllowed == false &&
		r.DemocraticSkip == false &&
		r.SkipVoteThreshold == 0 &&
		r.MaxContinuousAdds == 0 &&
		r.RemoveOnPlay == false &&
		r.LoopQueue == false &&
		r.AllowDuplicates == false &&
		len(r.EnabledSources) == 0 &&
		r.OnlyAdminAddSongs == false
}

// DefaultRoomSettings returns sensible defaults
func DefaultRoomSettings() RoomSettings {
	return RoomSettings{
		SkipAllowed:       true,
		DemocraticSkip:    true,
		SkipVoteThreshold: 0.5,
		MaxContinuousAdds: 3,
		RemoveOnPlay:      false,
		LoopQueue:         true,
		AllowDuplicates:   false,
		EnabledSources:    []string{"youtube", "spotify", "soundcloud"},
		OnlyAdminAddSongs: false,
	}
}

// Room represents a music room
type Room struct {
	ID                string       `json:"id"`
	Name              string       `json:"name"`
	Mode              string       `json:"mode"`
	HostID            string       `json:"hostId,omitempty"`
	AdminPasswordHash string       `json:"-"`
	HasPassword       bool         `json:"hasPassword"`
	Settings          RoomSettings `json:"settings"`
	CreatedAt         time.Time    `json:"createdAt"`
	UserCount         int          `json:"userCount,omitempty"`
	IsAdmin           bool         `json:"isAdmin"`
	UserID            string       `json:"userId,omitempty"`
	ActiveSources     []string     `json:"activeSources"`
	IsGenerating      bool         `json:"isGenerating"`
}

// RoomNameSuggestion is an available, memorable name for a new room.
type RoomNameSuggestion struct {
	Name string `json:"name"`
}

// RoomHostInfo holds info about a host update
type RoomHostInfo struct {
	RoomID    string
	NewHostID string
}

// CreateRoomRequest is the request payload for creating a room.
type CreateRoomRequest struct {
	Name     string        `json:"name"`
	Mode     string        `json:"mode,omitempty"`
	Password string        `json:"password,omitempty"`
	Settings *RoomSettings `json:"settings,omitempty"`
}

// UpdateRoomRequest is the request payload for updating a room.
type UpdateRoomRequest struct {
	Mode     string        `json:"mode,omitempty"`
	Settings *RoomSettings `json:"settings,omitempty"`
}

// IsEmpty returns true if the room is empty/not found
func (r *Room) IsEmpty() bool {
	return r.ID == ""
}

// RoomFetcher fetches room data
type RoomFetcher interface {
	GetRoom(ctx context.Context, id string, userID string) (*Room, error)
}

// RoomNameSuggester finds an available room name from a set of candidates.
type RoomNameSuggester interface {
	SuggestRoomName(ctx context.Context, candidates []string) (*RoomNameSuggestion, error)
}

// RoomExistenceChecker checks whether a room ID is already in use.
type RoomExistenceChecker interface {
	RoomExists(ctx context.Context, roomID string) (bool, error)
}

// RoomCreator creates rooms
type RoomCreator interface {
	CreateRoom(ctx context.Context, room *Room) (*Room, error)
}

type RoomCreatorAdminRoomLister interface {
	RoomCreator
	RoomExistenceChecker
	AdminRoomLister
}

// RoomUpdater updates room data
type RoomUpdater interface {
	UpdateRoom(ctx context.Context, room *Room) (*Room, error)
}

// RoomSettingsUpdater fetches and updates room data
type RoomSettingsUpdater interface {
	RoomFetcher
	RoomUpdater
}

// GenerateRoomNameCandidates generates random three-word room names.
func GenerateRoomNameCandidates() ([]string, error) {
	words := []string{
		"amber",
		"apple",
		"bake",
		"blue",
		"bold",
		"brave",
		"bright",
		"calm",
		"chase",
		"cheer",
		"clever",
		"cloud",
		"cozy",
		"dance",
		"dream",
		"drift",
		"eager",
		"easy",
		"echo",
		"fast",
		"fire",
		"float",
		"fly",
		"fresh",
		"frost",
		"gentle",
		"glad",
		"glow",
		"gold",
		"green",
		"happy",
		"hike",
		"honey",
		"hope",
		"jolly",
		"jump",
		"kind",
		"laugh",
		"light",
		"lucky",
		"magic",
		"mint",
		"moon",
		"neat",
		"north",
		"orange",
		"peach",
		"play",
		"quick",
		"quiet",
		"river",
		"roam",
		"round",
		"shine",
		"silver",
		"sing",
		"sky",
		"soft",
		"spark",
		"star",
		"sunny",
		"swift",
		"tall",
		"tidy",
		"tiny",
		"travel",
		"warm",
		"wave",
		"wild",
		"wise",
		"yellow",
		"young",
		"acorn",
		"air",
		"alpine",
		"aqua",
		"arrow",
		"aspen",
		"aurora",
		"autumn",
		"badger",
		"bamboo",
		"banana",
		"basil",
		"beach",
		"bear",
		"beacon",
		"berry",
		"birch",
		"bird",
		"bloom",
		"boat",
		"brick",
		"breeze",
		"brook",
		"bubble",
		"bunny",
		"butter",
		"cabin",
		"camel",
		"candle",
		"canoe",
		"canyon",
		"cedar",
		"cherry",
		"cocoa",
		"comet",
		"copper",
		"coral",
		"creek",
		"crystal",
		"daisy",
		"dawn",
		"deer",
		"delta",
		"dolphin",
		"dove",
		"dune",
		"eagle",
		"elm",
		"ember",
		"falcon",
		"feather",
		"fern",
		"finch",
		"fjord",
		"flame",
		"flower",
		"flute",
		"forest",
		"fox",
		"garden",
		"gem",
		"ginger",
		"grape",
		"grove",
		"harbor",
		"hazel",
		"heron",
		"hill",
		"island",
		"ivy",
		"jade",
		"jazz",
		"juniper",
		"kiwi",
		"kite",
		"lagoon",
		"lake",
		"leaf",
		"lemon",
		"lighthouse",
		"lime",
		"lion",
		"lotus",
		"mango",
		"maple",
		"marsh",
		"meadow",
		"melon",
		"mist",
		"moose",
		"morning",
		"moss",
		"mountain",
		"oak",
		"ocean",
		"olive",
		"orchid",
		"otter",
		"owl",
		"palm",
		"panda",
		"pebble",
		"penguin",
		"pearl",
		"pepper",
		"pine",
		"planet",
		"plum",
		"pond",
		"poppy",
		"puffin",
		"rabbit",
		"rain",
		"raven",
		"reef",
		"robin",
		"rocket",
		"rose",
		"ruby",
		"salmon",
		"sand",
		"seal",
		"shell",
		"snow",
		"sparrow",
		"spring",
		"spruce",
		"stone",
		"storm",
		"sun",
		"sunset",
		"surf",
		"thunder",
		"tiger",
		"tulip",
		"valley",
		"violet",
		"walnut",
		"waterfall",
		"whale",
		"wheat",
		"willow",
		"wind",
		"winter",
		"wolf",
		"wood",
		"zebra",
	}

	const candidateCount = 512

	var maximum big.Int
	maximum.SetInt64(int64(len(words)))

	pickWord := func(selectedWords []string) (string, error) {
		for {
			randomIndex, err := rand.Int(rand.Reader, &maximum)
			if err != nil {
				return "", fmt.Errorf("error generating random room name index: %w", err)
			}

			word := words[int(randomIndex.Int64())]
			if slices.Contains(selectedWords, word) {
				continue
			}

			return word, nil
		}
	}

	generateCandidate := func() (string, error) {
		selectedWords := make([]string, 0, 3)

		firstWord, err := pickWord(selectedWords)
		if err != nil {
			return "", fmt.Errorf("error selecting first room name word: %w", err)
		}
		selectedWords = append(selectedWords, firstWord)

		secondWord, err := pickWord(selectedWords)
		if err != nil {
			return "", fmt.Errorf("error selecting second room name word: %w", err)
		}
		selectedWords = append(selectedWords, secondWord)

		thirdWord, err := pickWord(selectedWords)
		if err != nil {
			return "", fmt.Errorf("error selecting third room name word: %w", err)
		}

		return firstWord + "-" + secondWord + "-" + thirdWord, nil
	}

	candidates := make([]string, 0, candidateCount)
	seenCandidates := make(map[string]bool, candidateCount)
	for len(candidates) < candidateCount {
		candidate, err := generateCandidate()
		if err != nil {
			return nil, fmt.Errorf("error generating room name candidate: %w", err)
		}

		if seenCandidates[candidate] {
			continue
		}

		seenCandidates[candidate] = true
		candidates = append(candidates, candidate)
	}

	return candidates, nil
}

// RoomModeServer is the mode where the server controls playback
const RoomModeServer = "server"

// RoomModeHost is the mode where a host controls playback
const RoomModeHost = "host"
