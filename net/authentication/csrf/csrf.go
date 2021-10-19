package csrf

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis"
	"golang.org/x/crypto/bcrypt"
)

type CSRFToken struct {
	Token     string
	CreatedAt time.Time
	ExpiredAt time.Time
}

type CSRFService struct {
	redisClient *redis.Client
}

func (c *CSRFService) GenerateToken() (string, error) {
	hasher := sha1.New()
	now := time.Now().UTC()
	bytes, err := bcrypt.GenerateFromPassword([]byte(fmt.Sprintf("%v", now)), 14)
	if err != nil {
		return "", err
	}

	hasher.Write(bytes)
	token := hex.EncodeToString(hasher.Sum(nil))

	if err = c.redisClient.Set(
		fmt.Sprintf("%s:%s", "csrf-token", token),
		fmt.Sprintf("%v", time.Now().UTC()),
		time.Minute*time.Duration(30),
	).Err(); err != nil {
		return "", err
	}

	return token, nil
}

func (c *CSRFService) ValidateToken(token string) bool {
	_, err := c.redisClient.Get("csrf-token:" + token).Result()
	if err != nil {
		log.Printf("Error while collecting `csrf-token` [%s]: %v", token, err)
		return false
	}

	c.redisClient.Del(token)

	return true
}
