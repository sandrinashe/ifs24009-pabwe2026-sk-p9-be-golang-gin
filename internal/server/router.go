// Package server merakit router Gin.
package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"app/docs"
	"app/internal/config"
	"app/internal/features/ai"
	"app/internal/features/auth"
	"app/internal/features/post"
	"app/internal/middleware"
	"app/internal/response"
)

// Deps seluruh dependensi router (memudahkan pengujian dengan fake).
type Deps struct {
	Config config.Config
	Auth   auth.Repository
	Posts  post.Repository
	Chat   ai.Chat
	Search ai.Searcher
}

// NewRouter mendaftarkan seluruh rute.
func NewRouter(d Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.CORS(), middleware.RateLimiter(1200, time.Minute))
	r.NoRoute(response.NotFound)

	authSvc := auth.NewService(d.Auth, d.Config.AppKey)
	postSvc := post.NewService(d.Posts)
	ah, ph := auth.NewHandler(authSvc), post.NewHandler(postSvc)
	aih := ai.NewHandler(ai.NewService(d.Chat, d.Search), post.Categories)
	guard := middleware.Auth(d.Config.AppKey, authSvc.Loader())

	r.GET("/", func(c *gin.Context) {
		response.Success(c, http.StatusOK, "Delcom Blog API berjalan", gin.H{"docs": "/swagger/index.html"})
	})
	r.GET("/swagger", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, "/swagger/index.html") })
	r.GET("/swagger/*any", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(docs.SwaggerUIHTML))
	})
	r.GET("/api.json", func(c *gin.Context) {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		b, err := docs.GetSwaggerJSON(c.Request.Host, scheme)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "Gagal membuat spesifikasi")
			return
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", b)
	})

	api := r.Group("/api")
	pub := api.Group("/public")
	pub.GET("/categories", ph.Categories)
	pub.GET("/posts", ph.PublicList)
	pub.GET("/posts/:slug", ph.PublicDetail)

	a := api.Group("/auth")
	a.POST("/register", ah.Register)
	a.POST("/login", ah.Login)
	a.POST("/logout", ah.Logout)
	a.GET("/profile", guard, ah.Profile)
	a.PUT("/profile", guard, ah.UpdateProfile)

	p := api.Group("/posts", guard)
	p.GET("", ph.List)
	p.POST("", ph.Create)
	p.GET("/stats", ph.Stats)
	p.POST("/ai-generate", aih.Generate)
	p.GET("/:id", ph.Get)
	p.PUT("/:id", ph.Update)
	p.PATCH("/:id/toggle-publish", ph.Toggle)
	p.DELETE("/:id", ph.Delete)

	adm := api.Group("/admin", guard, middleware.RequireRole("admin"))
	adm.GET("/users", ph.AdminUsers)
	adm.GET("/posts", ph.AdminPosts)
	adm.GET("/users/:id/posts", ph.AdminUserPosts)
	return r
}
