# Archive standart modülü

[English](ARCHIVE.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [PDF](PDF_TR.md)

Archive, dosyaları yalnızca Go standart kütüphanesini (`archive/zip`,
`archive/tar`, `compress/gzip`) kullanarak çevrimdışı gerçek ZIP, TAR ve
TAR.GZ arşivlerine paketler. Açıkça içe aktarın:

```ahd
bring Archive
from Archive bring ArchiveError
```

Kanonik modül kimliği `builtin:Archive`'dır; bir kardeş `Archive.ahd` dosyası
onu gölgeleyemez.

Archive **yalnızca oluşturma** amaçlıdır: çıkarma, listeleme veya arşiv nesne
modeli yoktur. `Archive.extract`, `Archive.open` ve benzerleri yoktur ve bu
modüle eklenmeyecektir — bkz. [Bu sürümde yok](#bu-sürümde-yok).

## Yüzey

```text
Archive.zip(output: String, entries: Pair<String, String>)     -> Nothing
Archive.tar(output: String, entries: Pair<String, String>)     -> Nothing
Archive.tarGzip(output: String, entries: Pair<String, String>) -> Nothing

ArchiveError
```

## Girdi eşlemesi

`entries` sıradan bir `Pair<String, String>`'dır: her anahtar arşiv
*içindeki* yoldur, her değer ise paketlenecek *kaynak dosya sistemi
yoludur*. Eşleme her zaman açıktır — Archive hiçbir zaman bir kaynak yoldan
hedef adı tahmin etmez.

```ahd
files := {
    "report/report.pdf": "output/report.pdf"
    "data/results.json": "results.json"
    "images/chart.png": "chart.png"
}

Archive.zip("submission.zip", files)
```

## Yalnızca normal dosyalar

v0.1.20 Archive yalnızca normal dosyaları kabul eder — dizin kaynağı yoktur,
özyinelemeli genişletme yoktur. Bir dizin, sembolik bağlantı veya başka bir
normal olmayan dosya olan kaynak `ArchiveError` fırlatır. Bu, ilk sürüm için
güvenlik argümanını (yol doğrulama, sembolik bağlantı işleme, sıralama) küçük
ve tamamen denetlenebilir tutar.

## Girdi yolu güvenliği

Archive üye adları kanonik göreli ileri-eğik-çizgi yollarıdır. Aşağıdakilerin
her biri doğrudan reddedilir — asla sessizce başka bir şeye normalleştirilmez:

- boş ad
- mutlak yol (`/etc/...`)
- bir `..` veya `.` yol parçası (`../escape`, `a/../b`, `./file`)
- çift eğik çizgi (`a//b`)
- ters eğik çizgi (`a\b`)
- NUL baytı
- Windows sürücü-ön-eki benzeri bir parça (`C:file`)

## Sembolik bağlantılar

Sembolik bağlantı olan bir kaynak, sessizce izlenmek, saklanmak veya
çözülmek yerine `ArchiveError` ile reddedilir.

## Çakışmalar

`Pair` zaten benzersiz anahtarlar garanti eder, bu yüzden iki girdi aynı
arşiv üyesini adlandıramaz; Archive ayrıca bunu savunma amaçlı kontrol eder.
`son kazanır`, `ilk kazanır` veya sessiz üzerine yazma davranışı yoktur.

## Determinizm ve sıralama

Archive üye sırası tam olarak `Pair` ekleme sırasını izler. Aksi takdirde
çalıştırmadan çalıştırmaya değişebilecek arşiv meta verisi normalleştirilir
(zaman damgaları, sahip/grup, gzip başlık alanları kaldırılır; sabit `0644`
kipi kullanılır). Dosya **içeriği** tam olarak korunur; eşdeğer girdilerden
oluşturulan iki arşiv bayt-bayt aynıdır.

## Biçim ve uzantı

Çağırdığınız fonksiyon biçimi seçer, ancak uyumsuz bir uzantı yine de yanlış
baytları yanlış ada sessizce yazmak yerine `ArchiveError` fırlatır:

```text
Archive.zip     ->  çıktı .zip ile bitmeli
Archive.tar     ->  çıktı .tar ile bitmeli
Archive.tarGzip ->  çıktı .tar.gz ile bitmeli (.tgz değil)
```

## Çıktı güvenliği

Archive, tam arşivi aynı dizinde bir geçici dosyaya inşa eder, sonra
hedefin üzerine atomik olarak yeniden adlandırır. Başarısız bir inşa, hedefte
mevcut geçerli bir arşive asla dokunmaz; hedef arşivin kendisine çözülen bir
kaynak yol, hiçbir şey yazılmadan önce reddedilir.

## Hatalar

`ArchiveError`, her Archive'e özgü hatayı kapsar: eksik veya okunamayan bir
kaynak, desteklenmeyen bir kaynak türü, geçersiz bir girdi yolu, yanlış bir
çıktı uzantısı ve bir arşiv yazıcısı hatası.

## Bu sürümde yok

Archive çıkarma, arşiv listeleme, bir arşiv nesne modeli, RAR, 7z, BZIP2, XZ,
bağımsız bir Compress modülü, şifreli/parola korumalı arşivler ve dizin
kaynağı özyinelemesi v0.1.20'nin parçası değildir.
