package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/zoff-music/vibes-backend/internalerror"
	"github.com/zoff-music/vibes-backend/server/internal/helper"
	"github.com/zoff-music/vibes-backend/vibe"
)

// AdminUsers handles GET /api/v1/admin/users
//
//	@Summary	List admin users
//	@Tags		admin
//	@Produce	json
//	@Success	200	{array}		vibe.AdminUser
//	@Failure	401	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
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
//	@Failure	400		{object}	map[string]string
//	@Failure	401		{object}	map[string]string
//	@Failure	409		{object}	map[string]string
//	@Failure	500		{object}	map[string]string
//	@Router		/api/v1/admin/users [post]
func AdminCreateUser(
	creator vibe.AdminUserCreator,
	passwordPepper string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request vibe.AdminCreateUserRequest
		err := json.NewDecoder(r.Body).Decode(&request)
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
//	@Failure	400	{object}	map[string]string
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
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
		err := json.NewDecoder(r.Body).Decode(&request)
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
//	@Failure	400	{object}	map[string]string
//	@Failure	401	{object}	map[string]string
//	@Failure	403	{object}	map[string]string
//	@Failure	404	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
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
