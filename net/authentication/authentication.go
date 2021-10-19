package authentication

type Authentication interface {
	GenerateToken() (string, error)
	ValidateToken(token string) bool
}

type AuthenticationWorker interface {
	Start()
}
