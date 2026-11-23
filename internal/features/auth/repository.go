package auth

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// ErrNotFound dikembalikan repository bila data tidak ada.
var ErrNotFound = errors.New("data tidak ditemukan")

// Repository abstraksi penyimpanan user.
type Repository interface {
	Create(u *User) error
	Update(u *User) error
	FindByID(id uint) (*User, error)
	FindByLogin(login string) (*User, error)
	Exists(username, email string) (bool, error)
	EmailTaken(email string, exceptID uint) (bool, error)
}

type gormRepository struct{ db *gorm.DB }

// NewRepository membuat Repository berbasis GORM.
func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) Create(u *User) error { return r.db.Create(u).Error }
func (r *gormRepository) Update(u *User) error { return r.db.Save(u).Error }

func one(db *gorm.DB) (*User, error) {
	var u User
	if err := db.Limit(1).Take(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *gormRepository) FindByID(id uint) (*User, error) {
	return one(r.db.Where("id = ?", id))
}

func (r *gormRepository) FindByLogin(login string) (*User, error) {
	return one(r.db.Where("username = ? OR email = ?", login, strings.ToLower(login)))
}

func (r *gormRepository) Exists(username, email string) (bool, error) {
	var n int64
	err := r.db.Model(&User{}).Where("username = ? OR email = ?", username, email).Count(&n).Error
	return n > 0, err
}

func (r *gormRepository) EmailTaken(email string, exceptID uint) (bool, error) {
	var n int64
	err := r.db.Model(&User{}).Where("email = ? AND id <> ?", email, exceptID).Count(&n).Error
	return n > 0, err
}
