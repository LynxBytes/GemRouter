package gem

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Rule func(value any) string

type GemValidator struct {
	errors       []ValidationError
	emailChecker func(string) bool
}

func NewValidator() *GemValidator {
	return &GemValidator{}
}

func (v *GemValidator) Check(field string, value any, rules ...Rule) *GemValidator {
	for _, rule := range rules {
		if msg := rule(value); msg != "" {
			v.errors = append(v.errors, ValidationError{
				Field:   field,
				Message: msg,
			})
			break
		}
	}
	return v
}

func (v *GemValidator) Valid() bool {
	return len(v.errors) == 0
}

func (v *GemValidator) Errors() []ValidationError {
	return v.errors
}

func (v *GemValidator) SetEmailChecker(fn func(string) bool) *GemValidator {
	v.emailChecker = fn
	return v
}

func Required() Rule {
	return func(value any) string {
		switch v := value.(type) {
		case string:
			if v == "" {
				return "Is required"
			}
		case int:
			if v == 0 {
				return "Is required"
			}
		case float64:
			if v == 0 {
				return "Is required"
			}
		case bool:
			return "" // false es válido
		case nil:
			return "Is required"
		}
		return ""
	}
}

func Min(n int) Rule {
	return func(value any) string {
		switch v := value.(type) {
		case string:
			if utf8.RuneCountInString(v) < n {
				return fmt.Sprintf("Must be at least %d characters", n)
			}
		case int:
			if v < n {
				return fmt.Sprintf("Must be at least %d", n)
			}
		case float64:
			if v < float64(n) {
				return fmt.Sprintf("Must be at least %d", n)
			}
		}
		return ""
	}
}

func Max(n int) Rule {
	return func(value any) string {
		switch v := value.(type) {
		case string:
			if utf8.RuneCountInString(v) > n {
				return fmt.Sprintf("Must be at most %d characters", n)
			}
		case int:
			if v > n {
				return fmt.Sprintf("Must be at most %d", n)
			}
		case float64:
			if v > float64(n) {
				return fmt.Sprintf("Must be at most %d", n)
			}
		}
		return ""
	}
}

func Len(n int) Rule {
	return func(value any) string {
		s, ok := value.(string)
		if !ok {
			return ""
		}
		if utf8.RuneCountInString(s) != n {
			return fmt.Sprintf("Must be exactly %d characters", n)
		}
		return ""
	}
}

func Email(checker ...func(string) bool) Rule {
	return func(value any) string {
		s, ok := value.(string)
		if !ok {
			return "Must be a valid email"
		}

		// custom override
		if len(checker) > 0 && checker[0] != nil {
			if !checker[0](s) {
				return "Must be a valid email"
			}
			return ""
		}

		at := strings.LastIndex(s, "@")
		if at <= 0 || at == len(s)-1 {
			return "Must be a valid email"
		}

		domain := s[at+1:]
		if !strings.Contains(domain, ".") || strings.HasSuffix(domain, ".") {
			return "Must be a valid email"
		}

		return ""
	}
}

func Enum(valid func() bool, message string) Rule {
	return func(_ any) string {
		if !valid() {
			return message
		}
		return ""
	}
}
