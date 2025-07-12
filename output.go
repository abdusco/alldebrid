package main

import (
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
        <script>window.app.setFloating(false)</script>
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
