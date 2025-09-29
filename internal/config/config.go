package config

import "github.com/spf13/pflag"

// FlagDefiner is an interface for modules that define their own CLI flags.
type FlagDefiner interface {
	// DefineFlags adds flags to the provided FlagSet.
	DefineFlags(flags *pflag.FlagSet)
}
