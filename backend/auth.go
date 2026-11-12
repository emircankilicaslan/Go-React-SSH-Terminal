package main

import (
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
)

const (
	tokenTTL      = 12 * time.Hour
	maxPasswordLn = 72 // bcrypt sınırı
)

// bcryptCost testlerde düşürülebilsin diye değişkendir.
var bcryptCost = 12

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *App) register(c *gin.Context) {
	var in credentials
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "Geçersiz istek gövdesi")
		return
	}
	email, err := normalizeEmail(in.Email)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if l := len(in.Password); l < 8 || l > maxPasswordLn {
		fail(c, http.StatusBadRequest, "Şifre 8 ile 72 karakter arasında olmalı")
		return
	}

	var n int64
	a.db.Model(&User{}).Where("email = ?", email).Count(&n)
	if n > 0 {
		fail(c, http.StatusConflict, "Bu e-posta zaten kayıtlı")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcryptCost)
	if err != nil {
		fail(c, http.StatusInternalServerError, "Hesap oluşturulamadı")
		return
	}
	if err := a.db.Create(&User{Email: email, PasswordHash: string(hash)}).Error; err != nil {
		fail(c, http.StatusConflict, "Hesap oluşturulamadı (e-posta kullanımda olabilir)")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Kayıt başarılı"})
}

func (a *App) login(c *gin.Context) {
	if !a.limiter.allow(c.ClientIP()) {
		fail(c, http.StatusTooManyRequests, "Çok fazla deneme, bir dakika sonra tekrar deneyin")
		return
	}
	var in credentials
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "Geçersiz istek gövdesi")
		return
	}
	var user User
	err := a.db.Where("email = ?", strings.ToLower(strings.TrimSpace(in.Email))).First(&user).Error
	// Kullanıcı yoksa da hash karşılaştırması yapılır: yanıt süresiyle hesap varlığı sızmasın.
	hash := user.PasswordHash
	if err != nil {
		hash = dummyHash
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil || err != nil {
		fail(c, http.StatusUnauthorized, "E-posta veya şifre hatalı")
		return
	}
	token, err := a.issueToken(user.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "Token üretilemedi")
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user_id": user.ID, "email": user.Email})
}

var dummyHash = func() string {
	h, _ := bcrypt.GenerateFromPassword([]byte("dummy-password"), bcryptCost)
	return string(h)
}()

func (a *App) issueToken(userID uint) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatUint(uint64(userID), 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.cfg.JWTSecret)
}

// authRequired, Bearer token'ı doğrular ve kullanıcı kimliğini bağlama koyar.
func (a *App) authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		raw, ok := strings.CutPrefix(h, "Bearer ")
		if !ok || raw == "" {
			fail(c, http.StatusUnauthorized, "Kimlik doğrulama gerekli")
			c.Abort()
			return
		}
		claims := &jwt.RegisteredClaims{}
		tok, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) { return a.cfg.JWTSecret, nil },
			jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
		if err != nil || !tok.Valid {
			fail(c, http.StatusUnauthorized, "Geçersiz veya süresi dolmuş token")
			c.Abort()
			return
		}
		id, err := strconv.ParseUint(claims.Subject, 10, 64)
		if err != nil {
			fail(c, http.StatusUnauthorized, "Geçersiz token")
			c.Abort()
			return
		}
		c.Set("uid", uint(id))
		c.Next()
	}
}

func userID(c *gin.Context) uint { return c.GetUint("uid") }

func normalizeEmail(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" || len(s) > 254 {
		return "", errString("Geçerli bir e-posta girin")
	}
	if addr, err := mail.ParseAddress(s); err != nil || addr.Address != s {
		return "", errString("Geçerli bir e-posta girin")
	}
	return s, nil
}

type errString string

func (e errString) Error() string { return string(e) }

func fail(c *gin.Context, status int, msg string) { c.JSON(status, gin.H{"error": msg}) }

// loginLimiter, IP başına dakikada sınırlı giriş denemesine izin verir (kaba kuvvet koruması).
type loginLimiter struct {
	mu    sync.Mutex
	byIP  map[string]*rate.Limiter
	limit rate.Limit
	burst int
}

func newLoginLimiter(perMinute int) *loginLimiter {
	return &loginLimiter{byIP: map[string]*rate.Limiter{}, limit: rate.Every(time.Minute / time.Duration(perMinute)), burst: perMinute}
}

func (l *loginLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	lim, ok := l.byIP[ip]
	if !ok {
		lim = rate.NewLimiter(l.limit, l.burst)
		l.byIP[ip] = lim
	}
	return lim.Allow()
}
