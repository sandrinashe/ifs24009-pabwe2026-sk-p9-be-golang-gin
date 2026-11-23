package ai

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"app/internal/response"
)

// Handler POST /api/posts/ai-generate.
type Handler struct {
	svc        *Service
	categories []string
}

// NewHandler membuat Handler.
func NewHandler(svc *Service, categories []string) *Handler {
	return &Handler{svc: svc, categories: categories}
}

// Generate menghasilkan draf artikel.
func (h *Handler) Generate(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Topic) == "" {
		response.Fail(c, http.StatusBadRequest, "Field topic wajib diisi")
		return
	}
	d, err := h.svc.Generate(c.Request.Context(), req, h.categories)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Draf artikel berhasil dibuat", d)
}
