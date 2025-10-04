package fromimmich

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/gen"
	"github.com/simulot/immich-go/internal/immichfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TODO add Locked folder option
type FromImmichCmd struct {
	// CLI flags
	client          app.Client
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
	InclusionFlags  cliflags.InclusionFlags `mapstructure:",squash" yaml:",inline" json:",inline" toml:",inline"`                                                 // controls the file extensions to be included in the import process.

	// internal fields
	albumIDs  []string
	tagIDs    []string
	peopleIDs []string
	ifs       *immichfs.ImmichFS
	ic        *filenames.InfoCollector
	app       *app.Application
}

func (fic *FromImmichCmd) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVar(&fic.Make, "from-make", "", "Get only assets with this make")
	flags.StringVar(&fic.Model, "from-model", "", "Get only assets with this model")
	flags.StringVar(&fic.Country, "from-country", "", "Get only assets from this country")
	flags.StringVar(&fic.State, "from-state", "", "Get only assets from this state")
	flags.StringVar(&fic.City, "from-city", "", "Get only assets from this city")
	flags.StringSliceVar(&fic.Albums, "from-albums", nil, "Get assets only from those albums, can be used multiple times")
	flags.StringSliceVar(&fic.Tags, "from-tags", nil, "Get assets only with those tags, can be used multiple times")
	flags.StringSliceVar(&fic.People, "from-people", nil, "Get assets only with those people, can be used multiple times")
	flags.BoolVar(&fic.IncludePartners, "from-partners", false, "Get partner's assets as well")
	flags.BoolVar(&fic.OnlyArchived, "from-archived", false, "Get only archived assets")
	flags.BoolVar(&fic.OnlyTrashed, "from-trash", false, "Get only trashed assets")
	flags.BoolVar(&fic.OnlyFavorite, "from-favorite", false, "Get only favorite assets")
	flags.BoolVar(&fic.OnlyNoAlbum, "from-no-album", false, "Get only assets that are not in any album")
	flags.IntVar(&fic.MinimalRating, "from-minimal-rating", 0, "Get only assets with a rating greater or equal to this value")
	flags.StringVar(&fic.client.Server, "from-server", fic.client.Server, "Immich server address (example http://your-ip:2283 or https://your-domain)")
	flags.StringVar(&fic.client.APIKey, "from-api-key", "", "API Key")
	flags.BoolVar(&fic.client.APITrace, "from-api-trace", false, "Enable trace of api calls")
	flags.BoolVar(&fic.client.SkipSSL, "from-skip-verify-ssl", false, "Skip SSL verification")
	flags.DurationVar(&fic.client.ClientTimeout, "from-client-timeout", 20*time.Minute, "Set server calls timeout")
	fic.InclusionFlags.RegisterFlags(flags, "from-")
	fic.client.RegisterFlags(flags, "from-")
}

// NewFromImmichCommand creates a new Cobra command for fetching photos from an Immich server.
// It registers all relevant flags, sets up the command context, and binds the execution logic.
func NewFromImmichCommand(ctx context.Context, parent *cobra.Command, app *app.Application, runner adapters.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-immich [flags]",              // Command usage
		Short: "Get photos from an Immich server", // Short description
		Args:  cobra.MaximumNArgs(0),              // No positional arguments allowed
	}
	cmd.SetContext(ctx)            // Set command context
	fic := &FromImmichCmd{}        // Create command handler
	fic.RegisterFlags(cmd.Flags()) // Register CLI flags
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Execute the command logic
		return fic.Run(ctx, cmd, app, runner)
	}
	return cmd
}

