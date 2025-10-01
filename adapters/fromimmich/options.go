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
	Albums          []string                // get assets only from those albums
	Tags            []string                // get assets only with those tags
	People          []string                // get assets only with those people
	IncludePartners bool                    // get partner's assets as well
	OnlyArchived    bool                    // get only archived assets
	OnlyTrashed     bool                    // get only trashed assets
	OnlyFavorite    bool                    // get only favorite assets
	OnlyNoAlbum     bool                    // get only assets that are not in any album
	MinimalRating   int                     // get only assets with a rating greater or equal to this value
	Make            string                  // get only assets with this make
	Model           string                  // get only assets with this model
	Country         string                  // get only assets from the country
	State           string                  // get only assets from this state
	City            string                  // get only assets from this city
	client          app.Client              // client to use for the import
	InclusionFlags  cliflags.InclusionFlags // controls the file extensions to be included in the import process.
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
