package postgres

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ref: https://jonnylangefeld.com/blog/how-to-write-a-go-api-part-3-testing-with-dockertest

var databasePort int = 5432
var lock sync.Mutex
var dbInstanceLock sync.Mutex

func nextPort() int {
	lock.Lock()
	databasePort = databasePort + 1

	defer lock.Unlock()
	return databasePort
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

func GenerateInstance(pool *dockertest.Pool) (*gorm.DB, *dockertest.Resource) {
	dbInstanceLock.Lock()
	defer dbInstanceLock.Unlock()

	var db *gorm.DB
	port := getAvailablePort()

	// Pull an image, create a container based on it and set all necessary parameters
	opts := dockertest.RunOptions{
		Repository:   "mdillon/postgis",
		Tag:          "latest",
		Env:          []string{"POSTGRES_PASSWORD=password"},
		ExposedPorts: []string{"5432"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5432": {
				{HostIP: "0.0.0.0", HostPort: port},
			},
		},
	}

	// Run the Docker container
	resource, err := pool.RunWithOptions(&opts)
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	guid, _ := uuid.NewRandom()
	databaseName := fmt.Sprintf("postgres%s", guid.String())

	// Exponential retry to connect to database while it is booting
	if err := pool.Retry(func() error {
		databaseConnStr := fmt.Sprintf("host=localhost port=%s user=postgres dbname=%s password=password sslmode=disable", port, databaseName)
		db, err = gorm.Open(postgres.Open(databaseConnStr), &gorm.Config{})
		if err != nil {
			log.Println("Database not ready yet (it is booting up, wait for a few tries)...")
			return err
		}

		// Tests if database is reachable
		dbinstance, err := db.DB()
		if err != nil {
			log.Println("Database instance cannot created")
			return err
		}
		return dbinstance.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	return db, resource
}
