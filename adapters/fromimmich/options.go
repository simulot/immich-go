package fromimmich

import (
	"time"

	"github.com/simulot/immich-go/app"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/spf13/cobra"
)

type FromImmichFlags struct {
	DateRange     cliflags.DateRange // get assets only within this date range  (fromat: YYYY-MM-DD,YYYY-MM-DD)
	Albums        []string           // get assets only from those albums
	Tags          []string           // get assets only with those tags
	WithArchived  bool               // get archived assets too
	WithTrashed   bool               // get trashed assets too
	Favorite      bool               // get only favorite assets
	MinimalRating int                // get only assets with a rating greater or equal to this value
	Make          string             // get only assets with this make
	Model         string             // get only assets with this model
	client        app.Client         // client to use for the import
}

func (o *FromImmichFlags) AddFromImmichFlags(cmd *cobra.Command, parent *cobra.Command) {
	// cmd.Flags().StringVar(&o.Make, "from-make", "", "Get only assets with this make")
	// cmd.Flags().StringVar(&o.Model, "from-model", "", "Get only assets with this model")
	cmd.Flags().StringSliceVar(&o.Albums, "from-album", nil, "Get assets only from those albums, can be used multiple times")
	// cmd.Flags().StringSliceVar(&o.Tags, "from-tags", nil, "Get assets only with those tags")
	cmd.Flags().Var(&o.DateRange, "from-date-range", "Get assets only within this date range (fromat: YYYY[-MM[-DD[,YYYY-MM-DD]]])")
	// cmd.Flags().BoolVar(&o.WithArchived, "from-archived", false, "Get archived assets too")
	// cmd.Flags().BoolVar(&o.WithTrashed, "from-trashed", false, "Get trashed assets too")
	// cmd.Flags().BoolVar(&o.Favorite, "from-favorite", false, "Get only favorite assets")
	// cmd.Flags().IntVar(&o.MinimalRating, "from-minimal-rating", 0, "Get only assets with a rating greater or equal to this value")

	cmd.Flags().StringVar(&o.client.Server, "from-server", o.client.Server, "Immich server address (example http://your-ip:2283 or https://your-domain)")
	cmd.Flags().StringVar(&o.client.APIKey, "from-api-key", "", "API Key")
	cmd.Flags().BoolVar(&o.client.APITrace, "from-api-trace", false, "Enable trace of api calls")
	cmd.Flags().BoolVar(&o.client.SkipSSL, "from-skip-verify-ssl", false, "Skip SSL verification")
	cmd.Flags().DurationVar(&o.client.ClientTimeout, "from-client-timeout", 5*time.Minute, "Set server calls timeout")
}