// Run executes the FromImmichCmd command, initializing the Immich client and validating filter values
// (such as Make, Model, Country, State, and City) against Immich's search suggestions. It also resolves
// albums, tags, and people filters before delegating execution to the provided runner. Returns an error
// if any validation or resolution step fails.
func (fic *FromImmichCmd) Run(ctx context.Context, cmd *cobra.Command, app *app.Application, runner adapters.Runner) error {
	err := fic.client.Open(ctx, app)
	if err != nil {
		return err
	}
	ifs := immichfs.NewImmichFS(ctx, fic.client.Server, fic.client.Immich)
	f := FromImmichCmd{
		ifs: ifs,
		ic:  filenames.NewInfoCollector(time.Local, fic.client.Immich.SupportedMedia()),
		app: app,
	}
	// check filters values against immich suggestions
	if fic.Make != "" {
		err = f.checkSuggestion(ctx, immich.SearchSuggestionRequest{
			Type: immich.SearchSuggestionTypeCameraMake,
		}, fic.Make)
		if err != nil {
			return fmt.Errorf("invalid make: %w", err)
		}
	}
	if fic.Model != "" {
		err = f.checkSuggestion(ctx, immich.SearchSuggestionRequest{
			Type: immich.SearchSuggestionTypeCameraModel,
			Make: fic.Make,
		}, fic.Model)
		if err != nil {
			return fmt.Errorf("invalid model: %w", err)
		}
	}
	if fic.Country != "" {
		err = f.checkSuggestion(ctx, immich.SearchSuggestionRequest{
			Type: immich.SearchSuggestionTypeCountry,
		}, fic.Country)
		if err != nil {
			return fmt.Errorf("invalid country: %w", err)
		}
	}
	if fic.State != "" {
		err = f.checkSuggestion(ctx, immich.SearchSuggestionRequest{
			Type:    immich.SearchSuggestionTypeState,
			Country: fic.Country,
		}, fic.State)
		if err != nil {
			return fmt.Errorf("invalid state: %w", err)
		}
	}
	if fic.City != "" {
		err = f.checkSuggestion(ctx, immich.SearchSuggestionRequest{
			Type:    immich.SearchSuggestionTypeCity,
			Country: fic.Country,
			State:   fic.State,
		}, fic.City)
		if err != nil {
			return fmt.Errorf("invalid city: %w", err)
		}
	}

	err = fic.resolveAlbums(ctx)
	if err != nil {
		return err
	}

	err = fic.resolveTags(ctx)
	if err != nil {
		return err
	}

	err = fic.resolvePeople(ctx)
	if err != nil {
		return err
	}

	err = runner.Run(cmd, fic)

	return err
}

func (fic *FromImmichCmd) checkSuggestion(ctx context.Context, q immich.SearchSuggestionRequest, suggestion string) error {
	sug := fic.client.Immich.(immich.ImmichGetSuggestion)
	suggestions, err := sug.GetSearchSuggestions(ctx, q)
	if err != nil {
		return err
	}
	if slices.Contains(suggestions, suggestion) {
		return nil
	}
	return fmt.Errorf("there is not '%s' in the suggestions, accepted values: %s", suggestion, formatQuotedStrings(suggestions))
}

func (fic *FromImmichCmd) resolveAlbums(ctx context.Context) error {
	if len(fic.Albums) == 0 {
		return nil
	}
	albums, err := fic.client.Immich.GetAllAlbums(ctx)
	if err != nil {
		return err
	}
	unknownAlbums := []string{}

	for _, fromAlbum := range fic.Albums {
		found := false
		for _, a := range albums {
			if a.AlbumName == fromAlbum {
				fic.albumIDs = gen.AddOnce(fic.albumIDs, a.ID)
				found = true
			}
		}
		if !found {
			unknownAlbums = append(unknownAlbums, fromAlbum)
		}
	}

	if len(unknownAlbums) == 0 {
		return nil
	}

	availables := []string{}
	for _, a := range albums {
		availables = append(availables, a.AlbumName)
	}
	return fmt.Errorf("unknown album(s): %v, available album(s): %v", formatQuotedStrings(unknownAlbums), formatQuotedStrings(availables))
}

func (fic *FromImmichCmd) resolveTags(ctx context.Context) error {
	if len(fic.Tags) == 0 {
		return nil
	}
	tags, err := fic.client.Immich.GetAllTags(ctx)
	if err != nil {
		return err
	}
	unknownTags := []string{}

	for _, fromTag := range fic.Tags {
		found := false
		for _, t := range tags {
			if t.Value == fromTag {
				fic.tagIDs = gen.AddOnce(fic.tagIDs, t.ID)
				found = true
			}
		}
		if !found {
			unknownTags = append(unknownTags, fromTag)
		}
	}

	if len(unknownTags) == 0 {
		return nil
	}

	availables := []string{}
	for _, t := range tags {
		availables = append(availables, t.Value)
	}
	return fmt.Errorf("unknown tag(s): %v, available tag(s): %v", formatQuotedStrings(unknownTags), formatQuotedStrings(availables))
}

