package configuration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Configuration struct {
	APIURL    string `json:",omitempty"`
	ServerURL string `json:",omitempty"`
	APIKey    string

	// Server configuration
	Server ServerConfig `mapstructure:"server" yaml:"server" json:"server" toml:"server"`

	// Upload configuration
	Upload UploadConfig `mapstructure:"upload" yaml:"upload" json:"upload" toml:"upload"`

	// Archive configuration
	Archive ArchiveConfig `mapstructure:"archive" yaml:"archive" json:"archive" toml:"archive"`

	// Stack configuration
	Stack StackConfig `mapstructure:"stack" yaml:"stack" json:"stack" toml:"stack"`

	// Logging configuration
	Logging LogConfig `mapstructure:"logging" yaml:"logging" json:"logging" toml:"logging"`

	// UI configuration
	UI UIConfig `mapstructure:"ui" yaml:"ui" json:"ui" toml:"ui"`
}

// ServerConfig contains server-specific configuration
type ServerConfig struct {
	URL            string        `mapstructure:"url" yaml:"url" json:"url" toml:"url"`
	APIKey         string        `mapstructure:"api_key" yaml:"api_key" json:"api_key" toml:"api_key"`
	AdminAPIKey    string        `mapstructure:"admin_api_key" yaml:"admin_api_key" json:"admin_api_key" toml:"admin_api_key"`
	SkipSSL        bool          `mapstructure:"skip_ssl" yaml:"skip_ssl" json:"skip_ssl" toml:"skip_ssl"`
	ClientTimeout  time.Duration `mapstructure:"client_timeout" yaml:"client_timeout" json:"client_timeout" toml:"client_timeout"`
	DeviceUUID     string        `mapstructure:"device_uuid" yaml:"device_uuid" json:"device_uuid" toml:"device_uuid"`
	TimeZone       string        `mapstructure:"time_zone" yaml:"time_zone" json:"time_zone" toml:"time_zone"`
	OnServerErrors string        `mapstructure:"on_server_errors" yaml:"on_server_errors" json:"on_server_errors" toml:"on_server_errors"`
}

// UploadConfig contains upload-specific configuration
type UploadConfig struct {
	ConcurrentUploads int    `mapstructure:"concurrent_uploads" yaml:"concurrent_uploads" json:"concurrent_uploads" toml:"concurrent_uploads"`
	DryRun            bool   `mapstructure:"dry_run" yaml:"dry_run" json:"dry_run" toml:"dry_run"`
	CreateAlbums      bool   `mapstructure:"create_albums" yaml:"create_albums" json:"create_albums" toml:"create_albums"`
	GooglePhotos      bool   `mapstructure:"google_photos" yaml:"google_photos" json:"google_photos" toml:"google_photos"`
	Partner           string `mapstructure:"partner" yaml:"partner" json:"partner" toml:"partner"`
	Album             string `mapstructure:"album" yaml:"album" json:"album" toml:"album"`
	ImportFromAlbum   string `mapstructure:"import_from_album" yaml:"import_from_album" json:"import_from_album" toml:"import_from_album"`
	SkipVerify        bool   `mapstructure:"skip_verify" yaml:"skip_verify" json:"skip_verify" toml:"skip_verify"`
	DiscardArchived   bool   `mapstructure:"discard_archived" yaml:"discard_archived" json:"discard_archived" toml:"discard_archived"`
	KeepUntitled      bool   `mapstructure:"keep_untitled" yaml:"keep_untitled" json:"keep_untitled" toml:"keep_untitled"`
	ArchiveTimeout    string `mapstructure:"archive_timeout" yaml:"archive_timeout" json:"archive_timeout" toml:"archive_timeout"`
	DateRange         string `mapstructure:"date_range" yaml:"date_range" json:"date_range" toml:"date_range"`
	Overwrite         bool   `mapstructure:"overwrite" yaml:"overwrite" json:"overwrite" toml:"overwrite"`
	PauseImmichJobs   bool   `mapstructure:"pause_immich_jobs" yaml:"pause_immich_jobs" json:"pause_immich_jobs" toml:"pause_immich_jobs"`
}

