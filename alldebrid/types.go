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
	Files any `json:"files"`
}

// FileEntry represents a file entry in the torrent tree
type FileEntry struct {
	Link string `json:"link"`
	Path string `json:"path"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// flattenTree recursively flattens the torrent file tree structure
func flattenTree(data any) []FileEntry {
	return flattenTreeWithPath(data, "")
}

// flattenTreeWithPath recursively flattens the torrent file tree structure with path tracking
func flattenTreeWithPath(data any, currentPath string) []FileEntry {
	var files []FileEntry

	switch v := data.(type) {
	case []any:
		for _, item := range v {
			files = append(files, flattenTreeWithPath(item, currentPath)...)
		}
	case map[string]any:
		if e, exists := v["e"]; exists {
			// It's a directory
			var dirName string
			if n, hasN := v["n"]; hasN {
				dirName = n.(string)
			}

			var newPath string
			if currentPath == "" {
				newPath = dirName
			} else if dirName != "" {
				newPath = currentPath + "/" + dirName
			} else {
				newPath = currentPath
			}

			files = append(files, flattenTreeWithPath(e, newPath)...)
		} else if l, hasL := v["l"]; hasL {
			// It's a file
			if n, hasN := v["n"]; hasN {
				if s, hasS := v["s"]; hasS {
					fileName := n.(string)
					var filePath string
					if currentPath == "" {
						filePath = fileName
					} else {
						filePath = currentPath + "/" + fileName
					}

					files = append(files, FileEntry{
						Link: l.(string),
						Path: filePath,
						Name: fileName,
						Size: int64(s.(float64)),
					})
				}
			}
		}
	}

	return files
}
