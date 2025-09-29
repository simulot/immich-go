package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

// mockFlagDefiner is a mock implementation of FlagDefiner for testing.
type mockFlagDefiner struct {
	defineFunc func(fs *pflag.FlagSet)
}

func (m *mockFlagDefiner) DefineFlags(fs *pflag.FlagSet) {
	if m.defineFunc != nil {
		m.defineFunc(fs)
	}
}

func TestProcessCommand(t *testing.T) {
	t.Run("should process flags for the root command", func(t *testing.T) {
		// Arrange
		cm := New()
		rootCmd := &cobra.Command{Use: "root"}
		definer := &mockFlagDefiner{
			defineFunc: func(fs *pflag.FlagSet) {
				fs.String("test-flag", "", "a test flag")
			},
		}
		cm.Register(rootCmd, definer)
		cm.v.Set("test-flag", "viper-value")

		// Act
		err := cm.ProcessCommand(rootCmd)

		// Assert
		assert.NoError(t, err)
		flagValue, err := rootCmd.Flags().GetString("test-flag")
		assert.NoError(t, err)
		assert.Equal(t, "viper-value", flagValue)
	})

	t.Run("should process flags for a subcommand", func(t *testing.T) {
		// Arrange
		cm := New()
		rootCmd := &cobra.Command{Use: "root"}
		subCmd := &cobra.Command{Use: "sub"}
		rootCmd.AddCommand(subCmd)

		definer := &mockFlagDefiner{
			defineFunc: func(fs *pflag.FlagSet) {
				fs.String("sub-flag", "", "a sub test flag")
			},
		}
		cm.Register(subCmd, definer)
		cm.v.Set("sub.sub-flag", "viper-sub-value")

		// Act
		err := cm.ProcessCommand(rootCmd)

		// Assert
		assert.NoError(t, err)
		flagValue, err := subCmd.Flags().GetString("sub-flag")
		assert.NoError(t, err)
		assert.Equal(t, "viper-sub-value", flagValue)
	})

	t.Run("should not override flags set by command line", func(t *testing.T) {
		// Arrange
		cm := New()
		rootCmd := &cobra.Command{Use: "root"}
		definer := &mockFlagDefiner{
			defineFunc: func(fs *pflag.FlagSet) {
				// Use a flag that doesn't exist to avoid redefinition panic
				if fs.Lookup("test-flag") == nil {
					fs.String("test-flag", "", "a test flag")
				}
			},
		}
		cm.Register(rootCmd, definer)
		definer.DefineFlags(rootCmd.Flags())

		cm.v.Set("test-flag", "viper-value")

		// Simulate setting the flag via command line, which marks it as "Changed"
		err := rootCmd.Flags().Set("test-flag", "cli-value")
		assert.NoError(t, err)

		// Act
		err = cm.ProcessCommand(rootCmd)

		// Assert
		assert.NoError(t, err)
		flagValue, err := rootCmd.Flags().GetString("test-flag")
		assert.NoError(t, err)
		assert.Equal(t, "cli-value", flagValue)
	})

	t.Run("should process flags for nested subcommands", func(t *testing.T) {
		// Arrange
		cm := New()
		rootCmd := &cobra.Command{Use: "root"}
		subCmd1 := &cobra.Command{Use: "sub1"}
		subCmd2 := &cobra.Command{Use: "sub2"}
		rootCmd.AddCommand(subCmd1)
		subCmd1.AddCommand(subCmd2)

		definer := &mockFlagDefiner{
			defineFunc: func(fs *pflag.FlagSet) {
				fs.String("deep-flag", "", "a deep test flag")
			},
		}
		cm.Register(subCmd2, definer)
		cm.v.Set("sub1.sub2.deep-flag", "deep-value")

		// Act
		err := cm.ProcessCommand(rootCmd)

		// Assert
		assert.NoError(t, err)
		flagValue, err := subCmd2.Flags().GetString("deep-flag")
		assert.NoError(t, err)
		assert.Equal(t, "deep-value", flagValue)
	})

	t.Run("should only process commands once", func(t *testing.T) {
		// Arrange
		cm := New()
		rootCmd := &cobra.Command{Use: "root"}
		callCount := 0
		definer := &mockFlagDefiner{
			defineFunc: func(fs *pflag.FlagSet) {
				callCount++
				fs.String("test-flag", "", "a test flag")
			},
		}
		cm.Register(rootCmd, definer)

		// Act
		err1 := cm.ProcessCommand(rootCmd)
		err2 := cm.ProcessCommand(rootCmd)

		// Assert
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Equal(t, 1, callCount, "DefineFlags should only be called once")
		assert.True(t, cm.processed)
	})

	t.Run("should handle commands with no registered definers", func(t *testing.T) {
		// Arrange
		cm := New()
		rootCmd := &cobra.Command{Use: "root"}
		subCmd := &cobra.Command{Use: "sub"}
		rootCmd.AddCommand(subCmd)
		// No definers are registered

		// Act
		err := cm.ProcessCommand(rootCmd)

		// Assert
		assert.NoError(t, err)
	})
}

