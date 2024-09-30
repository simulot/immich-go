package cmd

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/simulot/immich-go/helpers/configuration"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/tzone"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ImmichServerFlags provides all flags to establish a connection with an immich server
type ImmichServerFlags struct {
	Server        string        // Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)
	API           string        // Immich api endpoint (http://container_ip:3301)
	Key           string        // API Key
	APITrace      bool          // Enable API call traces
	SkipSSL       bool          // Skip SSL Verification
	ClientTimeout time.Duration // Set the client request timeout
	DeviceUUID    string        // Set a device UUID
	DryRun        bool          // Protect the server from changes
	TimeZone      string        // Override default TZ

	// Jnl                *fileevent.Recorder    // Program's logger
	APITraceWriter     io.WriteCloser         // API tracer
	APITraceWriterName string                 // API trace log name
	Immich             immich.ImmichInterface // Immich client

	// NoUI               bool           // Disable user interface
	// DebugFileList      bool           // When true, the file argument is a file wile the list of Takeout files
}

// NewImmichServerFlagSet add server flags to the command cmd
func AddImmichServerFlagSet(cmd *cobra.Command, rootFlags *RootImmichFlags) *ImmichServerFlags {
	flags := ImmichServerFlags{}
	//  Server flags are available for sub commands
	flags.DeviceUUID, _ = os.Hostname()

	cmd.Flags().StringVarP(&flags.Server, "server", "s", flags.Server, "Immich server address (example http://your-ip:2283 or https://your-domain)")
	cmd.Flags().StringVar(&flags.API, "api", flags.API, "Immich api endpoint (example http://container_ip:3301)")
	cmd.Flags().StringVarP(&flags.Key, "key", "k", flags.Key, "API Key")
	cmd.Flags().BoolVar(&flags.APITrace, "api-trace", false, "Enable trace of api calls")
	cmd.Flags().BoolVar(&flags.SkipSSL, "skip-verify-ssl", false, "Skip SSL verification")
	cmd.Flags().DurationVar(&flags.ClientTimeout, "client-timeout", 5*time.Minute, "Set server calls timeout")
	cmd.Flags().StringVar(&flags.DeviceUUID, "device-uuid", flags.DeviceUUID, "Set a device UUID")
	cmd.Flags().BoolVar(&flags.DryRun, "dry-run", false, "Simulate all actions")
	cmd.Flags().StringVar(&flags.TimeZone, "time-zone", flags.TimeZone, "Override the system time zone")

	cmd.PostRunE = func(cmd *cobra.Command, args []string) error {
		return flags.Close(rootFlags)
	}
	return &flags
}

// Initialize the ImmichServerFlags flagset
// Validate the flags and initialize the server as required
// - fields fs.Server and fs.API are mutually exclusive
// - either fields fs.Server or fs.API must be given
// - Key is mandatory
func (SrvFlags *ImmichServerFlags) Open(rootFlags *RootImmichFlags) error {
	var err error
	// Bind the Server flag with the environment variable IMMICH_HOST
	if err := viper.BindEnv("server", "IMMICH_HOST"); err != nil {
		return err
	}
	SrvFlags.Server = viper.GetString("server")

	// Bind the Key flag with the environment variable IMMICH_KEY
	if err := viper.BindEnv("key", "IMMICH_KEY"); err != nil {
		return err
	}
	SrvFlags.Key = viper.GetString("key")

	if SrvFlags.Server != "" && SrvFlags.API != "" {
		err = errors.Join(err, errors.New(`flags 'server' and 'api' are mutually exclusive`))
	}
	if SrvFlags.Server == "" && SrvFlags.API == "" {
		err = errors.Join(err, errors.New(`either 'server' or 'api' flag must be provided`))
	}
	if SrvFlags.Key == "" {
		err = errors.Join(err, errors.New(`flag 'key' is mandatory`))
	}
	if SrvFlags.TimeZone != "" {
		if _, e := tzone.SetLocal(SrvFlags.TimeZone); e != nil {
			err = errors.Join(err, e)
		}
	}
	if err != nil {
		return err
	}

	rootFlags.Message(`Connection to the server %s`, SrvFlags.Server)
	SrvFlags.Immich, err = immich.NewImmichClient(SrvFlags.Server, SrvFlags.Key,
		immich.OptionVerifySSL(SrvFlags.SkipSSL),
		immich.OptionConnectionTimeout(SrvFlags.ClientTimeout),
		immich.OptionDryRun(SrvFlags.DryRun),
	)
	if err != nil {
		return err
	}
	if SrvFlags.API != "" {
		SrvFlags.Immich.SetEndPoint(SrvFlags.API)
	}
	if SrvFlags.DeviceUUID != "" {
		SrvFlags.Immich.SetDeviceUUID(SrvFlags.DeviceUUID)
	}

	if SrvFlags.APITrace {
		if SrvFlags.APITraceWriter == nil {
			if err := configuration.MakeDirForFile(rootFlags.LogFile); err != nil {
				return err
			}
			SrvFlags.APITraceWriterName = strings.TrimSuffix(rootFlags.LogFile, filepath.Ext(rootFlags.LogFile)) + ".trace.log"
			SrvFlags.APITraceWriter, err = os.OpenFile(SrvFlags.APITraceWriterName, os.O_CREATE|os.O_WRONLY, 0o664)
			if err != nil {
				return err
			}
			SrvFlags.Immich.EnableAppTrace(SrvFlags.APITraceWriter)
			rootFlags.Message("API log file: %s", SrvFlags.APITraceWriterName)
		}
	}

	ctx := rootFlags.Command.Context()
	if err := SrvFlags.Immich.PingServer(ctx); err != nil {
		return err
	}
	rootFlags.Message(`Server status: OK`)

	user, err := SrvFlags.Immich.ValidateConnection(ctx)
	if err != nil {
		return err
	}
	rootFlags.Message(`Connected, user: %s`, user.Email)

	return err
}

func (flags *ImmichServerFlags) Close(rootFlags *RootImmichFlags) error {
	if flags.APITrace {
		flags.APITraceWriter.Close()
		rootFlags.Message("Check the API traces files %s", flags.APITraceWriterName)
	}
	return rootFlags.Close()
}
