package database

import (
	"os"
	"testing"

	"gorm.io/gorm"

	"app/internal/config"
	"app/internal/features/auth"
	"app/internal/features/post"
)

// Tes integrasi: hanya jalan bila TEST_DB_DATABASE diisi (PostgreSQL nyata).
// Di Jenkins (tanpa PostgreSQL) tes ini otomatis dilewati.
func testConfig(t *testing.T) config.Config {
	t.Helper()
	name := os.Getenv("TEST_DB_DATABASE")
	if name == "" {
		t.Skip("TEST_DB_DATABASE tidak diset; tes integrasi PostgreSQL dilewati")
	}
	cfg := config.Load()
	cfg.DBDatabase = name
	return cfg
}

func reset(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec("DROP TABLE IF EXISTS posts, users CASCADE").Error; err != nil {
		t.Fatal(err)
	}
}

func TestPostgresRepositories(t *testing.T) {
	cfg := testConfig(t)
	db, err := Connect(cfg)
	if err != nil {
		t.Fatal(err)
	}
	reset(t, db)
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ { // idempoten
		if err := Seed(db, cfg); err != nil {
			t.Fatal(err)
		}
	}

	ur := auth.NewRepository(db)
	pr := post.NewRepository(db)
	admin, err := ur.FindByLogin("admin")
	if err != nil || admin.Role != "admin" {
		t.Fatalf("admin seed: %v", err)
	}
	if byEmail, err := ur.FindByLogin("ADMIN@delcom.local"); err != nil || byEmail.ID != admin.ID {
		t.Fatal("login via email")
	}
	if _, err := ur.FindByID(99999); err != auth.ErrNotFound {
		t.Fatal("harus ErrNotFound")
	}
	if ok, _ := ur.Exists("admin", "x@y"); !ok {
		t.Fatal("Exists")
	}
	if ok, _ := ur.EmailTaken("admin@delcom.local", admin.ID); ok {
		t.Fatal("EmailTaken oleh dirinya sendiri")
	}
	admin.Email = "admin2@delcom.local"
	if err := ur.Update(admin); err != nil {
		t.Fatal(err)
	}

	list, err := pr.ListPublished("golang", "Teknologi")
	if err != nil || len(list) != 1 || list[0].AuthorName != "member" {
		t.Fatalf("ListPublished: %v %+v", err, list)
	}
	if all, _ := pr.ListPublished("", ""); len(all) != 5 {
		t.Fatalf("published seharusnya 5, dapat %d", len(all))
	}
	if _, err := pr.FindPublishedBySlug("tidak-ada"); err != post.ErrNotFound {
		t.Fatal("slug tidak ada")
	}
	d, err := pr.FindPublishedBySlug(list[0].Slug)
	if err != nil || d.AuthorName != "member" {
		t.Fatal("detail publik")
	}
	if ok, _ := pr.SlugExists(d.Slug); !ok {
		t.Fatal("SlugExists")
	}
	member, _ := ur.FindByLogin("member")
	total, pub, err := pr.Stats(member.ID)
	if err != nil || total != 6 || pub != 5 {
		t.Fatalf("stats %d/%d %v", total, pub, err)
	}
	users, err := pr.ListUsers()
	if err != nil || len(users) != 2 || users[1].TotalArticles != 6 || users[1].TotalPublished != 5 {
		t.Fatalf("ListUsers %+v %v", users, err)
	}
	if rows, _ := pr.ListAll(); len(rows) != 6 {
		t.Fatal("ListAll")
	}
	if rows, _ := pr.ListByUser(member.ID); len(rows) != 6 {
		t.Fatal("ListByUser")
	}
	p, err := pr.FindByID(d.ID)
	if err != nil {
		t.Fatal(err)
	}
	p.Title = "Diubah"
	if err := pr.Update(p); err != nil {
		t.Fatal(err)
	}
	if err := pr.Delete(p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pr.FindByID(p.ID); err != post.ErrNotFound {
		t.Fatal("harus terhapus")
	}
	// hapus user -> artikelnya ikut terhapus (ON DELETE CASCADE)
	if err := db.Delete(&auth.User{}, member.ID).Error; err != nil {
		t.Fatal(err)
	}
	if rows, _ := pr.ListAll(); len(rows) != 0 {
		t.Fatal("cascade delete gagal")
	}
}

func TestConnectFailure(t *testing.T) {
	cfg := config.Load()
	cfg.DBHost, cfg.DBPort = "127.0.0.1", "1"
	if _, err := Connect(cfg); err == nil {
		t.Fatal("harus gagal")
	}
}
