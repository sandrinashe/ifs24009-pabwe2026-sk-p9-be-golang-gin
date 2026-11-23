package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SearchClient memanggil Search Engine eksternal.
type SearchClient struct {
	Endpoint string
	APIKey   string
	HTTP     *http.Client
}

// NewSearchClient membuat client dengan timeout wajar.
func NewSearchClient(endpoint, key string) *SearchClient {
	return &SearchClient{Endpoint: endpoint, APIKey: key, HTTP: &http.Client{Timeout: 20 * time.Second}}
}

// Search mengambil maksimal 5 rujukan. Bentuk respons JSON dibaca secara toleran
// (results/data/items), karena format Search Engine Delcom perlu diverifikasi.
func (s *SearchClient) Search(ctx context.Context, query string) ([]Source, error) {
	u, err := url.Parse(s.Endpoint)
	if err != nil || s.Endpoint == "" {
		return nil, fmt.Errorf("endpoint search engine tidak valid")
	}
	q := u.Query()
	q.Set("q", query)
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if s.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.APIKey)
	}
	req.Header.Set("Accept", "application/json")
	res, err := s.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("search engine membalas status %d", res.StatusCode)
	}
	return parseSources(body)
}

func pick(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func parseSources(body []byte) ([]Source, error) {
	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("respons search engine bukan JSON")
	}
	list := findList(raw)
	out := []Source{}
	for _, it := range list {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		src := Source{
			Title:   pick(m, "title", "name"),
			URL:     pick(m, "url", "link", "href"),
			Snippet: pick(m, "snippet", "content", "description", "text"),
		}
		if src.URL == "" && src.Title == "" {
			continue
		}
		out = append(out, src)
		if len(out) == 5 {
			break
		}
	}
	return out, nil
}

func findList(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case map[string]any:
		for _, k := range []string{"results", "data", "items", "organic", "web"} {
			if inner, ok := t[k]; ok {
				if l := findList(inner); l != nil {
					return l
				}
			}
		}
	}
	return nil
}
