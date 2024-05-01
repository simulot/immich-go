package duplicatelist

// DuplicateLoadingMsg informs the model about the progression of the asset checking
type DuplicateLoadingMsg struct {
	Total      int
	Checked    int
	Duplicated int
}
