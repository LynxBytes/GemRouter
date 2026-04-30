package validators

import (
	"net/mail"
	"strings"
)

type EmailValidator struct {
	allowDomains map[string]struct{}
	blockDomains map[string]struct{}
}

func NewEmailValidator() *EmailValidator {
	return &EmailValidator{
		allowDomains: nil,
		blockDomains: nil,
	}
}

func (e *EmailValidator) AllowDomains(allowedDomains ...string) *EmailValidator {
	e.allowDomains = make(map[string]struct{})
	for _, d := range allowedDomains {
		e.allowDomains[strings.ToLower(d)] = struct{}{}
	}
	return e
}

func (e *EmailValidator) BlockDomains(blockedDomains ...string) *EmailValidator {
	e.blockDomains = make(map[string]struct{})
	for _, d := range blockedDomains {
		e.blockDomains[strings.ToLower(d)] = struct{}{}
	}
	return e
}

func (e *EmailValidator) IsValid(email string) bool {
	email = strings.TrimSpace(email)

	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}

	at := strings.LastIndex(addr.Address, "@")
	if at < 0 {
		return false
	}

	domain := strings.ToLower(addr.Address[at+1:])

	if e.blockDomains != nil {
		if _, blocked := e.blockDomains[domain]; blocked {
			return false
		}
	}

	if e.allowDomains != nil && len(e.allowDomains) > 0 {
		if _, ok := e.allowDomains[domain]; !ok {
			return false
		}
	}

	if len(domain) < 3 || !strings.Contains(domain, ".") {
		return false
	}

	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}

	return true
}
