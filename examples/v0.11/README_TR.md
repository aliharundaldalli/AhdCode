# AhdCode v0.11 örnekleri

[English](README.md) · Türkçe

[Proje README'sine dön](../../README_TR.md)

Bu örnekler, v0.11.0 ile gelen `MySQL` standart modülünü gösterir: bağlanma,
parametreli CRUD, işlemler (transactions) ve `MySQL`'i aynı süreçte
`SQLite` ile birlikte kullanma.

| Dosya | Konu |
|---|---|
| `01_mysql_connect.ahd` | Varsayılan veritabanı olmadan bağlan, kimlik bilgilerinin görebildiği her veritabanını listele |
| `02_mysql_crud.ahd` | Parametreli INSERT/SELECT/UPDATE/DELETE, `affectedRows`, `lastInsertId` |
| `03_mysql_transaction.ahd` | Birlikte işlenen veya birlikte geri alınan iki satırlık bir transfer |
| `04_mysql_and_sqlite.ahd` | Aynı programda kullanılan `MySQL` ve `SQLite`, çakışma yok |

Tüm örnekler bağlantı bilgilerini `Env`'den okur ve yer tutucu kimlik
bilgileri kullanır. Gerçek bir parolayı asla kaynak koduna yazmayın.

## Çalıştırma

```bash
export MYSQL_HOST=127.0.0.1
export MYSQL_USERNAME=app
export MYSQL_PASSWORD=change-me
export MYSQL_DATABASE=ahdcode_demo
export MYSQL_SECURITY=tls   # güvenilir yerel geliştirme sunucusu için "none"

ahdcode run 01_mysql_connect.ahd
ahdcode run 02_mysql_crud.ahd
ahdcode run 03_mysql_transaction.ahd
ahdcode run 04_mysql_and_sqlite.ahd
```

Bu örnekler gerçek, erişilebilir bir MySQL sunucusuna ihtiyaç duyar; hiçbiri
kendisi bir sunucu başlatmaz.

## Ayrıca bakınız

- [MySQL modül belgeleri](../../docs/MYSQL.md)
- [Öğrenci Rehberi — MySQL bölümü](../../docs/STUDENT_GUIDE_TR.md#51-mysql-a%C4%9F-veritaban%C4%B1-sunucusu)
- [v0.12 MySQL çekiliş](../v0.12/raffle/README_TR.md)
