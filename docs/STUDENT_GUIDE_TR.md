# AhdCode v0.1 Türkçe Öğrenci Rehberi

Bu rehber, **daha önce hiç programlama yapmamış birinin de takip edebilmesi** için hazırlanmıştır. Baştan sona sırayla okuyabilirsiniz; her bölümde önce ne yapmak istediğimizi görecek, sonra çalışan bir örnek yazacak, en son gerekli kuralları öğreneceksiniz.

Kodların yanında zaman zaman İngilizce teknik terimler de göreceksiniz. Bunları ilk okumada ezberlemeniz gerekmez. Bir kutu **Teknik not** diye başlıyorsa, programı çalıştırabilmek için o ayrıntıyı hemen bilmek zorunda değilsiniz.

En iyi öğrenme yolu, örnekleri yalnızca okumak değil çalıştırmaktır. Bir örneği kopyalayın, içindeki sayıyı veya metni değiştirin ve sonucun nasıl değiştiğine bakın.

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

Bir programlama dili, bilgisayara ne yapmasını istediğimizi anlatmanın bir yoludur. AhdCode ile örneğin bir hesaplama yaptırabilir, kullanıcıdan bilgi alabilir, dosya okuyabilir veya kendi küçük programlarınızı oluşturabilirsiniz.

İlk AhdCode programınız tek satır bile olabilir:

```ahd
write("Merhaba!")
```

Bu program ekrana şunu yazar:

```text
Merhaba!
```

AhdCode, programı çalıştırmadan önce yazdığınız kodu kontrol eder. Örneğin bir metni sayı gibi kullanmaya çalışırsanız veya `null` olabilecek bir değeri kontrol etmeden kullanırsanız, mümkün olduğunda hatayı daha program başlamadan söyler. Ama başlangıçta bunun ayrıntılarını düşünmeniz gerekmiyor; ilerleyen bölümlerde örneklerle göreceğiz.

AhdCode v0.1 hâlâ gelişen bir sürümdür. Küçük komut satırı programlarını doğrudan çalıştırabilir veya onları tek başına çalışan yerel uygulamalara dönüştürebilirsiniz.

> **Teknik not:** Program çalışmadan önce türlerin kontrol edilmesine *static checking* denir.

## 2. Kurulum ve ilk programınız

AhdCode'u kaynak kodundan kurmak için bilgisayarınızda Go 1.25 veya daha yeni bir sürüm bulunmalıdır. Proje klasöründe şu komutları çalıştırın:

```bash
cd AhdCode
go test ./...
go install ./cmd/ahdcode
export PATH="$(go env GOPATH)/bin:$PATH"
ahdcode --version
```

Son komut AhdCode sürümünü gösteriyorsa hazırsınız.

Şimdi `hello.ahd` adında bir dosya oluşturun ve içine şunu yazın:

```ahd
name: String := "AhdCode"
write("Merhaba {name}")
```

Dosyayı çalıştırın:

```bash
ahdcode run hello.ahd
```

Beklenen çıktı:

```text
Merhaba AhdCode
```

Burada `name` adında küçük bir bilgi sakladık ve sonra onu yazının içine yerleştirdik. Şimdilik `:=` işaretinin neden kullanıldığını ezberlemenize gerek yok; bir sonraki bölümde öğreneceğiz.

**Siz deneyin:** `AhdCode` yerine kendi adınızı yazıp programı tekrar çalıştırın.

## 3. Kod yazımının temelleri

AhdCode programları `.ahd` uzantılı dosyalara yazılır. Her işlem genellikle kendi satırında durur; satır sonuna `;` koymanız gerekmez.

### Bilgisayarın okumayacağı notlar: yorumlar

Kodun içine kendiniz için açıklama bırakabilirsiniz. `//` ile başlayan bölüm o satırda yorumdur:

```ahd
// Bu satır sadece bizi bilgilendirir.
write("Bu satır çalışır")
```

Daha uzun yorumlar `/*` ile başlayıp `*/` ile bitebilir:

```ahd
/*
Bu açıklama birkaç
satır sürebilir.
*/
write("Devam")
```

### Bir bilgiyi saklamak: değişken

Programın daha sonra kullanacağı bir bilgiyi bir isim altında saklayabiliriz:

```ahd
name: String := "Ayşe"
age: Int := 19
```

Burada `name` ve `age` birer **değişkendir**. Sağ taraftaki değerleri daha sonra bu isimlerle kullanabiliriz:

```ahd
name: String := "Ayşe"
age: Int := 19

write(name)
write(age)
```

### `:=` yeni bir değişken oluşturur, `=` var olanı değiştirir

İlk kez oluştururken `:=` kullanın:

```ahd
score: Int := 10
```

Daha sonra aynı değişkenin değerini değiştirmek için `=` kullanın:

```ahd
score: Int := 10
write(score)

score = 20
write(score)
```

Çıktı:

```text
10
20
```

Henüz oluşturulmamış bir değişkene doğrudan `score = 10` yazamazsınız. Aynı yerde ikinci kez `score := ...` yazmak da yeni bir değişken oluşturmaya çalışmak anlamına gelir ve hatadır.

### Türü açıkça belirtmek zorunludur

AhdCode'da yeni bir değişken oluştururken türünü mutlaka açıkça yazmalısınız. Bu kural, programınızın daha güvenli çalışmasını sağlar ve hataları henüz kodu çalıştırırken bulmanıza yardımcı olur:

```ahd
age: Int := 19
name: String := "Ayşe"
active: Bool := true
```

AhdCode türü bir kez belirledikten sonra değişkeni başka türde bir değere çeviremezsiniz:

```ahd
name: String := "Ayşe"
// name = 5   // HATA: name bir String olarak oluşturuldu
```

İç bloklarda kullanacağımız `Local` kuralını 11. bölümde ayrıca öğreneceğiz; şimdilik dosyanın en üst seviyesindeki örneklere odaklanın.

> **Teknik not:** Derleyicinin başlangıç değerine bakıp türü bulmasına *type inference* denir. Bu, değişkenin çalışma sırasında istediği türe dönüşebileceği anlamına gelmez; AhdCode statik türleri korur.

## 4. Temel türler

Bir değişkende ne tür bilgi tuttuğumuzu belirtmek için **türler** kullanılır. İlk aşamada şu türleri tanımanız yeterlidir:

| Tür | Ne tutar? | Örnek |
|---|---|---|
| `Int` | Tam sayı | `42`, `-3` |
| `Real` | Ondalıklı sayı | `3.5`, `-0.25` |
| `String` | Metin | `"Ayşe"` |
| `Bool` | Doğru/yanlış değeri | `true`, `false` |
| `List<T>` | Sıralı değerler | `[1, 2, 3]` |
| `Pair<K, V>` | Anahtar-değer eşleşmeleri | `{"Ali": 90}` |
| `Function` | Çalıştırılabilir bir fonksiyon | ileride göreceğiz |
| `Class` | Kendi veri yapınızı tanımlar | ileride göreceğiz |
| `Nothing` | Bir fonksiyonun değer döndürmediğini söyler | ileride göreceğiz |

