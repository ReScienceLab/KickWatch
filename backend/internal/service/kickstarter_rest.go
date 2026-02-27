package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/kickwatch/backend/internal/model"
)

const restBaseURL = "https://www.kickstarter.com/discover/advanced.json"

type restProject struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Blurb         string `json:"blurb"`
	State         string `json:"state"`
	PercentFunded int    `json:"percent_funded"`
	Goal          string `json:"goal"`
	Pledged       string `json:"pledged"`
	Currency      string `json:"currency"`
	Deadline      int64  `json:"deadline"`
	URL           string `json:"urls"`
	Slug          string `json:"slug"`
	Photo         struct {
		Full string `json:"full"`
	} `json:"photo"`
	Creator struct {
		Name string `json:"name"`
	} `json:"creator"`
	Category struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		ParentID *int   `json:"parent_id"`
	} `json:"category"`
	URLs struct {
		Web struct {
			Project string `json:"project"`
		} `json:"web"`
	} `json:"urls"`
}

type restResponse struct {
	Projects []restProject `json:"projects"`
}

type KickstarterRESTClient struct {
	httpClient *http.Client
}

func NewKickstarterRESTClient() *KickstarterRESTClient {
	return &KickstarterRESTClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *KickstarterRESTClient) DiscoverCampaigns(categoryID string, sort string, page int) ([]model.Campaign, error) {
	params := url.Values{}
	params.Set("sort", sort)
	params.Set("page", strconv.Itoa(page))
	params.Set("per_page", "20")
	if categoryID != "" {
		params.Set("category_id", categoryID)
	}

	reqURL := restBaseURL + "?" + params.Encode()
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rest discover: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rest discover: status %d", resp.StatusCode)
	}

	var result restResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("rest decode: %w", err)
	}

	campaigns := make([]model.Campaign, 0, len(result.Projects))
	for _, p := range result.Projects {
		goal, _ := strconv.ParseFloat(p.Goal, 64)
		pledged, _ := strconv.ParseFloat(p.Pledged, 64)
		deadline := time.Unix(p.Deadline, 0)

		campaigns = append(campaigns, model.Campaign{
			PID:           strconv.FormatInt(p.ID, 10),
			Name:          p.Name,
			Blurb:         p.Blurb,
			PhotoURL:      p.Photo.Full,
			GoalAmount:    goal,
			GoalCurrency:  p.Currency,
			PledgedAmount: pledged,
			Deadline:      deadline,
			State:         p.State,
			CategoryID:    strconv.Itoa(p.Category.ID),
			CategoryName:  p.Category.Name,
			ProjectURL:    p.URLs.Web.Project,
			CreatorName:   p.Creator.Name,
			PercentFunded: float64(p.PercentFunded),
			Slug:          p.Slug,
		})
	}
	return campaigns, nil
}
