package main

import (
	"errors"
	"flag"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"alldebrid/alldebrid"
)

type cliArgs struct {
	Link            *string
	Magnet          *string
	TorrentFilePath *string

	Token       string
	PrintAsHTML bool
}

func (a cliArgs) Validate() error {
	// only one of Link, Magnet, or TorrentFilePath should be set
	if a.Link == nil && a.Magnet == nil && a.TorrentFilePath == nil {
		return errors.New("either link, magnet, or torrent file path must be provided")
	}

	if a.Token == "" {
		return errors.New("token must be provided")
	}

	return nil
}

func parseArgs() cliArgs {
	var args cliArgs
	flag.StringVar(&args.Token, "token", os.Getenv("ALLDEBRID_TOKEN"), "Alldebrid API Token")
	flag.BoolVar(&args.PrintAsHTML, "html", false, "Print links as HTML")
	flag.Parse()

	input := flag.Arg(0)
	switch {
	case strings.HasPrefix(input, "magnet:"):
		args.Magnet = &input
	case strings.HasPrefix(input, "http:") || strings.HasPrefix(input, "https:"):
		args.Link = &input
	case strings.HasSuffix(strings.ToLower(input), ".torrent"):
		args.TorrentFilePath = &input
	}

	return args
}

func main() {
	// Configure zerolog
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.DefaultContextLogger = &log.Logger

	// Parse command line arguments
	args := parseArgs()

	// Validate arguments
	if err := args.Validate(); err != nil {
		log.Fatal().Err(err).Msg("invalid arguments")
	}

	client := alldebrid.NewClient(args.Token)
	printer := alldebrid.PrintLinks
	if args.PrintAsHTML {
		printer = alldebrid.PrintLinksAsHTML
	}

	switch {
	case args.Magnet != nil:
		magnetID, err := client.UploadMagnet(*args.Magnet)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to upload magnet")
		}
		links, err := client.WaitForDownloadLinks(magnetID, 10*time.Minute)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get download links")
		}
		printer(links)

	case args.Link != nil:
		link, err := client.UnrestrictURL(*args.Link)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to unrestrict URL")
		}
		printer([]*alldebrid.Link{link})

	case args.TorrentFilePath != nil:
		torrentID, err := client.UploadTorrent(*args.TorrentFilePath)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to upload torrent")
		}
		links, err := client.WaitForDownloadLinks(torrentID, 10*time.Minute)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get download links")
		}
		printer(links)
	}
}
