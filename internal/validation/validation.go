package validation

import (
	"reflect"
	"strings"

	"github.com/google/uuid"

	"github.com/go-playground/validator/v10"
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return "validation failed"
	}
	return e.Message
}

var validate = validator.New()

func init() {
	validate.RegisterValidation("notblank", func(fl validator.FieldLevel) bool {
		if fl.Field().Kind() == reflect.Pointer {
			if fl.Field().IsNil() {
				return true
			}

			return strings.TrimSpace(fl.Field().Elem().String()) != ""
		}

		return strings.TrimSpace(fl.Field().String()) != ""
	})

	validate.RegisterValidation("valid_uuid4", func(fl validator.FieldLevel) bool {
		id, err := uuid.Parse(fl.Field().String())
		if err != nil {
			return false
		}

		return id.Version() == 4
	})
}

func Validator(dto any) error {
	err := validate.Struct(dto)
	if err == nil {
		return nil
	}

	return buildMessage(dto, err)
}