Basit bir örnek:

```ahd
student: String := "Ayşe"
age: Int := 19
average: Real := 87.5
passed: Bool := average >= 50.0

write("{student}, {age}, {average}, {passed}")
```

Çıktı:

```text
Ayşe, 19, 87.5, true
```

Burada `passed` değişkenine doğrudan `true` yazmadık. `average >= 50.0` karşılaştırmasının sonucu zaten doğru veya yanlış olduğu için AhdCode onu `Bool` olarak anladı.

Bazı durumlarda `Int` güvenli biçimde `Real` gereken yerde kullanılabilir. Fakat koleksiyonların türleri birbirinden ayrıdır; örneğin `List<Int>` ile `List<Real>` aynı tür değildir.

Bir değerin bazen hiç bulunmaması gerekiyorsa türün sonuna `?` eklenebilir. Örneğin `String?`, "bir String veya `null`" anlamına gelir. Bunu 16. bölümde örneklerle ele alacağız.

> **Teknik not:** `List<Int>` ile `List<Real>` gibi generic türlerin birbirine otomatik dönüşmemesi *invariance* olarak adlandırılır.

## 5. Operatörler

Operatörler sayılarla işlem yapmak, değerleri karşılaştırmak veya koşulları birleştirmek için kullandığımız işaret ve kelimelerdir.

### Dört işlem ve diğer sayısal işlemler

```ahd
write(10 + 5)   // 15
write(10 - 5)   // 5
write(10 * 5)   // 50
write(10 / 4)   // 2.5
write(10 % 3)   // 1
write(2 ^ 3)    // 8
```

`/` işlemi her zaman `Real` sonuç verir. Bu yüzden `5 / 2` sonucu `2` değil `2.5` olur.

`%` yalnızca `Int` değerlerle kullanılır ve kalanını verir. `^` üs alma işlemidir.

AhdCode tam sayı hesaplarında taşmayı kontrol eder. Bir sonuç `Int` sınırlarını aşarsa yanlış bir sayı üretmek yerine `OverflowError` verir.

### Var olan değeri kısa yoldan güncellemek

```ahd
score: Int := 10
score += 5
write(score) // 15
```

`+=`, `-=`, `*=`, `/=`, `%=`, `^=` biçimleri vardır. `/` sonucu `Real` olduğu için `Int` bir değişkende `/=` kullanamazsınız.

Bir `Int` değeri yalnızca bir artırmak veya azaltmak için:

```ahd
count: Int := 0
count++
count++
count--
write(count) // 1
```

`++` ve `--` kendi satırlarında tek başına kullanılır.

### Değerleri karşılaştırmak

```ahd
age: Int := 20

write(age == 20) // true
write(age != 18) // true
write(age > 18)  // true
write(age <= 30) // true
```

Temel karşılaştırmalar: `==`, `!=`, `<`, `<=`, `>`, `>=`.

### Bir değerin bir yerde bulunup bulunmadığını sormak

`in` ve `not in` çok okunaklıdır:

```ahd
numbers: List<Int> := [10, 20, 30]
write(20 in numbers)      // true
write(99 not in numbers)  // true

text: String := "AhdCode"
write("Code" in text)     // true
```

`Pair` üzerinde `in`, değerleri değil anahtarları arar:

```ahd
scores: Pair<String, Int> := {
    "Ali": 90
    "Ayşe": 95
}

write("Ali" in scores) // true
```

### `and`, `or`, `not`

Birden fazla doğru/yanlış koşulunu birleştirebilirsiniz:

```ahd
age: Int := 20
hasTicket: Bool := true

if age >= 18 and hasTicket {
    write("Hoş geldiniz!")
}
```

- `and`: iki taraf da doğruysa `true`
- `or`: en az bir taraf doğruysa `true`
- `not`: sonucu tersine çevirir

### `same`, `is` ve `has` ne zaman lazım olacak?

Bunlar biraz daha ileride işimize yarar:

- `same`: iki değişkenin **aynı nesneyi** gösterip göstermediğini sorar.
- `is` / `is not`: bir nesnenin belirli bir Sınıftan olup olmadığını sorar.
- `has` / `has not`: bir Sınıf nesnesinde belirli bir özellik veya metot bulunup bulunmadığını sorar.

Bu kavramları List referansları ve Sınıflar bölümlerinde örneklerle tekrar göreceksiniz. İlk okumada ezberlemeniz gerekmez.

## 6. Metinler (Strings)

Metin tutmak için `String` kullanılır:

```ahd
name: String := "Ayşe"
city: String := 'Hatay'
```

Tek tırnak ve çift tırnak kullanılabilir. Birkaç satırlık metin için üçlü tırnak kullanabilirsiniz:

```ahd
poem: String := """
Güller kırmızı,
Menekşeler mavi.
"""
```

### Bir değişkeni metnin içine koymak

Süslü parantez kullanın:

```ahd
name: String := "Ali"
age: Int := 20
write("{name} {age} yaşında")
```

Çıktı:

```text
Ali 20 yaşında
```

Buna *interpolation* denir.

### Sık kullanılan String işlemleri

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
write(clean.endsWith("Ayşe"))
write(clean.count("i"))
```

`String` değerleri yerinde değişmez. Örneğin `clean.upper()` yeni bir String üretir; `clean` değişkeninin kendisini değiştirmez.

### Bir karaktere ulaşmak

İlk karakterin numarası `0`'dır:

```ahd
word: String := "AhdCode"
write(len(word)) // 7
write(word[0])   // A
write(word[-1])  // e
```

Negatif indeksler sondan sayar; `-1` son karakterdir. Geçersiz bir indeks `IndexError` üretir. `String.index(...)` aradığını bulamazsa `DomainError` üretir.

### Kaçış karakterleri

Metnin içinde özel karakterler yazmak için `\` kullanılabilir. Örneğin `\n` yeni satır, `\"` ise çift tırnak karakteridir.

> **Teknik not:** AhdCode String'leri Unicode karakterleriyle çalışır ve immutable'dır; yani yerinde değiştirilemez.

## 7. Girdi, çıktı ve dönüşümler

Şimdi programımız kullanıcıyla konuşsun.

### Ekrana yazmak: `write`

```ahd
write("Merhaba")
write(42)
```

`write(...)` değeri ekrana yazar ve yeni satıra geçer.

### Kullanıcıdan bilgi almak: `take`

```ahd
name: String := take("İsim: ")
write("Merhaba {name}")
```

Örnek kullanım:

```text
İsim: Ali
Merhaba Ali
```

`take()` her zaman **String** döndürür. Kullanıcı `20` yazsa bile ilk anda elimizde sayı değil, `"20"` metni vardır.

Yaşla matematik yapmak istiyorsak onu sayıya dönüştürürüz:

```ahd
age: Int := int(take("Yaş: "))
write("Gelecek yıl {age + 1} yaşında olacaksınız.")
```

