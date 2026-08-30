# AhdCode v0.1 Türkçe Öğrenci Rehberi

Bu rehber, programlamaya yeni başlayanlar için tasarlanmıştır. Sizi günlük bir dil kullanarak adım adım AhdCode diliyle tanıştıracak ve yazacağınız ilk kod satırından itibaren size yol gösterecektir.

## İçindekiler
- [1. AhdCode nedir?](#1-ahdcode-nedir)
- [2. Kurulum ve ilk programınız](#2-kurulum-ve-ilk-programınız)
- [3. Kod yazımının temelleri](#3-kod-yazımının-temelleri)
- [4. Temel türler](#4-temel-türler)
- [5. Operatörler](#5-operatörler)
- [6. Metinler (Strings)](#6-metinler-strings)
- [7. Girdi, çıktı ve dönüşümler](#7-girdi-çıktı-ve-dönüşümler)
- [8. Koşullar: if ve state](#8-koşullar-if-ve-state)
- [9. Döngüler: while, until ve for](#9-döngüler-while-until-ve-for)
- [10. Fonksiyon yazmak ve çağırmak](#10-fonksiyon-yazmak-ve-çağırmak)
- [11. Local ve Global](#11-local-ve-global)
- [12. Listelerle çalışmak (List)](#12-listelerle-çalışmak-list)
- [13. Referans davranışı (Reference Behavior)](#13-referans-davranışı-reference-behavior)
- [14. Pair ile çalışmak](#14-pair-ile-çalışmak)
- [15. Sabitler (Constant)](#15-sabitler-constant)
- [16. Null güvenliği (Null safety)](#16-null-güvenliği-null-safety)
- [17. Sınıflar (Class) ve Özellikler (Attributes)](#17-sınıflar-class-ve-özellikler-attributes)
- [18. Hata yönetimi (`attempt`, `except`, `ultimately` ve `toss`)](#18-hata-yönetimi-attempt-except-ultimately-ve-toss)
- [19. Modüller ve bring](#19-modüller-ve-bring)
- [20. Temel işlevler modülü (Fundamentals)](#20-temel-işlevler-modülü-fundamentals)
- [21. Matematik modülü (Math)](#21-matematik-modülü-math)
- [22. Zaman modülü (Time)](#22-zaman-modülü-time)
- [23. Latex modülü (Latex)](#23-latex-modülü-latex)
- [24. Kod biçimlendirici (Formatter)](#24-kod-biçimlendirici-formatter)
- [25. Komut satırı (CLI)](#25-komut-satırı-cli)
- [26. Etkileşimli kabuk (REPL)](#26-etkileşimli-kabuk-repl)
- [27. Başlangıçta sık yapılan hatalar](#27-başlangıçta-sık-yapılan-hatalar)
- [28. Küçük Projeler](#28-küçük-projeler)
- [29. Alıştırmalar](#29-alıştırmalar)
- [30. Çözüm İpuçları](#30-çözüm-ipuçları)
- [31. Sonraki adımlar ve teknik belgeler](#31-sonraki-adımlar-ve-teknik-belgeler)

## 1. AhdCode nedir?

AhdCode, okunabilir kod ve öngörülebilir davranış için tasarlanmış genel amaçlı
bir dildir. Derleyici, programınızı çalıştırmadan önce kontrol eder. Bu sayede,
yanlış türde bir değer kullanmak veya `null` olabilecek bir değere güvenmek gibi
birçok hata erkenden yakalanır.

> **Teknik not:** Program çalışmadan önce türlerin kontrol edilmesine static
> checking denir.

0.1 sürümü deneysel bir öğrenim sürümüdür. Küçük CLI (komut satırı) programlarını
doğrudan çalıştırabilir veya onları tek başına çalışan yerel bir uygulamaya çevirebilir.

## 2. Kurulum ve ilk programınız

AhdCode'u kaynak kodundan derlemek için şu an Go 1.25 veya daha yeni bir sürüm gerekir:

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

Deneme sırası sizde: `AhdCode` kelimesini kendi adınızla değiştirin ve dosyayı
yeniden çalıştırın.

## 3. Kod yazımının temelleri

Her AhdCode programı bir `.ahd` dosyasına yazılır. Satırların sonuna noktalı virgül (`;`) koymanıza gerek yoktur, her komut genellikle kendi satırında yer alır. Kod blokları süslü parantez `{` ve `}` arasına yazılır.

Kendiniz veya diğer yazılımcılar için not bırakmak isterseniz, tek satırlık yorumlar için satıra `//` ile başlayabilirsiniz. Daha uzun notlar için `/*` ile başlayıp `*/` ile biten çok satırlı yorumlar (multiline comments) kullanabilirsiniz. Derleyici bu yorum satırlarını yok sayar. (Not: çok satırlı yorumlar iç içe geçemez).

```ahd
// Bu bir yorum satırıdır. Derleyici bunu görmezden gelir.
write("Bu satır çalışır")
```

Bir değeri hatırlamanız gerektiğinde bir "değişken" (variable) oluşturursunuz. Değişken isimleri (identifiers) bir harf veya `_` ile başlayabilir. Sonraki karakterlerde harfler, sayılar ve `_` kullanılabilir.

### Değişken tanımlama ve değiştirme: `:=` ve `=`

Yeni bir değişken oluşturmak için `:=` kullanın. Daha önce oluşturulmuş ve
değiştirilebilen bir değişkene yeni bir değer vermek için ise `=` kullanın.

```ahd
score: Int := 10
name: String := "Ayşe"

write(score)

score = 20
write(score)
```

Beklenen çıktı:

```text
10
20
```

İlk satırda yalnızca `=` kullanmak bir hatadır çünkü `score` henüz
oluşturulmamıştır. Örneğin, önceden tanımlanmamış bir değişkene `score = 10` yazmak derleyici tarafından reddedilir. Aynı blok içinde ikinci kez `:=` kullanmak, aynı
değişkeni tekrar oluşturmaya çalışmak demektir; bu da bir hatadır.

AhdCode, başlangıç değerinden açıkça anlaşılabilen statik türü çıkarabilir;
okunabilirlik için türü açıkça yazmaya da devam edebilirsiniz:

```ahd
age: Int := 19
name := "Ayşe"  // String çıkarılır
```

Tür çıkarımı dinamik tür demek değildir: `name = 5` yine hatadır. İç blokta
`name: Local := "Ayşe"` yazılır; kapsam niyeti otomatik çıkarılmaz.

> **Teknik not:** Bir değişken adının kullanılabileceği bölgeye onun "scope"u (kapsamı) denir. Türün derleyici tarafından otomatik bulunmasına ise "type inference" denir.

## 4. Temel türler

Başlangıçta en sık karşılaşacağınız türler şunlardır:

| Tür | Anlamı | Örnek |
|---|---|---|
| `Int` | İşaretli 64-bit tam sayı | `42` |
| `Real` | Kesirli/ondalıklı sayı (floating-point) | `3.5` |
| `String` | Değiştirilemeyen (immutable) metin | `"Ayşe"` |
| `Bool` | Mantıksal değer | `true`, `false` |
| `List<T>` | Sıralı ve değiştirilebilir değer koleksiyonu | `[1, 2]` |
| `Pair<K, V>` | Ekleme sırasını koruyan anahtar/değer sözlüğü | `{"Ali": 90}` |
| `Function` | Tekrar kullanılabilen kod bloğu | |
| `Class` | Özel veri yapısı (custom data structure) | |
| `Nothing` | Değer döndürmeyen bir fonksiyonun türü | |

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

Dilin izin verdiği yerlerde bir `Int`, güvenle bir `Real` olarak
kullanılabilir. Fakat `List<Int>` ile `List<Real>` farklı türlerdir. Bir
`List<Int>` değerini doğrudan `List<Real>` gereken yere veremezsiniz.

> **Teknik not:** Generic türler (koleksiyonlar) için geçerli olan bu katı kurala invariance denir.

AhdCode'da `T?` nullable (null olabilen) türdür. Derleyici ayrıca böyle bir
değerin o anda kesinlikle var olup olmadığını takip eder. Bunu Null Güvenliği
bölümünde işleyeceğiz.

## 5. Operatörler

AhdCode standart matematiksel ve mantıksal operatörleri destekler.

### Aritmetik
- `+` toplama
- `-` çıkarma
- `*` çarpma
- `/` bölme (her zaman `Real` döndürür, yani `5 / 2` sonucu `2.5`'tir)
- `%` kalan (mod alma, yalnızca `Int` değerlerle çalışır)
- `^` üs alma (sağdan birleşimlidir, yani `2 ^ 3 ^ 2`, `2 ^ (3 ^ 2)` anlamına gelir)

```ahd
write(10 + 5)
write(10 / 4)
write(10 % 3)
write(2 ^ 3)
```

Beklenen çıktı:
```text
15
2.5
1
8
```

AhdCode tam sayı (integer) matematiğini taşmalara karşı kontrol eder (overflow). Eğer bir sonuç bir `Int`'e sığmayacak kadar büyük veya küçükse, program yanlış sayılar üretmek yerine `OverflowError` vererek güvenli bir şekilde durur.

> **Dikkat:** Bölme işlemi `/` her zaman bir `Real` (kesirli sayı) döndürür. Tam sayı bölmesi istiyorsanız `int(a / b)` kullanmalısınız.

### Atama ve bileşik atama (Compound assignment)
Değişkenleri matematiksel işlemlerle doğrudan güncelleyebilirsiniz:
- `+=`, `-=`, `*=`, `/=`, `%=`, `^=`

```ahd
score: Int := 10
score += 5
write(score)
```

> **Dikkat:** `/` her zaman `Real` döndürdüğü için, `Int` bir değişken üzerinde `/=` kullanamazsınız. Eğer `score` bir `Int` ise `score /= 2` geçersizdir. Yalnızca `Real` değişkenler için geçerlidir.

### Birer artırma ve azaltma
Bir `Int` değişkenini tam olarak `1` artırmak veya azaltmak için `++` veya `--` kullanın. Bunlar kendi satırlarında tek başına durmalıdır; başka bir işlemin veya atamanın içinde kullanılamazlar.

```ahd
count: Int := 0
count++
write(count)
```

### Karşılaştırma, Kimlik ve Üyelik (Comparison, Identity, and Membership)

- `==` eşittir (iki değerin içeriği kendi türüne göre aynı mı diye bakar)
- `!=` eşit değildir
- `<` küçüktür
- `<=` küçük eşittir
- `>` büyüktür
- `>=` büyük eşittir

AhdCode ayrıca kimlik, tür ve üyelik kontrolleri için şu operatörleri sunar:

- `same` iki değişkenin bellekte tam olarak aynı nesneye işaret edip etmediğine bakar (kimlik).
- `is` / `is not` bir nesnenin belirli bir Sınıf (Class) türünden olup olmadığını kontrol eder.
- `in` / `not in` bir değerin bir koleksiyon veya String içinde bulunup bulunmadığını kontrol eder.
- `has` / `has not` bir Sınıf nesnesinin belirli bir üyeye (özellik veya metot) sahip olup olmadığını kontrol eder.

**`in` ve `not in` kullanımı:**
Bir değerin bir `List`'te, bir metin parçasının bir `String`'de veya bir anahtarın (key) bir `Pair`'de olup olmadığını kontrol edebilirsiniz. `Pair` için `in` değerleri (values) değil, yalnızca anahtarları arar.

```ahd
numbers: List<Int> := [10, 20, 30]
write(20 in numbers)
write(99 not in numbers)

text: String := "AhdCode"
write("Code" in text)

scores: Pair<String, Int> := {
    "Ali": 90
    "Ayşe": 95
}
write("Ali" in scores)
```

**`has` ve `has not` kullanımı:**
Bunlar yalnızca bir Sınıf (Class) nesnesinin belirli bir üyeye sahip olup olmadığını kontrol etmek için kullanılır. Sağ taraf bir String değil, doğrudan üyenin adı olmalıdır.

Bir değişkenin türü üst Sınıf olsa bile içinde tuttuğu gerçek nesne bir alt Sınıf olabilir. Teknik olarak `has`, değişkenin yazılı türüne değil, nesnenin çalışma anındaki gerçek Sınıfına bakar. Ayrıca üst Sınıftan miras alınan (inherited) üyeler de var kabul edilir.

```ahd
Person: Class<> := {
    structure: Attributes := ( name: String )
}
Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Int
    )
}

person: Person := Student(name: "Ali", number: 42)

write(person has number) // true
write(person has not nickname) // true
```

*Not: `person is Student` bu nesnenin bir `Student` olup olmadığını sorarken, `person has number` bu nesnenin `number` adında bir üyesi olup olmadığını sorar.*

### Mantıksal operatörler
- `and` (ikisi de doğruysa `true` üretir)
- `or` (en az biri doğruysa `true` üretir)
- `not` (doğruyu yanlışa, yanlışı doğruya çevirir)

```ahd
age: Int := 20
hasTicket: Bool := true

if age >= 18 and hasTicket {
    write("Hoş geldiniz!")
}
```

## 6. Metinler (Strings)

Bir `String` metin tutar. AhdCode Unicode'u tam olarak destekler; bu da herhangi bir dildeki harflerin ve emojilerin sorunsuz çalıştığı anlamına gelir. String'ler değiştirilemezdir (immutable): Bir String oluşturulduktan sonra yerinde güncellenemez. String'ler üzerindeki işlemler yeni bir String döndürür.

Metinleri üç şekilde yazabilirsiniz:
1. `"Çift tırnak"`
2. `'Tek tırnak'`
3. `"""Üçlü tırnak"""` (çok satırlı metinler için)

```ahd
greeting: String := "Merhaba"
letter: String := 'A'
poem: String := """
Güller kırmızı,
Menekşeler mavi.
"""
```

### Kaçış karakterleri ve değer yerleştirme (Interpolation)
Tırnak işaretleri veya yeni satır (`\n`) gibi özel karakterler için ters eğik çizgi `\` kullanın (`\"`).
Süslü parantez `{ }` kullanarak değişkenleri doğrudan bir metnin içine yerleştirebilirsiniz. Buna interpolation denir.

```ahd
name: String := "Ali"
write("Merhaba, {name}!")
```

### String API (Metotlar)
AhdCode, metinler için birçok faydalı metot sunar:

```ahd
text: String := "  Ali,Veli,Ayşe  "
clean: String := text.trim()

write(clean.lower())
write(clean.upper())
write(clean.capitalize())
write(clean.split(","))
write(clean.replace("Veli", "Can"))
write(clean.contains("Ayşe"))
write(clean.startsWith("Ali"))
write(clean.endsWith("Can"))
write(clean.count("i"))
write("a✓b✓".index("✓"))
```

Beklenen çıktı:

```text
ali,veli,ayşe
ALI,VELI,AYŞE
Ali,veli,ayşe
["Ali", "Veli", "Ayşe"]
Ali,Can,Ayşe
true
true
false
1
1
```

Aranan metni bulamayan bir `String.index` araması `DomainError` hatası üretir.

### İndeksleme ve Uzunluk
Köşeli parantez `[ ]` kullanarak tek bir karaktere erişebilir veya `len()` ile metnin kaç karakterden oluştuğunu bulabilirsiniz. İndeksler `0`'dan başlar. Metnin sonundan geriye doğru saymak için negatif indeksler kullanabilirsiniz (`-1` son karakterdir).

```ahd
word: String := "AhdCode"
write(len(word))
write(word[0])
write(word[-1])
```

Beklenen çıktı:
```text
7
A
e
```

Geçersiz bir sıradan indeksleme işlemi `IndexError` üretir.

## 7. Girdi, çıktı ve dönüşümler

`write(value)` bir değeri yazdırır ve ardından yeni bir satıra geçer. `take()`
kullanıcıdan bir satırlık metin okur; `take(prompt)` ise metin okumadan önce ekrana kısa bir mesaj (istem) gösterir. `take` her zaman bir `String` döndürür.

```ahd
name: String := take("İsim: ")
age: Int := int(take("Yaş: "))

write("{name} {age} yaşında")
```

Örnek etkileşim:

```text
İsim: Ali
Yaş: 20
Ali 20 yaşında
```

### `int`, `real` ve `str` ile dönüşümler

Bu fonksiyonlar her modülde, hiçbir içeri aktarma (bring) yapmadan mevcuttur.

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

`int(Real)` ondalık kısmı keser (sıfıra doğru yuvarlar).

`int(String)` metnin başındaki ve sonundaki boşlukları yoksayar ve yalnızca isteğe bağlı bir `+` veya `-` işareti ve ardından rakamları kabul eder. Ondalık nokta, üs (exponent), alt çizgi veya taban (base) ön eklerini kabul **etmez**.

`real(String)` ondalık tam sayıları, kesirleri ve üsleri kabul eder; ancak `NaN` veya sonsuzluk (infinity) kabul etmez.

Geçersiz bir metin `DomainError` üretir; çok büyük bir sayı ise `OverflowError` üretir. AhdCode metinleri sayılara otomatik dönüştürmez; bunu `int()` veya `real()` ile açıkça siz yapmalısınız.

## 8. Koşullar: `if` ve `state`

### `if` ve `else`
AhdCode'da her koşul mutlaka bir `Bool` olmalıdır. Sıfır, boş bir String veya boş bir List için otomatik doğru/yanlış kabulü (truthiness) yoktur.

```ahd
score: Int := 72

if score >= 85 {
    write("Mükemmel")
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

`if score` geçersizdir çünkü `score` bir `Int`'tir, `Bool` değil. `if score > 0` gibi açık bir karşılaştırma yazmalısınız.

### `state`, `condition` ve `default`
Bir değeri birçok belirli eşleşmeyle karşılaştırmanız gerektiğinde `state` kullanın. Bu, peş peşe yazılmış birçok `else if` zincirinden daha temizdir.

```ahd
status: String := "active"

state status {
    condition "active" {
        write("Hesap aktif")
    }
    condition "blocked" {
        write("Hesap engelli")
    }
    condition default {
        write("Bilinmeyen durum")
    }
}
```

Beklenen çıktı:
```text
Hesap aktif
```

`state` bloğu yalnızca eşleşen ilk `condition` bloğunu çalıştırır. Sonraki koşula "düşmez" (no fall-through), bu yüzden `break` yazmanıza gerek yoktur. Eğer hiçbir koşul eşleşmezse `condition default` çalışır.

> **Dikkat:** `state`, `condition` veya `default` kelimelerini "değişken" sanmayın. Bunlar aynı `if` ve `else` gibi programın akışını kontrol eden anahtar kelimelerdir.

## 9. Döngüler: `while`, `until` ve `for`

Döngüler, kodu tekrar tekrar çalıştırmanızı sağlar.

### `while` ve `until`
`while`, içindeki kodu çalıştırmadan önce koşulunu kontrol eder. `until` ise
tam tersini yapar: Önce gövdeyi çalıştırır, koşulu daha sonra kontrol eder.
Bu yüzden gövdesi en az bir kez çalışır ve koşul `true` olduğunda döngü durur.

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
```

Beklenen çıktı:

```text
while 0
while 1
until 3
until 4
```

> **Teknik not:** `while` bir pre-check (önceden kontrol eden), `until` ise post-check (sonradan kontrol eden) loop'tur.

### `break` ve `continue`
Bir döngüyü `break` ile erken durdurabilir veya `continue` ile mevcut turun geri kalanını atlayıp doğrudan bir sonraki tura geçebilirsiniz.

```ahd
count: Int := 0
while count < 10 {
    count++
    if count == 2 {
        continue
    }
    if count == 4 {
        break
    }
    write("sayaç {count}")
}
```

Beklenen çıktı:
```text
sayaç 1
sayaç 3
```

### `for` ve `between`
Bir koleksiyondaki öğelerin veya belirli bir sayı aralığının üzerinden geçmek (loop yapmak) için `for` kullanın.
`between(start, stop)`, başlangıcı içeren ve bitişi **dışlayan** (exclude) bir aralık oluşturur. Üçüncü bir argüman adım (step) değerini ayarlar.

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

Negatif adımlar desteklenir (ör. `between(10, 0, -2)` geriye doğru sayar). Sıfır adımı (zero step) sonsuz döngü yaratacağından `DomainError` üretir. `between` oldukça verimlidir; bellekte devasa bir List oluşturmaz, sadece sayıları anlık (lazy) olarak üretir.

Bir `for` değişkeninin türünü genellikle yazmanıza gerek yoktur; derleyici türü
ziyaret edilen değerlerden kendi öğrenebilir. Ancak isterseniz türü açıkça yazabilirsiniz:

```ahd
for value in [10, 20, 30] {
    write(value)
}

for value: Int in [10, 20, 30] {
    write(value)
}
```

Her iki kullanımda da `value` yalnızca o döngü için oluşturulur. Zaten o döngüye özeldir (Local). Başlangıcına `Local` eklemeyin.
Şu kullanım geçersizdir:

```ahd
// GEÇERSİZ:
for value: Local Int in [10, 20] {
    write(value)
}
```

> **Teknik not:** Türün derleyici tarafından bulunmasına type inference denir. `for` değişkenleri varsayılan olarak zaten Local kabul edilir. Listeler için "snapshot iteration" (anlık görüntü iterasyonu) kullanılır; yani döngünün başında var olan değerler üzerinden ilerlenir.


## 10. Fonksiyon yazmak ve çağırmak

v0.1'de her fonksiyonun bir adı olmalıdır. Bir fonksiyonun içine başka bir yeni
fonksiyon tanımlayamazsınız ve isimsiz fonksiyonlar (lambdas) yoktur. Her
parametrenin ve döndürülen değerin (return value) türü açıkça belirtilir.

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

Bir fonksiyon çağrılırken, değerleri ya tamamen sırayla (positional) ya da
tamamen isim vererek (named) göndermelisiniz; iki şekli aynı çağrıda
karıştıramazsınız. `title`'ın varsayılan bir değeri (`"Öğrenci"`) olduğu için fonksiyonu çağırırken onu girmek isteğe bağlıdır.

Hiçbir değer döndürmeyen bir fonksiyon `Nothing` dönüş türünü kullanır. `Nothing` döndüren bir fonksiyonda, yalın bir `return` ifadesi herhangi bir
değer üretmeden fonksiyonu anında bitirir. Fonksiyondan erken çıkmak
gerekmiyorsa `return` yazmak isteğe bağlıdır; kodun doğal olarak sonuna ulaşması yeterlidir (natural fall-through).

```ahd
showStatus: Function := (
    score: Int
) -> Nothing {
    if score < 0 {
        write("Geçersiz not")
        return
    }

    write("Not: {score}")
}

hello: Function := (
    name: String
) -> Nothing {
    write("Merhaba {name}")
}

showStatus(-5)
showStatus(80)
hello("Ayşe")
```

Beklenen çıktı:

```text
Geçersiz not
Not: 80
Merhaba Ayşe
```

İlk `showStatus` çağrısındaki yalın `return`, sonraki `write` satırının
çalışmasını engeller. `hello` ise gövdesinin sonuna ulaşarak doğal şekilde
tamamlanır.

### Öz yineleme (Recursion)
Fonksiyonlar kendi kendilerini çağırabilirler. Buna öz yineleme (recursion) denir. Fonksiyonun sonsuza kadar çalışmaması için kendisini çağırmayı durduracak bir koşul yazdığınızdan emin olmalısınız.

```ahd
countdown: Function := (
    n: Int
) -> Nothing {
    if n <= 0 {
        write("Ateşle!")
        return
    }
    write(n)
    countdown(n - 1)
}

countdown(3)
```

Beklenen çıktı:
```text
3
2
1
Ateşle!
```

### Aynı isimde birden fazla fonksiyon: Overloads

Parametre türleri farklı olduğu sürece aynı fonksiyon adı için birkaç farklı
versiyon tanımlayabilirsiniz. İlkini normal `Function`, sonrakileri ise
`Overload Function` olarak yazın:

```ahd
describe: Function := (
    value: Int
) -> String {
    return "Int {value}"
}

describe: Overload Function := (
    value: Real
) -> String {
    return "Real {value}"
}

write(describe(2))
write(describe(2.5))
```

Beklenen çıktı:

```text
Int 2
Real 2.5
```

Derleyici her zaman öncelikle parametre türünün tam eşleştiği (exact match) versiyonu
seçer. Gerektiğinde güvenli `Int`'ten `Real`'e çeviriyi (widening) kullanabilir. Ayrıca birden çok fonksiyon uyuyorsa, daha az varsayılan parametre kullanan versiyonu tercih eder. Eğer iki versiyon aynı derecede uygun görünüyorsa, çağrı belirsizdir
(ambiguous) ve derleme işlemi hata verip durur. Sadece dönüş türünün farklı
olması, hangi versiyonun seçileceğini belirlemeye yetmez.

> **Teknik not:** Bu seçim işlemine "overload resolution" (aşırı yükleme
> çözümlemesi) denir. Fonksiyonlar dinamik değildir; program çalışmadan önce
> derleyicinin her çağrının hangi versiyonu kullanacağını kesin olarak
> belirlemesi gerekir.

## 11. `Local` ve `Global`

Fonksiyonun parametrelerini fonksiyonun içinde doğrudan kullanabilirsiniz;
başlarına `Local` eklemeyin. Fonksiyonun veya `if`, `for`, `while` gibi bir iç
bloğun içinde yeni bir değişken oluştururken `Local` yazın. Dosyanın en üst
seviyesinde oluşturulmuş bir değişkeni fonksiyonun içinden kullanmak için ise o
erişimi `Global` ile beyan edin.

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

Burada `counter` dosyanın en üst seviyesinde oluşturulmuştur. `counter: Global Int`
satırı yeni bir sayaç oluşturmaz; sadece fonksiyona mevcut değişkeni
kullanmasını söyler. `next` ise yalnızca fonksiyonun içinde oluşturulur, bu
yüzden `Local` kullanır.

> **Teknik not:** Bu kurallar bir değişkenin "scope"unu, yani programın hangi
> kısımlarında kullanılabileceğini tanımlar. `Global` gizli bir kopya
> oluşturmaz, doğrudan modül kökündeki asıl değişkene işaret eder.

## 12. Listelerle çalışmak (List)

`List`, sıralı bir değer koleksiyonudur. İlk indeks `0`'dır ve negatif indeksler sondan geriye sayar (`-1` son öğedir).

### Ekleme, sıralama ve ters çevirme
```ahd
bring Math

values: List<Int> := [4, 1, 3]
values.add(2)
values.sort()
write(values)

values.reverse()
write(values)

Math.seed(42)
values.shuffle()
write(values)
```
Tohum (seed) açıkça belirtildiği için çıktı her seferinde aynıdır:
```text
[1, 2, 3, 4]
[4, 3, 2, 1]
[2, 4, 1, 3]
```

Bu işlemler (`sort`, `reverse`, `shuffle`) yeni bir List oluşturmaz. Sahip olduğunuz List'in sırasını doğrudan yerinde değiştirirler. `sort` doğal artan sıralamayı kullanır.

### Temizleme, Çıkarma ve Dilimleme (Slicing)
Bir List'ten öğeleri silebilirsiniz. `eject(index)` belirtilen indeksteki öğeyi yerinde siler. `clear(list)` tüm koleksiyonu boşaltır.

```ahd
letters: List<String> := ["A", "B", "C", "D"]
letters.eject(1)
write(letters)

clear(letters)
write(letters)
```
Beklenen çıktı:
```text
["A", "C", "D"]
[]
```

Bir List'in bir kısmını `[başlangıç:bitiş]` (slice) kullanarak alabilirsiniz. Bu yeni bir List döndürür.
```ahd
nums: List<Int> := [10, 20, 30, 40, 50]
slice: List<Int> := nums[1:4]
write(slice)
```
Beklenen çıktı:
```text
[20, 30, 40]
```

### Arama ve Sayma
`count(value)` bir değerin List'te kaç kez geçtiğini döndürür. `index(value)` değerin ilk bulunduğu konumu bulur.
```ahd
data: List<Int> := [7, 8, 7, 9]
write(data.count(7))
write(data.index(8))
```
Beklenen çıktı:
```text
2
1
```
> **Dikkat:** Eğer `index()` değeri bulamazsa `-1` döndürmez. `DomainError` fırlatır.

### Map, Filter ve Özel Sıralama (Keyed Sort)
v0.1 sürümünde isimsiz fonksiyon (lambda) yoktur, bu yüzden "callback"ler isimlendirilmiş Function değerleridir. `map` ve `filter` yeni Listeler döndürür ve kaynaklarını değiştirmezler. `sort(keyFunction)`, Listeyi sizin fonksiyonunuzun ürettiği sonuçlara göre sıralar (bu kararlı -stable- bir sıralamadır).

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

absSort: Function := (
    value: Int
) -> Int {
    return abs(value)
}

values: List<Int> := [3, -1, 4, -2]
doubled: List<Int> := values.map(double)
evens: List<Int> := values.filter(isEven)

values.sort(absSort)

write(doubled)
write(evens)
write(values)
```

Beklenen çıktı:

```text
[6, -2, 8, -4]
[4, -2]
[-1, -2, 3, 4]
```

## 13. Referans davranışı (Reference Behavior)

Eğer iki değişken aynı List'e bağlıysa, ikisi de aynı koleksiyonu görür. Bir
değişken üzerinden yapılan değişiklik, diğerinden de görülür.

```ahd
numbers: List<Int> := [10, 20, 30]
alias: List<Int> := numbers

alias[0] = 99
numbers.add(40)

write(numbers)
write(alias)
write(numbers same alias)
write(numbers == alias)
```

Beklenen çıktı:

```text
[99, 20, 30, 40]
[99, 20, 30, 40]
true
true
```

`same` anahtar kelimesi, her iki değişkenin bellekte tam olarak aynı nesneyi gösterip göstermediğini kontrol eder. `==` ise içeriklerinin tamamen aynı olup olmadığına bakar. Bu durumda, aynı nesneyi paylaştıkları için ikisi de doğrudur.

> **Teknik not:** Aynı List'i bu şekilde paylaşmaya "reference semantics" (referans semantiği) denir. Aynı nesneyi gösteren ikinci bir değişken adına genellikle "alias" (takma ad) denir.

## 14. Pair ile çalışmak

`Pair<K, V>` (çift), değerleri anahtarlarla (keys) eşleştirerek saklar ve
ekleme sırasını korur. v0.1'de anahtar türü yalnızca `String`, `Int` veya
`Bool` olabilir.

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

Eğer iki değişken aynı Pair'e işaret ediyorsa, biri üzerinden yapılan
değişiklik diğerinden de görülür (Referans Davranışı). Olmayan bir anahtarı aramak `KeyError` üretir. Var olan bir
anahtarın değerini güncellemek, o anahtarın sıradaki yerini korur; onu silip
yeniden eklemek ise listenin en sonuna taşır. Bir anahtarı `eject(key)` ile silebilir veya `clear(pair)` ile tüm Pair'i boşaltabilirsiniz.

```ahd
scores: Pair<String, Int> := {"Ali": 85}
scores.eject("Ali")
clear(scores)
```

## 15. Sabitler (Constant)

Bir `Constant` koleksiyon değiştirilemez. Eğer onu doğrudan değiştirmeye çalışırsanız derleme aşamasında reddedilir.

```ahd
locked: Constant List<Int> := [1, 2, 3]
// locked[0] = 99 // Bu bir derleme zamanı hatasına neden olur
```

Ulaşılabilir tüm nesne ağı derin dondurulur (deep-frozen). Eğer bir `Constant` içinde başka koleksiyonlar veya nesneler varsa, bu iç nesneler de dondurulur. Halihazırda dondurulmuş bir nesneyi, Constant olarak işaretlenmemiş başka bir değişken (alias/takma ad) üzerinden değiştirmeye çalışırsanız, program çalışırken (runtime) bir `ConstantError` üretebilir.

> **Teknik not:** Bir `Constant` değer, başlangıç değeri olarak `null` alamaz.

## 16. Null güvenliği (Null safety)

Yalın `T` null olamaz. Bilinen bir türün geçici olarak değeri olmayabilmesi için
`T?` yazılır. Program kodun farklı yollarından ilerledikçe derleyici, nullable
bir değerin "kesinlikle var", "kesinlikle `null`" veya
"belki `null`" (possibly null) olup olmadığını anbean takip eder.

```ahd
message: String? := null

if message == null {
    message = "hazır"
}

if message != null and message.contains("azı") {
    write(message.upper())
}
```

Beklenen çıktı:

```text
HAZIR
```

Eğer değerin `null` olma ihtimali varsa, onun üyelerine erişmek (member
access), fonksiyon gibi çağırmak veya indekslemek (indexing) derleme
zamanında bir hatadır. `message != null` kontrolünden sonra derleyici blok
içinde değerin var olduğunu bilir. `List<User?>` null elemanlı bir List,
`List<User>?` null olabilen bir List, `List<User?>?` ise ikisinin birleşimidir.
`value := null` geçersizdir; `value: User? := null` yazılır. `fetchUser()`
`User?` döndürüyorsa `user := fetchUser()` tam `User?` türünü çıkarır.

Eğer `null` olabilecek bir değeri önceden kontrol etmeden kullanmaya çalışırsanız derleme zamanında bir hata alırsınız.

> **Teknik not:** Dokümantasyon, bu üç ihtimale `Null`, `MaybeNull` ve
> `NonNull` adını verir. Bir kontrol yaptıktan sonra derleyicinin değer hakkında
> daha kesin bilgi sahibi olmasına "null refinement" denir.


## 17. Sınıflar (Class) ve Özellikler (Attributes)

Bir Sınıf (Class), özel bir veri yapısı tanımlar ve birbiriyle ilgili
fonksiyonları (metotları) bir araya getirir. Sınıfı oluşturmak için gereken girdiler `structure: Attributes` kısmında tanımlanır. Başına `Local` yazılmayan her yapı girdisi otomatik olarak bir örnek özelliğine (instance attribute) dönüşür.

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

`Constant` olarak tanımlanan bir özellik daha sonra değiştirilemez. Başına `Local` eklenen
bir yapı girdisi yalnızca nesne oluşturulurken kullanılır ve bir özelliğe
dönüşmez. Başına `Confidential` (gizli) eklenen üyelere ise sınıfın dışından
normal yollarla erişilemez.

### Üst ve alt Sınıflar (Parent / Child)

Bir Sınıf, başka bir Sınıfın özelliklerini devralabilir. Önce üst Sınıfı
(parent), ardından onu genişleten alt Sınıfı (child) okuyun:

```ahd
Person: Class<> := {
    structure: Attributes := (
        name: String
    )

    describe: Function := (
    ) -> String {
        return "Kişi {attribute.name}"
    }
}

Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Int
    )

    describe: Override Function := (
    ) -> String {
        return "{SuperClass.describe()} #{attribute.number}"
    }
}

student: Student := Student(name: "Ayşe", number: 7)
person: Person := student
write(person.describe())
```

Beklenen çıktı:

```text
Kişi Ayşe #7
```

`Student`, `Person`'ın bir alt Sınıfıdır. `SuperClass.attributes`, üst Sınıfın oluşturulması için gereken
girdileri aynen alt sınıfa da aktarır. `Override` kelimesi, üst sınıftan
miras alınan bir metodun kasten değiştirildiğini (üzerine yazıldığını)
belirtir. `SuperClass.describe()` ise üst Sınıfın kendi versiyonunu çağırır.

Her ne kadar `person` değişkeninin türü `Person` olsa da, o değişkenin
tuttuğu gerçek nesne bir `Student` olduğu için `Student.describe` çalışır.

Bir nesnenin asıl (gerçek) türünü kontrol etmek için `is` anahtar kelimesini kullanabilirsiniz:
```ahd
Person: Class<> := { structure: Attributes := ( name: String ) }
Student: Class<Person> := { structure: Attributes := ( SuperClass.attributes ) }

person: Person := Student(name: "Ayşe")
if person is Student {
    write("Bu kişi bir öğrenci!")
}
```

> **Teknik not:** Bir alt Sınıf nesnesini üst Sınıf türündeki bir değişkende
> tutmaya "upcasting" denir. Hangi metodun çalışacağının, nesnenin asıl türüne
> bakılarak çalışma anında seçilmesine ise "dynamic dispatch" denir.

## 18. Hata yönetimi (`attempt`, `except`, `ultimately` ve `toss`)

Eğer `attempt` (dene) içindeki kod bir hata üretirse, o hataya uygun bir
`except` (hariç) bloğu devreye girebilir. `ultimately` (en nihayetinde) bloğu
ise, bir hata oluşsun veya oluşmasın her zaman en son adım olarak çalışır. Kendi
kodunuz kasten bir Hata (Error) üretmek istediğinde `toss` (fırlat) kullanın.

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
    write("Domain hatası: {error.message}")
}
except IndexError as error {
    write("İndeks hatası: {error.message}")
}
ultimately {
    write("Bitti")
}
```

Beklenen çıktı:

```text
Domain hatası: değer pozitif olmalı
Bitti
```

Dilin sunduğu yaygın hata türleri arasında `DomainError`, `ValueError`, `IndexError`, `KeyError`,
`OverflowError`, `DivisionByZeroError`, `NullError` ve `ConstantError`
bulunur. Farklı hatalara farklı şekilde tepki verebilmek için birden fazla `except` bloğu kullanabilirsiniz.

Ayrıca yerleşik `Error` sınıfından miras alarak kendi özel hatalarınızı oluşturabilirsiniz:

```ahd
InvalidAgeError: Class<Error> := {
    structure: Attributes := (
        message: String
    )
}

attempt {
    age: Local Int := -5
    if age < 0 {
        toss (InvalidAgeError("Yaş negatif olamaz"))
    }
}
except InvalidAgeError as error {
    write("Özel hata yakalandı: {error.message}")
}
```

> **Teknik not:** AhdCode çalışma zamanı (runtime) hataları, yakalanabilen
> (catchable) normal Sınıf (Class) değerleridir.

## 19. Modüller ve `bring`

Yerel bir modül, onu içe aktaran (import) dosyayla aynı klasörde bulunan bir
`.ahd` dosyasıdır. Örneğin, `Greeting` (Selamlama) modül adı `Greeting.ahd`
dosyasını temsil eder.

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

Bir modülden öğeleri içe aktarmanın (import) birkaç yolu vardır:
- `bring Greeting` bir "isim uzayını" (namespace) içe aktarır, çağrı `Greeting.greet("Ayşe")` şekline dönüşür.
- `from Greeting bring greet` öğenin ismini doğrudan kullanılabilir hale getirir.
- `from Greeting bring ( greet, farewell )` aynı anda birden fazla ismi farklı satırlarda okunaklı şekilde içe aktarmanızı sağlar.
- `from Greeting bring all` modüldeki gizli (`Confidential`) olmayan (public) tüm isimleri içe aktarır.

Aynı isimleri taşıyan çakışan içe aktarımlar (import collisions) ve döngüsel bağlılıklar (circular dependencies) derleme hatasıdır.


### Modül takma adları (Alias)

Bir modülü içe aktarırken `as` kullanarak o modülün ad alanına yeni bir takma ad verebilirsiniz. Bu, kodunuzu daha kısa tutmak için kullanışlıdır.

```ahd
bring Time as T

write(T.Calendar.isLeapYear(2028))
```

Takma ad, o içe aktarma işlemi için orijinal adın yerini alır. `bring Time as T` yazdığınızda sadece `T` ismini kullanabilirsiniz, otomatik olarak `Time` ismine de sahip olmazsınız. Eğer uzun halini tercih ederseniz normal `bring Time` kullanımı hala geçerlidir. Kendi yerel modüllerinize de takma ad verebilirsiniz.

Bu özellik tür (type) isimlendirmesinde kullanılamaz. Türleri kullanmak için eski tarzı koruyun ve türün adını doğrudan içe aktarın:

```ahd
bring Time as T
from Time bring DateTime

current: DateTime := T.now()
```

Tür olarak `T.DateTime` yazmayın; bu geçersizdir. Ayrıca, `from Time bring DateTime as DT` gibi tekil öğelere takma ad veremezsiniz.

### File ve Path temelleri

`Path`, dosya sistemine erişmeden yolları birleştirir ve inceler. `File`; UTF-8
metin okur/yazar, klasör oluşturur, metin ekler, siler, varlığı kontrol eder ve
bir klasördeki doğrudan isimleri kararlı sıralı biçimde listeler.

```ahd
bring Path
bring File
from File bring FileError

path := Path.join(["notlar", "bugun.txt"])
File.createDir("notlar")
File.writeText(path, "merhaba")

attempt {
    write(File.readText("olmayan.txt"))
}
except FileError as error {
    write("Dosya okunamadı")
}
```

`FileError`, `IOError` sınıfından türemiştir. Göreli yollar programın veya REPL
oturumunun çalışma klasörünü kullanır.

## 20. Temel işlevler modülü (Fundamentals)

Aşağıdaki isimler her modülde zaten hazır bulunur ve onları kullanmak için
`bring` ile içe aktarmanıza gerek yoktur. Bunlar standart girdi/çıktı, metin
dönüşümleri ve sayısal işlemleri kapsar.

```text
write take str int real len clear between abs sum min max
```

| Fonksiyon | Davranış |
|---|---|
| `write(value)` | bir değeri yazdırır ve ardından yeni bir satıra geçer |
| `take()` / `take(prompt)` | kullanıcıdan bir satır metin (String) okur |
| `str(value)` | nesnenin metin karşılığını üretir |
| `int(Real)` | ondalık kısmı kesip atar (sıfıra doğru yuvarlar) |
| `int(String)` | karakterleri sadece işareti ve sayıları olan bir tam sayıya çevirir |
| `real(Int)` | tam sayıyı ondalıklı sayıya çevirir (widening) |
| `real(String)` | ondalıklı sayı içeren metni dönüştürür |
| `len(value)` | String'deki karakter, List'teki eleman, Pair'deki çift sayısını verir |
| `clear(collection)` | List'i veya Pair'i yerinde tamamen boşaltır |
| `between(...)` | bitiş noktasını dışlayan anlık (lazy) bir sayı aralığı üretir |
| `abs(number)` | sayının mutlak değerini (magnitude) hesaplar |
| `sum(list)` | listedeki sayıları toplar; boş bir List `0` veya `0.0` verir |
| `min(list)` / `max(list)` | listedeki en küçük/büyük sayıyı bulur; List boşsa `DomainError` fırlatır |

`abs`, `sum`, `min` ve `max` hem `Int` hem de `Real` türleri üzerinde çalışır. `clear` mevcut koleksiyonu yerinde (in place) boşaltır, bu yüzden aynı koleksiyona işaret eden diğer tüm değişkenler (alias'lar) onu boş olarak görür. Sayısal hesaplama işlemleri (`sum`, `min`, `max`) List'i değiştirmez (pure reads), bu nedenle `Constant List` üzerinde de güvenle çalışırlar.

## 21. Matematik modülü (Math)

`Math` modülü gelişmiş matematiksel işlemler ve rastgele (random) sayı üretim fonksiyonları sunar. Kullanmadan önce açıkça içe aktarılmalıdır.

### Fonksiyonlar ve Sabitler

```ahd
bring Math

write(Math.PI)
write(Math.sqrt(81))
write(Math.round(3.14159, 2))
```

Beklenen çıktı:

```text
3.141592653589793
9.0
3.14
```

Kullanabileceğiniz tüm Math özellikleri şunlardır:

| Öğe | Açıklama |
|---|---|
| `PI`, `E` | Matematiksel sabitler. |
| `round`, `floor`, `ceil` | `round(değer, basamak)` tam buçuklu sayıları sıfırdan uzaklaşacak şekilde yuvarlar; basamak isteğe bağlıdır (0..15). `floor` ve `ceil` ise `Int` döndürür. |
| `sqrt`, `exp` | Karekök ve üstel fonksiyon ($e^x$). |
| `sin`, `cos`, `tan` | Radyan kullanan trigonometrik fonksiyonlar. |
| `log`, `log10` | Doğal logaritma ve 10 tabanında logaritma. |
| `seed`, `random`, `randomInt` | Rastgele sayı üretim fonksiyonları. |

Üs alma işlemi için `^` operatörünü kullanın, dilde `Math.pow` yoktur. Ayrıca `abs`, `sum`, `min` ve `max` gibi fonksiyonlar Math değil, Temel İşlevler (Fundamentals) modülündedir.

### Rastgele sayı durumu (Random state)

```ahd
bring Math

Math.seed(42)
write(Math.randomInt(1, 6))
write(Math.random())
```
`randomInt(min, max)` verilen **iki sınırı da içerir**. `random()` ise `0.0 <= value < 1.0` aralığında bir değer üretir.

Aynı tohum (seed) değerini yeniden verirseniz, aynı rastgele sayı dizisini tekrar elde edersiniz. Eğer bir tohum (`Math.seed`) vermezseniz, her yeni program çalışması başlangıç değerini işletim sisteminden (OS) alır. Bu rastgele sayı üreticisi, güvenlik veya şifreleme amacıyla kesinlikle **kullanılmamalıdır**.

`Math.random`, `Math.randomInt` ve `List.shuffle` işlemleri, programın
genelindeki bu tek, paylaşılan durumu tüketir. Sınırları aynı olan (örneğin
`randomInt(5, 5)`) çağrılar ile boş/tek öğeli bir List üzerinde yapılan `shuffle`
(karıştırma) işlemi rastgelelik durumunu tüketmez.

> **Teknik not:** Üretilen dizi sözde rastgeledir (pseudo-random). Tohum
> (seed) verilmemiş bir çalışmada başlangıç değeri işletim sisteminin
> entropisinden alınır.


## 22. Zaman modülü (Time)

`Time` modülü programınızın saati okumasını, tarihlerle çalışmasını ve beklemesini sağlar. `Math` gibi bu modül de açıkça içe aktarılmalıdır.

AhdCode'da `Time.DateTime` biçiminde tip yazımı yoktur; bu yüzden adını kullanacağınız tipleri içe aktarırsınız:

```ahd
bring Time
from Time bring DateTime
from Time bring Duration
```

### Şu anki tarih ve saat

```ahd
bring Time
from Time bring DateTime

current: DateTime := Time.now()

write(current.year)
write(current.month)
write(current.day)
```

`Time.now()` bilgisayarınızın **yerel** saatini verir. Sürüm 0.1'de saat dilimi özelliği hiç yoktur, bu yüzden ayarlanacak bir şey de yoktur.

Bir `DateTime` içinde okuyabileceğiniz sekiz bilgi vardır:

```text
year  month  day  hour  minute  second  millisecond  weekday
```

`weekday` Pazartesi'den başlar:

| gün | sayı |
|---|---|
| Pazartesi | 1 |
| Salı | 2 |
| Çarşamba | 3 |
| Perşembe | 4 |
| Cuma | 5 |
| Cumartesi | 6 |
| Pazar | 7 |

Bu değerler yalnızca okunur. `current.year = 2030` yazmak, herhangi bir `Constant` değeri değiştirmek gibi hatadır.

### Kendi tarihinizi oluşturmak

```ahd
bring Time
from Time bring DateTime

birthday: DateTime := Time.dateTime(
    year: 2028,
    month: 2,
    day: 29
)

write(birthday.toString())
```

Beklenen çıktı:

```text
2028-02-29 00:00:00
```

`hour`, `minute`, `second` ve `millisecond` isteğe bağlıdır ve `0` ile başlar.

Var olmayan bir tarih isterseniz `ValueError` alırsınız. 2026 artık yıl olmadığı için `2026-02-29` gerçek bir gün değildir; AhdCode bunu sessizce 1 Mart'a kaydırmak yerine reddeder.

`toString()` her dilde ve her bilgisayarda hep `YYYY-MM-DD HH:MM:SS` biçiminde yazar.

### İki anı karşılaştırmak

AhdCode tarihlerde `<` ve `>` kullanmaz. Soruyu kelimelerle sorarsınız:

```ahd
bring Time
from Time bring DateTime

morning: DateTime := Time.dateTime(year: 2026, month: 1, day: 1, hour: 9)
evening: DateTime := Time.dateTime(year: 2026, month: 1, day: 1, hour: 21)

write(morning.before(evening))
write(morning.after(evening))
write(morning.sameMoment(morning))
```

Beklenen çıktı:

```text
true
false
true
```

İki tarihin aynı anı gösterip göstermediğini sormak için `sameMoment` kullanın. `==` farklı bir şey sorar: aynı nesne olup olmadıklarını. Bu, her Sınıf (Class) için geçerli olan normal kuraldır.

### Aralarında ne kadar zaman var

```ahd
bring Time
from Time bring DateTime
from Time bring Duration

first: DateTime := Time.dateTime(year: 2026, month: 1, day: 1)
second: DateTime := Time.dateTime(year: 2026, month: 1, day: 2)

gap: Duration := Time.between(first, second)

write(gap.milliseconds)
write(gap.seconds)
```

Beklenen çıktı:

```text
86400000
86400.0
```

`Time.between(first, second)` "second eksi first" demektir. İkisinin yerini değiştirirseniz negatif bir `Duration` elde edersiniz; bir şeyin geçmişte kaldığını anlamak için bu kullanışlıdır.

`Time.duration(milliseconds: 1500)` ile doğrudan bir `Duration` da oluşturabilirsiniz.

### Takvime soru sormak

Bazen belirli bir tarihi değil, doğrudan takvimi merak edersiniz:

```ahd
bring Time

write(Time.Calendar.isLeapYear(2028))
write(Time.Calendar.isLeapYear(2100))
write(Time.Calendar.daysInMonth(2028, 2))
write(Time.Calendar.weekday(2026, 8, 29))
```

Beklenen çıktı:

```text
true
false
29
6
```

Artık yıl 4'e bölünür, ancak `00` ile biten bir yılın 400'e de bölünmesi gerekir. 2000 artık yıldır, 2100 değildir.

### Ölçmek ve beklemek

```ahd
bring Time

start: Real := Time.monotonic()

Time.sleep(500)

elapsed: Real := Time.monotonic() - start

write(elapsed >= 0.5)
```

Birimlere dikkat edin, bilerek farklıdır:

- `Time.sleep(...)` **milisaniye** cinsinden bekler.
- `Time.monotonic()` **saniye** cinsinden değer verir.

`Time.monotonic()` tek başına bir tarih değildir ve kendi başına bir anlam taşımaz. Yalnızca iki okuma arasındaki fark olarak işe yarar; bir işin ne kadar sürdüğünü ölçmek için istediğiniz tam olarak budur.

`Time.sleep(0)` hemen geri döner. Negatif bir bekleme `ValueError` verir, çünkü "eksi bir milisaniye" beklemek gerçek bir istek değildir.


## 23. Latex modülü (Latex)

`Latex` standart kütüphanesi, doğrudan AhdCode programlarınızdan PDF belgeleri oluşturmanızı sağlar. Kendi içinde gömülü, çevrimdışı çalışan bir Tectonic motoru kullanır; bu nedenle TeX Live, MiKTeX veya başka bir dış yazılım kurmanıza gerek yoktur. Güvenlik amacıyla sistem kabuğuna erişim (shell escape) kasıtlı olarak devre dışı bırakılmıştır.

Modül, belgeleri güvenli bir şekilde oluşturmak için pratik ve yeni başlayanlara uygun bir API sunar:

```ahd
bring Latex as L

document: String := L.document(
    L.section("İlk Belgem") +
    L.escape("Merhaba! Bu sıradan bir metin bölümüdür.") +
    L.subsection("Matematik Örneği") +
    L.equation("E = mc^2")
)

attempt {
    L.pdfFile("cikti.pdf", document)
    write("PDF başarıyla oluşturuldu!")
}
except LatexError as error {
    write("PDF oluşturulamadı: {error.message}")
}
```

- `Latex.escape(text)`: Sıradan metinleri, LaTeX kodu olarak algılanmaması için güvenli bir şekilde kaçış (escape) dizisine dönüştürür.
- `Latex.section(text)` ve `Latex.subsection(text)`: Başlıklar oluşturur.
- `Latex.equation(math)`: Saf LaTeX matematik kodlarını kabul eder.
- `Latex.document(content)`: İçeriğinizi tam, derlenmeye hazır bir belgenin içine sarar.
- `Latex.pdf(document)`: Belgeyi derler ve PDF verisini (byte olarak) döndürür.
- `Latex.pdfFile(path, document)`: Belgeyi derler ve doğrudan bir dosyaya kaydeder.
- `LatexError`: Derleme başarısız olursa (örneğin matematiğinizde bir sözdizimi hatası varsa) ortaya çıkar.

## 24. Kod biçimlendirici (Formatter)

Programınızdaki boşluklar veya satır düzeni dağınık olsa bile kodunuz
çalışabilir. Ancak kod biçimlendirici (formatter), yazdığınız yorum satırlarını koruyarak dosyanızı AhdCode'un ortak standart stiline göre yeniden düzenler:

```bash
ahdcode format hello.ahd
ahdcode format --check hello.ahd
```

İlk komut dosyayı doğrudan düzenleyerek günceller (bu işlem "idempotent"tir: tekrar tekrar çalıştırsanız da aynı sonucu verir ve fazladan değişiklik yapmaz). İkinci komut ise sadece dosyanın stilini kontrol eder, hiçbir şeyi değiştirmez. Bu komut, ekip projelerinde kod stilinin düzgün olduğundan emin olmak için kullanışlıdır.

### Geçerli sözdizimi ile önerilen stil

AhdCode'un grameri yazım konusunda esnektir: aynı satırdaki iki öğe arasında virgül zorunludur, ama başka her yerde -- bir Fonksiyonun parametreleri, bir çağrının argümanları, bir List veya Pair'in öğeleri arasında -- virgül yalnızca iki öğe aynı satırı paylaşıyorsa gereklidir, ve sondaki virgül her zaman isteğe bağlıdır. Aşağıdaki üç çağrı aynı anlama gelir:

```ahd
add(2, 3)

add(
    2,
    3
)

add(
    2
    3
)
```

Serbest olmayan tek yerleşim kuralı şudur: `:=` veya `=` işaretinden sonraki değer, işaretle aynı satırda başlamalıdır (bkz. [Language tour](LANGUAGE_TOUR.md) belgesindeki "Declarations and mutation" bölümü).

Stili elle seçmenize gerek yok -- hangisi rahatınıza geliyorsa onu yazın (ya da bir yapay zeka asistanının size verdiği hâliyle bırakın), ardından `ahdcode format` çalıştırın. Sonuç her zaman aynıdır: kısa yapılar tek satıra toplanır, sığmayanlar ise sonunda virgül olmadan her öğe kendi satırında olacak şekilde bölünür. Örneğin:

```ahd
calculate: Function := (first: Int, second: Int, description: String, flag: Bool) -> Real {
    return first
}
```

şu hâle gelir:

```ahd
calculate: Function := (
    first: Int
    second: Int
    description: String
    flag: Bool
) -> Real {
    return first
}
```

kısa bir imza olan `check: Function := (x: Int) -> Bool { ... }` ise tek satırda kalır.

## 24. Komut satırı (CLI)

AhdCode basit bir komut satırı arayüzüyle gelir.

- `ahdcode run file.ahd`: Bir programı doğrudan çalıştırır.
- `ahdcode build file.ahd`: Programı, AhdCode derleyicisine ihtiyaç duymadan kendi başına çalışabilen yerel bir uygulamaya (native executable) dönüştürür.
- `ahdcode format file.ahd`: Dosyayı standart stile göre biçimlendirir.
- `ahdcode --help`: Tüm komutlar için yardım ekranını gösterir.
- `ahdcode --version`: Mevcut derleyici sürümünü gösterir.

## 25. Etkileşimli kabuk (REPL)

Küçük denemeler yapmak için `ahdcode` komutunu tek başına çalıştırarak
REPL'i (Oku-Değerlendir-Yazdır Döngüsü) açabilirsiniz. Bunun için bir `.ahd`
dosyasına ihtiyacınız yoktur.

REPL, dosya derleyicisi ile birebir aynı kuralları kullanır. Başarılı komutlar oturum (session) boyunca hafızada kalır. Başarısız bir komut son çalışan durumu silmez, bu yüzden rahatça tekrar deneyebilirsiniz.

```text
> x := 5
> x := 7
error: duplicate declaration
> x = 7
> x
7
```

REPL; değerleri, takma adları, Function ve Class tanımlarını, modülleri ve Math
rastgele sayı durumunu tek kalıcı oturumda tutar. Önceki komutlar yeniden
çalıştırılmaz. `take` gerçek bir cevap satırı okur; yerel modüller ile göreli
File yolları `ahdcode` komutunu başlattığınız klasörü kullanır.

## 26. Başlangıçta sık yapılan hatalar

İşte karşılaşabileceğiniz yaygın hatalar ve onları düzeltme yolları:

**1. Değişken tanımlamadan `=` kullanmak**
- Yanlış: `score = 10`
- Neden: Bir değişkene değer atamadan önce onu oluşturmalısınız.
- Doğru: `score: Int := 10`

**2. Aynı değişkeni iki kez tanımlamak (Duplicate declaration)**
- Yanlış: `score: Int := 10 \n score: Int := 20`
- Neden: `score` o blokta zaten mevcut.
- Doğru: `score: Int := 10 \n score = 20`

**3. İç bloklarda `Local` eksikliği**
- Yanlış: `if true { result: Int := 1 }`
- Neden: `if` gibi iç bloklarda oluşturulan yeni değişkenlerde `Local` gereklidir.
- Doğru: `if true { result: Local Int := 1 }`

**4. `for` döngüsünde yanlış `Local` kullanımı**
- Yanlış: `for item: Local Int in items`
- Neden: `for` değişkeni zaten tasarımsal olarak yereldir.
- Doğru: `for item: Int in items`

**5. Modül seviyesindeki değişken için `Global` eksikliği**
- Yanlış: `count: Int := 0 \n f: Function := () -> Nothing { count = 1 }`
- Neden: Modül kökündeki değişkeni değiştirmek için açıkça `Global` ile beyan etmelisiniz.
- Doğru: `f: Function := () -> Nothing { count: Global Int \n count = 1 }`

**6. Truthiness (Otomatik doğru/yanlış kabulü)**
- Yanlış: `if 1 { write("Evet") }`
- Neden: Koşullar kesinlikle `Bool` türünde olmalıdır.
- Doğru: `if 1 > 0 { write("Evet") }`

**7. Güvensiz `null` kullanımı**
- Yanlış: `name: String? := null \n write(name.upper())`
- Neden: `name` null olabilir ve bu bir çökmeye yol açabilir. Derleyici buna izin vermez.
- Doğru: `if name != null { write(name.upper()) }`

**8. Positional ve named (isimli) parametreleri karıştırmak**
- Yanlış: `greet("Ali", title: "Dr")`
- Neden: Ya tüm argümanları sırasıyla ya da hepsini isimleriyle kullanmalısınız.
- Doğru: `greet(name: "Ali", title: "Dr")`

**9. Overload (Aşırı yükleme) belirsizliği**
- Yanlış: Varsayılan (default) değerlere sahip hem `f(Int)` hem de `f(Real)` fonksiyonunuz varken, kodu sadece `f()` diye çağırmak.
- Neden: Derleyici hangisini kastettiğinizi tahmin edemez (ambiguous).
- Doğru: Eşleşmenin tam (exact) olması için çağrıda argümanları verin.

**10. List'e yanlış türde eleman koymak**
- Yanlış: `list: List<Int> := [1, 2.5]`
- Neden: `2.5` bir `Real`dir, `Int` değil.
- Doğru: `List<Real> := [1.0, 2.5]` kullanın veya `int(2.5)` ile dönüştürün.

**11. Sabiti (Constant) değiştirmeye çalışmak**
- Yanlış: `locked: Constant List<Int> := [1] \n locked[0] = 2`
- Neden: Sabitler yerinde değiştirilemez.
- Doğru: Eğer değiştirmeyi amaçlıyorsanız başındaki `Constant` kelimesini kaldırın.

**12. `between`'de sıfır adımı (Zero step)**
- Yanlış: `between(1, 10, 0)`
- Neden: Sıfır adımlı bir döngü sonsuza kadar devam eder, bu yüzden `between` bunu reddeder.
- Doğru: `between(1, 10, 1)`

**13. Geçersiz sayı dönüşümü**
- Yanlış: `int("3.14")`
- Neden: `int()` dönüşümü çok katıdır ve ondalık nokta kabul etmez.
- Doğru: `int(real("3.14"))`

**14. `Real` üzerinde `%` kullanmak**
- Yanlış: `5.5 % 2.0`
- Neden: `%` (kalan) operatörü sadece `Int` ile çalışır.
- Doğru: `Int` değerler kullanın, örneğin `5 % 2`.

**15. `Int` değişkenine bölme ataması yapmak**
- Yanlış: `count: Int := 4 \n count /= 2`
- Neden: `/` işlemi `Real` döndürür ve bir `Real` doğrudan `Int` değişkene atanamaz.
- Doğru: `count = int(count / 2)`

**16. Eksik modül dosyası**
- Yanlış: Aynı klasörde modül dosyası olmadığı halde `bring Greeting` yazmak. (`Math` gömülüdür, ancak sizin modülleriniz için dosya bulunmalıdır).
- Neden: Modüller içe aktarılan kodla aynı klasörde (kardeş dosya) olmalıdır.
- Doğru: `Greeting.ahd` dosyasının projenizde aynı klasörde olduğundan emin olun.

**17. `Override` kelimesinin yanlış kullanımı**
- Yanlış: Üst Sınıfta (parent Class) olmayan bir metoda `Override` yazmak.
- Neden: `Override` kesin olarak "üst sınıftaki mevcut bir metodu değiştiriyorum" demektir.
- Doğru: Eğer yepyeni bir metot ekliyorsanız `Override` kelimesini silin.

**18. Geçersiz `return`**
- Yanlış: `-> Nothing` yazan bir fonksiyonun içinde `return "Bitti"` kullanmak.
- Neden: Fonksiyon geriye "hiçbir şey" döndüreceğine söz vermişti.
- Doğru: Sadece yalın bir `return` kullanın.

**19. Metni (String) yerinde değiştirmeye çalışmak**
- Yanlış: `name[0] = 'B'`
- Neden: Metinler (Strings) değiştirilemezdir (immutable).
- Doğru: Yeni bir karakter değiştirmek için `replace` kullanın veya yeni bir metin oluşturun.

**20. Tekrarlanabilir olmayan (Unseeded) rastgele sayılar beklemek**
- Yanlış: `Math.seed(42)` kullanmadığınız halde `Math.randomInt(1,6)` kodunun her çalışmada aynı kalmasını beklemek.
- Neden: Tohum (seed) verilmemiş rastgelelik, OS entropisini kullanır ve tekrarlanamaz.
- Doğru: Zar atmadan önce `Math.seed(42)` gibi bir tohum değeri verin.

## 27. Küçük Projeler

Bu küçük projeler rehberde öğretilenleri bir araya getirir. Onları tek başınıza kurmayı deneyin!

1. **Not Ortalaması Hesaplayıcı**: Kullanıcıdan 5 not isteyin. Onları bir `List<Int>` içine koyun. Geçersiz notları (0'dan küçük veya 100'den büyük) listeden çıkarın (filter). Kalan notların ortalamasını, minimum ve maksimum değerini, son olarak da öğrencinin geçip (ortalama >= 50) geçmediğini yazdırın.
2. **Basit Hesap Makinesi**: İki sayı ve bir operatör (`+`, `-`, `*`, `/`) almak için `take()` kullanın. İşlemi seçmek için operatör üzerinde `state` kullanın ve sonucu yazdırın. Sıfıra bölünme ihtimalini `attempt`/`except` ile yönetin.
3. **Sayı İstatistikleri**: `Math.randomInt(1, 100)` ile 100 adet rastgele sayı üretin. Bunlardan kaç tanesinin tek, kaç tanesinin çift olduğunu sayın (count) ve listeyi sıralayın. Bir sayının asal olup olmadığını kontrol eden bir fonksiyon yazın ve listeyi sadece asalları gösterecek şekilde filtreleyin (filter).
4. **Kelime Analizi**: Kullanıcıdan bir cümle girmesini isteyin. Kelimeleri ayırmak için `split(" ")` kullanın. Kelime sayısını bulun, en uzun kelimeyi bulun ve her kelimenin kendi uzunluğuyla eşleştiği bir `Pair<String, Int>` oluşturun.
5. **Menülü Program**: Bir `until` döngüsü kullanarak küçük bir banka simülasyonu yapın. Bir menü gösterin: 1. Para Yatır, 2. Para Çek, 3. Bakiye, 0. Çıkış. Bakiyeyi bir `Int` içinde saklayın ve kullanıcı 0 girene kadar programı döndürün.
6. **Sınıflarla (Class) Öğrenci Kaydı**: Bir `Student` sınıfı ve bir `Course` (Kurs) sınıfı oluşturun. Course içinde bir `List<Student>` bulunsun. Kursa yeni bir öğrenci eklemek için bir metot, kursun genel not ortalamasını hesaplamak için başka bir metot yazın.
7. **Tohumlu (Seeded) Rastgele Oyun**: `Math.seed(42)` kullanarak 1 ile 100 arasında "gizli bir sayı" üretin. Kullanıcıdan sayıyı tahmin etmesini isteyin. Doğru tahmin edene kadar "daha yüksek" veya "daha düşük" diye yönlendirin. Tohum kullanıldığı için, gizli sayı programı her çalıştırdığınızda aynı olacaktır—test yapmak için mükemmel!

## 28. Alıştırmalar

Tam çözümleri hemen aramak yerine her programı küçük adımlarla kurun.

### Başlangıç Seviyesi
1. Kullanıcıdan adını ve yaşını alın; gelecek yıl kaç yaşında olacağını yazdırın.
2. Santigrat değerini `Real` olarak okuyup Fahrenheit karşılığını hesaplayın (`C * 9/5 + 32`).
3. Bir `Int` okuyun ve `%` kullanarak tek veya çift olduğunu yazdırın.
4. `until` ile en az bir kez görünen ve kullanıcı `0` girince duran küçük bir menü kurun.
5. Çevresindeki boşluğu temizleyip adı küçük harfe dönüştüren, sonra ilk karakteri büyüten bir Fonksiyon yazın.
6. Boş bir `List<Int>` oluşturun, içine 3 sayı ekleyin ve `sum` (toplam) ile `len` (uzunluk) yazdırın.
7. `between(10, 0, -1)` üzerinde dönerek geriye doğru bir sayaç yazdırın.

### Orta Seviye
8. Bir cümle okuyun ve içindeki tüm boşlukları alt çizgi (underscore) ile değiştirin.
9. Bir not List'inde `min` ve `max` sonuçlarını gösterin; ancak List'in boş olma ihtimalini önceden kontrol edin.
10. Bir `Pair` kullanarak isimleri notlara bağlayın, bir notu güncelleyin ve tüm kayıtları yazdırın.
11. `Math.seed(42)` fonksiyonunu çağırın, `randomInt(1, 6)` ile on kez zar atın. Programı yeniden çalıştırdığınızda tamamen aynı 10 sonucu aldığınızı doğrulayın.
12. Bir List'teki tüm sayıların karesini almak için `map` kullanın.
13. `name` ve `Constant number` özellikleri olan bir `Student` sınıfı (Class) ve bir özet döndüren metot yazın.
14. Bir sayının faktöriyelini hesaplayan öz yinelemeli (recursive) bir fonksiyon yazın.

### İleri Seviye (Challenge)
15. Kullanıcı girdiğinde ortaya çıkabilecek geçersiz sayı harflerini `int()` ile dönüştürürken `DomainError` almayı önlemek için `attempt` kullanın.
16. Bir `List<String>` listesindeki metinleri, uzunluklarına (length) göre sıralamak için bir `keyFunction` kullanıp `sort` çağrısı yapın.
17. Bir `Shape` (Şekil) üst Sınıfı (parent Class) oluşturun, sonra alan hesaplamasını yapan `Override` bir metoda sahip bir `Circle` (Daire) alt sınıfı oluşturun.
18. Parametre olarak bir `String` alan ve metnin içindeki her karakterin kaç defa geçtiğini sayan bir `Pair` döndüren bir fonksiyon yazın.
19. `break` ve `continue` kullanarak çok geniş bir aralık içindeki ilk 5 çift sayıyı bulun, ancak 3'e bölünenleri `continue` ile atlayın.
20. Dikdörtgen alanı hesaplayan bir fonksiyona sahip `MathUtils.ahd` adında bir modül oluşturun ve bir `main.ahd` içinden `bring` ile çağırarak kullanın.

## 29. Çözüm İpuçları

1. `take` sonucu String'dir; yaş için `int(...)` ve yeni yaş için `+ 1` kullanın.
2. Formülü küçük parçalara ayırın; `real(take(...))` ile başlayın ve Real sayılarını kullanın.
3. `value % 2 == 0` bir `Bool` üretir.
4. `until` post-check (sonradan kontrol) olduğu için menü yazısını gövdenin başına koyabilirsiniz.
5. `trim`, `lower` ve `capitalize` işlemlerini tek bir dönüş (return) satırında zincirlemeyi deneyin.
6. Boş `List<Int>` için türü açıkça yazın; her girdiyi `add` ile ekleyin.
7. Negatif adımlar geriye doğru sayar; `between`'in stop (bitiş) değerini içermediğini unutmayın.
8. `String.replace(" ", "_")` kullanın.
9. `min` ve `max` boş List'te `DomainError` üretir; önce `len(grades) > 0` kontrolü yapın.
10. `Pair<String, Int>` kullanın; Pair üzerinde `for` anahtarları ekleme sırasıyla verir.
11. Tohumu (seed) zarları atmadan hemen önce bir kez ayarlayın. Sınırlar bitişi içerdiği için doğrudan `1, 6` kullanabilirsiniz.
12. Sizin yazdığınız callback fonksiyonu `value * value` döndürmelidir.
13. Başlangıçtaki sınıf örneğini `structure: Attributes` için model olarak alın.
14. Rekürsif (öz yinelemeli) fonksiyonun taban şartı `n <= 1` olmalı ve 1 döndürmelidir.
15. `int(take())` kısmını `attempt` içine koyun ve `except DomainError` yakalayın.
16. Anahtar (key) fonksiyonunuz bir `String` parametresi almalı ve `len(value)` döndürmelidir.
17. Alan için `Math.PI * (yarıçap ^ 2)` kullanın.
18. String'in içindeki her harfi döngüye alın, Pair'in içinde var mı diye kontrol edip sayısını 1 artırın.
19. `if i % 3 == 0 { continue }`. `if count == 5 { break }`.
20. `from MathUtils bring alanHesapla` kullanabilirsiniz.

## 30. Sonraki adımlar ve teknik belgeler

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