// ArchiveConfig contains archive-specific configuration
type ArchiveConfig struct {
	DateRange string `mapstructure:"date_range" yaml:"date_range" json:"date_range" toml:"date_range"`
	DryRun    bool   `mapstructure:"dry_run" yaml:"dry_run" json:"dry_run" toml:"dry_run"`
}

// StackConfig contains stack-specific configuration
type StackConfig struct {
	ManageHEICJPEG      string `mapstructure:"manage_heic_jpeg" yaml:"manage_heic_jpeg" json:"manage_heic_jpeg" toml:"manage_heic_jpeg"`
	ManageRawJPEG       string `mapstructure:"manage_raw_jpeg" yaml:"manage_raw_jpeg" json:"manage_raw_jpeg" toml:"manage_raw_jpeg"`
	ManageBurst         string `mapstructure:"manage_burst" yaml:"manage_burst" json:"manage_burst" toml:"manage_burst"`
	ManageEpsonFastFoto bool   `mapstructure:"manage_epson_fastfoto" yaml:"manage_epson_fastfoto" json:"manage_epson_fastfoto" toml:"manage_epson_fastfoto"`
	DateRange           string `mapstructure:"date_range" yaml:"date_range" json:"date_range" toml:"date_range"`
}

// LogConfig contains logging-specific configuration
type LogConfig struct {
	Level    string `mapstructure:"level" yaml:"level" json:"level" toml:"level"`
	File     string `mapstructure:"file" yaml:"file" json:"file" toml:"file"`
	APITrace bool   `mapstructure:"api_trace" yaml:"api_trace" json:"api_trace" toml:"api_trace"`
}

// UIConfig contains UI-specific configuration
type UIConfig struct {
	NoUI bool `mapstructure:"no_ui" yaml:"no_ui" json:"no_ui" toml:"no_ui"`
}

// DefaultConfigFile return the default configuration file name
// Return a local file when the default UserHomeDir can't be determined,
func DefaultConfigFile() string {
	config, err := os.UserConfigDir()
	if err != nil {
		// $XDG_CONFIG_HOME nor $HOME is set
		// return current
		return "./immich-go.json"
	}
	return filepath.Join(config, "immich-go", "immich-go.json")
}

// DefaultConfigDir returns the default configuration directory
func DefaultConfigDir() string {
	config, err := os.UserConfigDir()
	if err != nil {
		return "."
	}
	return filepath.Join(config, "immich-go")
}

// DefaultConfigName returns the base configuration file name without extension
func DefaultConfigName() string {
	return "immich-go"
}

// InitializeConfig initializes Viper with configuration file, environment variables, and defaults
func InitializeConfig(configFile string) error {
	if configFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(configFile)
	} else {
		// Search for config in default locations
		viper.AddConfigPath(DefaultConfigDir())
		viper.AddConfigPath(".")
		viper.SetConfigName(DefaultConfigName())
		viper.SetConfigType("yaml") // Default to YAML, but will try others
	}

	// Environment variable support
	viper.SetEnvPrefix("IMMICHGO")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	// Set default values
	setDefaults()

	// Read the config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error
		} else {
			// Config file was found but another error was produced
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	return nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	hostname, _ := os.Hostname()
	viper.SetDefault("server.device_uuid", hostname)
	viper.SetDefault("server.skip_ssl", false)
	viper.SetDefault("server.client_timeout", "20m")
	viper.SetDefault("server.on_server_errors", "stop")

	// Upload defaults
	viper.SetDefault("upload.concurrent_uploads", runtime.NumCPU())
	viper.SetDefault("upload.dry_run", false)
	viper.SetDefault("upload.overwrite", false)
	viper.SetDefault("upload.pause_immich_jobs", true)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.api_trace", false)

	// UI defaults
	viper.SetDefault("ui.no_ui", false)
}

