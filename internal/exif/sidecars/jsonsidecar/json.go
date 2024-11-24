package jsonsidecar

import (
	"encoding/json"
	"io"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
)

type meta struct {
	Software string `json:"software"`
	assets.Metadata
}

func Write(md *assets.Metadata, w io.Writer) error {
	v := meta{
		Software: app.GetVersion(),
		Metadata: *md,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func Read(r io.Reader, md *assets.Metadata) error {
	var v meta
	dec := json.NewDecoder(r)
	if err := dec.Decode(&v); err != nil {
		return err
	}
	*md = v.Metadata
	return nil
}
