# MySQL çekiliş vitrini

Yerel, sunucuda üretilen bir çekiliş; küçük bir web uygulamasında kullanılan
AhdCode modüllerini gösterir: Env, HTTP, HTML, oturumlar, CSRF, Security ve
MySQL (işlemler ve `MySQLResult` dahil).

Bu bir örnek programdır; bir ürün değildir ve AhdDataStudio’nun parçası
değildir.

## Amaç

Tek gerçekçi uygulamada şunları incelemek:

- çekilişe katılıp katılım kodu almak
- diğer kodları listelemeden bilet doğrulamak
- yönetici yaşam döngüsü: idle → open → closed → drawn
- bir sonraki turda da kalan geçmiş kazananlar
- yönetici denetim kaydı

## Gösterilen özellikler

[Bu örneğin gösterdikleri](#bu-örneğin-gösterdikleri) bölümüne bakın.

## Gereksinimler

- AhdCode v0.12.0 veya sonrası (`ahdcode version`)
- Erişilebilir bir MySQL sunucusu
- Yapılandırılan kullanıcının tablo oluşturabildiği bir veritabanı

## Veritabanı oluşturma

Boş bir şema oluşturun, `MYSQL_DATABASE` ile onu gösterin:

```sql
CREATE DATABASE ahdcode_demo;
```

Tablolar ilk başlangıçta oluşur:

- `raffle_admin` — yalnızca Argon2id parola hash’leri
- `raffle` — tekil mevcut tur (`idle` / `open` / `closed` / `drawn`)
- `raffle_entry` — mevcut katılımcılar
- `raffle_history` — tamamlanmış çekilişler
- `raffle_audit` — yönetici eylemleri (parola yok, oturum kimliği yok)

[AhdDataStudio](../../../tools/AhdDataStudio/README_TR.md) ile
inceleyebilirsiniz.

## Ortam kurulumu

```bash
cd examples/v0.12/raffle
cp .env.example .env
```

Yer tutucuları düzenleyin. **`.env` dosyasını asla commit etmeyin.**

`RAFFLE_ADMIN_PASSWORD` yalnızca `raffle_admin` boşken zorunludur. İlk
çalıştırmada yoksa süreç yapılandırma hatasıyla durur ve parolayı yazmaz.
Mevcut bir yönetici satırı değiştirilmez.

## Çalıştırma

```bash
cd examples/v0.12/raffle
ahdcode run app.ahd
```

Açın: [http://127.0.0.1:8082/](http://127.0.0.1:8082/)

Yönetici: [http://127.0.0.1:8082/admin](http://127.0.0.1:8082/admin)

Durdurma:

```bash
cd examples/v0.12/raffle
ahdcode kill app.run
```

Sunucu yalnızca **127.0.0.1:8082** dinler. `0.0.0.0` kullanılmaz; 8081’deki
AhdDataStudio ile çarpışmaz.

## Rotalar

Herkese açık:

| Yöntem | Yol | Amaç |
|---|---|---|
| GET | `/` | Durum, katılım formu veya kazanan |
| POST | `/join` | Çekiliş açıkken katıl |
| GET | `/ticket` | Katılım belgesi |
| GET | `/verify` | Bilet doğrulama formu |
| POST | `/verify` | Mevcut katılım kodunu doğrula |
| GET | `/history` | Önceki kazananlar (son 25) |

Yönetici:

| Yöntem | Yol | Amaç |
|---|---|---|
| GET | `/admin` | Giriş veya pano |
| POST | `/admin/login` | Argon2id giriş |
| POST | `/admin/logout` | Yönetici oturumunu bitir |
| GET | `/admin/participants` | Sınırlı katılımcı listesi + arama |
| POST | `/admin/participant/delete` | Mevcut bir kaydı sil |
| GET | `/admin/history` | Önceki kazananlar (son 50) |
| GET | `/admin/audit` | Son 100 denetim satırı |
| POST | `/admin/start` | Yeni tur aç (`idle` iken) |
| POST | `/admin/close` | Kaydı durdur (`open` iken) |
| POST | `/admin/draw` | İşlem içinde kazanan seç (`closed` iken) |
| POST | `/admin/reset` | Sonraki tura hazırlan (`drawn` iken) |

## Güvenlik modeli

- Dinleme adresi yalnızca döngü arayüzüdür.
- Yönetici parolaları `Security.passwordHash` (Argon2id) ile hash’lenir.
  Düz metin saklanmaz ve sayfada gösterilmez.
- Giriş hataları geneldir: **Login failed**.
- Başarılı giriş `session.rotate()` çağırır ve yeni CSRF jetonu verir.
- Tarayıcıdaki her durum değişikliği POST + CSRF’tir; karşılaştırma
  `Security.secureEqual` iledir.
- Katılım kodları `Security.token()` ile üretilir ve tektir.
- Bilet kontrolü saklanan kodu `Security.secureEqual` ile karşılaştırır.
- Katılımcı tabloları tam kodları listelemez; yönetici maskeli sonek görür.
- Ad, not, kod ve denetim ayrıntısı `HTML.text` üzerinden gider (öznitelikler
  HTML üreticisi tarafından kaçırılır).
- SQL değerleri bağlı parametrelerdir. Tablo/sütun adları uygulama
  sabitleridir.
- `redact` başlangıç hata metninden MySQL ve yönetici parolalarını çıkarır.
- Listeleyen sorgular `LIMIT 25`, `50` veya `100` kullanır.

## Yalnızca yerel kullanım

Bu program güvenilir bir yerel makine içindir. Genel internete göre
sertleştirilmemiştir: oturumlar bellek içi HTTP deposudur, MySQL TLS
`MYSQL_SECURITY` ile isteğe bağlıdır, HTTP sunucusu 127.0.0.1 üzerinde
düz metindir.

## Bu örneğin gösterdikleri

- MySQL bağlantısı (`MySQL.connect`, `db.ping()`)
- parametreli sorgular (`?` + `MySQL.fromString` / `fromInt`)
- işlemler (`begin`, `MySQLTransaction` üzerinde `execute`/`query`, `commit`, `rollback`)
- `MySQLResult` (`affectedRows()`, `lastInsertId()`)
- Argon2id parolalar (`Security.passwordHash` / `passwordVerify`)
- güvenli rastgele jetonlar (`Security.token`)
- oturumlar ve girişten sonra `session.rotate()`
- CSRF (`Security.token` + `Security.secureEqual`)
- güvenli HTML (`HTML.text` / kaçırılmış öznitelikler)
- sınırlı sorgular (`LIMIT`)
- yaşam döngüsü / durum yönetimi (`idle` / `open` / `closed` / `drawn`)
- hata redaksiyonu (`redact`)
- boş olabilir veritabanı değerleri (`MySQLValue.isNull`, DECIMAL `AVG` String kalır)

## Dil

[English](README.md) · [Türkçe](README_TR.md)
