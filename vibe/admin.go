package vibe

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

type AdminUpdateRoomRequest struct {
	Flagged            *bool   `json:"flagged,omitempty"`
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

type AdminSearchUsagePoint struct {
	Window    string    `json:"window"`
	Timestamp time.Time `json:"timestamp"`
	Provider  string    `json:"provider"`
	Total     int       `json:"total"`
	Unique    int       `json:"unique"`
	Cached    int       `json:"cached"`
	Live      int       `json:"live"`
}

type AdminSearchUsage struct {
	RoomPoints  []RoomSearchUsagePoint  `json:"roomPoints"`
	Points      []AdminSearchUsagePoint `json:"points"`
	GeneratedAt time.Time               `json:"generatedAt"`
}

// MarshalJSON preserves empty collections in the public JSON contract.
func (v AdminSearchUsage) MarshalJSON() ([]byte, error) {
	type payload AdminSearchUsage
	result := payload(v)

	if result.RoomPoints == nil {
		result.RoomPoints = make([]RoomSearchUsagePoint, 0)
	}

	if result.Points == nil {
		result.Points = make([]AdminSearchUsagePoint, 0)
	}

	body, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("error marshaling AdminSearchUsage: %w", err)
	}

	return body, nil
}

type RoomSearchUsagePoint struct {
	RoomType  RoomType  `json:"roomType,omitempty"`
	RoomID    string    `json:"roomId"`
	Window    string    `json:"window"`
	Timestamp time.Time `json:"timestamp"`
	Provider  string    `json:"provider"`
	Total     int       `json:"total"`
	Cached    int       `json:"cached"`
	Live      int       `json:"live"`
}

type CachedAdminSearchUsage struct {
	Usage AdminSearchUsage
	Found bool
}

func (u *CachedAdminSearchUsage) IsEmpty() bool {
	return !u.Found
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
	ListAdminRoomSearchUsage(ctx context.Context) ([]RoomSearchUsagePoint, error)
	ListAdminSearchUsage(ctx context.Context) ([]AdminSearchUsagePoint, error)
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

type AdminRoomReview struct {
	Rooms []AdminRoomSummaryV2
}

func (r *AdminRoomReview) IsEmpty() bool {
	return len(r.Rooms) == 0
}

type AdminRoomReviewFetcher interface {
	GetNextAdminRoomReview(ctx context.Context) (*AdminRoomReview, error)
}

type AdminRoomUpdater interface {
	UpdateAdminRoom(ctx context.Context, roomID string, request AdminUpdateRoomRequest) (bool, error)
}

type AdminRoomDeleter interface {
	DeleteAdminRoom(ctx context.Context, roomID string) (bool, error)
}

const AdminRoomsUpdate = "admin_rooms_update"

const AdminUsernameMinimumLength = 3

const AdminUsernameMaximumLength = 64

const AdminPasswordMinimumLength = 16

const AdminPasswordMaximumLength = 128
