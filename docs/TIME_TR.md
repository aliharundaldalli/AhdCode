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
parseISO(text: String) -> DateTime
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

Dönüşümler aynı anı korur:

```text
value.timestamp() -> Int
value.toUTC() -> DateTime
value.toLocal() -> DateTime
value.toOffset(offsetMinutes: Int) -> DateTime
```

`before`, `after`, `sameMoment` ve `Time.between`, görünen saat alanlarını
değil anları karşılaştırır; bu yüzden farklı ofsetli değerler doğru
karşılaştırılır.

## DateTime

Dokuz salt okunur `Int` öznitelik vardır: `year`, `month`, `day`, `hour`,
`minute`, `second`, `millisecond`, `weekday`, `offsetMinutes`. Hafta günleri
Pazartesi=1 ile Pazar=7 arasındadır. `offsetMinutes`, UTC'nin doğusundaki
ofsettir.

Üyeler `before`, `after`, `sameMoment`, `timestamp`, `toUTC`, `toLocal`,
`toOffset`, `toString`, `toISO`, `add` ve `subtract`'tir. Mevcut `toString()` çıktısı
`YYYY-MM-DD HH:MM:SS` olarak kalır; milisaniye veya ofset eklemez.
`str(value)` ve `write(value)`, `DateTime(` ardından `toISO()` metni ve `)`
gösterir (v1.8.0); bir Duration `Duration(1500 ms)` olarak
görünür.

`DateTime`, `CCompare` veya `CEqual` uygulamaz. Adlandırılmış an işlemlerini
kullanın; sıradan `==` ve `same`, Class kimliği anlamını korur.

## Doğrulama, Duration ve Calendar

Sivil kurucular; yıl 1..9999, Gregoryen tarih, saat 0..23, dakika/saniye
0..59 ve milisaniye 0..999 aralıklarını doğrular. `DateTime` ve `Duration`
doğrudan oluşturulamaz.

`Duration`, salt okunur `milliseconds: Int` ve `seconds: Real` sunar.
`between(first, second)`, `second - first` anlamına gelir.

```text
duration.add(other: Duration) -> Duration
duration.subtract(other: Duration) -> Duration
duration.negate() -> Duration
duration.abs() -> Duration
```

Duration aritmetiği tam sayı (Int) milisaniye aritmetiğidir. Int dışına çıkan
bir sonuç `OverflowError` fırlatır; en negatif Duration'ın `negate()` ve
`abs()` sonuçları da öyle. Duration yalnızca bir süre uzunluğudur: ay, yıl
veya takvim dönemi yoktur.

```text
Calendar.isLeapYear(year: Int) -> Bool
Calendar.daysInMonth(year: Int, month: Int) -> Int
Calendar.weekday(year: Int, month: Int, day: Int) -> Int
```

`monotonic()` gerilemeyen bir saatten geçen saniyeleri döndürür. `sleep`
milisaniye alır; negatif değer `ValueError` fırlatır.

## ISO 8601 metni

`Time.parseISO(text)`, RFC 3339'un tek ve katı bir alt kümesini okur;
`value.toISO()` onu yazar:

```text
YYYY-MM-DDTHH:MM:SSZ
YYYY-MM-DDTHH:MM:SS±HH:MM
YYYY-MM-DDTHH:MM:SS.fZ          (Z veya ±HH:MM öncesinde .f, .ff ya da .fff)
```

- `T` ayırıcısı ve saat dilimi göstergesi zorunludur: UTC için `Z` ya da
  -14:00..+14:00 aralığında bir `±HH:MM` ofseti. Harfler büyük, rakamlar
  ASCII'dir; değerin önünde veya arkasında başka bir şey olamaz.
- Kesir bir ile üç basamaktır ve milisaniye anlamına gelir: `.3` 300 ms,
  `.35` 350 ms'dir. Dört veya daha fazla basamak yuvarlanmaz, reddedilir.
- Diğer her şey `ValueError` fırlatır: saatsiz tarih, göstergesiz saat, `T`
  yerine boşluk, `Sep 18 2026`, `18/09/2026`, `UTC+3`, `Europe/Istanbul` ve
  `2026-02-30T00:00:00Z`, saat 24 ya da saniye 60 gibi imkânsız değerler.

Sonuç yazılan ofseti korur: `Time.parseISO("2026-09-18T13:30:00+03:00")`
değerinin `offsetMinutes` alanı 180'dir. `toISO()` her zaman üç milisaniye
basamağı, sıfır ofset için `Z`, diğerleri için `±HH:MM` yazar; böylece yazdığı
her değer aynı an olarak geri okunur:
`Time.parseISO(value.toISO()).sameMoment(value)` doğrudur.

```ahd
bring Time

meeting := Time.parseISO("2026-09-18T13:30:00.5+03:00")
write(meeting.toISO())
write(meeting.toUTC().toISO())
write(meeting.offsetMinutes)
attempt {
    Time.parseISO("2026-09-18 13:30")
} except ValueError as error {
    write("rejected")
}
```

```text
2026-09-18T13:30:00.500+03:00
2026-09-18T10:30:00.500Z
180
rejected
```

Saniye içeren tarihsel bir yerel ofsetin (yukarıya bakın) `±HH:MM` biçimi
yoktur; bu yüzden `toISO()` böyle bir değer için `ValueError` fırlatır. Böyle
bir anı `value.toUTC().toISO()` ile yazın.

## An aritmetiği

```text
value.add(duration: Duration) -> DateTime
value.subtract(duration: Duration) -> DateTime
```

`add` ve `subtract` anı tam milisaniye kadar kaydırır ve değerin ofsetini
korur: sabit ofset aynı kalır, UTC UTC kalır. Yerel bir değer sahip olduğu
ofseti korur; sonuca yaz saati kuralı uygulanmaz. 1..9999 yılları dışına
çıkan bir sonuç `ValueError` fırlatır. `Time.between(first, second)` yine
`second - first`'tür; AhdCode'da DateTime veya Duration için `+` ya da `-`
operatörü yoktur.

```ahd
bring Time

start := Time.parseISO("2026-09-18T23:30:00+03:00")
hour := Time.duration(3600000)
later := start.add(hour)
write(later.toISO())
write(later.subtract(duration: hour.add(hour)).toISO())
write(Time.between(start, later).milliseconds)
write(hour.negate().abs().milliseconds)
```

```text
2026-09-19T00:30:00.000+03:00
2026-09-18T22:30:00.000+03:00
3600000
3600000
```

## Kasıtlı sınır

Time, UTC ve sabit dakika ofsetleri sunar; saat dilimi veritabanı değildir.
Adlandırılmış/IANA bölgeleri, DST kuralları veya yapılandırma nesneleri,
esnek ya da doğal dil tarih ayrıştırıcıları, biçim dizeleri, yerelleştirilmiş
adlar, takvim dönemleri (ay veya yıl) ya da DateTime operatörleri yoktur.
`parseISO` yalnızca yukarıdaki katı RFC 3339 alt kümesini okur.
