package backend

import (
	"errors"
	"fmt"
	"strings"
)

type resolvedTrackLinks struct {
	TidalURL  string
	AmazonURL string
	DeezerURL string
	ISRC      string
}

const (
	linkResolverProviderSongstats      = "songstats"
	linkResolverProviderDeezerSongLink = "deezer-songlink"
)

func (s *SongLinkClient) resolveSpotifyTrackLinks(spotifyTrackID string, region string) (*resolvedTrackLinks, error) {
	links := &resolvedTrackLinks{}
	var attempts []string

	isrc, err := s.lookupSpotifyISRC(spotifyTrackID)
	if err != nil {
		attempts = append(attempts, fmt.Sprintf("spotify isrc: %v", err))
	} else {
		links.ISRC = isrc
	}

	resolvers := orderedLinkResolvers()
	for _, resolver := range resolvers {
		switch resolver {
		case linkResolverProviderSongstats:
			addedData, songstatsErr := s.resolveLinksViaSongstats(links)
			if songstatsErr != nil {
				attempts = append(attempts, fmt.Sprintf("songstats: %v", songstatsErr))
			} else if addedData {
				fmt.Println("Using Songstats as configured link resolver")
			}
		case linkResolverProviderDeezerSongLink:
			addedData, songLinkErr := s.resolveLinksViaDeezerSongLink(links, spotifyTrackID, region)
			if songLinkErr != nil {
				attempts = append(attempts, fmt.Sprintf("songlink: %v", songLinkErr))
			} else if addedData {
				fmt.Println("Using Songlink as configured link resolver")
			}
		}

		if links.TidalURL != "" && links.AmazonURL != "" {
			return links, nil
		}
	}

	if hasAnySongLinkData(links) {
		return links, nil
	}

	if len(attempts) == 0 {
		attempts = append(attempts, "no streaming URLs found")
	}

	return links, errors.New(strings.Join(attempts, " | "))
}

func orderedLinkResolvers() []string {
	preferred := GetLinkResolverSetting()
	if !GetLinkResolverAllowFallback() {
		if preferred == linkResolverProviderDeezerSongLink {
			return []string{linkResolverProviderDeezerSongLink}
		}
		return []string{linkResolverProviderSongstats}
	}

	if preferred == linkResolverProviderDeezerSongLink {
		return []string{
			linkResolverProviderDeezerSongLink,
			linkResolverProviderSongstats,
		}
	}

	return []string{
		linkResolverProviderSongstats,
		linkResolverProviderDeezerSongLink,
	}
}

func (s *SongLinkClient) resolveLinksViaSongstats(links *resolvedTrackLinks) (bool, error) {
	if links == nil || links.ISRC == "" {
		return false, fmt.Errorf("ISRC is required for Songstats resolver")
	}

	before := *links

	fmt.Printf("Fetching Songstats links for ISRC %s\n", links.ISRC)
	if err := s.populateLinksFromSongstats(links, links.ISRC); err != nil {
		return false, err
	}

	return *links != before, nil
}

func (s *SongLinkClient) resolveLinksViaDeezerSongLink(links *resolvedTrackLinks, spotifyTrackID string, region string) (bool, error) {
	if links == nil {
		return false, fmt.Errorf("links is required for song.link resolver")
	}

	before := *links
	var attempts []string

	if trackID, err := extractSpotifyTrackID(spotifyTrackID); err != nil {
		attempts = append(attempts, fmt.Sprintf("spotify track id: %v", err))
	} else {
		fmt.Printf("Scraping song.link via Spotify track %s\n", trackID)
		data, scrapeErr := s.scrapeSongLinkPage(fmt.Sprintf("https://song.link/s/%s", trackID), region)
		if scrapeErr != nil {
			attempts = append(attempts, fmt.Sprintf("song.link spotify: %v", scrapeErr))
		} else {
			mergeSongLinkScrape(links, data)
		}
	}

	if (links.TidalURL == "" || links.AmazonURL == "") && links.ISRC != "" {
		if links.DeezerURL == "" {
			fmt.Printf("Resolving Deezer track from ISRC %s\n", links.ISRC)
			deezerURL, err := s.lookupDeezerTrackURLByISRC(links.ISRC)
			if err != nil {
				attempts = append(attempts, fmt.Sprintf("deezer isrc: %v", err))
			} else {
				links.DeezerURL = deezerURL
				fmt.Printf("Found Deezer URL: %s\n", links.DeezerURL)
			}
		}

		if links.DeezerURL != "" {
			if deezerTrackID, err := extractDeezerTrackID(links.DeezerURL); err != nil {
				attempts = append(attempts, fmt.Sprintf("deezer track id: %v", err))
			} else {
				fmt.Printf("Scraping song.link via Deezer track %s\n", deezerTrackID)
				data, scrapeErr := s.scrapeSongLinkPage(fmt.Sprintf("https://song.link/d/%s", deezerTrackID), region)
				if scrapeErr != nil {
					attempts = append(attempts, fmt.Sprintf("song.link deezer: %v", scrapeErr))
				} else {
					mergeSongLinkScrape(links, data)
				}
			}
		}
	}

	if *links != before {
		if len(attempts) == 0 {
			return true, nil
		}
		return true, errors.New(strings.Join(attempts, " | "))
	}

	if len(attempts) == 0 {
		attempts = append(attempts, "no links found via song.link")
	}

	return false, errors.New(strings.Join(attempts, " | "))
}
