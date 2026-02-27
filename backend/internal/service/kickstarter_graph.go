package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/kickwatch/backend/internal/model"
)

const (
	ksBaseURL  = "https://www.kickstarter.com"
	ksGraphURL = "https://www.kickstarter.com/graph"
	sessionTTL = 12 * time.Hour
)

var csrfPattern = regexp.MustCompile(`<meta[^>]+name="csrf-token"[^>]+content="([^"]+)"`)

type graphSession struct {
	cookie    string
	csrfToken string
	fetchedAt time.Time
}

type KickstarterGraphClient struct {
	mu         sync.Mutex
	session    *graphSession
	httpClient *http.Client
}

func NewKickstarterGraphClient() *KickstarterGraphClient {
	return &KickstarterGraphClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *KickstarterGraphClient) ensureSession() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session != nil && time.Since(c.session.fetchedAt) < sessionTTL {
		return nil
	}

	req, _ := http.NewRequest("GET", ksBaseURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("bootstrap session: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read bootstrap body: %w", err)
	}

	matches := csrfPattern.FindSubmatch(body)
	if len(matches) < 2 {
		return fmt.Errorf("csrf token not found in page")
	}
	csrfToken := string(matches[1])

	var sessionCookie string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "_ksr_session" {
			sessionCookie = cookie.Value
			break
		}
	}
	if sessionCookie == "" {
		return fmt.Errorf("_ksr_session cookie not found")
	}

	c.session = &graphSession{
		cookie:    sessionCookie,
		csrfToken: csrfToken,
		fetchedAt: time.Now(),
	}
	log.Println("Kickstarter GraphQL session refreshed")
	return nil
}

func (c *KickstarterGraphClient) doGraphQL(query string, variables map[string]interface{}, result interface{}) error {
	if err := c.ensureSession(); err != nil {
		return err
	}

	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", ksGraphURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("x-csrf-token", c.session.csrfToken)
	req.Header.Set("Cookie", "_ksr_session="+c.session.cookie)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("graphql request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		c.mu.Lock()
		c.session = nil
		c.mu.Unlock()
		return fmt.Errorf("graphql 403: session expired")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("graphql status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

const searchQuery = `
query Search($term: String, $sort: ProjectSort, $categoryId: String, $state: PublicProjectState, $first: Int, $cursor: String) {
  projects(term: $term, sort: $sort, categoryId: $categoryId, state: $state, after: $cursor, first: $first) {
    nodes {
      pid
      name
      state
      deadlineAt
      percentFunded
      url
      image { url(width: 1024) }
      goal { amount currency }
      pledged { amount currency }
      creator { name }
      category { id name }
    }
    totalCount
    pageInfo { endCursor hasNextPage }
  }
}`

type graphSearchResp struct {
	Data struct {
		Projects struct {
			Nodes []struct {
				PID           string  `json:"pid"`
				Name          string  `json:"name"`
				State         string  `json:"state"`
				DeadlineAt    *string `json:"deadlineAt"`
				PercentFunded float64 `json:"percentFunded"`
				URL           string  `json:"url"`
				Image         *struct {
					URL string `json:"url"`
				} `json:"image"`
				Goal *struct {
					Amount   string
					Currency string
				} `json:"goal"`
				Pledged *struct {
					Amount   string
					Currency string
				} `json:"pledged"`
				Creator  *struct{ Name string } `json:"creator"`
				Category *struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"category"`
			} `json:"nodes"`
			TotalCount int `json:"totalCount"`
			PageInfo   struct {
				EndCursor   string `json:"endCursor"`
				HasNextPage bool   `json:"hasNextPage"`
			} `json:"pageInfo"`
		} `json:"projects"`
	} `json:"data"`
}

type SearchResult struct {
	Campaigns   []model.Campaign
	TotalCount  int
	NextCursor  string
	HasNextPage bool
}

func (c *KickstarterGraphClient) Search(term, categoryID, sort, cursor string, first int) (*SearchResult, error) {
	vars := map[string]interface{}{
		"term":  term,
		"sort":  sort,
		"first": first,
		"state": "LIVE",
	}
	if categoryID != "" {
		vars["categoryId"] = categoryID
	}
	if cursor != "" {
		vars["cursor"] = cursor
	}

	var resp graphSearchResp
	if err := c.doGraphQL(searchQuery, vars, &resp); err != nil {
		return nil, err
	}

	campaigns := make([]model.Campaign, 0, len(resp.Data.Projects.Nodes))
	for _, n := range resp.Data.Projects.Nodes {
		cam := model.Campaign{
			PID:        n.PID,
			Name:       n.Name,
			State:      n.State,
			ProjectURL: n.URL,
		}
		if n.Image != nil {
			cam.PhotoURL = n.Image.URL
		}
		if n.Goal != nil {
			cam.GoalAmount, _ = strconv.ParseFloat(n.Goal.Amount, 64)
			cam.GoalCurrency = n.Goal.Currency
		}
		if n.Pledged != nil {
			cam.PledgedAmount, _ = strconv.ParseFloat(n.Pledged.Amount, 64)
		}
		if n.Creator != nil {
			cam.CreatorName = n.Creator.Name
		}
		if n.Category != nil {
			cam.CategoryID = n.Category.ID
			cam.CategoryName = n.Category.Name
		}
		if n.DeadlineAt != nil {
			cam.Deadline, _ = time.Parse(time.RFC3339, *n.DeadlineAt)
		}
		cam.PercentFunded = n.PercentFunded
		campaigns = append(campaigns, cam)
	}

	return &SearchResult{
		Campaigns:   campaigns,
		TotalCount:  resp.Data.Projects.TotalCount,
		NextCursor:  resp.Data.Projects.PageInfo.EndCursor,
		HasNextPage: resp.Data.Projects.PageInfo.HasNextPage,
	}, nil
}

const categoriesQuery = `
query FetchRootCategories {
  rootCategories {
    id
    name
    subcategories {
      nodes { id name parentId }
    }
  }
}`

type graphCategoriesResp struct {
	Data struct {
		RootCategories []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Subcategories struct {
				Nodes []struct {
					ID       string `json:"id"`
					Name     string `json:"name"`
					ParentID string `json:"parentId"`
				} `json:"nodes"`
			} `json:"subcategories"`
		} `json:"rootCategories"`
	} `json:"data"`
}

func (c *KickstarterGraphClient) FetchCategories() ([]model.Category, error) {
	var resp graphCategoriesResp
	if err := c.doGraphQL(categoriesQuery, nil, &resp); err != nil {
		return nil, err
	}

	var cats []model.Category
	for _, rc := range resp.Data.RootCategories {
		cats = append(cats, model.Category{ID: rc.ID, Name: rc.Name})
		for _, sub := range rc.Subcategories.Nodes {
			cats = append(cats, model.Category{ID: sub.ID, Name: sub.Name, ParentID: sub.ParentID})
		}
	}
	return cats, nil
}
