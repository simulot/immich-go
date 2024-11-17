package jsonsidecar

import (
	"encoding/json"
	"io"

	"github.com/simulot/immich-go/commands/application"
	"github.com/simulot/immich-go/internal/assets"
)

type MasterData struct {
	Software string `json:"software"`
	assets.Metadata
}

func Write(md *assets.Metadata, w io.Writer) error {
	v := MasterData{
		Software: application.GetVersion(),
		Metadata: *md,
	}
	return json.NewEncoder(w).Encode(v)
}
