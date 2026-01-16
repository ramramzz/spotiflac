package backend

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	SpotifyAuthURL     = "https://accounts.spotify.com/authorize"
	SpotifyTokenURL    = "https://accounts.spotify.com/api/token"
	SpotifyAPIBaseURL  = "https://api.spotify.com/v1"
	SpotifyRedirectURI = "http://localhost:8888/callback"
	SpotifyClientID    = "d89c5ffc44564509a0fb8b4c70f9fa07"
)

type SpotifyAuthClient struct {
	client       *http.Client
	accessToken  string
	refreshToken string
	expiresAt    time.Time
	codeVerifier string
	mu           sync.RWMutex
}

type SpotifyTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type SpotifyUserProfile struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Images      []struct {
		URL    string `json:"url"`
		Height int    `json:"height"`
		Width  int    `json:"width"`
	} `json:"images"`
	Country string `json:"country"`
	Product string `json:"product"`
}

type SpotifyLibraryTrack struct {
	AddedAt string `json:"added_at"`
	Track   struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		DurationMs int    `json:"duration_ms"`
		Explicit   bool   `json:"explicit"`
		ExternalIDs struct {
			ISRC string `json:"isrc"`
		} `json:"external_ids"`
		Album struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			ReleaseDate string `json:"release_date"`
			TotalTracks int    `json:"total_tracks"`
			Images      []struct {
				URL    string `json:"url"`
				Height int    `json:"height"`
				Width  int    `json:"width"`
			} `json:"images"`
			Artists []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"artists"`
		} `json:"album"`
		Artists []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"artists"`
		TrackNumber int `json:"track_number"`
		DiscNumber  int `json:"disc_number"`
	} `json:"track"`
}

type SpotifyLibraryResponse struct {
	Items    []SpotifyLibraryTrack `json:"items"`
	Total    int                   `json:"total"`
	Limit    int                   `json:"limit"`
	Offset   int                   `json:"offset"`
	Next     string                `json:"next"`
	Previous string                `json:"previous"`
}

type SpotifyPlaylistItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Public      bool   `json:"public"`
	Owner       struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"owner"`
	Images []struct {
		URL    string `json:"url"`
		Height int    `json:"height"`
		Width  int    `json:"width"`
	} `json:"images"`
	Tracks struct {
		Total int    `json:"total"`
		Href  string `json:"href"`
	} `json:"tracks"`
	ExternalURLs struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
}

type SpotifyPlaylistsResponse struct {
	Items    []SpotifyPlaylistItem `json:"items"`
	Total    int                   `json:"total"`
	Limit    int                   `json:"limit"`
	Offset   int                   `json:"offset"`
	Next     string                `json:"next"`
	Previous string                `json:"previous"`
}

type SpotifyPlaylistTracksResponse struct {
	Items []struct {
		AddedAt string `json:"added_at"`
		Track   struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			DurationMs int    `json:"duration_ms"`
			Explicit   bool   `json:"explicit"`
			ExternalIDs struct {
				ISRC string `json:"isrc"`
			} `json:"external_ids"`
			Album struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				ReleaseDate string `json:"release_date"`
				TotalTracks int    `json:"total_tracks"`
				Images      []struct {
					URL    string `json:"url"`
					Height int    `json:"height"`
					Width  int    `json:"width"`
				} `json:"images"`
				Artists []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"artists"`
			} `json:"album"`
			Artists []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"artists"`
			TrackNumber int `json:"track_number"`
			DiscNumber  int `json:"disc_number"`
		} `json:"track"`
	} `json:"items"`
	Total    int    `json:"total"`
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
}

type AuthTokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

var globalAuthClient *SpotifyAuthClient
var authClientMu sync.Mutex

func NewSpotifyAuthClient() *SpotifyAuthClient {
	authClientMu.Lock()
	defer authClientMu.Unlock()

	if globalAuthClient != nil {
		return globalAuthClient
	}

	globalAuthClient = &SpotifyAuthClient{
		client: &http.Client{Timeout: 30 * time.Second},
	}

	if err := globalAuthClient.loadTokens(); err != nil {
		fmt.Printf("No saved tokens found: %v\n", err)
	}

	return globalAuthClient
}

func (c *SpotifyAuthClient) getTokenPath() (string, error) {
	dir, err := GetFFmpegDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "spotify_tokens.json"), nil
}

func (c *SpotifyAuthClient) saveTokens() error {
	tokenPath, err := c.getTokenPath()
	if err != nil {
		return err
	}

	c.mu.RLock()
	tokens := AuthTokens{
		AccessToken:  c.accessToken,
		RefreshToken: c.refreshToken,
		ExpiresAt:    c.expiresAt,
	}
	c.mu.RUnlock()

	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tokenPath, data, 0600)
}

func (c *SpotifyAuthClient) loadTokens() error {
	tokenPath, err := c.getTokenPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return err
	}

	var tokens AuthTokens
	if err := json.Unmarshal(data, &tokens); err != nil {
		return err
	}

	c.mu.Lock()
	c.accessToken = tokens.AccessToken
	c.refreshToken = tokens.RefreshToken
	c.expiresAt = tokens.ExpiresAt
	c.mu.Unlock()

	return nil
}

func (c *SpotifyAuthClient) clearTokens() error {
	c.mu.Lock()
	c.accessToken = ""
	c.refreshToken = ""
	c.expiresAt = time.Time{}
	c.mu.Unlock()

	tokenPath, err := c.getTokenPath()
	if err != nil {
		return err
	}

	return os.Remove(tokenPath)
}

func generateRandomString(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}

func generateCodeVerifier() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.RawURLEncoding.EncodeToString(bytes)
}

func generateCodeChallenge(verifier string) string {
	return verifier
}

func (c *SpotifyAuthClient) GetAuthURL() (string, error) {
	c.codeVerifier = generateCodeVerifier()
	state := generateRandomString(16)

	params := url.Values{}
	params.Set("client_id", SpotifyClientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", SpotifyRedirectURI)
	params.Set("scope", "user-library-read playlist-read-private playlist-read-collaborative user-read-private user-read-email")
	params.Set("state", state)
	params.Set("code_challenge_method", "plain")
	params.Set("code_challenge", generateCodeChallenge(c.codeVerifier))

	return fmt.Sprintf("%s?%s", SpotifyAuthURL, params.Encode()), nil
}

func (c *SpotifyAuthClient) StartAuthFlow(ctx context.Context) (string, error) {
	authURL, err := c.GetAuthURL()
	if err != nil {
		return "", err
	}

	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	listener, err := net.Listen("tcp", ":8888")
	if err != nil {
		return "", fmt.Errorf("failed to start callback server: %v", err)
	}

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}

			code := r.URL.Query().Get("code")
			errorParam := r.URL.Query().Get("error")

			if errorParam != "" {
				errChan <- fmt.Errorf("authorization error: %s", errorParam)
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprintf(w, `<html><body><h1>Authorization Failed</h1><p>%s</p><script>window.close();</script></body></html>`, errorParam)
				return
			}

			if code == "" {
				errChan <- fmt.Errorf("no authorization code received")
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprint(w, `<html><body><h1>Authorization Failed</h1><p>No code received</p><script>window.close();</script></body></html>`)
				return
			}

			codeChan <- code
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><body><h1>Authorization Successful!</h1><p>You can close this window and return to SpotiFLAC.</p><script>window.close();</script></body></html>`)
		}),
	}

	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	select {
	case code := <-codeChan:
		return code, nil
	case err := <-errChan:
		return "", err
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(5 * time.Minute):
		return "", fmt.Errorf("authorization timeout")
	}
}

func (c *SpotifyAuthClient) ExchangeCode(code string) error {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", SpotifyRedirectURI)
	data.Set("client_id", SpotifyClientID)
	data.Set("code_verifier", c.codeVerifier)

	req, err := http.NewRequest("POST", SpotifyTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokenResp SpotifyTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return err
	}

	c.mu.Lock()
	c.accessToken = tokenResp.AccessToken
	c.refreshToken = tokenResp.RefreshToken
	c.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	c.mu.Unlock()

	return c.saveTokens()
}

func (c *SpotifyAuthClient) RefreshAccessToken() error {
	c.mu.RLock()
	refreshToken := c.refreshToken
	c.mu.RUnlock()

	if refreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", SpotifyClientID)

	req, err := http.NewRequest("POST", SpotifyTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("token refresh failed: %s", string(body))
	}

	var tokenResp SpotifyTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return err
	}

	c.mu.Lock()
	c.accessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		c.refreshToken = tokenResp.RefreshToken
	}
	c.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	c.mu.Unlock()

	return c.saveTokens()
}

