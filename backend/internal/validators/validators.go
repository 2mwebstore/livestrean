// Package validators registers request validation rules and turns
// validation failures into per-field messages for the API.
package validators

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// Channel URL names: 3-40 chars, lowercase letters, digits and hyphens,
// not starting or ending with a hyphen.
var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$`)

// Slugs that would be confusing as channel URLs.
var reservedSlugs = map[string]bool{
	"admin": true, "api": true, "dashboard": true, "embed": true, "live": true,
	"login": true, "logout": true, "register": true, "settings": true, "ws": true,
}

func ValidSlug(s string) bool {
	return slugPattern.MatchString(s) && !reservedSlugs[s]
}

// Register hooks our rules into Gin's validator. Call once at startup.
func Register() error {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return errors.New("unexpected validator engine")
	}
	// Report JSON field names ("email") instead of Go names ("Email").
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name := strings.SplitN(f.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	return v.RegisterValidation("slug", func(fl validator.FieldLevel) bool {
		return ValidSlug(fl.Field().String())
	})
}

// FieldErrors converts a binding error into {"field": "message"}.
func FieldErrors(err error) map[string]string {
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		out := make(map[string]string, len(verrs))
		for _, fe := range verrs {
			out[fe.Field()] = message(fe)
		}
		return out
	}
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(err, &typeErr):
		return map[string]string{typeErr.Field: "Has the wrong type"}
	case errors.As(err, &syntaxErr), err.Error() == "EOF":
		return map[string]string{"body": "Request body must be valid JSON"}
	}
	return map[string]string{"body": "Invalid request"}
}

func message(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Enter a valid email address"
	case "min":
		return fmt.Sprintf("Must be at least %s characters", fe.Param())
	case "max":
		return fmt.Sprintf("Must be at most %s characters", fe.Param())
	case "oneof":
		return "Must be one of: " + strings.ReplaceAll(fe.Param(), " ", ", ")
	case "slug":
		return "Use 3-40 lowercase letters, numbers or hyphens (not at the start or end)"
	}
	return "Invalid value"
}
