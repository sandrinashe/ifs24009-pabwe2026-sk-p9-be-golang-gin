package config

import "testing"

func TestLoadDefaultsAndOverrides(t *testing.T) {
	for _, k := range []string{"APP_PORT", "PORT", "DB_HOST", "OPENAI_KEY"} {
		t.Setenv(k, "")
	}
	c := Load()
	if c.AppPort != "8081" || c.Addr() != ":8081" || c.DBDatabase != "pabwe_p9_blog" {
		t.Fatalf("default salah: %+v", c)
	}
	t.Setenv("PORT", "9000")
	if Load().AppPort != "9000" {
		t.Fatal("PORT fallback")
	}
	t.Setenv("APP_PORT", "7000")
	t.Setenv("DB_HOST", "db")
	t.Setenv("DB_PASSWORD", "pw")
	c = Load()
	if c.AppPort != "7000" || c.DSN() != "host=db user=postgres password=pw dbname=pabwe_p9_blog port=5432 sslmode=disable TimeZone=Asia/Jakarta" {
		t.Fatalf("override salah: %s", c.DSN())
	}
}