func (fic *FromImmichCmd) resolvePeople(ctx context.Context) error {
	if len(fic.People) == 0 {
		return nil
	}

	icP := fic.client.Immich.(immich.ImmichPeopleInterface)
	// Get people by names using the new GetAllPeople endpoint
	peopleMap, err := icP.GetPeopleByNames(ctx, fic.People)
	if err != nil {
		return fmt.Errorf("failed to resolve people names: %w", err)
	}

	unknownPeople := []string{}
	fic.peopleIDs = nil // Reset people IDs

	for _, fromPerson := range fic.People {
		if person, found := peopleMap[fromPerson]; found {
			fic.peopleIDs = gen.AddOnce(fic.peopleIDs, person.ID)
		} else {
			unknownPeople = append(unknownPeople, fromPerson)
		}
	}

	if len(unknownPeople) > 0 {
		// Get all available people names for error message
		var availablePeople []string
		err := icP.GetAllPeopleIterator(ctx, func(person *immich.PersonResponseDto) error {
			if person.Name != "" {
				availablePeople = append(availablePeople, person.Name)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("unknown people: %v (failed to get available people: %w)", formatQuotedStrings(unknownPeople), err)
		}

		return fmt.Errorf("unknown people: %v, available people: %v", formatQuotedStrings(unknownPeople), formatQuotedStrings(availablePeople))
	}

	return nil
}

func (fic *FromImmichCmd) Browse(ctx context.Context) chan *assets.Group {
	gOut := make(chan *assets.Group)
	go func() {
		defer close(gOut)

		err := fic.getAssets(ctx, gOut)
		if err = fic.app.ProcessError(err); err != nil {
			return
		}
	}()
	return gOut
}

func (fic *FromImmichCmd) getAssets(ctx context.Context, grpChan chan *assets.Group) error {
	fic.MinimalRating = min(max(0, fic.MinimalRating), 5)

	so := immich.SearchOptions()

	if !fic.OnlyArchived && !fic.OnlyTrashed && !fic.OnlyFavorite {
		so.All()
	} else {
		if fic.OnlyArchived {
			so.WithOnlyArchived()
		}
		if fic.OnlyTrashed {
			so.WithOnlyTrashed()
		}
		if fic.OnlyFavorite {
			so.WithOnlyFavorite()
		}
	}

	if fic.Make != "" {
		so.WithOnlyMake(fic.Make)
	}

	if fic.Model != "" {
		so.WithOnlyMake(fic.Model)
	}
	if fic.Country != "" {
		so.WithOnlyCountry(fic.Country)
	}
	if fic.State != "" {
		so.WithOnlyState(fic.State)
	}
	if fic.City != "" {
		so.WithOnlyCity(fic.City)
	}

	if fic.OnlyNoAlbum {
		so.WithNotInAlbum()
	} else {
		so.WithAlbums(fic.albumIDs...)
	}

	if fic.InclusionFlags.DateRange.IsSet() {
		so.WithDateRange(fic.InclusionFlags.DateRange)
	}

	if fic.MinimalRating > 1 {
		so.WithMinimalRate(fic.MinimalRating)
	}

	if fic.Make != "" {
		so.WithOnlyMake(fic.Make)
	}

	if len(fic.albumIDs) > 0 {
		so.WithAlbums(fic.albumIDs...)
	}

	if len(fic.tagIDs) > 0 {
		so.WithTags(fic.tagIDs...)
	}

	if len(fic.peopleIDs) > 0 {
		so.WithPeople(fic.peopleIDs...)
	}

	return fic.client.Immich.GetFilteredAssetsFn(ctx, so, func(a *immich.Asset) error {
		// filters on data returned by the search API
		if !fic.IncludePartners && a.OwnerID != fic.client.User.ID {
			return nil
		}

		// Fetch details
		a, err := fic.client.Immich.GetAssetInfo(ctx, a.ID)
		if err = fic.app.ProcessError(err); err != nil {
			return err
		}

		asset := a.AsAsset()
		asset.FromApplication = &assets.Metadata{
			FileName:    a.OriginalFileName,
			Latitude:    a.ExifInfo.Latitude,
			Longitude:   a.ExifInfo.Longitude,
			Description: a.ExifInfo.Description,
			DateTaken:   a.ExifInfo.DateTimeOriginal.Time,
			Trashed:     a.IsTrashed,
			Archived:    a.IsArchived,
			Favorited:   a.IsFavorite,
			Rating:      byte(a.ExifInfo.Rating),
			Tags:        asset.Tags,
		}
		asset.UseMetadata(asset.FromApplication)
		asset.File = fshelper.FSName(fic.ifs, a.ID)

		// Transfer the album
		simplifiedA, err := fic.client.Immich.GetAssetAlbums(ctx, a.ID)
		if err = fic.app.ProcessError(err); err != nil {
			return err
		}

		albums := immich.AlbumsFromAlbumSimplified(simplifiedA)
		// clear the ID of the album that exists in from server, but not in to server
		for i := range albums {
			albums[i].ID = ""
		}

		asset.Albums = albums

		// Transfer tags
		for t := range asset.Tags {
			asset.Tags[t].ID = ""
		}

		g := assets.NewGroup(assets.GroupByNone, asset)
		select {
		case grpChan <- g:
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	})
}

func formatQuotedStrings(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	quoted := make([]string, len(ss))
	for i, s := range ss {
		quoted[i] = fmt.Sprintf("'%s'", s)
	}
	return strings.Join(quoted, ", ")
}
