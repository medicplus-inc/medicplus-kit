package csrf

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type CSRFToken struct {
	Token     string
	CreatedAt time.Time
	ExpiredAt time.Time
}

var inMemToken map[string]CSRFToken

func init() {
	inMemToken = map[string]CSRFToken{}
}

func (c *CSRFToken) GenerateToken() string {
	hasher := sha1.New()
	now := time.Now().UTC()
	bytes, err := bcrypt.GenerateFromPassword([]byte(fmt.Sprintf("%v", now)), 14)
	if err != nil {
		log.Println("Error while generating CSRF token: ", err)
	}
	hasher.Write(bytes)
	token := hex.EncodeToString(hasher.Sum(nil))
	inMemToken[token] = CSRFToken{
		Token:     token,
		CreatedAt: now,
		ExpiredAt: now.Add(time.Minute * time.Duration(30)),
	}

	return token
}

func ValidateToken(token string) bool {
	v, ok := inMemToken[token]
	if ok {
		now := time.Now().UTC()
		if now.Before(v.ExpiredAt) || now.Equal(v.ExpiredAt) {
			delete(inMemToken, token)
			return true
		}
		delete(inMemToken, token)
	}
	return false
}
