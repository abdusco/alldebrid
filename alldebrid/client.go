package alldebrid

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/imroc/req/v3"
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
func (c *Client) UnrestrictURL(ctx context.Context, url string) (*Link, error) {
	resp, err := c.client.R().
		SetContext(ctx).
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
func (c *Client) UploadTorrent(ctx context.Context, torrentPath string) (int, error) {
	file, err := os.Open(torrentPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open torrent file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Ctx(ctx).Warn().Err(closeErr).Msg("failed to close torrent file")
		}
	}()

	resp, err := c.client.R().
		SetContext(ctx).
		SetFileUpload(req.FileUpload{
			ParamName: "files[]",
			FileName:  filepath.Base(torrentPath),
			GetFileContent: func() (io.ReadCloser, error) {
				if _, err := file.Seek(0, 0); err != nil {
					return nil, fmt.Errorf("failed to seek file: %w", err)
				}
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
func (c *Client) UploadMagnet(ctx context.Context, magnetURI string) (int, error) {
	resp, err := c.client.R().
		SetContext(ctx).
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
func (c *Client) GetTorrentLinks(ctx context.Context, torrentID int) ([]*Link, error) {
	resp, err := c.client.R().
		SetContext(ctx).
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
	var allFiles []*Link

	for _, file := range files {
		link := &Link{
			URL:      file.Link,
			Filename: file.Name,
			Size:     file.Size,
		}
		allFiles = append(allFiles, link)
	}

	return allFiles, nil
}

func (c *Client) unrestrictLink(ctx context.Context, link *Link) *Link {
	unrestrictedLink, err := c.UnrestrictURL(ctx, link.URL)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("url", link.URL).Msg("failed to unrestrict link")
		return link
	}
	link.DownloadURL = unrestrictedLink.DownloadURL
	return link
}

// WaitForDownloadLinks waits for torrent processing to complete and returns download links
func (c *Client) WaitForDownloadLinks(ctx context.Context, magnetID int, timeout time.Duration) ([]*Link, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			links, err := c.GetTorrentLinks(ctx, magnetID)
			if err != nil {
				log.Ctx(ctx).Warn().Err(err).Msg("failed to get torrent links")
				continue
			}
			if len(links) > 0 {
				// Unrestrict links concurrently
				var wg sync.WaitGroup
				for _, link := range links {
					wg.Add(1)
					go func(l *Link) {
						defer wg.Done()
						c.unrestrictLink(ctx, l)
					}(link)
				}
				wg.Wait()
				return links, nil
			}
		}
	}
}
