package config

import (
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