### Tür dönüşümleri

```ahd
write(int(3.7))       // 3
write(int(" +42 "))  // 42
write(real(2))        // 2.0
write(real("1e3"))   // 1000.0
write(str(true))      // true
```

- `int(...)`: `Int` üretir.
- `real(...)`: `Real` üretir.
- `str(...)`: String üretir.

AhdCode metni kendiliğinden sayıya çevirmez. Kullanıcı geçersiz bir sayı yazarsa örneğin `int("merhaba")`, `DomainError` oluşur. Çok büyük değerlerde `OverflowError` oluşabilir. Hataları nasıl yakalayacağımızı 18. bölümde göreceğiz.

> **Ayrıntı:** `int(String)` yalnızca işaret ve rakamlardan oluşan tam sayı metinlerini kabul eder; `"3.14"` gibi bir metni doğrudan kabul etmez. `real(String)` ondalıklı ve üslü gösterimleri kabul eder fakat `NaN` ve sonsuzluğu kabul etmez.

## 8. Koşullar: `if` ve `state`

Programların her zaman aynı şeyi yapması gerekmez. Bir koşula göre farklı davranmasını sağlayabiliriz.

### `if` ve `else`

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

Çıktı:

```text
Geçti
```

`if` sonrasında mutlaka `true` veya `false` üreten bir ifade olmalıdır. Bu yüzden:

```ahd
// if score { ... }  // geçersiz
```

yerine:

```ahd
if score > 0 {
    write("Pozitif")
}
```

yazarız. AhdCode'da `0`, boş metin veya boş liste otomatik olarak doğru/yanlış kabul edilmez.

### Aynı değeri birçok seçenekle karşılaştırmak: `state`

Bir menü seçimi gibi durumlarda `state` okunaklı olabilir:

```ahd
choice: Int := 2

state choice {
    condition 1 {
        write("Yeni oyun")
    }
    condition 2 {
        write("Ayarlar")
    }
    condition default {
        write("Bilinmeyen seçim")
    }
}
```

İlk eşleşen `condition` çalışır. Hiçbiri eşleşmezse `condition default` çalışır. Bir sonraki seçeneğe otomatik geçiş olmadığı için `break` gerekmez.

## 9. Döngüler: `while`, `until` ve `for`

Döngü, aynı işi tekrar tekrar yaptırır.

### `while`: koşul doğru olduğu sürece devam et

```ahd
count: Int := 1

while count <= 3 {
    write(count)
    count++
}
```

Çıktı:

```text
1
2
3
```

`while` önce koşula bakar. Koşul başlangıçta yanlışsa gövde hiç çalışmayabilir.

### `until`: en az bir kez çalıştır, sonra durma koşuluna bak

```ahd
count: Int := 0

until count == 3 {
    count++
    write(count)
}
```

Çıktı:

```text
1
2
3
```

`until` önce gövdeyi çalıştırır, sonra koşulu kontrol eder. Bu yüzden gövde **en az bir kez** çalışır. Koşul `true` olduğunda döngü biter.

### `for`: bir listedeki veya aralıktaki değerleri sırayla kullan

```ahd
for value in [10, 20, 30] {
    write(value)
}
```

ve sayı aralığı için:

```ahd
for value in between(1, 6, 2) {
    write(value)
}
```

Çıktı:

```text
1
3
5
```

`between(start, stop)` başlangıcı içerir, bitişi içermez. Üçüncü değer adım miktarıdır. Negatif adımla geriye gidebilirsiniz; `0` adım `DomainError` üretir.

`for` değişkeninin türünü genellikle yazmanız gerekmez:

```ahd
for value in [10, 20, 30] {
    write(value)
}
```

İsterseniz açıkça da yazabilirsiniz:

```ahd
for value: Int in [10, 20, 30] {
    write(value)
}
```

`for` değişkeni zaten o döngüye ait kabul edilir; başına `Local` eklenmez.

### Döngüyü erken bitirmek veya bir turu atlamak

```ahd
for value in between(1, 10) {
    if value == 2 {
        continue
    }
    if value == 5 {
        break
    }
    write(value)
}
```

- `continue`: o turun geri kalanını atlar.
- `break`: döngüyü tamamen bitirir.

> **Teknik not:** `while` pre-check, `until` post-check bir döngüdür. `between` bütün sayıları önceden bir List'e doldurmaz; değerleri ihtiyaç oldukça üretir. List üzerinde `for` başladığında iterasyon bir snapshot üzerinden ilerler.

## 10. Fonksiyon yazmak ve çağırmak

Aynı kodu tekrar tekrar yazmak yerine ona bir isim verip istediğimiz zaman çalıştırabiliriz. Buna **fonksiyon** denir.

Önce en küçük örnek:

```ahd
square: Function := (number: Int) -> Int {
    return number * number
}

write(square(5))
```

Çıktı:

```text
25
```

Bu satırı parçalara ayıralım:

```text
square: Function := (number: Int) -> Int
^^^^^^^              ^^^^^^^^^^^     ^^^
fonksiyon adı         aldığı değer    döndürdüğü tür
```

`number` fonksiyonun içine gönderdiğimiz değerin adıdır. `return`, fonksiyonun sonucunu geri verir.

### Birden fazla parametre

```ahd
add: Function := (a: Int, b: Int) -> Int {
    return a + b
}

write(add(2, 3))
```

Parametreleri alt alta da yazabilirsiniz:

```ahd
greet: Function := (
    name: String
    title: String := "Öğrenci"
) -> String {
    return "Merhaba {title} {name}"
}
```

`title` için başlangıç değeri verdiğimiz için çağrıda yazmak zorunda değiliz:

```ahd
write(greet("Ali"))
write(greet(name: "Ayşe", title: "Dr"))
```

Bir çağrıda ya bütün değerleri sırayla verin ya da hepsini isimleriyle verin. Şu ikisi geçerlidir:

```ahd
greet("Ali")
greet(name: "Ayşe", title: "Dr")
```

ama positional ve named biçimleri aynı çağrıda karıştırmayın.

### Sonuç döndürmeyen fonksiyon: `Nothing`

Sadece bir iş yapıp değer döndürmeyen fonksiyon `Nothing` kullanır:

```ahd
sayHello: Function := (name: String) -> Nothing {
    write("Merhaba {name}")
}

sayHello("Ali")
```

Böyle bir fonksiyonun sonuna `return` yazmak zorunda değilsiniz. Ama erken çıkmak isterseniz yalın `return` kullanabilirsiniz:

```ahd
showScore: Function := (score: Int) -> Nothing {
    if score < 0 {
        write("Geçersiz not")
        return
    }

    write("Not: {score}")
}
```

### Fonksiyonun kendisini çağırması

Bir fonksiyon kendisini tekrar çağırabilir. Örneğin:

```ahd
countdown: Function := (n: Int) -> Nothing {
    if n <= 0 {
        write("Ateşle!")
        return
    }

    write(n)
    countdown(n - 1)
}

countdown(3)
```

