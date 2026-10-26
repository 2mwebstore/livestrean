// Package repositories is the only place that talks to the database.
package repositories

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrNotFound  = errors.New("record not found")
	ErrDuplicate = errors.New("duplicate record")
)

// translate maps GORM errors to repository errors so services don't depend on GORM.
func translate(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return ErrNotFound
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return ErrDuplicate
	default:
		return err
	}
}

func first[T any](q *gorm.DB) (*T, error) {
	var v T
	if err := q.First(&v).Error; err != nil {
		return nil, translate(err)
	}
	return &v, nil
}

// affected turns "no rows matched" into ErrNotFound.
func affected(res *gorm.DB) error {
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// containsPattern builds a LIKE pattern that matches s literally.
func containsPattern(s string) string {
	return "%" + likeEscaper.Replace(s) + "%"
}

func offset(page, limit int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * limit
}
