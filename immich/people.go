package immich

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

const peopleQueryLimit = 500

// PersonResponseDto represents a person in the Immich system
type PersonResponseDto struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	BirthDate     ImmichTime `json:"birthDate,omitzero"`
	Color         string     `json:"color,omitempty"`      // Added in v1.126.0
	IsFavorite    bool       `json:"isFavorite,omitempty"` // Added in v1.126.0
	IsHidden      bool       `json:"isHidden"`
	ThumbnailPath string     `json:"thumbnailPath"`
	UpdatedAt     ImmichTime `json:"updatedAt,omitempty"` // Added in v1.107.0
}

// PeopleResponseDto represents the response from the getAllPeople endpoint
type PeopleResponseDto struct {
	HasNextPage bool                `json:"hasNextPage,omitempty"` // Added in v1.110.0
	Hidden      int                 `json:"hidden"`
	Total       int                 `json:"total"`
	People      []PersonResponseDto `json:"people"`
}

// GetAllPeopleOptions represents the query parameters for the getAllPeople endpoint
type GetAllPeopleOptions struct {
	ClosestAssetId  string // UUID of the closest asset
	ClosestPersonId string // UUID of the closest person
	Page            int    // Page number for pagination (default: 1, minimum: 1)
	Size            int    // Number of items per page (default: 500, minimum: 1, maximum: 1000)
	WithHidden      *bool  // Whether to include hidden people
}

// SetURL implements the URL parameter setting for GetAllPeopleOptions
func (opts GetAllPeopleOptions) SetURL(u *url.URL) error {
	if opts.Page == 0 {
		opts.Page = 1
	}
	if opts.Size == 0 {
		opts.Size = peopleQueryLimit
	}

	qv := u.Query()

	if opts.ClosestAssetId != "" {
		qv.Set("closestAssetId", opts.ClosestAssetId)
	}

	if opts.ClosestPersonId != "" {
		qv.Set("closestPersonId", opts.ClosestPersonId)
	}

	qv.Set("page", strconv.Itoa(opts.Page))
	qv.Set("size", strconv.Itoa(opts.Size))

	if opts.WithHidden != nil {
		qv.Set("withHidden", strconv.FormatBool(*opts.WithHidden))
	}

	u.RawQuery = qv.Encode()
	return nil
}

// GetAllPeople retrieves all people from the Immich server
// This method implements the getAllPeople endpoint with pagination support
func (ic *ImmichClient) GetAllPeople(ctx context.Context, opts ...GetAllPeopleOptions) (*PeopleResponseDto, error) {
	var options GetAllPeopleOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	// Set defaults
	if options.Page == 0 {
		options.Page = 1
	}
	if options.Size == 0 {
		options.Size = peopleQueryLimit
	}

	// Validate parameters
	if options.Page < 1 {
		return nil, fmt.Errorf("page must be at least 1")
	}
	if options.Size < 1 || options.Size > 1000 {
		return nil, fmt.Errorf("size must be between 1 and 1000")
	}

	var response PeopleResponseDto
	err := ic.newServerCall(ctx, EndPointGetAllPeople).do(
		getRequest("/people", UrlRequest(options)),
		responseJSON(&response),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get all people: %w", err)
	}

	return &response, nil
}

// GetAllPeopleIterator retrieves all people using pagination automatically
// This is a convenience method that handles pagination internally
func (ic *ImmichClient) GetAllPeopleIterator(ctx context.Context, fn func(*PersonResponseDto) error, opts ...GetAllPeopleOptions) error {
	var options GetAllPeopleOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	// Set defaults
	if options.Page == 0 {
		options.Page = 1
	}
	if options.Size == 0 {
		options.Size = peopleQueryLimit
	}

	for {
		response, err := ic.GetAllPeople(ctx, options)
		if err != nil {
			return fmt.Errorf("failed to iterate people: %w", err)
		}

		// Process each person
		for _, person := range response.People {
			if err := fn(&person); err != nil {
				return err
			}
		}

		// Check if we need to continue pagination
		if !response.HasNextPage {
			break
		}

		options.Page++
	}

	return nil
}

// GetPersonByName finds a person by their name
// Returns nil if no person with the given name is found
func (ic *ImmichClient) GetPersonByName(ctx context.Context, name string, opts ...GetAllPeopleOptions) (*PersonResponseDto, error) {
	var foundPerson *PersonResponseDto

	err := ic.GetAllPeopleIterator(ctx, func(person *PersonResponseDto) error {
		if person.Name == name {
			foundPerson = person
			return fmt.Errorf("found") // Use error to break iteration
		}
		return nil
	}, opts...)

	if err != nil && err.Error() != "found" {
		return nil, fmt.Errorf("failed to search for person by name: %w", err)
	}

	return foundPerson, nil
}

// GetPeopleByNames finds multiple people by their names
// Returns a map of name to PersonResponseDto, missing names will not be in the map
func (ic *ImmichClient) GetPeopleByNames(ctx context.Context, names []string, opts ...GetAllPeopleOptions) (map[string]*PersonResponseDto, error) {
	if len(names) == 0 {
		return make(map[string]*PersonResponseDto), nil
	}

	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	result := make(map[string]*PersonResponseDto)

	err := ic.GetAllPeopleIterator(ctx, func(person *PersonResponseDto) error {
		if nameSet[person.Name] {
			result[person.Name] = person
			delete(nameSet, person.Name)

			// If we found all requested names, we can stop
			if len(nameSet) == 0 {
				return fmt.Errorf("all found") // Use error to break iteration
			}
		}
		return nil
	}, opts...)

	if err != nil && err.Error() != "all found" {
		return nil, fmt.Errorf("failed to search for people by names: %w", err)
	}

	return result, nil
}
