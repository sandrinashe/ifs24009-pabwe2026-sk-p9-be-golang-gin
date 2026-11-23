package ai

// GenerateRequest payload POST /api/posts/ai-generate.
type GenerateRequest struct {
	Topic            string `json:"topic"`
	Category         string `json:"category"`
	IncludeWebSearch *bool  `json:"include_web_search"`
}

// Source satu rujukan hasil search engine.
type Source struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// Draft draf artikel hasil AI.
type Draft struct {
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Excerpt  string   `json:"excerpt"`
	Content  string   `json:"content"`
	Sources  []Source `json:"sources"`
}
