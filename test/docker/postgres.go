package docker

import (
	"fmt"
	"log"

	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const databasePort = 5433

func generatePostgresInstance(pool *dockertest.Pool) (*gorm.DB, *dockertest.Resource) {
	var db *gorm.DB
	// Pull an image, create a container based on it and set all necessary parameters
	opts := dockertest.RunOptions{
		Repository:   "mdillon/postgis",
		Tag:          "latest",
		Env:          []string{"POSTGRES_PASSWORD=mysecretpassword"},
		ExposedPorts: []string{"5432"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5432": {
				{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", databasePort)},
			},
		},
	}

	// Run the Docker container
	resource, err := pool.RunWithOptions(&opts)
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	// Exponential retry to connect to database while it is booting
	if err := pool.Retry(func() error {
		databaseConnStr := fmt.Sprintf("host=localhost port=%d user=postgres dbname=postgres password=mysecretpassword sslmode=disable", databasePort)
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
