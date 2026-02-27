package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/kickwatch/backend/internal/model"
)

// parseDiscoverPageHTML parses Kickstarter discover page HTML and extracts campaign data
func parseDiscoverPageHTML(html string) ([]model.Campaign, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	var campaigns []model.Campaign

	// Find project cards - Kickstarter uses React components with data attributes
	doc.Find("[data-project]").Each(func(i int, s *goquery.Selection) {
		dataProject, exists := s.Attr("data-project")
		if !exists {
			return
		}

		// Parse JSON data attribute
		var projectData map[string]interface{}
		if err := json.Unmarshal([]byte(dataProject), &projectData); err != nil {
			return
		}

		campaign := extractCampaignFromData(projectData)
		if campaign.PID != "" {
			campaigns = append(campaigns, campaign)
		}
	})

	// Fallback: parse from HTML structure if no data attributes found
	if len(campaigns) == 0 {
		campaigns = parseFromHTMLStructure(doc)
	}

	return campaigns, nil
}

func extractCampaignFromData(data map[string]interface{}) model.Campaign {
	campaign := model.Campaign{}

	// Extract PID
	if pid, ok := data["id"].(float64); ok {
		campaign.PID = fmt.Sprintf("%.0f", pid)
	} else if pid, ok := data["id"].(string); ok {
		campaign.PID = pid
	}

	// Extract name
	if name, ok := data["name"].(string); ok {
		campaign.Name = name
	}

	// Extract blurb
	if blurb, ok := data["blurb"].(string); ok {
		campaign.Blurb = blurb
	}

	// Extract photo URL
	if photo, ok := data["photo"].(map[string]interface{}); ok {
		if url, ok := photo["full"].(string); ok {
			campaign.PhotoURL = url
		} else if url, ok := photo["1024x576"].(string); ok {
			campaign.PhotoURL = url
		}
	}

	// Extract goal and pledged
	if goal, ok := data["goal"].(float64); ok {
		campaign.GoalAmount = goal
	}
	if pledged, ok := data["pledged"].(float64); ok {
		campaign.PledgedAmount = pledged
	}

	// Extract currency
	if currency, ok := data["currency"].(string); ok {
		campaign.GoalCurrency = currency
	}

	// Extract deadline
	if deadline, ok := data["deadline"].(float64); ok {
		campaign.Deadline = time.Unix(int64(deadline), 0)
	}

	// Extract state
	if state, ok := data["state"].(string); ok {
		campaign.State = state
	}

	// Extract percent funded
	if percentFunded, ok := data["percent_funded"].(float64); ok {
		campaign.PercentFunded = percentFunded
	}

	// Extract creator
	if creator, ok := data["creator"].(map[string]interface{}); ok {
		if name, ok := creator["name"].(string); ok {
			campaign.CreatorName = name
		}
	}

	// Extract category
	if category, ok := data["category"].(map[string]interface{}); ok {
		if id, ok := category["id"].(float64); ok {
			campaign.CategoryID = fmt.Sprintf("%.0f", id)
		} else if id, ok := category["id"].(string); ok {
			campaign.CategoryID = id
		}
		if name, ok := category["name"].(string); ok {
			campaign.CategoryName = name
		}
	}

	// Extract slug
	if slug, ok := data["slug"].(string); ok {
		campaign.Slug = slug
	}

	// Build project URL - prefer canonical URL from urls.web.project
	if urls, ok := data["urls"].(map[string]interface{}); ok {
		if web, ok := urls["web"].(map[string]interface{}); ok {
			if project, ok := web["project"].(string); ok {
				campaign.ProjectURL = project
			}
		}
	}
	// Fallback to building from slug only if no canonical URL provided
	if campaign.ProjectURL == "" && campaign.Slug != "" {
		campaign.ProjectURL = fmt.Sprintf("https://www.kickstarter.com/projects/%s", campaign.Slug)
	}

	return campaign
}

func parseFromHTMLStructure(doc *goquery.Document) []model.Campaign {
	var campaigns []model.Campaign

	// Look for project cards in various possible selectors
	doc.Find(".js-react-proj-card, .project-card, [class*='ProjectCard']").Each(func(i int, s *goquery.Selection) {
		campaign := model.Campaign{}

		// Try to extract from various possible structures
		campaign.Name = s.Find("h3, .project-title, [class*='title']").First().Text()
		campaign.Name = strings.TrimSpace(campaign.Name)

		campaign.Blurb = s.Find("p, .project-blurb, [class*='blurb']").First().Text()
		campaign.Blurb = strings.TrimSpace(campaign.Blurb)

		// Extract image
		if img := s.Find("img").First(); img.Length() > 0 {
			if src, exists := img.Attr("src"); exists {
				campaign.PhotoURL = src
			} else if src, exists := img.Attr("data-src"); exists {
				campaign.PhotoURL = src
			}
		}

		// Extract creator
		campaign.CreatorName = s.Find(".creator, [class*='creator']").First().Text()
		campaign.CreatorName = strings.TrimSpace(campaign.CreatorName)

		// Extract URL
		if link := s.Find("a[href*='/projects/']").First(); link.Length() > 0 {
			if href, exists := link.Attr("href"); exists {
				campaign.ProjectURL = href
				if !strings.HasPrefix(href, "http") {
					campaign.ProjectURL = "https://www.kickstarter.com" + href
				}
				// Extract PID from URL
				campaign.PID = extractPIDFromURL(href)
			}
		}

		if campaign.Name != "" && campaign.ProjectURL != "" {
			campaigns = append(campaigns, campaign)
		}
	})

	return campaigns
}

func extractPIDFromURL(urlStr string) string {
	// Extract project ID from URL like /projects/creator/project-name or /projects/123456789/project-name
	re := regexp.MustCompile(`/projects/([^/]+)/([^/?]+)`)
	matches := re.FindStringSubmatch(urlStr)
	if len(matches) >= 3 {
		// If first part is numeric, that's the PID
		if _, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			return matches[1]
		}
		// Otherwise use creator/slug combination
		return matches[1] + "/" + matches[2]
	}
	return ""
}

// parseGoalPledgedText parses text like "$50,000 pledged of $100,000 goal"
func parseGoalPledgedText(text string) (goal, pledged float64, currency string) {
	// Match patterns like "$50,000" or "£1,234.56"
	re := regexp.MustCompile(`([\$£€¥])?([\d,]+(?:\.\d{2})?)`)
	matches := re.FindAllStringSubmatch(text, -1)

	if len(matches) >= 2 {
		// First match is typically pledged, second is goal
		currency = matches[0][1]
		if currency == "" {
			currency = "USD"
		}

		pledgedStr := strings.ReplaceAll(matches[0][2], ",", "")
		pledged, _ = strconv.ParseFloat(pledgedStr, 64)

		goalStr := strings.ReplaceAll(matches[1][2], ",", "")
		goal, _ = strconv.ParseFloat(goalStr, 64)
	}

	return
}
