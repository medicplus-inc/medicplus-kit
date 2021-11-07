package redis

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/go-redis/redis"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
)

var redisPort int = 6379
var lock sync.Mutex

func nextPort() int {
	lock.Lock()
	redisPort = redisPort + 1

	defer lock.Unlock()
	return redisPort
}

func isOpen(port string) bool {
	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("", port), timeout)
	if err != nil {
		return true
	}
	if conn != nil {
		defer conn.Close()
		return false
	}
	return false
}

func getAvailablePort() string {
	for {
		p := fmt.Sprintf("%d", nextPort())
		if isOpen(p) {
			return p
		}
	}
}

func GenerateInstance(pool *dockertest.Pool) (*redis.Client, *dockertest.Resource, string) {
	var redisClient *redis.Client
	port := getAvailablePort()

	// Pull an image, create a container based on it and set all necessary parameters
	opts := dockertest.RunOptions{
		Repository:   "bitnami/redis",
		Tag:          "latest",
		ExposedPorts: []string{"6379"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"6379": {
				{HostIP: "0.0.0.0", HostPort: port},
			},
		},
	}

	// Run the Docker container
	resource, err := pool.RunWithOptions(&opts)
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	// // if run with docker-machine the hostname needs to be set
	// u, err := url.Parse(pool.Client.Endpoint())
	// if err != nil {
	// 	log.Fatalf("Could not parse endpoint: %s", pool.Client.Endpoint())
	// }

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

	return redisClient, resource, port
}
