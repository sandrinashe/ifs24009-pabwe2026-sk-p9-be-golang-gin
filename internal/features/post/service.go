package post

import (
	"crypto/rand"
	"errors"
	"math/big"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"app/internal/features/auth"
)

// Error domain.
var (
	ErrValidation = errors.New("validasi gagal")
	ErrForbidden  = errors.New("bukan pemilik artikel")
)

// ValidationError membawa pesan untuk klien.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string   { return e.Msg }
func (e *ValidationError) Is(t error) bool { return t == ErrValidation }

func invalid(m string) error { return &ValidationError{Msg: m} }

// Service logika bisnis artikel.
type Service struct {
	repo Repository
	now  func() time.Time
}

// NewService membuat Service.
func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify mengubah judul menjadi slug (a-z0-9 dan tanda hubung).
func Slugify(title string) string {
	s := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(title), "-"), "-")
	if s == "" {
		s = "artikel"
	}
	if len(s) > 200 {
		s = strings.Trim(s[:200], "-")
	}
	return s
}

// ReadingTime = ceil(kata / 200), minimal 1 menit.
func ReadingTime(content string) int {
	words := len(strings.Fields(content))
	n := (words + 199) / 200
	if n < 1 {
		return 1
	}
	return n
}

func randomSuffix() string {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	v := int64(100000)
	if err == nil {
		v += n.Int64()
	}
	return big.NewInt(v).String()
}

func (s *Service) uniqueSlug(title string) (string, error) {
	base := Slugify(title)
	for i := 0; i < 10; i++ {
		slug := base + "-" + randomSuffix()
		exists, err := s.repo.SlugExists(slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
	}
	return "", errors.New("gagal membuat slug unik")
}

func normCategory(c string) (string, error) {
	c = strings.TrimSpace(c)
	if c == "" {
		return "Umum", nil
	}
	for _, k := range Categories {
		if strings.EqualFold(k, c) {
			return k, nil
		}
	}
	return "", invalid("kategori tidak valid")
}

func respond(p Post, author string) PostResponse {
	return PostResponse{Post: p, AuthorName: author, ReadingTime: ReadingTime(p.Content)}
}

func toPublic(r PostRow) PublicPost {
	return PublicPost{
		ID: r.ID, Title: r.Title, Slug: r.Slug, Category: r.Category, Excerpt: r.Excerpt,
		Content: r.Content, CoverImagePath: r.CoverImagePath, PublishedAt: r.PublishedAt,
		AuthorName: r.AuthorName, ReadingTime: ReadingTime(r.Content),
	}
}

// PublicCategories daftar kategori.
func (s *Service) PublicCategories() []string { return append([]string(nil), Categories...) }

// PublicList artikel terbit dengan pencarian q dan filter kategori ("Semua"/kosong = tanpa filter).
func (s *Service) PublicList(q, category string) ([]PublicPost, error) {
	category = strings.TrimSpace(category)
	if strings.EqualFold(category, "semua") {
		category = ""
	}
	rows, err := s.repo.ListPublished(strings.TrimSpace(q), category)
	if err != nil {
		return nil, err
	}
	out := make([]PublicPost, 0, len(rows))
	for _, r := range rows {
		out = append(out, toPublic(r))
	}
	return out, nil
}

// PublicDetail artikel terbit berdasarkan slug.
func (s *Service) PublicDetail(slug string) (*PublicPost, error) {
	r, err := s.repo.FindPublishedBySlug(slug)
	if err != nil {
		return nil, err
	}
	p := toPublic(*r)
	return &p, nil
}

// Create membuat artikel baru milik userID.
func (s *Service) Create(userID uint, req CreateRequest) (*PostResponse, error) {
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" || utf8.RuneCountInString(req.Title) > 255 {
		return nil, invalid("judul wajib diisi (maksimal 255 karakter)")
	}
	if strings.TrimSpace(req.Content) == "" {
		return nil, invalid("konten wajib diisi")
	}
	cat, err := normCategory(req.Category)
	if err != nil {
		return nil, err
	}
	slug, err := s.uniqueSlug(req.Title)
	if err != nil {
		return nil, err
	}
	p := &Post{
		UserID: userID, Title: req.Title, Slug: slug, Category: cat,
		Excerpt: strings.TrimSpace(req.Excerpt), Content: req.Content,
		CoverImagePath: strings.TrimSpace(req.CoverImagePath), IsPublished: req.IsPublished,
	}
	if p.IsPublished {
		t := s.now()
		p.PublishedAt = &t
	}
	if err := s.repo.Create(p); err != nil {
		return nil, err
	}
	r := respond(*p, "")
	return &r, nil
}

