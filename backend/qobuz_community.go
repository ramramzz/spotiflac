package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func mapQobuzQualityToCommunity(quality string) string {
	switch strings.TrimSpace(quality) {
	case "27", "7":
		return "24"
	default:
		return "16"
	}
}

func (q *QobuzDownloader) getQobuzCommunityDownloadURL(trackID int64, quality string) (string, error) {
	payload, err := json.Marshal(map[string]string{
		"id":      fmt.Sprintf("%d", trackID),
		"quality": mapQobuzQualityToCommunity(quality),
	})
	if err != nil {
		return "", err
	}

	resp, err := doCommunityRequest(q.client, "Qobuz", func() (*http.Request, error) {
		req, err := NewRequestWithDefaultHeaders(http.MethodPost, GetQobuzCommunityDownloadURL(), bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		if err := setCommunityRequestHeaders(req); err != nil {
			return nil, err
		}
		return req, nil
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("qobuz community API returned status %d", resp.StatusCode)
	}

	downloadURL := extractQobuzStreamingURL(body)
	if downloadURL == "" {
		return "", fmt.Errorf("no streamable URL in qobuz community response")
	}
	return downloadURL, nil
}

func (q *QobuzDownloader) getQobuzCustomDownloadURL(trackID int64, quality string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(q.customURL), "/")
	if base == "" {
		return "", fmt.Errorf("no custom Qobuz instance configured")
	}

	qualityCode := strings.TrimSpace(quality)
	switch qualityCode {
	case "5", "6", "7", "27":
	default:
		qualityCode = "27"
	}

	endpoint := fmt.Sprintf("%s/api/download-music?track_id=%d&quality=%s", base, trackID, url.QueryEscape(qualityCode))
	req, err := NewRequestWithDefaultHeaders(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := q.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("qobuz custom instance returned status %d", resp.StatusCode)
	}

	var parsed struct {
		Success bool `json:"success"`
		Data    struct {
			URL string `json:"url"`
		} `json:"data"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("failed to decode qobuz custom response: %w", err)
	}
	if !parsed.Success || strings.TrimSpace(parsed.Data.URL) == "" {
		if strings.TrimSpace(parsed.Error) != "" {
			return "", fmt.Errorf("qobuz custom instance error: %s", parsed.Error)
		}
		return "", fmt.Errorf("no download URL in qobuz custom response")
	}
	return strings.TrimSpace(parsed.Data.URL), nil
}
