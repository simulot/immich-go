package shared

import (
	"github.com/simulot/immich-go/internal/filters"
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

	Filters []filters.Filter
}

func (so *StackOptions) RegisterFlags(flags *pflag.FlagSet) {
	flags.Var(&so.ManageHEICJPG, "manage-heic-jpeg", "Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG")
	flags.Var(&so.ManageRawJPG, "manage-raw-jpeg", "Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG")
	flags.Var(&so.ManageBurst, "manage-burst", "Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG")
	flags.BoolVar(&so.ManageEpsonFastFoto, "manage-epson-fastfoto", false, "Manage Epson FastFoto file (default: false)")
}
