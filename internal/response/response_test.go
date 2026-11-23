package response

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func run(h gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	h(c)
	return w
}

func TestHelpers(t *testing.T) {
	cases := []struct {
		h    gin.HandlerFunc
		code int
		want string
	}{
		{func(c *gin.Context) { Success(c, 200, "ok", gin.H{"a": 1}) }, 200, `"status":"success"`},
		{func(c *gin.Context) { Fail(c, 400, "x") }, 400, `"status":"fail"`},
		{func(c *gin.Context) { Error(c, 500, "x") }, 500, `"status":"error"`},
		{func(c *gin.Context) { AbortFail(c, 401, "x") }, 401, `"status":"fail"`},
		{func(c *gin.Context) { AbortError(c, 502, "x") }, 502, `"status":"error"`},
		{NotFound, http.StatusNotFound, "Endpoint not found"},
	}
	for _, tc := range cases {
		w := run(tc.h)
		if w.Code != tc.code || !strings.Contains(w.Body.String(), tc.want) {
			t.Fatalf("got %d %s", w.Code, w.Body.String())
		}
	}
}
