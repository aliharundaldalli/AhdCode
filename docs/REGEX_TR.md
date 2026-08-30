# Regex standart modülü

[English](REGEX.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Türler ve Null](TYPES_AND_NULL_TR.md)

Regex, Math ve Time gibi açıktır (explicit):

```ahd
bring Regex
from Regex bring Pattern
from Regex bring RegexError
```

Kanonik kimlik `builtin:Regex`'tir; kardeş bir `Regex.ahd` onun yerini
alamaz (shadow edemez). Her argüman `NonNull`'dur.

## Bir desen (pattern) derlemek

```text
Regex.compile(pattern: String) -> Pattern
```

`Regex.compile`, Go `regexp` (RE2) sözdiziminde yazılmış bir deseni derler
ve bir `Pattern` örneği döndürür. Geçersiz bir desen, yakalanabilir
`RegexError`'ı fırlatır. `Pattern`, derleyici tarafından sağlanan
(compiler-supplied) bir Class'tır -- hiçbir zaman doğrudan oluşturulmaz,
yalnızca `Regex.compile` tarafından üretilir; tıpkı `Time.dateTime`'ın bir
`DateTime`'ın tek kaynağı olması gibi.

Class'ın adı `Regex` değil `Pattern`'dır: `bring Regex` zaten `Regex` ismini
modül isim uzayına bağlar, bu yüzden derlenmiş-desen türünün, `from Regex
bring Pattern` ile kendi başına içe aktarılabilmesi için kendi adına
ihtiyacı vardır.

```ahd
bring Regex
from Regex bring Pattern

digits: Pattern := Regex.compile("[0-9]+")
```

## Yüzey (Surface)

```text
matches(text: String)                      -> Bool
find(text: String)                         -> String?
findAll(text: String)                      -> List<String>
groups(text: String)                       -> List<String>?
replace(text: String, replacement: String) -> String
split(text: String)                        -> List<String>
```

`matches`, desenin `text` içinde **herhangi bir yerde** bulunup
bulunmadığını bildirir -- örtük bir tam-string eşleşmesi değildir. Buna
ihtiyacınız varsa deseni kendiniz sınırlandırın (`"^...$"`).

```ahd
digits.matches("order #482")   // true
digits.matches("no digits")    // false
```

`find`, ilk eşleşmeyi döndürür veya desen hiç geçmiyorsa `null` döndürür.
Sonuç `String?` olduğundan, kullanmadan önce sıradan null olabilirlik
kuralları geçerlidir:

```ahd
first: String? := digits.find("order #482, item #7")
if first != null {
    write(first)
}
```

`findAll`, sırayla çakışmayan (non-overlapping) her eşleşmeyi döndürür;
eşleşme yoksa boş bir `List<String>`. `replace`, **her** eşleşmeyi,
`$1`, `$2` gibi yakalama gruplarına (capture groups) başvurabilen
`replacement` ile değiştirir. `split`, `text`'i desenin her eşleşmesinde
böler.

```ahd
write(digits.findAll("order #482, item #7"))       // ["482", "7"]
write(digits.replace("order #482, item #7", "N"))  // "order #N, item #N"

whitespace: Pattern := Regex.compile("\\s+")
write(whitespace.split("one   two\tthree"))          // ["one", "two", "three"]
```

`groups`, ilk eşleşmenin tam eşleşme metnini, ardından yakalama gruplarını
(indeks `0` tam eşleşmedir) döndürür veya desen hiç geçmiyorsa `null`
döndürür. Eşleşmemiş isteğe bağlı bir grup, boş bir `String` olarak
bildirilir.

```ahd
entry: Pattern := Regex.compile("([a-zA-Z]+)-([0-9]+)")
parts: List<String>? := entry.groups("item-42")
if parts != null {
    write("whole: {parts[0]}, name: {parts[1]}, number: {parts[2]}")
}
```

## RegexError

`RegexError`, doğrudan `Error`'dan türer (`IOError`'dan değil) ve yalnızca
`Regex.compile` tarafından geçersiz desen sözdizimi için fırlatılır. Hiçbir
`Pattern` işlemi -- `matches`, `find`, `findAll`, `groups`, `replace`,
`split` -- bir `Pattern` var olduktan sonra başarısız olamaz.

```ahd
bring Regex
from Regex bring Pattern
from Regex bring RegexError

attempt {
    Regex.compile("(unterminated")
} except RegexError as error {
    write(error.message)
}
```

## Önbellekleme (caching) ve determinizm

Bir `Pattern`'ın tek gözlemlenebilir durumu kaynak desen metnidir; derlenmiş
eşleştiricinin (matcher) kendisi bir uygulama detayıdır ve dahili olarak
desen metnine göre önbelleğe alınır, bu yüzden aynı `Pattern` değerini
tekrar tekrar kullanmak -- veya aynı desen dizesiyle `Regex.compile`'ı
tekrar çağırmak -- derleme maliyetini tekrar tekrar ödemez. Eşleştirme,
değiştirme (replacement) ve bölme (splitting), belirli bir desen ve girdi
için deterministiktir ve yerel arkayüzde (native backend) ile kalıcı REPL'de
aynı şekilde davranır.
