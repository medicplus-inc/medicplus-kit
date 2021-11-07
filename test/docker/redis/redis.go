package redis

import (
	"log"
	"net"

	"github.com/go-redis/redis"
	"github.com/ory/dockertest"
)

func GenerateInstance(pool *dockertest.Pool) (*redis.Client, *dockertest.Resource) {
	var redisClient *redis.Client

	// Run the Docker container
	resource, err := pool.Run("redis", "latest", nil)
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	// Exponential retry to connect to redis while it is booting
	if err := pool.Retry(func() error {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     net.JoinHostPort("localhost", resource.GetPort("6379/tcp")),
			Password: "", // no password set
			DB:       0,  // use default DB
		})

		ping := redisClient.Ping()
		return ping.Err()
	}); err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	return redisClient, resource
}
