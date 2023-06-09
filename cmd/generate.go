package cmd

import (
	"context"
	"math/rand"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"

	"github.com/femnad/mare"
	"github.com/femnad/spoaut"
)

const (
	// This is enforced by the API.
	maxSeedsNum         = 5
	configFile          = "~/.config/sporand/sporand.yml"
	playlistName        = "sporand"
	playlistDescription = "import random; random.choice(music)"
	tokenFile           = "~/.local/share/spotify-tokens/sporand.json"
)

var (
	scopes = []string{
		spotifyauth.ScopePlaylistModifyPrivate,
		spotifyauth.ScopePlaylistReadPrivate,
		spotifyauth.ScopeUserReadPrivate,
		spotifyauth.ScopeUserTopRead,
	}
)

func selectRandomElements(all []spotify.ID, count int) []spotify.ID {
	var selected []spotify.ID
	set := mapset.NewSet[spotify.ID](all...)

	for i := 0; i < count; i++ {
		idx := rand.Int() % count
		id := all[idx]
		set.Remove(id)
		all = set.ToSlice()
		selected = append(selected, id)
	}

	return selected
}

func getSeeds(artistIDs []spotify.ID, trackIDs []spotify.ID) spotify.Seeds {
	artistCount := rand.Int()%maxSeedsNum + 1
	selectedArtist := selectRandomElements(artistIDs, artistCount)

	trackCount := 5 - artistCount
	selectedTracks := selectRandomElements(trackIDs, trackCount)

	return spotify.Seeds{
		Artists: selectedArtist,
		Tracks:  selectedTracks,
	}
}

func truncatePlaylist(ctx context.Context, c *spotify.Client, pID spotify.ID) error {
	items, err := c.GetPlaylistItems(ctx, pID)
	if err != nil {
		return err
	}

	trackIDs := mare.Map[spotify.PlaylistItem, spotify.ID](items.Items, func(item spotify.PlaylistItem) spotify.ID {
		return item.Track.Track.ID
	})
	_, err = c.RemoveTracksFromPlaylist(ctx, pID, trackIDs...)
	if err != nil {
		return err
	}

	return nil
}

func getPlaylistID(ctx context.Context, c *spotify.Client) (pID spotify.ID, err error) {
	currentUser, err := c.CurrentUser(ctx)
	if err != nil {
		return
	}

	playlists, err := c.GetPlaylistsForUser(ctx, currentUser.ID)
	if err != nil {
		return
	}

	var found bool
	for _, playlist := range playlists.Playlists {
		if playlist.Name == playlistName && playlist.Description == playlistDescription {
			pID = playlist.ID
			found = true
			break
		}
	}

	if found {
		err = truncatePlaylist(ctx, c, pID)
		return
	}

	playlist, err := c.CreatePlaylistForUser(ctx, currentUser.ID, playlistName, playlistDescription, false, false)
	if err != nil {
		return
	}

	return playlist.ID, nil
}

func Generate(ctx context.Context) error {
	config := spoaut.Config{
		ConfigFile: configFile,
		Scopes:     scopes,
		TokenFile:  tokenFile,
	}

	c, err := spoaut.Client(ctx, config)
	if err != nil {
		return err
	}

	topArtists, err := c.CurrentUsersTopArtists(ctx, spotify.Timerange(spotify.ShortTermRange))
	if err != nil {
		return err
	}
	artistIds := mare.Map[spotify.FullArtist, spotify.ID](topArtists.Artists, func(artist spotify.FullArtist) spotify.ID {
		return artist.ID
	})

	topTracks, err := c.CurrentUsersTopTracks(ctx)
	if err != nil {
		return err
	}
	trackIds := mare.Map[spotify.FullTrack, spotify.ID](topTracks.Tracks, func(track spotify.FullTrack) spotify.ID {
		return track.ID
	})

	seeds := getSeeds(artistIds, trackIds)

	recs, err := c.GetRecommendations(ctx, seeds, &spotify.TrackAttributes{})
	if err != nil {
		return err
	}
	recIds := mare.Map[spotify.SimpleTrack, spotify.ID](recs.Tracks, func(track spotify.SimpleTrack) spotify.ID {
		return track.ID
	})

	playlistID, err := getPlaylistID(ctx, c)
	if err != nil {
		return err
	}
	_, err = c.AddTracksToPlaylist(ctx, playlistID, recIds...)
	if err != nil {
		return err
	}

	return nil
}
