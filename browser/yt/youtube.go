package yt

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/simulot/immich-go/browser"
	"github.com/simulot/immich-go/helpers/fileevent"
	"github.com/simulot/immich-go/helpers/fshelper"
	"github.com/simulot/immich-go/helpers/namematcher"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/immich/metadata"
)

type SynthesizedYouTubeVideo struct {
	Channel   *YouTubeChannel
	Playlists []*YouTubePlaylist
	Video     *YouTubeVideo
	Recording *YouTubeVideoRecording
	Fsys      fs.FS
	Filename  string
}

type Takeout struct {
	fsyss      []fs.FS
	videos     []*SynthesizedYouTubeVideo
	faves      map[string]bool
	log        *fileevent.Recorder
	sm         immich.SupportedMedia
	banned     namematcher.List // Banned files
}

func NewTakeout(ctx context.Context, l *fileevent.Recorder, sm immich.SupportedMedia, fsyss ...fs.FS) (*Takeout, error) {
	to := Takeout{
		fsyss:  fsyss,
		videos: []*SynthesizedYouTubeVideo{},
		faves:  map[string]bool{},
		log:    l,
		sm:     sm,
	}

	return &to, nil
}

func (to *Takeout) SetBannedFiles(banned namematcher.List) *Takeout {
	to.banned = banned
	return to
}

// Prepare scans all files to build gather and aggregate the metadata
func (to *Takeout) Prepare(ctx context.Context) error {
	smExts := []string{}
	for ext := range to.sm {
		if to.sm[ext] == immich.TypeVideo {
			smExts = append(smExts, ext[1:])
		}
	}
	pattern := "\\(\\d+\\)\\.(?:" + strings.Join(smExts, "|") + ")"
	re := regexp.MustCompile(pattern)

	for _, fsys := range to.fsyss {
		tofs, err := fs.Sub(fsys, "Takeout")
		if err != nil {
			return err
		}
		ytfs, err := fs.Sub(tofs, "YouTube and YouTube Music")
		if err != nil {
			return err
		}
		cfs, err := fs.Sub(ytfs, "channels")
		if err != nil {
			return err
		}
		pfs, err := fs.Sub(ytfs, "playlists")
		if err != nil {
			return err
		}
		vfs, err := fs.Sub(ytfs, "videos")
		if err != nil {
			return err
		}
		vmfs, err := fs.Sub(ytfs, "video metadata")
		if err != nil {
			return err
		}

		// ChannelID => YouTubeChannel
		channels := map[string]*YouTubeChannel{}
		ytchannels, err := fshelper.ReadCSV[YouTubeChannel](cfs, "channel.csv")
		if err != nil {
			return err
		}
		for i, _ := range ytchannels {
			channel := ytchannels[i]
			channels[channel.ChannelID] = &channel
		}

		// VideoID => YouTubePlaylist
		playlists := map[string][]*YouTubePlaylist{}
		ytplaylists, err := fshelper.ReadCSV[YouTubePlaylist](pfs, "playlists.csv")
		playlistTitlesToIDs := map[string]string{}
		if err != nil {
			return err
		}
		hasFavorites := false
		for i, _ := range ytplaylists {
			playlist := ytplaylists[i]
			if playlist.Title == "Watch later" {
				to.log.Record(ctx, fileevent.DiscoveredDiscarded, nil, "Watch later.csv", "reason", "useless file")
				continue
			} else if playlist.Title == "Favorites" {
				to.log.Record(ctx, fileevent.AnalysisAssociatedMetadata, nil, "Favorites-videos.csv", "reason", "playlist file referenced in playlists.csv")
				hasFavorites = true
				continue
			}
			playlistID, ok := playlistTitlesToIDs[playlist.Title]
			if !ok {
				playlistTitlesToIDs[playlist.Title] = playlist.PlaylistID
			} else {
				to.log.Record(ctx, fileevent.AnalysisLocalDuplicate, nil, "playlists.csv", "duplicate playlist name", "Playlist IDs " + playlistID + " and " + playlist.PlaylistID + " are both named '" + playlist.Title + "'; there's no way of knowing which playlist is actually being stored")
			}

			to.log.Record(ctx, fileevent.AnalysisAssociatedMetadata, nil, playlist.Filename(), "reason", "playlist file referenced in playlists.csv")
			ytplaylistvideos, err := fshelper.ReadCSV[YouTubePlaylistVideo](pfs, playlist.Filename())
			if err != nil {
				return err
			}
			for j, _ := range ytplaylistvideos {
				playlistvideo := ytplaylistvideos[j]
				playlists[playlistvideo.VideoID] = append(playlists[playlistvideo.VideoID], &playlist)
			}
		}

		if hasFavorites {
			favoriteVideos, err := fshelper.ReadCSV[YouTubePlaylistVideo](pfs, "Favorites-videos.csv")
			if err != nil {
				return err
			}
			for i, _ := range favoriteVideos {
				playlistvideo := favoriteVideos[i]
				to.faves[playlistvideo.VideoID] = true
			}
		}

		// VideoID => YouTubeVideoRecording
		recordings := map[string]*YouTubeVideoRecording{}
		ytrecordings, err := fshelper.ReadCSV[YouTubeVideoRecording](vmfs, "video recordings.csv")
		if err != nil {
			return err
		}
		for i, _ := range ytrecordings {
			recording := ytrecordings[i]
			recordings[recording.VideoID] = &recording
		}

		// Finally, add the other metadata to the videos
		videos, err := fshelper.ReadCSV[YouTubeVideo](vmfs, "videos.csv")
		if err != nil {
			return err
		}

		filenames := map[string]int{}
		for i, _ := range videos {
			video := videos[i]

			glob, full := video.Glob()

			// XXX Haven't tested if counts are per basename or
			// XXX include the file extension, i.e., is it:
			//   title.mp4 and title(1).mp4, but title.avi
			// or
			//   title.mp4 and title(1).mp4 and title(2).avi
			count, count_ok := filenames[glob]
			if !count_ok {
				filenames[glob] = 1
			} else {
				filenames[glob] = count + 1
				glob += "(" + strconv.Itoa(count) + ")"
			}

			if full {
				glob += "."
			}
			glob += "*"

			filenames, err := fs.Glob(vfs, glob)
			if err != nil {
				to.log.Record(ctx, fileevent.Error, nil, glob, "reason", "no matching files found")
				continue
			} else if len(filenames) != 1{
				if !count_ok {
					// This is the first instance of this
					// glob, so ignore all the matches that
					// include a counter.  This is only
					// really a problem when !full as well,
					// but there could always be a . in the
					// filename so not including that in
					// the conditional.
					uncountedFilenames := []string{}
					for i, _ := range filenames {
						if !re.MatchString(filenames[i]) {
							uncountedFilenames = append(uncountedFilenames, filenames[i])
						}
					}

					if len(uncountedFilenames) == 1 {
						filenames = uncountedFilenames
					}
				}

				if len(filenames) != 1 {
					to.log.Record(ctx, fileevent.Error, nil, glob, "reason", fmt.Sprintf("%d matching files found", len(filenames)))
					continue
				}
			}

			filename := filenames[0]
			ext := strings.ToLower(path.Ext(filename))
			switch to.sm.TypeFromExt(ext) {
			case immich.TypeImage:
				to.log.Record(ctx, fileevent.DiscoveredImage, nil, filename)
			case immich.TypeVideo:
				to.log.Record(ctx, fileevent.DiscoveredVideo, nil, filename)
			case immich.TypeUnknown:
				to.log.Record(ctx, fileevent.DiscoveredDiscarded, nil, filename, "reason", "unsupported file type")
				continue
			}

			// We've got to get to here to actually determine the
			// filename and increment the extension-specific counter
			// correctly.
			if to.banned.Match(video.Title) {
				to.log.Record(ctx, fileevent.DiscoveredDiscarded, nil, video.Title, "reason", "banned title")
				continue
			} else if to.banned.Match(filename) {
				to.log.Record(ctx, fileevent.DiscoveredDiscarded, nil, filename, "reason", "banned filename")
				continue
			}

			synth := SynthesizedYouTubeVideo{
				Channel:   channels[video.ChannelID],
				Playlists: playlists[video.VideoID],
				Video:     &video,
				Recording: recordings[video.VideoID],
				Fsys:      vfs,
				Filename:  filename,
			}
			to.videos = append(to.videos, &synth)
		}
	}

	return nil
}

