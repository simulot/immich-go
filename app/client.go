// Package app provides the main application logic for immich-go,
// including client management for connecting to Immich servers.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/simulot/immich-go/immich"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Client represents an Immich server client with configuration and connection management.
// It handles authentication, API communication, and various client-side settings.
type Client struct {
	Server                    string                      // Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)
	APIKey                    string                      // API Key
	AdminAPIKey               string                      // API Key for admin
	APITrace                  bool                        // Enable API call traces
	SkipSSL                   bool                        // Skip SSL Verification
	ClientTimeout             time.Duration               // Set the client request timeout
	DeviceUUID                string                      // Set a device UUID
	DryRun                    bool                        // Protect the server from changes
	TimeZone                  string                      // Override default TZ
	TZ                        *time.Location              // Time zone to use
	APITraceWriter            io.WriteCloser              // API tracer
	APITraceWriterName        string                      // API trace log name
	Immich                    immich.ImmichInterface      // Immich client
	AdminImmich               immich.ImmichInterface      // Immich client for admin
	ClientLog                 *slog.Logger                // Logger
	OnServerErrors            cliflags.OnServerErrorsFlag // Behavior on server errors, (stop|continue| <n> errors)
	User                      immich.User                 // User info corresponding to the API key
	PauseImmichBackgroundJobs bool                        // Pause Immich background jobs
}

// RegisterFlags adds client-related command-line flags to the provided flag set.
// These flags control server connection, authentication, and client behavior.
func (client *Client) RegisterFlags(flags *pflag.FlagSet) {
	client.DeviceUUID, _ = os.Hostname()

	flags.StringVarP(&client.Server, "server", "s", client.Server, "Immich server address (example http://your-ip:2283 or https://your-domain)")
	flags.StringVarP(&client.APIKey, "api-key", "k", "", "API Key")
	flags.StringVar(&client.AdminAPIKey, "admin-api-key", "", "Admin's API Key for managing server's jobs")
	flags.BoolVar(&client.APITrace, "api-trace", false, "Enable trace of api calls")

	flags.BoolVar(&client.PauseImmichBackgroundJobs, "pause-immich-jobs", true, "Pause Immich background jobs during upload operations")
	flags.BoolVar(&client.SkipSSL, "skip-verify-ssl", false, "Skip SSL verification")
	flags.DurationVar(&client.ClientTimeout, "client-timeout", 20*time.Minute, "Set server calls timeout")
	flags.StringVar(&client.DeviceUUID, "device-uuid", client.DeviceUUID, "Set a device UUID")
	flags.BoolVar(&client.DryRun, "dry-run", false, "Simulate all actions")
	flags.StringVar(&client.TimeZone, "time-zone", client.TimeZone, "Override the system time zone")
	flags.Var(&client.OnServerErrors, "on-server-errors", "Action to take on server errors, (stop|continue| <n> errors)")
}

// AddClientFlags registers client flags for a command and sets up pre/post run hooks
// for opening and closing the client connection. The dryRun parameter overrides the client's dry-run setting.
func AddClientFlags(ctx context.Context, cmd *cobra.Command, app *Application, dryRun bool) {
	client := app.Client()
	client.DryRun = dryRun
	client.RegisterFlags(cmd.PersistentFlags())

	cmd.PersistentPreRunE = ChainRunEFunctions(cmd.PersistentPreRunE, OpenClient, ctx, cmd, app)
	cmd.PersistentPostRunE = ChainRunEFunctions(cmd.PersistentPostRunE, CloseClient, ctx, cmd, app)
}

// OpenClient is a pre-run hook that opens the client connection.
// It validates configuration and establishes connection to the Immich server.
func OpenClient(ctx context.Context, cmd *cobra.Command, app *Application) error {
	client := app.Client()
	return client.Open(ctx, app)
}

// CloseClient is a post-run hook that closes the client connection.
// It handles cleanup of API trace writers and logs relevant information.
func CloseClient(ctx context.Context, cmd *cobra.Command, app *Application) error {
	if app.Client() != nil {
		if app.Client().APITraceWriter != nil {
			app.Client().APITraceWriter.Close()
			app.log.Message("Check the API-TRACE file: %s", app.Client().APITraceWriterName)
		}
		return app.Client().Close()
	}
	return nil
}

