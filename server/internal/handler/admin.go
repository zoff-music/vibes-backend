package handler

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// AdminLogin handles POST /api/v1/admin/sessions
//
//	@Summary		Sign in to room administration
//	@Description	Validates an admin username and password and creates an authenticated admin session for the current user.
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		request	body		vibe.AdminLoginRequest	true	"Admin login payload"
//	@Success	200		{object}	vibe.AdminSessionResponse
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	403		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/sessions [post]
func AdminLogin(
	fetcher vibe.AdminUserByUsernameFetcher,
	passwordPepper string,
	cookieSecret string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req vibe.AdminLoginRequest
		err := jsonv2.UnmarshalRead(r.Body, &req)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding request body: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		username, err := vibe.NormalizeAdminUsername(req.Username)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error validating admin username: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}

		err = vibe.ValidateAdminPassword(req.Password)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error validating admin password: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}

		session, ok := helper.GetSessionFromContext(ctx)
		if !ok || session.UserID == "" {
			handleError(
				w,
				fmt.Errorf("error unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		admin, err := fetcher.GetAdminUserByUsername(ctx, username)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error getting admin user in AdminLogin handler: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if admin.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error invalid admin credentials"),
				http.StatusForbidden,
				false,
			)
			return
		}

		passwordMatches, err := helper.VerifyAdminPassword(
			req.Password,
			passwordPepper,
			admin.PasswordHash,
		)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error verifying admin password in AdminLogin handler: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if !passwordMatches {
			handleError(
				w,
				fmt.Errorf("error invalid admin credentials"),
				http.StatusForbidden,
				false,
			)
			return
		}

		payload := vibe.AdminAuthPayload{
			UserID:         session.UserID,
			AdminID:        admin.ID,
			SessionVersion: admin.SessionVersion,
			IssuedAt:       time.Now().Unix(),
		}

		signed, err := helper.SignAdminAuthPayload(payload, cookieSecret)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error signing admin session: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     helper.AdminAuthCookieName,
			Value:    signed,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(adminSessionDuration),
		})

		resp := vibe.AdminSessionResponse{
			Authorized: true,
			User:       admin,
		}

		body, err := json.Marshal(resp)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshal response: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// AdminSession handles GET /api/v1/admin/sessions
//
//	@Summary	Get the current admin session
//	@Tags		admin
//	@Produce	json
//	@Success	200	{object}	vibe.AdminSessionResponse
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/sessions [get]
func AdminSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := helper.GetAdminUserFromContext(r.Context())
		if !ok || admin.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		response := vibe.AdminSessionResponse{
			Authorized: true,
			User:       admin,
		}
		body, err := json.Marshal(response)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling admin session response: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// AdminLogout handles DELETE /api/v1/admin/sessions
//
//	@Summary		Sign out of room administration
//	@Description	Clears the current user's admin session.
//	@Tags		admin
//	@Produce	json
//	@Success	200	{object}	vibe.AdminSessionResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/sessions [delete]
func AdminLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     helper.AdminAuthCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})

		resp := vibe.AdminSessionResponse{
			Authorized: false,
		}

		body, err := json.Marshal(resp)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshal response: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// AdminSearchUsage handles GET /api/v1/admin/searches/usage
