# Terminal standart modülü

[English](TERMINAL.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Temel İşlevler](FUNDAMENTALS_TR.md) · [Env](ENV_TR.md)

`Terminal`, AhdCode v1.5.0 ile gelen ve derleyicide kayıtlı olan
`builtin:Terminal` modülüdür. Temel işlevler `write`, `take` ve `str`'nin
bilerek sunmadığı terminale özgü davranışları ekler: String'leri standart
çıktıya birleştirerek yazmak, standart hataya yazmak, tamponu boşaltmak,
standart çıktının etkileşimli bir terminal olup olmadığını ve boyutunu sormak,
metni biçimlendirmek ve koleksiyonları okunabilir biçimde düzenlemek. Yalnızca
Go standart kütüphanesiyle yazılmıştır ve `ahdcode run`, yerel derlemeler ve
REPL içinde aynı davranır.

## Genel arayüz

```text
bring Terminal
from Terminal bring TerminalError

Terminal.emit(parts: List<String>, separator: String := " ", ending: String := "\n") -> Nothing
Terminal.error(text: String, ending: String := "\n")                               -> Nothing
Terminal.flush()                                                                   -> Nothing
Terminal.isInteractive()                                                           -> Bool
Terminal.width()                                                                   -> Int?
Terminal.height()                                                                  -> Int?
Terminal.supportsColor()                                                           -> Bool
Terminal.style(
    text: String
    foreground: String := "default"
    background: String := "default"
    bold: Bool := false
    underline: Bool := false
)                                                                                  -> String
Terminal.pretty(value)                                                             -> Nothing

TerminalError  (Error'dan türer)
```

`pretty` türe göre çalışan bir işlemdir: `value`, `write`'ın kabul ettiği her
değer olabilir. `Lists.chunk` gibi kendi Function değeri yoktur; doğrudan
çağırın.

## write, take ve str ile ilişkisi

| | Ne yapar |
| --- | --- |
| `str(value)` | bir değerin kanonik ve belirlenimci metni |
| `write(value)` | bu metni ve bir satır sonunu standart çıktıya yazar |
| `take()` / `take(prompt)` | standart girdiden bir satır okur |
| `Terminal` | bu üçünün bilerek dışarıda bıraktığı terminale özgü davranışlar |

`write`, `take` ve `str` v1.5.0'da değişmedi. `Terminal`, onların zaten
yaptığı bir işi ikinci bir yolla yapmaz: `emit` sizin ürettiğiniz String'leri
alır, `pretty` ise `write` ve `str`'nin ürettiği metni aynen kullanır.

## Terminal fonksiyonlarını çağırmak

Bir AhdCode çağrısı ya tamamen sıralı ya da tamamen isimli argümanlarla
yapılır (dil spesifikasyonu §15.3). İkisi de geçerlidir:

```ahd
bring Terminal

Terminal.emit(["Ali", "95", "Geçti"], " | ")
Terminal.emit(parts: ["Ali", "95", "Geçti"], separator: " | ")
```

Aynı çağrıda sıralı bir `parts` List'i ile isimli bir `separator` yazmak iki
biçimi karıştırır ve derleme hatasıdır. Yazılmayan bir parametre, iki biçimde de
varsayılan değerini alır.

## emit

`Terminal.emit(parts, separator, ending)` parçaları sırasıyla, ayırıcıyı
yalnızca iki parçanın arasına ve sonu tam bir kez yazar.

```ahd
bring Terminal

Terminal.emit(["Ali", "95", "Geçti"])
Terminal.emit(["Ali", "95", "Geçti"], " | ")
Terminal.emit(parts: ["Yükleniyor"], ending: "")
Terminal.emit(parts: ["..."], separator: "", ending: "\n")
```

tam olarak şunu yazar:

```text
Ali 95 Geçti
Ali | 95 | Geçti
Yükleniyor...
```

- Boş bir List yalnızca sonu yazar.
- Ayırıcı da son da boş olabilir.
- Unicode, bir parçanın içindeki satır sonları ve ayırıcı dahil metin aynen
  yazılır.
- `emit`, `List<String>` alır ve hiçbir şeyi dönüştürmez. Metni kendiniz
  üretin: `Terminal.emit([ad, str(puan)])` ya da
  `Terminal.emit(["{ad}: {puan}"])`. `List<String?>` derleme sırasında
  reddedilir ve değişken sayıda argüman alan bir biçim yoktur.

`emit`, `write` ile aynı standart çıktıya yazar; ikisi sıralarını korur.

## error

`Terminal.error(text, ending)` metni ve sonu **standart hataya** yazar. Hiçbir
şey eklenmez: ön ek, zaman damgası, önem düzeyi ya da renk yoktur.

```ahd
bring Terminal

Terminal.error("Yapılandırma bulunamadı")
```

standart hataya `Yapılandırma bulunamadı` ve bir satır sonu yazar; standart
çıktıya hiçbir şey yazmaz. `ahdcode run app.ahd 2> hatalar.txt` ile satır
`hatalar.txt` dosyasına gider.

