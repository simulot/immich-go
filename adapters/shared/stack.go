package shared

import (
	"errors"
	"fmt"

	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/groups"
	"github.com/spf13/pflag"
)

type StackOptions struct {
	// ManageHEICJPG determines whether to manage HEIC to JPG conversion options.
	ManageHEICJPG filters.HeicJpgFlag `mapstructure:"manage_heic_jpg" json:"manage_heic_jpg" toml:"manage_heic_jpg" yaml:"manage_heic_jpg"`

	// ManageRawJPG determines how to manage raw and JPEG files.
	ManageRawJPG filters.RawJPGFlag `mapstructure:"manage_raw_jpg" json:"manage_raw_jpg" toml:"manage_raw_jpg" yaml:"manage_raw_jpg"`

	// BurstFlag determines how to manage burst photos.
	ManageBurst filters.BurstFlag `mapstructure:"manage_burst" json:"manage_burst" toml:"manage_burst" yaml:"manage_burst"`

	// ManageEpsonFastFoto enables the management of Epson FastFoto files.
	ManageEpsonFastFoto bool `mapstructure:"manage_epson_fast_foto" json:"manage_epson_fast_foto" toml:"manage_epson_fast_foto" yaml:"manage_epson_fast_foto"`

	Filters  []filters.Filter
	Groupers []groups.Grouper
}

func (so *StackOptions) RegisterFlags(flags *pflag.FlagSet) {
	flags.Var(&so.ManageHEICJPG, "manage-heic-jpeg", "Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG")
	flags.Var(&so.ManageRawJPG, "manage-raw-jpeg", "Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG")
	flags.Var(&so.ManageBurst, "manage-burst", "Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG")
	flags.BoolVar(&so.ManageEpsonFastFoto, "manage-epson-fastfoto", false, "Manage Epson FastFoto file (default: false)")
}

// ReadFlags sets so from the --manage-* flags found in flags. upload registers them as persistent
// flags; once a sub-command's command line is parsed, its flag set holds them with the values
// given on the command line or in the configuration. A source adapter reads them there when it
// runs: when its own flags are registered, it is not yet a sub-command of upload. A flag that is
// not in flags (archive has none) leaves its option as it is.
func (so *StackOptions) ReadFlags(flags *pflag.FlagSet) error {
	// A flag set of our own, bound to so's fields: the flag names are RegisterFlags' only.
	keep := *so
	own := pflag.NewFlagSet("", pflag.ContinueOnError)
	so.RegisterFlags(own)
	*so = keep // RegisterFlags sets the bool to its default
	var err error
	own.VisitAll(func(f *pflag.Flag) {
		if given := flags.Lookup(f.Name); given != nil {
			if e := f.Value.Set(given.Value.String()); e != nil {
				err = errors.Join(err, fmt.Errorf("--%s: %w", f.Name, e))
			}
		}
	})
	return err
}
