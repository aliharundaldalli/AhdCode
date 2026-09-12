# Cron standart modülü

[English](CRON.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Time](TIME_TR.md) · [Web](WEB_TR.md)

`Cron`, AhdCode v1.2.0 ile gelen, derleyiciye kayıtlı `builtin:Cron`
modülüdür. AhdCode Function'larını, onları zamanlayan program çalıştığı sürece
klasik beş alanlı zamanlamalarla çalıştırır. Kardeş bir `Cron.ahd` onun yerini
alamaz.

```ahd
bring Cron
from Cron bring (Scheduler, CronError)
```

## Cron nedir, ne değildir

Cron, tek bir AhdCode süreci içinde uygulama düzeyinde zamanlamadır.

```text
AhdCode Cron  !=  işletim sisteminin crontab'ı
AhdCode Cron  !=  launchd
AhdCode Cron  !=  bir systemd timer
AhdCode Cron  !=  Windows Görev Zamanlayıcı (Task Scheduler)
AhdCode Cron  !=  kalıcı bir iş kuyruğu veya worker kümesi
```

- İşler yalnızca programları çalışırken çalışır. Süreç durduğunda — işini
  bitirdiğinde, hata verdiğinde, kesildiğinde veya makine yeniden başladığında —
  işleri de durur. Hiçbir şey saklanmaz ve yeniden başlatmadan sonra hiçbir şey
  kaldığı yerden devam etmez.
- Cron hiçbir sistem zamanlayıcı yapılandırmasını kurmaz, düzenlemez veya okumaz;
  hiçbir daemon ya da servis başlatmaz ve ağ uç noktası açmaz.
- Hiçbir AhdCode programı çalışmıyorken yapılması gereken iş, işletim sisteminin
  kendi zamanlayıcısına aittir; o zamanlayıcı sıradan bir AhdCode programını
  başlatabilir.

## Yüzey

```text
Cron.scheduler()                                        -> Scheduler
Cron.next(expression: String, after: DateTime)          -> DateTime

Scheduler.add(expression: String, task: () -> Nothing)  -> Nothing
Scheduler.run()                                         -> Nothing
Scheduler.stop()                                        -> Nothing

CronError
```

`DateTime`, [Time](TIME_TR.md) modülünün Class'ıdır. Bir `Scheduler` yalnızca
`Cron.scheduler()`'dan gelir; doğrudan oluşturmak derleme zamanı hatasıdır.

## Zamanlama sözdizimi

Bir zamanlama, boşluk veya sekmeyle ayrılmış tam olarak beş alandan oluşur:

```text
dakika  saat  ayın-günü  ay     haftanın-günü
0-59    0-23  1-31       1-12   0-7 (0 ve 7 Pazar'dır)
```

Her alan, virgülle ayrılmış öğelerden oluşan bir listedir:

| Öğe | Anlamı | Örnek |
|---|---|---|
| `*` | her değer | `*` |
| `n` | tek değer | `30` |
| `a-b` | `a` ≤ `b` olan kapsayıcı aralık | `9-17` |
| `*/s` | alanın her `s`'inci değeri | `*/15` |
| `a-b/s` | aralık içindeki her `s`'inci değer | `0-30/10` |
| `x,y` | yukarıdaki öğelerin listesi | `0,30` |

Sayılar düz ondalık rakamlardır. Adım en az `1`'dir ve üzerinden geçtiği
aralıktan geniş olamaz. Baştaki ve sondaki boşluk veya sekmeler yok sayılır.

```text
* * * * *        her dakika
*/15 * * * *     her 15 dakikada bir
0 9 * * 1-5      hafta içi 09:00'da
30 2 1 * *       her ayın ilk günü 02:30'da
0 0,12 * * *     gece yarısı ve öğlen
0 0 29 2 *       29 Şubat gece yarısı
```

