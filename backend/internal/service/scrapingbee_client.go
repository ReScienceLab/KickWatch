package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const scrapingBeeBaseURL = "https://app.scrapingbee.com/api/v1"

// UsageResult holds the response from /api/v1/usage.
type UsageResult struct {
	MaxCredits         int    `json:"max_api_credit"`
	UsedCredits        int    `json:"used_api_credit"`
	MaxConcurrency     int    `json:"max_concurrency"`
	CurrentConcurrency int    `json:"current_concurrency"`
	RenewalDate        string `json:"renewal_subscription_date"`
}

type ScrapingBeeClient struct {
	apiKey      string
	baseURL     string
	httpClient  *http.Client
	rateLimiter *RateLimiter
}

type RateLimiter struct {
	semaphore    chan struct{}
	requestDelay time.Duration
}

func NewRateLimiter(maxConcurrent int, requestDelay time.Duration) *RateLimiter {
	return &RateLimiter{
		semaphore:    make(chan struct{}, maxConcurrent),
		requestDelay: requestDelay,
	}
}

func (rl *RateLimiter) Acquire(ctx context.Context) error {
	select {
	case rl.semaphore <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (rl *RateLimiter) Release() {
	<-rl.semaphore
	if rl.requestDelay > 0 {
		time.Sleep(rl.requestDelay)
	}
}

func NewScrapingBeeClient(apiKey string, maxConcurrent int) *ScrapingBeeClient {
	return &ScrapingBeeClient{
		apiKey:  apiKey,
		baseURL: scrapingBeeBaseURL,
		// ScrapingBee timeout param is 30s; add 5s margin for network round-trip.
		httpClient:  &http.Client{Timeout: 35 * time.Second},
		rateLimiter: NewRateLimiter(maxConcurrent, 500*time.Millisecond),
	}
}

// FetchHTML fetches raw HTML without JS rendering (1 credit).
// Kickstarter's discover pages are server-side rendered so JS is not needed.
func (c *ScrapingBeeClient) FetchHTML(ctx context.Context, targetURL string) (string, error) {
	return c.doRequest(ctx, targetURL, false, "", "", 0)
}

// FetchHTMLInSession fetches raw HTML using a sticky session_id so all requests
// for the same crawl pass share the same proxy IP (1 credit).
func (c *ScrapingBeeClient) FetchHTMLInSession(ctx context.Context, targetURL string, sessionID int) (string, error) {
	return c.doRequest(ctx, targetURL, false, "", "", sessionID)
}

// ExtractWithAI fetches and extracts data using AI (6 credits: 1 base + 5 AI).
// aiSelector narrows the AI's focus to a CSS selector, speeding up extraction.
func (c *ScrapingBeeClient) ExtractWithAI(ctx context.Context, targetURL, query, aiSelector string) (string, error) {
	return c.doRequest(ctx, targetURL, true, query, aiSelector, 0)
}

// FetchUsage returns the current monthly credit consumption (not rate-limited).
func (c *ScrapingBeeClient) FetchUsage(ctx context.Context) (*UsageResult, error) {
	params := url.Values{}
	params.Set("api_key", c.apiKey)
	reqURL := fmt.Sprintf("%s/usage?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var usage UsageResult
	if err := json.NewDecoder(resp.Body).Decode(&usage); err != nil {
		return nil, err
	}
	return &usage, nil
}

func (c *ScrapingBeeClient) doRequest(ctx context.Context, targetURL string, useAI bool, aiQuery, aiSelector string, sessionID int) (string, error) {
	if err := c.rateLimiter.Acquire(ctx); err != nil {
		return "", fmt.Errorf("rate limiter: %w", err)
	}
	defer c.rateLimiter.Release()

	// buildParams constructs the query string, optionally upgrading to premium proxy.
	buildParams := func(premiumProxy bool) string {
		params := url.Values{}
		params.Set("api_key", c.apiKey)
		params.Set("url", targetURL)
		// Kickstarter discover pages are SSR — render_js=false costs 1 credit (vs 5).
		params.Set("render_js", "false")
		// Fail fast: 30s is more than enough for an SSR page; default is 140s.
		params.Set("timeout", "30000")
		// Forward Accept-Language so the request looks like real browser traffic.
		params.Set("forward_headers", "true")
		if premiumProxy {
			// Residential premium proxy (10 credits) as fallback when standard blocked.
			params.Set("premium_proxy", "true")
		}
		if useAI && aiQuery != "" {
			params.Set("ai_query", aiQuery)
		}
		if useAI && aiSelector != "" {
			params.Set("ai_selector", aiSelector)
		}
		if sessionID > 0 {
			params.Set("session_id", strconv.Itoa(sessionID))
		}
		return fmt.Sprintf("%s?%s", c.baseURL, params.Encode())
	}

	var lastErr error
	premiumProxy := false

	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * 2 * time.Second
			// On the 3rd retry, escalate to premium_proxy (residential IP).
			if attempt == 3 && !premiumProxy {
				premiumProxy = true
				log.Printf("ScrapingBee escalating to premium_proxy for %s", targetURL)
			} else {
				log.Printf("ScrapingBee retry attempt %d after %v", attempt, backoff)
			}
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}

		reqURL := buildParams(premiumProxy)
		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err != nil {
			return "", fmt.Errorf("create request: %w", err)
		}
		// Forward a realistic Accept-Language header to Kickstarter.
		req.Header.Set("Spb-Accept-Language", "en-US,en;q=0.9")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http request: %w", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 429 {
			lastErr = fmt.Errorf("rate limited (429)")
			continue
		}
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error (%d)", resp.StatusCode)
			continue
		}
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("read response: %w", err)
		}

		credits := resp.Header.Get("Spb-Cost")
		log.Printf("ScrapingBee success: url=%s, credits=%s, useAI=%v, premium=%v", targetURL, credits, useAI, premiumProxy)

		return string(body), nil
	}

	return "", fmt.Errorf("failed after 4 attempts: %w", lastErr)
}
