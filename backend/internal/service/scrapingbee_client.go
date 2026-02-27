package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

const scrapingBeeBaseURL = "https://app.scrapingbee.com/api/v1"

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
		apiKey:      apiKey,
		baseURL:     scrapingBeeBaseURL,
		httpClient:  &http.Client{Timeout: 60 * time.Second},
		rateLimiter: NewRateLimiter(maxConcurrent, 500*time.Millisecond),
	}
}

// FetchHTML fetches raw HTML from a URL using ScrapingBee (5 credits)
func (c *ScrapingBeeClient) FetchHTML(ctx context.Context, targetURL string) (string, error) {
	return c.doRequest(ctx, targetURL, false, "")
}

// ExtractWithAI fetches and extracts data using AI (10 credits)
func (c *ScrapingBeeClient) ExtractWithAI(ctx context.Context, targetURL string, query string) (string, error) {
	return c.doRequest(ctx, targetURL, true, query)
}

func (c *ScrapingBeeClient) doRequest(ctx context.Context, targetURL string, useAI bool, aiQuery string) (string, error) {
	// Rate limiting
	if err := c.rateLimiter.Acquire(ctx); err != nil {
		return "", fmt.Errorf("rate limiter: %w", err)
	}
	defer c.rateLimiter.Release()

	// Build ScrapingBee API URL
	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("url", targetURL)
	params.Set("render_js", "true")

	if useAI && aiQuery != "" {
		params.Set("ai_query", aiQuery)
	}

	reqURL := fmt.Sprintf("%s?%s", c.baseURL, params.Encode())

	// Retry logic
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * 2 * time.Second
			log.Printf("ScrapingBee retry attempt %d after %v", attempt+1, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err != nil {
			return "", fmt.Errorf("create request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http request: %w", err)
			continue
		}
		defer resp.Body.Close()

		// Check for rate limiting or server errors
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

		// Read response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("read response: %w", err)
		}

		// Log success
		credits := resp.Header.Get("Spb-Cost")
		log.Printf("ScrapingBee success: url=%s, credits=%s, useAI=%v", targetURL, credits, useAI)

		return string(body), nil
	}

	return "", fmt.Errorf("failed after 3 attempts: %w", lastErr)
}
