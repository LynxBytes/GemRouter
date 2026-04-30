package gem

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/LynxBytes/GemRouter/validators"
)

type ValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Rule func(value any) *ValidationError

type Validator struct {
	errors         []ValidationError
	EmailValidator validators.EmailChecker
}

func NewValidator() *Validator {
	return &Validator{
		EmailValidator: validators.NewEmailValidator(),
	}
}

func (v *Validator) SetEmailValidator(ev validators.EmailChecker) *Validator {
	v.EmailValidator = ev
	return v
}

func (v *Validator) Check(field string, value any, rules ...Rule) *Validator {
	for _, rule := range rules {
		if err := rule(value); err != nil {
			v.errors = append(v.errors, ValidationError{
				Field:   field,
				Code:    err.Code,
				Message: err.Message,
			})
			break
		}
	}
	return v
}

func (v *Validator) Valid() bool {
	return len(v.errors) == 0
}

func (v *Validator) Errors() []ValidationError {
	return v.errors
}

func If(do bool, rules ...Rule) Rule {
	return func(value any) *ValidationError {
		if !do {
			return nil
		}

		for _, rule := range rules {
			if err := rule(value); err != nil {
				return err
			}
		}

		return nil
	}
}

func And(rules ...Rule) Rule {
	return func(value any) *ValidationError {
		for _, rule := range rules {
			if err := rule(value); err != nil {
				return err
			}
		}
		return nil
	}
}

func Or(rules ...Rule) Rule {
	return func(value any) *ValidationError {
		var lastErr *ValidationError

		for _, rule := range rules {
			err := rule(value)
			if err == nil {
				return nil
			}
			lastErr = err
		}

		if lastErr != nil {
			return lastErr
		}

		return &ValidationError{
			Code:    "VALIDATION_ERROR_OR_FAILED",
			Message: "No rule in OR group matched",
		}
	}
}

func Null() Rule {
	return func(value any) *ValidationError {
		if value == nil {
			return nil
		}
		return &ValidationError{
			Code:    "VALIDATION_ERROR_NULL",
			Message: "Must be null",
		}
	}
}

func NotNull() Rule {
	return func(value any) *ValidationError {
		if value != nil {
			return nil
		}

		return &ValidationError{
			Code:    "VALIDATION_ERROR_NOTNULL",
			Message: "Must be not null",
		}
	}
}

func Empty() Rule {
	return func(value any) *ValidationError {
		switch v := value.(type) {
		case string:
			if v == "" {
				return nil
			}
		default:
			return nil
		}

		return &ValidationError{
			Code:    "VALIDATION_ERROR_EMPTY",
			Message: "Must be empty",
		}
	}
}

func NotEmpty() Rule {
	return func(value any) *ValidationError {
		switch v := value.(type) {
		case string:
			if v != "" {
				return nil
			}
		}

		return &ValidationError{
			Code:    "VALIDATION_ERROR_NOTEMPTY",
			Message: "Must be empty",
		}
	}
}

func Required() Rule {
	return func(value any) *ValidationError {
		switch v := value.(type) {
		case string:
			if v == "" {
				return &ValidationError{
					Code:    "VALIDATION_ERROR_REQUIRED",
					Message: "Is required",
				}
			}
		case int:
			if v == 0 {
				return &ValidationError{
					Code:    "VALIDATION_ERROR_REQUIRED",
					Message: "Is required",
				}
			}
		case float64:
			if v == 0 {
				return &ValidationError{
					Code:    "VALIDATION_ERROR_REQUIRED",
					Message: "Is required",
				}
			}
		case bool:
			return nil
		case nil:
			return &ValidationError{
				Code:    "VALIDATION_ERROR_REQUIRED",
				Message: "Is required",
			}
		}
		return nil
	}
}

func Min(n int) Rule {
	return func(value any) *ValidationError {
		switch v := value.(type) {
		case string:
			if utf8.RuneCountInString(v) < n {
				return &ValidationError{
					Code:    "VALIDATION_ERROR_MIN",
					Message: fmt.Sprintf("Must be at least %d characters", n),
				}
			}
		case int:
			if v < n {
				return &ValidationError{
					Code:    "VALIDATION_ERROR_MIN",
					Message: fmt.Sprintf("Must be at least %d", n),
				}
			}
		case float64:
			if v < float64(n) {
				return &ValidationError{
					Code:    "VALIDATION_ERROR_MIN",
					Message: fmt.Sprintf("Must be at least %d", n),
				}
			}
		}
		return nil
	}
}

func Max(n int) Rule {
	return func(value any) *ValidationError {
		switch v := value.(type) {
		case string:
			if utf8.RuneCountInString(v) > n {
				return &ValidationError{
					Code:    "VALIDATION_ERROR_MAX",
					Message: fmt.Sprintf("Must be at most %d characters", n),
				}
			}
		case int:
			if v > n {
				return &ValidationError{
					Code:    "VALIDATION_ERROR_MAX",
					Message: fmt.Sprintf("Must be at most %d", n),
				}
			}
		case float64:
			if v > float64(n) {
				return &ValidationError{
					Code:    "VALIDATION_ERROR_MAX",
					Message: fmt.Sprintf("Must be at most %d", n),
				}
			}
		}
		return nil
	}
}

func Len(n int) Rule {
	return func(value any) *ValidationError {
		s, ok := value.(string)
		if !ok {
			return nil
		}

		if utf8.RuneCountInString(s) != n {
			return &ValidationError{
				Code:    "VALIDATION_ERROR_LEN",
				Message: fmt.Sprintf("Must be exactly %d characters", n),
			}
		}

		return nil
	}
}

func Email(ev validators.EmailChecker) Rule {
	return func(value any) *ValidationError {
		s, ok := value.(string)
		if !ok {
			return &ValidationError{
				Code:    "VALIDATION_ERROR_EMAIL",
				Message: "Must be a valid email",
			}
		}

		if ev == nil {
			ev = validators.NewEmailValidator()
		}

		if !ev.IsValid(strings.TrimSpace(s)) {
			return &ValidationError{
				Code:    "VALIDATION_ERROR_EMAIL",
				Message: "Must be a valid email",
			}
		}

		return nil
	}
}

func Enum(valid func() bool, message string) Rule {
	return func(_ any) *ValidationError {
		if !valid() {
			return &ValidationError{
				Code:    "VALIDATION_ERROR_ENUM",
				Message: message,
			}
		}
		return nil
	}
}
