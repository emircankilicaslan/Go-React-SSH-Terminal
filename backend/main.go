package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// App, bağımlılıkları (ayar, veritabanı, bilet deposu, hız sınırlayıcı) bir arada tutar; testlerde kolayca kurulur.
type App struct {
	cfg     Config
	db      *gorm.DB
	tickets *ticketStore
	limiter *loginLimiter
}

func newApp(cfg Config, db *gorm.DB) (*App, error) {
	if err := db.AutoMigrate(&User{}, &Connection{}); err != nil {
		return nil, err
	}
	return &App{cfg: cfg, db: db, tickets: newTicketStore(), limiter: newLoginLimiter(10)}, nil
}

func (a *App) routes() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger(), a.cors())

	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.POST("/register", a.register)
	r.POST("/login", a.login)
	r.GET("/ws/:ticket", a.handleSSHWebSocket) // bilet tek kullanımlıktır, kimlik doğrulama bilettedir

	api := r.Group("/", a.authRequired())
	api.GET("/connections", a.listConnections)
	api.POST("/connections", a.addConnection)
	api.DELETE("/connections/:id", a.deleteConnection)
	api.POST("/connections/:id/ticket", a.issueTicket)
	return r
}

func (a *App) originAllowed(origin string) bool {
	return origin != "" && slices.Contains(a.cfg.AllowedOrigins, origin)
}

// cors yalnızca izin verilen kaynaklara yanıt verir ("*" kullanılmaz).
func (a *App) cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		if origin := c.GetHeader("Origin"); a.originAllowed(origin) {
			h := c.Writer.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Vary", "Origin")
			h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			h.Set("Access-Control-Allow-Methods", "POST, GET, DELETE, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func openDB(path string) (*gorm.DB, error) {
	if dir := filepath.Dir(path); dir != "." && path != ":memory:" && dir != "" {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, err
		}
	}
	return gorm.Open(sqlite.Open(path), &gorm.Config{})
}

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("yapılandırma hatası: %v", err)
	}
	db, err := openDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("veritabanı açılamadı: %v", err)
	}
	app, err := newApp(cfg, db)
	if err != nil {
		log.Fatalf("migrasyon hatası: %v", err)
	}
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: app.routes(), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("dinleniyor :%s", cfg.Port)
	log.Fatal(srv.ListenAndServe())
}
