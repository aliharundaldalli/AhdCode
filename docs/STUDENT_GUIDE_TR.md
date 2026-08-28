# AhdCode v0.1 Türkçe Öğrenci Rehberi

English version: [English Student Guide](STUDENT_GUIDE_EN.md)

Bu rehber, daha önce hiç program yazmamış olsanız bile küçük AhdCode komut
satırı programları kurabilmeniz için hazırlanmıştır. Bölümleri sırayla okuyun,
örnekleri çalıştırın ve her bölümün sonundaki küçük değişiklikleri kendiniz
deneyin.

## 1. AhdCode nedir?

AhdCode; okunabilir sözdizimi, açık niyet ve öngörülebilir davranış hedefleyen,
statik olarak denetlenen genel amaçlı bir programlama dilidir. Derleyici,
program çalışmadan önce tür uyuşmazlığı veya güvenli olmayan `null` kullanımı
gibi birçok hatayı yakalar.

v0.1 öğrenme ve deney aşamasındadır. Küçük CLI programları yazabilir, bunları
doğrudan çalıştırabilir veya native executable olarak derleyebilirsiniz.

## 2. Kurulum ve ilk program

AhdCode'u kaynaktan kurmak için Go 1.25 veya daha yeni bir sürüm gerekir:

```bash
cd AhdCode
go test ./...
go install ./cmd/ahdcode
export PATH="$(go env GOPATH)/bin:$PATH"
ahdcode --version
```

`hello.ahd` adında bir dosya oluşturun:

```ahd
name: String := "AhdCode"
write("Merhaba {name}")
```

Çalıştırın:

```bash
ahdcode run hello.ahd
```

Beklenen çıktı:

```text
Merhaba AhdCode
```

Küçük deneme: `name` değerini kendi adınızla değiştirip programı yeniden
çalıştırın.

## 3. Değişken tanımlama: `:=` ve `=`

`:=` yeni bir binding tanımlar. `=` ise daha önce tanımlanmış, değiştirilebilir
bir binding'in değerini günceller.

```ahd
score: Int := 70
write(score)

score = 85
write(score)
```

Beklenen çıktı:

```text
70
85
```

İlk satırda yalnız `=` kullanmak hatadır; çünkü henüz değiştirilecek bir
binding yoktur. Aynı scope içinde ikinci kez `:=` kullanmak da yeni bir
tanımlama çakışması oluşturur.

## 4. Temel türler

Başlangıçta en çok şu türleri kullanacaksınız:

| Tür | Anlamı | Örnek |
|---|---|---|
| `Int` | İşaretli 64-bit tam sayı | `42` |
| `Real` | Sonlu ondalıklı sayı | `3.5` |
| `String` | Değiştirilemeyen Unicode metin | `"Ayşe"` |
| `Bool` | Mantıksal değer | `true`, `false` |
| `List<T>` | Sıralı ve değiştirilebilir değer koleksiyonu | `[1, 2]` |
| `Pair<K, V>` | Ekleme sırasını koruyan anahtar/değer koleksiyonu | `{"Ali": 90}` |

```ahd
student: String := "Ayşe"
age: Int := 19
average: Real := 87.5
passed: Bool := average >= 50.0

write("{student}, {age}, {average}, {passed}")
```

Beklenen çıktı:

```text
Ayşe, 19, 87.5, true
```

Dil izin verdiği yerlerde `Int`, güvenli biçimde `Real` değerine genişleyebilir.
Ancak değiştirilebilir generic koleksiyonlar invariant'tır: `List<Int>`,
`List<Real>` yerine kullanılamaz.

## 5. Ekrana yazma ve kullanıcıdan veri alma

`write(value)` değeri ve ardından yeni satır yazar. `take()` bir satır metin
okur; `take(prompt)` önce kısa bir istem gösterir. `take` her zaman `String`
döndürür.

```ahd
name: String := take("Ad: ")
age: Int := int(take("Yaş: "))

write("{name} {age} yaşında")
```

Örnek etkileşim:

```text
Ad: Ali
Yaş: 20
Ali 20 yaşında
```

Küçük deneme: Şehir için üçüncü bir `take` çağrısı ekleyin.

## 6. `int`, `real` ve `str` dönüşümleri

Bu adlar her modülde hazırdır; `bring` gerekmez.

```ahd
write(int(3.7))
write(int(-3.7))
write(int(" +42 "))
write(real(2))
write(real("1e3"))
write(str(true))
```

Beklenen çıktı:

```text
3
-3
42
2.0
1000.0
true
```

