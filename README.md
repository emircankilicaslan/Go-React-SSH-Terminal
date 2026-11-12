# Go-React-SSH-Terminal – Tarayıcıdan SSH Terminali

![CI](https://github.com/emircankilicaslan/Go-React-SSH-Terminal/actions/workflows/ci.yml/badge.svg)

> **English summary:** A web-based SSH terminal built with Go (Gin) and React (xterm.js). Users sign up, save servers and open real SSH sessions in the browser over a WebSocket bridge. Security highlights: JWT auth, AES-256-GCM encrypted stored passwords, single-use WebSocket tickets, trust-on-first-use host key pinning. 14 backend tests, CI on GitHub Actions.

**Go** ve **React** ile geliştirilmiş, web tabanlı bir SSH terminal istemcisi. Kullanıcılar kayıt olur, sunucularını kaydeder ve tarayıcıda WebSocket köprüsü (xterm.js) üzerinden gerçek SSH oturumları açar.

## Güvenlik modeli

| Konu | Nasıl ele alınıyor |
|---|---|
| Kimlik doğrulama | JWT (HS256, 12 saat geçerlilik, algoritma sabitlenmiş); tüm sunucu uç noktaları `Authorization: Bearer <token>` ister |
| Parolalar | bcrypt (maliyet 12), 8–72 karakter, sabit işlem süreli giriş (zamanlamayla hesap varlığı tespit edilemez) |
| Kaba kuvvet | `/login` üzerinde IP başına hız sınırı (dakikada 10) |
| Saklanan SSH parolaları | **AES-256-GCM** ile şifrelenir; API'den asla döndürülmez; anahtar `ENCRYPTION_KEY`'den gelir |
| Yetkilendirme (IDOR) | Her bağlantı sorgusu token sahibi kullanıcıyla sınırlıdır; başkaları `404` alır |
| WebSocket kimlik doğrulaması | Token hiçbir zaman URL'ye konmaz. İstemci **tek kullanımlık, 30 sn'lik bir bilet** (`POST /connections/:id/ticket`) alır ve `/ws/:ticket` adresine bağlanır |
| Sunucu anahtarı doğrulama | **İlk kullanımda güven (TOFU):** sunucu parmak izi ilk bağlantıda sabitlenir, değişen anahtar (olası MITM) reddedilir |
| CORS / WebSocket origin | `ALLOWED_ORIGINS` izin listesi; joker (`*`) yok |
| Gizli bilgiler | Yalnızca ortam değişkenlerinden okunur; repoda gizli bilgi yoktur (`.env`, `*.db` git'e eklenmez) |

Bilinen sınırlar (demo ölçekli bir proje için bilerek bırakıldı): Uygulama kullanıcının girdiği herhangi bir sunucuya bağlanır, bu yüzden dışarı erişimin kısıtlandığı bir ortamda çalıştırılmalıdır (iç ağdaki sunuculara ulaşmak için kullanılabilir, yani SSRF benzeri bir risk taşır); hız sınırlayıcı ve biletler bellektedir (tek örnek); süre dolmadan token iptali yoktur.

## Mimari

```
React + Vite + xterm.js  ──HTTP/JSON──▶  Go (Gin)  ──GORM──▶  SQLite
        │                                   │
        └── WebSocket (bilet) ──────────────┴──▶ golang.org/x/crypto/ssh ──▶ uzak sunucu
```

| Dosya | Sorumluluk |
|---|---|
| `backend/main.go` | Bağlantı kurulumu, rotalar, CORS, sunucu başlatma |
| `backend/config.go` | Ortam yapılandırması ve doğrulama |
| `backend/auth.go` | Kayıt/giriş, JWT, kimlik doğrulama ara katmanı, giriş hız sınırlayıcı |
| `backend/connections.go` | Sunucu CRUD (kullanıcıya özel), girdi doğrulama, WebSocket biletleri |
| `backend/crypto.go` | AES-256-GCM yardımcıları |
| `backend/ssh.go` | WebSocket ⇄ SSH köprüsü, sunucu anahtarı sabitleme |
| `backend/app_test.go` | 14 test (kimlik doğrulama, IDOR, şifreleme, biletler, CORS, sunucu anahtarları, yapılandırma) |
| `frontend/src` | React arayüzü (`App.jsx`, `TerminalComponent.jsx`, `api.js`) |

## API

| Yöntem | Yol | Kimlik | Açıklama |
|---|---|---|---|
| GET | `/health` | – | Sağlık kontrolü |
| POST | `/register` | – | Hesap oluştur |
| POST | `/login` | – | `{ token, user_id, email }` döndürür |
| GET | `/connections` | Bearer | Sunucularımı listele |
| POST | `/connections` | Bearer | Sunucu ekle (`name, host, port, username, password`) |
| DELETE | `/connections/:id` | Bearer | Sunucularımdan birini sil |
| POST | `/connections/:id/ticket` | Bearer | Tek kullanımlık WebSocket bileti |
| GET | `/ws/:ticket` | bilet | WebSocket üzerinden SSH oturumu |

## Yerelde çalıştırma

```bash
cp .env.example .env
# JWT_SECRET (openssl rand -base64 48) ve ENCRYPTION_KEY (openssl rand -base64 32) değerlerini doldurun

# Arka uç
cd backend
export $(grep -v '^#' ../.env | xargs)   # veya değişkenleri kabuğunuzda tanımlayın
go run .

# Ön yüz (başka bir terminalde)
cd frontend
npm install
npm run dev        # http://localhost:5173
```

Ya da Docker ile: `docker compose up --build` (arka uç `:8080` portunda).

Ön yüz `VITE_API_URL` değişkenini okur (varsayılan: aynı sunucu, 8080 portu).

## Test ve CI

```bash
cd backend && go vet ./... && go test -race ./...
cd frontend && npm run lint && npm run build
```

GitHub Actions her push ve pull request'te biçim denetimini, `go vet`'i, yarış dedektörlü testleri, lint'i ve üretim derlemesini çalıştırır.

## Ekran Görüntüleri

Hedef olarak gerçek bir SSH sunucusuyla yerelde çalışan uygulama (Go arka uç + React ön yüz):

![Giriş](docs/screenshots/giris.jpg)

![Sunucu listesi](docs/screenshots/sunucu-listesi.jpg)

![Tarayıcıda SSH oturumu](docs/screenshots/terminal.jpg)