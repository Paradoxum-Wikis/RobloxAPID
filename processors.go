package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"robloxapid/pkg/checker"
	"robloxapid/pkg/config"
	"robloxapid/pkg/fetcher"
	"robloxapid/pkg/storage"
	"robloxapid/pkg/wiki"
)

func processEndpoint(wikiClient *wiki.WikiClient, cfg *config.Config, endpointType, id, category string) error {
	urlTemplate, ok := cfg.DynamicEndpoints.APIMap[endpointType]
	if !ok {
		return fmt.Errorf("unknown endpoint type: %s", endpointType)
	}

	url := fmt.Sprintf(urlTemplate, id)
	path := fmt.Sprintf("%s-%s.json", endpointType, id)

	var (
		newData []byte
		err     error
	)

	switch endpointType {
	case "users", "groups":
		if cfg.OpenCloud.APIKey == "" {
			return fmt.Errorf("open cloud api key required for %s", endpointType)
		}
		headers := map[string]string{
			"x-api-key": cfg.OpenCloud.APIKey,
			"Accept":    "application/json",
		}
		newData, err = fetcher.FetchWithHeaders(url, headers)
	default:
		newData, err = fetcher.Fetch(url)
	}

	if err != nil {
		return fmt.Errorf("error fetching data from %s: %v", url, err)
	}

	hasChanged, err := checker.HasChanged(path, newData)
	if err != nil {
		return fmt.Errorf("error checking changes for %s: %v", path, err)
	}

	log.Printf("Updating data for %s...", url)
	dataToPush, err := storage.Save(path, newData)
	if err != nil {
		return fmt.Errorf("error saving data to %s: %v", path, err)
	}

	if !hasChanged {
		log.Printf("No meaningful changes for %s (only roLastUpdated or none), skipping wiki push.", url)
		return nil
	}

	log.Printf("Meaningful changes detected for %s, pushing to wiki.", url)
	wikiTitle := fmt.Sprintf("%s:roapid/%s-%s.json", cfg.Wiki.Namespace, endpointType, id)
	summary := fmt.Sprintf("Automated update from %s", url)
	err = wikiClient.Push(wikiTitle, string(dataToPush), summary)
	if err != nil {
		return fmt.Errorf("error pushing to wiki for %s: %v", wikiTitle, err)
	}

	if err := wikiClient.PurgeCategoryMembers(category); err != nil {
		log.Printf("Error purging pages for %s: %v", category, err)
	}

	log.Printf("Successfully updated %s", wikiTitle)
	return nil
}

func processAboutEndpoint(wikiClient *wiki.WikiClient, cfg *config.Config) error {
	const aboutFilename = "about.json"
	localPath := filepath.Join("config", aboutFilename)

	aboutJSON, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", localPath, err)
	}

	hasChanged, err := checker.HasChanged(aboutFilename, aboutJSON)
	if err != nil {
		return fmt.Errorf("error checking changes for %s: %w", aboutFilename, err)
	}
	if !hasChanged {
		log.Printf("%s unchanged; skipping wiki update.", aboutFilename)
		return nil
	}

	dataToPush, err := storage.Save(aboutFilename, aboutJSON)
	if err != nil {
		return fmt.Errorf("error saving about data: %w", err)
	}

	wikiTitle := fmt.Sprintf("%s:roapid/about.json", cfg.Wiki.Namespace)
	summary := "Automated sync of about information"
	if err := wikiClient.Push(wikiTitle, string(dataToPush), summary); err != nil {
		return fmt.Errorf("error pushing about page to wiki: %w", err)
	}

	if err := wikiClient.PurgePages([]string{wikiTitle}); err != nil {
		log.Printf("Error purging %s: %v", wikiTitle, err)
	}

	log.Printf("Successfully synced %s", wikiTitle)
	return nil
}

func processBadgesRoot(wikiClient *wiki.WikiClient, cfg *config.Config) error {
	const badgesFilename = "badges.json"
	localPath := filepath.Join("config", badgesFilename)

	content, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", localPath, err)
	}

	hasChanged, err := checker.HasChanged(badgesFilename, content)
	if err != nil {
		return fmt.Errorf("error checking changes for %s: %w", badgesFilename, err)
	}
	if !hasChanged {
		log.Printf("%s unchanged; skipping wiki update.", badgesFilename)
		return nil
	}

	dataToPush, err := storage.Save(badgesFilename, content)
	if err != nil {
		return fmt.Errorf("error saving badges data: %w", err)
	}

	wikiTitle := fmt.Sprintf("%s:roapid/badges.json", cfg.Wiki.Namespace)
	summary := "Automated sync of badges index"
	if err := wikiClient.Push(wikiTitle, string(dataToPush), summary); err != nil {
		return fmt.Errorf("error pushing badges index to wiki: %w", err)
	}

	if err := wikiClient.PurgePages([]string{wikiTitle}); err != nil {
		log.Printf("Error purging %s: %v", wikiTitle, err)
	}

	log.Printf("Successfully synced %s", wikiTitle)
	return nil
}
