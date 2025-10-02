package config

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// CommandVisitor is a function type for visiting commands during traversal
type CommandVisitor func(cmd *cobra.Command, path []string) map[string]any

// TraverseCommands recursively traverses cobra commands and calls the visitor for each,
// building nested maps for subcommands
func TraverseCommands(cmd *cobra.Command, path []string, visitor CommandVisitor) map[string]any {
	m := visitor(cmd, path)
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		subM := TraverseCommands(c, append(path, c.Name()), visitor)
		if len(subM) > 0 {
			m[c.Name()] = subM
		}
	}
	return m
}

func ToMap(cmd *cobra.Command) map[string]any {
	return TraverseCommands(cmd, []string{}, func(cmd *cobra.Command, path []string) map[string]any {
		m := map[string]any{}
		if len(path) == 0 && cmd.HasPersistentFlags() {
			fs := cmd.PersistentFlags()
			fs.VisitAll(func(f *pflag.Flag) {
				if f.Name != "config" && f.Name != "help" {
					m[f.Name] = f.Value
				}
			})
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
	})
}