// GetConfiguration returns the current configuration
func GetConfiguration() (*Configuration, error) {
	var config Configuration
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode configuration: %w", err)
	}

	// Handle legacy configuration migration
	if config.Server.URL == "" && config.ServerURL != "" {
		config.Server.URL = config.ServerURL
	}
	if config.Server.APIKey == "" && config.APIKey != "" {
		config.Server.APIKey = config.APIKey
	}

	return &config, nil
}

// GetConfigurationInfo returns information about the configuration sources
func GetConfigurationInfo() ConfigurationInfo {
	info := ConfigurationInfo{
		ConfigFile: viper.ConfigFileUsed(),
		Sources:    make(map[string]interface{}),
	}

	// Get all settings that have been set
	for key, value := range viper.AllSettings() {
		info.Sources[key] = value
	}

	return info
}

// ConfigurationInfo holds information about configuration sources and resolved values
type ConfigurationInfo struct {
	ConfigFile string                 `json:"config_file"`
	Sources    map[string]interface{} `json:"sources"`
}

// WriteConfigFile creates a sample configuration file
func WriteConfigFile(filename string) error {
	config := &Configuration{}

	// Set example values
	config.Server.URL = "http://your-immich-server:2283"
	config.Server.APIKey = "your-api-key-here"
	config.Server.AdminAPIKey = "your-admin-api-key-here"
	config.Server.SkipSSL = false
	config.Server.ClientTimeout = 20 * time.Minute
	config.Server.TimeZone = "Local"
	config.Server.OnServerErrors = "stop"

	config.Upload.ConcurrentUploads = 4
	config.Upload.DryRun = false
	config.Upload.Overwrite = false
	config.Upload.PauseImmichJobs = true

	config.Logging.Level = "info"
	config.Logging.APITrace = false
	config.Logging.File = DefaultLogFile()

	config.UI.NoUI = false

	// Create directory if it doesn't exist
	if err := MakeDirForFile(filename); err != nil {
		return err
	}

	// Write the configuration based on file extension
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".json":
		return writeJSONConfig(config, filename)
	case ".yaml", ".yml":
		return writeYAMLConfig(filename)
	case ".toml":
		return writeTOMLConfig(filename)
	default:
		// Default to YAML
		return writeYAMLConfig(filename)
	}
}

func writeJSONConfig(config *Configuration, filename string) error {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(config)
}

func writeYAMLConfig(filename string) error {
	yamlContent := `# Immich-go Configuration File
# This file contains default settings for immich-go
# You can override any of these settings with command-line flags or environment variables

server:
  # Immich server URL (required)
  url: "http://your-immich-server:2283"
  
  # API key for accessing Immich (required)
  api_key: "your-api-key-here"
  
  # Admin API key for managing server jobs (optional, defaults to api_key if not set)
  admin_api_key: "your-admin-api-key-here"
  
  # Skip SSL certificate verification (default: false)
  skip_ssl: false
  
  # Client timeout for API requests (default: 20m)
  client_timeout: "20m"
  
  # Device UUID for uploads (default: hostname)
  device_uuid: ""
  
  # Time zone override (default: system timezone)
  time_zone: ""
  
  # Action to take on server errors: stop, continue, or number of errors to tolerate
  on_server_errors: "stop"

upload:
  # Number of concurrent upload workers (1-20, default: 4)
  concurrent_uploads: 4
  
  # Enable dry-run mode (no actual uploads)
  dry_run: false
  
  # Always overwrite files on server with local versions
  overwrite: false
  
  # Pause Immich background jobs during uploads
  pause_immich_jobs: true

archive:
  # Enable dry-run mode (no actual archiving)
  dry_run: false
  
  # Date range filter for archiving photos
  date_range: ""

stack:
  # Manage coupled HEIC and JPEG files
  # Options: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG
  manage_heic_jpeg: "NoStack"
  
  # Manage coupled RAW and JPEG files  
  # Options: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG
  manage_raw_jpeg: "NoStack"
  
  # Manage burst photos
  # Options: NoStack, Stack, StackKeepRaw, StackKeepJPEG
  manage_burst: "NoStack"
  
  # Manage Epson FastFoto file groups
  manage_epson_fastfoto: false
  
  # Date range filter for stacking photos
  date_range: ""

logging:
  # Log level: debug, info, warn, error
  level: "info"
  
  # Log file path (empty means no file logging)
  file: ""
  
  # Enable API call tracing
  api_trace: false

ui:
  # Disable the user interface
  no_ui: false
`

	return os.WriteFile(filename, []byte(yamlContent), 0o600)
}

