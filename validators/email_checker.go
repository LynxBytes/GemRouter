package validators

type EmailChecker interface {
	IsValid(email string) bool
}
