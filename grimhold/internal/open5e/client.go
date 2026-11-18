// Package open5e fetches WotC SRD stat blocks from the Open5e API.
package open5e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const srdSlug = "wotc-srd"

// Client caches GET responses and refuses creature or spell sheets
// that are not from the WotC SRD.
type Client struct {
	base  string
	httpc *http.Client
	cache *Cache
}

// New returns a client with the given cache TTL.
func New(ttl time.Duration) *Client {
	return &Client{
		base:  "https://api.open5e.com/v1",
		httpc: &http.Client{Timeout: 10 * time.Second},
		cache: newCache(ttl),
	}
}

// Reap purges expired cache entries until ctx is cancelled.
func (c *Client) Reap(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.cache.purge()
		}
	}
}

// Monster loads one creature by slug. Search is not used: a search for
// "white dragon" also returns non-SRD dragons.
func (c *Client) Monster(ctx context.Context, slug string) (Monster, error) {
	var m Monster
	u := c.base + "/monsters/" + url.PathEscape(slug) + "/"
	if err := c.getJSON(ctx, u, &m); err != nil {
		return Monster{}, err
	}
	if m.DocumentSlug != srdSlug {
		return Monster{}, fmt.Errorf("cette fiche n'est pas utilisable")
	}
	if m.Name == "" {
		return Monster{}, fmt.Errorf("fiche vide")
	}
	return m, nil
}

// Spell loads one spell by slug and requires the WotC SRD.
func (c *Client) Spell(ctx context.Context, slug string) (Spell, error) {
	var s Spell
	u := c.base + "/spells/" + url.PathEscape(slug) + "/"
	if err := c.getJSON(ctx, u, &s); err != nil {
		return Spell{}, err
	}
	if s.DocumentSlug != srdSlug {
		return Spell{}, fmt.Errorf("ce sort n'est pas utilisable")
	}
	if s.Name == "" {
		return Spell{}, fmt.Errorf("sort vide")
	}
	return s, nil
}

// Page loads a monster list. An empty pageURL requests the first page.
// Later pages must be the next or previous URL from the API, unchanged.
func (c *Client) Page(ctx context.Context, pageURL string) (Page, error) {
	if pageURL == "" {
		pageURL = c.base + "/monsters/?limit=10"
	}
	var p Page
	if err := c.getJSON(ctx, pageURL, &p); err != nil {
		return Page{}, fmt.Errorf("bestiaire: %w", err)
	}
	return p, nil
}

func (c *Client) getJSON(ctx context.Context, rawURL string, dest any) error {
	if b, ok := c.cache.get(rawURL); ok {
		if err := json.Unmarshal(b, dest); err != nil {
			return fmt.Errorf("décodage: %w", err)
		}
		return nil
	}
	body, err := c.fetch(ctx, rawURL)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("décodage: %w", err)
	}
	c.cache.set(rawURL, body)
	return nil
}

func (c *Client) fetch(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("requête: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "grimhold/1.0")
	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requête: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("lecture: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("introuvable")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("statut %d", resp.StatusCode)
	}
	return body, nil
}
