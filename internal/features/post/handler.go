package post

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"app/internal/middleware"
	"app/internal/response"
)

// Handler endpoint /api/public/*, /api/posts/*, /api/admin/*.
type Handler struct{ svc *Service }

// NewHandler membuat Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) fail(c *gin.Context, err error) {
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		response.Fail(c, http.StatusBadRequest, ve.Msg)
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, "Artikel tidak ditemukan")
	case errors.Is(err, ErrForbidden):
		response.Fail(c, http.StatusForbidden, "Anda bukan pemilik artikel ini")
	default:
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}

func idParam(c *gin.Context) (uint, bool) {
	n, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID harus berupa angka")
		return 0, false
	}
	return uint(n), true
}

// Categories GET /api/public/categories
func (h *Handler) Categories(c *gin.Context) {
	response.Success(c, http.StatusOK, "Daftar kategori", h.svc.PublicCategories())
}

// PublicList GET /api/public/posts
func (h *Handler) PublicList(c *gin.Context) {
	out, err := h.svc.PublicList(c.Query("q"), c.Query("category"))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Daftar artikel", out)
}

// PublicDetail GET /api/public/posts/:slug
func (h *Handler) PublicDetail(c *gin.Context) {
	p, err := h.svc.PublicDetail(c.Param("slug"))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Detail artikel", p)
}

// List GET /api/posts
func (h *Handler) List(c *gin.Context) {
	out, err := h.svc.List(middleware.UserID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Daftar artikel Anda", out)
}

// Stats GET /api/posts/stats
func (h *Handler) Stats(c *gin.Context) {
	s, err := h.svc.Stats(middleware.UserID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Statistik artikel", s)
}

// Create POST /api/posts
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Body JSON tidak valid")
		return
	}
	p, err := h.svc.Create(middleware.UserID(c), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Artikel berhasil dibuat", p)
}

// Get GET /api/posts/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	p, err := h.svc.Get(middleware.UserID(c), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Detail artikel", p)
}

// Update PUT /api/posts/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Body JSON tidak valid")
		return
	}
	p, err := h.svc.Update(middleware.UserID(c), id, req)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Artikel berhasil diperbarui", p)
}

// Toggle PATCH /api/posts/:id/toggle-publish
func (h *Handler) Toggle(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	p, err := h.svc.TogglePublish(middleware.UserID(c), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Status publikasi diubah", p)
}

// Delete DELETE /api/posts/:id
func (h *Handler) Delete(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(middleware.UserID(c), id); err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Artikel berhasil dihapus", nil)
}

// AdminUsers GET /api/admin/users
func (h *Handler) AdminUsers(c *gin.Context) {
	out, err := h.svc.AdminUsers()
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Daftar pengguna", out)
}

// AdminPosts GET /api/admin/posts
func (h *Handler) AdminPosts(c *gin.Context) {
	out, err := h.svc.AdminPosts()
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Seluruh artikel", out)
}

// AdminUserPosts GET /api/admin/users/:id/posts
func (h *Handler) AdminUserPosts(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	out, err := h.svc.AdminUserPosts(id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Artikel pengguna", out)
}
