package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex;size:254" json:"email"`
	PasswordHash string    `gorm:"column:password" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// Connection: SSH şifresi AES-GCM ile şifreli saklanır ve API yanıtlarında asla dönmez.
type Connection struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	UserID      uint   `gorm:"index" json:"-"`
	Name        string `json:"name"`
	Host        string `json:"host"`
	Port        string `json:"port"`
	Username    string `json:"username"`
	PasswordEnc string `gorm:"column:password_enc" json:"-"`
	HostKeyFP   string `gorm:"column:host_key_fp" json:"host_key_fingerprint,omitempty"`
}

type connectionInput struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (in *connectionInput) validate() string {
	in.Name, in.Host, in.Port, in.Username = strings.TrimSpace(in.Name), strings.TrimSpace(in.Host),
		strings.TrimSpace(in.Port), strings.TrimSpace(in.Username)
	switch {
	case in.Name == "" || len(in.Name) > 100:
		return "Sunucu adı 1-100 karakter olmalı"
	case in.Host == "" || len(in.Host) > 253 || strings.ContainsAny(in.Host, " /\\@"):
		return "Geçerli bir host girin"
	case in.Username == "" || len(in.Username) > 64:
		return "Kullanıcı adı 1-64 karakter olmalı"
	case in.Password == "" || len(in.Password) > 256:
		return "SSH şifresi gerekli"
	}
	if p, err := strconv.Atoi(in.Port); err != nil || p < 1 || p > 65535 {
		return "Port 1-65535 arasında olmalı"
	}
	return ""
}

func (a *App) listConnections(c *gin.Context) {
	var list []Connection
	if err := a.db.Where("user_id = ?", userID(c)).Order("id").Find(&list).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Liste alınamadı")
		return
	}
	if list == nil {
		list = []Connection{}
	}
	c.JSON(http.StatusOK, list)
}

func (a *App) addConnection(c *gin.Context) {
	var in connectionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "Geçersiz istek gövdesi")
		return
	}
	if msg := in.validate(); msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	enc, err := encrypt(a.cfg.EncryptionKey, in.Password)
	if err != nil {
		fail(c, http.StatusInternalServerError, "Kayıt yapılamadı")
		return
	}
	conn := Connection{UserID: userID(c), Name: in.Name, Host: in.Host, Port: in.Port, Username: in.Username, PasswordEnc: enc}
	if err := a.db.Create(&conn).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Kayıt yapılamadı")
		return
	}
	c.JSON(http.StatusCreated, conn)
}

func (a *App) deleteConnection(c *gin.Context) {
	res := a.db.Where("id = ? AND user_id = ?", c.Param("id"), userID(c)).Delete(&Connection{})
	if res.Error != nil || res.RowsAffected == 0 {
		fail(c, http.StatusNotFound, "Sunucu bulunamadı")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Silindi"})
}

func (a *App) ownedConnection(id string, uid uint) (*Connection, error) {
	var conn Connection
	if err := a.db.Where("id = ? AND user_id = ?", id, uid).First(&conn).Error; err != nil {
		return nil, err
	}
	return &conn, nil
}

// --- WebSocket bileti: tarayıcı WebSocket'ine başlık eklenemediği için, token URL'de taşınmaz.
// Kimliği doğrulanmış kullanıcı kısa ömürlü, tek kullanımlık bir bilet alır.

const ticketTTL = 30 * time.Second

type ticket struct {
	connID  string
	userID  uint
	expires time.Time
}

type ticketStore struct {
	mu sync.Mutex
	m  map[string]ticket
}

func newTicketStore() *ticketStore { return &ticketStore{m: map[string]ticket{}} }

func (s *ticketStore) issue(connID string, uid uint) (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	id := hex.EncodeToString(b)
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, t := range s.m { // süresi dolanları temizle
		if now.After(t.expires) {
			delete(s.m, k)
		}
	}
	s.m[id] = ticket{connID: connID, userID: uid, expires: now.Add(ticketTTL)}
	return id, nil
}

// consume bileti tek seferlik kullandırır.
func (s *ticketStore) consume(id string) (ticket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.m[id]
	delete(s.m, id)
	if !ok || time.Now().After(t.expires) {
		return ticket{}, false
	}
	return t, true
}

func (a *App) issueTicket(c *gin.Context) {
	conn, err := a.ownedConnection(c.Param("id"), userID(c))
	if err != nil {
		fail(c, http.StatusNotFound, "Sunucu bulunamadı")
		return
	}
	id, err := a.tickets.issue(strconv.FormatUint(uint64(conn.ID), 10), userID(c))
	if err != nil {
		fail(c, http.StatusInternalServerError, "Bilet üretilemedi")
		return
	}
	c.JSON(http.StatusOK, gin.H{"ticket": id, "expires_in": int(ticketTTL.Seconds())})
}
