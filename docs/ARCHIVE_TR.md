# Archive standart modülü

[English](ARCHIVE.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [PDF](PDF_TR.md)

Archive, dosyaları gerçek ZIP, TAR ve TAR.GZ arşivlerine paketler ve v2.5.0
ile birlikte ZIP, TAR ve TAR.GZ arşivlerini listeler ve **güvenli biçimde
çıkarır** — çevrimdışı ve yalnızca Go standart kütüphanesiyle (`archive/zip`,
`archive/tar`, `compress/gzip`). Açıkça içe aktarın:

```ahd
bring Archive
from Archive bring (ArchiveEntry, ArchiveError)
```

Kanonik modül kimliği `builtin:Archive`'dır; bir kardeş `Archive.ahd` dosyası
onu gölgeleyemez.

Çıkarma **varsayılan olarak güvenlidir ve güvensiz bir biçimi yoktur**: yol
kaçışı, mutlak ve sürücü yolları, bağlantılar ve arşiv bombaları reddedilir;
başarısız bir çıkarma geride hiçbir şey bırakmaz. Bkz.
[Listeleme ve güvenli çıkarma](#listeleme-ve-güvenli-çıkarma-v250).

## Yüzey

```text
Archive.zip(output: String, entries: Pair<String, String>)     -> Nothing
Archive.tar(output: String, entries: Pair<String, String>)     -> Nothing
Archive.tarGzip(output: String, entries: Pair<String, String>) -> Nothing

Archive.list(archive: String, maxFiles: Int = 10000)            -> List<ArchiveEntry>
Archive.extract(archive: String, destination: String,
                maxFiles: Int = 10000, maxBytes: Int = 1073741824) -> Nothing

ArchiveEntry.path() -> String   // arşivde saklandığı haliyle ad
ArchiveEntry.kind() -> String   // "file", "directory", "symlink", "hardlink" veya "other"
ArchiveEntry.size() -> Int      // sıkıştırılmamış bayt; dosya olmayanlar için 0

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

## Listeleme ve güvenli çıkarma (v2.5.0)

`Archive.list` ve `Archive.extract`, ZIP, TAR ve TAR.GZ okur. Biçim, dosyanın
adından değil **içeriğinden** (imzasından) tanınır; gerçekte ZIP olan
`release.bin` çalışır, adı değiştirilmiş bir metin dosyası ise
`unsupported archive format` ile reddedilir.

### Listeleme

```ahd
entries: List<ArchiveEntry> := Archive.list("release.zip")
for entry in entries {
    write(entry.path() + " " + entry.kind() + " " + str(entry.size()))
}
```

`list` hiçbir şey yazmaz ve her girdiyi arşiv sırasıyla **saklandığı gibi**
bildirir — sembolik bağlantılar, sabit bağlantılar ve `../evil` gibi güvensiz
adlar dahil — böylece program karar vermeden önce yüklemeyi inceleyebilir.
`ArchiveEntry` opak bir değerdir: doğrudan kurulamaz, yalnızca `Archive.list`
ile elde edilir. En fazla `maxFiles` girdi kabul edilir.

### Çıkarma

```ahd
Archive.extract(
    archive: "uploads/release-42.zip",
    destination: "releases/42",
    maxFiles: 5000,
    maxBytes: 1073741824
)
```

Sözleşme:

- **Hedef mevcut olmamalıdır**; üst dizini mevcut olmalıdır. `extract`
  yalnızca yeni bir dizin oluşturur. Mevcut içeriğe asla karışmaz, üzerine
  yazmaz ya da onu silmez — aynı hedefe iki kez çıkarmak `ArchiveError`
  fırlatır.
- **Ya hep ya hiç.** Girdiler, hedefin yanındaki özel bir hazırlık dizinine
  (`.<ad>.ahdextract-…`) yazılır; bu dizin yalnızca her girdi başarılı
  olduktan sonra hedefe yeniden adlandırılır. Herhangi bir hatada bu çağrının
  oluşturduğu hazırlık dizini silinir; başarısız bir çıkarma, başarılı
  görünen yarım bir sürüm bırakmaz.
- **Her girdi yolu, hiçbir şey yazılmadan önce doğrulanır.** Reddedilenler:
  boş adlar, NUL baytları, mutlak yollar (`/etc/x`), UNC yolları
  (`//server/share`), sürücü yolları (`C:/x`, `C:x`), her ters eğik çizgi
  (böylece `..\evil` ve karışık `a/..\..\x` geçemez), her iki nokta üst üste
  ve her `..` parçası (`../x`, `a/../../x`). `.` parçaları ve baştaki `./`
  zararsızdır ve kabul edilir; tar araçları bunları sıkça yazar. Birleştirmeden
  sonra hedef, hazırlık dizinine göre gerçek bir göreli yol hesabıyla yeniden
  denetlenir — asla metin öneki karşılaştırmasıyla değil; böylece
  `releases/42-evil` gibi bir kardeş dizin `releases/42`'nin "içinde"
  sayılamaz. Windows'ta nokta ya da boşlukla biten veya ayrılmış bir aygıt
  adı (`CON`, `NUL`, …) taşıyan adlar da reddedilir.
- **Bağlantılar reddedilir; asla izlenmez ya da dönüştürülmez.** Sembolik
  bağlantı ve sabit bağlantı girdileri ile aygıt, FIFO ve seyrek (sparse)
  girdiler tüm çıkarmayı başarısız kılar. `allowSymlinks` seçeneği ve güvensiz
  bir biçim yoktur. Çıkarılan hiçbir şey bağlantı olamayacağı için hiçbir
  girdi sonraki bir yazmayı hedefin dışına yönlendiremez; ayrıca bir dosya
  oluşturulmadan önce her üst dizin çözülüp yeniden denetlenir.
- **Sınırlıdır.** `maxFiles` girdi (dosya ve dizin) sayısını, `maxBytes`
  toplam sıkıştırılmamış boyutu sınırlar. İkisi de önce arşivin üst
  verisine göre denetlenir — saçma bir bildirilmiş boyut, tek bayt
  yazılmadan reddedilir — ve ardından akış sırasında yeniden denetlenir,
  çünkü üst veri yalan söyleyebilir: bildirdiğinden fazla bayt üreten bir
  girdi ya da payını aşan bir TAR.GZ akışı çıkarmayı başarısız kılar.
  Boyut toplamı taşmaya karşı güvenlidir.
- Dosyalar `0644` kipiyle, arşiv girdiyi çalıştırılabilir olarak işaretlemişse
  `0755` ile oluşturulur; dizinler `0755` alır (ikisi de süreç umask'ı
  uygulanmadan önce). setuid, setgid ve sticky bitleri asla uygulanmaz,
  arşivdeki sahip bilgileri yok sayılır. Sonrasında `File.setPermissions` ile
  ayarlayın.
- Bir TAR PAX genel başlığı (`git archive`'in yazdığı `pax_global_header`
  kaydı) bir girdi değil, arşiv üst verisidir: ne listelenir ne yazılır.
- Şifreli ZIP girdileri reddedilir. Yinelenen dosya girdileri (büyük/küçük
  harfe duyarsız bir dosya sisteminde çakışan adlar dahil) reddedilir;
  yinelenen bir dizin girdisi zararsızdır.
- Bir ZIP'i okumak, merkezi dizinini (girdi dizinini) arşivin boyutuyla
  orantılı olarak belleğe yükler. Güvenilmeyen yüklemelerin boyutunu,
  listelemeden ya da çıkarmadan önce sınırlayın.

| Sınır | Varsayılan | İzin verilen aralık |
|---|---|---|
| `maxFiles` | 10 000 | 1 – 1 000 000 |
| `maxBytes` | 1 073 741 824 (1 GiB) | 1 – 68 719 476 736 (64 GiB) |

Aralık dışındaki bir değer `ArchiveError` fırlatır; çıkarma asla sınırsız
değildir.

### Hatalar

Her hata, iletisi işlemi ve arşivi, ardından tek bir kararlı nedeni adlandıran
bir `ArchiveError`'dır; örneğin:

```text
extract "evil.zip" failed: unsafe entry path "../escaped.txt": the path escapes the destination
extract "links.tar.gz" failed: entry "current" is a symbolic link; safe extraction rejects links
extract "big.zip" failed: file-count limit exceeded: the archive has more than 5000 entries (maxFiles)
extract "bomb.zip" failed: byte limit exceeded: the archive declares more than 1073741824 uncompressed bytes (maxBytes)
extract "notes.txt" failed: unsupported archive format; expected ZIP, TAR, or TAR.GZ
extract "broken.tar.gz" failed: malformed archive: invalid header
extract "release.zip" failed: destination "releases/42" already exists; Archive.extract only creates a new directory
```

Ham Go hata metni asla gösterilmez.

## Bu sürümde yok

RAR, 7z, BZIP2, XZ, bağımsız bir Compress modülü, şifreli ya da parola
korumalı arşivler, rastgele erişimli bir arşiv nesne modeli, bağlantıları ya
da sahipliği koruyan çıkarma, güvensiz bir çıkarma kipi ve oluşturma için
dizin kaynağı özyinelemesi AhdCode'un parçası değildir.
