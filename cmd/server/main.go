package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/armin/expenses/backend/internal/database"
	"github.com/armin/expenses/backend/internal/handlers"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	dbPath := envOr("DATABASE_PATH", "./data/expenses.db")
	staticDir := envOr("STATIC_DIR", "./static")
	addr := envOr("ADDR", ":8080")

	db, err := database.Open(dbPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(cors.Default())

	h := handlers.New(db)

	api := r.Group("/api")
	{
		api.GET("/persons", h.ListPersons)
		api.GET("/shops", h.ListShops)
		api.POST("/shops", h.CreateShop)
		api.DELETE("/shops/:id", h.DeleteShop)
		api.GET("/invoices", h.ListInvoices)
		api.GET("/invoices/:id", h.GetInvoice)
		api.POST("/invoices", h.CreateInvoice)
		api.DELETE("/invoices/:id", h.DeleteInvoice)
	}

	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		r.Static("/assets", filepath.Join(staticDir, "assets"))
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.File(filepath.Join(staticDir, "index.html"))
		})
	}

	log.Printf("listening on %s (db: %s)", addr, dbPath)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
