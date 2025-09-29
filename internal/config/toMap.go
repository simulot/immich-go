package config

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func ToMap(cmd *cobra.Command) map[string]any {
	m := map[string]any{}
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		m[c.Name()] = ToMap(c)
	}
	if cmd.HasFlags() {
		fs := cmd.Flags()
		fs.VisitAll(func(f *pflag.Flag) {
			if f.Name != "config" && f.Name != "help" {
				m[f.Name] = f.Value
			}
		})
	}
	return m
}
