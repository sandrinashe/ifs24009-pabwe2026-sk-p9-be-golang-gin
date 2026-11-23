package server

import (
	"log"

	"github.com/gin-gonic/gin"

	"app/internal/config"
	"app/internal/database"
	"app/internal/features/ai"
	"app/internal/features/auth"
	"app/internal/features/post"
)

// Run memuat konfigurasi, menyiapkan database, lalu menjalankan server HTTP.
func Run() error {
	cfg := config.Load()
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	db, err := database.Connect(cfg)
	if err != nil {
		return err
	}
	if err := database.Migrate(db); err != nil {
		return err
	}
	if err := database.Seed(db, cfg); err != nil {
		return err
	}
	r := NewRouter(Deps{
		Config: cfg,
		Auth:   auth.NewRepository(db),
		Posts:  post.NewRepository(db),
		Chat:   ai.NewChatClient(cfg.OpenAIEndpoint, cfg.OpenAIKey, cfg.OpenAIModel),
		Search: ai.NewSearchClient(cfg.SearchEngineEndpoint, cfg.OpenAIKey),
	})
	log.Printf("Delcom Blog API mendengarkan di %s", cfg.Addr())
	return r.Run(cfg.Addr())
}
