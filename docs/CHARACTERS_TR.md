# Characters standart modülü

[English](CHARACTERS.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [String API](STRING_API_TR.md) · [Regex](REGEX_TR.md)

`Characters`, AhdCode v1.2.0 ile gelen, derleyiciye kayıtlı
`builtin:Characters` modülüdür. Karakter ve kod noktası (code point)
işlemlerinin tek açık yeridir. Kardeş bir `Characters.ahd` onun yerini alamaz.

```ahd
bring Characters
from Characters bring CharactersError
```

## Birim bir Unicode kod noktasıdır

Bu modülde bir karakter, bir Unicode **kod noktasıdır** (Unicode skaler
değeri). Asla bir UTF-8 baytı değildir ve genişletilmiş bir grafem kümesi
(extended grapheme cluster) de değildir.

- AhdCode'da **`Char` türü yoktur**. Tek bir karakter, tam olarak bir kod
  noktası taşıyan sıradan bir `String`'dir.
- `"A"`, `"ş"`, `"ğ"`, `"İ"` ve `"😊"` her biri tek kod noktasıdır.
- Okuyucunun tek sembol olarak gördüğü bir glif birden fazla kod noktası
  olabilir. `"e"` ve ardından gelen U+0301 COMBINING ACUTE ACCENT ekranda `é`
  olarak görünür ama burada **iki** karakterdir. Characters metni grafem
  kümelerine bölmez.

Bu, `String`'in zaten kullandığı birimdir: `len(text)`, `text[index]` ve
`for character in text` kod noktalarını sayar ve indeksler. Characters,
String'de olmayan işlemleri — List'e dönüşüm, kod noktası değerleri ve Unicode
sınıflandırma — metin için ikinci bir model kurmadan ekler.

## Yüzey

```text
list(text: String)            -> List<String>
count(text: String)           -> Int

codePoint(character: String)  -> Int
fromCodePoint(value: Int)     -> String

isLetter(character: String)       -> Bool
isDigit(character: String)        -> Bool
isWhitespace(character: String)   -> Bool
isUpper(character: String)        -> Bool
isLower(character: String)        -> Bool
isAlphaNumeric(character: String) -> Bool
isPunctuation(character: String)  -> Bool
isSymbol(character: String)       -> Bool

CharactersError
```

Her argüman `NonNull`'dır. Argüman türleri derleme zamanında denetlenir:
`Characters.codePoint(65)` veya `Characters.fromCodePoint("A")` bir çalışma
zamanı hatası değil, sıradan `SEM004` tür uyuşmazlığıdır.

## list ve count

`list(text)`, `text`'in her kod noktasını kaynak sırasıyla kendi tek
karakterlik String'i olarak döndürür. `text` değişmez.

```ahd
bring Characters

write(Characters.list("Aş😊"))
write(Characters.count("Aş😊"))
```

=>

```text
["A", "ş", "😊"]
3
```

`count(text)` kod noktası sayısıdır ve her zaman `len(text)`'e eşittir;
`Characters.count`, karakter odaklı kodun öyle okunması için vardır. İkisi de
UTF-8 baytı saymaz: `"çğıöşü"` altı karakter, on iki bayttır.

Boş bir String `[]` ve `0` verir. Satır sonu, sekme ve boşluk da diğerleri gibi
birer karakterdir.

## codePoint ve fromCodePoint

`codePoint(character)`, tam olarak bir kod noktası taşıyan bir String'in
Unicode skaler değerini döndürür. `fromCodePoint(value)` bunun tersidir.

```ahd
bring Characters

write(Characters.codePoint("A"))
write(Characters.codePoint("ş"))
write(Characters.fromCodePoint(128522))
```

=>

```text
65
351
😊
```

`fromCodePoint` yalnızca Unicode skaler değerlerini kabul eder: `0..1114111`,
`55296..57343` (U+D800..U+DFFF) vekil (surrogate) aralığı hariç. Negatif bir
değer, U+10FFFF üzerindeki bir değer veya bir vekil `CharactersError`
fırlatır. Hiçbir şey kesilmez, sarılmaz (wrap) veya U+FFFD ile değiştirilmez.

## Sınıflandırma

Her yüklem (predicate) tam olarak bir karakter alır ve AhdCode'un birlikte
dağıtıldığı araç zincirindeki Unicode karakter veritabanını kullanır (Go
1.27.0 için Unicode 17.0.0). Derleyici ve derlenmiş bir program aynı
tabloları kullanır.

| Fonksiyon | Doğru olduğu durum |
|---|---|
| `isLetter` | Genel Kategori L: `Lu`, `Ll`, `Lt`, `Lm`, `Lo` |
| `isDigit` | Yalnızca Genel Kategori `Nd` — `7` ve `٣` gibi her yazı sistemindeki ondalık rakamlar |
| `isWhitespace` | Unicode `White_Space` özelliği; `String.trim`'in kullandığı tanımın aynısı |
| `isUpper` | Genel Kategori `Lu` |
| `isLower` | Genel Kategori `Ll` |
| `isAlphaNumeric` | bir harf (`L`) veya bir ondalık rakam (`Nd`) |
| `isPunctuation` | Genel Kategori P: `Pc`, `Pd`, `Ps`, `Pe`, `Pi`, `Pf`, `Po` |
| `isSymbol` | Genel Kategori S: `Sm`, `Sc`, `Sk`, `So` |

`isDigit` bilinçli olarak dardır: `"²"` (`No`) ve `"Ⅻ"` (`Nl`) sayısal
karakterlerdir ama ondalık rakam değildir; `isDigit` onlar için `false`'tur.
`"語"` gibi büyük/küçük harfi olmayan bir harf, ne büyük ne küçük olan bir
harftir. `"😊"` gibi bir emoji bir semboldür (`So`). ASCII `+`, `$`, `^` ve `|`
noktalama değil, semboldür.

Türkçe harfler, hiçbir yerel ayar (locale) kuralı olmadan kendi kod
noktalarıyla sınıflanır:

```ahd
bring Characters

write(Characters.isUpper("İ"))
write(Characters.isLower("ı"))
write(Characters.isLetter("ğ"))
```

=>

```text
true
true
true
```

## Tam olarak bir karakter ya da CharactersError

`character: String` alan olarak belgelenen bir fonksiyon, tam olarak bir kod
noktası taşıyan bir String ister. Asla yalnızca ilk kod noktasına bakmaz;
çünkü bu, çağıranın hatasını gizlerdi.

```ahd
bring Characters
from Characters bring CharactersError

attempt {
    write(Characters.isLetter("AB"))
} except CharactersError as error {
    write(error.message)
}
```

=>

```text
Characters.isLetter requires exactly one character; received 2 characters
```

Mesajlar kararlıdır:

```text
Characters.codePoint requires exactly one character; received an empty String
Characters.isLetter requires exactly one character; received 2 characters
Characters.fromCodePoint: -1 is outside the Unicode range 0..1114111
Characters.fromCodePoint: 55296 is a surrogate code point, not a Unicode scalar value
```

`CharactersError`, `Error`'dan türer. Yanlış statik tür bir derleme zamanı
tanısıdır; kod noktası sayısı yanlış olan bir String veya skaler değer olmayan
bir Int ise bu çalışma zamanı hatasıdır.

## Küçük bir metin incelemesi

```ahd
bring Characters

text: String := "AhdCode – Türkçe: çğıöşü 😊"
letters: Int := 0
spaces: Int := 0
for character in Characters.list(text) {
    if Characters.isLetter(character) {
        letters += 1
    }
    if Characters.isWhitespace(character) {
        spaces += 1
    }
}
write(str(Characters.count(text)) + " characters, " + str(letters) + " letters, " + str(spaces) + " spaces")
```

=>

```text
26 characters, 19 letters, 4 spaces
```

Çalıştırılabilir tam sürüm:
[`examples/v0.1/59_characters.ahd`](../examples/v0.1/59_characters.ahd).

## Çalıştırma kipleri

`ahdcode run`, `ahdcode build`, taşınmış (relocated) derlenmiş bir program ve
kalıcı REPL aynı Characters uygulamasını çalıştırır; aynı sonuçları ve
mesajları verir. Characters dosya sistemi, ağ veya yardımcı süreç kullanmaz.

## Bu modülde olmayanlar

`Char` türü, bayt String'leri, grafem kümesi bölütleme, Unicode normalleştirme
(NFC/NFD), `String.lower`/`upper` ötesinde harf dönüşümü, yerel ayara özgü
kurallar, karakter adları ve yazı sistemi (script) özellikleri yoktur.