`int(Real)` sıfıra doğru keser. `int(String)` çevresindeki boşluğu temizler,
isteğe bağlı `+`/`-` işaretini ve yalnız ondalık rakamları kabul eder. Ondalık
nokta, exponent, underscore ve taban öneki kabul etmez. `real(String)` ondalık
tam sayı, kesir ve exponent kabul eder; `NaN` veya infinity kabul etmez.
Geçersiz metin `DomainError`, aralık dışı sonuç `OverflowError` üretir.

## 7. `if`, `else` ve `Bool` koşulları

AhdCode'da koşul mutlaka `Bool` olmalıdır. Sıfır, boş String veya boş List için
truthiness yoktur.

```ahd
score: Int := 72

if score >= 85 {
    write("Pekiyi")
}
else if score >= 50 {
    write("Geçti")
}
else {
    write("Kaldı")
}
```

Beklenen çıktı:

```text
Geçti
```

`if score` geçersizdir; `if score > 0` gibi açık bir karşılaştırma yazın.

## 8. `while`, `until` ve `for`

`while` gövdeye girmeden önce koşulu kontrol eder. `until` ise **post-check**
döngüdür: gövdesi en az bir kez çalışır, sonra koşul `true` olduğunda durur.

```ahd
count: Int := 0

while count < 2 {
    write("while {count}")
    count++
}

until count == 4 {
    count++
    write("until {count}")
}

for value in [10, 20, 30] {
    write("for {value}")
}
```

Beklenen çıktı:

```text
while 0
while 1
until 3
until 4
for 10
for 20
for 30
```

Executable bir nested scope içinde yeni binding tanımlarken `Local` gerekir.
`break` en yakın döngüyü bitirir, `continue` bir sonraki adıma geçer.

## 9. `between` ile sayı aralığı

`between(start, stop)` başlangıcı içerir, bitişi **içermez**. Üçüncü argüman
adım büyüklüğüdür.

```ahd
for value in between(1, 6, 2) {
    write(value)
}
```

Beklenen çıktı:

```text
1
3
5
```

Negatif adım desteklenir. Sıfır adım `DomainError` üretir.

## 10. Function yazma ve çağırma

v0.1'de Function'lar adlandırılır; nested Function ve lambda yoktur.
Parametrelerin ve dönüş değerinin türü açıkça yazılır.

```ahd
greet: Function := (
    name: String
    title: String := "Öğrenci"
) -> String {
    return "Merhaba {title} {name}"
}

write(greet("Ali"))
write(greet(name: "Ayşe", title: "Dr"))
```

Beklenen çıktı:

```text
Merhaba Öğrenci Ali
Merhaba Dr Ayşe
```

Bir çağrı tamamen positional veya tamamen named olmalıdır; iki biçim aynı
çağrıda karıştırılmaz. Değer döndürmeyen bir Function'ın dönüş türü
`Nothing`'dir.

## 11. `Local` ve `Global`

Function parametreleri zaten kendi lexical scope'undadır. Executable nested
scope içinde oluşturduğunuz yeni binding'e `Local` yazarsınız. Bir Function,
module root'taki binding'e erişecekse gerekli `Global` bildirimini yapar.

```ahd
counter: Int := 0

increase: Function := (
) -> Nothing {
    counter: Global Int
    next: Local Int := counter + 1
    counter = next
}

increase()
increase()
write(counter)
```

Beklenen çıktı:

```text
2
```

`Global` yeni bir değer oluşturmaz; Function'a module-root binding'ini
kullandığını açıkça bildirir.

## 12. String işlemleri

String değiştirilemez ve Unicode karakterlerine göre indekslenir; UTF-8 byte
sayısına göre değil.

```ahd
text: String := "  Ali,Veli,Ayşe  "
clean: String := text.trim()

write(clean.lower())
write(clean.split(","))
write(clean.replace("Veli", "Can"))
write(clean.contains("Ayşe"))
write("A✓B" [1])
write("a✓b✓".index("✓"))
```

Beklenen çıktı:

```text
ali,veli,ayşe
["Ali", "Veli", "Ayşe"]
Ali,Can,Ayşe
true
✓
1
```

Yararlı diğer işlemler: `upper`, `capitalize`, `startsWith`, `endsWith` ve
`count`. `String.index` aranan metni bulamazsa `-1` değil `DomainError`
üretir. Sıradan geçersiz String indeksi `IndexError` üretir.

## 13. List ile çalışma

List sıralıdır, ilk indeks `0`'dır ve negatif indeks destekler. List bir
reference object'tir; aynı List'i gösteren alias'lar değişikliği görür.