**Ayın günü ve haftanın günü.** İki gün alanı da kısıtlıysa — hiçbiri `*` ile
başlamıyorsa — alanlardan **herhangi biri** eşleşen gün eşleşir:
`0 0 1,15 * 1` ayın 1'inde, 15'inde ve her Pazartesi çalışır. Gün alanlarından
biri `*` ile başlıyorsa gün **ikisine birden** uymalıdır: `0 0 */10 * 1`, aynı
zamanda Pazartesi olan bir 1'inde, 11'inde, 21'inde veya 31'inde çalışır. Bu,
geleneksel cron kuralıdır.

**Desteklenmeyenler.** Ay veya gün adları (`JAN`, `MON`), `@daily` veya
`@reboot` gibi takma adlar, saniye alanı, `?`, `L`, `W`, `#` ve ifade içinde saat
dilimi. Her biri reddedilir; hiçbiri başka türlü yorumlanmaz.

**Hiçbir şey zamanlanmadan önce doğrulanır.** `Scheduler.add` ve `Cron.next`
önce zamanlamanın tamamını denetler. Hatalı bir zamanlama `CronError` fırlatır ve
asla kaydedilmez; bu yüzden sessizce başka bir zamanlamaya dönüşemez, hemen
çalışamaz veya hiç çalışmadan kalamaz. Ayın günü değerleri aylarının hiçbirinde
bulunmayan `0 0 30 2 *` gibi bir zamanlama, asla çalışamayacağı için reddedilir.

## Cron.next

`Cron.next(expression, after)`, `after`'dan kesin olarak sonraki ilk eşleşen
dakikayı döndürür. Zamanlayıcının kendi kullandığı hesaptır; bu yüzden bir
zamanlamayı doğrulamanın veya bir işin ne zaman çalışacağını göstermenin kolay
yoludur.

```ahd
bring Cron
bring Time

after := Time.dateTimeUTC(2026, 9, 12, 10, 15, 30)
write(Cron.next("*/15 * * * *", after).toString())
write(Cron.next("0 9 * * 1-5", after).toString())
```

=>

```text
2026-09-12 10:30:00
2026-09-14 09:00:00
```

`next`, zamanlamayı `after`'ın taşıdığı sabit UTC ofsetinde değerlendirir ve
sonuç aynı ofseti taşır: `Time.utc()` UTC'yi, `Time.dateTimeOffset` sabit bir
ofseti, `Time.now()` ise ana bilgisayarın o andaki ofsetini verir. Bir
`DateTime`'ın kendine ait yaz saati kuralları yoktur.

## Saat dilimi ve yaz saati

`Scheduler.run`, zamanlamaları **ana bilgisayarın yerel saat diliminde** —
`Time.now()`'un okuduğu saatte —, o dilimin yaz saati kurallarıyla birlikte
değerlendirir. Zamanlama başına saat dilimi yoktur. UTC'ye göre zamanlamak için
programı yerel saat diliminin UTC olduğu yerde çalıştırın.

Her gerçekleşme gerçek bir andır:

- Yaz saati değişiminin **atladığı** yerel saat gerçekleşmez. Saatin 02:00'dan
  03:00'e atladığı yerde `30 2 * * *` o gece çalışmaz.
- Bir değişimin **tekrarladığı** yerel saat bir saat arayla iki kez gerçekleşir.
  Saatin 02:00'dan 01:00'e geri alındığı yerde `30 1 * * *` iki 01:30'da da
  çalışır.

Türkiye'ninki gibi yaz saati uygulamayan sabit ofsetli bir dilim bu durumların
hiçbiriyle karşılaşmaz.

## Görevler

Bir görev, hiçbir şey almayan ve `Nothing` döndüren, adıyla bildirilip `add`'e
verilen bir Function'dır. Biçimi derleme zamanında denetlenir: parametresi veya
sonucu olan bir Function, sıradan `SEM004` tür uyuşmazlığıdır.

