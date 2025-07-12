package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"alldebrid/alldebrid"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

type cliArgs struct {
	Link            *string
	Magnet          *string
	TorrentFilePath *string

	Token                    string
	PrintAsHTML              bool
	PrintAsJSON              bool
	Debug                    bool
	Version                  bool
	Timeout                  time.Duration
	IgnoreFilesSmallerThanMB float64
}

func (a cliArgs) Validate() error {
	// Skip validation if version flag is set
	if a.Version {
		return nil
	}

	if a.Link == nil && a.Magnet == nil && a.TorrentFilePath == nil {
		return errors.New("either link, magnet, or torrent file path must be provided")
	}

	if a.Token == "" {
		return errors.New("token must be provided")
	}

	if a.PrintAsHTML && a.PrintAsJSON {
		return errors.New("only one of html or json output can be specified")
	}

	return nil
}

func parseArgs() cliArgs {
	var args cliArgs
	flag.StringVar(&args.Token, "token", os.Getenv("ALLDEBRID_TOKEN"), "Alldebrid API Token")
	flag.BoolVar(&args.PrintAsHTML, "html", false, "Print links as HTML")
	flag.BoolVar(&args.PrintAsJSON, "json", false, "Print links as JSON")
	flag.BoolVar(&args.Debug, "debug", false, "Enable debug mode")
	flag.BoolVar(&args.Version, "version", false, "Print version information")
	flag.DurationVar(&args.Timeout, "timeout", 10*time.Minute, "Timeout for waiting for download links (default: 10m)")
	flag.Float64Var(&args.IgnoreFilesSmallerThanMB, "ignore-files-smaller-than-mb", 5.0, "Ignore files smaller than this size in MB (default: 5.0)")
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

// filterLargeFiles filters links to only include files larger than the specified size in MB
func filterLargeFiles(links []*alldebrid.Link, minSizeMB float64) []*alldebrid.Link {
	var largeFiles []*alldebrid.Link
	for _, link := range links {
		if link.SizeMB() > minSizeMB {
			largeFiles = append(largeFiles, link)
		}
	}
	return largeFiles
}

func run(ctx context.Context, args cliArgs) error {
	client := alldebrid.NewClient(args.Token)

	var links []*alldebrid.Link
	switch {
	case args.Magnet != nil:
		magnetID, err := client.UploadMagnet(ctx, *args.Magnet)
		if err != nil {
			return fmt.Errorf("failed to upload magnet: %w", err)
		}
		links, err = client.WaitForDownloadLinks(ctx, magnetID, args.Timeout)
		if err != nil {
			return fmt.Errorf("failed to get download links: %w", err)
		}

	case args.Link != nil:
		link, err := client.UnrestrictURL(ctx, *args.Link)
		if err != nil {
			return fmt.Errorf("failed to unrestrict URL: %w", err)
		}
		links = []*alldebrid.Link{link}

	case args.TorrentFilePath != nil:
		torrentID, err := client.UploadTorrent(ctx, *args.TorrentFilePath)
		if err != nil {
			return fmt.Errorf("failed to upload torrent: %w", err)
		}
		links, err = client.WaitForDownloadLinks(ctx, torrentID, args.Timeout)
		if err != nil {
			return fmt.Errorf("failed to get download links: %w", err)
		}
	}

	if args.IgnoreFilesSmallerThanMB > 0 {
		links = lo.Filter(links, func(link *alldebrid.Link, _ int) bool {
			return link.SizeMB() >= args.IgnoreFilesSmallerThanMB
		})
	}

	printLinksFn := PrintLinks
	if args.PrintAsHTML {
		printLinksFn = PrintLinksAsHTML
	} else if args.PrintAsJSON {
		printLinksFn = PrintLinksAsJSON
	}

	printLinksFn(links)

	return nil
}

func main() {
	args := parseArgs()

	// Print version information and exit if version flag is set
	if args.Version {
		fmt.Println(version)
		return
	}

	level := zerolog.InfoLevel
	if args.Debug {
		level = zerolog.DebugLevel
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).Level(level)
	zerolog.DefaultContextLogger = &log.Logger

	if err := args.Validate(); err != nil {
		log.Fatal().Err(err).Msg("invalid arguments")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, args); err != nil {
		if ctx.Err() != nil {
			switch ctx.Err() {
			case context.Canceled:
				log.Info().Msg("operation cancelled by user")
			case context.DeadlineExceeded:
				log.Error().Msg("operation timed out")
			}
			os.Exit(1)
		}
		log.Fatal().Err(err).Msg("operation failed")
	}
}
