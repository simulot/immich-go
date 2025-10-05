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
	"github.com/spf13/pflag"
)

// Client represents an Immich server client with configuration and connection management.
// It handles authentication, API communication, and various client-side settings.
type Client struct {
	Server                    string         `mapstructure:"server" json:"server" toml:"server" yaml:"server"`                                                                                         // Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)
	APIKey                    string         `mapstructure:"api_key" json:"api_key" toml:"api_key" yaml:"api_key"`                                                                                     // API Key
	AdminAPIKey               string         `mapstructure:"admin_api_key" json:"admin_api_key" toml:"admin_api_key" yaml:"admin_api_key"`                                                             // API Key for admin
	APITrace                  bool           `mapstructure:"api_trace" json:"api_trace" toml:"api_trace" yaml:"api_trace"`                                                                             // Enable API call traces
	SkipSSL                   bool           `mapstructure:"skip_ssl" json:"skip_ssl" toml:"skip_ssl" yaml:"skip_ssl"`                                                                                 // Skip SSL Verification
	ClientTimeout             time.Duration  `mapstructure:"client_timeout" json:"client_timeout" toml:"client_timeout" yaml:"client_timeout"`                                                         // Set the client request timeout
	DeviceUUID                string         `mapstructure:"device_uuid" json:"device_uuid" toml:"device_uuid" yaml:"device_uuid"`                                                                     // Set a device UUID
	TimeZone                  string         `mapstructure:"time_zone" json:"time_zone" toml:"time_zone" yaml:"time_zone"`                                                                             // Override default TZ
	APITraceWriter            io.WriteCloser `mapstructure:"api_trace_writer" json:"api_trace_writer" toml:"api_trace_writer" yaml:"api_trace_writer"`                                                 // API tracer
	APITraceWriterName        string         `mapstructure:"api_trace_writer_name" json:"api_trace_writer_name" toml:"api_trace_writer_name" yaml:"api_trace_writer_name"`                             // API trace log name
	User                      immich.User    `mapstructure:"user" json:"user" toml:"user" yaml:"user"`                                                                                                 // User info corresponding to the API key
	PauseImmichBackgroundJobs bool           `mapstructure:"pause_immich_background_jobs" json:"pause_immich_background_jobs" toml:"pause_immich_background_jobs" yaml:"pause_immich_background_jobs"` // Pause Immich background jobs

	TZ          *time.Location         // Time zone to use
	Immich      immich.ImmichInterface // Immich client
	AdminImmich immich.ImmichInterface // Immich client for admin
	ClientLog   *slog.Logger           // Logger
	app         *Application
	DryRun      bool // Protect the server from changes
}

// RegisterFlags adds client-related command-line flags to the provided flag set.
// These flags control server connection, authentication, and client behavior.
func (client *Client) RegisterFlags(flags *pflag.FlagSet, prefix string) {
	client.DeviceUUID, _ = os.Hostname()

	if prefix == "" {
		flags.StringVarP(&client.Server, prefix+"server", "s", client.Server, "Immich server address (example http://your-ip:2283 or https://your-domain)")
		flags.StringVarP(&client.APIKey, prefix+"api-key", "k", "", "API Key")
	} else {
		flags.StringVar(&client.Server, prefix+"server", client.Server, "Immich server address (example http://your-ip:2283 or https://your-domain)")
		flags.StringVar(&client.APIKey, prefix+"api-key", "", "API Key")
	}
	flags.StringVar(&client.AdminAPIKey, prefix+"admin-api-key", "", "Admin's API Key for managing server's jobs")
	flags.BoolVar(&client.APITrace, prefix+"api-trace", false, "Enable trace of api calls")
	flags.BoolVar(&client.PauseImmichBackgroundJobs, prefix+"pause-immich-jobs", true, "Pause Immich background jobs during upload operations")
	flags.BoolVar(&client.SkipSSL, prefix+"skip-verify-ssl", false, "Skip SSL verification")
	flags.DurationVar(&client.ClientTimeout, prefix+"client-timeout", 20*time.Minute, "Set server calls timeout")
	flags.StringVar(&client.DeviceUUID, prefix+"device-uuid", client.DeviceUUID, "Set a device UUID")
	flags.BoolVar(&client.DryRun, prefix+"dry-run", false, "Simulate all actions")
	flags.StringVar(&client.TimeZone, prefix+"time-zone", client.TimeZone, "Override the system time zone")
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
			err = log.sLevel.UnmarshalText([]byte(strings.ToUpper(log.Level))) // TODO implement a flag.Value
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
	if client.APITraceWriter != nil {
		client.APITraceWriter.Close()
		client.app.log.Message("Check the API-TRACE file: %s", client.APITraceWriterName)
	}
	return nil
}
