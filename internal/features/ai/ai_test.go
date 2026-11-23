package ai

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchClient(t *testing.T) {
	var gotAuth, gotQ string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotQ = r.Header.Get("Authorization"), r.URL.Query().Get("q")
		switch r.URL.Query().Get("q") {
		case "bad":
			w.WriteHeader(500)
		case "notjson":
			_, _ = w.Write([]byte("<html>"))
		case "flat":
			_, _ = w.Write([]byte(`[{"name":"A","link":"https://a"},{"x":1},"str",{"title":"","url":""}]`))
		default:
			_, _ = w.Write([]byte(`{"data":{"results":[{"title":"T1","url":"https://1","snippet":"s1"},{"title":"T2","link":"https://2","content":"s2"},{"title":"T3"},{"title":"T4"},{"title":"T5"},{"title":"T6"}]}}`))
		}
	}))
	defer srv.Close()
	c := NewSearchClient(srv.URL, "KEY")
	got, err := c.Search(context.Background(), "golang")
	if err != nil || len(got) != 5 || got[0].Snippet != "s1" || got[1].URL != "https://2" || gotAuth != "Bearer KEY" || gotQ != "golang" {
		t.Fatalf("search: %v %+v", err, got)
	}
	if got, err := c.Search(context.Background(), "flat"); err != nil || len(got) != 1 || got[0].Title != "A" {
		t.Fatalf("flat: %v %+v", err, got)
	}
	if _, err := c.Search(context.Background(), "bad"); err == nil {
		t.Fatal("status 500 harus error")
	}
	if _, err := c.Search(context.Background(), "notjson"); err == nil {
		t.Fatal("non-JSON harus error")
	}
	if _, err := (&SearchClient{Endpoint: "", HTTP: http.DefaultClient}).Search(context.Background(), "x"); err == nil {
		t.Fatal("endpoint kosong")
	}
	if _, err := (&SearchClient{Endpoint: "http://[::1", HTTP: http.DefaultClient}).Search(context.Background(), "x"); err == nil {
		t.Fatal("endpoint rusak")
	}
	if _, err := (&SearchClient{Endpoint: "http://127.0.0.1:1", HTTP: http.DefaultClient}).Search(context.Background(), "x"); err == nil {
		t.Fatal("koneksi gagal")
	}
	if _, err := (&SearchClient{Endpoint: srv.URL, HTTP: http.DefaultClient}).Search(nil, "x"); err == nil { //nolint:staticcheck
		t.Fatal("ctx nil")
	}
	if got, _ := parseSources([]byte(`{"foo":1}`)); len(got) != 0 {
		t.Fatal("tanpa list")
	}
}

func TestChatClient(t *testing.T) {
	var gotAuth, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPath = r.Header.Get("Authorization"), r.URL.Path
		switch r.Header.Get("X-Mode") {
		case "500":
			w.WriteHeader(500)
		case "weird":
			_, _ = w.Write([]byte(`{"choices":[]}`))
		default:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"halo"}}]}`))
		}
	}))
	defer srv.Close()
	c := NewChatClient(srv.URL+"/", "K", "m")
	out, err := c.Complete(context.Background(), "s", "u")
	if err != nil || out != "halo" || gotAuth != "Bearer K" || gotPath != "/chat/completions" {
		t.Fatalf("chat: %v %q %q %q", err, out, gotAuth, gotPath)
	}
	for _, mode := range []string{"500", "weird"} {
		c.HTTP = &http.Client{Transport: rt(func(r *http.Request) { r.Header.Set("X-Mode", mode) })}
		if _, err := c.Complete(context.Background(), "s", "u"); err == nil {
			t.Fatalf("mode %s harus error", mode)
		}
	}
	if _, err := (&ChatClient{}).Complete(context.Background(), "s", "u"); err == nil {
		t.Fatal("endpoint kosong")
	}
	if _, err := (&ChatClient{Endpoint: "http://127.0.0.1:1", HTTP: http.DefaultClient}).Complete(context.Background(), "s", "u"); err == nil {
		t.Fatal("koneksi gagal")
	}
	if _, err := (&ChatClient{Endpoint: "http://a b", HTTP: http.DefaultClient}).Complete(context.Background(), "s", "u"); err == nil {
		t.Fatal("URL rusak")
	}
}

type rt func(*http.Request)

func (f rt) RoundTrip(r *http.Request) (*http.Response, error) {
	f(r)
	return http.DefaultTransport.RoundTrip(r)
}

type stubChat struct {
	out string
	err error
	got string
}

func (s *stubChat) Complete(_ context.Context, _, user string) (string, error) {
	s.got = user
	return s.out, s.err
}

type stubSearch struct {
	src []Source
	err error
	n   int
}

func (s *stubSearch) Search(context.Context, string) ([]Source, error) { s.n++; return s.src, s.err }

func TestServiceGenerate(t *testing.T) {
	valid := []string{"Teknologi", "Umum"}
	js := `{"title":"T","category":"teknologi","excerpt":"E","content":"C"}`
	ch := &stubChat{out: js}
	se := &stubSearch{src: []Source{{Title: "S", URL: "u", Snippet: "sn"}}}
	d, err := NewService(ch, se).Generate(context.Background(), GenerateRequest{Topic: " X ", Category: "Bisnis"}, valid)
	if err != nil || d.Category != "Teknologi" || len(d.Sources) != 1 || !strings.Contains(ch.got, "[1] S") || !strings.Contains(ch.got, "Bisnis") {
		t.Fatalf("generate: %v %+v %q", err, d, ch.got)
	}
	off := false
	se.n = 0
	if _, err := NewService(ch, se).Generate(context.Background(), GenerateRequest{Topic: "x", IncludeWebSearch: &off}, valid); err != nil || se.n != 0 {
		t.Fatal("web search harus dilewati")
	}
	se.err = errors.New("down")
	if d, err := NewService(ch, se).Generate(context.Background(), GenerateRequest{Topic: "x"}, valid); err != nil || len(d.Sources) != 0 {
		t.Fatal("search gagal tidak boleh menggagalkan")
	}
	if _, err := NewService(&stubChat{err: errors.New("x")}, nil).Generate(context.Background(), GenerateRequest{Topic: "x"}, valid); !errors.Is(err, ErrAI) {
		t.Fatal("chat error")
	}
	for _, bad := range []string{"tidak json", `{"title":"","content":"x"}`} {
		if _, err := NewService(&stubChat{out: bad}, nil).Generate(context.Background(), GenerateRequest{Topic: "x"}, valid); !errors.Is(err, ErrAI) {
			t.Fatalf("bad %q", bad)
		}
	}
	if normalizeCategory("???", "", valid) != "Umum" {
		t.Fatal("fallback kategori")
	}
}
