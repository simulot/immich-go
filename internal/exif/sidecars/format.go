package sidecars

import (
	"fmt"
	"strings"
)

// SidecarFormat specifies which sidecar format(s) to create during archive operations
type SidecarFormat string

const (
	// FormatJSON creates JSON sidecars only (default, supports later upload with immich-go without losing data like `Trashed`, `Archived`, `FromPartner`, `FileName`)
	FormatJSON SidecarFormat = "json"
	// FormatXMP creates XMP sidecars only (interoperable with other applications and supported by Immich in external libraries or on immich-cli upload)
	FormatXMP SidecarFormat = "xmp"
	// FormatBoth creates both JSON and XMP sidecars
	FormatBoth SidecarFormat = "both"
)

// String returns the string representation of the format
func (f SidecarFormat) String() string {
	return string(f)
}

// Set implements pflag.Value interface
func (f *SidecarFormat) Set(s string) error {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "json", "xmp", "both":
		*f = SidecarFormat(s)
		return nil
	default:
		return fmt.Errorf("invalid sidecar format %q: must be json, xmp, or both", s)
	}
}

// Type implements pflag.Value interface
func (f *SidecarFormat) Type() string {
	return "string"
}

// CreatesJSON returns true if this format creates JSON sidecars
func (f SidecarFormat) CreatesJSON() bool {
	return f == FormatJSON || f == FormatBoth
}

// CreatesXMP returns true if this format creates XMP sidecars
func (f SidecarFormat) CreatesXMP() bool {
	return f == FormatXMP || f == FormatBoth
}