//
//	@Summary	List search usage activity
//	@Tags		admin
//	@Produce	json
//	@Success	200	{object}	vibe.AdminSearchUsage
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/searches/usage [get]
func AdminSearchUsage(
	db vibe.AdminSearchUsageLister,
	cache vibe.CachedAdminSearchUsageFetcherCreator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		cachedUsage, err := cache.GetCachedAdminSearchUsage(ctx)
		if err != nil {
			log.Printf("error getting cached admin search usage: %v", err)
			cachedUsage = &vibe.CachedAdminSearchUsage{}
		}

		usage := &cachedUsage.Usage
		if cachedUsage.IsEmpty() {
			points, err := db.ListAdminSearchUsage(ctx)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error fetching admin search usage: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			roomPoints, err := db.ListAdminRoomSearchUsage(ctx)
			if err != nil {
				handleError(
					w,
					fmt.Errorf("error fetching room search usage: %w", err),
					http.StatusInternalServerError,
					true,
				)
				return
			}

			usage = &vibe.AdminSearchUsage{
				RoomPoints:  roomPoints,
				Points:      points,
				GeneratedAt: time.Now().UTC(),
			}
			err = cache.CacheAdminSearchUsage(ctx, *usage)
			if err != nil {
				log.Printf("error caching admin search usage: %v", err)
			}
		}

		body, err := json.Marshal(usage)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling admin search usage: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// AdminUsers handles GET /api/v1/admin/users
//
//	@Summary	List admin users
//	@Tags		admin
//	@Produce	json
//	@Success	200	{array}		vibe.AdminUser
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/users [get]
func AdminUsers(lister vibe.AdminUserLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := lister.ListAdminUsers(r.Context())
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error listing admin users in AdminUsers handler: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(users)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling admin users response: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// AdminCreateUser handles POST /api/v1/admin/users
//
//	@Summary	Create an admin user
//	@Tags		admin
//	@Accept		json
//	@Produce	json
//	@Param		request	body		vibe.AdminCreateUserRequest	true	"Admin user"
//	@Success	201		{object}	vibe.AdminUser
//	@Failure	400		{object}	vibe.ErrorResponse
//	@Failure	401		{object}	vibe.ErrorResponse
//	@Failure	409		{object}	vibe.ErrorResponse
//	@Failure	500		{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/users [post]
func AdminCreateUser(
	creator vibe.AdminUserCreator,
	passwordPepper string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request vibe.AdminCreateUserRequest
		err := jsonv2.UnmarshalRead(r.Body, &request)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding admin user request: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		username, err := vibe.NormalizeAdminUsername(request.Username)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error validating admin username: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}

		err = vibe.ValidateAdminPassword(request.Password)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error validating admin password: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}

		passwordHash, err := helper.GenerateAdminPasswordHash(
			request.Password,
			passwordPepper,
		)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error hashing admin password: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		user, err := creator.CreateAdminUser(r.Context(), vibe.AdminUser{
			ID:           uuid.NewString(),
			Username:     username,
			PasswordHash: passwordHash,
		})
		if err != nil {
			var usernameError internalerror.ErrAdminUsernameUnavailable
			if errors.As(err, &usernameError) {
				handleError(
					w,
					fmt.Errorf("error admin username unavailable: %w", err),
					http.StatusConflict,
					false,
				)
				return
			}

			handleError(
				w,
				fmt.Errorf("error creating admin user: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		body, err := json.Marshal(user)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error marshaling admin user response: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(body)
	}
}

// AdminUpdateUser handles PATCH /api/v1/admin/users/{id}
//
//	@Summary	Reset another admin user's password
//	@Tags		admin
//	@Accept		json
//	@Param		id		path	string						true	"Admin user ID"
//	@Param		request	body	vibe.AdminUpdateUserRequest	true	"New password"
//	@Success	204
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	403	{object}	vibe.ErrorResponse
//	@Failure	404	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/users/{id} [patch]
func AdminUpdateUser(
	updater vibe.AdminUserPasswordUpdater,
	passwordPepper string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := helper.GetAdminUserFromContext(r.Context())
		if !ok || admin.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		adminID := mux.Vars(r)["id"]
		if adminID == "" {
			handleError(
				w,
				fmt.Errorf("error admin user id required"),
				http.StatusBadRequest,
				false,
			)
			return
		}
		if adminID == admin.ID {
			handleError(
				w,
				fmt.Errorf("error cannot reset your own password"),
				http.StatusForbidden,
				false,
			)
			return
		}

		var request vibe.AdminUpdateUserRequest
		err := jsonv2.UnmarshalRead(r.Body, &request)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error decoding admin password request: %w", err),
				http.StatusBadRequest,
				true,
			)
			return
		}

		err = vibe.ValidateAdminPassword(request.Password)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error validating admin password: %w", err),
				http.StatusBadRequest,
				false,
			)
			return
		}

		passwordHash, err := helper.GenerateAdminPasswordHash(
			request.Password,
			passwordPepper,
		)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error hashing admin password: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}

		updated, err := updater.UpdateAdminUserPassword(
			r.Context(),
			adminID,
			passwordHash,
		)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error updating admin password: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if !updated {
			handleError(
				w,
				fmt.Errorf("error admin user not found"),
				http.StatusNotFound,
				false,
			)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// AdminDeleteUser handles DELETE /api/v1/admin/users/{id}
//
//	@Summary	Delete another admin user
//	@Tags		admin
//	@Param		id	path	string	true	"Admin user ID"
//	@Success	204
//	@Failure	400	{object}	vibe.ErrorResponse
//	@Failure	401	{object}	vibe.ErrorResponse
//	@Failure	403	{object}	vibe.ErrorResponse
//	@Failure	404	{object}	vibe.ErrorResponse
//	@Failure	500	{object}	vibe.ErrorResponse
//	@Router		/api/v1/admin/users/{id} [delete]
func AdminDeleteUser(deleter vibe.AdminUserDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := helper.GetAdminUserFromContext(r.Context())
		if !ok || admin.IsEmpty() {
			handleError(
				w,
				fmt.Errorf("error unauthorized"),
				http.StatusUnauthorized,
				false,
			)
			return
		}

		adminID := mux.Vars(r)["id"]
		if adminID == "" {
			handleError(
				w,
				fmt.Errorf("error admin user id required"),
				http.StatusBadRequest,
				false,
			)
			return
		}
		if adminID == admin.ID {
			handleError(
				w,
				fmt.Errorf("error cannot delete your own admin user"),
				http.StatusForbidden,
				false,
			)
			return
		}

		deleted, err := deleter.DeleteAdminUser(r.Context(), adminID)
		if err != nil {
			handleError(
				w,
				fmt.Errorf("error deleting admin user: %w", err),
				http.StatusInternalServerError,
				true,
			)
			return
		}
		if !deleted {
			handleError(
				w,
				fmt.Errorf("error admin user not found"),
				http.StatusNotFound,
				false,
			)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

const adminSessionDuration = 24 * time.Hour

const adminRoomPageSize = 10

const adminRoomMaximumPageSize = 50

const adminRoomQueryMaximumLength = 100

const adminRoomOrderAscending = "asc"

const adminRoomOrderDescending = "desc"
