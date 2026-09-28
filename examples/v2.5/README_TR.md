# v2.5.0 sürüm örnekleri

[English](README.md) · [Türkçe]

v2.5.0 dağıtım ve sistem temel işlemlerinin küçük, odaklı örnekleri. Birlikte,
bir dağıtım işçisi olmadan onun ihtiyaç duyduğu her şeyi kapsarlar:

- [Güvenli arşiv çıkarma](archive_extract/README_TR.md): sınırlarla
  `Archive.list` ve `Archive.extract` ile retler.
- [Dosya sistemi dağıtımı](filesystem_deploy/README_TR.md): kopyalama, atomik
  yazma, izinler, atomik taşıma, gezinme ve atomik bir `current` bağlantı
  geçişi.
- [Kabuksuz süreçler](process_safe/README_TR.md): argüman listesiyle
  `Process.run`, çıkış kodları, stderr ve sınırlı hatalar.
- [SQLite yedeği](sqlite_backup/README_TR.md): canlı bir WAL veritabanının
  tutarlı `db.backupTo` anlık görüntüsü.
