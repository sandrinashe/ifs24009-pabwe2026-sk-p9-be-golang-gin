# Delcom Blog — Backend (Golang Gin)

REST API blog multi-tier (publik / penulis / admin) dengan AI Blog Generator + Search Engine.
Go 1.27, Gin, GORM, PostgreSQL, JWT. Respons seragam `{status, message, data?}`.

## Menjalankan
```bash
cp .env.example .env        # lalu isi nilainya
psql -U postgres -c "CREATE DATABASE pabwe_p9_blog;"
go run cmd/server/main.go   # http://localhost:8081
```
Swagger UI: http://localhost:8081/swagger/index.html — spesifikasi: `/api.json`.
Akun bawaan (seeder): `admin` / nilai `ADMIN_PASSWORD`, `member` / `member123`.

## Build & uji
```bash
go build -o server .        # biner di root, dipakai Jenkins
go test -cover ./...
# tes integrasi PostgreSQL (opsional): buat DB kosong lalu
TEST_DB_DATABASE=pabwe_p9_test go test ./internal/database/
```

## Struktur
`internal/{config,database,middleware,response,server}` dan `internal/features/{auth,post,ai}`
(handler → service → repository). `main.go` di root dan `cmd/server/main.go` sama-sama memanggil `server.Run()`.
