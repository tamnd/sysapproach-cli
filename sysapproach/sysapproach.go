// Package sysapproach is the library behind the sysapproach CLI.
package sysapproach

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DefaultUserAgent identifies the client to the server. An honest User-Agent
// is both polite and the thing most likely to keep a session unblocked.
const DefaultUserAgent = "sysapproach-cli/dev (+https://github.com/tamnd/sysapproach-cli)"

// Host is the canonical hostname of the book site.
const Host = "book.systemsapproach.org"

// Config holds per-client tunables, all of which have sane defaults.
type Config struct {
	BaseURL   string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
	UserAgent string
}

// DefaultConfig returns a Config with polite, safe defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://" + Host,
		Rate:      500 * time.Millisecond,
		Timeout:   30 * time.Second,
		Retries:   3,
		UserAgent: DefaultUserAgent,
	}
}

// Client talks to book.systemsapproach.org over HTTPS.
type Client struct {
	cfg  Config
	http *http.Client
	last time.Time
}

// NewClient returns a Client using cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

var chapRe = regexp.MustCompile(`href="([\w-]+\.html)"[^>]*>\s*(Chapter (\d+):\s+[^<]+)</a>`)

// Chapters fetches the home page and returns the book's chapters sorted by
// chapter number. Duplicate links (Sphinx repeats chapter links in the sidebar
// and in the main table of contents) are deduplicated by chapter number.
func (c *Client) Chapters(ctx context.Context) ([]*Chapter, error) {
	body, err := c.get(ctx, c.cfg.BaseURL+"/")
	if err != nil {
		return nil, err
	}
	html := string(body)

	seen := map[int]bool{}
	var chapters []*Chapter

	for _, m := range chapRe.FindAllStringSubmatch(html, -1) {
		href := m[1]
		full := m[2] // e.g. "Chapter 1:  Foundation"
		numStr := m[3]

		number, err := strconv.Atoi(numStr)
		if err != nil {
			continue
		}
		if seen[number] {
			continue
		}
		seen[number] = true

		prefix := "Chapter " + numStr + ":"
		title := strings.TrimSpace(strings.TrimPrefix(full, prefix))
		url := c.cfg.BaseURL + "/" + href

		chapters = append(chapters, &Chapter{
			Number: number,
			Title:  title,
			URL:    url,
		})
	}

	sort.Slice(chapters, func(i, j int) bool {
		return chapters[i].Number < chapters[j].Number
	})
	for i, ch := range chapters {
		ch.Rank = i + 1
	}

	return chapters, nil
}

// get fetches url with retries and exponential backoff on transient errors.
func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, url string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

// pace blocks until at least cfg.Rate has elapsed since the last request.
func (c *Client) pace() {
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	return min(time.Duration(attempt)*500*time.Millisecond, 5*time.Second)
}