```ahd
numbers: List<Int> := [10, 20, 30]
alias: List<Int> := numbers

alias[0] = 99
numbers.add(40)

write(numbers)
write(numbers[-1])
write(numbers same alias)
```

Beklenen çıktı:

```text
[99, 20, 30, 40]
40
true
```

Geçersiz sıradan indeksleme `IndexError` üretir. `List.index(value)` ise değer
bulunmadığında `DomainError` üretir; bunlar farklı durumlardır.

`Constant List<T>` yalnız binding'i değil, erişilebilen reference yapısını da
deep-freeze eder. Bir alias üzerinden değiştirme de engellenir. Mutable generic
koleksiyonların invariant olduğunu unutmayın.

## 14. `sort`, `reverse` ve `shuffle`

Bu üç işlem yeni List üretmez; mevcut List'i yerinde değiştirir. Bu nedenle
alias aynı değişimi görür.

```ahd
bring Math

values: List<Int> := [4, 1, 3, 2]
alias: List<Int> := values

values.sort()
write(values)

values.reverse()
write(values)

Math.seed(42)
values.shuffle()
write(values)
write(alias)
```

Beklenen çıktı, explicit seed nedeniyle her çalıştırmada aynıdır:

```text
[1, 2, 3, 4]
[4, 3, 2, 1]
[2, 4, 1, 3]
[2, 4, 1, 3]
```

`shuffle`, `Math.random` ve `Math.randomInt` ile aynı program-geneli
pseudo-random state'i tüketir. Seed verilmemiş native process başlangıcında bu
state OS entropy ile başlatılır; sonuçlar tekrarlanabilir değildir. Güvenlik
amaçlı rastgelelik için uygun değildir. Boş veya tek elemanlı List'i shuffle
etmek random state tüketmez.

## 15. `map`, `filter` ve Function callback'leri

v0.1'de lambda yoktur; callback olarak named Function değeri verilir. `map` ve
`filter` yeni List döndürür, kaynak List'i değiştirmez.

```ahd
double: Function := (
    value: Int
) -> Int {
    return value * 2
}

isEven: Function := (
    value: Int
) -> Bool {
    return value % 2 == 0
}

values: List<Int> := [1, 2, 3, 4]
doubled: List<Int> := values.map(double)
evens: List<Int> := values.filter(isEven)

write(values)
write(doubled)
write(evens)
```

Beklenen çıktı:

```text
[1, 2, 3, 4]
[2, 4, 6, 8]
[2, 4]
```

## 16. Pair ile çalışma

`Pair<K, V>` anahtar/değer eşlemesi tutar ve ekleme sırasını korur. v0.1'de
anahtar türü yalnız `String`, `Int` veya `Bool` olabilir.

```ahd
scores: Pair<String, Int> := {
    "Ali": 85
    "Ayşe": 92
}

scores["Ali"] = 90
scores["Veli"] = 78

for name in scores {
    write("{name}: {scores[name]}")
}
```

Beklenen çıktı:

```text
Ali: 90
Ayşe: 92
Veli: 78
```

Pair da reference object'tir; alias mutation'ı görür. Eksik anahtar `KeyError`
üretir. Bir anahtarı güncellemek sırasını değiştirmez; silip yeniden eklemek
onu sona taşır. `Constant Pair` erişilebilen reference graph'ını deep-freeze
eder.

## 17. Class ve attributes

Bir Class'ın constructor girdileri `structure: Attributes` bölümünde yazılır.
`Local` olmayan structure girdileri instance attribute olur.

```ahd
Student: Class<> := {
    structure: Attributes := (
        name: String
        number: Constant Int
    )

    describe: Function := (
    ) -> String {
        return "#{attribute.number} {attribute.name}"
    }
}

student: Student := Student(name: "Ali", number: 42)
write(student.describe())
```

Beklenen çıktı:

```text
#42 Ali
```

`Constant` attribute değiştirilemez; reference değer taşıyorsa deep-freeze
uygulanır. `Local` structure girdisi yalnız constructor içinde kullanılır ve
attribute olmaz. `Confidential` members sıradan dış erişime kapalıdır.

## 18. `null` güvenliği

`null` ayrı bir nullable type değildir; bir değerin mevcut olmama state'idir.
Derleyici bir binding'in `Null`, `MaybeNull` veya `NonNull` durumunu akışa göre
izler.

```ahd
message: String := null

if message == null {
    message = "hazır"
}

if message != null and message.contains("haz") {
    write(message.upper())
}
```

Beklenen çıktı:

```text
HAZIR
```

