# Formatter

[English](FORMATTER.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [CLI](CLI_TR.md)

Bir kaynak dosyayı yerinde biçimlendirin:

```bash
ahdcode format program.ahd
```

Dosyayı değiştirmeden biçimlendirmeyi kontrol edin:

```bash
ahdcode format --check program.ahd
```

v2.8.0'dan itibaren `format` istenen sayıda dosya ve klasör alır. Bir klasör,
altındaki bütün `.ahd` dosyaları demektir; adı nokta ile başlayan klasörler
atlanır. Biri başarısız olsa da bütün hedefler işlenir; herhangi bir dosya
geçersizse ya da `--check` ile kanonik değilse çıkış kodu 1'dir:

```bash
ahdcode format src tests main.ahd
ahdcode format --check .
```

Bir Function başlığı kaynakta satırlara yayılabilir: `->`, dönüş türü,
`uses [...]` ya da `{` yeni birer satırda başlayabilir. Kanonik başlık tek
satırdır; böyle bir satır arasına yazılmış yorum bildirimin üstüne taşınır.
(v2.7.0 ve öncesi böyle bir Function'ı geçersiz kaynağa çeviriyordu; bu
biçimde yazılmış dosyaları biçimlendirmeden önce güncelleyin.)

Formatter, yazmadan önce düzeninin kaynaktaki belirteç ve yorumların tam olarak
aynısını taşıdığını denetler. Taşımıyorsa `FMT001` bildirir ve dosyayı
değiştirmeden bırakır.

Formatter, AST-farkındadır (AST-aware). Yorumları, string kaçış
karakterlerini (escapes), interpolasyonu ve tam çok satırlı string içeriğini
korur; geri kalan her şey için tek bir kanonik (standart) düzeni işler:

- Bir çağrı, List literali, Pair literali veya Function/structure parametre
  listesi, satır 80 sütuna sığdığında virgülle ayrılmış olarak tek bir
  satıra toplanır.
- Sığmayan bir tanesi, her öğe kendi satırında olacak şekilde bölünür ve
  **hiç virgül kullanılmaz** — sonda bile.
- `require("Path/To/File.ahd")` asla satır kırılmaz. Yol derleme zamanı bir
  dize değişmezidir; satırlara bölmek geçersiz kaynaktır.
- Bir Function imzasının `(parametreler) -> DönüşTürü` şekli, her zaman
  gövdesini açan satırda bir arada tutulur.
- Lambda aynı parametre-listesi düzenini ve kanonik boşluğu kullanır:
  `lambda (x: Int) -> x > 0`; gerektiğinde yalnızca parametre listesi bölünür.
- Girinti (indentation), kaynağın nasıl girintilenmiş olduğundan bağımsız
  olarak, her zaman seviye başına 4 boşluktur.

Biçimlendirme deterministik ve idempotenttir: `format(format(source)) ==
format(source)`. Yerinde güncellemeler, atomik bir geçici dosya
değiştirmesi (temporary-file replacement) kullanır ve dosyanın izin
bitlerini korur. Geçersiz kaynak kod teşhis edilir ve değiştirilmeden
bırakılır — formatter, henüz ayrıştırılamayan (parse edilemeyen) bir dosyayı
asla kısmen yeniden yazmaz.

## Önerilen stil ile geçerli sözdizimi

Bunlar iki farklı şeydir. **Geçerli sözdizimi (valid syntax)**, ayrıştırıcının
(parser) kabul ettiği her şeydir: AhdCode, aynı satırdaki iki öğe arasında
virgül gerektirir, ancak satır sonları ve sondaki virgüller aksi halde
tamamen isteğe bağlıdır ve girinti hiçbir anlam taşımaz. Kesin gramer kuralı
için [Dil turu](LANGUAGE_TOUR_TR.md)'na bakın. **Önerilen stil (recommended
style)** ise `ahdcode format`'ın herhangi birinden ürettiği tek düzendir.
Hangi geçerli biçim rahatınıza geliyorsa onu yazın — bir dosya genelinde
stilleri karıştıran üretilmiş (generated) kod sorun değildir — ve
formatter'ın onu normalleştirmesine izin verin.

Sadece stil olmayan tek yerleştirme kuralı şudur: `:=` veya `=`
işaretinden hemen sonraki token, işaretle aynı fiziksel satırda
görünmelidir. Bu, o token bir açılış parantezi olsa bile geçerlidir —
`[`, `(` veya `{` işaretini işaretin kendi satırında yazmak zorunludur;
`:=`/`=` ile parantez *arasında* bir satır sonu reddedilir. Açılış
parantezinin kendisi işaretin satırında olduğunda, içindeki her satır sonu
serbesttir:

```ahd
values: List<Int> := [
    1
    2
    3
]
```

geçerlidir (`[`, `:=`'nin hemen ardından açılır), oysa

```ahd
values: List<Int> :=
    [1, 2, 3]
```

reddedilir, çünkü satır sonu `:=` ile `[` arasına düşer.

Kısa bir çağrı, eşit derecede geçerli birkaç yazım:

```ahd
add(1, 2)

add(
    1,
    2
)

add(
    1
    2
)
```

`ahdcode format` üçünü de şu şekilde işler:

```ahd
add(1, 2)
```

Bir Function imzası, kısa ve uzun hali — formatter satır bölünmesine
kaynağın nasıl yazıldığına değil, genişliğe göre karar verir:

```ahd
check: Function := (
    x: Int,
) -> Bool {
    return x > 0
}

calculate: Function := (first: Int, second: Int, description: String, flag: Bool) -> Real {
    return first
}
```

şu hale gelir:

```ahd
check: Function := (x: Int) -> Bool {
    return x > 0
}

calculate: Function := (
    first: Int
    second: Int
    description: String
    flag: Bool
) -> Real {
    return first
}
```

Bir List ve bir Pair, iki farklı geçerli şekilde yazılmış:

```ahd
numbers: List<Int> := [
    1,
    2,
    3,
]

scores: Pair<String, Int> := {"Ali": 85, "Ayşe": 92}
```

şuna dönüşür:

```ahd
numbers: List<Int> := [1, 2, 3]

scores: Pair<String, Int> := {"Ali": 85, "Ayşe": 92}
```

`scores` zaten tek satırda kalacak kadar kısaydı, bu yüzden onunla ilgili
hiçbir şey değişmedi — yalnızca boşluk ve `numbers`'daki sondaki virgül
değişti.
