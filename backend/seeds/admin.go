// Package seeds creates initial data that can't live in migrations because
// it comes from configuration.
package seeds

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"livestream/internal/auth"
	"livestream/internal/models"
	"livestream/internal/repositories"
	"livestream/internal/services"
)

// EnsureAdmin creates an admin account if none exists with this email. An
// existing account is left alone, so changing ADMIN_PASSWORD later does
// not reset anything.
func EnsureAdmin(ctx context.Context, users repositories.UserRepository, roles *services.RoleService, name, email, password string, log *slog.Logger) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		log.Warn("ADMIN_EMAIL / ADMIN_PASSWORD not set; no admin account created")
		return nil
	}
	if len(password) < 12 {
		return fmt.Errorf("ADMIN_PASSWORD must be at least 12 characters")
	}

	_, err := users.FindByEmail(ctx, email)
	if err == nil {
		return nil
	}
	if !errors.Is(err, repositories.ErrNotFound) {
		return err
	}

	role, ok := roles.BySlug(auth.RoleAdmin)
	if !ok {
		return fmt.Errorf("role %q missing", auth.RoleAdmin)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	admin := &models.User{Name: name, Email: email, PasswordHash: hash, RoleID: role.ID, Status: models.StatusActive}
	if err := users.Create(ctx, admin); err != nil && !errors.Is(err, repositories.ErrDuplicate) {
		return err
	}
	log.Info("admin account created", "email", email)
	return nil
}