Kontrol yapılmadan belki-null bir değerde member access, çağrı veya indeksleme
derleme hatasıdır. List elemanları ve Pair değerleri null olabileceği için
okuduktan sonra refinement gerekebilir. `Constant` null ile başlatılamaz.

## 19. `attempt`, `except`, `ultimately` ve `toss`

Runtime errors yakalanabilir Class değerleridir. `attempt` korunan kodu,
`except` eşleşen hatayı, `ultimately` ise sonuç ne olursa olsun son adımı
çalıştırır. `toss` bir Error yükseltir.

```ahd
requirePositive: Function := (
    value: Int
) -> Int {
    if value <= 0 {
        toss (DomainError("değer pozitif olmalı"))
    }

    return value
}

attempt {
    result: Local Int := requirePositive(-1)
    write(result)
}
except DomainError as error {
    write("Yakalandı: {error.message}")
}
ultimately {
    write("Bitti")
}
```

Beklenen çıktı:

```text
Yakalandı: değer pozitif olmalı
Bitti
```

Yaygın türler arasında `DomainError`, `IndexError`, `KeyError`,
`OverflowError`, `DivisionByZeroError`, `NullError` ve `ConstantError` bulunur.

## 20. `bring` ve modüller

Yerel modül, import eden dosyayla aynı klasördeki `.ahd` dosyasıdır. Örneğin
`Greeting` adı `Greeting.ahd` dosyasını bulur.

`Greeting.ahd`:

```ahd
greet: Function := (
    name: String
) -> String {
    return "Modülden merhaba, {name}"
}
```

`main.ahd`:

```ahd
from Greeting bring greet

write(greet("Ayşe"))
```

`main.ahd` çalıştırıldığında beklenen çıktı:

```text
Modülden merhaba, Ayşe
```

`bring Greeting` namespace'i getirir ve çağrı `Greeting.greet("Ayşe")` olur.
`from Greeting bring greet` adı doğrudan getirir. Selective multiline import ve
`bring all` da desteklenir; `all` yalnız public, `Confidential` olmayan adları
getirir. Import çakışmaları ve circular dependencies hatadır.

## 21. Math modülü

`Math` açıkça import edilir. `randomInt(min, max)` iki sınırı da **içerir**.

```ahd
bring Math

write(Math.sqrt(81))
write(Math.round(3.14159, 2))

Math.seed(42)
write(Math.randomInt(1, 6))
write(Math.randomInt(1, 6))
```

Beklenen çıktı:

```text
9.0
3.14
2
2
```

Aynı seed aynı pseudo-random diziyi yeniden üretir. `Math.random()` değeri
`0.0 <= value < 1.0` aralığındadır. Seed verilmezse native process başlangıcı
OS entropy kullanır. Bu generator cryptographic güvenlik sağlamaz.

## 22. Küçük birleşik uygulama: not özeti

Bu uygulama input, String, List, Function, loop, koşul, numeric reduction ve
error handling'i bir araya getirir:

```ahd
checkGrade: Function := (
    grade: Int
) -> Int {
    if grade < 0 or grade > 100 {
        toss (DomainError("not 0 ile 100 arasında olmalı"))
    }

    return grade
}

name: String := take("Öğrenci: ").trim().capitalize()
grades: List<Int> := []

for index in between(1, 4) {
    attempt {
        grade: Local Int := checkGrade(int(take("Not {index}: ")))
        grades.add(grade)
    }
    except DomainError as error {
        write("Geçersiz giriş: {error.message}")
    }
}

if len(grades) > 0 {
    average: Local Real := sum(grades) / len(grades)
    write("{name}: {average}")
    write("En düşük: {min(grades)}")
    write("En yüksek: {max(grades)}")

    if average >= 50.0 {
        write("Geçti")
    }
    else {
        write("Kaldı")
    }
}
else {
    write("Geçerli not girilmedi")
}
```

`ali`, `90`, `80`, `70` girişleriyle örnek etkileşim:

```text
Öğrenci: ali
Not 1: 90
Not 2: 80
Not 3: 70
Ali: 80.0
En düşük: 70
En yüksek: 90
Geçti
```

Küçük deneme: Geçersiz bir not girip `except` kolunun mesajını gözlemleyin.

## 23. Başlangıçta sık yapılan hatalar

- Yeni binding için `=` değil `:=` kullanın.
- `if value` yazmayın; koşulu `value > 0` gibi bir `Bool` expression yapın.
- Executable nested scope'taki yeni declaration'a `Local` ekleyin.
- Function içinden module-root binding kullanırken gereken `Global` bildirimini
  yazın.
