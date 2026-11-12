package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

func init() { bcryptCost = bcrypt.MinCost } // testler hızlı çalışsın

func testConfig() Config {
	return Config{
		Port:           "0",
		JWTSecret:      []byte(strings.Repeat("s", 32)),
		EncryptionKey:  bytes.Repeat([]byte{7}, 32),
		AllowedOrigins: []string{"http://localhost:5173"},
	}
}

func newTestApp(t *testing.T) (*App, *gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := openDB(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	app, err := newApp(testConfig(), db)
	if err != nil {
		t.Fatal(err)
	}
	return app, app.routes(), db
}

func do(r http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func registerAndLogin(t *testing.T, r http.Handler, email string) string {
	t.Helper()
	cred := map[string]string{"email": email, "password": "correct-horse-battery"}
	if w := do(r, "POST", "/register", "", cred); w.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", w.Code, w.Body)
	}
	w := do(r, "POST", "/login", "", cred)
	if w.Code != http.StatusOK {
		t.Fatalf("login: %d %s", w.Code, w.Body)
	}
	var out struct{ Token string }
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return out.Token
}

var sampleServer = map[string]string{"name": "web", "host": "10.0.0.5", "port": "22", "username": "root", "password": "ssh-secret-123"}

func TestRegisterValidationAndDuplicate(t *testing.T) {
	_, r, _ := newTestApp(t)
	for _, bad := range []map[string]string{
		{"email": "not-an-email", "password": "longenough1"},
		{"email": "a@b.co", "password": "short"},
		{"email": "", "password": ""},
	} {
		if w := do(r, "POST", "/register", "", bad); w.Code != http.StatusBadRequest {
			t.Errorf("%v: beklenen 400, gelen %d", bad, w.Code)
		}
	}
	registerAndLogin(t, r, "ayse@example.com")
	dup := map[string]string{"email": "AYSE@example.com", "password": "correct-horse-battery"}
	if w := do(r, "POST", "/register", "", dup); w.Code != http.StatusConflict {
		t.Errorf("aynı e-posta: beklenen 409, gelen %d", w.Code)
	}
}

func TestLoginWrongPasswordAndUnknownUser(t *testing.T) {
	_, r, _ := newTestApp(t)
	registerAndLogin(t, r, "ayse@example.com")
	for _, c := range []map[string]string{
		{"email": "ayse@example.com", "password": "wrong-password"},
		{"email": "yok@example.com", "password": "whatever-123"},
	} {
		if w := do(r, "POST", "/login", "", c); w.Code != http.StatusUnauthorized {
			t.Errorf("%v: beklenen 401, gelen %d", c, w.Code)
		}
	}
}

func TestLoginRateLimited(t *testing.T) {
	app, r, _ := newTestApp(t)
	app.limiter = &loginLimiter{byIP: map[string]*rate.Limiter{}, limit: 0, burst: 10} // yenilenme yok: zamandan bağımsız test
	var last int
	for i := 0; i < 12; i++ {
		last = do(r, "POST", "/login", "", map[string]string{"email": "x@y.zz", "password": "nope-nope-1"}).Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("12. denemede 429 beklenirdi, gelen %d", last)
	}
}

func TestProtectedRoutesRequireValidToken(t *testing.T) {
	_, r, _ := newTestApp(t)
	for _, tok := range []string{"", "garbage", "eyJhbGciOiJub25lIn0.e30."} {
		if w := do(r, "GET", "/connections", tok, nil); w.Code != http.StatusUnauthorized {
			t.Errorf("token %q: beklenen 401, gelen %d", tok, w.Code)
		}
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	app, r, _ := newTestApp(t)
	registerAndLogin(t, r, "ayse@example.com")
	tok, _ := app.issueToken(1)
	app.cfg.JWTSecret = []byte(strings.Repeat("x", 32)) // imza anahtarı değişti
	if w := do(r, "GET", "/connections", tok, nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("farklı anahtarla imzalı token kabul edilmemeli, gelen %d", w.Code)
	}
}

func TestConnectionPasswordIsEncryptedAndNeverReturned(t *testing.T) {
	_, r, db := newTestApp(t)
	tok := registerAndLogin(t, r, "ayse@example.com")

	w := do(r, "POST", "/connections", tok, sampleServer)
	if w.Code != http.StatusCreated {
		t.Fatalf("ekleme: %d %s", w.Code, w.Body)
	}
	if strings.Contains(w.Body.String(), "ssh-secret-123") || strings.Contains(strings.ToLower(w.Body.String()), "password") {
		t.Fatalf("yanıt şifre içeriyor: %s", w.Body)
	}
	list := do(r, "GET", "/connections", tok, nil)
	if strings.Contains(list.Body.String(), "ssh-secret-123") {
		t.Fatal("liste şifre içeriyor")
	}

	var stored Connection
	db.First(&stored)
	if stored.PasswordEnc == "" || strings.Contains(stored.PasswordEnc, "ssh-secret-123") {
		t.Fatal("şifre veritabanında düz metin saklanıyor")
	}
	plain, err := decrypt(testConfig().EncryptionKey, stored.PasswordEnc)
	if err != nil || plain != "ssh-secret-123" {
		t.Fatalf("şifre çözülemedi: %v %q", err, plain)
	}
}

func TestConnectionValidation(t *testing.T) {
	_, r, _ := newTestApp(t)
	tok := registerAndLogin(t, r, "ayse@example.com")
	for name, mutate := range map[string]func(m map[string]string){
		"port":     func(m map[string]string) { m["port"] = "99999" },
		"host":     func(m map[string]string) { m["host"] = "evil.com/../x" },
		"username": func(m map[string]string) { m["username"] = "" },
		"password": func(m map[string]string) { m["password"] = "" },
	} {
		body := map[string]string{}
		for k, v := range sampleServer {
			body[k] = v
		}
		mutate(body)
		if w := do(r, "POST", "/connections", tok, body); w.Code != http.StatusBadRequest {
			t.Errorf("%s: beklenen 400, gelen %d", name, w.Code)
		}
	}
}

// Başka kullanıcının kaydına erişim (IDOR) engellenmeli.
func TestUsersCannotAccessEachOthersConnections(t *testing.T) {
	_, r, _ := newTestApp(t)
	alice := registerAndLogin(t, r, "alice@example.com")
	bob := registerAndLogin(t, r, "bob@example.com")

	var created Connection
	_ = json.Unmarshal(do(r, "POST", "/connections", alice, sampleServer).Body.Bytes(), &created)

	if body := do(r, "GET", "/connections", bob, nil).Body.String(); strings.TrimSpace(body) != "[]" {
		t.Errorf("bob alice'in sunucularını görmemeli: %s", body)
	}
	if w := do(r, "DELETE", fmt.Sprintf("/connections/%d", created.ID), bob, nil); w.Code != http.StatusNotFound {
		t.Errorf("bob silememeli, gelen %d", w.Code)
	}
	if w := do(r, "POST", fmt.Sprintf("/connections/%d/ticket", created.ID), bob, nil); w.Code != http.StatusNotFound {
		t.Errorf("bob bilet alamamalı, gelen %d", w.Code)
	}
	if w := do(r, "DELETE", fmt.Sprintf("/connections/%d", created.ID), alice, nil); w.Code != http.StatusOK {
		t.Errorf("alice silebilmeli, gelen %d", w.Code)
	}
}

func TestTicketIsSingleUseAndExpires(t *testing.T) {
	app, _, _ := newTestApp(t)
	id, err := app.tickets.issue("1", 5)
	if err != nil {
		t.Fatal(err)
	}
	if tk, ok := app.tickets.consume(id); !ok || tk.userID != 5 {
		t.Fatal("ilk kullanım başarılı olmalı")
	}
	if _, ok := app.tickets.consume(id); ok {
		t.Fatal("bilet ikinci kez kullanılamamalı")
	}

	id2, _ := app.tickets.issue("1", 5)
	app.tickets.mu.Lock()
	tk := app.tickets.m[id2]
	tk.expires = time.Now().Add(-time.Second)
	app.tickets.m[id2] = tk
	app.tickets.mu.Unlock()
	if _, ok := app.tickets.consume(id2); ok {
		t.Fatal("süresi dolan bilet kabul edilmemeli")
	}
}

func TestWebSocketRejectsInvalidTicket(t *testing.T) {
	_, r, _ := newTestApp(t)
	if w := do(r, "GET", "/ws/uydurma-bilet", "", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("beklenen 401, gelen %d", w.Code)
	}
}

func TestCORSOnlyForAllowedOrigins(t *testing.T) {
	_, r, _ := newTestApp(t)
	req := httptest.NewRequest("OPTIONS", "/login", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("izinli origin yansıtılmalı, gelen %q", got)
	}

	req = httptest.NewRequest("OPTIONS", "/login", nil)
	req.Header.Set("Origin", "http://evil.example")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("izinsiz origin için başlık olmamalı, gelen %q", got)
	}
}

func TestHostKeyTrustOnFirstUseThenPin(t *testing.T) {
	app, _, db := newTestApp(t)
	conn := &Connection{UserID: 1, Name: "x", Host: "h", Port: "22", Username: "u", PasswordEnc: "x"}
	db.Create(conn)

	newKey := func() ssh.PublicKey {
		pub, _, _ := ed25519.GenerateKey(rand.Reader)
		k, _ := ssh.NewPublicKey(pub)
		return k
	}
	first, other := newKey(), newKey()
	cb := app.hostKeyCallback(conn)

	if err := cb("h", nil, first); err != nil {
		t.Fatalf("ilk bağlantı kabul edilmeli: %v", err)
	}
	var saved Connection
	db.First(&saved, conn.ID)
	if saved.HostKeyFP != ssh.FingerprintSHA256(first) {
		t.Fatal("parmak izi kaydedilmedi")
	}
	if err := app.hostKeyCallback(&saved)("h", nil, first); err != nil {
		t.Fatalf("aynı anahtar kabul edilmeli: %v", err)
	}
	if err := app.hostKeyCallback(&saved)("h", nil, other); err == nil {
		t.Fatal("farklı sunucu anahtarı reddedilmeli (MITM)")
	}
}

func TestCryptoRoundTripAndTamper(t *testing.T) {
	key := bytes.Repeat([]byte{1}, 32)
	enc, err := encrypt(key, "gizli")
	if err != nil {
		t.Fatal(err)
	}
	enc2, _ := encrypt(key, "gizli")
	if enc == enc2 {
		t.Fatal("her şifrelemede farklı nonce kullanılmalı")
	}
	if got, err := decrypt(key, enc); err != nil || got != "gizli" {
		t.Fatalf("round-trip başarısız: %v %q", err, got)
	}
	raw, _ := base64.StdEncoding.DecodeString(enc)
	raw[len(raw)-1] ^= 0xff
	if _, err := decrypt(key, base64.StdEncoding.EncodeToString(raw)); err == nil {
		t.Fatal("bozulan veri reddedilmeli")
	}
	if _, err := decrypt(bytes.Repeat([]byte{2}, 32), enc); err == nil {
		t.Fatal("yanlış anahtar reddedilmeli")
	}
}

func TestLoadConfigRequiresSecrets(t *testing.T) {
	t.Setenv("JWT_SECRET", "kisa")
	t.Setenv("ENCRYPTION_KEY", "")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("kısa JWT_SECRET reddedilmeli")
	}
	t.Setenv("JWT_SECRET", strings.Repeat("a", 32))
	if _, err := LoadConfig(); err == nil {
		t.Fatal("eksik ENCRYPTION_KEY reddedilmeli")
	}
	t.Setenv("ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32)))
	cfg, err := LoadConfig()
	if err != nil || len(cfg.EncryptionKey) != 32 || cfg.AllowedOrigins[0] != "http://localhost:5173" {
		t.Fatalf("geçerli ayar kabul edilmeli: %v", err)
	}
}
