package alldebrid

import (
	"context"
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
type APIResponse[T any] struct {
	Status string `json:"status"`
	Data   T      `json:"data,omitempty"`
	Error  *Error `json:"error,omitempty"`
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
			"User-Agent":    "alldebrid downloader https://github.com/abdusco/alldebrid",
		}).
		SetBaseURL("https://api.alldebrid.com/v4").
		OnAfterResponse(func(client *req.Client, res *req.Response) error {
			if res.IsErrorState() {
				var result struct {
					Error *Error `json:"error,omitempty"`
				}
				if err := res.UnmarshalJson(&result); err != nil {
					return fmt.Errorf("failed to unmarshal error as json: %w", err)
				}
				return result.Error
			}
			return nil
		})

	return &Client{
		client: client,
	}
}

// UnrestrictURL unrestricts a premium link and returns download information
func (c *Client) UnrestrictURL(ctx context.Context, url string) (*Link, error) {
	log.Ctx(ctx).Debug().
		Str("url", url).
		Msg("Starting link unrestriction")

	res, err := c.client.R().
		SetContext(ctx).
		SetQueryParam("link", url).
		Get("/link/unlock")
	if err != nil {
		return nil, fmt.Errorf("failed to send unrestrict request: %w", err)
	}

	var result APIResponse[UnrestrictResponse]
	if err := res.UnmarshalJson(&result); err != nil {
		return nil, fmt.Errorf("failed to parse unlock response: %w", err)
	}

	return &Link{
		Filename:    result.Data.Filename,
		URL:         url,
		Size:        result.Data.Filesize,
		DownloadURL: result.Data.Link,
	}, nil
}

// UploadTorrent uploads a torrent file and returns the magnet ID
func (c *Client) UploadTorrent(ctx context.Context, torrentPath string) (int, error) {
	log.Ctx(ctx).Debug().
		Str("torrent_path", torrentPath).
		Msg("Starting torrent file upload")

	file, err := os.Open(torrentPath)
	if err != nil {
		log.Ctx(ctx).Error().
			Err(err).
			Str("torrent_path", torrentPath).
			Msg("Failed to open torrent file")
		return 0, fmt.Errorf("failed to open torrent file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Ctx(ctx).Warn().Err(closeErr).Msg("failed to close torrent file")
		}
	}()

	res, err := c.client.R().
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
		Post("/magnet/upload/file")
	if err != nil {
		return 0, fmt.Errorf("failed to upload magnet: %w", err)
	}

	var result APIResponse[UploadResponse]
	if err := res.UnmarshalJson(&res); err != nil {
		log.Ctx(ctx).Error().
			Err(err).
			Msg("Failed to parse torrent upload response")
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Data.Files) == 0 {
		return 0, fmt.Errorf("no files uploaded")
	}

	magnetID := result.Data.Files[0].ID

	log.Ctx(ctx).Debug().
		Int("magnet_id", magnetID).
		Str("torrent_path", torrentPath).
		Msg("Torrent uploaded successfully")

	return magnetID, nil
}

// UploadMagnet uploads a magnet URI and returns the magnet ID
func (c *Client) UploadMagnet(ctx context.Context, magnetURI string) (int, error) {
	res, err := c.client.R().
		SetContext(ctx).
		SetFormData(map[string]string{
			"magnets[]": magnetURI,
		}).
		Post("/magnet/upload")
	if err != nil {
		return 0, fmt.Errorf("upload failed: %w", err)
	}

	var result APIResponse[UploadResponse]
	if err := res.UnmarshalJson(&result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Data.Magnets) == 0 {
		log.Ctx(ctx).Error().Msg("No magnets uploaded in response")
		return 0, fmt.Errorf("no magnets uploaded")
	}

	magnetID := result.Data.Magnets[0].ID
	log.Ctx(ctx).Debug().
		Int("magnet_id", magnetID).
		Str("magnet_uri", magnetURI).
		Msg("Magnet uploaded successfully")

	return magnetID, nil
}

// GetTorrentLinks retrieves download links for a torrent/magnet ID
func (c *Client) GetTorrentLinks(ctx context.Context, torrentID int) ([]*Link, error) {
	res, err := c.client.R().
		SetContext(ctx).
		SetFormData(map[string]string{
			"id[]": fmt.Sprintf("%d", torrentID),
		}).
		Post("/magnet/files")

	if err != nil {
		return nil, fmt.Errorf("failed send request to list magnet files: %w", err)
	}

	var result APIResponse[MagnetFilesResponse]
	if err := res.UnmarshalJson(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Data.Magnets) == 0 {
		log.Ctx(ctx).Debug().
			Int("torrent_id", torrentID).
			Msg("No magnets found in response")
		return nil, nil
	}

	files := flattenTree(result.Data.Magnets[0].Files)
	var allFiles []*Link

	for _, file := range files {
		link := &Link{
			URL:      file.Link,
			Filename: file.Name,
			Size:     file.Size,
		}
		allFiles = append(allFiles, link)
	}

	log.Ctx(ctx).Debug().
		Int("torrent_id", torrentID).
		Int("file_count", len(allFiles)).
		Msg("Retrieved torrent links successfully")

	return allFiles, nil
}

func (c *Client) unrestrictLink(ctx context.Context, link *Link) *Link {
	unrestrictedLink, err := c.UnrestrictURL(ctx, link.URL)
	if err != nil {
		log.Ctx(ctx).Error().
			Err(err).
			Str("url", link.URL).
			Str("filename", link.Filename).
			Msg("Failed to unrestrict link")
		return link
	}

	link.DownloadURL = unrestrictedLink.DownloadURL

	return link
}

// WaitForDownloadLinks waits for torrent processing to complete and returns download links
func (c *Client) WaitForDownloadLinks(ctx context.Context, magnetID int, timeout time.Duration) ([]*Link, error) {
	log.Ctx(ctx).Debug().
		Int("magnet_id", magnetID).
		Dur("timeout", timeout).
		Msg("Starting to wait for download links")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	attempts := 0
	for {
		select {
		case <-ctx.Done():
			log.Ctx(ctx).Warn().
				Int("magnet_id", magnetID).
				Int("attempts", attempts).
				Dur("timeout", timeout).
				Msg("Timeout reached while waiting for download links")
			return nil, ctx.Err()
		case <-ticker.C:
			attempts++
			log.Ctx(ctx).Debug().
				Int("magnet_id", magnetID).
				Int("attempt", attempts).
				Msg("Checking for torrent links")

			links, err := c.GetTorrentLinks(ctx, magnetID)
			if err != nil {
				log.Ctx(ctx).Warn().
					Err(err).
					Int("magnet_id", magnetID).
					Int("attempt", attempts).
					Msg("Failed to get torrent links, retrying")
				continue
			}
			if len(links) > 0 {
				log.Ctx(ctx).Debug().
					Int("magnet_id", magnetID).
					Int("link_count", len(links)).
					Int("attempts", attempts).
					Msg("Found download links, starting unrestriction")

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

			log.Ctx(ctx).Debug().
				Int("magnet_id", magnetID).
				Int("attempt", attempts).
				Msg("No links ready yet, waiting for next check")
		}
	}
}