- `until` gövdesinin en az bir kez çalıştığını unutmayın.
- `between` stop değerini içermez.
- List'in ilk indeksinin `0` olduğunu unutmayın; negatif indeksler sondan
  erişir.
- `List.index`/`String.index` bulunamayan aramada `DomainError`, sıradan geçersiz
  indeksleme ise `IndexError` üretir.
- `sort`, `reverse` ve `shuffle` kaynak List'i değiştirir; `map` ve `filter`
  değiştirmez.
- Callback için lambda yazmayın; named Function değeri kullanın.
- Named ve positional argument'ları aynı çağrıda karıştırmayın.
- Belki-null bir değeri member access veya index öncesinde kontrol edin.

## 24. Alıştırmalar

Tam çözümleri hemen aramak yerine her programı küçük adımlarla kurun.

1. **Ad ve yaş:** Kullanıcıdan adını ve yaşını alın; gelecek yıl kaç yaşında
   olacağını yazın.
2. **Santigrat dönüşümü:** Santigrat değerini `Real` olarak okuyup Fahrenheit
   karşılığını hesaplayın.
3. **Tek veya çift:** Bir `Int` okuyun ve `%` kullanarak tek/çift mesajı yazın.
4. **Not ortalaması:** Üç notu List'e ekleyin; `sum` ve `len` ile ortalamasını
   bulun.
5. **En düşük ve en yüksek:** Bir not List'inde `min` ve `max` sonuçlarını
   gösterin; boş List durumunu önleyin.
6. **Basit menü döngüsü:** `until` ile en az bir kez görünen ve kullanıcı `0`
   girince duran küçük bir menü kurun.
7. **String normalizasyonu:** Çevresindeki boşluğu temizleyip adı küçük harfe
   dönüştüren, sonra ilk karakteri büyüten bir Function yazın.
8. **Öğrenci-not Pair'i:** İsimleri notlara bağlayın, bir notu güncelleyin ve
   ekleme sırasıyla yazdırın.
9. **Deterministic zar:** `Math.seed(42)` sonrası inclusive `randomInt(1, 6)`
   ile on zar atışı üretin ve aynı programın tekrarlandığını doğrulayın.
10. **Class tabanlı kayıt:** `name` ve `Constant number` attribute'ları olan bir
    `Student` Class'ı ve özet döndüren bir method yazın.

## 25. Çözüm İpuçları

1. `take` sonucu String'dir; yaş için `int(...)` ve yeni yaş için `+ 1` kullanın.
2. Formülü küçük parçalara ayırın; `real(take(...))` ile başlayın ve Real
   literal'ları kullanın.
3. `value % 2 == 0` bir `Bool` üretir.
4. Boş `List<Int>` için türü açıkça yazın; her girdiyi `add` ile ekleyin.
5. `min` ve `max` boş List'te `DomainError` üretir; önce `len(grades) > 0`
   kontrolü yapın.
6. `until` post-check olduğu için menü yazısını gövdenin başına koyabilirsiniz.
7. `trim`, `lower` ve `capitalize` işlemlerini bir dönüş expression'ında
   zincirlemeyi deneyin.
8. `Pair<String, Int>` kullanın; Pair üzerinde `for` anahtarları ekleme sırasıyla
   verir.
9. Seed'i atışlardan önce bir kez ayarlayın; stop-inclusive olduğu için sınırlar
   doğrudan `1, 6` olabilir.
10. Curated Class örneğindeki `structure: Attributes`, named construction ve
    `attribute.name` kullanımını model alın.

## 26. Sonraki adımlar ve teknik belgeler

Bu rehberi tamamladıktan sonra dilin ayrıntılarını şu belgelerden
derinleştirebilirsiniz:

- [Başlangıç / Getting Started](GETTING_STARTED.md)
- [Dil turu / Language Tour](LANGUAGE_TOUR.md)
- [Türler ve null / Types and Null](TYPES_AND_NULL.md)
- [Control Flow](CONTROL_FLOW.md)
- [Functions](FUNCTIONS.md)
- [Classes](CLASSES.md)
- [Collections](COLLECTIONS.md)
- [Modules](MODULES.md)
- [Errors](ERRORS.md)
- [Fundamentals](FUNDAMENTALS.md)
- [String API](STRING_API.md)
- [List API](LIST_API.md)
- [Math](MATH.md)
- [CLI](CLI.md)
- [Formatter](FORMATTER.md)
- [REPL](REPL.md)
- [Tam v0.1 spesifikasyonu](../AHDCODE_LANGUAGE_SPEC_v0.1.md)

Çalışan daha fazla örnek için [curated v0.1 examples](../examples/v0.1/README.md)
klasörünü inceleyin.
