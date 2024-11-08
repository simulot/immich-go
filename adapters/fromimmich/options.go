package fromimmich

import (
	"time"

	"github.com/simulot/immich-go/commands/application"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/spf13/cobra"
)

type FromImmichFlags struct {
	DateRange     cliflags.DateRange // get assets only within this date range  (format: YYYY-MM-DD,YYYY-MM-DD)
	Albums        []string           // get assets only from those albums
	Tags          []string           // get assets only with those tags
	WithArchived  bool               // get archived assets too
	WithTrashed   bool               // get trashed assets too
	Favorite      bool               // get only favorite assets
	MinimalRating int                // get only assets with a rating greater or equal to this value
	Make          string             // get only assets with this make
	Model         string             // get only assets with this model
	client        application.Client // client to use for the import
}

func (o *FromImmichFlags) AddFromImmichFlags(cmd *cobra.Command, parent *cobra.Command) {
	cmd.Flags().StringVar(&o.Make, "make", "", "Get only assets with this make")
	cmd.Flags().StringVar(&o.Model, "model", "", "Get only assets with this model")
	cmd.Flags().Var(&o.DateRange, "date-range", "Get assets only within this date range (format: YYYY[-MM[-DD[,YYYY-MM-DD]]])")
	cmd.Flags().StringSliceVar(&o.Albums, "albums", nil, "Get assets only from those albums")
	cmd.Flags().StringSliceVar(&o.Tags, "tags", nil, "Get assets only with those tags")
	cmd.Flags().BoolVar(&o.WithArchived, "archived", false, "Get archived assets too")
	cmd.Flags().BoolVar(&o.WithTrashed, "trashed", false, "Get trashed assets too")
	cmd.Flags().BoolVar(&o.Favorite, "favorite", false, "Get only favorite assets")
	cmd.Flags().IntVar(&o.MinimalRating, "minimal-rating", 0, "Get only assets with a rating greater or equal to this value")

	cmd.Flags().StringVarP(&o.client.Server, "from-server", "s", o.client.Server, "Immich server address (example http://your-ip:2283 or https://your-domain)")
	cmd.Flags().StringVarP(&o.client.APIKey, "from-api-key", "k", "", "API Key")
	cmd.Flags().BoolVar(&o.client.APITrace, "from-api-trace", false, "Enable trace of api calls")
	cmd.Flags().BoolVar(&o.client.SkipSSL, "from-skip-verify-ssl", false, "Skip SSL verification")
	cmd.Flags().DurationVar(&o.client.ClientTimeout, "from-client-timeout", 5*time.Minute, "Set server calls timeout")
}
