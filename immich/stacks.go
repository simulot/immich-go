package immich

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Stack is a stack as GET /stacks lists it. Immich lists only its members that are not trashed and
// are on the timeline or archived.
type Stack struct {
	ID             string `json:"id"`
	PrimaryAssetID string `json:"primaryAssetId"`
	Assets         []struct {
		ID string `json:"id"`
	} `json:"assets"`
}

// IsInStack tells whether the asset is in a stack: a search for it with withStacked=false (trashed
// included) finds it only when it is in none. A failed search is no answer: false, with the error.
func (ic *ImmichClient) IsInStack(ctx context.Context, id string) (bool, error) {
	if id == "" {
		return false, errNoAssetID("IsInStack")
	}
	unstacked := false
	q := SearchMetadataQuery{ID: id, WithStacked: new(false), WithDeleted: true}
	err := ic.callSearchMetadata(ctx, &q, func(a *Asset) error {
		unstacked = unstacked || a.ID == id
		return nil
	})
	if err != nil {
		return false, err
	}
	return !unstacked, nil
}

// GetAllStacks lists the user's stacks (GET /stacks).
func (ic *ImmichClient) GetAllStacks(ctx context.Context) ([]Stack, error) {
	var result []Stack
	err := ic.newServerCall(ctx, "GetAllStacks").do(getRequest("/stacks", setAcceptJSON()), responseJSON(&result))
	return result, err
}

// CreateStack create a stack with the given assets, the 1st asset is the cover, return the stack ID
func (ic *ImmichClient) CreateStack(ctx context.Context, ids []string) (string, error) {
	// remove the empty ids
	for i := 0; i < len(ids); i++ {
		if ids[i] == "" {
			ids = append(ids[:i], ids[i+1:]...)
			i--
		}
	}
	if len(ids) < 2 {
		return "", fmt.Errorf("stack must have at least 2 assets")
	}

	if ic.dryRun {
		return uuid.NewString(), nil
	}

	param := struct {
		AssetIds []string `json:"assetIds"`
	}{
		AssetIds: ids,
	}

	var result struct {
		ID             string `json:"id"`
		PrimaryAssetID string `json:"primaryAssetId"`
	}

	err := ic.newServerCall(ctx, "createStack").do(postRequest("/stacks", "application/json", setAcceptJSON(), setJSONBody(param)), responseJSON(&result))
	return result.ID, err
}
