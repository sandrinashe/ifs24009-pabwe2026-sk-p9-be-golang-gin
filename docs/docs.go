// Package docs menyajikan spesifikasi OpenAPI 3 dan halaman Swagger UI.
package docs

import "encoding/json"

type op struct {
	method, path, tag, summary string
	auth                       string // "", "bearer"
	body                       map[string]string
}

var ops = []op{
	{"get", "/api/public/categories", "Publik", "Daftar kategori", "", nil},
	{"get", "/api/public/posts", "Publik", "Daftar artikel terbit (?q=&category=)", "", nil},
	{"get", "/api/public/posts/{slug}", "Publik", "Detail artikel terbit", "", nil},
	{"post", "/api/auth/register", "Auth", "Registrasi penulis", "", map[string]string{"username": "string", "email": "string", "password": "string"}},
	{"post", "/api/auth/login", "Auth", "Login (username/email + password)", "", map[string]string{"username": "string", "password": "string"}},
	{"post", "/api/auth/logout", "Auth", "Logout", "", nil},
	{"get", "/api/auth/profile", "Auth", "Profil saya", "bearer", nil},
	{"put", "/api/auth/profile", "Auth", "Ubah email/password", "bearer", map[string]string{"email": "string", "password": "string"}},
	{"get", "/api/posts", "Penulis", "Artikel milik saya", "bearer", nil},
	{"post", "/api/posts", "Penulis", "Buat artikel", "bearer", map[string]string{"title": "string", "category": "string", "excerpt": "string", "content": "string", "is_published": "boolean"}},
	{"get", "/api/posts/stats", "Penulis", "Statistik 3 kartu", "bearer", nil},
	{"get", "/api/posts/{id}", "Penulis", "Detail artikel saya", "bearer", nil},
	{"put", "/api/posts/{id}", "Penulis", "Perbarui artikel", "bearer", map[string]string{"title": "string", "content": "string"}},
	{"patch", "/api/posts/{id}/toggle-publish", "Penulis", "Draf <-> Terbit", "bearer", nil},
	{"delete", "/api/posts/{id}", "Penulis", "Hapus artikel", "bearer", nil},
	{"post", "/api/posts/ai-generate", "Penulis", "Generate draf artikel (AI + search engine)", "bearer", map[string]string{"topic": "string", "category": "string", "include_web_search": "boolean"}},
	{"get", "/api/admin/users", "Admin", "Seluruh pengguna", "bearer", nil},
	{"get", "/api/admin/posts", "Admin", "Seluruh artikel", "bearer", nil},
	{"get", "/api/admin/users/{id}/posts", "Admin", "Artikel milik user", "bearer", nil},
}

// GetSwaggerJSON membangun dokumen OpenAPI untuk host/scheme pemanggil.
func GetSwaggerJSON(host, scheme string) ([]byte, error) {
	paths := map[string]map[string]any{}
	for _, o := range ops {
		item := map[string]any{
			"tags": []string{o.tag}, "summary": o.summary,
			"responses": map[string]any{"200": map[string]string{"description": "OK"}},
		}
		var params []map[string]any
		for _, seg := range []string{"slug", "id"} {
			if contains(o.path, "{"+seg+"}") {
				params = append(params, map[string]any{"name": seg, "in": "path", "required": true, "schema": map[string]string{"type": "string"}})
			}
		}
		if len(params) > 0 {
			item["parameters"] = params
		}
		if o.auth != "" {
			item["security"] = []map[string][]string{{"bearerAuth": {}}}
		}
		if o.body != nil {
			props := map[string]any{}
			for k, t := range o.body {
				props[k] = map[string]string{"type": t}
			}
			item["requestBody"] = map[string]any{"content": map[string]any{"application/json": map[string]any{
				"schema": map[string]any{"type": "object", "properties": props}}}}
		}
		if paths[o.path] == nil {
			paths[o.path] = map[string]any{}
		}
		paths[o.path][o.method] = item
	}
	doc := map[string]any{
		"openapi": "3.0.3",
		"info":    map[string]string{"title": "Delcom Blog API", "version": "1.0.0", "description": "Respons standar {status, message, data?}"},
		"servers": []map[string]string{{"url": scheme + "://" + host}},
		"paths":   paths,
		"components": map[string]any{"securitySchemes": map[string]any{
			"bearerAuth": map[string]string{"type": "http", "scheme": "bearer", "bearerFormat": "JWT"}}},
	}
	return json.MarshalIndent(doc, "", "  ")
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// SwaggerUIHTML halaman Swagger UI yang membaca /api.json.
const SwaggerUIHTML = `<!doctype html>
<html lang="id"><head><meta charset="utf-8"><title>Delcom Blog API - Swagger UI</title>
<meta name="viewport" content="width=device-width,initial-scale=1">
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui.css"></head>
<body><div id="swagger-ui"></div>
<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.17.14/swagger-ui-bundle.js"></script>
<script>window.ui=SwaggerUIBundle({url:"/api.json",dom_id:"#swagger-ui"});</script></body></html>`
