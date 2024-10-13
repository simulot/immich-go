package cliflags

import (
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

type InclusionFlags struct {
	ExcludedExtensions ExtensionList
	IncludedExtensions ExtensionList
	DateRange          DateRange
}

func AddInclusionFlags(cmd *cobra.Command, flags *InclusionFlags) {
	cmd.Flags().Var(&flags.DateRange, "date-range", "Only import photos taken within the specified date range")
	cmd.Flags().Var(&flags.ExcludedExtensions, "exclude-extensions", "Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none)")
	cmd.Flags().Var(&flags.IncludedExtensions, "include-extensions", "Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all)")
}

// Validate validates the common flags.
func (flags *InclusionFlags) Validate() {
	flags.ExcludedExtensions = flags.ExcludedExtensions.Validate()
	flags.IncludedExtensions = flags.IncludedExtensions.Validate()
}

// An ExtensionList is a list of file extensions, where each extension is a string that starts with a dot (.) and is in lowercase.
type ExtensionList []string

// Validate validates the extension list by converting to lowercase.
func (sl ExtensionList) Validate() ExtensionList {
	vl := ExtensionList{}
	for _, e := range sl {
		e = strings.ToLower(strings.TrimSpace(e))
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		vl = append(vl, e)
	}
	return vl
}

// Include checks if the extension list includes a given extension.
func (sl ExtensionList) Include(s string) bool {
	if len(sl) == 0 {
		return true
	}
	s = strings.ToLower(s)
	return slices.Contains(sl, strings.ToLower(s))
}

// Exclude checks if the extension list excludes a given extension.
func (sl ExtensionList) Exclude(s string) bool {
	if len(sl) == 0 {
		return false
	}
	s = strings.ToLower(s)
	return slices.Contains(sl, strings.ToLower(s))
}

// Implements the flag interface
func (sl *ExtensionList) Set(s string) error {
	exts := strings.Split(s, ",")
	for _, ext := range exts {
		ext = strings.TrimSpace(ext)
		if ext != "" {
			*sl = append(*sl, ext)
		}
	}
	return nil
}

func (sl ExtensionList) String() string {
	return strings.Join(sl, ", ")
}

func (sl ExtensionList) Type() string {
	return "ExtensionList"
}
