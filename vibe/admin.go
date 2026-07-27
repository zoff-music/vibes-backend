package vibe

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

type AdminRoomSummary struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	UserCount        int      `json:"userCount"`
	SongCount        int      `json:"songCount"`
	ActiveSources    []string `json:"activeSources"`
	HasAdminPassword bool     `json:"hasAdminPassword"`
}

type AdminRoomSearch struct {
	Query      string
	SortBy     AdminRoomSort
	Descending bool
	From       int
	To         int
}

type AdminRoomResult struct {
	Rooms []AdminRoomSummary `json:"rooms"`
	From  int                `json:"from"`
	To    int                `json:"to"`
	Total int                `json:"total"`
	Count int                `json:"count"`
}

type AdminRoomSort string

type AdminUpdateRoomRequest struct {
	Name               *string `json:"name,omitempty"`
	ClearAdminPassword *bool   `json:"clearAdminPassword,omitempty"`
}

type AdminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AdminSessionResponse struct {
	Authorized bool       `json:"authorized"`
	User       *AdminUser `json:"user,omitempty"`
}

type AdminUser struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	PasswordHash   string    `json:"-"`
	SessionVersion int64     `json:"-"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (u *AdminUser) IsEmpty() bool {
	return u.ID == ""
}

type AdminCreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AdminUpdateUserRequest struct {
	Password string `json:"password"`
}

func NormalizeAdminUsername(username string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(username))
	if len(normalized) < AdminUsernameMinimumLength ||
		len(normalized) > AdminUsernameMaximumLength {
		return "", fmt.Errorf(
			"error admin username must contain between %d and %d characters",
			AdminUsernameMinimumLength,
			AdminUsernameMaximumLength,
		)
	}

	for _, character := range normalized {
		valid := character >= 'a' && character <= 'z'
		valid = valid || character >= '0' && character <= '9'
		valid = valid || character == '-' || character == '_'
		if !valid {
			return "", fmt.Errorf(
				"error admin username contains an unsupported character",
			)
		}
	}

	return normalized, nil
}

func ValidateAdminPassword(password string) error {
	length := utf8.RuneCountInString(password)
	if length < AdminPasswordMinimumLength ||
		length > AdminPasswordMaximumLength {
		return fmt.Errorf(
			"error admin password must contain between %d and %d characters",
			AdminPasswordMinimumLength,
			AdminPasswordMaximumLength,
		)
	}

	return nil
}

type AdminSearchUsageSummary struct {
	Window   string `json:"window"`
	Provider string `json:"provider"`
	Total    int64  `json:"total"`
	Unique   int64  `json:"unique"`
	Cached   int64  `json:"cached"`
	Live     int64  `json:"live"`
}

type AdminSearchUsage struct {
	Summaries   []AdminSearchUsageSummary `json:"summaries"`
	GeneratedAt time.Time                 `json:"generatedAt"`
}

type CachedAdminSearchUsage struct {
	Usage AdminSearchUsage
	Found bool
}

func (u *CachedAdminSearchUsage) IsEmpty() bool {
	return !u.Found
}

type AdminEvent struct {
	Type    string `json:"type"`
	Payload []byte `json:"payload"`
}

type AdminRoomLister interface {
	ListAdminRooms(ctx context.Context) ([]AdminRoomSummary, error)
}

type AdminRoomSearcher interface {
	SearchAdminRooms(ctx context.Context, search AdminRoomSearch) (*AdminRoomResult, error)
}

type AdminUserFetcher interface {
	GetAdminUser(ctx context.Context, adminID string) (*AdminUser, error)
}

type AdminUserByUsernameFetcher interface {
	GetAdminUserByUsername(ctx context.Context, username string) (*AdminUser, error)
}

type AdminUserLister interface {
	ListAdminUsers(ctx context.Context) ([]AdminUser, error)
}

type AdminUserCreator interface {
	CreateAdminUser(ctx context.Context, user AdminUser) (*AdminUser, error)
}

type AdminUserPasswordUpdater interface {
	UpdateAdminUserPassword(
		ctx context.Context,
		adminID string,
		passwordHash string,
	) (bool, error)
}

type AdminUserDeleter interface {
	DeleteAdminUser(ctx context.Context, adminID string) (bool, error)
}

type AdminSearchUsageLister interface {
	ListAdminSearchUsage(ctx context.Context) ([]AdminSearchUsageSummary, error)
}

type CachedAdminSearchUsageFetcher interface {
	GetCachedAdminSearchUsage(ctx context.Context) (*CachedAdminSearchUsage, error)
}

type CachedAdminSearchUsageCreator interface {
	CacheAdminSearchUsage(ctx context.Context, usage AdminSearchUsage) error
}

type CachedAdminSearchUsageFetcherCreator interface {
	CachedAdminSearchUsageFetcher
	CachedAdminSearchUsageCreator
}

type AdminRoomUpdater interface {
	UpdateAdminRoom(ctx context.Context, roomID string, request AdminUpdateRoomRequest) (bool, error)
}

type AdminRoomDeleter interface {
	DeleteAdminRoom(ctx context.Context, roomID string) (bool, error)
}

// AdminRoomUpdaterLister updates a room and lists the resulting room state.
type AdminRoomUpdaterLister interface {
	AdminRoomLister
	AdminRoomUpdater
}

// AdminRoomDeleterLister deletes a room and lists the resulting room state.
type AdminRoomDeleterLister interface {
	AdminRoomLister
	AdminRoomDeleter
}

type AdminEventNotifier interface {
	NotifyAdminUpdate(ctx context.Context, event AdminEvent) error
}

const AdminRoomsUpdate = "admin_rooms_update"

const AdminRoomSortListeners AdminRoomSort = "listeners"

const AdminRoomSortSongs AdminRoomSort = "songs"

const AdminUsernameMinimumLength = 3

const AdminUsernameMaximumLength = 64

const AdminPasswordMinimumLength = 16

const AdminPasswordMaximumLength = 128
