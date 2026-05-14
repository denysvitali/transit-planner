// Command router is a small CLI to exercise the Go router locally.
//
// Subcommands:
//
//	route -feed <dir> -from <stop_id> -to <stop_id> -depart HH:MM[:SS] [-max-transfers N]
//	stops -feed <dir> [-prefix P]
//	info  -feed <dir>
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/denysvitali/transit-planner/router"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// run is the testable entry point. It parses the subcommand and dispatches.
func run(args []string, stdout io.Writer) error {
	if len(args) < 1 {
		printUsage(stdout)
		return errors.New("missing subcommand")
	}
	switch args[0] {
	case "route":
		return runRoute(args[1:], stdout)
	case "stops":
		return runStops(args[1:], stdout)
	case "info":
		return runInfo(args[1:], stdout)
	case "-h", "--help", "help":
		printUsage(stdout)
		return nil
	default:
		printUsage(stdout)
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: router <subcommand> [flags]")
	fmt.Fprintln(w, "Subcommands:")
	fmt.Fprintln(w, "  route   plan a trip between two stops")
	fmt.Fprintln(w, "  stops   list stops by name prefix")
	fmt.Fprintln(w, "  info    show feed counts")
}

func runRoute(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("route", flag.ContinueOnError)
	fs.SetOutput(stdout)
	feedDir := fs.String("feed", "", "GTFS feed directory")
	from := fs.String("from", "", "origin stop_id")
	to := fs.String("to", "", "destination stop_id")
	depart := fs.String("depart", "", "departure time HH:MM or HH:MM:SS")
	maxTransfers := fs.Int("max-transfers", 3, "maximum number of transfers")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *feedDir == "" || *from == "" || *to == "" || *depart == "" {
		return errors.New("route: -feed, -from, -to and -depart are required")
	}
	departure, err := parseClockTime(*depart)
	if err != nil {
		return fmt.Errorf("route: %w", err)
	}
	feed, err := router.LoadGTFS(*feedDir)
	if err != nil {
		return fmt.Errorf("load feed: %w", err)
	}
	engine := router.NewEngine(feed)
	itinerary, err := engine.Route(*from, *to, departure, router.Options{MaxTransfers: *maxTransfers})
	if err != nil {
		return fmt.Errorf("route: %w", err)
	}
	printItinerary(stdout, feed, itinerary, *from, *to, departure)
	return nil
}

func runStops(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("stops", flag.ContinueOnError)
	fs.SetOutput(stdout)
	feedDir := fs.String("feed", "", "GTFS feed directory")
	prefix := fs.String("prefix", "", "case-insensitive stop name prefix")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *feedDir == "" {
		return errors.New("stops: -feed is required")
	}
	feed, err := router.LoadGTFS(*feedDir)
	if err != nil {
		return fmt.Errorf("load feed: %w", err)
	}
	want := strings.ToLower(*prefix)
	matches := make([]router.Stop, 0, len(feed.Stops))
	for _, stop := range feed.Stops {
		if want == "" || strings.HasPrefix(strings.ToLower(stop.Name), want) {
			matches = append(matches, stop)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Name == matches[j].Name {
			return matches[i].ID < matches[j].ID
		}
		return matches[i].Name < matches[j].Name
	})
	const limit = 50
	if len(matches) > limit {
		matches = matches[:limit]
	}
	for _, stop := range matches {
		fmt.Fprintf(stdout, "%s\t%s\t%.6f,%.6f\n", stop.ID, stop.Name, stop.Lat, stop.Lon)
	}
	return nil
}

func runInfo(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("info", flag.ContinueOnError)
	fs.SetOutput(stdout)
	feedDir := fs.String("feed", "", "GTFS feed directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *feedDir == "" {
		return errors.New("info: -feed is required")
	}
	feed, err := router.LoadGTFS(*feedDir)
	if err != nil {
		return fmt.Errorf("load feed: %w", err)
	}
	fmt.Fprintf(stdout, "stops:     %d\n", len(feed.Stops))
	fmt.Fprintf(stdout, "routes:    %d\n", len(feed.Routes))
	fmt.Fprintf(stdout, "trips:     %d\n", len(feed.Trips))
	fmt.Fprintf(stdout, "transfers: %d\n", len(feed.Transfers))
	return nil
}

// parseClockTime accepts HH:MM or HH:MM:SS and returns seconds since midnight.
func parseClockTime(value string) (int, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return 0, fmt.Errorf("invalid time %q: want HH:MM or HH:MM:SS", value)
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid hour in %q: %w", value, err)
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid minute in %q: %w", value, err)
	}
	second := 0
	if len(parts) == 3 {
		second, err = strconv.Atoi(parts[2])
		if err != nil {
			return 0, fmt.Errorf("invalid second in %q: %w", value, err)
		}
	}
	if hour < 0 || minute < 0 || minute >= 60 || second < 0 || second >= 60 {
		return 0, fmt.Errorf("invalid time %q", value)
	}
	return hour*3600 + minute*60 + second, nil
}

// formatClockTime renders seconds-since-midnight as HH:MM:SS.
func formatClockTime(seconds int) string {
	if seconds < 0 {
		return "--:--:--"
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func printItinerary(w io.Writer, feed *router.Feed, it router.Itinerary, fromID, toID string, depart int) {
	originName := stopName(feed, fromID)
	destName := stopName(feed, toID)
	fmt.Fprintf(w, "From: %s (%s)\n", originName, fromID)
	fmt.Fprintf(w, "To:   %s (%s)\n", destName, toID)
	fmt.Fprintf(w, "Depart: %s\n", formatClockTime(depart))
	fmt.Fprintln(w, "Legs:")
	for i, leg := range it.Legs {
		switch leg.Mode {
		case "walk":
			fmt.Fprintf(w, "  %d. walk    %s -> %s  (%s - %s)\n",
				i+1, leg.FromStop.Name, leg.ToStop.Name,
				formatClockTime(leg.Departure), formatClockTime(leg.Arrival))
		default:
			label := routeLabel(feed, leg.RouteID, leg.TripID)
			fmt.Fprintf(w, "  %d. %-7s %s -> %s  (%s - %s)\n",
				i+1, label, leg.FromStop.Name, leg.ToStop.Name,
				formatClockTime(leg.Departure), formatClockTime(leg.Arrival))
		}
	}
	fmt.Fprintf(w, "Arrival: %s  Transfers: %d  Legs: %d\n",
		formatClockTime(it.Arrival), it.Transfers, len(it.Legs))
}

func stopName(feed *router.Feed, id string) string {
	if stop, ok := feed.Stops[id]; ok && stop.Name != "" {
		return stop.Name
	}
	return id
}

func routeLabel(feed *router.Feed, routeID, tripID string) string {
	if route, ok := feed.Routes[routeID]; ok {
		if route.ShortName != "" {
			return route.ShortName
		}
		if route.LongName != "" {
			return route.LongName
		}
	}
	if routeID != "" {
		return routeID
	}
	return tripID
}
