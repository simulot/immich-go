package epsonfastfoto

import (
	"context"
	"regexp"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/filetypes"
)

var epsonFastFotoRegex = regexp.MustCompile(`^(.*_\d+)(_[ab])?(\.[a-z]+)$`)

type Group struct {
	lastRadical string
	coverIndex  int
	group       []*assets.Asset
}

func (g Group) Group(ctx context.Context, in <-chan *assets.Asset, out chan<- *assets.Asset, gOut chan<- *assets.Group) {
	for {
		select {
		case <-ctx.Done():
			return
		case a, ok := <-in:
			if !ok {
				g.sendGroup(ctx, out, gOut)
				return
			}
			ni := a.NameInfo
			matches := epsonFastFotoRegex.FindStringSubmatch(a.File.Name())
			if matches == nil {
				g.sendGroup(ctx, out, gOut)
				select {
				case out <- a:
				case <-ctx.Done():
				}
				continue
			}

			radical := matches[1]
			// exclude movies,  burst images
			dontGroupMe := ni.Type != filetypes.TypeImage ||
				ni.Kind == assets.KindBurst

			if dontGroupMe {
				g.sendGroup(ctx, out, gOut)
				continue
			}
			if g.lastRadical != radical {
				g.sendGroup(ctx, out, gOut)
			}
			g.group = append(g.group, a)
			g.lastRadical = radical
			if matches[2] == "_a" {
				g.coverIndex = len(g.group) - 1
			}
		}
	}
}

func (g *Group) sendGroup(ctx context.Context, out chan<- *assets.Asset, outg chan<- *assets.Group) {
	defer func() {
		g.group = nil
		g.lastRadical = ""
		g.coverIndex = 0
	}()
	if len(g.group) == 0 {
		return
	}
	if len(g.group) < 2 {
		select {
		case out <- g.group[0]:
		case <-ctx.Done():
		}
		return
	}

	gr := assets.NewGroup(assets.GroupByOther, g.group...)
	gr.CoverIndex = g.coverIndex

	select {
	case <-ctx.Done():
		return
	case outg <- gr:
	}
}
