package login

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/config"
	"github.com/spf13/cobra"
)

// NewLoginCommand creates the `login` command for storing server credentials globally.
func NewLoginCommand(ctx context.Context, a *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store Immich server credentials in the global configuration",
		Long:  `Interactively prompts for server URL and API key, validates the connection, and saves credentials to ~/.config/immich-go/config.yaml.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(cmd.Context())
		},
	}
	return cmd
}

func runLogin(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)

	// Prompt for server URL
	fmt.Print("Immich server URL (e.g. http://192.168.1.100:2283 or https://immich.example.com): ")
	serverURL, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading server URL: %w", err)
	}
	serverURL = strings.TrimSpace(serverURL)
	serverURL = strings.TrimSuffix(serverURL, "/")

	if serverURL == "" {
		return fmt.Errorf("server URL cannot be empty")
	}

	// Validate server reachability
	fmt.Println("Checking server...")
	client, err := immich.NewImmichClient(serverURL, "")
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}
	if err := client.PingServer(ctx); err != nil {
		return fmt.Errorf("server not reachable at %s: %w", serverURL, err)
	}
	fmt.Println("Server is reachable.")

	// Prompt for API key
	fmt.Println("Enter your API key (Immich → User Settings → API Keys):")
	fmt.Print("API Key: ")
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading API key: %w", err)
	}
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Validate connection with the API key
	fmt.Println("Validating credentials...")
	client, err = immich.NewImmichClient(serverURL, apiKey)
	if err != nil {
		return fmt.Errorf("creating authenticated client: %w", err)
	}
	user, err := client.ValidateConnection(ctx)
	if err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}
	fmt.Printf("Authenticated as: %s\n", user.Email)

	// Save to global config
	if err := config.SaveGlobal(map[string]string{
		"server":  serverURL,
		"api-key": apiKey,
	}); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	configPath, _ := config.GlobalConfigPath()
	fmt.Printf("Credentials saved to %s\n", configPath)
	return nil
}
