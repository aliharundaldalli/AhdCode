# AhdCode v2.12.0 sürümüne geçiş

Yükseltme öncesinde indirdiğiniz dağıtım paketlerini `SHA256SUMS.txt` dosyasıyla doğrulayın. [Kurulum kılavuzuna](INSTALLATION_TR.md) ve [sürüm notlarına](RELEASE_2.12.0_TR.md) bakın.

1. Mevcut derleyiciyi `ahdcode version` ile kaydedin. Uygulama veritabanını yedekleyin ve geri yüklemeyi ayrı bir dizinde doğrulayın.
2. Üretim uygulamasını geçirmeden önce v2.12.0 sürümünü ayrı bir kurulum ön ekinde (prefix) test edin. Geri dönüş için çalışan v2.11 kurulumunu ve uygulama verilerini saklayın.
3. `ahdcode check app.ahd` komutunu çalıştırın, ardından `go.mod` dosyasındaki Go sürümüyle uygulamayı derleyin. `SQLite.openWithOptions` kullanıyorsanız derleyiciyi ve SQLite yardımcısını aynı dağıtım paketinden güncelleyin.
4. Üretim ortamına geçmeden önce geçici verilerle rotaları, yeniden başlatma sonrası oturumları, çıkışı, yetkili dosya indirmelerini ve PWA'nın çevrimdışı davranışını deneyin.

JSON denetleyicisi başarı için 0, kaynak veya giriş hatası için 1, kullanım hatası için 2 döndürür. [CLI şeması](CLI_TR.md) ve [SQLite seçenekleri ve yedekleme](SQLITE_TR.md) ayrıntıları açıklar. Özel dosyaları genel varlık dizinlerinin dışında tutun ve sunucu tarafındaki yetki denetiminden sonra açın. macOS'ta Gatekeeper korumalarını açık tutun ve indirdiğiniz kurucuların Developer ID imzasını doğrulayın.