Buna **recursion (öz yineleme)** denir. Mutlaka duracağı bir koşul bulunmalıdır.

### Aynı isimde farklı parametrelerle fonksiyonlar

Daha ileri bir kullanım olarak aynı adı farklı parametre türleriyle kullanabilirsiniz:

```ahd
describe: Function := (value: Int) -> String {
    return "Int {value}"
}

describe: Overload Function := (value: Real) -> String {
    return "Real {value}"
}
```

AhdCode çağrının hangi sürüme ait olduğunu parametrelerden belirler. Tam eşleşme önce gelir; güvenli `Int -> Real` genişletmesi gerektiğinde kullanılabilir. Hangi sürümün seçileceği belirsizse derleyici tahmin etmek yerine hata verir.

> **Teknik not:** Bu seçime *overload resolution* denir. v0.1'de fonksiyonların adı olmalıdır; nested function ve lambda yoktur. Fonksiyon parametreleri ve dönüş türü açıkça yazılır. Fonksiyon bildirimi `name: Function := (...) -> T { ... }` biçimini korur.

## 11. `Local` ve `Global`

Bu bölüm ilk bakışta biraz farklı gelebilir. Mantığı aslında basit: AhdCode bir değişkenin **nerede yaşadığını** açıkça görmenizi ister.

### Fonksiyonun parametresi zaten fonksiyonun içindedir

```ahd
greet: Function := (name: String) -> Nothing {
    write(name)
}
```

Buradaki `name` fonksiyonun parametresidir. Onu ayrıca `Local` diye tanımlamayız; fonksiyonun içine gönderildiği zaten bellidir.

### Fonksiyonun içinde yeni bir değişken oluşturuyorsanız `Local` yazın

```ahd
greet: Function := (name: String) -> Nothing {
    message: Local String := "Merhaba {name}"
    write(message)
}
```

`message` yalnızca bu fonksiyonun içinde kullanılmak üzere oluşturuldu. Bu yüzden `Local` yazdık.

Aynı kural `if`, `while` gibi iç bloklarda oluşturduğunuz yeni değişkenler için de geçerlidir:

```ahd
if true {
    message: Local String := "Bu değişken bu bloğa ait"
    write(message)
}
```

### `for` değişkenine `Local` yazılmaz

`for` zaten kendi değişkenini oluşturur:

```ahd
for value in [10, 20, 30] {
    write(value)
}
```

Bu nedenle `for value: Local Int ...` yazmayın.

### Dışarıdaki bir değişkeni fonksiyonun içinde kullanmak: `Global`

Şimdi dosyanın en üstünde bir sayaç oluşturalım:

```ahd
counter: Int := 0
```

Bir fonksiyonun bu **aynı** sayacı değiştirmesini istiyorsak fonksiyon içinde `Global` ile bunu açıkça söyleriz:

```ahd
counter: Int := 0

increase: Function := () -> Nothing {
    counter: Global Int
    counter++
}

increase()
increase()
write(counter)
```

Çıktı:

```text
2
```

`counter: Global Int` yeni bir sayaç oluşturmaz. "Bu fonksiyonun dışındaki `counter` değişkenini kullanacağım" demektir.

İsterseniz modül kökündeki değişkenin türü belliyse mevcut kurallara göre yalnız kapsam niyetini belirterek de çalışabilirsiniz; öğrenci rehberinde açık türü görmek ilk aşamada daha okunaklı olduğu için örneklerde `Global Int` biçimini kullandık.

Kısacası:

| Durum | Ne yazılır? |
|---|---|
| Fonksiyon parametresi | Ekstra `Local` gerekmez |
| Fonksiyon / `if` / `while` içinde yeni değişken | `Local` |
| `for` değişkeni | `Local` yazılmaz |
| Dosya kökündeki değişkene içeriden erişim | `Global` |

> **Teknik not:** Bir ismin kodun hangi bölgesinde kullanılabildiğine *scope* (kapsam) denir. AhdCode'da `Local` ve `Global` tür değil, kapsam niyetidir.

## 12. Listelerle çalışmak (List)

Birden fazla değeri sırayla tutmak için `List` kullanabilirsiniz:

```ahd
numbers: List<Int> := [10, 20, 30]
write(numbers)
```

İlk elemanın indeksi `0`'dır:

```ahd
write(numbers[0])  // 10
write(numbers[-1]) // 30
```

### Eleman eklemek ve çıkarmak

```ahd
numbers: List<Int> := [10, 20]
numbers.add(30)
write(numbers)

numbers.eject(1)
write(numbers)
```

Çıktı:

```text
[10, 20, 30]
[10, 30]
```

`clear(numbers)` listenin tamamını boşaltır.

### Listenin bir bölümünü almak

```ahd
nums: List<Int> := [10, 20, 30, 40, 50]
part: List<Int> := nums[1:4]
write(part)
```

Çıktı:

```text
[20, 30, 40]
```

Bu işlem yeni bir List üretir.

### Sıralamak, ters çevirmek ve karıştırmak

```ahd
bring Math

values: List<Int> := [4, 1, 3, 2]
values.sort()
write(values)

values.reverse()
write(values)

Math.seed(42)
values.shuffle()
write(values)
```

`sort`, `reverse` ve `shuffle` mevcut List'i yerinde değiştirir.

### Aramak

```ahd
data: List<Int> := [7, 8, 7, 9]
write(data.count(7)) // 2
write(data.index(8)) // 1
```

`index()` aradığı değeri bulamazsa `DomainError` üretir.

### `map`, `filter` ve anahtara göre sıralama

Bir listedeki her değeri dönüştürmek için `map`, bazılarını seçmek için `filter` kullanabilirsiniz. v0.1'de lambda olmadığı için kullanacağınız işlemi önce isimli bir Function olarak yazarsınız:

```ahd
double: Function := (value: Int) -> Int {
    return value * 2
}

isEven: Function := (value: Int) -> Bool {
    return value % 2 == 0
}

values: List<Int> := [3, -1, 4, -2]
write(values.map(double))
write(values.filter(isEven))
```

`map` ve `filter` kaynak List'i değiştirmez; yeni List döndürür.

Bir sıralama anahtarı da verebilirsiniz:

```ahd
absSort: Function := (value: Int) -> Int {
    return abs(value)
}

values: List<Int> := [3, -1, 4, -2]
values.sort(absSort)
write(values)
```

Bu sıralama kararlıdır (stable).

## 13. Referans davranışı (Reference Behavior)

Şu örneğe bakın:

```ahd
numbers: List<Int> := [10, 20, 30]
alias: List<Int> := numbers

alias[0] = 99
write(numbers)
```

Çıktı:

```text
[99, 20, 30]
```

"Ben `alias` değişkenini değiştirdim, `numbers` neden değişti?" diye düşünebilirsiniz. Çünkü `alias := numbers` yazdığımızda ikinci bir List kopyası oluşturmadık. İki değişken de **aynı List'i** gösteriyor.

Bunu açıkça kontrol edebiliriz:

