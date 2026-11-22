# AhdCode v2.12.0 — Bağımsız Denetleyici, SQLite Seçenekleri ve Platform Güvenilirliği

[English](RELEASE_2.12.0.md) · [Kurulum](INSTALLATION_TR.md) · [Geçiş](UPGRADE_2.12_TR.md)

- `ahdcode check app.ahd --format json` modül çözümleme, ayrıştırma ve anlamsal analizi çalıştırır. Go kurulu olmadan çalışır ve UTF-16 konumlarıyla sürüm 1 JSON tanıları üretir.
- `SQLite.openWithOptions(path, readOnly, busyTimeoutMilliseconds)`, salt okunur bağlantı ve belirlenebilir kilit bekleme süresi sağlar. Mevcut `SQLite.open` varsayılanları korunur. Yedekleme tutarlı bir anlık görüntü alır ve mevcut hedef dosyayı reddeder.
- LSP taraması sınırlıdır. Eksik taramada yeniden adlandırma reddedilir. `require(...)` dosyalarındaki düzenlemeler konumun ait olduğu kaynağa uygulanır.
- Genel HTTP bağlamaları dizin ve dosya kimliğini denetler. Dosya yanıtlarının varsayılanı `Cache-Control: no-store` değeridir. Sahiplik ve ret yanıtlarının önbellek politikası uygulamanın sunucu tarafındaki sorumluluğudur.
- Studio eksik `.env` dosyasını özel izinlerle ve üzerine yazmayı önleyerek oluşturur. Windows'ta GUI kaynakları ve genel HTTP bağlamaları dizin bağlantılarının hedefini açılan tanıtıcıdan çözer. Disk incelemesi mevcut dosya yollarını da kabul eder.
- Yerel paketler v2.11 araç zinciri düzenini korur. macOS `.pkg` ve `.dmg` kurucuları Apple Developer ID ile imzalanmış, noter onayından geçmiş, bilet zımbalanmış ve Gatekeeper ile doğrulanmıştır; `.tar.gz` arşivleri imzalı Mach-O ikilileri taşır, Windows kurucusu ve Linux arşivleri ise `SHA256SUMS.txt` doğrulama özetleriyle yayımlanır.

Paketleme hedefleri Windows x64, Linux amd64/arm64 ve macOS arm64/x64'tür. Çapraz derlenen paketler hedef sistemde yerel doğrulama gerektirir. Yalnızca araç zinciri içeren paketler için derleme ve paketleme komutları `go.mod` dosyasındaki Go sürümünü gerektirir.
