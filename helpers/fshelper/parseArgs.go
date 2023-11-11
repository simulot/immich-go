package fshelper

import (
	"errors"
	"fmt"
	"immich-go/helpers/gen"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type argParser struct {
	files       []string
	paths       map[string][]string
	zips        []string
	unsupported map[string]any
	err         error
}

func ParsePath(args []string) ([]fs.FS, error) {
	p := argParser{
		unsupported: map[string]any{},
		paths:       map[string][]string{},
	}

	for _, f := range args {
		if !HasMagic(f) {
			p.handleFile(f)
			continue
		} else {
			globs, err := filepath.Glob(f)
			if err != nil {
				p.err = errors.Join(err)
				continue
			}
			if len(globs) == 0 {
				p.err = errors.Join(fmt.Errorf("no file matches '%s'", f))
				continue
			}

			for _, f := range globs {
				p.handleFile(f)
			}
		}
	}

	fsys := []fs.FS{}

	for _, f := range p.files {
		d, b := filepath.Split(f)
		d = filepath.Clean(d)
		l := append(p.paths[d], b)
		p.paths[d] = l
	}

	for pa, l := range p.paths {
		f, err := newPathFS(pa, l)
		if err != nil {
			p.err = errors.Join(err)
		} else {
			fsys = append(fsys, f)
		}
	}

	if len(p.zips) > 0 {
		f, err := multiZip(p.zips...)
		if err != nil {
			p.err = errors.Join(err)
		} else {
			fsys = append(fsys, f)
		}
	}
	if len(p.unsupported) > 0 {
		keys := gen.MapKeys(p.unsupported)
		for _, k := range keys {
			p.err = errors.Join(fmt.Errorf("files with extension '%s' are not supported", k))
		}
	}
	return fsys, p.err
}

func (p *argParser) handleFile(f string) {
	i, err := os.Stat(f)
	if err != nil {
		p.err = errors.Join(err)
		return
	}
	if i.IsDir() {
		if _, exists := p.paths[f]; !exists {
			p.paths[f] = nil
		}
		return
	}
	ext := strings.ToLower(filepath.Ext(f))
	if ext == ".zip" {
		p.zips = append(p.zips, f)
		return
	}
	if _, err = MimeFromExt(ext); err == nil {
		p.files = append(p.files, f)
	} else {
		p.unsupported[ext] = nil
	}
}
