package authentication

type Authentication interface {
	GenerateToken() string
	ValidateToken(token string) bool
}

type AuthenticationWorker interface {
	Start()
}
