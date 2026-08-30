# Time standart modülü

[English](TIME.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Tanılamalar](DIAGNOSTICS_TR.md)

`Time`, açık ve derleyiciye kayıtlı `builtin:Time` modülüdür. Kardeş bir
`Time.ahd` onun yerini alamaz. Class'ları adlandırmadan önce içe aktarın:

```ahd
bring Time
from Time bring DateTime
from Time bring Duration
```

## Yüzey

```text
now() -> DateTime
utc() -> DateTime
timestamp() -> Int
fromTimestamp(milliseconds: Int) -> DateTime
dateTime(year, month, day, hour = 0, minute = 0, second = 0, millisecond = 0) -> DateTime
dateTimeUTC(year, month, day, hour = 0, minute = 0, second = 0, millisecond = 0) -> DateTime
dateTimeOffset(year, month, day, offsetMinutes, hour = 0, minute = 0, second = 0, millisecond = 0) -> DateTime
monotonic() -> Real
sleep(milliseconds: Int) -> Nothing
duration(milliseconds: Int) -> Duration
between(first: DateTime, second: DateTime) -> Duration
```

`now` ana bilgisayarın yerel sivil saatini, `utc` UTC'yi kullanır. `dateTime`
yerel, `dateTimeUTC` UTC, `dateTimeOffset` ise tam dakika cinsinden sabit
ofsetli bir değer oluşturur. Desteklenen ofset aralığı -840..840'tır. Geçersiz
sivil bileşenler, ofsetler ve temsil edilemeyen timestamp'ler `ValueError`
fırlatır.

Herkese açık ofset modeli dakika hassasiyetindedir: `offsetMinutes` her zaman
tam dakikadır ve AhdCode kaynağının adlandırabildiği her ofset tam dakikadır.
Birkaç tarihsel host-yerel bölge, saniye içeren bir ofsette bulunur -- örneğin
`Europe/Istanbul` 1880 öncesinde `+01:55:52`'dir. Böyle bir an yine de tam
olarak temsil edilir: `offsetMinutes` tam dakika kısmını bildirir, artan
saniyeler ise kırpılmak yerine çalışma zamanı gösterimi olarak saklanır, bu
yüzden an asla kaymaz. Saniye artığı yayınlanan bir öznitelik değildir; bu
nedenle okunamaz ve `has` onu bildirmez.

## Unix milisaniyeleri ve dönüşümler

Timestamp, `1970-01-01 00:00:00 UTC` anından itibaren işaretli milisaniyedir.
`Time.timestamp()` güncel timestamp'i okur. `Time.fromTimestamp(value)` UTC
görünümünü döndürür; sonuç yılı 1..9999 içindeyse negatif değerler desteklenir.

```ahd
epoch: DateTime := Time.fromTimestamp(0)
turkey: DateTime := epoch.toOffset(180)

write(epoch.timestamp())
write(turkey.hour)
write(epoch.sameMoment(turkey))
```

Dönüşümler aynı anı korur: `timestamp()`, `toUTC()`, `toLocal()` ve
`toOffset(offsetMinutes)`. `before`, `after`, `sameMoment` ve `Time.between`,
görünen saat alanlarını değil anları karşılaştırır.

## DateTime

Dokuz salt okunur `Int` öznitelik vardır: `year`, `month`, `day`, `hour`,
`minute`, `second`, `millisecond`, `weekday`, `offsetMinutes`. Hafta günleri
Pazartesi=1 ile Pazar=7 arasındadır. `offsetMinutes`, UTC'nin doğusundaki
ofsettir.

Üyeler `before`, `after`, `sameMoment`, `timestamp`, `toUTC`, `toLocal`,
`toOffset` ve `toString`'dir. Mevcut `toString()` çıktısı
`YYYY-MM-DD HH:MM:SS` olarak kalır; milisaniye veya ofset eklemez.
`str(value)` sıradan Class gösterimi olan `<DateTime>`'ı korur.

`DateTime`, `CCompare` veya `CEqual` uygulamaz. Adlandırılmış an işlemlerini
kullanın; sıradan `==` ve `same`, Class kimliği anlamını korur.

## Doğrulama, Duration ve Calendar

Sivil kurucular; yıl 1..9999, Gregoryen tarih, saat 0..23, dakika/saniye
0..59 ve milisaniye 0..999 aralıklarını doğrular. `DateTime` ve `Duration`
doğrudan oluşturulamaz. `Duration`, salt okunur `milliseconds: Int` ve
`seconds: Real` sunar. `between(first, second)`, `second - first` anlamına gelir.

`Calendar.isLeapYear`, `Calendar.daysInMonth` ve `Calendar.weekday` Gregoryen
takvim sorularını yanıtlar. `monotonic()` gerilemeyen bir saatten geçen
saniyeleri döndürür. `sleep` milisaniye alır; negatif değer `ValueError`
fırlatır.

## Kasıtlı sınır

v0.1.11, UTC ve sabit dakika ofsetleri ekler; saat dilimi veritabanı eklemez.
Adlandırılmış/IANA bölgeleri, DST yapılandırma nesneleri, tarih ayrıştırıcıları,
ISO-8601/RFC 3339 okuyucuları, biçim dizeleri, yerelleştirilmiş adlar veya doğal
dil tarihleri yoktur.
