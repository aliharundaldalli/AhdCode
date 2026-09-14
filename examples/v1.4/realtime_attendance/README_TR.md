# v1.4.0 gerçek zamanlı yoklama

Yoklama kayıtlarını PostgreSQL'e yazan ve yenilerini açık olan her panoda
WebSocket üzerinden anında gösteren küçük bir Web uygulaması. v1.4.0'ın
dogfood uygulamasıdır: Web, PostgreSQL, WebSocket, UUID v7, `Env.secret` ve
Cron bir arada. npm, CDN ya da derleme veya çalışma sırasında indirme yoktur.

## Çalıştırma

```bash
cp .env.example .env
```

`.env` içinde `DB_PASSWORD` (ya da parolayı tutan dosyanın yolu olan
`DB_PASSWORD_FILE`) ve `NOTIFY_TOKEN` değerlerini ayarlayın, ardından tabloyu
bir kez oluşturun:

```bash
psql "host=127.0.0.1 dbname=attendance user=attendance" -f schema.sql
```

```bash
ahdcode dev app.ahd
```

İsterseniz ikinci bir terminalde:

```bash
ahdcode run jobs.ahd
```

## Ne yapar

| İstek | Sonuç |
| --- | --- |
| `GET /` | giriş formu ya da son 20 yoklamayı gösteren pano |
| `POST /sign-in` | CSRF denetimli; oturuma `user_id` ve `user_name` yazar |
| `POST /check-in` | CSRF denetimli; `UUID.v7()` ile ekler, `created_at` değerini `RETURNING` ile geri okur, tek satır JSON yayınlar, yönlendirir |
| `GET /live` (WebSocket) | yalnızca aynı köken; giriş yapılmış oturum yoksa `401` ile reddedilir; salt okunur |
| `POST /internal/notify` | `Authorization: Bearer <NOTIFY_TOKEN>` gerekir; gövdeyi duyuru olarak yayınlar |
| `POST /sign-out` | CSRF denetimli; oturumu temizler |

`jobs.ahd` ikinci bir programdır. Bir Web sunucusu da bir Cron zamanlayıcısı
da onları çalıştıran programı meşgul eder; bu yüzden ikisi yan yana çalışır ve
veritabanını paylaşır. Dakikada bir, son bir saatteki yoklamaları sayar ve
özeti `/internal/notify` adresine gönderir.

## Dosyalar

| Dosya | Görevi |
| --- | --- |
| `app.ahd` | yapılandırma, rotalar, `/live` uç noktası, yönetilen varlıklar |
| `Config/Database.ahd` | paylaşılan PostgreSQL havuzu; parola `Env.secret("DB_PASSWORD")` ile gelir |
| `Config/Sessions.ahd` | rotaların ve `/live` kabul denetiminin kullandığı oturum deposu |
| `Live.ahd` | WebSocket uç noktası, kabul denetimi ve soket kaydı |
| `Pages/` | giriş, pano, yoklama ve bearer token korumalı bildirim uç noktası |
| `jobs.ahd` | özeti `HTTP.client` ile gönderen Cron programı |
| `public/js/live.js` | panonun canlı listesi; görünür bir geri sayımla yeniden bağlanır |
| `schema.sql` | `check_ins` tablosu |

## Notlar

- Canlı mesajlar `JSON.object` ile oluşturulur ve `live.js` bunları
  `textContent` ile yazar; öğrenci adı her zaman metin olarak kalır. Sunucuda
  işlenen adları `Web.UI` kaçışlar.
- WebSocket geri çağrıları HTTP işleyicileriyle birlikte sırayla, tek tek
  çalışır; bu yüzden soket kaydı sıradan bir `Pair<String, WebSocket>`'tir.
- Bildirim koruması `request.header`, `Env.secret` ve `Security.secureEqual`
  ile kurulmuş bir kalıptır. Kapsamlı bir API kimlik doğrulama çatısı değildir.
- Ters vekil (reverse proxy) arkasında `/live` için `Upgrade` ve `Connection`
  başlıklarını iletin; bkz. [WebSocket](../../../docs/WEBSOCKET_TR.md).
