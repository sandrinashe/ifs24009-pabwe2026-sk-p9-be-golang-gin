package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func sign(t *testing.T, secret, sub string, method jwt.SigningMethod, exp time.Duration) string {
	claims := Claims{Role: "member", RegisteredClaims: jwt.RegisteredClaims{Subject: sub, ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp))}}
	var key any = []byte(secret)
	if method == jwt.SigningMethodNone {
		key = jwt.UnsafeAllowNoneSignatureType
	}
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func call(h gin.HandlerFunc, auth string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", h, func(c *gin.Context) { c.JSON(200, gin.H{"id": UserID(c)}) })
	req := httptest.NewRequest("GET", "/", nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuth(t *testing.T) {
	load := func(id uint) (*Principal, error) {
		if id == 0 {
			return nil, errors.New("tidak ada")
		}
		return &Principal{ID: id, Role: "member"}, nil
	}
	h := Auth("secret", load)
	good := sign(t, "secret", "7", jwt.SigningMethodHS256, time.Hour)
	cases := map[string]int{
		"Bearer " + good: 200,
		"":               401,
		"Bearer ":        401,
		good:             401,
		"Bearer x.y.z":   401,
		"Bearer " + sign(t, "lain", "7", jwt.SigningMethodHS256, time.Hour):     401,
		"Bearer " + sign(t, "secret", "7", jwt.SigningMethodHS256, -time.Hour):  401,
		"Bearer " + sign(t, "secret", "7", jwt.SigningMethodHS512, time.Hour):   401,
		"Bearer " + sign(t, "secret", "abc", jwt.SigningMethodHS256, time.Hour): 401,
		"Bearer " + sign(t, "secret", "", jwt.SigningMethodHS256, time.Hour):    401,
		"Bearer " + sign(t, "secret", "0", jwt.SigningMethodHS256, time.Hour):   401,
	}
	for hdr, want := range cases {
		if w := call(h, hdr); w.Code != want {
			t.Fatalf("%q: got %d want %d", hdr, w.Code, want)
		}
	}
	if id := UserID(&gin.Context{}); id != 0 {
		t.Fatal("UserID tanpa context harus 0")
	}
}

func TestRequireRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mk := func(role string) int {
		r := gin.New()
		r.GET("/", func(c *gin.Context) { c.Set(CtxRole, role) }, RequireRole("admin"), func(c *gin.Context) { c.Status(200) })
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		return w.Code
	}
	if mk("admin") != 200 || mk("member") != 403 {
		t.Fatal("RequireRole salah")
	}
}

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.GET("/", func(c *gin.Context) { c.Status(200) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Header().Get("Access-Control-Allow-Origin") != "*" || w.Code != 200 {
		t.Fatal("CORS GET")
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodOptions, "/", nil))
	if w.Code != 204 {
		t.Fatal("CORS OPTIONS")
	}
}

func TestRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now()
	r := gin.New()
	r.Use(rateLimiter(2, time.Minute, func() time.Time { return now }))
	r.GET("/", func(c *gin.Context) { c.Status(200) })
	hit := func() int {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		return w.Code
	}
	if hit() != 200 || hit() != 200 || hit() != 429 {
		t.Fatal("limit")
	}
	now = now.Add(2 * time.Minute)
	if hit() != 200 {
		t.Fatal("jendela harus reset")
	}
	if RateLimiter(1, time.Second) == nil {
		t.Fatal("konstruktor")
	}
}
