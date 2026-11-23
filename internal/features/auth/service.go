package auth

import (
	"errors"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"app/internal/middleware"
)

// Error domain yang dipetakan handler ke status HTTP.
var (
	ErrValidation   = errors.New("validasi gagal")
	ErrInvalidCreds = errors.New("username atau password salah")
)

// ValidationError membawa pesan validasi untuk klien.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string   { return e.Msg }
func (e *ValidationError) Is(t error) bool { return t == ErrValidation }

func invalid(msg string) error { return &ValidationError{Msg: msg} }

// Service logika bisnis autentikasi.
type Service struct {
	repo   Repository
	secret string
	ttl    time.Duration
	cost   int
}

// NewService membuat Service.
func NewService(repo Repository, secret string) *Service {
	return &Service{repo: repo, secret: secret, ttl: 24 * time.Hour, cost: bcrypt.DefaultCost}
}

func validEmail(e string) bool {
	a, err := mail.ParseAddress(e)
	if err != nil || a.Address != e {
		return false
	}
	at := strings.LastIndex(e, "@")
	return strings.Contains(e[at+1:], ".")
}

func (s *Service) hash(p string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(p), s.cost)
	return string(b), err
}

func (s *Service) token(u *User) (string, error) {
	now := time.Now()
	claims := middleware.Claims{
		Role: u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatUint(uint64(u.ID), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.secret))
}

// Register mendaftarkan akun baru. Role hanya "admin"/"member"; selain itu menjadi "member".
func (s *Service) Register(req RegisterRequest) (*AuthResult, error) {
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	switch {
	case len(req.Username) < 3:
		return nil, invalid("username wajib diisi (minimal 3 karakter)")
	case !validEmail(req.Email):
		return nil, invalid("email wajib diisi dengan format yang valid")
	case len(req.Password) < 6:
		return nil, invalid("password wajib diisi (minimal 6 karakter)")
	}
	exists, err := s.repo.Exists(req.Username, req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, invalid("username atau email sudah digunakan")
	}
	role := "member"
	if req.Role == "admin" {
		role = "admin"
	}
	hashed, err := s.hash(req.Password)
	if err != nil {
		return nil, err
	}
	u := &User{Username: req.Username, Email: req.Email, Password: hashed, Role: role}
	if err := s.repo.Create(u); err != nil {
		return nil, err
	}
	tok, err := s.token(u)
	if err != nil {
		return nil, err
	}
	return &AuthResult{Token: tok, User: u}, nil
}

// Login memverifikasi kredensial (username atau email) dan menerbitkan JWT.
func (s *Service) Login(req LoginRequest) (*AuthResult, error) {
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		return nil, invalid("username dan password wajib diisi")
	}
	u, err := s.repo.FindByLogin(req.Username)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvalidCreds
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)) != nil {
		return nil, ErrInvalidCreds
	}
	tok, err := s.token(u)
	if err != nil {
		return nil, err
	}
	return &AuthResult{Token: tok, User: u}, nil
}

// Profile mengambil profil user.
func (s *Service) Profile(id uint) (*User, error) { return s.repo.FindByID(id) }

// UpdateProfile memperbarui email dan/atau password.
func (s *Service) UpdateProfile(id uint, req UpdateProfileRequest) (*User, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" && req.Password == "" {
		return nil, invalid("email atau password wajib diisi")
	}
	u, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if req.Email != "" {
		if !validEmail(req.Email) {
			return nil, invalid("format email tidak valid")
		}
		taken, err := s.repo.EmailTaken(req.Email, id)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, invalid("email sudah digunakan")
		}
		u.Email = req.Email
	}
	if req.Password != "" {
		if len(req.Password) < 6 {
			return nil, invalid("password minimal 6 karakter")
		}
		if u.Password, err = s.hash(req.Password); err != nil {
			return nil, err
		}
	}
	if err := s.repo.Update(u); err != nil {
		return nil, err
	}
	return u, nil
}

// Loader menghasilkan UserLoader untuk middleware Auth.
func (s *Service) Loader() middleware.UserLoader {
	return func(id uint) (*middleware.Principal, error) {
		u, err := s.repo.FindByID(id)
		if err != nil {
			return nil, err
		}
		return &middleware.Principal{ID: u.ID, Role: u.Role}, nil
	}
}

// EnsureUser membuat akun bila belum ada (dipakai seeder).
func (s *Service) EnsureUser(username, email, password, role string) error {
	exists, err := s.repo.Exists(username, email)
	if err != nil || exists {
		return err
	}
	hashed, err := s.hash(password)
	if err != nil {
		return err
	}
	return s.repo.Create(&User{Username: username, Email: email, Password: hashed, Role: role})
}
