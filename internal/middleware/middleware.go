// Package middleware: CORS, autentikasi JWT, pembatasan role, dan rate limiter.
package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"app/internal/response"
)

// Kunci context.
const (
	CtxUserID = "user_id"
	CtxRole   = "role"
)

// Principal adalah identitas minimal yang dimuat dari database tiap request.
type Principal struct {
	ID   uint
	Role string
}

// UserLoader memuat Principal terkini; mengembalikan error bila user tidak ada.
type UserLoader func(id uint) (*Principal, error)

// Claims adalah isi token JWT.
type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// CORS mengizinkan frontend lintas-origin.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Authorization,Content-Type,Accept")
		h.Set("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// Auth memverifikasi "Authorization: Bearer <jwt>" lalu menaruh user pada context.
func Auth(secret string, load UserLoader) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) || len(header) == len(prefix) {
			response.AbortFail(c, http.StatusUnauthorized, "Token autentikasi tidak ditemukan")
			return
		}
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(header[len(prefix):], claims, func(t *jwt.Token) (any, error) {
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{"HS256"}))
		if err != nil || !token.Valid {
			response.AbortFail(c, http.StatusUnauthorized, "Token tidak valid atau kedaluwarsa")
			return
		}
		var id uint
		if n, ok := parseUint(claims.Subject); ok {
			id = n
		}
		p, err := load(id)
		if err != nil || p == nil {
			response.AbortFail(c, http.StatusUnauthorized, "Pengguna tidak ditemukan")
			return
		}
		c.Set(CtxUserID, p.ID)
		c.Set(CtxRole, p.Role)
		c.Next()
	}
}

func parseUint(s string) (uint, bool) {
	var n uint
	if s == "" {
		return 0, false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + uint(r-'0')
	}
	return n, true
}

// RequireRole hanya meloloskan role yang diizinkan.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString(CtxRole)
		for _, r := range roles {
			if r == role {
				c.Next()
				return
			}
		}
		response.AbortFail(c, http.StatusForbidden, "Anda tidak memiliki akses ke sumber daya ini")
	}
}

// UserID mengambil id user dari context.
func UserID(c *gin.Context) uint {
	v, _ := c.Get(CtxUserID)
	id, _ := v.(uint)
	return id
}

// RateLimiter membatasi jumlah request per IP dalam satu jendela waktu.
func RateLimiter(limit int, window time.Duration) gin.HandlerFunc {
	return rateLimiter(limit, window, time.Now)
}

type bucket struct {
	count int
	reset time.Time
}

func rateLimiter(limit int, window time.Duration, now func() time.Time) gin.HandlerFunc {
	var mu sync.Mutex
	buckets := map[string]*bucket{}
	return func(c *gin.Context) {
		ip := c.ClientIP()
		t := now()
		mu.Lock()
		b, ok := buckets[ip]
		if !ok || !t.Before(b.reset) {
			if len(buckets) > 10000 {
				buckets = map[string]*bucket{}
			}
			b = &bucket{reset: t.Add(window)}
			buckets[ip] = b
		}
		b.count++
		over := b.count > limit
		mu.Unlock()
		if over {
			response.AbortFail(c, http.StatusTooManyRequests, "Terlalu banyak permintaan, coba lagi nanti")
			return
		}
		c.Next()
	}
}
