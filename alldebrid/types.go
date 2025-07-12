package alldebrid

// Link represents a downloadable link with metadata
type Link struct {
	Filename    string `json:"filename"`
	URL         string `json:"url"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"download_url,omitempty"`
}

// SizeMB returns the file size in megabytes
func (l *Link) SizeMB() float64 {
	return float64(l.Size) / 1048576
}

// UnlockResponse represents the response from the link unlock API
type UnlockResponse struct {
	Filename string `json:"filename"`
	Filesize int64  `json:"filesize"`
	Link     string `json:"link"`
}

// UploadResponse represents the response from upload APIs
type UploadResponse struct {
	Files   []FileUpload   `json:"files,omitempty"`
	Magnets []MagnetUpload `json:"magnets,omitempty"`
}

// FileUpload represents an uploaded file
type FileUpload struct {
	ID int `json:"id"`
}

// MagnetUpload represents an uploaded magnet
type MagnetUpload struct {
	ID int `json:"id"`
}

// MagnetFilesResponse represents the response from magnet files API
type MagnetFilesResponse struct {
	Magnets []MagnetFiles `json:"magnets"`
}

// MagnetFiles represents the files in a magnet
type MagnetFiles struct {
	Files interface{} `json:"files"`
}

// FileEntry represents a file entry in the torrent tree
type FileEntry struct {
	Link string `json:"link"` // link
	Name string `json:"name"` // name
	Size int64  `json:"size"` // size
}

// DirEntry represents a directory entry in the torrent tree
type DirEntry struct {
	E interface{} `json:"e"` // entries
}

// flattenTree recursively flattens the torrent file tree structure
func flattenTree(data interface{}) []FileEntry {
	var files []FileEntry

	switch v := data.(type) {
	case []interface{}:
		for _, item := range v {
			files = append(files, flattenTree(item)...)
		}
	case map[string]interface{}:
		if e, exists := v["e"]; exists {
			// It's a directory
			files = append(files, flattenTree(e)...)
		} else if l, hasL := v["l"]; hasL {
			// It's a file
			if n, hasN := v["n"]; hasN {
				if s, hasS := v["s"]; hasS {
					files = append(files, FileEntry{
						Link: l.(string),
						Name: n.(string),
						Size: int64(s.(float64)),
					})
				}
			}
		}
	}

	return files
}
