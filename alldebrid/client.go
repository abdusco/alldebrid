package alldebrid

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/imroc/req/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Error represents an Alldebrid API error
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// APIResponse represents the standard Alldebrid API response structure
type APIResponse struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

// Client represents the Alldebrid API client
type Client struct {
	client *req.Client
	logger zerolog.Logger
}

// NewClient creates a new Alldebrid client with the provided API token
func NewClient(apiToken string) *Client {
	client := req.C().
		SetTimeout(10 * time.Second).
		SetCommonHeaders(map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", apiToken),
			"User-Agent":    "alldebrid downloader for abdusco",
		})

	return &Client{
		client: client,
		logger: log.With().Str("component", "alldebrid").Logger(),
	}
}

func (c *Client) checkError(resp *req.Response) error {
	var apiResp APIResponse
	if err := resp.UnmarshalJson(&apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Status == "error" && apiResp.Error != nil {
		return *apiResp.Error
	}

	return nil
}

// UnrestrictURL unrestricts a premium link and returns download information
func (c *Client) UnrestrictURL(url string) (*Link, error) {
	resp, err := c.client.R().
		SetQueryParam("link", url).
		Get("https://api.alldebrid.com/v4/link/unlock")

	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if err := c.checkError(resp); err != nil {
		return nil, err
	}

	var apiResp APIResponse
	if err := resp.UnmarshalJson(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var unlockResp UnlockResponse
	if err := json.Unmarshal(apiResp.Data, &unlockResp); err != nil {
		return nil, fmt.Errorf("failed to parse unlock response: %w", err)
	}

	return &Link{
		Filename:    unlockResp.Filename,
		URL:         url,
		Size:        unlockResp.Filesize,
		DownloadURL: unlockResp.Link,
	}, nil
}

// UploadTorrent uploads a torrent file and returns the magnet ID
func (c *Client) UploadTorrent(torrentPath string) (int, error) {
	file, err := os.Open(torrentPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open torrent file: %w", err)
	}
	defer file.Close()

	resp, err := c.client.R().
		SetFileUpload(req.FileUpload{
			ParamName: "files[]",
			FileName:  filepath.Base(torrentPath),
			GetFileContent: func() (io.ReadCloser, error) {
				file.Seek(0, 0)
				return file, nil
			},
		}).
		Post("https://api.alldebrid.com/v4/magnet/upload/file")

	if err != nil {
		return 0, fmt.Errorf("upload failed: %w", err)
	}

	if err := c.checkError(resp); err != nil {
		return 0, err
	}

	var apiResp APIResponse
	if err := resp.UnmarshalJson(&apiResp); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	var uploadResp UploadResponse
	if err := json.Unmarshal(apiResp.Data, &uploadResp); err != nil {
		return 0, fmt.Errorf("failed to parse upload response: %w", err)
	}

	if len(uploadResp.Files) == 0 {
		return 0, fmt.Errorf("no files uploaded")
	}

	return uploadResp.Files[0].ID, nil
}

// UploadMagnet uploads a magnet URI and returns the magnet ID
func (c *Client) UploadMagnet(magnetURI string) (int, error) {
	resp, err := c.client.R().
		SetFormData(map[string]string{
			"magnets[]": magnetURI,
		}).
		Post("https://api.alldebrid.com/v4/magnet/upload")

	if err != nil {
		return 0, fmt.Errorf("upload failed: %w", err)
	}

	if err := c.checkError(resp); err != nil {
		return 0, err
	}

	var apiResp APIResponse
	if err := resp.UnmarshalJson(&apiResp); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	var uploadResp UploadResponse
	if err := json.Unmarshal(apiResp.Data, &uploadResp); err != nil {
		return 0, fmt.Errorf("failed to parse upload response: %w", err)
	}

	if len(uploadResp.Magnets) == 0 {
		return 0, fmt.Errorf("no magnets uploaded")
	}

	return uploadResp.Magnets[0].ID, nil
}

// GetTorrentLinks retrieves download links for a torrent/magnet ID
func (c *Client) GetTorrentLinks(torrentID int) ([]*Link, error) {
	resp, err := c.client.R().
		SetFormData(map[string]string{
			"id[]": fmt.Sprintf("%d", torrentID),
		}).
		Post("https://api.alldebrid.com/v4/magnet/files")

	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if err := c.checkError(resp); err != nil {
		return nil, err
	}

	var apiResp APIResponse
	if err := resp.UnmarshalJson(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var magnetResp MagnetFilesResponse
	if err := json.Unmarshal(apiResp.Data, &magnetResp); err != nil {
		return nil, fmt.Errorf("failed to parse magnet response: %w", err)
	}

	if len(magnetResp.Magnets) == 0 {
		return nil, nil
	}

	files := flattenTree(magnetResp.Magnets[0].Files)
	var largeFiles []*Link

	for _, file := range files {
		link := &Link{
			URL:      file.Link,
			Filename: file.Name,
			Size:     file.Size,
		}
		if link.SizeMB() > 5 {
			largeFiles = append(largeFiles, link)
		}
	}

	if len(largeFiles) == 0 {
		return nil, fmt.Errorf("torrent only contains small files")
	}

	return largeFiles, nil
}

func (c *Client) unrestrictLink(link *Link) *Link {
	unrestrictedLink, err := c.UnrestrictURL(link.URL)
	if err != nil {
		c.logger.Error().Err(err).Str("url", link.URL).Msg("failed to unrestrict link")
		return link
	}
	link.DownloadURL = unrestrictedLink.DownloadURL
	return link
}

// WaitForDownloadLinks waits for torrent processing to complete and returns download links
func (c *Client) WaitForDownloadLinks(magnetID int, timeout time.Duration) ([]*Link, error) {
	start := time.Now()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			links, err := c.GetTorrentLinks(magnetID)
			if err != nil {
				c.logger.Warn().Err(err).Msg("failed to get torrent links")
				continue
			}
			if len(links) > 0 {
				// Unrestrict links concurrently
				var wg sync.WaitGroup
				for _, link := range links {
					wg.Add(1)
					go func(l *Link) {
						defer wg.Done()
						c.unrestrictLink(l)
					}(link)
				}
				wg.Wait()
				return links, nil
			}
		case <-time.After(timeout):
			return nil, fmt.Errorf("timeout waiting for download links")
		}

		if time.Since(start) >= timeout {
			return nil, fmt.Errorf("timeout waiting for download links")
		}
	}
}
