package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"

	"app/internal/config"
	"app/internal/features/auth"
	"app/internal/features/post"
)

// ---- fake repositories (in-memory) ----

type fakeAuth struct {
	mu    sync.Mutex
	users map[uint]*auth.User
	next  uint
	fail  bool
}

func newFakeAuth() *fakeAuth { return &fakeAuth{users: map[uint]*auth.User{}, next: 1} }

func (f *fakeAuth) Create(u *auth.User) error {
	if f.fail {
		return errors.New("db down")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	u.ID = f.next
	f.next++
	c := *u
	f.users[u.ID] = &c
	return nil
}
func (f *fakeAuth) Update(u *auth.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := *u
	f.users[u.ID] = &c
	return nil
}
func (f *fakeAuth) FindByID(id uint) (*auth.User, error) {
	if f.fail {
		return nil, errors.New("db down")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.users[id]; ok {
		c := *u
		return &c, nil
	}
	return nil, auth.ErrNotFound
}
func (f *fakeAuth) FindByLogin(l string) (*auth.User, error) {
	if f.fail {
		return nil, errors.New("db down")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.users {
		if u.Username == l || u.Email == strings.ToLower(l) {
			c := *u
			return &c, nil
		}
	}
	return nil, auth.ErrNotFound
}
func (f *fakeAuth) Exists(un, em string) (bool, error) {
	if f.fail {
		return false, errors.New("db down")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.users {
		if u.Username == un || u.Email == em {
			return true, nil
		}
	}
	return false, nil
}
func (f *fakeAuth) EmailTaken(em string, except uint) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.users {
		if u.Email == em && u.ID != except {
			return true, nil
		}
	}
	return false, nil
}

type fakePosts struct {
	mu    sync.Mutex
	posts map[uint]*post.Post
	next  uint
	au    *fakeAuth
	fail  bool
}

func newFakePosts(au *fakeAuth) *fakePosts {
	return &fakePosts{posts: map[uint]*post.Post{}, next: 1, au: au}
}

func (f *fakePosts) author(id uint) string {
	if u, ok := f.au.users[id]; ok {
		return u.Username
	}
	return ""
}
func (f *fakePosts) rows(filter func(*post.Post) bool) []post.PostRow {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []post.PostRow{}
	for _, p := range f.posts {
		if filter(p) {
			out = append(out, post.PostRow{Post: *p, AuthorName: f.author(p.UserID)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out
}
func (f *fakePosts) Create(p *post.Post) error {
	if f.fail {
		return errors.New("db down")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	p.ID = f.next
	f.next++
	c := *p
	f.posts[p.ID] = &c
	return nil
}
func (f *fakePosts) Update(p *post.Post) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := *p
	f.posts[p.ID] = &c
	return nil
}
func (f *fakePosts) Delete(id uint) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.posts, id)
	return nil
}
func (f *fakePosts) FindByID(id uint) (*post.Post, error) {
	if f.fail {
		return nil, errors.New("db down")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if p, ok := f.posts[id]; ok {
		c := *p
		return &c, nil
	}
	return nil, post.ErrNotFound
}
func (f *fakePosts) SlugExists(s string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range f.posts {
		if p.Slug == s {
			return true, nil
		}
	}
	return false, nil
}
func (f *fakePosts) FindPublishedBySlug(s string) (*post.PostRow, error) {
	r := f.rows(func(p *post.Post) bool { return p.Slug == s && p.IsPublished })
	if len(r) == 0 {
		return nil, post.ErrNotFound
	}
	return &r[0], nil
}
func (f *fakePosts) ListPublished(q, cat string) ([]post.PostRow, error) {
	if f.fail {
		return nil, errors.New("db down")
	}
	return f.rows(func(p *post.Post) bool {
		if !p.IsPublished || (cat != "" && !strings.EqualFold(p.Category, cat)) {
			return false
		}
		return q == "" || strings.Contains(strings.ToLower(p.Title+p.Excerpt+p.Content), strings.ToLower(q))
	}), nil
}
func (f *fakePosts) ListByUser(id uint) ([]post.PostRow, error) {
	if f.fail {
		return nil, errors.New("db down")
	}
	return f.rows(func(p *post.Post) bool { return p.UserID == id }), nil
}
func (f *fakePosts) ListAll() ([]post.PostRow, error) {
	if f.fail {
		return nil, errors.New("db down")
	}
	return f.rows(func(*post.Post) bool { return true }), nil
}
func (f *fakePosts) Stats(id uint) (int64, int64, error) {
	if f.fail {
		return 0, 0, errors.New("db down")
	}
	var t, p int64
	for _, r := range f.rows(func(p *post.Post) bool { return p.UserID == id }) {
		t++
		if r.IsPublished {
			p++
		}
	}
	return t, p, nil
}
func (f *fakePosts) ListUsers() ([]post.UserSummary, error) {
	if f.fail {
		return nil, errors.New("db down")
	}
	out := []post.UserSummary{}
	for _, u := range f.au.users {
		s := post.UserSummary{ID: u.ID, Username: u.Username, Email: u.Email, Role: u.Role}
		for _, p := range f.posts {
			if p.UserID == u.ID {
				s.TotalArticles++
				if p.IsPublished {
					s.TotalPublished++
				}
			}
		}
		out = append(out, s)
	}
	return out, nil
}

type fakeChat struct {
	out string
	err error
}

func (f fakeChat) Complete(context.Context, string, string) (string, error) { return f.out, f.err }

// ---- harness ----

type env struct {
	t  *testing.T
	r  http.Handler
	au *fakeAuth
	po *fakePosts
}

func newEnv(t *testing.T, chat fakeChat) *env {
	au := newFakeAuth()
	po := newFakePosts(au)
	cfg := config.Config{AppKey: "k_test_secret_key_k_test_secret_key"}
	r := NewRouter(Deps{Config: cfg, Auth: au, Posts: po, Chat: chat, Search: nil})
	return &env{t: t, r: r, au: au, po: po}
}

type resp struct {
	Code int
	Body struct {
		Status  string          `json:"status"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	Raw string
}

func (e *env) do(method, path, token string, body any) resp {
	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	out := resp{Code: w.Code, Raw: w.Body.String()}
	_ = json.Unmarshal(w.Body.Bytes(), &out.Body)
	return out
}

func (e *env) expect(r resp, code int, status string) {
	e.t.Helper()
	if r.Code != code || (status != "" && r.Body.Status != status) {
		e.t.Fatalf("want %d/%s got %d/%s: %s", code, status, r.Code, r.Body.Status, r.Raw)
	}
}

func (e *env) login(name, role string) (string, uint) {
	e.t.Helper()
	r := e.do("POST", "/api/auth/register", "", map[string]any{
		"username": name, "email": name + "@uji.local", "password": "Password123!", "role": role})
	e.expect(r, 201, "success")
	var d struct {
		Token string `json:"token"`
		User  struct {
			ID   uint   `json:"id"`
			Role string `json:"role"`
		} `json:"user"`
	}
	_ = json.Unmarshal(r.Body.Data, &d)
	return d.Token, d.User.ID
}

func idOf(r resp) uint {
	var d struct {
		ID uint `json:"id"`
	}
	_ = json.Unmarshal(r.Body.Data, &d)
	return d.ID
}

// ---- tests ----

func TestRootDocsAndNotFound(t *testing.T) {
	e := newEnv(t, fakeChat{})
	e.expect(e.do("GET", "/", "", nil), 200, "success")
	e.expect(e.do("GET", "/tidak/ada", "", nil), 404, "fail")
	e.expect(e.do("GET", "/swagger", "", nil), 301, "")
	if r := e.do("GET", "/swagger/index.html", "", nil); r.Code != 200 || !strings.Contains(r.Raw, "swagger-ui") {
		t.Fatal("swagger ui")
	}
	r := e.do("GET", "/api.json", "", nil)
	if r.Code != 200 || !strings.Contains(r.Raw, "/api/public/categories") {
		t.Fatal("api.json")
	}
	req := httptest.NewRequest("GET", "/api.json", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), "https://") {
		t.Fatal("scheme https")
	}
	req = httptest.NewRequest("OPTIONS", "/api/posts", nil)
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 204 || w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("cors preflight")
	}
}

func TestAuthFlow(t *testing.T) {
	e := newEnv(t, fakeChat{})
	tok, _ := e.login("penulis1", "")
	e.expect(e.do("POST", "/api/auth/register", "", map[string]any{"username": "penulis1", "email": "penulis1@uji.local", "password": "Password123!"}), 400, "fail")
	for _, b := range []map[string]any{
		{"username": "ab", "email": "a@b.co", "password": "123456"},
		{"username": "abc", "password": "123456"},
		{"username": "abc", "email": "bukan-email", "password": "123456"},
		{"username": "abc", "email": "a@b", "password": "123456"},
		{"username": "abc", "email": "a@b.co", "password": "123"},
	} {
		e.expect(e.do("POST", "/api/auth/register", "", b), 400, "fail")
	}
	e.expect(e.do("POST", "/api/auth/register", "", nil), 400, "fail")
	e.expect(e.do("POST", "/api/auth/login", "", nil), 400, "fail")

	e.expect(e.do("POST", "/api/auth/login", "", map[string]any{"username": "penulis1", "password": "Password123!"}), 200, "success")
	e.expect(e.do("POST", "/api/auth/login", "", map[string]any{"username": "PENULIS1@uji.local", "password": "Password123!"}), 200, "success")
	e.expect(e.do("POST", "/api/auth/login", "", map[string]any{"username": "penulis1", "password": "salah"}), 401, "fail")
	e.expect(e.do("POST", "/api/auth/login", "", map[string]any{"username": "hantu", "password": "x"}), 401, "fail")
	e.expect(e.do("POST", "/api/auth/login", "", map[string]any{"username": "penulis1"}), 400, "fail")
	e.expect(e.do("POST", "/api/auth/logout", "", nil), 200, "success")

	r := e.do("GET", "/api/auth/profile", tok, nil)
	e.expect(r, 200, "success")
	if strings.Contains(strings.ToLower(r.Raw), "password") {
		t.Fatal("password bocor")
	}
	e.expect(e.do("GET", "/api/auth/profile", "", nil), 401, "fail")
	e.expect(e.do("GET", "/api/auth/profile", "token.palsu.x", nil), 401, "fail")
	req := httptest.NewRequest("GET", "/api/auth/profile", nil)
	req.Header.Set("Authorization", tok)
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatal("tanpa Bearer harus 401")
	}
	req.Header.Set("Authorization", "Bearer ")
	w = httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatal("Bearer kosong harus 401")
	}

	e.expect(e.do("PUT", "/api/auth/profile", tok, map[string]any{"email": "baru@uji.local", "password": "PasswordBaru1!"}), 200, "success")
	e.expect(e.do("POST", "/api/auth/login", "", map[string]any{"username": "penulis1", "password": "Password123!"}), 401, "fail")
	e.expect(e.do("POST", "/api/auth/login", "", map[string]any{"username": "penulis1", "password": "PasswordBaru1!"}), 200, "success")
	e.expect(e.do("PUT", "/api/auth/profile", tok, map[string]any{}), 400, "fail")
	e.expect(e.do("PUT", "/api/auth/profile", tok, map[string]any{"email": "bukan"}), 400, "fail")
	e.expect(e.do("PUT", "/api/auth/profile", tok, map[string]any{"password": "123"}), 400, "fail")
	e.expect(e.do("PUT", "/api/auth/profile", tok, nil), 400, "fail")
	tok2, _ := e.login("penulis2", "")
	e.expect(e.do("PUT", "/api/auth/profile", tok2, map[string]any{"email": "baru@uji.local"}), 400, "fail")
	e.expect(e.do("PUT", "/api/auth/profile", "", map[string]any{"email": "x@y.co"}), 401, "fail")

	// user dihapus -> token lama ditolak
	e.au.mu.Lock()
	e.au.users = map[uint]*auth.User{}
	e.au.mu.Unlock()
	e.expect(e.do("GET", "/api/auth/profile", tok, nil), 401, "fail")
}

func TestRoleDefaultsAndDBErrors(t *testing.T) {
	e := newEnv(t, fakeChat{})
	e.login("kiri", "superuser") // role tak dikenal -> member
	r := e.do("GET", "/api/admin/users", "", nil)
	e.expect(r, 401, "fail")
	e.au.fail = true
	e.expect(e.do("POST", "/api/auth/register", "", map[string]any{"username": "abc", "email": "a@b.co", "password": "123456"}), 500, "error")
	e.expect(e.do("POST", "/api/auth/login", "", map[string]any{"username": "abc", "password": "123456"}), 500, "error")
	e.au.fail = false
	tok, _ := e.login("lagi", "")
	e.au.fail = true
	e.expect(e.do("GET", "/api/auth/profile", tok, nil), 401, "fail")
}

func TestPostCRUD(t *testing.T) {
	e := newEnv(t, fakeChat{})
	tok, uid := e.login("pena", "")
	other, _ := e.login("lain", "")

	e.expect(e.do("POST", "/api/posts", "", map[string]any{"title": "x"}), 401, "fail")
	e.expect(e.do("POST", "/api/posts", tok, map[string]any{"content": "isi"}), 400, "fail")
	e.expect(e.do("POST", "/api/posts", tok, map[string]any{"title": "Judul"}), 400, "fail")
	e.expect(e.do("POST", "/api/posts", tok, map[string]any{"title": "J", "content": "c", "category": "Ngawur"}), 400, "fail")
	e.expect(e.do("POST", "/api/posts", tok, map[string]any{"title": strings.Repeat("a", 300), "content": "c"}), 400, "fail")
	e.expect(e.do("POST", "/api/posts", tok, nil), 400, "fail")

	pub := e.do("POST", "/api/posts", tok, map[string]any{"title": "Artikel Uji Satu!", "content": "# Isi", "category": "teknologi", "is_published": true, "cover_image_path": "/x.png"})
	e.expect(pub, 201, "success")
	if !strings.Contains(pub.Raw, `"category":"Teknologi"`) || !strings.Contains(pub.Raw, "artikel-uji-satu-") || strings.Contains(pub.Raw, `"published_at":null`) {
		t.Fatalf("create payload: %s", pub.Raw)
	}
	draft := e.do("POST", "/api/posts", tok, map[string]any{"title": "Draf Saya", "content": "isi"})
	e.expect(draft, 201, "success")
	if !strings.Contains(draft.Raw, `"category":"Umum"`) || !strings.Contains(draft.Raw, `"published_at":null`) {
		t.Fatalf("draft payload: %s", draft.Raw)
	}
	pid, did := idOf(pub), idOf(draft)

	e.expect(e.do("GET", "/api/posts", tok, nil), 200, "success")
	if r := e.do("GET", "/api/posts", other, nil); strings.Contains(r.Raw, "Draf Saya") {
		t.Fatal("bocor ke penulis lain")
	}
	st := e.do("GET", "/api/posts/stats", tok, nil)
	if !strings.Contains(st.Raw, `"total_articles":2`) || !strings.Contains(st.Raw, `"total_selesai":1`) || !strings.Contains(st.Raw, `"belum_selesai":1`) {
		t.Fatalf("stats: %s", st.Raw)
	}
	path := func(id uint, suffix string) string { return "/api/posts/" + itoa(id) + suffix }
	e.expect(e.do("GET", path(pid, ""), tok, nil), 200, "success")
	e.expect(e.do("GET", "/api/posts/99999999", tok, nil), 404, "fail")
	e.expect(e.do("GET", "/api/posts/bukan-angka", tok, nil), 400, "fail")
	e.expect(e.do("GET", path(pid, ""), other, nil), 403, "fail")
	e.expect(e.do("GET", path(pid, ""), "", nil), 401, "fail")

	up := e.do("PUT", path(pid, ""), tok, map[string]any{"title": "Judul Baru", "content": "Konten baru", "category": "Bisnis", "excerpt": "ex", "cover_image_path": "/y.png", "is_published": false})
	e.expect(up, 200, "success")
	if !strings.Contains(up.Raw, "Judul Baru") || !strings.Contains(up.Raw, `"is_published":false`) {
		t.Fatalf("update: %s", up.Raw)
	}
	e.expect(e.do("PUT", path(pid, ""), tok, map[string]any{"category": "Ngawur"}), 400, "fail")
	e.expect(e.do("PUT", path(pid, ""), tok, map[string]any{"title": strings.Repeat("b", 300)}), 400, "fail")
	e.expect(e.do("PUT", path(pid, ""), tok, nil), 400, "fail")
	e.expect(e.do("PUT", path(pid, ""), "", map[string]any{"title": "x"}), 401, "fail")
	e.expect(e.do("PUT", "/api/posts/99999999", tok, map[string]any{"title": "x"}), 404, "fail")
	e.expect(e.do("PUT", path(pid, ""), other, map[string]any{"title": "x"}), 403, "fail")
	e.expect(e.do("PUT", path(pid, ""), tok, map[string]any{"is_published": true}), 200, "success")
	e.expect(e.do("PUT", path(pid, ""), tok, map[string]any{"is_published": true}), 200, "success")

	tg := e.do("PATCH", path(did, "/toggle-publish"), tok, nil)
	e.expect(tg, 200, "success")
	if !strings.Contains(tg.Raw, `"is_published":true`) || strings.Contains(tg.Raw, `"published_at":null`) {
		t.Fatalf("toggle: %s", tg.Raw)
	}
	e.expect(e.do("PATCH", path(did, "/toggle-publish"), "", nil), 401, "fail")
	e.expect(e.do("PATCH", "/api/posts/99999999/toggle-publish", tok, nil), 404, "fail")
	e.expect(e.do("PATCH", path(did, "/toggle-publish"), other, nil), 403, "fail")

	before := e.do("GET", "/api/posts/stats", tok, nil)
	e.expect(e.do("DELETE", path(did, ""), other, nil), 403, "fail")
	e.expect(e.do("DELETE", path(did, ""), "", nil), 401, "fail")
	e.expect(e.do("DELETE", "/api/posts/99999999", tok, nil), 404, "fail")
	e.expect(e.do("DELETE", "/api/posts/abc", tok, nil), 400, "fail")
	e.expect(e.do("DELETE", path(did, ""), tok, nil), 200, "success")
	e.expect(e.do("GET", path(did, ""), tok, nil), 404, "fail")
	if after := e.do("GET", "/api/posts/stats", tok, nil); after.Raw == before.Raw {
		t.Fatal("stats tidak berkurang")
	}
	_ = uid

	e.po.fail = true
	e.expect(e.do("GET", "/api/posts", tok, nil), 500, "error")
	e.expect(e.do("GET", "/api/posts/stats", tok, nil), 500, "error")
	e.expect(e.do("POST", "/api/posts", tok, map[string]any{"title": "a", "content": "b"}), 500, "error")
	e.expect(e.do("GET", path(pid, ""), tok, nil), 500, "error")
	e.expect(e.do("GET", "/api/public/posts", "", nil), 500, "error")
}

func itoa(n uint) string { b, _ := json.Marshal(n); return string(b) }

func TestPublic(t *testing.T) {
	e := newEnv(t, fakeChat{})
	tok, _ := e.login("pub", "")
	words := strings.Repeat("kata ", 450)
	r := e.do("POST", "/api/posts", tok, map[string]any{"title": "Belajar Golang", "content": words, "category": "Teknologi", "is_published": true})
	e.expect(r, 201, "success")
	slug := strings.Split(strings.Split(r.Raw, `"slug":"`)[1], `"`)[0]
	e.expect(e.do("POST", "/api/posts", tok, map[string]any{"title": "Rahasia Draf", "content": "x"}), 201, "success")

	c := e.do("GET", "/api/public/categories", "", nil)
	e.expect(c, 200, "success")
	l := e.do("GET", "/api/public/posts", "", nil)
	if strings.Contains(l.Raw, "Rahasia Draf") || strings.Contains(l.Raw, "user_id") || !strings.Contains(l.Raw, `"reading_time":3`) {
		t.Fatalf("public list: %s", l.Raw)
	}
	if !strings.Contains(e.do("GET", "/api/public/posts?category=Semua", "", nil).Raw, "Belajar Golang") {
		t.Fatal("Semua")
	}
	if strings.Contains(e.do("GET", "/api/public/posts?category=Bisnis", "", nil).Raw, "Belajar Golang") {
		t.Fatal("filter kategori")
	}
	if !strings.Contains(e.do("GET", "/api/public/posts?q=golang", "", nil).Raw, "Belajar Golang") {
		t.Fatal("pencarian")
	}
	if k := e.do("GET", "/api/public/posts?q=zzz", "", nil); !strings.Contains(k.Raw, `"data":[]`) && strings.Contains(k.Raw, "Belajar") {
		t.Fatal("kosong")
	}
	e.expect(e.do("GET", "/api/public/posts/"+slug, "", nil), 200, "success")
	e.expect(e.do("GET", "/api/public/posts/tidak-ada", "", nil), 404, "fail")
	e.expect(e.do("GET", "/api/public/posts/rahasia-draf", "", nil), 404, "fail")
}

func TestAdmin(t *testing.T) {
	e := newEnv(t, fakeChat{})
	member, mid := e.login("anggota", "")
	admin, _ := e.login("bosnya", "admin")
	e.expect(e.do("POST", "/api/posts", member, map[string]any{"title": "Tulisan Anggota", "content": "isi", "is_published": true}), 201, "success")

	e.expect(e.do("GET", "/api/admin/users", "", nil), 401, "fail")
	e.expect(e.do("GET", "/api/admin/users", member, nil), 403, "fail")
	u := e.do("GET", "/api/admin/users", admin, nil)
	e.expect(u, 200, "success")
	if strings.Contains(strings.ToLower(u.Raw), "password") || !strings.Contains(u.Raw, `"total_published":1`) {
		t.Fatalf("users: %s", u.Raw)
	}
	e.expect(e.do("GET", "/api/admin/posts", admin, nil), 200, "success")
	e.expect(e.do("GET", "/api/admin/posts", member, nil), 403, "fail")
	e.expect(e.do("GET", "/api/admin/users/"+itoa(mid)+"/posts", admin, nil), 200, "success")
	e.expect(e.do("GET", "/api/admin/users/99999999/posts", admin, nil), 200, "success")
	e.expect(e.do("GET", "/api/admin/users/abc/posts", admin, nil), 400, "fail")
	e.expect(e.do("GET", "/api/admin/users/"+itoa(mid)+"/posts", member, nil), 403, "fail")
	e.po.fail = true
	e.expect(e.do("GET", "/api/admin/users", admin, nil), 500, "error")
	e.expect(e.do("GET", "/api/admin/posts", admin, nil), 500, "error")
	e.expect(e.do("GET", "/api/admin/users/1/posts", admin, nil), 500, "error")
}

func TestAIGenerate(t *testing.T) {
	good := `Berikut: ` + "```json\n" + `{"title":"Judul AI","category":"teknologi","excerpt":"Ringkas","content":"## A\n\nIsi [1]"}` + "\n```"
	e := newEnv(t, fakeChat{out: good})
	tok, _ := e.login("aiuser", "")
	e.expect(e.do("POST", "/api/posts/ai-generate", "", map[string]any{"topic": "x"}), 401, "fail")
	r := e.do("POST", "/api/posts/ai-generate", tok, map[string]any{"category": "Teknologi"})
	e.expect(r, 400, "fail")
	if !strings.Contains(strings.ToLower(r.Body.Message), "topic") {
		t.Fatal("pesan harus menyebut topic")
	}
	e.expect(e.do("POST", "/api/posts/ai-generate", tok, nil), 400, "fail")
	ok := e.do("POST", "/api/posts/ai-generate", tok, map[string]any{"topic": "AI", "category": "Bisnis"})
	e.expect(ok, 200, "success")
	if !strings.Contains(ok.Raw, `"category":"Teknologi"`) || !strings.Contains(ok.Raw, `"sources":[]`) {
		t.Fatalf("ai: %s", ok.Raw)
	}

	for _, out := range []fakeChat{{err: errors.New("boom")}, {out: "bukan json"}, {out: `{"title":"","content":""}`}} {
		bad := newEnv(t, out)
		tk, _ := bad.login("aiuser", "")
		r := bad.do("POST", "/api/posts/ai-generate", tk, map[string]any{"topic": "x"})
		bad.expect(r, 500, "error")
		if !strings.Contains(strings.ToLower(r.Body.Message), "openai") {
			t.Fatal("pesan harus menyebut openai")
		}
	}
}

func TestRunFailsWithoutDatabase(t *testing.T) {
	t.Setenv("DB_HOST", "127.0.0.1")
	t.Setenv("DB_PORT", "1")
	t.Setenv("APP_ENV", "production")
	if err := Run(); err == nil {
		t.Fatal("Run harus gagal tanpa database")
	}
}
