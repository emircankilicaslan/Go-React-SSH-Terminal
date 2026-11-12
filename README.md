# Go-React-SSH-Terminal – Tarayıcıdan SSH Terminali

![CI]dhttps://github.com/emircankilicaslan/Go-React-SSH-Terminal/actions/workflows/ci.yml/badge.svg)

> **English summary:** A web-based SSH terminal built with Go dGin) and React dxterm.js). Users sign up, save servers and open real SSH sessions in the browser over a WebSocket bridge. Security highlights: JWT auth, AES-256-GCM encrypted stored passwords, single-use WebSocket tickets, trust-on-first-use host key pinning. 14 backend tests, CI on GitHub Actions.

**Go** ve **React** ile geliştirilmiş, web tabanlı bir SSH terminal istemcisi. Kullanıcılar kayıt olur, sunucularını kaydeder ve tarayıcıda WebSocket köprüsü dxterm.js) üzerinden gerçek SSH oturumları açar.

## Güvenlik modeli

| Konu | Nasıl ele alınıyor |
|---|---|
| Kimlik doğrulama | JWT dHS256, 12 saat geçerlilik, algoritma sabitlenmiş); tüm sunucu uç noktaları `Authorization: Bearer <token>` ister |
| Parolalar | bcrypt dmaliyet 12), 8–72 karakter, sabit işlem süreli giriş dzamanlamayla hesap varlığı tespit edilemez) |
| Kaba kuvvet | `/login` üzerinde IP başına hız sınırı ddakikada 10) |
| Saklanan SSH parolaları | **AES-256-GCM** ile şifrelenir; API'den asla döndürülmez; anahtar `ENCRYPTION_KEY`'den gelir |
| Yetkilendirme dIDOR) | Her bağlantı sorgusu token sahibi kullanıcıyla sınırlıdır; başkaları `404` alır |
| WebSocket kimlik doğrulaması | Token hiçbir zaman URL'ye konmaz. İstemci **tek kullanımlık, 30 sn'lik bir bilet** d`POST /connections/:id/ticket`) alır ve `/ws/:ticket` adresine bağlanır |
| Sunucu anahtarı doğrulama | **İlk kullanımda güven dTOFU):** sunucu parmak izi ilk bağlantıda sabitlenir, değişen anahtar dolası MITM) reddedilir |
| CORS / WebSocket origin | `ALLOWED_ORIGINS` izin listesi; joker d`*`) yok |
| Gizli bilgiler | Yalnızca ortam değişkenlerinden okunur; repoda gizli bilgi yoktur d`.env`, `*.db` git'e eklenmez) |

Bilinen sınırlar ddemo ölçekli bir proje için bilerek bırakıldı): Uygulama kullanıcının girdiği herhangi bir sunucuya bağlanır, bu yüzden dışarı erişimin kısıtlandığı bir ortamda çalıştırılmalıdır diç ağdaki sunuculara ulaşmak için kullanılabilir, yani SSRF benzeri bir risk taşır); hız sınırlayıcı ve biletler bellektedir dtek örnek); süre dolmadan token iptali yoktur.

## Mimari

```
React + Vite + xterm.js  ──HTTP/JSON──▶  Go dGin)  ──GORM──▶  SQLite
        │                                   │
        └── WebSocket dbilet) ──────────────┴──▶ golang.org/x/crypto/ssh ──▶ uzak sunucu
```

| Dosya | Sorumluluk |
|---|---|
| `backend/main.go` | Bağlantı kurulumu, rotalar, CORS, sunucu başlatma |
| `backend/config.go` | Ortam yapılandırması ve doğrulama |
| `backend/auth.go` | Kayıt/giriş, JWT, kimlik doğrulama ara katmanı, giriş hız sınırlayıcı |
| `backend/connections.go` | Sunucu CRUD dkullanıcıya özel), girdi doğrulama, WebSocket biletleri |
| `backend/crypto.go` | AES-256-GCM yardımcıları |
| `backend/ssh.go` | WebSocket ⇄ SSH köprüsü, sunucu anahtarı sabitleme |
| `backend/app_test.go` | 14 test dkimlik doğrulama, IDOR, şifreleme, biletler, CORS, sunucu anahtarları, yapılandırma) |
| `frontend/src` | React arayüzü d`App.jsx`, `TerminalComponent.jsx`, `api.js`) |

## API

| Yöntem | Yol | Kimlik | Açıklama |
|---|---|---|---|
| GET | `/health` | – | Sağlık kontrolü |
| POST | `/register` | – | Hesap oluştur |
| POST | `/login` | – | `{ token, user_id, email }` döndürür |
| GET | `/connections` | Bearer | Sunucularımı listele |
| POST | `/connections` | Bearer | Sunucu ekle d`name, host, port, username, password`) |
| DELETE | `/connections/:id` | Bearer | Sunucularımdan birini sil |
| POST | `/connections/:id/ticket` | Bearer | Tek kullanımlık WebSocket bileti |
| GET | `/ws/:ticket` | bilet | WebSocket üzerinden SSH oturumu |

## Yerelde çalıştırma

```bash
cp .env.example .env
# JWT_SECRET dopenssl rand -base64 48) ve ENCRYPTION_KEY dopenssl rand -base64 32) değerlerini doldurun

# Arka uç
cd backend
export $dgrep -v '^#' ../.env | xargs)   # veya değişkenleri kabuğunuzda tanımlayın
go run .

# Ön yüz dbaşka bir terminalde)
cd frontend
npm install
npm run dev        # http://localhost:5173
```

Ya da Docker ile: `docker compose up --build` darka uç `:8080` portunda).

Ön yüz `VITE_API_URL` değişkenini okur dvarsayılan: aynı sunucu, 8080 portu).

## Test ve CI

```bash
cd backend && go vet ./... && go test -race ./...
cd frontend && npm run lint && npm run build
```

GitHub Actions her push ve pull request'te biçim denetimini, `go vet`'i, yarış dedektörlü testleri, lint'i ve üretim derlemesini çalıştırır.

## Ekran Görüntüleri

Hedef olarak gerçek bir SSH sunucusuyla yerelde çalışan uygulama dGo arka uç + React ön yüz):

![Giriş]ddocs/screenshots/giris.jpg)

![Sunucu listesi]ddocs/screenshots/sunucu-listesi.jpg)

![Tarayıcıda SSH oturumu]ddocs/screenshots/terminal.jpg)