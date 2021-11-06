package vault

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/vault/api"
)

// ref: https://github.com/PolarGeospatialCenter/dockertest

const vaultTestRootToken = "701432d1-00e7-7c94-10c4-8450ab3c4b31"

type Instance struct {
	config *api.Config
	*Container
}

var activeContainer *Container

func init() {
	os.Setenv("VAULT_TOKEN", vaultTestRootToken)
}

func Run(ctx context.Context) (*Instance, error) {
	instance := &Instance{
		Container: &Container{
			Image: "docker.io/library/vault",
			Cmd:   []string{"vault", "server", "-dev", "-dev-root-token-id", vaultTestRootToken, "-dev-listen-address", "0.0.0.0:8200"},
		},
	}

	err := instance.Container.Run(ctx)
	if err != nil {
		return nil, err
	}

	port, err := instance.Container.GetPort(ctx, "8200/tcp")
	if err != nil {
		return nil, err
	}

	instance.config = api.DefaultConfig()
	instance.config.Address = fmt.Sprintf("http://0.0.0.0:%s", port)

	timeout := time.After(10 * time.Second)
	checkInterval := time.Tick(50 * time.Millisecond)
	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("vault failed to start after 10 seconds")
		case <-checkInterval:
			if instance.running() {
				return instance, nil
			}
		}
	}
}

func (i *Instance) running() bool {
	c := http.Client{}
	resp, err := c.Get(fmt.Sprintf("%s/v1/sys/seal-status", i.Config().Address))
	return err == nil && resp.StatusCode == 200
}

func (i *Instance) Config() *api.Config {
	return i.config
}

func (i *Instance) RootToken() string {
	return vaultTestRootToken
}

func GenerateInstance(data map[string]interface{}) (*api.Client, *api.Secret) {
	ctx := context.Background()
	instance, err := Run(ctx)
	if err != nil {
		log.Fatalf("unable to create vault instance: %v", err)
	}
	// defer instance.Container.Stop(ctx)

	client, err := api.NewClient(instance.Config())
	if err != nil {
		defer instance.Container.Stop(ctx)
		log.Fatalf("Unable to create vault client: %v", err)
	}

	client.SetToken(instance.RootToken())

	newdata := make(map[string]interface{})
	newdata["data"] = data
	_, err = client.Logical().Write("secret/data/test", newdata)
	if err != nil {
		defer instance.Container.Stop(ctx)
		log.Fatalf("Unable to write test value to vault: %v", err)
	}

	secret, err := client.Logical().Read("secret/data/test")
	if err != nil {
		defer instance.Container.Stop(ctx)
		log.Fatalf("Unable to read test value from vault: %v", err)
	}

	activeContainer = instance.Container

	return client, secret
}

func RemoveInstance() error {
	return activeContainer.Stop(context.Background())
}
