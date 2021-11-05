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
	instance.config.Address = fmt.Sprintf("http://127.0.0.1:%s", port)

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

func GenerateInstance(data map[string]interface{}) *api.Client {
	ctx := context.Background()
	instance, err := Run(ctx)
	if err != nil {
		log.Fatalf("unable to create vault instance: %v", err)
	}
	defer instance.Container.Stop(ctx)

	client, err := api.NewClient(instance.Config())
	if err != nil {
		log.Fatalf("Unable to create vault client: %v", err)
	}

	client.SetToken(instance.RootToken())

	log.Printf("%v", data)
	_, err = client.Logical().Write("medicplus/medicplus/development", data)
	if err != nil {
		log.Fatalf("Unable to write test value to vault: %v", err)
	}

	secret, err := client.Logical().Read("medicplus/medicplus/development")
	if err != nil {
		log.Fatalf("Unable to read test value from vault: %v", err)
	}

	log.Printf("Returned: %v", secret)
	resultData, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		log.Fatalf("Invalid data returned from vault: %v", secret.Data)
	}
	if testString, ok := resultData["test"].(string); !ok || testString != "Hello Vault!" {
		log.Fatalf("Wrong value returned from vault: %v", testString)
	}

	activeContainer = instance.Container

	return client
}

func RemoveInstance() error {
	return activeContainer.Stop(context.Background())
}
