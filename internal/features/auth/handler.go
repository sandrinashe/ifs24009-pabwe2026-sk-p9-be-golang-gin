package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"app/internal/middleware"
	"app/internal/response"
)

// Handler endpoint /api/auth/*.
type Handler struct{ svc *Service }

// NewHandler membuat Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) fail(c *gin.Context, err error) {
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		response.Fail(c, http.StatusBadRequest, ve.Msg)
	case errors.Is(err, ErrInvalidCreds):
		response.Fail(c, http.StatusUnauthorized, err.Error())
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, "Pengguna tidak ditemukan")
	default:
		response.Error(c, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}

// Register POST /api/auth/register
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Body JSON tidak valid")
		return
	}
	res, err := h.svc.Register(req)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Registrasi berhasil", res)
}

// Login POST /api/auth/login
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Body JSON tidak valid")
		return
	}
	res, err := h.svc.Login(req)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Login berhasil", res)
}

// Logout POST /api/auth/logout (JWT stateless: klien membuang token).
func (h *Handler) Logout(c *gin.Context) {
	response.Success(c, http.StatusOK, "Logout berhasil", nil)
}

// Profile GET /api/auth/profile
func (h *Handler) Profile(c *gin.Context) {
	u, err := h.svc.Profile(middleware.UserID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Profil berhasil diambil", u)
}

// UpdateProfile PUT /api/auth/profile
func (h *Handler) UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Body JSON tidak valid")
		return
	}
	u, err := h.svc.UpdateProfile(middleware.UserID(c), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Profil berhasil diperbarui", u)
}
