package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Chat abstraksi LLM; Searcher abstraksi mesin pencari.
type (
	Chat interface {
		Complete(ctx context.Context, system, user string) (string, error)
	}
	Searcher interface {
		Search(ctx context.Context, query string) ([]Source, error)
	}
)

// ErrAI dikembalikan bila OpenAI gagal / keluaran tak bisa dipakai.
var ErrAI = errors.New("OpenAI gagal menghasilkan artikel")

// Service orkestrasi search -> prompt -> parse.
type Service struct {
	chat   Chat
	search Searcher
}

// NewService membuat Service.
func NewService(chat Chat, search Searcher) *Service { return &Service{chat: chat, search: search} }

const systemPrompt = `Kamu penulis blog berbahasa Indonesia. Balas HANYA dengan satu objek JSON valid
berkunci: "title", "category", "excerpt", "content". "content" berformat Markdown (minimal 3 subjudul "##").
"category" harus salah satu: Teknologi, Pendidikan, Gaya Hidup, Bisnis, Opini, Umum. "excerpt" 1-2 kalimat.
Jika diberi sumber web, rujuk sebagai [1], [2], dst. dan jangan mengarang fakta di luar sumber.`

// Generate membuat draf artikel. Kegagalan search tidak menggagalkan proses.
func (s *Service) Generate(ctx context.Context, req GenerateRequest, validCategories []string) (*Draft, error) {
	topic := strings.TrimSpace(req.Topic)
	useSearch := req.IncludeWebSearch == nil || *req.IncludeWebSearch
	sources := []Source{}
	if useSearch && s.search != nil {
		if got, err := s.search.Search(ctx, topic); err == nil {
			sources = got
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Topik: %s\n", topic)
	if c := strings.TrimSpace(req.Category); c != "" {
		fmt.Fprintf(&b, "Kategori yang diinginkan: %s\n", c)
	}
	if len(sources) > 0 {
		b.WriteString("Sumber web terkini:\n")
		for i, s := range sources {
			fmt.Fprintf(&b, "[%d] %s - %s (%s)\n", i+1, s.Title, s.Snippet, s.URL)
		}
	}
	text, err := s.chat.Complete(ctx, systemPrompt, b.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAI, err)
	}
	d, err := parseDraft(text)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAI, err)
	}
	d.Category = normalizeCategory(d.Category, req.Category, validCategories)
	d.Sources = sources
	return d, nil
}

func parseDraft(text string) (*Draft, error) {
	t := strings.TrimSpace(text)
	if i := strings.Index(t, "{"); i >= 0 {
		if j := strings.LastIndex(t, "}"); j > i {
			t = t[i : j+1]
		}
	}
	var d Draft
	if err := json.Unmarshal([]byte(t), &d); err != nil {
		return nil, errors.New("keluaran bukan JSON")
	}
	d.Title, d.Excerpt, d.Content = strings.TrimSpace(d.Title), strings.TrimSpace(d.Excerpt), strings.TrimSpace(d.Content)
	if d.Title == "" || d.Content == "" {
		return nil, errors.New("judul atau konten kosong")
	}
	return &d, nil
}

func normalizeCategory(got, wanted string, valid []string) string {
	for _, cand := range []string{got, wanted} {
		for _, v := range valid {
			if strings.EqualFold(strings.TrimSpace(cand), v) {
				return v
			}
		}
	}
	return "Umum"
}