`Terminal.error` yazmadan önce bekleyen standart çıktıyı boşaltır; böylece iki
akış aynı terminale ya da dosyaya gittiğinde metin program sırasıyla görünür.

## flush

`Terminal.flush()`, AhdCode'un hâlâ tuttuğu standart çıktıyı yazar.

Derlenmiş bir program — ve bir program derleyen `ahdcode run` — standart
çıktıyı tamponlar. Tampon program bittiğinde, `take` bir satır okumadan önce,
`Terminal.error` yazmadan önce ve yakalanmamış bir hata bildirilmeden önce
yazılır. Sunucu, Cron zamanlayıcısı ya da bekleyen bir döngü gibi çalışmaya
devam eden bir program, hemen görünmesi gereken çıktıdan sonra
`Terminal.flush()` çağırmalıdır. Standart hata tamponlanmaz.

REPL'de çıktı üretildiği anda yazılır; bu yüzden `flush`'ın genellikle yapacak
bir işi olmaz. `flush`'ı art arda çağırmak zararsızdır. Hiçbir akışı yeniden
açmaz ya da değiştirmez ve terminalin kipini değiştirmez. Standart çıktı
yazılamazsa `flush`, `TerminalError` verir.

## isInteractive

`Terminal.isInteractive()`, **standart çıktı** etkileşimli bir terminale
bağlıysa `true`, bir dosyaya, bir boruya (pipe) ya da boş aygıta
yönlendirildiyse `false` döndürür. Standart girdiye bakılmaz.

```ahd
bring Terminal

if Terminal.isInteractive() {
    Terminal.emit(["Adınızı yazıp Enter'a basın."])
}
```

`ahdcode run` programa terminalin kendi standart çıktısını verir; yani yanıt,
derlenmiş programı doğrudan çalıştırdığınızdakiyle aynıdır. REPL'de REPL'in
çıktısını anlatır. Terminal olmaması asla bir hata değildir.

## width ve height

`Terminal.width()` ve `Terminal.height()`, standart çıktıdaki terminalin çağrı
anındaki sütun ve satır sayısıdır.

```ahd
bring Terminal

genislik: Int? := Terminal.width()
if genislik != null {
    Terminal.emit(["Terminal {genislik} sütun genişliğinde."])
}
```

Her biri pozitif bir `Int`'tir; standart çıktı bir terminal değilse ya da
terminal boyut bildirmiyorsa `null` olur. Sıfır olarak bildirilen bir boyut da
`null`'dır. Yedek değer yoktur: AhdCode asla 80 sütun ya da 24 satır uydurmaz,
`COLUMNS` ya da `LINES` değişkenlerini yanıt saymaz ve `tput`, `stty` ya da
PowerShell çalıştırmaz. Windows'ta boyut, kaydırma arabelleğinin değil konsol
penceresinin boyutudur.

## supportsColor

`Terminal.supportsColor()`, standart çıktıda biçimlendirilmiş metin kullanılıp
kullanılmayacağını yanıtlar. v1.5.0 politikası tam olarak şudur:

1. standart çıktı etkileşimli bir terminaldir, ve
2. terminal kaçış dizilerini yorumlar: macOS ve Linux'ta her zaman; Windows'ta
   yalnızca konsolda sanal terminal işleme zaten açıksa — AhdCode bunu okur,
   asla değiştirmez —, ve
3. `NO_COLOR` ayarlı değildir ya da boştur (NO_COLOR geleneği), ve
4. `TERM`, `dumb` değildir.

Başka hiçbir ortam değişkenine bakılmaz: `FORCE_COLOR`, `CLICOLOR` ve
`COLORTERM`'in etkisi yoktur. Politika her çağrıda yeniden değerlendirilir;
saklanan bir renk ayarı ve rengi açıp kapatan bir fonksiyon yoktur.

## style

`Terminal.style(...)` bir String döndürür; kendisi hiçbir şey yazmaz.

```ahd
bring Terminal

basari := Terminal.style(text: "Başarılı", foreground: "green", bold: true)
uyari := Terminal.style(text: "Uyarı", foreground: "yellow", underline: true)
Terminal.emit([basari, uyari], " / ")
```

Renk adları kapalı bir kümedir ve tam olarak küçük harfle yazılır:

| Ad | Ön plan | Arka plan |
| --- | --- | --- |
| `default` | renk kodu yok | renk kodu yok |
| `black` | 30 | 40 |
| `red` | 31 | 41 |
| `green` | 32 | 42 |
| `yellow` | 33 | 43 |
| `blue` | 34 | 44 |
| `magenta` | 35 | 45 |
| `cyan` | 36 | 46 |
| `white` | 37 | 47 |

- `"Red"`, `"bright-red"`, bir sayı ya da `""` dahil başka her ad
  `TerminalError` verir. Ad, renk kullanılamadığında bile denetlenir; böylece
  bir yazım hatası her makinede aynı biçimde başarısız olur. Bilinmeyen bir ad
  asla `default` sayılmaz.
