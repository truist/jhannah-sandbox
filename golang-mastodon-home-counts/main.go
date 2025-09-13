package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/mattn/go-mastodon"
)

type countEntry struct {
	Acct  string
	Count int
}

func sortByCountDescending(m map[string]int) []countEntry {
	var sorted []countEntry
	for k, v := range m {
		if v > 1 { // filter out single-count entries
			sorted = append(sorted, countEntry{Acct: k, Count: v})
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Count > sorted[j].Count
	})
	return sorted
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	server := os.Getenv("MASTODON_SERVER")
	token := os.Getenv("MASTODON_ACCESS_TOKEN")

	cutoffHoursStr := os.Getenv("MASTODON_CUTOFF_HOURS")
	cutoffHours, err := strconv.Atoi(cutoffHoursStr)
	if err != nil || cutoffHours <= 0 {
		log.Fatalf("Invalid MASTODON_CUTOFF_HOURS: %v", cutoffHoursStr)
	}
	cutoff := time.Now().Add(-time.Duration(cutoffHours) * time.Hour)

	if server == "" || token == "" {
		log.Fatal("MASTODON_SERVER and MASTODON_ACCESS_TOKEN must be set in .env")
	}

	client := mastodon.NewClient(&mastodon.Config{
		Server:      server,
		AccessToken: token,
	})

	tootCount := make(map[string]int)
	boostCount := make(map[string]int)

	var maxID mastodon.ID
	limit := 40
	pageNum := 1

	for {
		fmt.Printf("Fetching page #%d...\n", pageNum)

		pg := &mastodon.Pagination{
			MaxID: maxID,
			Limit: int64(limit),
		}

		statuses, err := client.GetTimelineHome(context.Background(), pg)
		if err != nil {
			log.Fatalf("Error fetching timeline: %v", err)
		}
		if len(statuses) == 0 {
			break
		}

		stop := false
		for _, status := range statuses {
			if status.CreatedAt.Before(cutoff) {
				stop = true
				break
			}
			if status.Reblog != nil {
				acct := status.Reblog.Account.Acct
				boostCount[acct]++
			} else {
				acct := status.Account.Acct
				tootCount[acct]++
			}
		}

		if stop {
			break
		}

		maxID = statuses[len(statuses)-1].ID
		pageNum++
	}

	// Calculate totals
	totalToots := 0
	for _, count := range tootCount {
		totalToots += count
	}
	totalBoosts := 0
	for _, count := range boostCount {
		totalBoosts += count
	}
	totalCombined := totalToots + totalBoosts

	fmt.Printf("\n📊 Total timeline posts in last %d hours:\n", cutoffHours)
	fmt.Printf("📝 Toots: %d\n", totalToots)
	fmt.Printf("🔁 Boosts: %d\n", totalBoosts)
	fmt.Printf("➕ Combined: %d\n", totalCombined)

	// Display sorted summaries
	fmt.Println("\n📊 Top Toots:")
	for _, entry := range sortByCountDescending(tootCount) {
		fmt.Printf("👤 @%s → %d\n", entry.Acct, entry.Count)
	}

	fmt.Println("\n📊 Top Boosts:")
	for _, entry := range sortByCountDescending(boostCount) {
		fmt.Printf("👤 @%s → %d\n", entry.Acct, entry.Count)
	}

	// Build combined counts
	combinedCount := make(map[string]int)
	for acct, count := range tootCount {
		combinedCount[acct] += count
	}
	for acct, count := range boostCount {
		combinedCount[acct] += count
	}

	fmt.Println("\n📊 Top Combined (Toots + Boosts):")
	for _, entry := range sortByCountDescending(combinedCount) {
		fmt.Printf("👤 @%s → %d\n", entry.Acct, entry.Count)
	}
}
