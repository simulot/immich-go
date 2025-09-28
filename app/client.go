package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/simulot/immich-go/immich"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/configuration"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Client struct {
	Server string // Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)
	// API                string                 // Immich api endpoint (http://container_ip:3301)
	APIKey      string // API Key
	AdminAPIKey string // API Key for admin

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
	OnServerErrors            cliflags.OnServerErrorsFlag // Behavior on server errors
	User                      immich.User                 // User info corresponding to the API key
	PauseImmichBackgroundJobs bool                        // Pause Immich background jobs
}

// add server flags to the command cmd
func AddClientFlags(ctx context.Context, cmd *cobra.Command, app *Application, dryRun bool) {
	client := app.Client()
	hostname, _ := os.Hostname()

	// Define flags with default values that will be overridden by Viper
	cmd.PersistentFlags().StringVarP(&client.Server, "server", "s", "", "Immich server address (example http://your-ip:2283 or https://your-domain)")
	cmd.PersistentFlags().StringVarP(&client.APIKey, "api-key", "k", "", "API Key")
	cmd.PersistentFlags().StringVar(&client.AdminAPIKey, "admin-api-key", "", "Admin's API Key for managing server's jobs")
	cmd.PersistentFlags().BoolVar(&client.APITrace, "api-trace", false, "Enable trace of api calls")
	cmd.PersistentFlags().BoolVar(&client.PauseImmichBackgroundJobs, "pause-immich-jobs", true, "Pause Immich background jobs during upload operations")
	cmd.PersistentFlags().BoolVar(&client.SkipSSL, "skip-verify-ssl", false, "Skip SSL verification")
	cmd.PersistentFlags().DurationVar(&client.ClientTimeout, "client-timeout", 20*time.Minute, "Set server calls timeout")
	cmd.PersistentFlags().StringVar(&client.DeviceUUID, "device-uuid", hostname, "Set a device UUID")
	cmd.PersistentFlags().BoolVar(&client.DryRun, "dry-run", dryRun, "Simulate all actions")
	cmd.PersistentFlags().StringVar(&client.TimeZone, "time-zone", "", "Override the system time zone")
	cmd.PersistentFlags().Var(&client.OnServerErrors, "on-server-errors", "Action to take on server errors, (stop|continue| <n> errors)")

	// Bind flags to Viper
	_ = viper.BindPFlag("server.url", cmd.PersistentFlags().Lookup("server"))
	_ = viper.BindPFlag("server.api_key", cmd.PersistentFlags().Lookup("api-key"))
	_ = viper.BindPFlag("server.admin_api_key", cmd.PersistentFlags().Lookup("admin-api-key"))
	_ = viper.BindPFlag("logging.api_trace", cmd.PersistentFlags().Lookup("api-trace"))
	_ = viper.BindPFlag("upload.pause_immich_jobs", cmd.PersistentFlags().Lookup("pause-immich-jobs"))
	_ = viper.BindPFlag("server.skip_ssl", cmd.PersistentFlags().Lookup("skip-verify-ssl"))
	_ = viper.BindPFlag("server.client_timeout", cmd.PersistentFlags().Lookup("client-timeout"))
	_ = viper.BindPFlag("server.device_uuid", cmd.PersistentFlags().Lookup("device-uuid"))
	_ = viper.BindPFlag("upload.dry_run", cmd.PersistentFlags().Lookup("dry-run"))
	_ = viper.BindPFlag("server.time_zone", cmd.PersistentFlags().Lookup("time-zone"))
	_ = viper.BindPFlag("server.on_server_errors", cmd.PersistentFlags().Lookup("on-server-errors"))

	cmd.PersistentPreRunE = ChainRunEFunctions(cmd.PersistentPreRunE, LoadConfigurationIntoClient, ctx, cmd, app)
	cmd.PersistentPreRunE = ChainRunEFunctions(cmd.PersistentPreRunE, OpenClient, ctx, cmd, app)
	cmd.PersistentPostRunE = ChainRunEFunctions(cmd.PersistentPostRunE, CloseClient, ctx, cmd, app)
}

