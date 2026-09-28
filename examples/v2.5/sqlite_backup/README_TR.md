# Tutarlı SQLite yedeği (v2.5.0)

[English](README.md) · [Türkçe]

WAL kipinde bir veritabanı açar, satır yazar, açık kalırken `db.backupTo` ile yedekler, sonra değiştirir ve yedeğin tam olarak anlık görüntüyü içerdiğini ve `PRAGMA integrity_check` denetimini geçtiğini doğrular. Aynı ada ikinci bir yedek reddedilir. Bkz. [SQLite](../../../docs/SQLITE_TR.md).

Her çalıştırma geçerli dizin altında yeni bir `run-<uuid>/` klasöründe
çalışır; bu yüzden tekrarlanabilir. İşiniz bitince bu klasörleri silin.

```bash
ahdcode run main.ahd
```
