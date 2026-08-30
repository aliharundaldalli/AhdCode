# Time standart modülü

[English](TIME.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Math modülü](MATH_TR.md)

Time, Math gibi açıktır (explicit). AhdCode'da isim uzayı nitelikli
(namespace-qualified) tür sözdizimi olmadığından, bir tür isimlendirilmeden
önce içe aktarılır:

```ahd
bring Time
from Time bring DateTime
from Time bring Duration
```

Kanonik kimlik `builtin:Time`'dır; kardeş bir `Time.ahd` onun yerini
alamaz (shadow edemez). Her argüman `NonNull` olmalıdır.

## Yüzey (Surface)

```text
now() -> DateTime
monotonic() -> Real
sleep(milliseconds: Int) -> Nothing
duration(milliseconds: Int) -> Duration
between(first: DateTime, second: DateTime) -> Duration
dateTime(year, month, day, hour = 0, minute = 0, second = 0, millisecond = 0) -> DateTime

Calendar.isLeapYear(year: Int) -> Bool
Calendar.daysInMonth(year: Int, month: Int) -> Int
Calendar.weekday(year: Int, month: Int, day: Int) -> Int
```

## Yalnızca yerel saat

`now`, ana bilgisayarın (host) yerel saatini bildirir ve `dateTime`, yerel
bir sivil (civil) anı oluşturur. v0.1'de UTC dönüşümü, saat dilimi
nesneleri veya isimleri, sabit ofsetler, DST yapılandırması veya saat
dilimi ayrıştırma/biçimlendirme yoktur.

## DateTime

Sekiz salt okunur (read-only) `Int` özniteliği (attribute): `year`,
`month`, `day`, `hour`, `minute`, `second`, `millisecond`, `weekday`.
Haftanın günleri Pazartesi=1'den Pazar=7'ye kadar gider.

Her öznitelik `Constant`'tır, bu yüzden `value.year = 2030`, Time'a özgü
bir kural değil, sıradan Constant tanılamasıdır (diagnostic).

Üyeler: `before`, `after`, `sameMoment` ve `toString`. `toString`
deterministiktir ve yerelden bağımsızdır (locale-independent),
`YYYY-MM-DD HH:MM:SS` biçimindedir; milisaniyeler `millisecond` özniteliği
üzerinden okunur. `str(value)`, `<DateTime>` olarak işlenir, çünkü Class
öznitelikleri hiçbir zaman otomatik olarak yazdırılmaz.

`DateTime`, `CCompare`/`CEqual` [Class Protocol Method](PROTOCOLS_TR.md)'larını
uygulamaz, bu yüzden sıralama `<`/`>` yerine `before`/`after`'dır. `==` ve
`same`, sıradan Class kimlik kuralını korur, bu yüzden ayrı ayrı
oluşturulmuş eşit anlar `==` değildir; `sameMoment` değer karşılaştırmasıdır.

## Doğrulama (Validation)

`dateTime`, `year` için 1..9999, `month` için 1..12, `day` için o yılın o
ayına göre, `hour` için 0..23, `minute` için 0..59, `second` için 0..59 ve
`millisecond` için 0..999 kontrol eder. İmkânsız bir an, sarmalamak
(rolling over) yerine yakalanabilir `ValueError` fırlatır, bu yüzden
`2026-02-29` ve `2026-02-30` ikisi de reddedilirken `2028-02-29` geçerlidir.

`DateTime` ve `Duration` hiçbir zaman doğrudan oluşturulmaz; yalnızca önce
doğrulama yapan Time fonksiyonlarından gelirler.

## Duration

Salt okunur `milliseconds: Int` ve `seconds: Real`. Bir Duration negatif
olabilir ve işaret, bir büyüklüğe (magnitude) indirgenmek yerine korunur.

`between(first, second)`, `second - first`'tir, bu yüzden argümanları
tersine çevirmek negatif bir Duration verir ve aynı an iki kez sıfır verir.

## Calendar

`Calendar`, bir DateTime olmadan Gregoryen takvim hakkındaki soruları
yanıtlar. Artık yıl (leap year), 4'e bölünür, ancak 400'e bölünmesi gereken
bir yüzyıl yılı (century year) hariç: 2028 ve 2000 artık yıllardır, 2100 ve
1900 değildir. Geçersiz bir yıl, ay veya tarih `ValueError` fırlatır. v0.1'de
ay veya gün isimleri, yerelleştirme (localization) veya takvim işleme
yoktur.

## Geçen zaman ve bekleme

`monotonic`, asla geriye gitmeyen bir saat üzerinde **saniye** bildirir.
Yalnızca farklar (differences) anlamlıdır; mutlak değerin takvimsel bir
anlamı yoktur.

`sleep`, **milisaniye** alır. Sıfır hemen döner ve negatif bir istek,
sıkıştırılmak (clamped) yerine `ValueError` fırlatır.

```ahd
start: Real := Time.monotonic()
Time.sleep(100)
elapsed: Real := Time.monotonic() - start
```

## Bu sürümde olmayanlar

Biçim dizeleri (format strings) yok, `parse` yok, ISO-8601 veya RFC 3339
okuyucusu yok, ay veya gün isimleri yok ve doğal dil tarihleri yok.
