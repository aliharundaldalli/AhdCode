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

Formatter, AST-farkındadır (AST-aware). Yorumları, string kaçış
karakterlerini (escapes), interpolasyonu ve tam çok satırlı string içeriğini
korur; geri kalan her şey için tek bir kanonik (standart) düzeni işler:

- Bir çağrı, List literali, Pair literali veya Function/structure parametre
  listesi, satır 80 sütuna sığdığında virgülle ayrılmış olarak tek bir
  satıra toplanır.
- Sığmayan bir tanesi, her öğe kendi satırında olacak şekilde bölünür ve
  **hiç virgül kullanılmaz** — sonda bile.
- Bir Function imzasının `(parametreler) -> DönüşTürü` şekli, her zaman
  gövdesini açan satırda bir arada tutulur.
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

Sadece stil olmayan tek yerleştirme kuralı şudur: `:=` veya `=` işaretinin
sağındaki ifade, işaretle aynı fiziksel satırda başlamalıdır. `:=`/`=`
işaretinden hemen sonraki ve açılış parantezinden önceki satır sonu dahil,
diğer her satır sonu serbesttir.

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