```ahd
write(numbers same alias) // true
```

`same`, iki değişkenin aynı nesneyi gösterip göstermediğini sorar.

`==` ise içeriklerinin eşit olup olmadığına bakar:

```ahd
a: List<Int> := [1, 2]
b: List<Int> := [1, 2]

write(a == b)    // true: içerikleri aynı
write(a same b)  // false: iki ayrı List
```

Bu fark özellikle List, Pair ve Sınıf nesnelerini değiştirirken önemlidir.

> **Teknik not:** Aynı nesneyi birden fazla isimle kullanmaya *aliasing*, bu davranışa *reference semantics* denir.

## 14. Pair ile çalışmak

Bir öğrenci adını notuyla, bir ürün kodunu fiyatıyla veya bir ayarı değeriyle eşleştirmek istediğinizde `Pair` kullanabilirsiniz.

```ahd
scores: Pair<String, Int> := {
    "Ali": 85
    "Ayşe": 92
}
```

Burada `"Ali"` anahtar, `85` ise o anahtara bağlı değerdir.

### Değeri okumak ve değiştirmek

```ahd
write(scores["Ali"])

scores["Ali"] = 90
scores["Veli"] = 78

write(scores["Ali"])
```

Olmayan bir anahtarı okumaya çalışırsanız `KeyError` oluşur.

### Pair üzerinde dolaşmak

```ahd
for name in scores {
    write("{name}: {scores[name]}")
}
```

`for`, Pair'in anahtarlarını ekleme sırasıyla verir.

Bir anahtarı silmek için `eject(key)`, bütün Pair'i boşaltmak için `clear(pair)` kullanılır.

`Pair` de List gibi referans davranışı gösterir: iki değişken aynı Pair'i gösteriyorsa birinden yapılan değişiklik diğerinden de görülür.

v0.1'de Pair anahtarları `String`, `Int` veya `Bool` olabilir.

## 15. Sabitler (Constant)

Bazen bir değerin oluşturulduktan sonra değiştirilmesini istemezsiniz. `Constant` bunu açıkça belirtir:

```ahd
locked: Constant List<Int> := [1, 2, 3]
```

Artık bu List'i değiştirmeye çalışmak hatadır:

```ahd
// locked[0] = 99  // HATA
```

`Constant` yalnızca dıştaki List'i değil, onun içinden ulaşılabilen nesneleri de korur. Yani sabit bir yapının içinde başka List, Pair veya Sınıf nesneleri varsa onlar da dondurulur.

Bu yüzden aynı nesneye başka bir değişken üzerinden ulaşsanız bile onu değiştirmeye çalışmak çalışma sırasında `ConstantError` oluşturabilir.

```ahd
values: List<Int> := [1, 2, 3]
locked: Constant List<Int> := values

// values.add(4) çalışma sırasında ConstantError oluşturabilir;
// çünkü aynı List artık deep-frozen durumdadır.
```

Bir `Constant` başlangıçta `null` olamaz.

> **Teknik not:** Ulaşılabilir bütün nesne ağının dondurulmasına *deep freeze* denir.

## 16. Null güvenliği

Bazen bir değerin henüz bulunmaması normaldir. Örneğin bir kullanıcı arıyoruz ama kayıt yok:

```text
"Ali" bulundu  -> bir User var
kayıt yok      -> değer yok
```

AhdCode'da `null`, "burada şu anda bir değer yok" demektir.

Normal bir değişkene `null` atayabilirsiniz ancak AhdCode onu kullanmanıza hemen izin vermez:

```ahd
name: String := null
```

Bu, `name` değişkeninin bir `String` veya `null` tutabileceğini söyler.

### Kullanmadan önce kontrol edin

```ahd
message: String := null

if message == null {
    message = "hazır"
}

if message != null {
    write(message.upper())
}
```

Derleyici `message != null` kontrolünden sonra bu bloğun içinde gerçekten bir String bulunduğunu bilir.

Kontrol etmeden şunu yazmak geçersizdir:

```ahd
message: String := null
// write(message.upper()) // HATA: message null olabilir
```

### `null` tek başına türünü söylemez

Şu kullanım geçersizdir:

```ahd
// value := null // HATA: tür belirtilmeli
```

Çünkü AhdCode bunun `String`, `User` veya başka hangi tür olduğunu bilemez. Türü belirtin:

```ahd
value: String := null
```

Eğer bir fonksiyon `User` döndürüyor ama sonuç `null` olabiliyorsa:

```ahd
user: User := fetchUser()
```

### Koleksiyonlarda null kullanımı

Koleksiyonlarda, örneğin `List<User>`, listenin kendisi `null` olabileceği gibi, liste geçerli bir nesne olup içindeki elemanlar `null` olabilir. Bu ayrım AhdCode tarafından tamamen aynı akış analizi (null refinement) mantığı ile yönetilir.

> **Teknik not:** Derleyicinin bir kontrol sonrasında değer hakkında daha kesin bilgi edinmesine *null refinement* denir. Belgelendirmelerde akış durumları `Null`, `MaybeNull` ve `NonNull` olarak adlandırılabilir.

## 17. Sınıflar (Class) ve Özellikler (Attributes)

Bir öğrenciyi yalnızca adıyla değil, adı ve numarasıyla birlikte tutmak istediğinizi düşünün. Bu iki bilgiyi sürekli ayrı değişkenlerde taşımak yerine kendi veri yapınızı tanımlayabilirsiniz. AhdCode'da bunun için `Class` kullanılır.

### İlk Sınıfımız

```ahd
Student: Class<> := {
    structure: Attributes := (
        name: String
        number: Int
    )
}
```

Bu tanım, her `Student` nesnesinin `name` ve `number` bilgileri olacağını söyler.

Şimdi bir öğrenci oluşturalım:

```ahd
student: Student := Student(name: "Ali", number: 42)
```

Sınıfın içinden bu özelliklere `attribute` ile ulaşılır. Bir metot ekleyelim:

```ahd
Student: Class<> := {
    structure: Attributes := (
        name: String
        number: Constant Int
    )

    describe: Function := () -> String {
        return "#{attribute.number} {attribute.name}"
    }
}

student: Student := Student(name: "Ali", number: 42)
write(student.describe())
```

Çıktı:

```text
#42 Ali
```

`number: Constant Int` olduğu için öğrenci oluşturulduktan sonra numara değiştirilemez.

Bir `structure` girdisinin başına `Local` yazılırsa yalnız nesne oluşturulurken kullanılabilir; nesnenin kalıcı özelliği olmaz. `Confidential` ise üyeye sınıfın dışından normal erişimi engeller.

### Bir Sınıfı başka bir Sınıftan genişletmek

Önce genel bir `Person` tanımlayalım:

```ahd
Person: Class<> := {
    structure: Attributes := (
        name: String
    )

    describe: Function := () -> String {
        return "Kişi {attribute.name}"
    }
}
```

`Student`, `Person`'ın sahip olduklarını devralabilir:

```ahd
Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Int
    )

    describe: Override Function := () -> String {
        return "{SuperClass.describe()} #{attribute.number}"
    }
}
```

- `Class<Person>`: `Student`, `Person`'dan türetilmiştir.
- `SuperClass.attributes`: üst sınıfın oluşturma girdilerini getirir.
- `Override`: üst sınıftaki bir metodu bilerek değiştirdiğimizi söyler.
- `SuperClass.describe()`: üst sınıfın kendi sürümünü çağırır.

```ahd
student: Student := Student(name: "Ayşe", number: 7)
person: Person := student
write(person.describe())
```

Nesne gerçekte `Student` olduğu için `Student.describe()` çalışır.

Gerçek türü sormak için:

```ahd
if person is Student {
    write("Bu kişi bir öğrenci!")
}
```

Bir üyeye sahip olup olmadığını sormak için `has` kullanılabilir:

```ahd
write(person has name)
```

> **Teknik not:** Alt sınıf nesnesini üst sınıf türünde tutmaya *upcasting*, çağrılacak metodun gerçek nesne türüne göre seçilmesine *dynamic dispatch* denir. İlk okumada bu terimleri ezberlemek zorunda değilsiniz.

## 18. Hata yönetimi (`attempt`, `except`, `ultimately` ve `toss`)

Bazı hatalar programı yazarken değil, program çalışırken ortaya çıkabilir. Örneğin kullanıcıdan sayı beklerken `abc` yazabilir.

```ahd
age: Int := int(take("Yaş: "))
```

Kullanıcı sayı yazarsa sorun yoktur. Geçersiz bir metin yazarsa `DomainError` oluşur. Programın bu durumda ne yapacağını biz belirleyebiliriz.

### Bir hatayı yakalamak: `attempt` ve `except`

```ahd
attempt {
    age: Local Int := int(take("Yaş: "))
    write("Yaşınız: {age}")
}
except DomainError as error {
    write("Lütfen geçerli bir tam sayı yazın.")
}
```

`attempt` içindeki kod denenir. Belirtilen hata oluşursa uygun `except` bloğu çalışır.

Birden fazla hata türü için birden fazla `except` yazabilirsiniz:

```ahd
attempt {
    // hata oluşturabilecek işlemler
}
except DomainError as error {
    write("Sayı geçersiz")
}
except IndexError as error {
    write("İndeks geçersiz")
}
```

### Ne olursa olsun çalışacak bölüm: `ultimately`

```ahd
attempt {
    write("İşlem deneniyor")
}
except DomainError as error {
    write("Hata oluştu")
}
ultimately {
    write("Bu satır her durumda çalışır")
}
```

### Kendi isteğimizle hata üretmek: `toss`

```ahd
requirePositive: Function := (value: Int) -> Int {
    if value <= 0 {
        toss (DomainError("değer pozitif olmalı"))
    }

    return value
}
```

AhdCode'un sık karşılaşılan hata türleri arasında `DomainError`, `ValueError`, `IndexError`, `KeyError`, `OverflowError`, `DivisionByZeroError`, `NullError` ve `ConstantError` bulunur.

### Kendi hata türünüzü oluşturmak

İhtiyaç olduğunda `Error` sınıfından yeni hata türü türetebilirsiniz:

```ahd
InvalidAgeError: Class<Error> := {
    structure: Attributes := (
        message: String
    )
}
```

ve sonra:

```ahd
attempt {
    age: Local Int := -5
    if age < 0 {
        toss (InvalidAgeError("Yaş negatif olamaz"))
    }
}
except InvalidAgeError as error {
    write(error.message)
}
```

> **Teknik not:** AhdCode runtime hataları yakalanabilir normal Class değerleri olarak modellenir.

## 19. Modüller ve `bring`

Program büyüdükçe her şeyi tek bir dosyaya yazmak istemezsiniz. Bir işi ayrı `.ahd` dosyasına koyup başka dosyadan kullanabilirsiniz. Buna **modül** diyebiliriz.

### Kendi modülünüzü oluşturmak

Aynı klasörde iki dosya olduğunu düşünün:

```text
main.ahd
Greeting.ahd
```

`Greeting.ahd`:

```ahd
greet: Function := (name: String) -> String {
    return "Modülden merhaba, {name}"
}
```

`main.ahd`:

```ahd
from Greeting bring greet

write(greet("Ayşe"))
```

Çıktı:

```text
Modülden merhaba, Ayşe
```

### Modülü hangi biçimde içe aktarabilirim?

Bir namespace olarak:

```ahd
bring Greeting
write(Greeting.greet("Ayşe"))
```

Yalnız istediğiniz ismi alarak:

```ahd
from Greeting bring greet
write(greet("Ayşe"))
```

Birden fazla isim:

```ahd
from Greeting bring (
    greet
    farewell
)
```

Tüm public isimler:

```ahd
from Greeting bring all
```

Aynı isimlerin çakışmasına yol açan importlar ve döngüsel modül bağımlılıkları derleme hatasıdır.

### Modüle kısa bir ad vermek

```ahd
bring Time as T

write(T.Calendar.isLeapYear(2028))
```

`as T` kullandığınızda bu import için `Time` yerine `T` kullanırsınız.

Bu kısaltma tür adlarında kullanılmaz. Örneğin:

```ahd
bring Time as T
from Time bring DateTime

current: DateTime := T.now()
```

`T.DateTime` bir tür yazımı değildir; türü ayrıca içe aktarın.

### File ve Path'e ilk bakış

AhdCode'un hazır `Path` ve `File` modülleri de `bring` ile kullanılır:

```ahd
bring Path
bring File

path: String := Path.join(["notlar", "bugun.txt"])
File.createDir("notlar")
File.writeText(path, "merhaba")
write(File.readText(path))
```

`Path` yol metinleriyle çalışır. `File` dosya ve klasör işlemleri yapar. Dosya işlemlerinde hata yakalamak istiyorsanız `FileError` türünü ayrıca içe aktarabilirsiniz:

```ahd
from File bring FileError

attempt {
    write(File.readText("olmayan.txt"))
}
except FileError as error {
    write("Dosya okunamadı")
}
```

`FileError`, `IOError` sınıfından türemiştir. Göreli yollar programın veya REPL oturumunun çalışma klasörünü kullanır.

## 20. Temel işlevler modülü (Fundamentals)

Bazı araçları kullanmak için hiçbir `bring` yazmanız gerekmez. Bunlar AhdCode programında doğrudan hazırdır:

```text
write take str int real len clear between abs sum min max
```

Çoğunu zaten kullandık. Kısa bir özet:

| Fonksiyon | Ne yapar? |
|---|---|
| `write(value)` | değeri ekrana yazar |
| `take()` / `take(prompt)` | kullanıcıdan bir satır String okur |
| `str(value)` | değeri String'e çevirir |
| `int(...)` | uygun değeri `Int` yapar |
| `real(...)` | uygun değeri `Real` yapar |
| `len(value)` | String/List/Pair uzunluğunu verir |
| `clear(collection)` | List veya Pair'i yerinde boşaltır |
| `between(...)` | sayı aralığı üretir |
| `abs(number)` | mutlak değer verir |
| `sum(list)` | listedeki sayıları toplar |
| `min(list)` / `max(list)` | en küçük / en büyük değeri bulur |

