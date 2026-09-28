# File ve Path modülleri

[English](FILESYSTEM.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Hatalar](ERRORS_TR.md)

Modülleri açıkça içe aktarın:

```ahd
bring Path
bring File
from File bring (FileEntry, FileError)
```

## Path

`Path`, ana bilgisayar işletim sistemine duyarlı (host-operating-system-
aware), saf yol işlemleri gerçekleştirir:

```text
Path.join(parts: List<String>) -> String
Path.ext(path: String)         -> String
Path.base(path: String)        -> String
Path.dir(path: String)         -> String
```

```ahd
filePath := Path.join(["reports", "result.txt"])
write(Path.ext(filePath))
write(Path.base(filePath))
write(Path.dir(filePath))
```

## File

```text
File.exists(path: String)                     -> Bool
File.readText(path: String)                   -> String
File.writeText(path: String, content: String) -> Nothing
File.append(path: String, content: String)    -> Nothing
File.delete(path: String)                     -> Nothing
File.createDir(path: String)                  -> Nothing
File.list(path: String)                       -> List<String>
```

Metin UTF-8'dir. `File.list`, doğrudan girdi adlarını kararlı, artan
sözlüksel (ascending lexical) sırada döndürür; özyinelemeli (recursive)
değildir. Göreli yollar, süreç çalışma dizinini (process working directory)
kullanır. Bir REPL'de bu, `ahdcode`'un başlatıldığı dizindir.

```ahd
File.createDir("notes")
File.writeText("notes/today.txt", "first")
File.append("notes/today.txt", " second")
write(File.readText("notes/today.txt"))
write(File.list("notes"))
```

`File.exists`, eksik bir yol için `false` döndürür. Diğer File işlemlerinin
hataları, `IOError` ve `Error`'dan türeyen `FileError`'ı fırlatır:

```ahd
attempt {
    File.readText("missing.txt")
}
except FileError as error {
    write(error.message)
}
```

File işlemleri hiçbir zaman ana bilgisayar (host) hata nesnesini göstermez.
File'ın genel bir OS modülü, sahiplik (`chown`) API'si, dosya izleme ve genel
ikili okuma/yazma işlevi yoktur; aşağıdaki dağıtım temel işlemleri bilinçli
olarak dar tutulmuştur.

## Dağıtım temel işlemleri (v2.5.0)

```text
File.copy(source: String, destination: String)        -> Nothing
File.atomicWrite(path: String, content: String)       -> Nothing
File.atomicMove(source: String, destination: String)  -> Nothing
File.symlink(target: String, link: String)            -> Nothing
File.readLink(path: String)                           -> String
File.isSymlink(path: String)                          -> Bool
File.permissions(path: String)                        -> String   // "0755"
File.setPermissions(path: String, mode: String)       -> Nothing  // "0755" ya da "755"
File.walk(path: String, maxEntries: Int = 100000)     -> List<FileEntry>

FileEntry.path()         -> String   // kök ile göreli yolun birleşimi
FileEntry.relativePath() -> String   // köke göre, eğik çizgilerle
FileEntry.kind()         -> String   // "file", "directory", "symlink" ya da "other"
FileEntry.size()         -> Int      // dosyalar için bayt; diğerleri için 0
FileEntry.isSymlink()    -> Bool
```

Her hata; işlemi, programın yazdığı haliyle yol(lar)ı ve tek bir sade nedeni
(`no such file or directory`, `permission denied`, `the path already exists`,
…) içeren bir `FileError` fırlatır.

### `File.copy` — ikili güvenli kopyalama

`copy`, bir **normal dosyayı** bayt bayt çoğaltır. Sabit 64 KiB'lık bir
arabellekle akış halinde çalışır; dosyayı belleğe almaz ve asla `String`
içinden geçmez: ikili dosyalar, görseller ve arşivler birebir kopyalanır.

- Kaynak normal bir dosya olmalıdır. Sembolik bağlantı kaynağı **izlenmez,
  reddedilir**; bir dizin reddedilir.
