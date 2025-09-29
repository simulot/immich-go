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

	// Bind and apply viper values
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		key := getViperKey(cmd, f)
		cm.v.BindPFlag(key, f)
		if !f.Changed && cm.v.IsSet(key) {
			val := cm.v.Get(key)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})

	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		key := getViperKey(cmd, f)
		cm.v.BindPFlag(key, f)
		if !f.Changed && cm.v.IsSet(key) {
			val := cm.v.Get(key)
			cmd.PersistentFlags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})

	// Recurse for subcommands
	for _, c := range cmd.Commands() {
		cm.processCommand(c)
	}
}

func getViperKey(cmd *cobra.Command, f *pflag.Flag) string {
	isInherited := cmd.Parent() != nil && cmd.Parent().PersistentFlags().Lookup(f.Name) != nil
	if isInherited {
		// Use parent path
		path := []string{}
		for c := cmd.Parent(); c.Parent() != nil; c = c.Parent() {
			path = append([]string{c.Name()}, path...)
		}
		if len(path) > 0 {
			return strings.Join(path, ".") + "." + f.Name
		}
		return f.Name
	} else {
		// Use current path
		path := []string{}
		for c := cmd; c.Parent() != nil; c = c.Parent() {
			path = append([]string{c.Name()}, path...)
		}
		if len(path) > 0 {
			return strings.Join(path, ".") + "." + f.Name
		}
		return f.Name
	}
}

func (cm *ConfigurationManager) Save(fileName string) error {
	return cm.v.WriteConfigAs(fileName)
}
