//go:build integration

package service

import (
	"context"
	"os"
	"strings"
	"testing"
)

// Run with:
//   SCRAPINGBEE_API_KEY=<key> go test -v -tags integration -run TestScrapingBee ./internal/service/
//   SCRAPINGBEE_API_KEY=<key> go test -v -tags integration -timeout 120s ./internal/service/

func testClient(t *testing.T) *ScrapingBeeClient {
	t.Helper()
	key := os.Getenv("SCRAPINGBEE_API_KEY")
	if key == "" {
		t.Fatal("SCRAPINGBEE_API_KEY env var not set")
	}
	return NewScrapingBeeClient(key, 2)
}

func TestScrapingBee_FetchUsage(t *testing.T) {
	client := testClient(t)
	usage, err := client.FetchUsage(context.Background())
	if err != nil {
		t.Fatalf("FetchUsage error: %v", err)
	}
	t.Logf("Usage: %d/%d credits (%.1f%%), renews %s",
		usage.UsedCredits, usage.MaxCredits,
		float64(usage.UsedCredits)/float64(usage.MaxCredits)*100,
		usage.RenewalDate,
	)
	if usage.MaxCredits <= 0 {
		t.Errorf("expected positive MaxCredits, got %d", usage.MaxCredits)
	}
}

// TestScrapingBee_FetchHTMLInSession costs 1 credit.
func TestScrapingBee_FetchHTMLInSession(t *testing.T) {
	client := testClient(t)
	targetURL := "https://www.kickstarter.com/discover/advanced?category_id=16&sort=magic"
	html, err := client.FetchHTMLInSession(context.Background(), targetURL, 42)
	if err != nil {
		t.Fatalf("FetchHTMLInSession error: %v", err)
	}
	if len(html) < 1000 {
		t.Errorf("HTML response suspiciously short: %d bytes", len(html))
	}
	if !strings.Contains(html, "kickstarter") {
		t.Errorf("response does not look like a Kickstarter page")
	}
	t.Logf("FetchHTMLInSession: got %d bytes", len(html))
}

// TestScrapingBee_DiscoverCampaigns costs 1 credit and validates HTML parsing.
func TestScrapingBee_DiscoverCampaigns(t *testing.T) {
	key := os.Getenv("SCRAPINGBEE_API_KEY")
	if key == "" {
		t.Fatal("SCRAPINGBEE_API_KEY env var not set")
	}
	svc := NewKickstarterScrapingService(key, 2)
	// category_id=16 = Technology; sort=magic; page=1
	campaigns, err := svc.DiscoverCampaigns("16", "magic", 1, 42)
	if err != nil {
		t.Fatalf("DiscoverCampaigns error: %v", err)
	}
	if len(campaigns) == 0 {
		t.Fatal("expected at least one campaign, got 0")
	}
	t.Logf("DiscoverCampaigns: got %d campaigns", len(campaigns))
	for i, c := range campaigns {
		if c.PID == "" {
			t.Errorf("campaign[%d] has empty PID", i)
		}
		if c.ProjectURL == "" {
			t.Errorf("campaign[%d] has empty ProjectURL", i)
		}
		t.Logf("  [%d] %s — %s", i, c.PID, c.Name)
	}
}

// TestScrapingBee_Search costs 1 credit and validates the Search() method.
func TestScrapingBee_Search(t *testing.T) {
	key := os.Getenv("SCRAPINGBEE_API_KEY")
	if key == "" {
		t.Fatal("SCRAPINGBEE_API_KEY env var not set")
	}
	svc := NewKickstarterScrapingService(key, 2)
	result, err := svc.Search("keyboard", "16", "magic", "", 12)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(result.Campaigns) == 0 {
		t.Fatal("expected at least one campaign, got 0")
	}
	t.Logf("Search: got %d campaigns, hasNextPage=%v", len(result.Campaigns), result.HasNextPage)
	for i, c := range result.Campaigns {
		t.Logf("  [%d] %s — %s", i, c.PID, c.Name)
	}
}

// TestScrapingBee_ExtractWithAI_RawAPI documents that AI extraction returns
// EMPTY_RESPONSE for [data-project] because project data is stored in HTML
// attributes (not text nodes). This test is informational — it verifies the
// raw API call succeeds and logs what ScrapingBee actually returns.
func TestScrapingBee_ExtractWithAI_RawAPI(t *testing.T) {
	client := testClient(t)
	targetURL := "https://www.kickstarter.com/discover/advanced?category_id=16&sort=magic"
	// Note: [data-project] holds JSON in an attribute, not text — AI sees no text to extract.
	result, err := client.ExtractWithAI(context.Background(), targetURL,
		"Extract all project names visible on the page.", "h2 a")
	if err != nil {
		t.Fatalf("ExtractWithAI error: %v", err)
	}
	preview := result
	if len(preview) > 300 {
		preview = preview[:300]
	}
	t.Logf("ExtractWithAI (h2 a selector) result (%d bytes): %s", len(result), preview)
}
