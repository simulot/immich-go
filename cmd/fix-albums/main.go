package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/simulot/immich-go/immich"
)

func main() {
	server := flag.String("server", "", "Immich server URL (e.g. http://synology:2283)")
	apiKey := flag.String("api-key", "", "Immich API key")
	dryRun := flag.Bool("dry-run", true, "Show what would be done without making changes (default: true)")
	flag.Parse()

	if *server == "" || *apiKey == "" {
		fmt.Fprintln(os.Stderr, "Usage: fix-albums --server URL --api-key KEY [--dry-run=false]")
		os.Exit(1)
	}

	ctx := context.Background()

	ic, err := immich.NewImmichClient(*server, *apiKey)
	if err != nil {
		log.Fatalf("Failed to connect to Immich: %v", err)
	}

	user, err := ic.ValidateConnection(ctx)
	if err != nil {
		log.Fatalf("Failed to validate connection: %v", err)
	}
	fmt.Printf("Connected as: %s %s (%s)\n", user.FirstName, user.LastName, user.Email)

	// Get all albums
	albums, err := ic.GetAllAlbums(ctx)
	if err != nil {
		log.Fatalf("Failed to get albums: %v", err)
	}
	fmt.Printf("Found %d albums\n\n", len(albums))

	// Pattern: album name ending with digits directly after a letter (no space/punctuation).
	// This matches iCloud CSV chunk suffixes like "Trip1", "Trip10" but not "Summer 2022".
	// e.g. "Memory Cooking Over the Years3" -> base "Memory Cooking Over the Years"
	suffixPattern := regexp.MustCompile(`^(.*[a-zA-Z_])(\d+)$`)

	// Group albums by base name
	type albumInfo struct {
		id   string
		name string
	}

	groups := make(map[string][]albumInfo)
	for _, a := range albums {
		baseName := a.AlbumName
		if m := suffixPattern.FindStringSubmatch(a.AlbumName); m != nil {
			// Only treat as duplicate if the base name also exists
			baseName = m[1]
		}
		groups[baseName] = append(groups[baseName], albumInfo{id: a.ID, name: a.AlbumName})
	}

	// Find groups with duplicates
	var duplicateGroups []string
	for baseName, group := range groups {
		if len(group) > 1 {
			duplicateGroups = append(duplicateGroups, baseName)
		}
	}
	sort.Strings(duplicateGroups)

	if len(duplicateGroups) == 0 {
		fmt.Println("No duplicate albums found.")
		return
	}

	fmt.Printf("Found %d album groups with duplicates:\n\n", len(duplicateGroups))

	totalMerged := 0
	totalDeleted := 0

	for _, baseName := range duplicateGroups {
		group := groups[baseName]

		// Sort: original (exact base name) first, then by suffix number
		sort.Slice(group, func(i, j int) bool {
			if group[i].name == baseName {
				return true
			}
			if group[j].name == baseName {
				return false
			}
			return group[i].name < group[j].name
		})

		// The first one should be the original
		original := group[0]
		duplicates := group[1:]

		// Verify the original is actually the base name
		if original.name != baseName {
			fmt.Printf("  SKIP: No original album found for base name %q (only suffixed versions exist)\n", baseName)
			continue
		}

		fmt.Printf("  Album: %q\n", baseName)
		fmt.Printf("    Original: %s (id: %s)\n", original.name, original.id)

		for _, dup := range duplicates {
			// Get assets from the duplicate album
			albumContent, err := ic.GetAlbumInfo(ctx, dup.id, false)
			if err != nil {
				fmt.Printf("    ERROR getting album %q: %v\n", dup.name, err)
				continue
			}

			assetIDs := make([]string, 0, len(albumContent.Assets))
			for _, a := range albumContent.Assets {
				assetIDs = append(assetIDs, a.ID)
			}

			fmt.Printf("    Duplicate: %s (%d assets) -> merge into %q\n", dup.name, len(assetIDs), original.name)

			if *dryRun {
				totalMerged += len(assetIDs)
				totalDeleted++
				continue
			}

			// Move assets to original album
			if len(assetIDs) > 0 {
				_, err = ic.AddAssetToAlbum(ctx, original.id, assetIDs)
				if err != nil {
					fmt.Printf("    ERROR adding assets to %q: %v\n", original.name, err)
					continue
				}
				totalMerged += len(assetIDs)
			}

			// Delete the duplicate album
			err = ic.DeleteAlbum(ctx, dup.id)
			if err != nil {
				fmt.Printf("    ERROR deleting album %q: %v\n", dup.name, err)
				continue
			}
			totalDeleted++
			fmt.Printf("    Deleted: %s\n", dup.name)
		}
		fmt.Println()
	}

	if *dryRun {
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("DRY RUN: Would merge %d assets and delete %d duplicate albums.\n", totalMerged, totalDeleted)
		fmt.Println("Run with --dry-run=false to apply changes.")
	} else {
		fmt.Printf("Done: merged %d assets, deleted %d duplicate albums.\n", totalMerged, totalDeleted)
	}
}