// Open establishes a connection to the Immich server.
// It validates configuration, sets up logging, creates client instances,
// and performs initial server validation including ping and authentication.
func (client *Client) Open(ctx context.Context, app *Application) error {
	var err error

	if client.Server != "" {
		client.Server = strings.TrimSuffix(client.Server, "/")
	}
	if client.TimeZone != "" {
		// Load the specified timezone
		client.TZ, err = time.LoadLocation(client.TimeZone)
		if err != nil {
			return err
		}
	}

	// Plug the journal on the Log
	log := app.Log()
	if log.File != "" {
		if log.mainWriter == nil {
			dir := filepath.Dir(log.File)
			err := os.MkdirAll(dir, 0o700)
			if err != nil {
				return err
			}
			f, err := os.OpenFile(log.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o664)
			if err != nil {
				return err
			}
			err = log.sLevel.UnmarshalText([]byte(strings.ToUpper(log.Level)))
			if err != nil {
				return err
			}
			log.setHandlers(f, nil)
		}
	}

	client.ClientLog = app.log.Logger

	var joinedErr error
	if client.APITrace {
		err = log.OpenAPITrace()
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
		}
	}

	// check server's parameters
	if client.APIKey == "" && client.AdminAPIKey != "" {
		client.APIKey = client.AdminAPIKey
		client.ClientLog.Warn("The parameter --api-key is empty. Using the admin's API key for for photos upload")
	} else if client.AdminAPIKey == "" && client.APIKey != "" {
		client.AdminAPIKey = client.APIKey
	}

	if client.Immich == nil {
		if client.Server == "" {
			joinedErr = errors.Join(joinedErr, errors.New("missing the parameter --server, Immich server address (http://<your-ip>:2283 or https://<your-domain>)"))
		}
		if client.APIKey == "" && client.AdminAPIKey == "" {
			joinedErr = errors.Join(joinedErr, errors.New("missing the parameter --api-key and/or --admin-api-key, Immich API keys"))
		}

		if joinedErr != nil {
			return joinedErr
		}
	}

	client.ClientLog.Info("Connection to the server " + client.Server)
	client.Immich, err = immich.NewImmichClient(
		client.Server,
		client.APIKey,
		immich.OptionVerifySSL(client.SkipSSL),
		immich.OptionConnectionTimeout(client.ClientTimeout),
		immich.OptionDryRun(client.DryRun),
	)
	if err != nil {
		return err
	}

	if t := log.APITracer(); t != nil {
		client.Immich.EnableAppTrace(t.DecorateRT)
	}

	adminTime := max(client.ClientTimeout, 10*time.Second)
	client.AdminImmich, err = immich.NewImmichClient(
		client.Server,
		client.AdminAPIKey,
		immich.OptionVerifySSL(client.SkipSSL),
		immich.OptionConnectionTimeout(adminTime),
		// no trace pulling job status
	)
	if err != nil {
		return err
	}

	if client.DeviceUUID != "" {
		client.Immich.SetDeviceUUID(client.DeviceUUID)
	}

	err = client.Immich.PingServer(ctx)
	if err != nil {
		return err
	}
	client.ClientLog.Info("Server status: OK")

	user, err := client.Immich.ValidateConnection(ctx)
	if err != nil {
		return err
	}
	client.User = user

	about, err := client.Immich.GetAboutInfo(ctx)
	if err != nil {
		return err
	}
	client.ClientLog.Info("Server information:", "version", about.Version)

	client.ClientLog.Info(fmt.Sprintf("Connected, user: %s, ID: %s", user.Email, user.ID))

	if client.DryRun {
		client.ClientLog.Info("Dry-run mode enabled. No changes will be made to the server.")
	}
	return nil
}

// Close cleans up the client connection.
// It logs dry-run status and performs any necessary cleanup operations.
func (client *Client) Close() error {
	if client.DryRun {
		client.ClientLog.Info("Dry-run mode enabled. No changes were made to the server.")
	}
	return nil
}
