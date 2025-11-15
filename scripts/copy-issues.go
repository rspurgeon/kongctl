package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	githubAPIBase = "https://api.github.com"
	userAgent     = "kongctl-issue-copier/1.0"
)

type Issue struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	State  string   `json:"state"`
	Labels []Label  `json:"labels"`
	User   User     `json:"user"`
	HTMLURL string  `json:"html_url"`
}

type Label struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type User struct {
	Login string `json:"login"`
}

type CreateIssueRequest struct {
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Labels []string `json:"labels,omitempty"`
}

type CreateIssueResponse struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
}

func main() {
	var (
		sourceRepo string
		targetRepo string
		token      string
		dryRun     bool
	)

	flag.StringVar(&sourceRepo, "source", "Kong/kongctl", "Source repository (owner/repo)")
	flag.StringVar(&targetRepo, "target", "rspurgeon/kongctl", "Target repository (owner/repo)")
	flag.StringVar(&token, "token", os.Getenv("GITHUB_TOKEN"), "GitHub personal access token")
	flag.BoolVar(&dryRun, "dry-run", false, "Print issues to be copied without creating them")
	flag.Parse()

	if token == "" {
		fmt.Fprintln(os.Stderr, "Error: GitHub token is required. Set GITHUB_TOKEN environment variable or use -token flag")
		os.Exit(1)
	}

	if err := run(sourceRepo, targetRepo, token, dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(sourceRepo, targetRepo, token string, dryRun bool) error {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	fmt.Printf("Fetching open issues from %s...\n", sourceRepo)
	issues, err := fetchOpenIssues(client, sourceRepo, token)
	if err != nil {
		return fmt.Errorf("failed to fetch issues: %w", err)
	}

	fmt.Printf("Found %d open issues\n\n", len(issues))

	if dryRun {
		fmt.Println("DRY RUN - Issues that would be copied:")
		for _, issue := range issues {
			fmt.Printf("  #%d: %s\n", issue.Number, issue.Title)
			fmt.Printf("       Labels: %s\n", getLabelsString(issue.Labels))
			fmt.Printf("       URL: %s\n\n", issue.HTMLURL)
		}
		return nil
	}

	fmt.Printf("Copying issues to %s...\n\n", targetRepo)

	successCount := 0
	errorCount := 0

	for i, issue := range issues {
		fmt.Printf("[%d/%d] Copying issue #%d: %s\n", i+1, len(issues), issue.Number, issue.Title)

		newIssue, err := createIssue(client, targetRepo, token, issue)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ❌ Failed: %v\n", err)
			errorCount++
			// Continue with other issues even if one fails
			continue
		}

		fmt.Printf("  ✓ Created as issue #%d: %s\n\n", newIssue.Number, newIssue.HTMLURL)
		successCount++

		// Rate limiting: wait between requests to avoid hitting GitHub API limits
		if i < len(issues)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Successfully copied: %d\n", successCount)
	fmt.Printf("  Failed: %d\n", errorCount)
	fmt.Printf("  Total: %d\n", len(issues))

	return nil
}

func fetchOpenIssues(client *http.Client, repo, token string) ([]Issue, error) {
	var allIssues []Issue
	page := 1
	perPage := 100

	for {
		url := fmt.Sprintf("%s/repos/%s/issues?state=open&per_page=%d&page=%d",
			githubAPIBase, repo, perPage, page)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
		}

		var issues []Issue
		if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
			return nil, err
		}

		// Filter out pull requests (they appear in the issues API)
		var filteredIssues []Issue
		for _, issue := range issues {
			// Pull requests have a "pull_request" field, but since we're not
			// unmarshaling it, we can check if the URL contains "/pull/"
			if !strings.Contains(issue.HTMLURL, "/pull/") {
				filteredIssues = append(filteredIssues, issue)
			}
		}

		allIssues = append(allIssues, filteredIssues...)

		// If we got fewer issues than requested, we've reached the last page
		if len(issues) < perPage {
			break
		}

		page++
	}

	return allIssues, nil
}

func createIssue(client *http.Client, repo, token string, sourceIssue Issue) (*CreateIssueResponse, error) {
	// Build the new issue body with reference to the original
	body := fmt.Sprintf("_Copied from original issue: %s_\n\n---\n\n%s",
		sourceIssue.HTMLURL, sourceIssue.Body)

	// Extract label names
	labelNames := make([]string, len(sourceIssue.Labels))
	for i, label := range sourceIssue.Labels {
		labelNames[i] = label.Name
	}

	createReq := CreateIssueRequest{
		Title:  sourceIssue.Title,
		Body:   body,
		Labels: labelNames,
	}

	jsonData, err := json.Marshal(createReq)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/repos/%s/issues", githubAPIBase, repo)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var createdIssue CreateIssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&createdIssue); err != nil {
		return nil, err
	}

	return &createdIssue, nil
}

func getLabelsString(labels []Label) string {
	if len(labels) == 0 {
		return "none"
	}
	names := make([]string, len(labels))
	for i, label := range labels {
		names[i] = label.Name
	}
	return strings.Join(names, ", ")
}
