package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/kickwatch/backend/internal/model"
)

type KickstarterScrapingService struct {
	client *ScrapingBeeClient
}

func NewKickstarterScrapingService(apiKey string, maxConcurrent int) *KickstarterScrapingService {
	if maxConcurrent == 0 {
		maxConcurrent = 10 // default
	}
	return &KickstarterScrapingService{
		client: NewScrapingBeeClient(apiKey, maxConcurrent),
	}
}

// Search searches for campaigns using AI extraction (10 credits per request)
func (s *KickstarterScrapingService) Search(term, categoryID, sort, cursor string, first int) (*SearchResult, error) {
	ctx := context.Background()

	// Parse page from cursor (cursor format: "page:N")
	page := 1
	if cursor != "" {
		if _, err := fmt.Sscanf(cursor, "page:%d", &page); err != nil {
			page = 1
		}
	}

	// Build Kickstarter discover URL with page
	discoverURL := s.buildDiscoverURL(term, categoryID, sort, page)

	// Try AI extraction first
	aiQuery := "Extract all projects from this page. For each project return a JSON object with these fields: name, slug, creator_slug (the creator's URL slug, e.g. 'john-doe' from kickstarter.com/projects/john-doe/...), project_url (full canonical Kickstarter URL), goal, pledged, currency, deadline, creator, category, photo_url, blurb."

	aiResult, err := s.client.ExtractWithAI(ctx, discoverURL, aiQuery)
	if err == nil {
		campaigns, parseErr := s.parseAIResponse(aiResult)
		if parseErr == nil && len(campaigns) > 0 {
			log.Printf("AI extraction successful: found %d campaigns (page %d)", len(campaigns), page)

			// Generate next cursor if we got a full page
			nextCursor := ""
			hasNextPage := len(campaigns) >= first
			if hasNextPage {
				nextCursor = fmt.Sprintf("page:%d", page+1)
			}

			return &SearchResult{
				Campaigns:   campaigns,
				TotalCount:  len(campaigns),
				NextCursor:  nextCursor,
				HasNextPage: hasNextPage,
			}, nil
		}
		log.Printf("AI extraction parse failed: %v, falling back to HTML", parseErr)
	} else {
		log.Printf("AI extraction failed: %v, falling back to HTML", err)
	}

	// Fallback to HTML parsing
	html, err := s.client.FetchHTML(ctx, discoverURL)
	if err != nil {
		return nil, fmt.Errorf("fetch HTML: %w", err)
	}

	campaigns, err := parseDiscoverPageHTML(html)
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	log.Printf("HTML parsing successful: found %d campaigns (page %d)", len(campaigns), page)

	// Generate next cursor if we got a full page
	nextCursor := ""
	hasNextPage := len(campaigns) >= first
	if hasNextPage {
		nextCursor = fmt.Sprintf("page:%d", page+1)
	}

	return &SearchResult{
		Campaigns:   campaigns,
		TotalCount:  len(campaigns),
		NextCursor:  nextCursor,
		HasNextPage: hasNextPage,
	}, nil
}

// DiscoverCampaigns fetches campaigns for a specific category using HTML parsing (5 credits)
func (s *KickstarterScrapingService) DiscoverCampaigns(categoryID string, sort string, page int) ([]model.Campaign, error) {
	ctx := context.Background()

	// Build URL
	discoverURL := s.buildDiscoverURL("", categoryID, sort, page)

	// Fetch HTML only (cheaper than AI extraction)
	html, err := s.client.FetchHTML(ctx, discoverURL)
	if err != nil {
		return nil, fmt.Errorf("fetch HTML: %w", err)
	}

	campaigns, err := parseDiscoverPageHTML(html)
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	log.Printf("Discovered %d campaigns for category %s (page %d)", len(campaigns), categoryID, page)

	return campaigns, nil
}

// FetchCategories returns hardcoded category list (0 credits)
func (s *KickstarterScrapingService) FetchCategories() ([]model.Category, error) {
	return kickstarterCategories, nil
}

func (s *KickstarterScrapingService) buildDiscoverURL(term, categoryID, sort string, page int) string {
	baseURL := "https://www.kickstarter.com/discover/advanced"

	params := url.Values{}

	if term != "" {
		params.Set("term", term)
	}

	if categoryID != "" {
		params.Set("category_id", categoryID)
	}

	// Map sort values
	switch sort {
	case "MAGIC", "trending":
		params.Set("sort", "magic")
	case "NEWEST", "newest":
		params.Set("sort", "newest")
	case "END_DATE", "ending":
		params.Set("sort", "end_date")
	default:
		params.Set("sort", "magic")
	}

	if page > 1 {
		params.Set("page", strconv.Itoa(page))
	}

	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

func (s *KickstarterScrapingService) parseAIResponse(jsonData string) ([]model.Campaign, error) {
	// Try to parse as array first
	var campaigns []model.Campaign
	if err := json.Unmarshal([]byte(jsonData), &campaigns); err == nil {
		return campaigns, nil
	}

	// Try to parse as object with projects field
	var response struct {
		Projects []struct {
			Name         string  `json:"name"`
			Slug         string  `json:"slug"`
			CreatorSlug  string  `json:"creator_slug"`
			ProjectURL   string  `json:"project_url"`
			Goal         float64 `json:"goal"`
			Pledged      float64 `json:"pledged"`
			Currency     string  `json:"currency"`
			Deadline     string  `json:"deadline"`
			Creator      string  `json:"creator"`
			Category     string  `json:"category"`
			PhotoURL     string  `json:"photo_url"`
			Blurb        string  `json:"blurb"`
			BackersCount int     `json:"backers_count"`
		} `json:"projects"`
	}

	if err := json.Unmarshal([]byte(jsonData), &response); err != nil {
		return nil, fmt.Errorf("parse AI response: %w", err)
	}

	for _, p := range response.Projects {
		campaign := model.Campaign{
			Name:          p.Name,
			Slug:          p.Slug,
			GoalAmount:    p.Goal,
			PledgedAmount: p.Pledged,
			GoalCurrency:  p.Currency,
			CreatorName:   p.Creator,
			CategoryName:  p.Category,
			PhotoURL:      p.PhotoURL,
			Blurb:         p.Blurb,
		}

		// Parse deadline
		if p.Deadline != "" {
			// Try various date formats
			formats := []string{
				time.RFC3339,
				"2006-01-02",
				"Jan 2 2006",
				"January 2, 2006",
			}
			for _, format := range formats {
				if t, err := time.Parse(format, p.Deadline); err == nil {
					campaign.Deadline = t
					break
				}
			}
		}

		// Use project URL from AI if provided, otherwise build from creator_slug + slug
		if p.ProjectURL != "" {
			campaign.ProjectURL = p.ProjectURL
			campaign.PID = extractPIDFromURL(p.ProjectURL)
			if campaign.PID == "" {
				campaign.PID = campaign.Slug
			}
		} else if p.CreatorSlug != "" && campaign.Slug != "" {
			campaign.ProjectURL = fmt.Sprintf("https://www.kickstarter.com/projects/%s/%s", p.CreatorSlug, campaign.Slug)
			campaign.PID = campaign.Slug
		} else if campaign.Slug != "" {
			// Cannot construct a valid URL without the creator slug; leave ProjectURL empty
			campaign.PID = campaign.Slug
		}

		// Calculate percent funded
		if campaign.GoalAmount > 0 {
			campaign.PercentFunded = (campaign.PledgedAmount / campaign.GoalAmount) * 100
		}

		campaigns = append(campaigns, campaign)
	}

	return campaigns, nil
}
