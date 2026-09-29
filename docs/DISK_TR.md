# Disk standart modülü

[English](DISK.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Service](SERVICE_TR.md) · [File](FILESYSTEM_TR.md) · [Hatalar](ERRORS_TR.md)

`Disk` (v2.7.0), bir yolu içeren dosya sisteminin kapasitesini bildirir: ne
kadar büyük olduğunu, ne kadarının kullanıldığını ve ne kadarının hâlâ boş
olduğunu. Salt okunurdur, işletim sisteminin kendi dosya sistemi API'sini
kullanır ve asla bir kabuk ya da `df`, `diskutil`, `wmic` veya PowerShell gibi
bir araç çalıştırmaz. Açıkça içe aktarın:

```ahd
bring Disk
from Disk bring (DiskInfo, DiskError)
```

Kanonik modül kimliği `builtin:Disk`'tir; bir kardeş `Disk.ahd` onu
gölgeleyemez. (v2.7.0'dan itibaren `Disk` bir standart modül adıdır: yerel bir
`Disk.ahd` artık `bring Disk`'in yüklediği şey değildir — böyle bir dosyayı
yeniden adlandırın.)

## Yüzey

```text
Disk.inspect(path: String) -> DiskInfo

DiskInfo.path()           -> String   // programın verdiği haliyle yol
DiskInfo.totalBytes()     -> Int      // dosya sisteminin boyutu
DiskInfo.usedBytes()      -> Int      // totalBytes - freeBytes
DiskInfo.freeBytes()      -> Int      // ayrılmış alan dahil tüm boş baytlar
DiskInfo.availableBytes() -> Int      // bu programın kullanıcısının kullanabileceği boş baytlar
DiskInfo.usedPercent()    -> Real     // usedBytes / totalBytes * 100

DiskError  (Error'dan türer)
```

`DiskInfo` yalnızca `Disk.inspect` tarafından üretilen opak bir değerdir.

```ahd
disk: DiskInfo := Disk.inspect("/")
write(str(disk.usedPercent()) + "% used")
write(str(disk.availableBytes()) + " bytes available")
```

## Hangi dosya sistemi

`Disk.inspect(path)`, `path`'i **içeren** dosya sistemini inceler. Yolun bir
bağlama noktası olması gerekmez: `Disk.inspect("/var/www/site")` o dizini
barındıran dosya sistemini bildirir; bu `/` ya da ayrıca bağlanmış bir birim
olabilir. Yol mevcut olmalıdır. Göreli bir yol çalışma dizinine göredir.

## Sayılar

Tüm değerler bayttır, asla negatif değildir ve tek bir çağrıda okunur; bu
yüzden birbirleriyle tutarlıdır:

- `totalBytes` — dosya sisteminin boyutu.
- `freeBytes` — sistemin yöneticisi için ayırdığı alan **dahil** tüm boş alan
  (Linux ext4'te bu genellikle %5'tir).
- `availableBytes` — geçerli kullanıcının sıradan bir sürecinin gerçekten
  yazabileceği boş alan. Hiçbir zaman `freeBytes`'tan büyük değildir.
- `usedBytes` — `totalBytes - freeBytes`.
- `usedPercent` — `usedBytes / totalBytes * 100`; 0 ile 100 arasında
  (kapasite bildirmeyen bir dosya sistemi için 0).

`usedBytes` ve `usedPercent` ayrılmış alanı boş sayar; bu yüzden `usedPercent`,
`used + available`'a bölen `df`'in "Use%" sütunundan biraz düşük olabilir.
"Bu kullanıcı hâlâ yazabilir mi?" kararları için `availableBytes` kullanın.

## Platform davranışı

| Platform | Kaynak | Notlar |
|---|---|---|
| Linux | `statfs` | blok sayıları temel blok boyutunda; `df -B1` ile bayt bayt aynı |
| macOS | `statfs` | APFS'te bir kaptaki birimler alanı paylaşır; birkaç birim aynı boş alanı bildirebilir |
| Windows | `GetDiskFreeSpaceExW` | `availableBytes` kullanıcının disk kotasını yansıtır; `freeBytes` tüm birimin boş alanıdır |

Bir `Int`'e sığmayacak bir değer, taşmak yerine bir `DiskError` ile reddedilir.

## Hatalar

Her hata, iletisi `inspect disk of "<yol>" failed: <neden>` olan bir
`DiskError` fırlatır:

```text
inspect disk of "" failed: the path is empty
inspect disk of "/missing" failed: no such file or directory
inspect disk of "/root/private" failed: permission denied
```

```ahd
attempt {
    disk: Local DiskInfo := Disk.inspect("/srv/data")
    if disk.usedPercent() > 90.0 {
        write("warning: /srv/data is " + str(disk.usedPercent()) + "% full")
    }
}
except DiskError as failure {
    write(failure.message)
}
```

"%90'ın üzerinde uyar" gibi eşikler uygulamanın politikasıdır; modül yalnızca
olguları bildirir.

## Güvenlik: güvenilmeyen girdi için yalnızca işçi

`Disk.inspect`, kendisine verilen mevcut her yolun boyutunu ve doluluğunu
açığa çıkarır. Ona güvenilmeyen web girdisi vermeyin. Bir kontrol panelinde onu
doğrulanmış, izin listesindeki yolları inceleyen özel bir işçide tutun:

```text
web uygulaması
    -> kimliği doğrulanmış kontrol kanalı
        -> özel bir AhdCode işçisi
            -> doğrulanmış yol
                -> Disk.inspect
```

## Bu sürümde yok

Bağlama noktalarını ya da birimleri listeleme, inode sayıları, dizin başına
kullanım (`du`), G/Ç istatistikleri, kota yönetimi ve disklerde her türlü
değişiklik `Disk`'in parçası değildir.