Örnek:

```ahd
numbers: List<Int> := [3, -5, 10]

write(len(numbers))
write(sum(numbers))
write(min(numbers))
write(max(numbers))
write(abs(-8))
```

`sum` boş List için `0` veya `0.0` verir. `min` ve `max` ise boş List'te `DomainError` üretir.

`clear` mevcut koleksiyonu yerinde değiştirdiği için aynı koleksiyonu gösteren diğer alias'lar da boş hali görür. `sum`, `min` ve `max` yalnızca okur; List'i değiştirmez.

## 21. Matematik modülü (Math)

Karekök, trigonometrik fonksiyonlar veya rastgele sayı gibi araçlar için `Math` modülünü kullanın:

```ahd
bring Math

write(Math.PI)
write(Math.sqrt(81))
write(Math.round(3.14159, 2))
```

Çıktı:

```text
3.141592653589793
9.0
3.14
```

Sık kullanılanlar:

| Öğe | Ne yapar? |
|---|---|
| `PI`, `E` | matematik sabitleri |
| `round`, `floor`, `ceil` | yuvarlama işlemleri |
| `sqrt`, `exp` | karekök ve $e^x$ |
| `sin`, `cos`, `tan` | trigonometrik fonksiyonlar (radyan) |
| `log`, `log10` | doğal ve 10 tabanında logaritma |
| `seed`, `random`, `randomInt` | rastgele sayı üretimi |

Üs alma için `Math.pow` yerine dilin `^` operatörünü kullanın. `abs`, `sum`, `min`, `max` ise `Math` içinde değil, doğrudan hazır Fundamentals işlevleridir.

### Rastgele sayı üretmek

```ahd
bring Math

write(Math.randomInt(1, 6))
write(Math.random())
```

`randomInt(1, 6)` hem `1` hem `6` dahil olmak üzere bu aralıkta bir tam sayı üretir. `random()` ise `0.0 <= value < 1.0` aralığındadır.

Test sırasında aynı sonuç dizisini tekrar elde etmek istiyorsanız bir seed verebilirsiniz:

```ahd
Math.seed(42)
```

Aynı seed tekrar verilirse aynı rastgele sayı dizisi baştan başlar. Seed verilmezse yeni program çalışması başlangıç durumunu işletim sisteminden alır.

`Math.random`, `Math.randomInt` ve `List.shuffle` aynı paylaşılan rastgelelik durumunu kullanır; her çağrı diziyi ilerletir. `randomInt(5, 5)` ile boş/tek elemanlı `shuffle` rastgelelik durumunu tüketmez.

> **Dikkat:** Bu rastgele sayı üreticisini kriptografi veya güvenlik amacıyla kullanmayın.

## 22. Zaman modülü (Time)

Tarih, saat ve bekleme işlemleri için `Time` modülü kullanılır.

### Şu anki zamanı almak

```ahd
bring Time
from Time bring DateTime

current: DateTime := Time.now()

write(current.year)
write(current.month)
write(current.day)
write(current.hour)
```

`Time.now()` bilgisayarınızın yerel saatini verir. v0.1'de ayrıca saat dilimi yönetimi yoktur.

Bir `DateTime` içinde şu bilgiler bulunur:

```text
year  month  day  hour  minute  second  millisecond  weekday
```

`weekday` Pazartesi için `1`, Pazar için `7` değerini kullanır. Bu alanlar yalnızca okunur.

### Belirli bir tarih oluşturmak

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

Çıktı:

```text
2028-02-29 00:00:00
```

`hour`, `minute`, `second` ve `millisecond` verilmezse `0` olur. Geçersiz bir tarih `ValueError` üretir; örneğin AhdCode `2026-02-29` değerini sessizce başka güne çevirmek yerine reddeder.

### İki zamanı karşılaştırmak

```ahd
morning: DateTime := Time.dateTime(year: 2026, month: 1, day: 1, hour: 9)
evening: DateTime := Time.dateTime(year: 2026, month: 1, day: 1, hour: 21)

write(morning.before(evening))
write(morning.after(evening))
write(morning.sameMoment(morning))
```

Tarihler için `<` ve `>` yerine bu okunaklı metotlar kullanılır.

### İki zaman arasındaki süre

```ahd
from Time bring Duration

first: DateTime := Time.dateTime(year: 2026, month: 1, day: 1)
second: DateTime := Time.dateTime(year: 2026, month: 1, day: 2)

gap: Duration := Time.between(first, second)
write(gap.milliseconds)
write(gap.seconds)
```

`Time.between(first, second)` ikinci zamandan birincisini çıkarır. `Time.duration(milliseconds: 1500)` ile doğrudan süre oluşturabilirsiniz.

### Takvim bilgileri

```ahd
write(Time.Calendar.isLeapYear(2028))
write(Time.Calendar.daysInMonth(2028, 2))
write(Time.Calendar.weekday(2026, 8, 29))
```

### Bir işi bekletmek veya süresini ölçmek

```ahd
start: Real := Time.monotonic()
Time.sleep(500)
elapsed: Real := Time.monotonic() - start
write(elapsed >= 0.5)
```

`Time.sleep(...)` **milisaniye**, `Time.monotonic()` ise **saniye** kullanır. `Time.monotonic()` tarih değildir; iki ölçüm arasındaki süreyi hesaplamak için kullanılır. Negatif `sleep` değeri `ValueError` üretir.

## 23. Latex modülü (Latex)

AhdCode ile doğrudan PDF üretmek istiyorsanız `Latex` modülünü kullanabilirsiniz. Modül gerekli Tectonic motorunu kendi kurulumuyla birlikte getirir; ayrıca TeX Live veya MiKTeX kurmanız gerekmez.

İlk örnek:

```ahd
bring Latex as L
from Latex bring LatexError

document: String := L.document(
    L.section("İlk Belgem") +
    L.escape("Merhaba! Bu sıradan bir metin bölümüdür.") +
    L.subsection("Matematik Örneği") +
    L.equation("E = mc^2")
)

attempt {
    L.pdf(document, "cikti.pdf")
    write("PDF başarıyla oluşturuldu!")
}
except LatexError as error {
    write("PDF oluşturulamadı: {error.message}")
}
```

Burada sırasıyla:

- `Latex.escape(text)`: normal metni LaTeX özel karakterlerine karşı kaçışlı hale getirir.
- `Latex.section(text)` / `subsection(text)`: başlık üretir.
- `Latex.equation(math)`: matematik ifadesini LaTeX olarak ekler.
- `Latex.document(content)`: parçaları tam bir belgeye dönüştürür.
- `Latex.pdf(source, output)`: String olarak verdiğiniz LaTeX kaynağını derleyip PDF dosyasına yazar.
- `Latex.pdfFile(input, output)`: zaten var olan `.tex` dosyasını derleyip PDF üretir.