- Hedef mevcut **olmamalıdır**: `copy` asla üzerine yazmaz. Bir dosyayı
  bilinçli olarak değiştirmek için `File.atomicMove` kullanın.
- Kopya, hedefin yanındaki geçici bir dosyaya yazılır, diske aktarılır ve
  ancak ondan sonra son adını alır; bir hata asla yarım bir hedef bırakmaz.
  Unix'te hedef, kaynağın izin bitlerini alır.

### `File.atomicWrite` — bir dosyayı tek adımda değiştirme

`atomicWrite`, UTF-8 metni `path` ile **aynı dizindeki** (dolayısıyla aynı
dosya sistemindeki) geçici bir dosyaya yazar, kalıcı depolamaya aktarır ve
`path` üzerine yeniden adlandırır. Okuyucular ya eski tam dosyayı ya da yeni
tam dosyayı görür, asla bir karışımı değil. Her hatada geçici dosya silinir.

- Mevcut bir dosya izin bitlerini korur; yeni bir dosya Unix'te `0644` ile
  oluşturulur.
- `path` konumundaki bir sembolik bağlantı yeni dosyayla değiştirilir
  (hedefi değil, bağlantının kendisi). `path` bir dizinse reddedilir.
- Unix'te `rename`, POSIX gereği atomiktir ve ardından dizin eşitlenir.
  Windows'ta değiştirme, var olanın yerine geçen `MoveFileEx` ile yapılır; bu,
  Windows'un sunduğu en güçlü adımdır ama her hata durumunda atomik olduğu
  belgelenmemiştir. AhdCode, platformun sağladığından fazlasını iddia etmez.

JSON ve süslü parantez içeren diğer metinleri, parantezler enterpolasyon
sayılmasın diye ham String olarak yazın:
`File.atomicWrite("config.json", r'{"port": 8080}')`.

### `File.atomicMove` — yeniden adlandır, asla kopyalama

`atomicMove`, işletim sisteminin tek bir `rename` işlemidir. "Hazırlık →
canlı" geçişinin temel işlemidir:

- Aynı dosya sistemi: taşıma atomiktir. `destination` konumundaki mevcut bir
  **dosya ya da sembolik bağlantı** atomik olarak değiştirilir.
- `destination` konumundaki mevcut bir **dizin** — boş olsa bile — her zaman
  reddedilir; böylece sonuç, platformun boş dizin kurallarına bağlı olmaz.
- Farklı dosya sistemleri:
  `source and destination are on different filesystems; File.atomicMove never copies`
  iletisiyle `FileError`. Gizli bir kopyala-ve-sil yoktur.

### Sembolik bağlantılar

```ahd
File.symlink(target: "releases/42", link: "current")
write(File.readLink("current"))      // releases/42
write(str(File.isSymlink("current"))) // true
```

**Argüman sırası önce `target`, sonra `link`'tir** — `ln -s target link` ile
aynı sıra: `link` oluşturulan yeni yol, `target` onun gösterdiği metindir.
Yukarıdaki gibi parametre adlarını kullanmak her çağrıyı kendiliğinden
açıklar. Hedef olduğu gibi saklanır; göreli bir hedefi işletim sistemi
bağlantının kendi dizinine göre çözer ve hedefin var olması gerekmez.
Bağlantı yolu henüz mevcut olmamalıdır.

`readLink`, saklanan hedefi çözmeden döndürür ve bağlantı olmayan bir yolda
başarısız olur. `isSymlink` yolun kendisine bakar (asla izlemez) ve
`File.exists` gibi eksik bir yol için `false` döndürür. `File.delete` bir
bağlantının hedefini değil, kendisini siler.

Dağıtım geçiş kalıbı, yeni bağlantıyı canlı olanın yanında oluşturur ve tek
bir atomik adımla onun üzerine yeniden adlandırır:

