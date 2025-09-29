package config

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type ConfigurationManager struct {
	v         *viper.Viper
	command   *cobra.Command
	definers  map[*cobra.Command][]FlagDefiner
	processed bool
}

func New() *ConfigurationManager {
	return &ConfigurationManager{
		v:        viper.New(),
		definers: map[*cobra.Command][]FlagDefiner{},
	}
}

func (cm *ConfigurationManager) Init(cfgFile string) error {
	if cfgFile != "" {
		cm.v.SetConfigFile(cfgFile)
	} else {
		cm.v.AddConfigPath(".")
		cm.v.SetConfigName("immich-go")
	}

	cm.v.SetEnvPrefix("IMMICH_GO")
	cm.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	cm.v.AutomaticEnv()

	if err := cm.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}
	return nil
}

func (cm *ConfigurationManager) Register(cmd *cobra.Command, definers ...FlagDefiner) {
	cm.definers[cmd] = definers
}

func (cm *ConfigurationManager) ProcessCommand(cmd *cobra.Command) error {
	if cm.processed {
		return nil
	}
	cm.command = cmd
	cm.processCommand(cmd)
	cm.processed = true
	return nil
}

func (cm *ConfigurationManager) processCommand(cmd *cobra.Command) {
	// get the definers for the command
	definers, ok := cm.definers[cmd]
	if ok {
		// let them register flags
		for _, d := range definers {
			d.DefineFlags(cmd.Flags())
		}
	}

	// bind flags of the current command
	cm.v.BindPFlags(cmd.Flags())

	// transfer viper values to flags
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Create a nested key for viper
		key := getViperKey(cmd, f.Name)

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && cm.v.IsSet(key) {
			val := cm.v.Get(key)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})

	// Let's do the same for sub commands
	for _, c := range cmd.Commands() {
		cm.processCommand(c)
	}
}

func getViperKey(cmd *cobra.Command, flagName string) string {
	path := []string{}
	for c := cmd; c.Parent() != nil; c = c.Parent() {
		path = append([]string{c.Name()}, path...)
	}
	if len(path) > 0 {
		return strings.Join(path, ".") + "." + flagName
	}
	return flagName
}

func (cm *ConfigurationManager) Save(fileName string) error {
	return cm.v.WriteConfigAs(fileName)
}
