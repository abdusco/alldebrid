package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"alldebrid/alldebrid"
)

// PrintLinks prints download URLs to stdout
func PrintLinks(links []*alldebrid.Link) {
	for _, link := range links {
		if link.DownloadURL != "" {
			fmt.Println(link.DownloadURL)
		}
	}
}

// PrintLinksAsHTML prints download links as an HTML table
func PrintLinksAsHTML(links []*alldebrid.Link) {
	css := `
        body {
            margin: 0;
            padding: 1rem;
            font-family: consolas, menlo, monospace;
            font-size: 14px;
        }
        table {
            border-collapse: collapse;
            width: 100%;
        }
        th, td {
            border: 1px solid black;
            padding: 8px;
            text-align: left;
        }
        td.numeric {
            text-align: right;
        }
    `

	var rowHTMLs []string
	for _, link := range links {
		if link.DownloadURL == "" {
			continue
		}
		encodedLink := url.QueryEscape(link.DownloadURL)
		alfredLink := fmt.Sprintf("alfred://runtrigger/piracy/direct_link/?argument=%s", encodedLink)
		rowHTML := fmt.Sprintf(`
            <tr>
                <td><a href='%s'>%s</a></td>
                <td class='numeric'>%.1f</td>
                <td><a href='%s'>Alfred</a></td>
            </tr>
        `, link.DownloadURL, link.Filename, link.SizeMB(), alfredLink)
		rowHTMLs = append(rowHTMLs, rowHTML)
	}

	tableHTML := fmt.Sprintf(`
        <style>%s</style>
        <table>
            <thead><tr>
                <th>Filename</th>
                <th>Size MB</th>
                <th>Action</th>
            </tr></thead>
            <tbody>%s</tbody>
        </table>
    `, css, strings.Join(rowHTMLs, ""))

	fmt.Print(tableHTML)
}

// PrintLinksAsJSON prints download links as JSON
func PrintLinksAsJSON(links []*alldebrid.Link) {
	type LinkOutput struct {
		Filename    string  `json:"filename"`
		DownloadURL string  `json:"download_url"`
		SizeBytes   int64   `json:"size_bytes"`
		SizeMB      float64 `json:"size_mb"`
	}

	output := make([]LinkOutput, 0, len(links))

	for _, link := range links {
		if link.DownloadURL != "" {
			linkData := LinkOutput{
				Filename:    link.Filename,
				DownloadURL: link.DownloadURL,
				SizeBytes:   link.Size,
				SizeMB:      link.SizeMB(),
			}
			output = append(output, linkData)
		}
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}

	fmt.Println(string(jsonData))
}