```ahd
bring Cron
from Cron bring Scheduler

jobs: Scheduler := Cron.scheduler()
tick: Function := () -> Nothing {
    write("tick")
}
jobs.add("* * * * *", tick)
```

Bir görev program durumuna, diğer her Function gibi açık bir `Global`
bağımlılığıyla ulaşır. Bir görevin kendi Scheduler'ını durdurması da böyledir:

```ahd
bring Cron
from Cron bring Scheduler

jobs: Scheduler := Cron.scheduler()
count: Int := 0

once: Function := () -> Nothing {
    jobs: Global Scheduler
    count: Global Int
    count += 1
    jobs.stop()
}
jobs.add("* * * * *", once)
```

## Yaşam döngüsü

**add** zamanlamayı doğrular ve işi ekler. Birden çok iş, her biri kendi
zamanlamasıyla, tek bir Scheduler'ı paylaşabilir. Scheduler çalışırken ona iş
eklenemez.

**run** programı bloklar ve işleri çalıştırır:

- İşler, programın kendi akışında birer birer çalışır. Cron arka plan iş
  parçacığı başlatmaz ve hiçbir şeyi paralel çalıştırmaz.
- Aynı dakikada birden çok iş vadesi geldiğinde her biri, eklendiği sırayla bir
  kez çalışır.
- Vadesi gelen bir gerçekleşme, bir çalıştırma demektir. Bir görev bittiğinde bir
  sonraki gerçekleşmesi o andan itibaren hesaplanır; bu yüzden görev hâlâ
  çalışırken geçen bir gerçekleşme kuyruğa alınmaz, atlanır.
- Bilgisayar uyursa veya program birkaç gerçekleşme boyunca duraklatılırsa,
  etkilenen her iş program devam ettiğinde bir kez çalışır ve ardından bir
  sonraki gelecek gerçekleşmesiyle sürer; kaçırılan gerçekleşmeler tekrar
  oynatılmaz. Geriye alınan bir duvar saati, hiçbir gerçekleşmeyi ikinci kez
  çalıştırmaz.
- Gerçekleşmeler arasında program sıradan bir zamanlayıcı üzerinde bekler,
  yoklama (polling) yapmaz. Bekleyen `write` çıktısı her beklemeden önce boşaltılır
  (flush); böylece bir görevin yazdıkları, program çalışmayı sürdürürken terminale
  veya yönlendirilmiş bir günlük dosyasına ulaşır.
- Scheduler'da hiç iş yoksa veya zaten çalışıyorsa `run` `CronError` fırlatır.

**stop**, çalışan bir Scheduler'dan geri dönmesini ister. O anki görev döndüğünde
etkili olur; o dakikada hâlâ vadesi gelmiş işler çalışmaz. Çalışmayan bir
Scheduler'da `stop` çağırmak veya onu yeniden çağırmak hiçbir şey yapmaz. `run`
döndükten sonra Scheduler yeniden çalıştırılabilir.

**Başarısız olan bir görev.** Bir görevin fırlattığı hata, kendi Class'ı ve
mesajıyla değişmeden `run`'dan dışarı yayılır ve çalıştırmayı bitirir; o dakikada
hâlâ vadesi gelmiş işler çalışmaz. Diğer her hata gibi ele alın:

```ahd
bring Cron
from Cron bring Scheduler

jobs: Scheduler := Cron.scheduler()
report: Function := () -> Nothing {
    toss(ValueError("report failed"))
}
jobs.add("* * * * *", report)

attempt {
    jobs.run()
} except ValueError as error {
    write("job failed: " + error.message)
}
```

**Kapanma.** Cron kendi sinyal işleyicisini kurmaz. Programı kesmek — Ctrl+C,
terminalini kapatmak, `ahdcode kill` veya işletim sisteminin süreci durdurması —
onu, diğer her AhdCode programı gibi hemen bitirir; o anda çalışan görev de
onunla durur. Süreç gittiğinde Scheduler'dan geriye hiçbir şey kalmaz.

