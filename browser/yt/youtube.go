package yt

import (
	"context"
	"io/fs"
	"path"
	"strconv"

	"github.com/simulot/immich-go/browser"
	"github.com/simulot/immich-go/helpers/fileevent"
	"github.com/simulot/immich-go/helpers/fshelper"
	"github.com/simulot/immich-go/immich"
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
	log        *fileevent.Recorder
	sm         immich.SupportedMedia
}

func NewTakeout(ctx context.Context, l *fileevent.Recorder, sm immich.SupportedMedia, fsyss ...fs.FS) (*Takeout, error) {
	to := Takeout{
		fsyss:  fsyss,
		videos: []*SynthesizedYouTubeVideo{},
		log:    l,
		sm:     sm,
	}

	return &to, nil
}


// Prepare scans all files to build gather and aggregate the metadata
func (to *Takeout) Prepare(ctx context.Context) error {
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
		for i, _ := range ytplaylists {
			playlist := ytplaylists[i]
			if playlist.Title == "Watch later" {
				to.log.Record(ctx, fileevent.DiscoveredDiscarded, nil, "Watch later.csv", "reason", "useless file")
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

			filename := video.Filename()
			count, ok := filenames[filename]
			if !ok {
				filenames[filename] = 1
			} else {
				filenames[filename] = count + 1
				filename = filename + "(" + strconv.Itoa(count) + ")"
			}
			filename += ".mp4"

			to.log.Record(ctx, fileevent.DiscoveredVideo, nil, filename)

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
					Name: video.Channel.Title + "'s YouTube channel",
				},
			}
			for _, playlist := range video.Playlists {
				// Immich albums support having a description,
				// and we have a description of each playlist
				// from playlists.csv, but immich-go doesn't
				// support passing those descriptions through
				// without many changes: UpCmd.updateAlbums is
				// a map from album name to a list of assets,
				// and UpCmd.AddToAlbum would need to have the
				// description added to its many calls.  Or
				// we'd just need a new way to provide album
				// description and I don't understand the code
				// well enough to undertake that.
				album := browser.LocalAlbum{
					Path: playlist.Title,
					Name: playlist.Title,
				}
				albums = append(albums, album)
			}

			a := browser.LocalAssetFile{
				FileName:    video.Filename,
				Title:       video.Video.Title + path.Ext(video.Filename),
				Description: video.Video.Description,
				Albums:      albums,

				DateTaken:   video.Video.Time(),
				Latitude:    video.Recording.Latitude,
				Longitude:   video.Recording.Longitude,
				Altitude:    video.Recording.Altitude,

				Trashed:     false,
				Archived:    false,
				FromPartner: false,
				Favorite:    false,

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
