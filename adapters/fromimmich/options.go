package fromimmich

import (
	"time"

	"github.com/simulot/immich-go/app"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TODO add Locked folder option
type FromImmichFlags struct {
	Albums          []string                `mapstructure:"from-albums" yaml:"from-albums" json:"from-albums" toml:"from-albums"`                                 // get assets only from those albums
	Tags            []string                `mapstructure:"from-tags" yaml:"from-tags" json:"from-tags" toml:"from-tags"`                                         // get assets only with those tags
	People          []string                `mapstructure:"from-people" yaml:"from-people" json:"from-people" toml:"from-people"`                                 // get assets only with those people
	IncludePartners bool                    `mapstructure:"from-partners" yaml:"from-partners" json:"from-partners" toml:"from-partners"`                         // get partner's assets as well
	OnlyArchived    bool                    `mapstructure:"from-archived" yaml:"from-archived" json:"from-archived" toml:"from-archived"`                         // get only archived assets
	OnlyTrashed     bool                    `mapstructure:"from-trash" yaml:"from-trash" json:"from-trash" toml:"from-trash"`                                     // get only trashed assets
	OnlyFavorite    bool                    `mapstructure:"from-favorite" yaml:"from-favorite" json:"from-favorite" toml:"from-favorite"`                         // get only favorite assets
	OnlyNoAlbum     bool                    `mapstructure:"from-no-album" yaml:"from-no-album" json:"from-no-album" toml:"from-no-album"`                         // get only assets that are not in any album
	MinimalRating   int                     `mapstructure:"from-minimal-rating" yaml:"from-minimal-rating" json:"from-minimal-rating" toml:"from-minimal-rating"` // get only assets with a rating greater or equal to this value
	Make            string                  `mapstructure:"from-make" yaml:"from-make" json:"from-make" toml:"from-make"`                                         // get only assets with this make
	Model           string                  `mapstructure:"from-model" yaml:"from-model" json:"from-model" toml:"from-model"`                                     // get only assets with this model
	Country         string                  `mapstructure:"from-country" yaml:"from-country" json:"from-country" toml:"from-country"`                             // get only assets from the country
	State           string                  `mapstructure:"from-state" yaml:"from-state" json:"from-state" toml:"from-state"`                                     // get only assets from this state
	City            string                  `mapstructure:"from-city" yaml:"from-city" json:"from-city" toml:"from-city"`                                         // get only assets from this city
	client          app.Client              // client to use for the import
	InclusionFlags  cliflags.InclusionFlags `mapstructure:",squash" yaml:",inline" json:",inline" toml:",inline"` // controls the file extensions to be included in the import process.
	albumIDs        []string
	tagIDs          []string
	peopleIDs       []string
}

func (o *FromImmichFlags) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.Make, "from-make", "", "Get only assets with this make")
	flags.StringVar(&o.Model, "from-model", "", "Get only assets with this model")
	flags.StringVar(&o.Country, "from-country", "", "Get only assets from this country")
	flags.StringVar(&o.State, "from-state", "", "Get only assets from this state")
	flags.StringVar(&o.City, "from-city", "", "Get only assets from this city")
	flags.StringSliceVar(&o.Albums, "from-albums", nil, "Get assets only from those albums, can be used multiple times")
	flags.StringSliceVar(&o.Tags, "from-tags", nil, "Get assets only with those tags, can be used multiple times")
	flags.StringSliceVar(&o.People, "from-people", nil, "Get assets only with those people, can be used multiple times")
	flags.BoolVar(&o.IncludePartners, "from-partners", false, "Get partner's assets as well")
	flags.BoolVar(&o.OnlyArchived, "from-archived", false, "Get only archived assets")
	flags.BoolVar(&o.OnlyTrashed, "from-trash", false, "Get only trashed assets")
	flags.BoolVar(&o.OnlyFavorite, "from-favorite", false, "Get only favorite assets")
	flags.BoolVar(&o.OnlyNoAlbum, "from-no-album", false, "Get only assets that are not in any album")
	flags.IntVar(&o.MinimalRating, "from-minimal-rating", 0, "Get only assets with a rating greater or equal to this value")
	flags.StringVar(&o.client.Server, "from-server", o.client.Server, "Immich server address (example http://your-ip:2283 or https://your-domain)")
	flags.StringVar(&o.client.APIKey, "from-api-key", "", "API Key")
	flags.BoolVar(&o.client.APITrace, "from-api-trace", false, "Enable trace of api calls")
	flags.BoolVar(&o.client.SkipSSL, "from-skip-verify-ssl", false, "Skip SSL verification")
	flags.DurationVar(&o.client.ClientTimeout, "from-client-timeout", 20*time.Minute, "Set server calls timeout")
}

func (o *FromImmichFlags) AddFromImmichFlags(cmd *cobra.Command, parent *cobra.Command) {
	o.InclusionFlags.RegisterFlags(cmd.Flags(), "from-")
	o.RegisterFlags(cmd.Flags())
}