func (c *SpotifyAuthClient) EnsureValidToken() error {
	c.mu.RLock()
	expiresAt := c.expiresAt
	accessToken := c.accessToken
	c.mu.RUnlock()

	if accessToken == "" {
		return fmt.Errorf("not authenticated")
	}

	if time.Now().Add(5 * time.Minute).After(expiresAt) {
		return c.RefreshAccessToken()
	}

	return nil
}

func (c *SpotifyAuthClient) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessToken != "" && c.refreshToken != ""
}

func (c *SpotifyAuthClient) Logout() error {
	return c.clearTokens()
}

func (c *SpotifyAuthClient) makeRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	if err := c.EnsureValidToken(); err != nil {
		return nil, err
	}

	c.mu.RLock()
	accessToken := c.accessToken
	c.mu.RUnlock()

	url := endpoint
	if !strings.HasPrefix(endpoint, "http") {
		url = SpotifyAPIBaseURL + endpoint
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	return c.client.Do(req)
}

func (c *SpotifyAuthClient) GetUserProfile() (*SpotifyUserProfile, error) {
	resp, err := c.makeRequest("GET", "/me", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get user profile: %s", string(body))
	}

	var profile SpotifyUserProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

func (c *SpotifyAuthClient) GetLikedSongs(limit, offset int) (*SpotifyLibraryResponse, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 50 {
		limit = 50
	}

	endpoint := fmt.Sprintf("/me/tracks?limit=%d&offset=%d", limit, offset)
	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get liked songs: %s", string(body))
	}

	var libraryResp SpotifyLibraryResponse
	if err := json.Unmarshal(body, &libraryResp); err != nil {
		return nil, err
	}

	return &libraryResp, nil
}

func (c *SpotifyAuthClient) GetAllLikedSongs(ctx context.Context) ([]SpotifyLibraryTrack, int, error) {
	var allTracks []SpotifyLibraryTrack
	offset := 0
	limit := 50
	total := 0

	for {
		select {
		case <-ctx.Done():
			return allTracks, total, ctx.Err()
		default:
		}

		resp, err := c.GetLikedSongs(limit, offset)
		if err != nil {
			return allTracks, total, err
		}

		total = resp.Total
		allTracks = append(allTracks, resp.Items...)

		if resp.Next == "" || len(resp.Items) == 0 {
			break
		}

		offset += limit
	}

	return allTracks, total, nil
}

func (c *SpotifyAuthClient) GetUserPlaylists(limit, offset int) (*SpotifyPlaylistsResponse, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 50 {
		limit = 50
	}

	endpoint := fmt.Sprintf("/me/playlists?limit=%d&offset=%d", limit, offset)
	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get playlists: %s", string(body))
	}

	var playlistsResp SpotifyPlaylistsResponse
	if err := json.Unmarshal(body, &playlistsResp); err != nil {
		return nil, err
	}

	return &playlistsResp, nil
}

func (c *SpotifyAuthClient) GetAllUserPlaylists(ctx context.Context) ([]SpotifyPlaylistItem, int, error) {
	var allPlaylists []SpotifyPlaylistItem
	offset := 0
	limit := 50
	total := 0

	for {
		select {
		case <-ctx.Done():
			return allPlaylists, total, ctx.Err()
		default:
		}

		resp, err := c.GetUserPlaylists(limit, offset)
		if err != nil {
			return allPlaylists, total, err
		}

		total = resp.Total
		allPlaylists = append(allPlaylists, resp.Items...)

		if resp.Next == "" || len(resp.Items) == 0 {
			break
		}

		offset += limit
	}

	return allPlaylists, total, nil
}

func (c *SpotifyAuthClient) GetPlaylistTracks(playlistID string, limit, offset int) (*SpotifyPlaylistTracksResponse, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 50 {
		limit = 50
	}

	endpoint := fmt.Sprintf("/playlists/%s/tracks?limit=%d&offset=%d", playlistID, limit, offset)
	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get playlist tracks: %s", string(body))
	}

	var tracksResp SpotifyPlaylistTracksResponse
	if err := json.Unmarshal(body, &tracksResp); err != nil {
		return nil, err
	}

	return &tracksResp, nil
}
