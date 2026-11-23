package post

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"app/internal/features/auth"
)

// ErrNotFound bila artikel tidak ada.
var ErrNotFound = errors.New("artikel tidak ditemukan")

// Repository abstraksi penyimpanan artikel.
type Repository interface {
	Create(p *Post) error
	Update(p *Post) error
	Delete(id uint) error
	FindByID(id uint) (*Post, error)
	SlugExists(slug string) (bool, error)
	FindPublishedBySlug(slug string) (*PostRow, error)
	ListPublished(q, category string) ([]PostRow, error)
	ListByUser(userID uint) ([]PostRow, error)
	ListAll() ([]PostRow, error)
	Stats(userID uint) (total, published int64, err error)
	ListUsers() ([]UserSummary, error)
}

type gormRepository struct{ db *gorm.DB }

// NewRepository membuat Repository berbasis GORM.
func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) rows() *gorm.DB {
	return r.db.Table("posts").
		Select("posts.*, users.username AS author_name").
		Joins("LEFT JOIN users ON users.id = posts.user_id")
}

func (r *gormRepository) Create(p *Post) error { return r.db.Omit("Author").Create(p).Error }
func (r *gormRepository) Update(p *Post) error { return r.db.Omit("Author").Save(p).Error }
func (r *gormRepository) Delete(id uint) error { return r.db.Delete(&Post{}, id).Error }

func (r *gormRepository) FindByID(id uint) (*Post, error) {
	var p Post
	if err := r.db.Limit(1).Take(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *gormRepository) SlugExists(slug string) (bool, error) {
	var n int64
	err := r.db.Model(&Post{}).Where("slug = ?", slug).Count(&n).Error
	return n > 0, err
}

func (r *gormRepository) FindPublishedBySlug(slug string) (*PostRow, error) {
	var row PostRow
	res := r.rows().Where("posts.slug = ? AND posts.is_published = ?", slug, true).Limit(1).Scan(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return &row, nil
}

func (r *gormRepository) ListPublished(q, category string) ([]PostRow, error) {
	tx := r.rows().Where("posts.is_published = ?", true)
	if category != "" {
		tx = tx.Where("LOWER(posts.category) = ?", strings.ToLower(category))
	}
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		tx = tx.Where("LOWER(posts.title) LIKE ? OR LOWER(posts.excerpt) LIKE ? OR LOWER(posts.content) LIKE ?", like, like, like)
	}
	var out []PostRow
	err := tx.Order("posts.published_at DESC, posts.id DESC").Scan(&out).Error
	return out, err
}

func (r *gormRepository) ListByUser(userID uint) ([]PostRow, error) {
	var out []PostRow
	err := r.rows().Where("posts.user_id = ?", userID).Order("posts.id DESC").Scan(&out).Error
	return out, err
}

func (r *gormRepository) ListAll() ([]PostRow, error) {
	var out []PostRow
	err := r.rows().Order("posts.id DESC").Scan(&out).Error
	return out, err
}

func (r *gormRepository) Stats(userID uint) (int64, int64, error) {
	var total, published int64
	if err := r.db.Model(&Post{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return 0, 0, err
	}
	err := r.db.Model(&Post{}).Where("user_id = ? AND is_published = ?", userID, true).Count(&published).Error
	return total, published, err
}

func (r *gormRepository) ListUsers() ([]UserSummary, error) {
	var out []UserSummary
	err := r.db.Model(&auth.User{}).
		Select(`users.id, users.username, users.email, users.role, users.created_at,
			COUNT(posts.id) AS total_articles,
			COALESCE(SUM(CASE WHEN posts.is_published THEN 1 ELSE 0 END), 0) AS total_published`).
		Joins("LEFT JOIN posts ON posts.user_id = users.id").
		Group("users.id").Order("users.id ASC").Scan(&out).Error
	return out, err
}
