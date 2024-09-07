package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/simulot/immich-go/helpers/configuration"
	"github.com/simulot/immich-go/helpers/fileevent"
	"github.com/simulot/immich-go/immich"
	"github.com/spf13/cobra"
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

	Jnl                *fileevent.Recorder    // Program's logger
	APITraceWriter     io.WriteCloser         // API tracer
	APITraceWriterName string                 // API trace log name
	Immich             immich.ImmichInterface // Immich client
	DebugCounters      bool                   // Enable CSV action counters per file

	// TimeZone      string        // Override default TZ
	// NoUI               bool           // Disable user interface
	// DebugFileList      bool           // When true, the file argument is a file wile the list of Takeout files
}

// NewImmichServerFlagSet add server flags to the command cmd
func NewImmichServerFlagSet(cmd *cobra.Command, serverFlags *ImmichServerFlags) {
	//  Server flags are available for sub commands
	serverFlags.DeviceUUID, _ = os.Hostname()
	cmd.PersistentFlags().StringVar(&serverFlags.Server, "server", serverFlags.Server, "Immich server address (example http://<your-ip>:2283 or https://<your-domain>)")
	cmd.PersistentFlags().StringVar(&serverFlags.API, "api", serverFlags.API, "Immich api endpoint (example http://container_ip:3301)")
	cmd.PersistentFlags().StringVar(&serverFlags.Key, "key", serverFlags.Key, "API Key")
	cmd.PersistentFlags().BoolVar(&serverFlags.APITrace, "api-trace", false, "enable trace of api calls")
	cmd.PersistentFlags().BoolVar(&serverFlags.SkipSSL, "skip-verify-ssl", false, "Skip SSL verification")
	cmd.PersistentFlags().DurationVar(&serverFlags.ClientTimeout, "client-timeout", 5*time.Minute, "Set server calls timeout")
	cmd.PersistentFlags().StringVar(&serverFlags.DeviceUUID, "device-uuid", serverFlags.DeviceUUID, "Set a device UUID")

	// cmd.PersistentFlags().BoolVar(&serverFlags.DebugCounters, "debug-counters", false, "generate a CSV file with actions per handled files")
	// fs.StringVar(&serverFlags.TimeZone, "time-zone", serverFlags.TimeZone, "Override the system time zone")
	// fs.BoolVar(&serverFlags.Debug, "debug", false, "enable debug messages")
	// fs.BoolVar(&serverFlags.NoUI, "no-ui", false, "Disable the user interface")
}

// Initialize the ImmichServerFlags flagset
// Validate the flags and initialize the server as required
// - fields fs.Server and fs.API are mutually exclusive
// - either fields fs.Server or fs.API must be given
// - fs.Key is mandatory
func (app *ImmichServerFlags) Initialize(rootFlags *RootImmichFlags) error {
	var err error

	if app.Server != "" && app.API != "" {
		err = errors.Join(err, errors.New(`flags 'server' and 'api' are mutually exclusive`))
	}
	if app.Server == "" && app.API == "" {
		err = errors.Join(err, errors.New(`either 'server' or 'api' flag must be provided`))
	}
	if app.Key == "" {
		err = errors.Join(err, errors.New(`flag 'key' is mandatory`))
	}

	rootFlags.Log.Info(`Connection to the server ` + app.Server)

	app.Immich, err = immich.NewImmichClient(app.Server, app.Key, immich.OptionVerifySSL(app.SkipSSL), immich.OptionConnectionTimeout(app.ClientTimeout))
	if err != nil {
		return err
	}
	if app.API != "" {
		app.Immich.SetEndPoint(app.API)
	}
	if app.DeviceUUID != "" {
		app.Immich.SetDeviceUUID(app.DeviceUUID)
	}

	if app.APITrace {
		if app.APITraceWriter == nil {
			err := configuration.MakeDirForFile(rootFlags.LogFile)
			if err != nil {
				return err
			}
			app.APITraceWriterName = strings.TrimSuffix(rootFlags.LogFile, filepath.Ext(rootFlags.LogFile)) + ".trace.log"
			app.APITraceWriter, err = os.OpenFile(app.APITraceWriterName, os.O_CREATE|os.O_WRONLY, 0o664)
			if err != nil {
				return err
			}
			app.Immich.EnableAppTrace(app.APITraceWriter)
		}
	}

	ctx := rootFlags.Command.Context()
	err = app.Immich.PingServer(ctx)
	if err != nil {
		return err
	}
	rootFlags.Log.Info(`Server status: OK`)

	user, err := app.Immich.ValidateConnection(ctx)
	if err != nil {
		return err
	}
	rootFlags.Log.Info(fmt.Sprintf(
		`Connected, user: %s`,
		user.Email,
	))

	return err
}
