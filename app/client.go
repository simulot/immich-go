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
	"github.com/simulot/immich-go/internal/configuration"
	"github.com/spf13/cobra"
)

// add server flags to the command cmd
func AddClientFlags(ctx context.Context, cmd *cobra.Command, app *Application, dryRun bool) {
	client := app.Client()
	client.DeviceUUID, _ = os.Hostname()

	cmd.PersistentFlags().StringVarP(&client.Server, "server", "s", client.Server, "Immich server address (example http://your-ip:2283 or https://your-domain)")
	cmd.PersistentFlags().StringVarP(&client.APIKey, "api-key", "k", "", "API Key")
	cmd.PersistentFlags().BoolVar(&client.APITrace, "api-trace", false, "Enable trace of api calls")
	cmd.PersistentFlags().BoolVar(&client.SkipSSL, "skip-verify-ssl", false, "Skip SSL verification")
	cmd.PersistentFlags().DurationVar(&client.ClientTimeout, "client-timeout", 5*time.Minute, "Set server calls timeout")
	cmd.PersistentFlags().StringVar(&client.DeviceUUID, "device-uuid", client.DeviceUUID, "Set a device UUID")
	cmd.PersistentFlags().BoolVar(&client.DryRun, "dry-run", dryRun, "Simulate all actions")
	cmd.PersistentFlags().StringVar(&client.TimeZone, "time-zone", client.TimeZone, "Override the system time zone")

	cmd.PersistentPreRunE = ChainRunEFunctions(cmd.PersistentPreRunE, OpenClient, ctx, cmd, app)
	cmd.PersistentPostRunE = ChainRunEFunctions(cmd.PersistentPostRunE, CloseClient, ctx, cmd, app)
}

func OpenClient(ctx context.Context, cmd *cobra.Command, app *Application) error {
	var err error
	client := app.Client()
	log := app.Log()

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
	if log.File != "" {
		if log.mainWriter == nil {
			err := configuration.MakeDirForFile(log.File)
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
			// prepare the trace file name
			client.APITraceWriterName = strings.TrimSuffix(log.File, filepath.Ext(log.File)) + ".trace.log"
		}
	}

	err = client.Initialize(ctx, app)
	if err != nil {
		return err
	}

	err = client.Open(ctx)
	if err != nil {
		return err
	}

	if client.APITrace {
		if client.APITraceWriter == nil {
			client.APITraceWriter, err = os.OpenFile(client.APITraceWriterName, os.O_CREATE|os.O_WRONLY, 0o664)
			if err != nil {
				return err
			}
			client.Immich.EnableAppTrace(client.APITraceWriter)
		}
		app.log.Message("Check the API-TRACE file: %s", client.APITraceWriterName)
	}
	return nil
}

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

type Client struct {
	Server string // Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)
	// API                string                 // Immich api endpoint (http://container_ip:3301)
	APIKey             string                 // API Key
	APITrace           bool                   // Enable API call traces
	SkipSSL            bool                   // Skip SSL Verification
	ClientTimeout      time.Duration          // Set the client request timeout
	DeviceUUID         string                 // Set a device UUID
	DryRun             bool                   // Protect the server from changes
	TimeZone           string                 // Override default TZ
	TZ                 *time.Location         // Time zone to use
	APITraceWriter     io.WriteCloser         // API tracer
	APITraceWriterName string                 // API trace log name
	Immich             immich.ImmichInterface // Immich client
	ClientLog          *slog.Logger           // Logger
}

func (client *Client) Initialize(ctx context.Context, app *Application) error {
	var joinedErr error

	// If the client isn't yet initialized
	if client.Immich == nil {
		if client.Server == "" {
			joinedErr = errors.Join(joinedErr, errors.New("missing the parameter --server, Immich server address (http://<your-ip>:2283 or https://<your-domain>)"))
		}
		if client.APIKey == "" {
			joinedErr = errors.Join(joinedErr, errors.New("missing the parameter --api-key, Immich API key"))
		}

		if client.APITrace {
			client.APITraceWriterName = strings.TrimSuffix(app.Log().File, filepath.Ext(app.Log().File)) + ".trace.log"
		}
		if joinedErr != nil {
			return joinedErr
		}
	}
	client.ClientLog = app.log.Logger
	return nil
}

func (client *Client) Open(ctx context.Context) error {
	var err error

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

	if client.DeviceUUID != "" {
		client.Immich.SetDeviceUUID(client.DeviceUUID)
	}

	if client.APITrace {
		if client.APITraceWriter == nil {
			client.APITraceWriter, err = os.OpenFile(client.APITraceWriterName, os.O_CREATE|os.O_WRONLY, 0o664)
			if err != nil {
				return err
			}
			client.Immich.EnableAppTrace(client.APITraceWriter)
		}
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
	client.ClientLog.Info(fmt.Sprintf("Connected, user: %s", user.Email))
	if client.DryRun {
		client.ClientLog.Info("Dry-run mode enabled. No changes will be made to the server.")
	}
	return nil
}

func (client *Client) Close() error {
	if client.DryRun {
		client.ClientLog.Info("Dry-run mode enabled. No changes were made to the server.")
	}
	return nil
}
