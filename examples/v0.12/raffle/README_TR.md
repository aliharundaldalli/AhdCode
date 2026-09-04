# MySQL çekiliş örneği

Küçük bir yerel çekiliş: üyeler katılıp bir kod alır, yönetici çekilişi
başlatır, sonra kazananı ilan eder. Parolalar `Security.passwordHash`
(Argon2id) ile saklanır; düz metin tutulmaz.

Bu bir örnek programdır; bir ürün değildir ve AhdDataStudio’nun parçası
değildir.

## Başlatma

```bash
cd examples/v0.12/raffle
cp .env.example .env   # ardından yer tutucuları düzenleyin
ahdcode run app.ahd
```

Açın: [http://127.0.0.1:8082/](http://127.0.0.1:8082/)

Yönetici: [http://127.0.0.1:8082/admin](http://127.0.0.1:8082/admin)

Varsayılan yönetici (`.env` içinde değiştirin): `admin` / `change-me-local-only`

İlk çalıştırma bu parolayı Argon2id ile hash’ler ve yalnızca hash’i saklar.
`.env`’i sonradan değiştirmek mevcut `raffle_admin` satırını güncellemez.

## Durdurma

```bash
cd examples/v0.12/raffle
ahdcode kill app.run
```

Sunucu yalnızca **127.0.0.1:8082** dinler; 8081’deki AhdDataStudio ile
çarpışmaz.

## Ne gösterir

- MySQL `CREATE TABLE` / parametreli `INSERT` / `UPDATE` / `SELECT`
- Kazananı seçmek için `ORDER BY RAND() LIMIT 1`
- Argon2id yönetici parola hash’i
- HTTP oturumları ve POST formlarında CSRF
- İsim ve kodlar için `HTML.text`

Tablolar `MYSQL_DATABASE` içinde oluşur (varsayılan `ahdcode_demo`).
AhdDataStudio ile inceleyebilirsiniz.

## Dil

[English](README.md) · [Türkçe](README_TR.md)