- `supportsColor()` `true` iken ve en az bir özellik istendiğinde sonuç, `ESC[`
  + kodlar + `m`, ardından metin, ardından sıfırlama dizisi `ESC[0m`'dir. Kodlar
  kalın (`1`), altı çizili (`4`), ön plan, arka plan sırasıyla gelir:
  `Terminal.style("OK", "green", "white", true, true)` sonucu
  `ESC[1;4;32;47mOK ESC[0m`'dir (okunabilirlik için boşlukla gösterildi; gerçekte
  boşluk yoktur).
- `supportsColor()` `false` iken — yönlendirilmiş çıktı, `NO_COLOR`,
  `TERM=dumb` — sonuç metnin kendisidir; böylece `komut > cikti.txt` ve
  `komut | baska-komut` hiçbir kaçış dizisi almaz.
- Bütün argümanlar varsayılan değerindeyken sonuç metnin kendisidir.
- Biçimlendirilmiş her String bir sıfırlamayla biter; biçim sonraki çıktıya
  asla taşmaz. 256 renk, RGB ya da tema desteği yoktur.

**İç içe kullanım.** `style`'a verilen metin zaten biçimlendirilmiş bir String
içeriyorsa dış biçim her iç sıfırlamadan sonra yeniden uygulanır; yani iç biçimin
bittiği yerde devam eder:

```ahd
bring Terminal

durum := Terminal.style(
    text: "Durum: " + Terminal.style(
        text: "gecikti"
        foreground: "yellow"
    ) + " bugün"
    bold: true
)
```

Burada `Durum:` ve `bugün` ikisi de kalındır.

**Standart hata.** `supportsColor` ve `style` standart çıktıyı anlatır. Standart
çıktı bir terminalken standart hata bir dosyaya yönlendirilmişse,
`Terminal.error` için biçimlendirilmiş metin o dosyaya kaçış dizileri taşır. Hata
metnini yalnızca iki akış aynı terminale gidiyorsa biçimlendirin.

## pretty

`Terminal.pretty(value)`, bir değeri okunmak üzere düzenleyip ardından bir satır
sonuyla standart çıktıya yazar.

```ahd
bring Terminal

notlar: Pair<String, List<Int>> := {"Ali": [90, 95, 100], "Ayşe": [85, 91, 97]}
Terminal.pretty(notlar)
```

şunu yazar:

```text
{
    "Ali": [
        90,
        95,
        100
    ],
    "Ayşe": [
        85,
        91,
        97
    ]
}
```

- Boş olmayan bir `List` ya da `Pair`, her düzey için dört boşluk girintiyle
  satır başına bir eleman yazılır; boş olanlar `[]` ya da `{}` olarak kalır.
- Bir koleksiyonun içindeki her şey, `str`'nin orada yazdığı gibi yazılır:
  String'ler çift tırnak içinde, `\"`, `\\`, `\n`, `\r` ve `\t` kaçışlanmış;
  `null`, `null` olarak; bir Class değeri `<ClassAdı>` olarak.
- List ya da Pair olmayan bir değer, bir Class'ın kendi `CStr`'si dahil,
  `write`'ın yazdığı gibi yazılır.
- `pretty` bir Class'ın içine asla bakmaz; bu yüzden `Confidential` özellikler
  hiçbir zaman gösterilmez.
- Bu, insanlar için bir düzendir, bir serileştirme biçimi değildir; programlar
  için veri üretmek üzere [JSON](JSON_TR.md) kullanın.

## Hatalar

Her hata `Error`'dan türeyen bir `TerminalError`'dır:

```text
unsupported foreground color "purple"; use default, black, red, green, yellow, blue, magenta, cyan, or white
unsupported background color "purple"; use default, black, red, green, yellow, blue, magenta, cyan, or white
standard output could not be flushed
```

Terminal olmaması bir hata değildir: `isInteractive()` `false`, `width()` ve
`height()` `null`, `supportsColor()` `false` olur. Tür ya da argüman sayısı
hataları derleme zamanı tanılamalarıdır.

## Platform notları

- **macOS ve Linux:** terminal tespiti ve boyut, çekirdeğin standart çıktı
  üzerindeki terminal isteklerinden gelir.
- **Windows:** tespit ve boyut konsol API'sinden gelir; kaçış dizileri yalnızca
  konsol onları zaten yorumluyorsa kullanılır.
- Terminal hiçbir süreç başlatmaz, hiçbir dosya okumaz ve ağa bağlanmaz.
  Okuduğu tek ortam değişkenleri `NO_COLOR` ve `TERM`'dir. Bağımlılık eklemez:
  üretilen programlar Go standart kütüphanesini kullanır.

## v1.5.0'da olmayanlar

İmleç hareketi, ekran temizleme, ham klavye girdisi, fare girdisi, ilerleme
çubukları, dönen göstergeler, menüler, günlük kaydı (logging), 256 renk ve RGB,
renk açma/kapama anahtarları, standart hata için ayrı yetenek sorguları ve
grafik arayüz.

Ayrıca bakın: [`examples/v1.5/terminal_demo`](../examples/v1.5/terminal_demo/README_TR.md) ·
[Temel İşlevler](FUNDAMENTALS_TR.md).
