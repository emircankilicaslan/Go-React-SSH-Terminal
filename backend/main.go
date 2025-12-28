package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)


type User struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Email    string `gorm:"unique" json:"email"`
	Password string `json:"-"` 
}

type Connection struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	UserID   uint   `json:"user_id"` 
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

var db *gorm.DB
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	
	var err error
	db, err = gorm.Open(sqlite.Open("fullstackx.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Veritabanı hatası:", err)
	}
	
	db.AutoMigrate(&User{}, &Connection{})

	r := gin.Default()
	r.Use(corsMiddleware())

	
	r.POST("/register", register)
	r.POST("/login", login)

	
	r.GET("/connections", getConnections)
	r.POST("/connections", addConnection)
	r.DELETE("/connections/:id", deleteConnection)

	
	r.GET("/ws/:id", handleSSHWebSocket)

	fmt.Println("🚀 FullStackX FINAL Backend 8080 portunda çalışıyor...")
	r.Run(":8080")
}



func register(c *gin.Context) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if c.BindJSON(&body) != nil {
		c.JSON(400, gin.H{"error": "Eksik veri"})
		return
	}
	
	hash, _ := bcrypt.GenerateFromPassword([]byte(body.Password), 14)
	user := User{Email: body.Email, Password: string(hash)}
	
	if err := db.Create(&user).Error; err != nil {
		c.JSON(400, gin.H{"error": "Bu email zaten kayıtlı!"})
		return
	}
	c.JSON(200, gin.H{"message": "Kayıt başarılı! Giriş yapın."})
}

func login(c *gin.Context) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if c.BindJSON(&body) != nil {
		c.JSON(400, gin.H{"error": "Eksik veri"})
		return
	}
	var user User
	if err := db.Where("email = ?", body.Email).First(&user).Error; err != nil {
		c.JSON(404, gin.H{"error": "Kullanıcı bulunamadı"})
		return
	}
	
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password)); err != nil {
		c.JSON(401, gin.H{"error": "Hatalı şifre"})
		return
	}
	
	c.JSON(200, gin.H{"token": "fake-jwt-token", "user_id": user.ID, "email": user.Email})
}

func getConnections(c *gin.Context) {
	var connections []Connection
	db.Find(&connections)
	c.JSON(200, connections)
}

func addConnection(c *gin.Context) {
	var conn Connection
	if err := c.ShouldBindJSON(&conn); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	db.Create(&conn)
	c.JSON(200, conn)
}

func deleteConnection(c *gin.Context) {
	id := c.Param("id")
	db.Delete(&Connection{}, id)
	c.JSON(200, gin.H{"message": "Silindi"})
}

func handleSSHWebSocket(c *gin.Context) {
	id := c.Param("id")
	var conn Connection
	if err := db.First(&conn, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "Sunucu Yok"})
		return
	}

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil { return }
	defer ws.Close()

	config := &ssh.ClientConfig{
		User: conn.Username,
		Auth: []ssh.AuthMethod{ ssh.Password(conn.Password) },
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout: 5 * time.Second,
	}

	client, err := ssh.Dial("tcp", conn.Host+":"+conn.Port, config)
	if err != nil {
		ws.WriteMessage(websocket.TextMessage, []byte("\r\nBAĞLANTI HATASI: "+err.Error()+"\r\n"))
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil { return }
	defer session.Close()

	modes := ssh.TerminalModes{ ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400 }
	session.RequestPty("xterm", 40, 80, modes)

	stdin, _ := session.StdinPipe()
	stdout, _ := session.StdoutPipe()
	
	go func() {
		io.Copy(ws.UnderlyingConn(), stdout)
	}()

	session.Shell()
	
	for {
		_, msg, err := ws.ReadMessage()
		if err != nil { break }
		stdin.Write(msg)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, DELETE, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}