## Sınırlı bir örnek

```ahd
bring Cron
bring Time
bring File
from Cron bring Scheduler

jobs: Scheduler := Cron.scheduler()
beats: Int := 0

heartbeat: Function := () -> Nothing {
    jobs: Global Scheduler
    beats: Global Int
    beats += 1
    line: Local String := "beat " + str(beats) + " at " + Time.now().toString()
    File.writeText("heartbeat.txt", line + "\n")
    write(line)
    if beats == 2 {
        jobs.stop()
    }
}

jobs.add("* * * * *", heartbeat)
jobs.run()
write("stopped after " + str(beats) + " beats")
```

Sonraki iki dakika sınırında bir kalp atışı yazar ve kendiliğinden biter.
Açıklamalı sürümü:
[`examples/v0.1/60_cron.ahd`](../examples/v0.1/60_cron.ahd).

## Bir Web uygulamasında Cron

`Scheduler.run` ve bir Web sunucusu, onları çağıran programı meşgul eder; bu
yüzden v1.2.0'da tek bir AhdCode programı aynı anda hem istek karşılayıp hem de
bir Scheduler çalıştırmaz. Zamanlanmış işi, Web uygulamasının yanında, onun
dosyalarını veya veritabanını paylaşan ikinci bir AhdCode programı olarak
çalıştırın. Bkz.
[Web: zamanlanmış uygulama işleri](WEB_TR.md#cron-zamanlanmış-uygulama-işleri).

## Hatalar

`CronError`, `Error`'dan türer. Mesajları kararlıdır:

```text
Scheduler.add: the schedule is empty
Scheduler.add: schedule "* * * *" has 4 fields; expected 5 (minute hour day-of-month month day-of-week)
Scheduler.add: schedule "60 * * * *" minute value 60 is outside 0..59
Scheduler.add: schedule "a * * * *" minute field "a" is not a number, range, step, or list
Scheduler.add: schedule "*/0 * * * *" minute step 0 is outside 1..59
Scheduler.add: schedule "5/10 * * * *" minute step needs a range such as */5 or 0-30/5
Scheduler.add: schedule "30-10 * * * *" minute range 30-10 runs backwards
Scheduler.add: schedule "0 0 30 2 *" can never run: none of its day-of-month values occurs in its months
Scheduler.add: a job cannot be added while the Scheduler is running
Scheduler.run: the Scheduler has no jobs; add one with Scheduler.add first
Scheduler.run: the Scheduler is already running
Cron.next: schedule "0 0 29 2 *" has no occurrence before the year 10000
```

`Cron.next`, zamanlama sorunlarını `Cron.next:` önekiyle bildirir. Zamanlama bir
metindir; bu yüzden geçersiz bir zamanlama derlenir ve kullanıldığında
`CronError` fırlatır. Yanlış bir argüman türü veya görev biçimi ise derleme
zamanı tanısıdır.

## Çalıştırma kipleri

`ahdcode run`, `ahdcode build` ve taşınmış (relocated) derlenmiş bir program tek
bir Cron uygulamasını paylaşır: aynı ayrıştırıcı, gerçekleşme araması ve
zamanlayıcı döngüsü. Kalıcı REPL de bu uygulamayı çalıştırır; ancak
`Scheduler.run`, bir görev onu durdurana kadar REPL oturumunu bloklar ve Ctrl+C
REPL'in kendisini sonlandırır. Zamanlamaları etkileşimli olarak `Cron.next` ile
deneyin, uzun ömürlü Scheduler'ları program olarak çalıştırın.

## Bu modülde olmayanlar

Kalıcı işler, yeniden denemeler, çakışan çalıştırmalar, worker havuzları, dağıtık
kilitleme, iş geçmişi, iş başına saat dilimi, saniye, cron takma adları, ay veya
gün adları, HTTP tetikleyicileri, kabuk komutu işleri ve sistem zamanlayıcı
entegrasyonu yoktur.
