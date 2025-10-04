package archive

import (
	"errors"
	"os"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/adapters/folder"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fshelper/osfs"
	"github.com/spf13/cobra"
)

func (ac *ArchiveCmd) Run(cmd *cobra.Command, adapter adapters.Reader) error {
	// ready to run
	ctx := cmd.Context()
	if ac.app.Jnl() == nil {
		ac.app.SetJnl(fileevent.NewRecorder(ac.app.Log().Logger))
		ac.app.Jnl().SetLogger(ac.app.Log().SetLogWriter(os.Stdout))
	}
	jnl := ac.app.Jnl()

	p := ac.ArchivePath
	err := os.MkdirAll(p, 0o755)
	if err != nil {
		return err
	}

	destFS := osfs.DirFS(p)
	ac.dest, err = folder.NewLocalAssetWriter(destFS, ".")
	if err != nil {
		return err
	}

	gChan := adapter.Browse(ctx)
	errCount := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case g, ok := <-gChan:
			if !ok {
				return nil
			}
			for _, a := range g.Assets {
				err := ac.dest.WriteAsset(ctx, a)
				if err == nil {
					err = a.Close()
				}
				if err != nil {
					jnl.Log().Error(err.Error())
					errCount++
					if errCount > 5 {
						err := errors.New("too many errors, aborting")
						jnl.Log().Error(err.Error())
						return err
					}
				} else {
					jnl.Record(ctx, fileevent.Written, a)
				}
			}
		}
	}
}
