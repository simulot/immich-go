package fromimmich

import (
	"time"

	"github.com/simulot/immich-go/app"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TODO add Locked option
type FromImmichFlags struct {
	DateRange      cliflags.DateRange      // get assets only within this date range  (fromat: YYYY-MM-DD,YYYY-MM-DD)
	Albums         []string                // get assets only from those albums
	Tags           []string                // get assets only with those tags
	OnlyArchived   bool                    // get only archived assets
	OnlyTrashed    bool                    // get only trashed assets
	OnlyFavorite   bool                    // get only favorite assets
	MinimalRating  int                     // get only assets with a rating greater or equal to this value
	Make           string                  // get only assets with this make
	Model          string                  // get only assets with this model
	client         app.Client              // client to use for the import
	InclusionFlags cliflags.InclusionFlags // controls the file extensions to be included in the import process.
}

var _ config.FlagDefiner = (*FromImmichFlags)(nil)

func (o *FromImmichFlags) DefineFlags(flags *pflag.FlagSet) {
	// flags.StringVar(&o.Make, "from-make", "", "Get only assets with this make")
	// flags.StringVar(&o.Model, "from-model", "", "Get only assets with this model")
	// flags.StringSliceVar(&o.Albums, "from-albums", nil, "Get assets only from those albums, can be used multiple times")
	// flags.StringSliceVar(&o.Tags, "from-tags", nil, "Get assets only with those tags, can be used multiple times")
	flags.Var(&o.DateRange, "from-date-range", "Get assets only within this date range (fromat: YYYY[-MM[-DD[,YYYY-MM-DD]]])")
	flags.BoolVar(&o.OnlyArchived, "from-archived", false, "Get only archived assets")
	flags.BoolVar(&o.OnlyTrashed, "from-trash", false, "Get only trashed assets")
	flags.BoolVar(&o.OnlyFavorite, "from-favorite", false, "Get only favorite assets")
	flags.IntVar(&o.MinimalRating, "from-minimal-rating", 0, "Get only assets with a rating greater or equal to this value")
	flags.StringVar(&o.client.Server, "from-server", o.client.Server, "Immich server address (example http://your-ip:2283 or https://your-domain)")
	flags.StringVar(&o.client.APIKey, "from-api-key", "", "API Key")
	flags.BoolVar(&o.client.APITrace, "from-api-trace", false, "Enable trace of api calls")
	flags.BoolVar(&o.client.SkipSSL, "from-skip-verify-ssl", false, "Skip SSL verification")
	flags.DurationVar(&o.client.ClientTimeout, "from-client-timeout", 20*time.Minute, "Set server calls timeout")
}

func (o *FromImmichFlags) AddFromImmichFlags(cmd *cobra.Command, parent *cobra.Command) {
	o.DefineFlags(cmd.Flags())
	cliflags.AddInclusionFlags(cmd, &o.InclusionFlags)
}