// owned memuat artikel dan memastikan kepemilikan.
func (s *Service) owned(userID, id uint) (*Post, error) {
	p, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if p.UserID != userID {
		return nil, ErrForbidden
	}
	return p, nil
}

// Get detail artikel milik penulis.
func (s *Service) Get(userID, id uint) (*PostResponse, error) {
	p, err := s.owned(userID, id)
	if err != nil {
		return nil, err
	}
	r := respond(*p, "")
	return &r, nil
}

// List artikel milik penulis.
func (s *Service) List(userID uint) ([]PostResponse, error) {
	rows, err := s.repo.ListByUser(userID)
	if err != nil {
		return nil, err
	}
	return toResponses(rows), nil
}

func toResponses(rows []PostRow) []PostResponse {
	out := make([]PostResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, respond(r.Post, r.AuthorName))
	}
	return out
}

// Update memperbarui artikel; field kosong tidak menimpa.
func (s *Service) Update(userID, id uint, req UpdateRequest) (*PostResponse, error) {
	p, err := s.owned(userID, id)
	if err != nil {
		return nil, err
	}
	if t := strings.TrimSpace(req.Title); t != "" {
		if utf8.RuneCountInString(t) > 255 {
			return nil, invalid("judul maksimal 255 karakter")
		}
		p.Title = t
	}
	if strings.TrimSpace(req.Category) != "" {
		cat, err := normCategory(req.Category)
		if err != nil {
			return nil, err
		}
		p.Category = cat
	}
	if req.Excerpt != "" {
		p.Excerpt = strings.TrimSpace(req.Excerpt)
	}
	if strings.TrimSpace(req.Content) != "" {
		p.Content = req.Content
	}
	if req.CoverImagePath != "" {
		p.CoverImagePath = strings.TrimSpace(req.CoverImagePath)
	}
	if req.IsPublished != nil {
		s.setPublished(p, *req.IsPublished)
	}
	if err := s.repo.Update(p); err != nil {
		return nil, err
	}
	r := respond(*p, "")
	return &r, nil
}

func (s *Service) setPublished(p *Post, v bool) {
	if v == p.IsPublished {
		return
	}
	p.IsPublished = v
	if v {
		t := s.now()
		p.PublishedAt = &t
	} else {
		p.PublishedAt = nil
	}
}

// TogglePublish membalik Draf <-> Terbit.
func (s *Service) TogglePublish(userID, id uint) (*PostResponse, error) {
	p, err := s.owned(userID, id)
	if err != nil {
		return nil, err
	}
	s.setPublished(p, !p.IsPublished)
	if err := s.repo.Update(p); err != nil {
		return nil, err
	}
	r := respond(*p, "")
	return &r, nil
}

// Delete menghapus artikel milik penulis.
func (s *Service) Delete(userID, id uint) error {
	if _, err := s.owned(userID, id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

// Stats ringkasan 3 kartu.
func (s *Service) Stats(userID uint) (*Stats, error) {
	total, pub, err := s.repo.Stats(userID)
	if err != nil {
		return nil, err
	}
	return &Stats{TotalArticles: total, TotalSelesai: pub, BelumSelesai: total - pub}, nil
}

// AdminUsers daftar pengguna + jumlah artikel.
func (s *Service) AdminUsers() ([]UserSummary, error) {
	u, err := s.repo.ListUsers()
	if u == nil {
		u = []UserSummary{}
	}
	return u, err
}

// AdminPosts seluruh artikel.
func (s *Service) AdminPosts() ([]PostResponse, error) {
	rows, err := s.repo.ListAll()
	if err != nil {
		return nil, err
	}
	return toResponses(rows), nil
}

// AdminUserPosts artikel milik user tertentu.
func (s *Service) AdminUserPosts(userID uint) ([]PostResponse, error) { return s.List(userID) }

// SeedSamples mengisi contoh artikel bila tabel kosong (agar galeri publik tidak hampa).
func (s *Service) SeedSamples(author *auth.User) error {
	all, err := s.repo.ListAll()
	if err != nil || len(all) > 0 {
		return err
	}
	for _, smp := range samples {
		if _, err := s.Create(author.ID, smp); err != nil {
			return err
		}
	}
	return nil
}
