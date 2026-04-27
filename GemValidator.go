package gemrouter

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type GemValidator struct {
	errors []ValidationError
}

func NewValidator() *GemValidator {
	return &GemValidator{}
}

func (v *GemValidator) Check(field string, value any, rules string) *GemValidator {
	for _, rule := range strings.Split(rules, ",") {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}

		tag, param, _ := strings.Cut(rule, "=")

		if msg := applyRule(tag, param, value); msg != "" {
			v.errors = append(v.errors, ValidationError{Field: field, Message: msg})
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

func applyRule(tag, param string, value any) string {
	switch tag {
	case "required":
		if isEmpty(value) {
			return "is required"
		}

	case "min":
		n, _ := strconv.Atoi(param)
		switch val := value.(type) {
		case string:
			if utf8.RuneCountInString(val) < n {
				return fmt.Sprintf("must be at least %d characters", n)
			}
		case int:
			if val < n {
				return fmt.Sprintf("must be at least %d", n)
			}
		case float64:
			if val < float64(n) {
				return fmt.Sprintf("must be at least %d", n)
			}
		}

	case "max":
		n, _ := strconv.Atoi(param)
		switch val := value.(type) {
		case string:
			if utf8.RuneCountInString(val) > n {
				return fmt.Sprintf("must be at most %d characters", n)
			}
		case int:
			if val > n {
				return fmt.Sprintf("must be at most %d", n)
			}
		case float64:
			if val > float64(n) {
				return fmt.Sprintf("must be at most %d", n)
			}
		}

	case "len":
		n, _ := strconv.Atoi(param)
		if s, ok := value.(string); ok && utf8.RuneCountInString(s) != n {
			return fmt.Sprintf("must be exactly %d characters", n)
		}

	case "email":
		s, ok := value.(string)
		if !ok || !isEmail(s) {
			return "must be a valid email"
		}
	}

	return ""
}

func isEmpty(value any) bool {
	switch v := value.(type) {
	case string:
		return v == ""
	case int:
		return v == 0
	case float64:
		return v == 0
	case bool:
		return !v
	case nil:
		return true
	}
	return false
}

func isEmail(s string) bool {
	at := strings.LastIndex(s, "@")
	if at <= 0 || at == len(s)-1 {
		return false
	}
	domain := s[at+1:]
	return strings.Contains(domain, ".") && !strings.HasSuffix(domain, ".")
}
