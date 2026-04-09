package snapchat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/fshelper/debugfiles"
)

type mergedFS struct {
	media mediaFile
	app   *app.Application
	name  string
}

func newMergedFS(media mediaFile, app *app.Application) fs.FS {
	full := fshelper.FSName(media.main.fsys, media.main.name).FullName()
	return &mergedFS{
		media: media,
		app:   app,
		name:  "snapchat-merged:" + full,
	}
}

func (m *mergedFS) Open(name string) (fs.File, error) {
	if name != mergedAssetName {
		return nil, fs.ErrNotExist
	}
	mainFile, err := m.media.main.fsys.Open(m.media.main.name)
	if err != nil {
		return nil, err
	}
	debugfiles.TrackOpenFile(mainFile, m.media.main.name)

	ext := strings.ToLower(filepath.Ext(m.media.main.base))
	typeMain := m.media.mainType()
	if m.media.overlay == nil {
		return mainFile, nil
	}

	overlayFile, err := m.media.overlay.fsys.Open(m.media.overlay.name)
	if err != nil {
		m.app.Log().Warn("can't open overlay, using main media", "main", m.media.main.name, "overlay", m.media.overlay.name, "err", err)
		return mainFile, nil
	}
	debugfiles.TrackOpenFile(overlayFile, m.media.overlay.name)

	temp, err := os.CreateTemp("", "immich-go-snapchat-merged-*")
	if err != nil {
		_ = overlayFile.Close()
		return nil, err
	}
	debugfiles.TrackOpenFile(temp, temp.Name())

	switch typeMain {
	case mediaTypeImage:
		err = mergeImage(mainFile, overlayFile, temp, ext)
	case mediaTypeVideo:
		err = mergeVideo(mainFile, overlayFile, temp)
	default:
		err = fmt.Errorf("unsupported media type for overlay merge")
	}

	_ = overlayFile.Close()
	if err != nil {
		_ = temp.Close()
		_ = os.Remove(temp.Name())
		m.app.Log().Warn("can't merge overlay, using main media", "main", m.media.main.name, "overlay", m.media.overlay.name, "err", err)
		return mainFile, nil
	}

	_, err = temp.Seek(0, io.SeekStart)
	if err != nil {
		_ = temp.Close()
		_ = os.Remove(temp.Name())
		return nil, err
	}

	_ = mainFile.Close()
	return &tempMergedFile{File: temp}, nil
}

type tempMergedFile struct {
	*os.File
}

func (f *tempMergedFile) Close() error {
	name := f.Name()
	err := f.File.Close()
	debugfiles.TrackCloseFile(f.File)
	return errors.Join(err, os.Remove(name))
}

func mergeImage(main io.Reader, overlay io.Reader, out io.Writer, ext string) error {
	base, _, err := image.Decode(main)
	if err != nil {
		return err
	}
	over, _, err := image.Decode(overlay)
	if err != nil {
		return err
	}
	over = imaging.Resize(over, base.Bounds().Dx(), base.Bounds().Dy(), imaging.Lanczos)
	merged := imaging.Overlay(base, over, image.Point{X: 0, Y: 0}, 1.0)
	if ext == ".png" {
		return png.Encode(out, merged)
	}
	return imaging.Encode(out, merged, imaging.JPEG)
}

func mergeVideo(main io.Reader, overlay io.Reader, out *os.File) error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found: %w", err)
	}

	mainTmp, err := os.CreateTemp("", "immich-go-snapchat-main-*.mp4")
	if err != nil {
		return err
	}
	defer os.Remove(mainTmp.Name())
	defer mainTmp.Close()
	if _, err := io.Copy(mainTmp, main); err != nil {
		return err
	}

	overlayTmp, err := os.CreateTemp("", "immich-go-snapchat-overlay-*.png")
	if err != nil {
		return err
	}
	defer os.Remove(overlayTmp.Name())
	defer overlayTmp.Close()
	if _, err := io.Copy(overlayTmp, overlay); err != nil {
		return err
	}

	outTmp, err := os.CreateTemp("", "immich-go-snapchat-video-merged-*.mp4")
	if err != nil {
		return err
	}
	outPath := outTmp.Name()
	_ = outTmp.Close()
	defer os.Remove(outPath)

	cmd := exec.CommandContext(context.Background(), "ffmpeg", "-y", "-loglevel", "error", "-i", mainTmp.Name(), "-i", overlayTmp.Name(), "-filter_complex", "[1:v][0:v]scale2ref=w=iw:h=ih[ovr][vid];[vid][ovr]overlay=0:0:eof_action=repeat[v]", "-map", "0:a?", "-map", "[v]", "-c:v", "libx264", "-c:a", "copy", outPath)
	cmd.Stderr = &bytes.Buffer{}
	cmd.Stdout = io.Discard

	if err := cmd.Run(); err != nil {
		if b, ok := cmd.Stderr.(*bytes.Buffer); ok {
			return fmt.Errorf("ffmpeg merge failed: %w: %s", err, strings.TrimSpace(b.String()))
		}
		return err
	}

	if _, err := out.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := out.Truncate(0); err != nil {
		return err
	}
	f, err := os.Open(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(out, f)
	if err != nil {
		return err
	}
	_, err = out.Seek(0, io.SeekStart)
	return err
}

type mediaKind int

const (
	mediaTypeUnknown mediaKind = iota
	mediaTypeImage
	mediaTypeVideo
)

func (m mediaFile) mainType() mediaKind {
	ext := strings.ToLower(filepath.Ext(m.main.base))
	t := fileType(ext)
	switch t {
	case "image":
		return mediaTypeImage
	case "video":
		return mediaTypeVideo
	default:
		return mediaTypeUnknown
	}
}

func fileType(ext string) string {
	supported := map[string]string{
		".jpg": "image", ".jpeg": "image", ".png": "image", ".webp": "image", ".heic": "image", ".heif": "image",
		".mp4": "video", ".mov": "video", ".m4v": "video", ".webm": "video",
	}
	if t, ok := supported[ext]; ok {
		return t
	}
	return ""
}

func (m *mergedFS) Stat(name string) (fs.FileInfo, error) {
	if name != mergedAssetName {
		return nil, fs.ErrNotExist
	}
	if m.media.main == nil {
		return nil, fs.ErrNotExist
	}
	return fshelper.Stat(m.media.main.fsys, m.media.main.name)
}

func (m *mergedFS) Name() string {
	return m.name
}

var _ fs.FS = (*mergedFS)(nil)
var _ fshelper.FSCanStat = (*mergedFS)(nil)
var _ fshelper.NameFS = (*mergedFS)(nil)