func LoadConfigurationIntoClient(ctx context.Context, cmd *cobra.Command, app *Application) error {
	// Load configuration values from Viper into the client struct
	client := app.Client()

	// Get configuration from Viper
	config, err := configuration.GetConfiguration()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get configuration info for logging
	configInfo := configuration.GetConfigurationInfo()

	// Log configuration source information
	if configInfo.ConfigFile != "" {
		app.Log().Message("Using configuration file: %s", configInfo.ConfigFile)
	} else {
		app.Log().Message("No configuration file found. Using defaults, environment variables, and command-line flags.")
	}

	// Track which values come from which sources for logging
	var configSources []string

	// Apply configuration values to client (only if not already set by flags)
	if client.Server == "" {
		client.Server = config.Server.URL
		if config.Server.URL != "" {
			configSources = append(configSources, "server.url from config")
		}
	} else {
		configSources = append(configSources, "server from CLI flag")
	}

	if client.APIKey == "" {
		client.APIKey = config.Server.APIKey
		if config.Server.APIKey != "" {
			configSources = append(configSources, "server.api_key from config")
		}
	} else {
		configSources = append(configSources, "api-key from CLI flag")
	}

	if client.AdminAPIKey == "" {
		client.AdminAPIKey = config.Server.AdminAPIKey
		if config.Server.AdminAPIKey != "" {
			configSources = append(configSources, "server.admin_api_key from config")
		}
	} else {
		configSources = append(configSources, "admin-api-key from CLI flag")
	}

	if client.DeviceUUID == "" {
		hostname, _ := os.Hostname()
		if config.Server.DeviceUUID != "" {
			client.DeviceUUID = config.Server.DeviceUUID
			configSources = append(configSources, "server.device_uuid from config")
		} else {
			client.DeviceUUID = hostname
			configSources = append(configSources, "device-uuid from hostname default")
		}
	} else {
		configSources = append(configSources, "device-uuid from CLI flag")
	}

	if client.TimeZone == "" {
		client.TimeZone = config.Server.TimeZone
		if config.Server.TimeZone != "" {
			configSources = append(configSources, "server.time_zone from config")
		}
	} else {
		configSources = append(configSources, "time-zone from CLI flag")
	}

	// Apply boolean and other settings from config
	if !cmd.PersistentFlags().Changed("skip-verify-ssl") {
		client.SkipSSL = config.Server.SkipSSL
		if config.Server.SkipSSL {
			configSources = append(configSources, "skip-verify-ssl=true from config")
		}
	} else {
		configSources = append(configSources, "skip-verify-ssl from CLI flag")
	}

	if !cmd.PersistentFlags().Changed("client-timeout") {
		client.ClientTimeout = config.Server.ClientTimeout
		if config.Server.ClientTimeout > 0 {
			configSources = append(configSources, fmt.Sprintf("client-timeout=%v from config", config.Server.ClientTimeout))
		}
	} else {
		configSources = append(configSources, "client-timeout from CLI flag")
	}

	if !cmd.PersistentFlags().Changed("api-trace") {
		client.APITrace = config.Logging.APITrace
		if config.Logging.APITrace {
			configSources = append(configSources, "api-trace=true from config")
		}
	} else {
		configSources = append(configSources, "api-trace from CLI flag")
	}

	if !cmd.PersistentFlags().Changed("pause-immich-jobs") {
		client.PauseImmichBackgroundJobs = config.Upload.PauseImmichJobs
		configSources = append(configSources, fmt.Sprintf("pause-immich-jobs=%v from config", config.Upload.PauseImmichJobs))
	} else {
		configSources = append(configSources, "pause-immich-jobs from CLI flag")
	}

	if !cmd.PersistentFlags().Changed("dry-run") {
		client.DryRun = config.Upload.DryRun
		if config.Upload.DryRun {
			configSources = append(configSources, "dry-run=true from config")
		}
	} else {
		configSources = append(configSources, "dry-run from CLI flag")
	}

	if !cmd.PersistentFlags().Changed("on-server-errors") && config.Server.OnServerErrors != "" {
		_ = client.OnServerErrors.Set(config.Server.OnServerErrors)
		configSources = append(configSources, fmt.Sprintf("on-server-errors=%s from config", config.Server.OnServerErrors))
	} else if cmd.PersistentFlags().Changed("on-server-errors") {
		configSources = append(configSources, "on-server-errors from CLI flag")
	}

	// Log final resolved configuration values (with sensitive data masked)
	app.Log().Message("Configuration sources resolved:")
	for _, source := range configSources {
		app.Log().Message("  - %s", source)
	}

	// Log final resolved values
	serverURL := client.Server
	if serverURL == "" {
		serverURL = "<not set>"
	}

	apiKey := client.APIKey
	if apiKey != "" {
		if len(apiKey) > 8 {
			apiKey = apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
		} else {
			apiKey = "****"
		}
	} else {
		apiKey = "<not set>"
	}

	app.Log().Message("Final configuration values:")
	app.Log().Message("  - Server URL: %s", serverURL)
	app.Log().Message("  - API Key: %s", apiKey)
	app.Log().Message("  - Device UUID: %s", client.DeviceUUID)
	app.Log().Message("  - Client Timeout: %v", client.ClientTimeout)
	app.Log().Message("  - Skip SSL: %v", client.SkipSSL)
	app.Log().Message("  - API Trace: %v", client.APITrace)
	app.Log().Message("  - Dry Run: %v", client.DryRun)
	app.Log().Message("  - Pause Immich Jobs: %v", client.PauseImmichBackgroundJobs)
	if client.TimeZone != "" {
		app.Log().Message("  - Time Zone: %s", client.TimeZone)
	}

	return nil
}

func OpenClient(ctx context.Context, cmd *cobra.Command, app *Application) error {
	client := app.Client()
	return client.Open(ctx, app)
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

func (client *Client) Close() error {
	if client.DryRun {
		client.ClientLog.Info("Dry-run mode enabled. No changes were made to the server.")
	}
	return nil
}
