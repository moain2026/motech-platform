// Command server is the Motech Platform backend entrypoint.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"

	"motech-platform/backend/internal/auth"
	"motech-platform/backend/internal/config"
	"motech-platform/backend/internal/db"
	"motech-platform/backend/internal/handlers"
	"motech-platform/backend/internal/netbird"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	conn, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	if err := db.MigrateDir(conn, "migrations"); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Println("migrations applied")

	am := auth.NewManager(cfg.JWTSecret)
	nb := netbird.New(cfg.NetbirdAPIURL, cfg.NetbirdAPIToken)
	if nb.IsMock() {
		log.Println("NetBird: MOCK mode (no NETBIRD_API_TOKEN set)")
	} else {
		log.Println("NetBird: LIVE mode ->", cfg.NetbirdAPIURL)
	}

	seedAdmin(conn, am, cfg)

	h := handlers.New(conn, cfg, am, nb)
	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer)

	r.Get("/health", h.Health)
	r.Post("/api/auth/login", h.Login)
	r.Post("/api/agent/register", h.AgentRegister)
	// public install page (one-time token in URL is the only secret)
	r.Get("/setup/{token}", h.SetupPage)

	// agent-authenticated
	r.Group(func(g chi.Router) {
		g.Use(am.Middleware("agent"))
		g.Post("/api/agent/heartbeat", h.AgentHeartbeat)
	})

	// admin-authenticated
	r.Group(func(g chi.Router) {
		g.Use(am.Middleware("admin"))
		g.Get("/api/clients", h.ListClients)
		g.Post("/api/clients", h.CreateClient)
		g.Get("/api/clients/{id}", h.GetClient)
		g.Put("/api/clients/{id}", h.UpdateClient)
		g.Get("/api/clients/{id}/connection", h.Connection)
		g.Get("/api/clients/{id}/private-key", h.PrivateKey)
		g.Post("/api/clients/{id}/rotate-key", h.RotateKey)
		g.Post("/api/clients/{id}/disable", h.DisableClient)
		g.Delete("/api/clients/{id}", h.DeleteClient)
		g.Get("/api/activity", h.Activity)
	})

	// static dashboard
	dashDir := "../dashboard"
	if _, err := os.Stat(dashDir); err == nil {
		r.Handle("/*", http.FileServer(http.Dir(dashDir)))
	}

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	log.Printf("Motech backend listening on :%s", cfg.Port)
	log.Fatal(srv.ListenAndServe())
}

// seedAdmin creates the first admin if the admins table is empty.
func seedAdmin(conn *sqlx.DB, am *auth.Manager, cfg *config.Config) {
	var count int
	if err := conn.Get(&count, `SELECT COUNT(*) FROM admins`); err != nil {
		log.Printf("seedAdmin: count failed: %v", err)
		return
	}
	if count > 0 {
		return
	}
	hash, err := auth.HashPassword(cfg.SeedAdminPass)
	if err != nil {
		log.Printf("seedAdmin: hash failed: %v", err)
		return
	}
	if _, err := conn.Exec(`INSERT INTO admins (email, password_hash) VALUES ($1,$2)`,
		cfg.SeedAdminEmail, hash); err != nil {
		log.Printf("seedAdmin: insert failed: %v", err)
		return
	}
	log.Printf("WARNING: seeded default admin %s with the configured password \u2014 change it after first login", cfg.SeedAdminEmail)
}