func writeTOMLConfig(filename string) error {
	tomlContent := `# Immich-go Configuration File
# This file contains default settings for immich-go
# You can override any of these settings with command-line flags or environment variables

[server]
# Immich server URL (required)
url = "http://your-immich-server:2283"

# API key for accessing Immich (required)
api_key = "your-api-key-here"

# Admin API key for managing server jobs (optional, defaults to api_key if not set)
admin_api_key = "your-admin-api-key-here"

# Skip SSL certificate verification (default: false)
skip_ssl = false

# Client timeout for API requests (default: 20m)
client_timeout = "20m"

# Device UUID for uploads (default: hostname)
device_uuid = ""

# Time zone override (default: system timezone)
time_zone = ""

# Action to take on server errors: stop, continue, or number of errors to tolerate
on_server_errors = "stop"

[upload]
# Number of concurrent upload workers (1-20, default: 4)
concurrent_uploads = 4

# Enable dry-run mode (no actual uploads)
dry_run = false

# Always overwrite files on server with local versions
overwrite = false

# Pause Immich background jobs during uploads
pause_immich_jobs = true

[archive]
# Enable dry-run mode (no actual archiving)
dry_run = false

# Date range filter for archiving photos
date_range = ""

[stack]
# Manage coupled HEIC and JPEG files
# Options: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG
manage_heic_jpeg = "NoStack"

# Manage coupled RAW and JPEG files  
# Options: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG
manage_raw_jpeg = "NoStack"

# Manage burst photos
# Options: NoStack, Stack, StackKeepRaw, StackKeepJPEG
manage_burst = "NoStack"

# Manage Epson FastFoto file groups
manage_epson_fastfoto = false

# Date range filter for stacking photos
date_range = ""

[logging]
# Log level: debug, info, warn, error
level = "info"

# Log file path (empty means no file logging)
file = ""

# Enable API call tracing
api_trace = false

[ui]
# Disable the user interface
no_ui = false
`

	return os.WriteFile(filename, []byte(tomlContent), 0o600)
}

// ConfigRead the configuration in file name
func ConfigRead(name string) (Configuration, error) {
	f, err := os.Open(name)
	if err != nil {
		return Configuration{}, err
	}
	defer f.Close()
	var c Configuration
	err = json.NewDecoder(f).Decode(&c)
	if err != nil {
		return Configuration{}, err
	}
	return c, nil
}

// Write the configuration in the file name
// Create the needed sub directories as needed
func (c Configuration) Write(name string) error {
	d, _ := filepath.Split(name)
	if d != "" {
		err := os.MkdirAll(d, 0o700)
		if err != nil {
			return err
		}
	}
	f, err := os.OpenFile(name, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o700)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(c)
}

// DefaultLogDir give the default log file
// Return the current dir when $HOME not $XDG_CACHE_HOME are not set
func DefaultLogFile() string {
	f := time.Now().Format("immich-go_2006-01-02_15-04-05.log")
	d, err := os.UserCacheDir()
	if err != nil {
		return f
	}
	return filepath.Join(d, "immich-go", f)
}

// MakeDirForFile create all dirs to write the given file
func MakeDirForFile(f string) error {
	dir := filepath.Dir(f)
	return os.MkdirAll(dir, 0o700)
}
