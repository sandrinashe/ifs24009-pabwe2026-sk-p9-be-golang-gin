package post

import (
	"time"

	"app/internal/features/auth"
)

// Kategori yang valid.
var Categories = []string{"Teknologi", "Pendidikan", "Gaya Hidup", "Bisnis", "Opini", "Umum"}

// Post adalah model tabel posts.
type Post struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         uint       `gorm:"not null;index" json:"user_id"`
	Author         *auth.User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Title          string     `gorm:"size:255;not null" json:"title"`
	Slug           string     `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Category       string     `gorm:"size:100;not null;default:Umum" json:"category"`
	Excerpt        string     `gorm:"type:text" json:"excerpt"`
	Content        string     `gorm:"type:text;not null" json:"content"`
	CoverImagePath string     `gorm:"type:text" json:"cover_image_path"`
	IsPublished    bool       `gorm:"not null;default:false" json:"is_published"`
	PublishedAt    *time.Time `json:"published_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// PostRow adalah Post + nama penulis hasil join.
type PostRow struct {
	Post
	AuthorName string `json:"-"`
}

// PostResponse dipakai untuk penulis/admin (memuat user_id).
type PostResponse struct {
	Post
	AuthorName  string `json:"author_name"`
	ReadingTime int    `json:"reading_time"`
}

// PublicPost dipakai untuk pembaca publik (tanpa user_id).
type PublicPost struct {
	ID             uint       `json:"id"`
	Title          string     `json:"title"`
	Slug           string     `json:"slug"`
	Category       string     `json:"category"`
	Excerpt        string     `json:"excerpt"`
	Content        string     `json:"content"`
	CoverImagePath string     `json:"cover_image_path"`
	PublishedAt    *time.Time `json:"published_at"`
	AuthorName     string     `json:"author_name"`
	ReadingTime    int        `json:"reading_time"`
}

// CreateRequest payload artikel baru.
type CreateRequest struct {
	Title          string `json:"title"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content"`
	CoverImagePath string `json:"cover_image_path"`
	IsPublished    bool   `json:"is_published"`
}

// UpdateRequest: field kosong/nil diabaikan.
type UpdateRequest struct {
	Title          string `json:"title"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content"`
	CoverImagePath string `json:"cover_image_path"`
	IsPublished    *bool  `json:"is_published"`
}

// Stats ringkasan 3 kartu statistik.
type Stats struct {
	TotalArticles int64 `json:"total_articles"`
	TotalSelesai  int64 `json:"total_selesai"`
	BelumSelesai  int64 `json:"belum_selesai"`
}

// UserSummary baris tabel pengguna pada panel admin.
type UserSummary struct {
	ID             uint      `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
	TotalArticles  int64     `json:"total_articles"`
	TotalPublished int64     `json:"total_published"`
}
