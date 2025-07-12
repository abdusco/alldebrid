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
	Files []TorrentNode `json:"files"`
}

// TorrentNode represents a node in the torrent file tree (can be a file or directory)
type TorrentNode struct {
	Name    string        `json:"n,omitempty"` // name
	Link    string        `json:"l,omitempty"` // link (only for files)
	Size    float64       `json:"s,omitempty"` // size (only for files)
	Entries []TorrentNode `json:"e,omitempty"` // entries (only for directories)
}

// IsFile returns true if this node represents a file
func (tn *TorrentNode) IsFile() bool {
	return tn.Link != ""
}

// IsDirectory returns true if this node represents a directory
func (tn *TorrentNode) IsDirectory() bool {
	return len(tn.Entries) > 0
}

// FileEntry represents a file entry in the torrent tree
type FileEntry struct {
	Link string `json:"link"`
	Path string `json:"path"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// flattenTree recursively flattens the torrent file tree structure
func flattenTree(nodes []TorrentNode) []FileEntry {
	var allFiles []FileEntry
	for _, node := range nodes {
		allFiles = append(allFiles, flattenTreeWithPath(node, "")...)
	}
	return allFiles
}

// flattenTreeWithPath recursively flattens the torrent file tree structure with path tracking
func flattenTreeWithPath(node TorrentNode, currentPath string) []FileEntry {
	var files []FileEntry

	if node.IsFile() {
		// It's a file
		var filePath string
		if currentPath == "" {
			filePath = node.Name
		} else {
			filePath = currentPath + "/" + node.Name
		}

		files = append(files, FileEntry{
			Link: node.Link,
			Path: filePath,
			Name: node.Name,
			Size: int64(node.Size),
		})
	} else if node.IsDirectory() {
		// It's a directory
		var newPath string
		if currentPath == "" {
			newPath = node.Name
		} else if node.Name != "" {
			newPath = currentPath + "/" + node.Name
		} else {
			newPath = currentPath
		}

		// Process all entries in the directory
		for _, entry := range node.Entries {
			files = append(files, flattenTreeWithPath(entry, newPath)...)
		}
	}

	return files
}
