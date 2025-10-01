package cliflags

import (
	"fmt"
	"slices"
	"strings"

	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/spf13/pflag"
)

type InclusionFlags struct {
	ExcludedExtensions ExtensionList `mapstructure:"exclude-extensions" yaml:"exclude-extensions" json:"exclude-extensions" toml:"exclude-extensions"`
	IncludedExtensions ExtensionList `mapstructure:"include-extensions" yaml:"include-extensions" json:"include-extensions" toml:"include-extensions"`
	IncludedType       IncludeType   `mapstructure:"include-type" yaml:"include-type" json:"include-type" toml:"include-type"`
	DateRange          DateRange     `mapstructure:"date-range" yaml:"date-range" json:"date-range" toml:"date-range"`
}

func (flags *InclusionFlags) RegisterFlags(fs *pflag.FlagSet, prefix string) {
	fs.Var(&flags.DateRange, prefix+"date-range", "Only import photos taken within the specified date range")
	fs.Var(&flags.ExcludedExtensions, prefix+"exclude-extensions", "Comma-separated list of extension to exclude. (e.g. .gif,.PM) (default: none)")
	fs.Var(&flags.IncludedExtensions, prefix+"include-extensions", "Comma-separated list of extension to include. (e.g. .jpg,.heic) (default: all)")
	fs.Var(&flags.IncludedType, prefix+"include-type", "Single file type to include. (VIDEO or IMAGE) (default: all)")
}

// An IncludeType is either of the constants below which
// represents a collection of extensions.
type IncludeType string

const (
	IncludeAll   IncludeType = ""
	IncludeVideo IncludeType = "VIDEO"
	IncludeImage IncludeType = "IMAGE"
)

// SetIncludeTypeExtensions must be called once flags are parsed
func (flags *InclusionFlags) SetIncludeTypeExtensions() {
	mediaToExtensionsMap := filetypes.MediaToExtensions()

	switch flags.IncludedType {
	case IncludeVideo:
		flags.IncludedExtensions = append(flags.IncludedExtensions, mediaToExtensionsMap[filetypes.TypeVideo]...)
	case IncludeImage:
		flags.IncludedExtensions = append(flags.IncludedExtensions, mediaToExtensionsMap[filetypes.TypeImage]...)
	}
	flags.IncludedExtensions = append(flags.IncludedExtensions, mediaToExtensionsMap[filetypes.TypeSidecar]...)
}

// Add the approprite extensions flags given the user inclusion flag
func setIncludeTypeExtensions(flags *InclusionFlags) {
	mediaToExtensionsMap := filetypes.MediaToExtensions()

	switch flags.IncludedType {
	case IncludeVideo:
		flags.IncludedExtensions = append(flags.IncludedExtensions, mediaToExtensionsMap[filetypes.TypeVideo]...)
	case IncludeImage:
		flags.IncludedExtensions = append(flags.IncludedExtensions, mediaToExtensionsMap[filetypes.TypeImage]...)
	}
	flags.IncludedExtensions = append(flags.IncludedExtensions, mediaToExtensionsMap[filetypes.TypeSidecar]...)
}

// Validate validates the common flags.
func (flags *InclusionFlags) Validate() {
	flags.ExcludedExtensions = flags.ExcludedExtensions.Validate()
	flags.IncludedExtensions = flags.IncludedExtensions.Validate()
}

// Implements the flag interface
func (t *IncludeType) Set(v string) error {
	v = strings.TrimSpace(strings.ToUpper(v))
	switch v {
	case string(IncludeVideo), string(IncludeImage):
		*t = IncludeType(v)
	default:
		return fmt.Errorf("invalid value for include type, expected %s or %s", IncludeVideo, IncludeImage)
	}
	return nil
}

func (t IncludeType) String() string {
	return string(t)
}

func (t IncludeType) Type() string {
	return "IncludeType"
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
