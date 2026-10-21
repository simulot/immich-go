package assets

// Stack is an asset's membership of a stack on the Immich server it was archived from
// (archive from-immich). upload rebuilds exactly these stacks.
type Stack struct {
	ID         string `json:"id"`                   // the stack's ID on the source server
	Primary    bool   `json:"primary"`              // this asset is the stack's primary asset
	AssetCount int    `json:"assetCount,omitempty"` // the stack's size as the source server reported it
}