func TestInit(t *testing.T) {
	t.Run("should read config from a specific file", func(t *testing.T) {
		// Arrange
		cm := New()
		// Create a temporary config file
		content := []byte("test-key: test-value")
		dir := t.TempDir()
		configFile := filepath.Join(dir, "config.yaml")
		err := os.WriteFile(configFile, content, 0o644)
		assert.NoError(t, err)

		// Act
		err = cm.Init(configFile)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "test-value", cm.v.GetString("test-key"))
	})

	t.Run("should return error for non-existent specific config file", func(t *testing.T) {
		// Arrange
		cm := New()

		// Act
		err := cm.Init("non-existent-file.yaml")

		// Assert
		assert.Error(t, err)
	})

	t.Run("should search for default config file and not return error if not found", func(t *testing.T) {
		// Arrange
		cm := New()
		dir := t.TempDir()

		// Change working directory to temp dir
		oldWd, err := os.Getwd()
		assert.NoError(t, err)
		err = os.Chdir(dir)
		assert.NoError(t, err)
		defer os.Chdir(oldWd)

		// Act
		err = cm.Init("") // empty string should trigger search in new empty CWD

		// Assert
		assert.NoError(t, err)
	})

	t.Run("should search for default config file and load it if present", func(t *testing.T) {
		// Arrange
		cm := New()
		content := []byte("another-key: another-value")
		dir := t.TempDir()

		// Change working directory to temp dir
		oldWd, err := os.Getwd()
		assert.NoError(t, err)
		err = os.Chdir(dir)
		assert.NoError(t, err)
		defer os.Chdir(oldWd)

		configFile := filepath.Join(dir, "immich-go.yaml")
		err = os.WriteFile(configFile, content, 0o644)
		assert.NoError(t, err)

		// Act
		err = cm.Init("") // empty string should trigger search

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "another-value", cm.v.GetString("another-key"))
	})

	t.Run("should bind environment variables", func(t *testing.T) {
		// Arrange
		cm := New()
		t.Setenv("IMMICH_GO_ENV_VAR", "env-value")
		t.Setenv("IMMICH_GO_MY_FLAG", "env-flag-value")

		// Act
		err := cm.Init("")

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "env-value", cm.v.GetString("env-var"))
		assert.Equal(t, "env-flag-value", cm.v.GetString("my-flag"))
	})

	t.Run("should use env vars with correct precedence over config file", func(t *testing.T) {
		// Arrange
		cm := New()
		content := []byte("my-flag: file-value")
		dir := t.TempDir()
		configFile := filepath.Join(dir, "config.yaml")
		err := os.WriteFile(configFile, content, 0o644)
		assert.NoError(t, err)

		t.Setenv("IMMICH_GO_MY_FLAG", "env-value")

		// Act
		err = cm.Init(configFile)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "env-value", cm.v.GetString("my-flag"))
	})
}

func TestViperEnvBinding(t *testing.T) {
	// Setup a root command and a subcommand
	rootCmd := &cobra.Command{Use: "root", Run: func(cmd *cobra.Command, args []string) {}}
	uploadCmd := &cobra.Command{Use: "upload", Run: func(cmd *cobra.Command, args []string) {}}
	fromFolderCmd := &cobra.Command{Use: "from-folder", Run: func(cmd *cobra.Command, args []string) {}}

	rootCmd.AddCommand(uploadCmd)
	uploadCmd.AddCommand(fromFolderCmd)

	// Add a persistent flag to the upload command
	var server string
	uploadCmd.PersistentFlags().StringVar(&server, "server", "", "server address")

	var overwrite bool
	uploadCmd.Flags().BoolVar(&overwrite, "overwrite", false, "overwrite flag")

	var recursive bool
	fromFolderCmd.Flags().BoolVar(&recursive, "recursive", false, "recursive flag")

	// Set environment variables for the subcommand's flags
	os.Setenv("IMMICH_GO_UPLOAD_SERVER", "http://test.com")
	os.Setenv("IMMICH_GO_UPLOAD_OVERWRITE", "true")
	os.Setenv("IMMICH_GO_UPLOAD_FROM_FOLDER_RECURSIVE", "true")
	defer os.Unsetenv("IMMICH_GO_UPLOAD_SERVER")
	defer os.Unsetenv("IMMICH_GO_UPLOAD_OVERWRITE")
	defer os.Unsetenv("IMMICH_GO_UPLOAD_FROM_FOLDER_RECURSIVE")

	// Initialize the configuration manager
	cm := New()
	cm.Init("") // Initialize with no config file

	// Process the command
	err := cm.ProcessCommand(rootCmd)
	assert.NoError(t, err)

	// Simulate running the command
	rootCmd.SetArgs([]string{"upload", "from-folder"})
	rootCmd.Execute()

	// Check if the flag values were set from the environment variables
	assert.Equal(t, "http://test.com", server, "The persistent flag should be set from the subcommand's environment variable")
	assert.Equal(t, true, overwrite, "The local flag should be set from the environment variable")
	assert.Equal(t, true, recursive, "The nested subcommand flag should be set from the environment variable")
}