// Browse returns a channel of assets
func (to *Takeout) Browse(ctx context.Context) chan *browser.LocalAssetFile {
	assetChan := make(chan *browser.LocalAssetFile)

	go func() {
		defer close(assetChan)
		for _, video := range to.videos {
			fileinfo, err := fs.Stat(video.Fsys, video.Filename)
			if err != nil {
				assetChan <- &browser.LocalAssetFile{Err: err}
				continue
			}

			albums := []browser.LocalAlbum{
				browser.LocalAlbum{
					Path: video.Channel.Title + "'s YouTube channel",
					Title: video.Channel.Title + "'s YouTube channel",
				},
			}
			for _, playlist := range video.Playlists {
				album := browser.LocalAlbum{
					Path: playlist.Title,
					Title: playlist.Title,
					Description: playlist.Description,
				}
				albums = append(albums, album)
			}

			description, title_ok := video.Video.CleanTitle()
			desc, desc_ok := video.Video.CleanDescription()
			if title_ok && desc_ok {
				description += "\n\n" + desc
			} else if !title_ok {
				description = desc
			}

			_, favorite := to.faves[video.Video.VideoID]

			m := metadata.Metadata{
				Description: description,
				DateTaken:   video.Video.Time(),
				Latitude:    video.Recording.Latitude,
				Longitude:   video.Recording.Longitude,
				Altitude:    video.Recording.Altitude,
			}

			a := browser.LocalAssetFile{
				FileName:    video.Filename,
				Title:       video.Video.Title + path.Ext(video.Filename),
				Albums:      albums,

				Metadata:    m,

				Trashed:     false,
				Archived:    false,
				FromPartner: false,
				Favorite:    favorite,

				FSys:        video.Fsys,
				FileSize:    int(fileinfo.Size()),
			}

			select{
			case <-ctx.Done():
				assetChan <- &browser.LocalAssetFile{Err: ctx.Err()}
			case assetChan <- &a:
			}
		}
	}()
	return assetChan
}

// Only exists for testing

func (to *Takeout) Videos() []*SynthesizedYouTubeVideo {
	return to.videos
}
