package main

import (
	"log"
	"net/http"
	"strings"
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
		log.Fatal(err)
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
	r.Run(":8080")
}

func register(c *gin.Context) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	c.BindJSON(&body)
	hash, _ := bcrypt.GenerateFromPassword([]byte(body.Password), 14)
	db.Create(&User{Email: body.Email, Password: string(hash)})
	c.JSON(200, gin.H{"message": "Success"})
}

func login(c *gin.Context) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	c.BindJSON(&body)
	var user User
	db.Where("email = ?", body.Email).First(&user)
	if user.ID == 0 || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password)) != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	c.JSON(200, gin.H{"user_id": user.ID, "email": user.Email})
}

func getConnections(c *gin.Context) {
	userID := c.Query("user_id")
	var connections []Connection
	if userID != "" {
		db.Where("user_id = ?", userID).Find(&connections)
	}
	c.JSON(200, connections)
}

func addConnection(c *gin.Context) {
	var conn Connection
	c.ShouldBindJSON(&conn)
	db.Create(&conn)
	c.JSON(200, conn)
}

func deleteConnection(c *gin.Context) {
	db.Delete(&Connection{}, c.Param("id"))
	c.JSON(200, gin.H{"message": "Deleted"})
}

func handleSSHWebSocket(c *gin.Context) {
	id := c.Param("id")
	var conn Connection
	if err := db.First(&conn, id).Error; err != nil { return }
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil { return }
	defer ws.Close()
	config := &ssh.ClientConfig{
		User:            conn.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(conn.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}
	addr := strings.TrimSpace(conn.Host) + ":" + strings.TrimSpace(conn.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		ws.WriteMessage(websocket.TextMessage, []byte("\r\nBAĞLANTI HATASI: " + err.Error()))
		return
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil { return }
	defer session.Close()
	stdin, _ := session.StdinPipe()
	stdout, _ := session.StdoutPipe()
	modes := ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400}
	session.RequestPty("xterm-256color", 40, 80, modes)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if err != nil { return }
			ws.WriteMessage(websocket.BinaryMessage, buf[:n])
		}
	}()
	go func() {
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil { return }
			stdin.Write(msg)
		}
	}()
	session.Shell()
	session.Wait()
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