package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

// hostKeyCallback "ilk kullanımda güven" (TOFU) uygular: ilk bağlantıda sunucu parmak izi kaydedilir,
// sonraki bağlantılarda farklıysa (olası MITM saldırısı) bağlantı reddedilir.
func (a *App) hostKeyCallback(conn *Connection) ssh.HostKeyCallback {
	return func(_ string, _ net.Addr, key ssh.PublicKey) error {
		fp := ssh.FingerprintSHA256(key)
		if conn.HostKeyFP == "" {
			conn.HostKeyFP = fp
			return a.db.Model(conn).Update("host_key_fp", fp).Error
		}
		if conn.HostKeyFP != fp {
			return fmt.Errorf("sunucu anahtarı değişti (olası MITM). Beklenen %s, gelen %s", conn.HostKeyFP, fp)
		}
		return nil
	}
}

func (a *App) upgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			return a.originAllowed(r.Header.Get("Origin"))
		},
	}
}

func (a *App) handleSSHWebSocket(c *gin.Context) {
	t, ok := a.tickets.consume(c.Param("ticket"))
	if !ok {
		fail(c, http.StatusUnauthorized, "Geçersiz veya süresi dolmuş bilet")
		return
	}
	conn, err := a.ownedConnection(t.connID, t.userID)
	if err != nil {
		fail(c, http.StatusNotFound, "Sunucu bulunamadı")
		return
	}
	ws, err := a.upgrader().Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return // Upgrade hatayı kendisi yanıtlar (ör. izin verilmeyen origin)
	}
	defer ws.Close()
	ws.SetReadLimit(64 * 1024)

	send := func(msg string) { _ = ws.WriteMessage(websocket.BinaryMessage, []byte(msg)) }

	password, err := decrypt(a.cfg.EncryptionKey, conn.PasswordEnc)
	if err != nil {
		send("\r\n--- Kayıtlı şifre çözülemedi; sunucuyu silip yeniden ekleyin ---\r\n")
		return
	}
	cfg := &ssh.ClientConfig{
		User:            conn.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: a.hostKeyCallback(conn),
		Timeout:         15 * time.Second,
	}
	addr := net.JoinHostPort(strings.TrimSpace(conn.Host), strings.TrimSpace(conn.Port))
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		send("\r\n--- SSH HATASI: " + err.Error() + " ---\r\n")
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		send("\r\n--- SSH oturumu açılamadı ---\r\n")
		return
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return
	}
	modes := ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400}
	if err := session.RequestPty("xterm-256color", 40, 80, modes); err != nil {
		send("\r\n--- PTY istenemedi ---\r\n")
		return
	}

	go func() { // SSH -> tarayıcı
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 && ws.WriteMessage(websocket.BinaryMessage, buf[:n]) != nil {
				return
			}
			if err != nil {
				return
			}
		}
	}()

	go func() { // tarayıcı -> SSH; tarayıcı kapanınca oturumu da kapat
		defer session.Close()
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				return
			}
			if _, err := stdin.Write(msg); err != nil && err != io.EOF {
				return
			}
		}
	}()

	if err := session.Shell(); err != nil {
		send("\r\n--- Kabuk başlatılamadı ---\r\n")
		return
	}
	_ = session.Wait()
}