Hata türünü `except LatexError` içinde kullanacaksanız `from Latex bring LatexError` ile içe aktarın.

Modül çevrimdışı çalışacak biçimde tasarlanmıştır ve shell escape kapalıdır. Daha gelişmiş tablo ve LaTeX ayrıntıları için `docs/LATEX.md` belgesine bakabilirsiniz.

## 24. Kod biçimlendirici (Formatter)

Kod çalışsa bile herkes farklı boşluk ve satır düzeni kullanırsa okumak zorlaşır. AhdCode formatter, geçerli kodu ortak bir stile dönüştürür:

```bash
ahdcode format hello.ahd
```

Bu komut dosyayı düzenler. Yalnız kontrol etmek için:

```bash
ahdcode format --check hello.ahd
```

AhdCode yazarken her virgülü veya satır kırılımını elle mükemmel ayarlamak zorunda değilsiniz. Örneğin şu çağrıların üçü de geçerlidir:

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

Formatter kısa yapıları tek satırda tutar, uzun yapıları okunaklı şekilde böler.

Önemli bir yerleşim kuralı vardır: `:=` veya `=` işaretinden sonraki değer **aynı satırda başlamalıdır**.

Geçerli:

```ahd
values: List<Int> := [
    1
    2
    3
]
```

Geçersiz:

```text
values :=
    [1, 2, 3]
```

Formatter idempotent'tir; aynı dosyada tekrar çalıştırmak yeni değişiklik üretmez.

## 25. Komut satırı (CLI)

AhdCode'u terminalden birkaç temel komutla kullanabilirsiniz:

```text
ahdcode run file.ahd
```

Programı çalıştırır.

```text
ahdcode build file.ahd
```

Programı kendi başına çalışan yerel bir executable'a dönüştürür.

```text
ahdcode format file.ahd
```

Dosyayı ortak stile göre biçimlendirir.

```text
ahdcode --help
ahdcode --version
```

Yardım ve sürüm bilgisini gösterir.

Yeni başlıyorsanız çoğu zaman kullanacağınız komut `ahdcode run ...` olacaktır.

## 26. Etkileşimli kabuk (REPL)

Küçük bir şeyi denemek için her seferinde dosya oluşturmak zorunda değilsiniz. Terminalde yalnızca:

```bash
ahdcode
```

çalıştırın. REPL açılır ve AhdCode komutlarını tek tek deneyebilirsiniz:

```text
> x: Int := 5
> x = x + 1
> x
6
```

REPL bir **oturum** gibi davranır. Önceki başarılı komutlarda oluşturduğunuz değerleri hatırlar:

```text
> name: String := "Ali"
> write(name)
Ali
```

Bir komutta hata yapmanız önceki çalışan durumu silmez:

```text
> x: Int := 5
> x: Int := 7
error: duplicate declaration
> x
5
```

Önceki komutların yan etkileri yeniden çalıştırılmaz. Örneğin:

```text
> write("bir")
bir
> write("iki")
iki
```

ikinci komutta `bir` tekrar yazılmaz.

`take()` REPL içinde de gerçek kullanıcı girdisini bekler:

```text
> name: String := take("İsim: ")
İsim: Ali
> write(name)
Ali
```

Function ve Class tanımları, modüller, List/Pair nesneleri ve Math rastgelelik durumu oturum boyunca korunur. Yerel modüller ve göreli File yolları, `ahdcode` komutunu başlattığınız klasöre göre çözülür.

REPL öğrenirken çok kullanışlıdır: bir fikri hızlıca deneyip sonucu görebilirsiniz. Daha uzun programlarda `.ahd` dosyası kullanmak daha düzenlidir.

## 27. Başlangıçta sık yapılan hatalar

Hata mesajı görmek programlamanın normal bir parçasıdır. Çoğu hata, bilgisayarın ne istediğinizi anlayamadığını söyler. Aşağıdaki örnekler yeni başlayanların sık karşılaştığı durumları ve nasıl düzelteceğinizi gösterir:

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

## 28. Küçük Projeler

Bu küçük projeler rehberde öğretilenleri bir araya getirir. Onları tek başınıza kurmayı deneyin!

1. **Not Ortalaması Hesaplayıcı**: Kullanıcıdan 5 not isteyin. Onları bir `List<Int>` içine koyun. Geçersiz notları (0'dan küçük veya 100'den büyük) listeden çıkarın (filter). Kalan notların ortalamasını, minimum ve maksimum değerini, son olarak da öğrencinin geçip (ortalama >= 50) geçmediğini yazdırın.
2. **Basit Hesap Makinesi**: İki sayı ve bir operatör (`+`, `-`, `*`, `/`) almak için `take()` kullanın. İşlemi seçmek için operatör üzerinde `state` kullanın ve sonucu yazdırın. Sıfıra bölünme ihtimalini `attempt`/`except` ile yönetin.
3. **Sayı İstatistikleri**: `Math.randomInt(1, 100)` ile 100 adet rastgele sayı üretin. Bunlardan kaç tanesinin tek, kaç tanesinin çift olduğunu sayın (count) ve listeyi sıralayın. Bir sayının asal olup olmadığını kontrol eden bir fonksiyon yazın ve listeyi sadece asalları gösterecek şekilde filtreleyin (filter).
4. **Kelime Analizi**: Kullanıcıdan bir cümle girmesini isteyin. Kelimeleri ayırmak için `split(" ")` kullanın. Kelime sayısını bulun, en uzun kelimeyi bulun ve her kelimenin kendi uzunluğuyla eşleştiği bir `Pair<String, Int>` oluşturun.
5. **Menülü Program**: Bir `until` döngüsü kullanarak küçük bir banka simülasyonu yapın. Bir menü gösterin: 1. Para Yatır, 2. Para Çek, 3. Bakiye, 0. Çıkış. Bakiyeyi bir `Int` içinde saklayın ve kullanıcı 0 girene kadar programı döndürün.
6. **Sınıflarla (Class) Öğrenci Kaydı**: Bir `Student` sınıfı ve bir `Course` (Kurs) sınıfı oluşturun. Course içinde bir `List<Student>` bulunsun. Kursa yeni bir öğrenci eklemek için bir metot, kursun genel not ortalamasını hesaplamak için başka bir metot yazın.
7. **Tohumlu (Seeded) Rastgele Oyun**: `Math.seed(42)` kullanarak 1 ile 100 arasında "gizli bir sayı" üretin. Kullanıcıdan sayıyı tahmin etmesini isteyin. Doğru tahmin edene kadar "daha yüksek" veya "daha düşük" diye yönlendirin. Tohum kullanıldığı için, gizli sayı programı her çalıştırdığınızda aynı olacaktır—test yapmak için mükemmel!

## 29. Alıştırmalar

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

## 30. Çözüm İpuçları

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

## 31. Sonraki adımlar ve teknik belgeler

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