```ahd
File.symlink(target: "releases/43", link: "current.next")
File.atomicMove(source: "current.next", destination: "current")
```

**Windows**'ta sembolik bağlantı oluşturmak Geliştirici Modu'nu ya da
`SeCreateSymbolicLinkPrivilege` yetkisini gerektirir; bu yoksa `File.symlink`
bunu söyleyen bir `FileError` fırlatır. Asla kopyalamaya geri dönmez.

### İzinler — sekizlik (octal) bir String

AhdCode'da sekizlik sayı sabitleri yoktur ve burada ondalık bir `Int`
tehlikelidir: Int olarak yazılan `0444`, ondalık 444 sayısı, yani `0674`
kipi olurdu. Bu yüzden dosya izinleri **sekizlik rakamlardan oluşan bir
String** kullanır:

```ahd
File.setPermissions("releases/42/app/bin/server", "0750")
write(File.permissions("releases/42/app/bin/server"))   // 0750
```

- Kabul edilen: isteğe bağlı olarak tek bir `0` ile başlayan tam üç sekizlik
  rakam (`"755"`, `"0755"`, `"0640"`, `"0000"`).
- `FileError` ile reddedilen: `"0o755"`, `"0x1ed"`, `"493"`, `"7777"`,
  `"4755"`, `"00755"`, `"rwxr-xr-x"`, boş metin ve diğer her şey. setuid,
  setgid ve sticky bitleri ayarlanamaz.
- `permissions` her zaman dört rakam döndürür (`"0644"`).
- Her iki işlev de sembolik bağlantıları izlemek yerine reddeder; böylece bir
  bağlantı bir izin değişikliğini asla başka bir dosyaya yönlendiremez.
- **Windows:** iki işlev de `FileError` fırlatır
  (`Unix permission bits are not supported on Windows`). Windows erişim
  denetim listeleri kip bitleri olarak dürüstçe gösterilemez; AhdCode orada
  `chmod` varmış gibi davranmaz.

### `File.walk` — bağlantıları asla izlemeyen özyinelemeli listeleme

```ahd
entries: List<FileEntry> := File.walk("releases/42")
for entry in entries {
    write(entry.relativePath() + " " + entry.kind() + " " + str(entry.size()))
}
```

- Kökün altındaki her girdiyi (kökün kendisi hariç) belirlenimci sözlüksel
  sırada, önce derinlik olarak listeler.
- **Sembolik bağlantılar asla izlenmez.** Bir bağlantı `"symlink"` türünde
  tek bir girdi olarak bildirilir ve bir dosyayı, dizini, kökün dışını ya da
  ağacın yukarısını göstersin, içine girilmez. Bu yüzden bağlantı döngüleri
  gezinmeyi sonsuz döngüye sokamaz. Kendisi bir bağlantı olan kök reddedilir
  (kastettiğiniz buysa `File.readLink` ile okuyup hedefi açıkça gezin).
- Sonuç bellekte oluşturulduğu için sınırlıdır: `maxEntries`'ten fazla girdi
  `FileError` fırlatır. Varsayılan 100 000'dir; izin verilen aralık
  1 – 1 000 000'dir.
- Bağlama noktaları algılanmaz; gezinme bağlanmış bir dizinin içine devam eder.

## Ayrıcalıklı programlar için güvenlik notları

Bu temel işlemler, yükseltilmiş haklarla çalışabilen dağıtım işçileri için
tasarlanmıştır: özgün davranışlarını koruyan `File.exists`, `readText`,
`writeText` ve `append` dışında hiçbir şey sembolik bağlantıyı örtük olarak
izlemez; `copy`, `permissions`, `setPermissions` ve `walk` bağlantıları
reddeder. Yollar yine önce denetlenip sonra kullanılır; üzerinde çalıştığınız
dizinleri eşzamanlı olarak yeniden yazabilen bir saldırgan her dosya sistemi
API'siyle yarışabilir — yalnızca işçinizin yazabildiği dizinlerde çalışın.
