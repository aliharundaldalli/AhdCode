# AhdCode v0.3.0 Türkçe Öğrenci Rehberi

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
- [16. Null güvenliği](#16-null-güvenliği)
- [17. Sınıflar (Class) ve Özellikler (Attributes)](#17-sınıflar-class-ve-özellikler-attributes)
- [18. Hata yönetimi (`attempt`, `except`, `ultimately` ve `toss`)](#18-hata-yönetimi-attempt-except-ultimately-ve-toss)
- [19. Modüller ve bring](#19-modüller-ve-bring)
- [20. Fundamentals modülü](#20-fundamentals-modülü)
- [21. Math modülü](#21-math-modülü)
- [22. Time modülü](#22-time-modülü)
- [23. Statistics modülü](#23-statistics-modülü)
- [24. Plot modülü](#24-plot-modülü)
- [25. Numeric modülü ve Complex](#25-numeric-modülü-ve-complex)
- [26. Latex modülü (Latex)](#26-latex-modülü-latex)
- [27. Word modülü](#27-word-modülü)
- [28. Excel modülü](#28-excel-modülü)
- [29. PDF modülü](#29-pdf-modülü)
- [30. Archive modülü](#30-archive-modülü)
- [31. JSON modülü](#31-json-modülü)
- [32. XML modülü](#32-xml-modülü)
- [33. Env modülü](#33-env-modülü)
- [34. Lists ve KeyValue modülleri](#34-lists-ve-keyvalue-modülleri)
- [35. SQLite: hatırlayan bir veritabanı](#35-sqlite-hatırlayan-bir-veritabanı)
- [36. Kod Biçimlendirici (Formatter)](#36-kod-biçimlendirici-formatter)
- [37. Komut satırı (CLI)](#37-komut-satırı-cli)
- [38. Etkileşimli kabuk (REPL)](#38-etkileşimli-kabuk-repl)
- [39. Sık yapılan başlangıç hataları](#39-sık-yapılan-başlangıç-hataları)
- [40. Küçük Projeler](#40-küçük-projeler)
- [41. Egzersizler](#41-egzersizler)
- [42. Çözüm İpuçları](#42-çözüm-i̇puçları)
- [43. Sonraki adımlar ve teknik belgeler](#43-sonraki-adımlar-ve-teknik-belgeler)

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

AhdCode v0.3.0 güncel sürümdür. Küçük komut satırı programlarını çalıştırabilir veya yerel executable uygulamalara derleyebilirsiniz; veriyi yerel bir SQLite veritabanında tutabilir ve dil sunucusunu (`ahdcode lsp`) VS Code gibi bir editörden kullanabilirsiniz. Bazı standart modüller, örneğin SQLite, derlenmiş uygulamanın yanında AhdCode'un sağladığı yardımcı çalışma zamanı bileşenlerini kullanabilir. v0.2.2 pratik günlük dil sunucusunu tamamladı; v0.3.0 gerçek uygulama geliştirmenin başlangıcıdır.

> **Teknik not:** Program çalışmadan önce türlerin kontrol edilmesine *static checking* denir.

## 2. Kurulum ve ilk programınız

AhdCode'u kaynak kodundan kurmak için bilgisayarınızda Go 1.25 veya daha yeni bir sürüm bulunmalıdır. Proje klasöründe şu komutları çalıştırın:

```bash
cd AhdCode
go test ./...
go install ./cmd/ahdcode ./cmd/ahdnumeric ./cmd/ahdplot ./cmd/ahdsqlite
export PATH="$(go env GOPATH)/bin:$PATH"
ahdcode --version
```

Eğer `Latex` modülünü kullanmayı planlıyorsanız, çevrimdışı (offline) Latex çalışma zamanını da hazırlamanız (stage) gerekir. Bu adım, sabitlenmiş kaynakları indirmek için bir defaya mahsus ağ bağlantısı kullanır:

```bash
go run ./tooling/latex/cmd/package-latex --output "$(go env GOPATH)"
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

### Türü açıkça yazmak: önerilen ama zorunlu olmayan bir alışkanlık

Yeni başlarken türü açıkça yazmanız önerilir; bu, kodunuzu okurken hangi değeri tuttuğunuzu hemen görmenizi sağlar ve hataları erkenden fark etmenize yardımcı olur:

```ahd
age: Int := 19
name: String := "Ayşe"
active: Bool := true
```

Ancak bu bir zorunluluk değildir. Başlangıç değeri türü açıkça belirtiyorsa, türü hiç yazmadan sadece `:=` kullanabilirsiniz; derleyici türü kendisi çıkarır:

```ahd
age := 19       // Int olarak çıkarılır
name := "Ayşe"  // String olarak çıkarılır
```

Bu iki yazım şekli tamamen eşdeğerdir. Bu rehberde okunabilirlik için genellikle türü açıkça yazacağız, ama ikisinden hangisini kullanacağınız size kalmış.

Tür ister açıkça yazılsın ister çıkarılsın, AhdCode türü bir kez belirledikten sonra değişkeni başka türde bir değere çeviremezsiniz:

```ahd
name: String := "Ayşe"
// name = 5   // HATA: name bir String olarak oluşturuldu
```

İç bloklarda kullanacağımız `Local` kuralını 11. bölümde ayrıca öğreneceğiz; şimdilik dosyanın en üst seviyesindeki örneklere odaklanın.

> **Teknik not:** Derleyicinin başlangıç değerine bakıp türü bulmasına *type inference* denir. Bu, değişkenin çalışma sırasında istediği türe dönüşebileceği anlamına gelmez; AhdCode statik türleri korur. `name = 5` yine hatadır, çünkü `name` bir kez `String` olarak çıkarılmıştır.

## 4. Temel türler

Bir değişkende ne tür bilgi tuttuğumuzu belirtmek için **türler** kullanılır. İlk aşamada şu türleri tanımanız yeterlidir:

| Tür | Ne tutar? | Örnek |
|---|---|---|
| `Int` | Tam sayı | `42`, `-3` |
| `Real` | Ondalıklı sayı | `3.5`, `-0.25` |
| `String` | Metin | `"Ayşe"` |
| `Bool` | Doğru/yanlış değeri | `true`, `false` |
| `Complex` | Karmaşık sayı | `2 + 3I` |
| `List<T>` | Sıralı değerler | `[1, 2, 3]` |
| `Pair<K, V>` | Anahtar-değer eşleşmeleri | `{"Ali": 90}` |
| `Function` | Çalıştırılabilir bir fonksiyon | ileride göreceğiz |
| `Class` | Kendi veri yapınızı tanımlar | ileride göreceğiz |
| `Nothing` | Fonksiyonun değer döndürmediğini belirtir | ileride göreceğiz |

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

### Ham (raw) String'ler

Normal bir String, `\` kaçışlarını ve `{...}` interpolasyonunu çözümler.
Bazen tam olarak yazdığınız metni istersiniz -- örneğin `{3}` gibi
niceleyicilerle dolu bir düzenli ifade (regex) deseni ya da ters eğik
çizgilerle dolu bir LaTeX kaynağı. Açılış tırnağının önüne `r` koyarak ham
bir String yazabilirsiniz; ne kaçışlar ne de interpolasyon işlenir:

```ahd
name: String := "Ali"

write(r"{name}")   // {name}, Ali değil -- interpolasyon yok
write(r"\n")       // bir satır sonu değil, iki karakter olarak \n

pattern: String := r"^MATH-[0-9]{3}$"
formula: String := r"\frac{x+1}{x-1}"
```

Ham çok satırlı string'ler de aynı şekilde çalışır: `r"""..."""` veya
`r'''...'''`. Kısaca:

```text
normal String = kaçışlar + interpolasyon
ham String    = ne kaçış ne de interpolasyon
```

Ham bir String hâlâ sıradan bir `String`'dir; ayrı bir ham tür yoktur.

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

> **Teknik not:** Bu seçime *overload resolution* denir. İsimli Function bildirimleri `name: Function := (...) -> T { ... }` biçimini korur ve iç içe yazılamaz. Tek ifadeli kısa işler için `lambda (x: Int) -> x > 0`, aynı `Function` türünde isimsiz bir değer oluşturur. Lambda parametreleri açıkça türlenir, dönüş türü ifadeden çıkarılır ve lambda'nın bloğu veya deyimleri yoktur.

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

Bir listedeki her değeri dönüştürmek için `map`, bazılarını seçmek için
`filter` kullanabilirsiniz. Birden fazla adım gerektiren işlemlerde isimli
Function kullanmak okunaklıdır:

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

Tek ifadeli kısa işlemlerde lambdayı doğrudan verebilirsiniz:

```ahd
values: List<Int> := [3, -1, 4, -2]
squares := values.map(lambda (value: Int) -> value^2)
positive := values.filter(lambda (value: Int) -> value > 0)
values.sort(lambda (value: Int) -> -value)

write(squares)
write(positive)
write(values)
```

Lambda ayrı bir tür değildir; var olan `Function` türünde bir değer oluşturur.
Her parametrenin statik türü açıkça yazılır, dönüş türü ifadeden çıkarılır ve
normal katı tipleme/null güvenliği kuralları geçerlidir. Lambda'nın bloğu veya
deyimleri yoktur.

Lambda kendi parametreleri dışındaki bir değişkeni kullanacaksa bu değişkeni
parametrelerden önce köşeli parantez içinde açıkça **yakalar**:

```ahd
scores: List<Int> := [35, 50, 72, 90]
minimum: Int := 50
passed := scores.filter(lambda [@minimum] (score: Int) -> score >= minimum)
write(passed)
```

- `#`, `Local` yakalamadır: lambda oluşturulduğu andaki değer kopyalanır.
- `@`, `Global` yakalamadır: canlı modül değişkenine bağlanır.

Dış bağımlılıklar kendiliğinden çıkarılmaz; kullanacağınız her yakalamayı
listelemelisiniz.

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

AhdCode'da `null`, "burada şu anda bir değer yok" demektir. Yalın `T` türü
null olamaz; değer `null` da olabilecekse türü `T?` yazarsınız. Null olmayan
bir `T`, güvenle `T?` olarak kullanılabilir; ters yön için önce kontrol gerekir.

```ahd
name: String? := null
```

Bu, `name` değişkeninin bir `String` veya `null` tutabileceğini söyler.
`name: String := null` ise geçersizdir; çünkü `String` null olamaz.

### Kullanmadan önce kontrol edin

```ahd
message: String? := null

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
message: String? := null
// write(message.upper()) // HATA: message null olabilir
```

### `null` tek başına türünü söylemez

Şu kullanım geçersizdir:

```ahd
// value := null // HATA: null temel türü belirleyemez
```

Çünkü AhdCode bunun `String`, `User` veya başka hangi tür olduğunu bilemez.
Açıkça yazılan türün kendisi de null olabilir olmalıdır; `value: String := null`
da geçersizdir. Doğrusu:

```ahd
value: String? := null
```

Bir arama fonksiyonu `User` veya `null` döndürebiliyorsa dönüş türü ve sonucu
alan değişken `User?` olur:

```ahd
user: User? := fetchUser()
```

`fetchUser()` gerçek bir `User` bulursa bu null olmayan değer güvenle `User?`
türüne genişler. `user` değerini `User` olarak kullanmadan önce null olmadığını
kontrol edin.

### Koleksiyonlarda null kullanımı

`?` tam olarak yazıldığı seviyeye uygulanır:

```text
List<User>    null olmayan List; User elemanları da null olamaz
List<User>?   kendisi null olabilen List; User elemanları null olamaz
List<User?>   null olmayan List; User elemanları null olabilir
List<User?>?  hem kendisi hem User elemanları null olabilir
```

Null olabilen bir List üzerinde indeks veya metot kullanmadan önce List'i
kontrol etmelisiniz. `List<User?>` içindeki her eleman da `User` olarak
kullanılmadan önce ayrı ayrı kontrol edilir.

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

### Sınıf Protokol Metotları

Kendi sınıfınızın `+`, `==`, `<` gibi operatörlerle çalışmasını isteyebilirsiniz. Bunun için AhdCode'da tam olarak on tane özel isimden biri kullanılır; bunlara **Sınıf Protokol Metotları** denir:

```text
CEqual CCompare CAdd CSubtract CMultiply CDivide CRemainder CPower CNegate CStr
```

Bu adlar yalnızca bir Class içindeki metot yuvasında özel anlam taşır. Başka
yerlerde sıradan tanımlayıcılardır; `C` harfinin kendisi ayrılmış değildir.

```ahd
Vector2: Class<> := {
    structure: Attributes := (
        x: Real
        y: Real
    )

    CEqual: Function := (
        other: Vector2
    ) -> Bool {
        return attribute.x == other.x and attribute.y == other.y
    }

    CAdd: Function := (
        other: Vector2
    ) -> Vector2 {
        return Vector2(x: attribute.x + other.x, y: attribute.y + other.y)
    }

    CNegate: Function := (
    ) -> Vector2 {
        return Vector2(x: -attribute.x, y: -attribute.y)
    }

    CStr: Function := (
    ) -> String {
        return "Vector2({attribute.x}, {attribute.y})"
    }
}

a: Vector2 := Vector2(x: 1.0, y: 2.0)
b: Vector2 := Vector2(x: 3.0, y: 4.0)

write(a + b)
write(-a)
write(a == b)
write(str(a))
```

Beklenen çıktı:

```text
Vector2(4.0, 6.0)
Vector2(-1.0, -2.0)
false
Vector2(1.0, 2.0)
```

Kısa eşleştirme şöyledir:

- `==` ve `!=` → `CEqual`; `!=`, `CEqual` sonucunun tersidir, ayrı bir `CNotEqual` yoktur.
- `<`, `<=`, `>` ve `>=` → `CCompare`; ayrı karşılaştırma protokolleri yoktur.
- `+ - * / % ^` → sırasıyla `CAdd CSubtract CMultiply CDivide CRemainder CPower`.
- Tekli `-` → `CNegate`.
- `str(object)` → `CStr`.

Protokol seçimi her zaman **sol taraftaki** değere göre yapılır. `vector + 3`
çalışıyorsa bu, `3 + vector` işleminin de çalışacağı anlamına gelmez; ters
operatör kuralı yoktur. Kalıtım ve `Override`, sıradan metotlarda olduğu gibi
çalışır. Tam kurallar için [Sınıf Protokol Metotları](PROTOCOLS_TR.md)
referansına bakın.

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

Farklı hata türleri için birden fazla `except` bloğu yazabilirsiniz:

```ahd
attempt {
    write([10, 20][5])
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

Gerektiğinde `Error` sınıfından yeni bir hata türü türetebilir, bu hatayı
`toss` ile fırlatıp kendi türüyle yakalayabilirsiniz:

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
    write(error.message)
}
```

> **Teknik not:** AhdCode çalışma zamanı hataları, yakalanabilen sıradan Class
> değerleri olarak modellenir.

## 19. Modüller ve `bring`

Program büyüdükçe her şeyi tek bir dosyaya yazmak istemezsiniz. Bir işi ayrı bir `.ahd` dosyasına koyup başka bir dosyadan kullanabilirsiniz. Buna **modül** diyebiliriz.

### Kendi modülünüzü oluşturmak

Aynı klasörde iki dosyanız olduğunu düşünün:

```text
main.ahd
Greeting.ahd
```

`Greeting.ahd`:

```ahd
greet: Function := (name: String) -> String {
    return "Hello from module, {name}"
}
```

`main.ahd`:

```ahd
from Greeting bring greet

write(greet("Ayşe"))
```

Çıktı:

```text
Hello from module, Ayşe
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

Birden çok isim:

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

Aynı ismin çakışmasına yol açan içe aktarımlar ve döngüsel modül bağımlılıkları derleme zamanı hatasıdır.

### Bir modüle kısa isim vermek

```ahd
bring Time as T

write(T.Calendar.isLeapYear(2028))
```

`as T` kullandığınızda, bu içe aktarım için `Time` yerine `T` kullanırsınız.

Bu kısaltma tür bildirimlerinde kullanılmaz. Örneğin:

```ahd
bring Time as T
from Time bring DateTime

current: DateTime := T.now()
```

`T.DateTime`, geçerli bir tür bildirim sözdizimi değildir; türü ayrıca içe aktarın.

### İlk bakış: File ve Path

Program bir süre sonra kapandıktan sonra da bir şeyi hatırlamak ister: bir not, küçük bir günlük, kullanıcının okunmasını istediği bir dosya. `Path` yol *String*'lerini kurar ve inceler. `File` gerçekten dosya sistemine gider. İki ayrı modül olmalarının nedeni budur — bir yol, `File` onu kullanana kadar yalnızca metindir.

Göreli yollar, programın veya REPL oturumunun çalışma klasörüne göre çözülür (`ahdcode`'u çalıştırdığınız klasör; `.ahd` dosyasının bulunduğu klasör olmak zorunda değildir).

Küçük bir not defteri iş akışı:

```ahd
bring Path
bring File

notesDir: String := "notes"
path: String := Path.join([notesDir, "bugun.txt"])

if not File.exists(notesDir) {
    File.createDir(notesDir)
}

File.writeText(path, "Süt al.\n")
write(File.readText(path))
write(Path.base(path))
write(File.exists(path))
```

Beklenen çıktı:

```text
Süt al.

bugun.txt
true
```

`Path.join` parçalardan yol kurar. `Path.base` son bileşendir (`bugun.txt`); `Path.dir` ve `Path.ext` diğer sık kullanılan incelemelerdir. `File.exists` eksik yolda hata fırlatmaz, `false` döner. `File.createDir` klasör oluşturur. `File.writeText` UTF-8 metin dosyasını yazar (veya üzerine yazar); `File.readText` geri okur. `File.append` dosyanın sonuna ekler.

Yol yokken *okumayı* istediğinizde bu bir `FileError`'dır (`IOError`'dan türer):

```ahd
bring File
from File bring FileError

attempt {
    write(File.readText("yok.txt"))
}
except FileError as error {
    write("Dosya okunamadı")
}
```

**Siz deneyin:** Not metnini değiştirip programı iki kez çalıştırın, sonra `writeText` yerine `append` kullanın ve dosyada iki satır görün.

Tüm işlem listesi [File ve Path referansındadır](FILESYSTEM_TR.md).

### İlk bakış: Regex

Düzenli ifade, metin şekillerini tarif eden küçük bir dildir: “bir veya daha fazla rakam”, “harfle başlar”. AhdCode'da bu tarifi bir kez `Pattern`'a derler, sonra String'lere soru sorarsınız.

```ahd
bring Regex
from Regex bring Pattern

digits: Pattern := Regex.compile("[0-9]+")

write(digits.matches("siparis #482"))
write(digits.find("siparis #482, urun #7"))
write(digits.findAll("siparis #482, urun #7"))
write(digits.replace("oda 12 ve oda 7", "N"))
write(digits.split("a12b34c"))
```

Beklenen çıktı:

```text
true
482
["482", "7"]
oda N ve oda N
["a", "b", "c"]
```

`Regex.compile`'ın ürettiği sınıfın adı `Pattern`'dır, `Regex` değil — `bring Regex` zaten modülü adlandırır; tür olarak yazmak için `from Regex bring Pattern` gerekir.

`matches`, desen metnin *herhangi bir yerinde* geçerse doğrudur (tüm String için `^` / `$` ile sınırlayın). `find` `String?` döndürür çünkü eşleşme olmayabilir:

```ahd
bring Regex
from Regex bring Pattern

digits: Pattern := Regex.compile("[0-9]+")
found: String? := digits.find("rakam yok")
if found == null {
    write("bulunamadi")
}
```

`findAll` tüm eşleşmeleri verir. `replace` **yeni** bir String döndürür (orijinal değişmez). `split` her eşleşmede metni keser. `groups` ilk eşleşmenin yakalanan gruplarını `List<String>?` olarak verir.

Geçersiz desen, `find` sırasında değil, `compile` anında `RegexError` fırlatır:

```ahd
bring Regex
from Regex bring RegexError

attempt {
    Regex.compile("(kapanmamis")
}
except RegexError as error {
    write("derlenemedi: {error.message}")
}
```

**Siz deneyin:** Üç harfli bir sözcüğü eşleyen bir desen derleyip `"kedi"` ve `"kediler"` üzerinde deneyin.

Ayrıntılar için [Regex modül referansına](REGEX_TR.md) bakın.

### İlk bakış: CSV

CSV, metin olarak elektronik tablodur: hücreler arasında virgül (veya başka ayırıcı), satırlar arasında yeni satır. `CSV` yalnızca **String** taşır. `"42"`nin `Int` veya `"2026-01-01"`in tarih olduğuna karar vermez — sütunun ne anlama geldiğini *siz* bildiğinizde dönüştürürsünüz.

İki ilk kullanım şekli:

- **satırlar** — `List<List<String>>`: her iç liste bir satırdır, başlık dahil
- **kayıtlar** — `List<Pair<String, String>>`: her Pair başlığı anahtar olarak kullanır

```ahd
bring CSV

text: String := "ad,yas\nAli,42\nMerve,19\n"

rows: List<List<String>> := CSV.parse(text)
records: List<Pair<String, String>> := CSV.parseRecords(text)

write(rows[1][0])
write(records[0]["ad"])

ages: List<Int> := []
for record in records {
    ages.add(int(record["yas"]))
}
write(sum(ages))
```

Beklenen çıktı:

```text
Ali
Ali
61
```

Bozuk tırnaklama, kötü ayırıcı veya başlıkla uyuşmayan kayıt `CSVError` fırlatır. Dosya için `CSV.read` / `CSV.write`, metne dönüş için `stringify` / `stringifyRecords` vardır.

**Siz deneyin:** CSV metnine üçüncü bir kişi ekleyip o kaydın yaşını `int(...)` ile yazdırın.

[CSV modül referansına](CSV_TR.md) bakın.

### İlk bakış: Data tabloları

Metin girdikten sonra `Data` size bir `Table` verir: adlı sütunlar, süzüp şekil verebileceğiniz satırlar. Her hücre hâlâ `String`'dir. Her dönüşüm **yeni** bir tablo döndürür — elinizdeki tablo değişmez.

```ahd
bring Data
from Data bring Table

table: Table := Data.fromCSV("ad,puan,sehir\nAli,91,Adana\nAyse,78,Ankara\nDeniz,85,Adana\n")

write(table.rowCount())
write(table.columns())

passed: Table := table.filter(
    lambda (row: Pair<String, String>) -> int(row["puan"]) >= 80
)
namesOnly: Table := passed.select(["ad", "puan"])

write(namesOnly.column("ad"))
write(table.rowCount())
```

Beklenen çıktı:

```text
3
["ad", "puan", "sehir"]
["Ali", "Deniz"]
3
```

Son satır önemli: `table` hâlâ üç satırlıdır, çünkü `filter` ve `select` yeni tablo döndürdü. `drop(["sehir"])` bir sütunu gizler.

`int(row["puan"])`'a dikkat edin. Data `"91"`in sayı olduğunu tahmin etmez. Tüm sayısal sütun aynı kuraldadır:

```ahd
bring Data
from Data bring Table

table: Table := Data.fromCSV("ad,puan\nAli,91\nAyse,78\n")
scores: List<Real> := table.column("puan").map(
    lambda (value: String) -> real(value)
)
write(scores)
```

Gerçekçi ikinci adım: geçenleri tutup CSV metni yazmak:

```ahd
bring Data
from Data bring Table

table: Table := Data.fromCSV("ad,puan,sehir\nAli,91,Adana\nAyse,78,Ankara\n")
passed: Table := table.filter(
    lambda (row: Pair<String, String>) -> int(row["puan"]) >= 80
)
namesOnly: Table := passed.select(["ad", "puan"])
write(namesOnly.toCSV())
```

Olmayan sütun `DataError` fırlatır:

```ahd
bring Data
from Data bring Table
from Data bring DataError

table: Table := Data.fromCSV("ad,puan\nAli,91\n")
attempt {
    write(table.column("not"))
}
except DataError as error {
    write("boyle bir sutun yok")
}
```

**Siz deneyin:** Filtreyi `>= 90` yapıp `passed.rowCount()` ile kaç satır kaldığına bakın.

Ayrıca `sort`, `rename`, `reverse`, `head`, `tail`, `transform`, `derive`, `unique`, `valueCounts` ve `groupBy` vardır. [Data modül referansına](DATA_TR.md) bakın.

## 20. Fundamentals modülü

Bazı araçları kullanmak için hiçbir `bring` yazmanız gerekmez. Bunlar AhdCode programında doğrudan hazırdır:

```text
write take str int real len clear between abs sum min max type id
```

Çoğunu zaten kullandık. Kısa özet:

| İşlev | Ne yapar |
|---|---|
| `write(value)` | değeri ekrana yazar |
| `take()` / `take(prompt)` | kullanıcıdan bir satır `String` okur |
| `str(value)` | değeri String'e çevirir |
| `int(...)` | uygun değeri `Int`'e çevirir |
| `real(...)` | uygun değeri `Real`'e çevirir |
| `len(value)` | String/List/Pair uzunluğunu verir |
| `clear(collection)` | List veya Pair'i yerinde boşaltır |
| `between(...)` | sayı aralığı üretir |
| `abs(number)` | mutlak değeri verir |
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

`take` her zaman `String` döndürür; yaş veya not istiyorsanız `int(...)` veya `real(...)` ile açıkça dönüştürün. String sessizce sayı olmaz.

`sum` boş List için `0` veya `0.0` verir. `min` ve `max` boş List'te `DomainError` üretir.

`clear` mevcut koleksiyonu yerinde değiştirdiği için aynı koleksiyona bakan diğer takma adlar da boşalmış görür. `sum`, `min` ve `max` yalnızca okur; List'i değiştirmezler.

### `type(value)`: bir değerin türünü öğrenmek

`type(value)` değerin AhdCode türünü bir `String` olarak döndürür. Özellikle REPL'de deneme yaparken işe yarar:

```ahd
write(type(5))
write(type(5.0))
write(type("Ali"))
write(type(true))
write(type(null))
write(type([1, 2, 3]))
```

Beklenen çıktı:

```text
Int
Real
String
Bool
Null
List<Int>
```

Bir Class örneğinde `type` her zaman değişkenin bildirilen türünü değil, nesnenin **gerçek çalışma zamanı Class**'ını yazar:

```ahd
Animal: Class<> := { structure: Attributes := (name: String) }
Dog: Class<Animal> := { structure: Attributes := (SuperClass.attributes) }

pet: Animal := Dog(name: "Rex")
write(type(pet))
```

Bu program `Animal` değil `Dog` yazdırır.

`type` bir yansıma (reflection) nesnesi döndürmez — yalnızca düz bir String.

### `id(reference)`: çalışma zamanı kimliği

`id(reference)` bir List, Pair veya Class örneği için opak bir kimlik sayısı (`Int`) döndürür. `Int`, `Real`, `String` veya `Bool` gibi ilkel değerlerde kullanılamaz.

```ahd
a: List<Int> := [1, 2]
b: List<Int> := a
c: List<Int> := [1, 2]

write(id(a) == id(b))
write(id(a) == id(c))

a.add(3)
write(id(a) == id(b))
```

Beklenen çıktı:

```text
true
false
true
```

`a` ile `b` aynı List'e bakar, bu yüzden kimlikleri eşleşir; `c` içeriği aynı görünse bile **farklı** bir List'tir. `a`'yı `add` ile değiştirmek kimliğini değiştirmez.

Bu sayı bir bellek adresi değildir, program çalıştırmaları arasında korunmaz ve yalnızca bu oturumda anlamlıdır. Sıradan kimlik karşılaştırması için `same` kullanın; `id` asıl olarak hata ayıklama ve günlük içindir.

## 21. Math modülü

Karekök, trigonometri veya rastgele sayı gibi araçlar için `Math` modülünü kullanın:

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

| Öğe | Ne yapar |
|---|---|
| `PI`, `E` | matematiksel sabitler |
| `round`, `floor`, `ceil` | yuvarlama |
| `sqrt`, `exp` | karekök ve $e^x$ |
| `sin`, `cos`, `tan` | trigonometri (**radyan**) |
| `log`, `log10` | doğal ve 10 tabanlı logaritma |
| `seed`, `random`, `randomInt` | rastgele sayı |

Üs almak için `Math.pow` yoktur; dilin `^` işleci kullanılır. `abs`, `sum`, `min` ve `max` `Math`'te değildir; Fundamentals'tadır.

Küçük bir sayısal örnek — hipotenüs:

```ahd
bring Math

a: Real := 3.0
b: Real := 4.0
c: Real := Math.sqrt((a ^ 2.0) + (b ^ 2.0))
write(c)
```

```text
5.0
```

### Rastgele sayı üretmek

```ahd
bring Math

write(Math.randomInt(1, 6))
write(Math.random())
```

`randomInt(1, 6)` her iki ucu da dahil bir tam sayı üretir. `random()` aralığı `0.0 <= value < 1.0`'dır.

Testte aynı diziyi yeniden istiyorsanız tohum verin:

```ahd
bring Math

Math.seed(42)
```

Aynı tohum tekrar verilirse aynı dizi baştan başlar. Tohum yoksa yeni çalışma işletim sisteminden başlangıç alır.

`Math.random`, `Math.randomInt` ve `List.shuffle` **aynı** paylaşılan rastgelelik durumunu kullanır; her çağrı diziyi ilerletir. `randomInt(5, 5)` veya boş/`1` elemanlı `shuffle` rastgelelik durumunu tüketmez.

> **Dikkat:** Bu üreteci kriptografi veya güvenlik için kullanmayın.

**Siz deneyin:** `Math.seed(1)` deyip üç kez `randomInt(1, 6)` yazdırın, programı yeniden çalıştırıp aynı üç sayıyı doğrulayın.

## 22. Time modülü

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

`Time.now()` bilgisayarınızın yerel saatini verir. `Time.utc()` UTC'yi verir. `Time.timestamp()` işaretli Unix milisaniyesidir. AhdCode sabit dakika ofsetlerini destekler; adlı/IANA zaman dilimi veritabanı yoktur.

Bir `DateTime` şunları içerir:

```text
year  month  day  hour  minute  second  millisecond  weekday  offsetMinutes
```

`weekday` Pazartesi için `1`, Pazar için `7`'dir. Bu alanlar salt okunurdur.

`Time.fromTimestamp(milliseconds)` UTC döndürür. `dateTimeUTC(...)` UTC kurar, `dateTimeOffset(..., offsetMinutes: 180)` sabit ofsetli değer kurar. `toUTC()`, `toLocal()` ve `toOffset(...)` anı korur; `timestamp()` Unix milisaniyesini geri verir. Ofsetler -840..840 aralığındadır.

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

`hour`, `minute`, `second` ve `millisecond` verilmezse `0` olur. Geçersiz tarih `ValueError` üretir; örneğin AhdCode `2026-02-29`'u sessizce başka güne çevirmez, reddeder.

### İki zamanı karşılaştırmak

```ahd
bring Time
from Time bring DateTime

morning: DateTime := Time.dateTime(year: 2026, month: 1, day: 1, hour: 9)
evening: DateTime := Time.dateTime(year: 2026, month: 1, day: 1, hour: 21)

write(morning.before(evening))
write(morning.after(evening))
write(morning.sameMoment(morning))
```

Tarihler için `<` ve `>` yerine bu okunaklı metotlar kullanılır.

### İki zaman arasındaki süre

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

`Time.between(first, second)` ikinci zamandan birincisini çıkarır. `Time.duration(milliseconds: 1500)` ile doğrudan süre oluşturabilirsiniz.

### Takvim bilgileri

```ahd
bring Time

write(Time.Calendar.isLeapYear(2028))
write(Time.Calendar.daysInMonth(2028, 2))
write(Time.Calendar.weekday(2026, 8, 29))
```

### Bir işi bekletmek veya süresini ölçmek

```ahd
bring Time

start: Real := Time.monotonic()
Time.sleep(500)
elapsed: Real := Time.monotonic() - start
write(elapsed >= 0.5)
```

`Time.sleep(...)` **milisaniye**, `Time.monotonic()` ise **saniye** kullanır. `Time.monotonic()` tarih değildir; iki ölçüm arasındaki süreyi hesaplamak için kullanılır. Negatif `sleep` değeri `ValueError` üretir.

## 23. Statistics modülü

Bir sayı listesi henüz cevap değildir. “Tipik değer nedir?”, “ne kadar yayılmış?”, “ortadaki değer hangisi?” — bunlar `Statistics`'e aittir. Modül resim çizmez (o `Plot`) ve tablo hücrelerini metin olarak okumaz (o `Data`). Ona `List<Int>` veya `List<Real>` verirsiniz, o bir sayı döndürür.

```ahd
bring Statistics

scores: List<Int> := [70, 80, 80, 90, 100]

write(Statistics.mean(scores))
write(Statistics.median(scores))
write(Statistics.mode(scores))
write(Statistics.stdDev(scores))
write(Statistics.min(scores))
write(Statistics.max(scores))
```

Beklenen çıktı:

```text
84.0
80.0
80
10.198039027185569
70
100
```

Sıradan dilde:

- `mean` — aritmetik ortalama (`List<Int>` olsa bile her zaman `Real`)
- `median` — sıralanınca ortadaki değer; çift sayıda elemanda iki ortanın ortalaması, bu yüzden yine her zaman `Real` (`median([1, 2, 3, 4])` → `2.5`)
- `mode` — en sık değer; eleman türünü korur
- `stdDev` / `variance` — **yığın** yayılımı (`n`'ye böler)
- `sampleStdDev` / `sampleVariance` — **örneklem** yayılımı (`n - 1`'e böler; en az iki değer)
- `min` / `max` / `range` — en küçük, en büyük ve `max - min`
- `quantile(values, probability: p)` — `0.0` ile `1.0` arasında olasılık `p`'deki değer

Buradaki `sum` Statistics aşırı yüklemesidir (boş liste → `0` / `0.0`). Boş girdi aksi halde sessiz sıfır değil `StatisticsError`'dır:

```ahd
bring Statistics
from Statistics bring StatisticsError

empty: List<Int> := []
attempt {
    write(Statistics.mean(empty))
}
except StatisticsError as error {
    write("bos listenin ortalamasi yok")
}
```

Bu **derlenmez**, çünkü Statistics rakam metnini ayrıştırmaz:

```text
Statistics.mean(["10", "20"])
```

Önce dönüştürün, Data'da yaptığınız gibi:

```ahd
bring Data
from Data bring Table
bring Statistics

table: Table := Data.fromCSV("ad,puan\nAli,91\nAyse,78\n")
numbers: List<Int> := table.column("puan").map(
    lambda (value: String) -> int(value)
)
write(Statistics.mean(numbers))
```

**Siz deneyin:** `[4, 1, 3, 2]` listesinin `median`'ının `2.5` olduğunu doğrulayın.

Formüller için [Statistics modül referansına](STATISTICS_TR.md) bakın.

## 24. Plot modülü

`Plot` sayı listelerini resim dosyasına çevirir: `.png`, `.svg` veya `.pdf`. Bu bölümden tek başına işe yarar bir grafik üretebilirsiniz. Statistics gibi Plot da `"91"`i sayı olarak okumaz.

```ahd
bring Plot
from Plot bring Chart

weeks: List<Int> := [1, 2, 3, 4]
scores: List<Int> := [62, 71, 68, 80]

chart: Chart := Plot.line(weeks, scores)
chart = chart.title("Sinav puanlari")
chart = chart.xLabel("Hafta")
chart = chart.yLabel("Puan")
chart.save("sinav-puanlari.png")
```

`title`, `xLabel` ve `yLabel` her biri **yeni** bir `Chart` döndürür. Orijinal değişmez; Word belgesinde olduğu gibi yeniden atarsınız. `save` `Nothing` döndürür — `chart = chart.save(...)` yazılmaz.

İkinci sık grafik, adlı kategorilerin çubuk grafiğidir:

```ahd
bring Plot
from Plot bring Chart

subjects: List<String> := ["Matematik", "Fizik", "Tarih"]
averages: List<Real> := [88.0, 74.5, 91.0]

bars: Chart := Plot.bar(subjects, averages)
bars = bars.title("Sinif ortalamalari")
bars.save("ortalamalar.svg")
```

`Plot.scatter`, nokta istediğinizde `line` ile aynı şekildedir (x ve y sayı listeleri). `List<Int>` ile `List<Real>` karışabilir; bir `Numeric` Vector de kabul edilir. Boş veri `PlotError` fırlatır.

**Siz deneyin:** Başlığı dersinizin adına çevirip `"sinav-puanlari.pdf"` olarak kaydedin.

Histogram, kutu ve alt grafikler için [Plot modül referansına](PLOT_TR.md) bakın.

## 25. Numeric modülü ve Complex

`Numeric` doğrusal cebirdir: `Vector` uzunluğu olan sıralı sayılar, `Matrix` satır ve sütunlu bir ızgaradır. Elemanlar `Int` veya `Real`'dir. İşlemler yeni değer döndürür; elinizdeki Vector veya Matrix yeniden yazılmaz.

```ahd
bring Numeric
from Numeric bring Vector
from Numeric bring Matrix

v: Vector := Numeric.vector([1, 2, 3])
write(v.length())

m: Matrix := Numeric.matrix([[1, 2], [3, 4]])
write(m.determinant())
write(m.transpose().rowCount())
```

`determinant` kare matriste tanımlıdır. Uzunlukları uymayan iki vektörü toplamak veya iç boyutları uyuşmayan matrisleri çarpmak sessizce doldurmak yerine `NumericError` fırlatır.

Bir `Vector`, düz List yerine `Plot.line` veya `Plot.scatter`'a verilebilir — aynı sayıları hem hesaplayıp hem çizmek için.

### Complex sayılar

`Complex` bir **çekirdek türdür**, `Numeric` alt modülü değil. Gerçek ve sanal kısımları `Real`'dir. Sayıya yapışık büyük harf `I` yazın:

```ahd
z: Complex := 2 + 3I
write(z)
write((z * z))
```

Beklenen çıktı:

```text
2.0+3.0I
-5.0+12.0I
```

- `3I` geçerlidir.
- `3i` geçersizdir.
- `3 I` (aralarında boşluk) geçersizdir.

`Complex` gereken yerde `Int` veya `Real` güvenle kullanılır. Toplama, çarpma, bölme vardır ama sıralama yoktur — `<` ve `>` yok.

Ters, çözme ve ayrışımlar için [Numeric modül referansına](NUMERIC_TR.md) bakın.

## 26. Latex modülü (Latex)

Yapılandırılmış metinden PDF üretmek — kısa makale, rapor, slayt — için `Latex` gerçek LaTeX yazar ve **çevrimdışı** derler. TeX Live kurmazsınız. Render motorunu bir kez hazırlamak [Kurulum ve ilk programınız](#2-kurulum-ve-ilk-programınız) adımıdır.

Belge, birleştirdiğiniz bir String'dir; `document` ile sarılır, `pdf` ile derlenir:

```ahd
bring Latex as L
from Latex bring LatexError

body: String := L.section("Ilk belgem") +
    L.escape("Merhaba! Maliyet $5, sinifin %100'u gecti.") +
    L.subsection("Enerji") +
    L.equation("E = mc^2")

doc: String := L.document(body: body, title: "Notlar", type: "Article")

attempt {
    L.pdf(doc, "cikti.pdf")
}
except LatexError as error {
    write("PDF olusturulamadi: {error.message}")
}
```

İki farklı metin:

- **`escape`** — sıradan dil. `$ % & { }` görünür metin olur, LaTeX komutu olmaz.
- **`equation`** — ham matematik. Kaçışlanmaz, çünkü `$` ve `^` matematiğin kendisidir.

`section` / `subsection` başlıkları sizin yerinize kaçışlanır. `document`'in asıl yükü adlı `body` parametresidir (konumsal `L.document(body)` de çalışır). `type` `"Article"` (varsayılan), `"Report"` veya `"Beamer"` olabilir. Beamer temaları yalnızca `"Default"`, `"Madrid"` ve `"Warsaw"`.

`L.pdf(doc, "cikti.pdf", "tex")` PDF'in yanına `cikti.tex` de yazar. Derleme/motor hatası `LatexError`; geçersiz tema veya tür `ValueError`.

**Siz deneyin:** `type`'ı `"Report"` yapıp ikinci bir `subsection` ekleyin.

Teorem/tablo/kaynakça yardımcılarının tümünü ilk programa kopyalamayın. [Latex modül referansına](LATEX_TR.md) bakın.

## 27. Word modülü

Word **değiştirilemez** `.docx` belgeler üretir. Microsoft Office gerekmez. Her metot **yeni** bir `Document` döndürür; önceki değer olduğu gibi kalır. `save` dosyayı yazar ve `Nothing` döndürür; statement olarak çağırın.

```ahd
bring Word
from Word bring Document

document: Document := Word.new()
document = document.heading("Laboratuvar raporu", 1)
document = document.paragraph("AhdCode ile hazirlandi.", "center", true)
document = document.table(
    ["Ornek", "pH"]
    [["A", "7.1"], ["B", "6.8"]]
)
document.save("lab-raporu.docx")
```

`heading(text, level)` başlık düzeyleri `1`..`6`. `paragraph` yalnızca konumsaldır: metin, sonra isteğe bağlı hizalama (`"left"` / `"center"` / `"right"`), sonra isteğe bağlı `bold`, `italic`, `underline`. `table` başlık dizeleri ve satır listesi alır. `image(path)` PNG gömer (Plot'tan kaydettiğiniz bir figür dahil).

`Word.read(path)` sınırlı anlamsal alt kümeyi (paragraf, başlık, tablo) geri `Document`'e yükler. “Aç, başlık ekle, tekrar kaydet” için işe yarar — her Word özelliğiyle piksel mükemmel tur değil.

**Siz deneyin:** `save` öncesi 2. düzey bir başlık ve kısa bir paragraf ekleyin.

Birleştirme ve görsel boyutu için [Word modül referansına](WORD_TR.md) bakın.

## 28. Excel modülü

Excel, Microsoft Office olmadan gerçek `.xlsx` çalışma kitapları oluşturur ve okur. Üç katmanlı model:

- **Workbook** — dosya
- **Sheet** — çalışma kitabının içindeki adlı ızgara
- **Cell** — 1 tabanlı satır/sütunda tek tipli değer (`(1, 1)` A1'dir)

`Excel.new()` **boş** başlar — otomatik `Sheet1` yoktur. Dönüşümler yeni değer döndürür. `book.sheet("Scores")` ile aldığınız sayfanın gizli geri işaretçisi yoktur; düzenledikten sonra `book.withSheet(sheet)` ile geri koyarsınız.

```ahd
bring Excel
from Excel bring Workbook
from Excel bring Sheet
from Excel bring Cell

book: Workbook := Excel.new().addSheet("Puanlar")
sheet: Sheet := book.sheet("Puanlar")
sheet = sheet.setRow(1, 1, [Excel.fromString("Ad"), Excel.fromString("Puan")])
sheet = sheet.setRow(2, 1, [Excel.fromString("Ali"), Excel.fromInt(91)])
sheet = sheet.setCell(3, 1, Excel.fromString("Merve"))
sheet = sheet.setCell(3, 2, Excel.fromInt(88))
sheet = sheet.setCell(4, 2, Excel.formula("=SUM(B2:B3)"))
book = book.withSheet(sheet)
book.save("puanlar.xlsx")
```

Hücre değerleri açık kuruculardır: `fromString`, `fromInt`, `fromReal`, `fromBool`, `blank`, `formula`. `=` ile başlayan String, `Excel.formula(...)` kullanılmadıkça **metindir**. AhdCode formülleri **saklar**; Excel uygulaması gibi hesaplamaz.

Aynı dosyayı okuyun:

```ahd
bring Excel
from Excel bring Workbook
from Excel bring Sheet
from Excel bring Cell

book: Workbook := Excel.new().addSheet("Puanlar")
sheet: Sheet := book.sheet("Puanlar")
sheet = sheet.setCell(1, 1, Excel.fromString("Ali"))
sheet = sheet.setCell(1, 2, Excel.fromInt(91))
book = book.withSheet(sheet)
book.save("puanlar.xlsx")

loaded: Workbook := Excel.read("puanlar.xlsx")
page: Sheet := loaded.sheet("Puanlar")
score: Cell := page.cell(1, 2)
write(score.kind())
write(score.int())
```

Yanlış tür erişimi (`int()` bir String hücrede) `ExcelError` fırlatır. Bilinmeyen sayfa adı ve sıfır sayfalı kitap kaydetmek de `ExcelError`'dır.

**Siz deneyin:** Üçüncü bir öğrenci satırı ekleyip formülün yeni hücreyi kapsamasını sağlayın.

Birleştirme ve stil ilk not defteri için gerekmez. [Excel modül referansına](EXCEL_TR.md) bakın.

## 29. PDF modülü

`PDF`, değiştirilemez bir `PDFDocument` oluşturur (heading, paragraph, table,
image, page break) ve bunu çevrimdışı gerçek bir `.pdf` dosyasına render
eder — Microsoft Office, LibreOffice veya ağ bağlantısı gerekmez:

```ahd
bring PDF
from PDF bring PDFDocument

doc: PDFDocument := PDF.new()
doc = doc.heading("Quarterly Report", 1)
doc = doc.paragraph("Prepared offline, no Office dependency.")
doc = doc.table(["Region", "Q1", "Q2"], [["North", "10", "12"]])
doc.save("report.pdf")
```

`Document` ve `Table` gibi, her `PDFDocument` işlemi yalnızca konumsaldır ve
*yeni* bir `PDFDocument` döndürür — alıcı hiç değişmez. `save()`, `Nothing`
döndürür; bu yüzden onu bir statement olarak çağırın. Verdiğiniz her String
(heading, paragraph, table hücresi) render edilmeden önce kaçışlanır, bu
yüzden `\ { } $ & #` gibi karakterler her zaman sıradan metin olarak görünür
— `PDF`'in ham LaTeX enjekte etmenin bir yolu yoktur.

`PDF`, başka bir modülün kendi tipli belgesinden de doğrudan, elle kopyalama
gerekmeden bir belge oluşturabilir:

```ahd
bring Word
from Word bring Document

wordDocument: Document := Word.new()
wordDocument = wordDocument.heading("Report", 1)
wordDocument = wordDocument.paragraph("Hello")

pdfFromWord: PDFDocument := PDF.fromWord(wordDocument)
pdfFromWord.save("report-from-word.pdf")
```

`.save()`, `Latex.pdf` ile aynı çevrimdışı render motorunu kullanır — bir
kerelik hazırlık adımı için [kurulum ve ilk programınız](#2-kurulum-ve-ilk-programınız)
bölümüne bakın. Görsel boyutlandırma, sayfa düzeni ve hata ayrıntıları için
[PDF modül referansına](PDF_TR.md) bakın.

## 30. Archive modülü

`Archive`, dosyaları gerçek, deterministik `.zip`, `.tar` veya `.tar.gz`
arşivlerine paketler — yalnızca oluşturma amaçlıdır ve hiçbir render motoru
veya ek kuruluma ihtiyaç duymaz, çünkü yalnızca Go standart kütüphanesini
kullanır:

```ahd
bring Archive
bring File

File.writeText("notlar.txt", "paketle")
File.writeText("veri.csv", "a,b\n1,2\n")

files: Pair<String, String> := {
    "notlar.txt": "notlar.txt"
    "veri.csv": "veri.csv"
}

Archive.zip("teslim.zip", files)
write(File.exists("teslim.zip"))
```

`files` içindeki anahtar arşivin *içindeki* yoldur; değer ise diskteki
kaynak dosyanın yoludur. Güvensiz giriş yolları (`../secret` gibi) ve
symlink kaynakları, sessizce atlanmak yerine bir `ArchiveError` ile
reddedilir. Çıkarma (extraction), listeleme veya okuma API'si
yoktur — `Archive` yalnızca arşiv oluşturur. `zip`, `tar` ve `tarGzip`
üç oluşturma çağrısıdır; çıktı uzantısı eşleşmelidir (`.zip` / `.tar` /
`.tar.gz`). Ayrıntılar için [Archive modül
referansına](ARCHIVE_TR.md) bakın.

### Hepsini bir araya getirmek

İki küçük iş akışı, PDF, Excel ve Archive'ın gerçek programlarda
nasıl birleştiğini gösterir.

**Rapor paketleme** — bir Workbook'u PDF'e dönüştürün, sonra ikisini tek bir
ZIP'te birleştirin:

```ahd
bring Excel
bring PDF
bring Archive
from Excel bring Workbook
from Excel bring Sheet

book: Workbook := Excel.new().addSheet("Scores")
sheet: Sheet := book.sheet("Scores")
sheet = sheet.setRow(1, 1, [Excel.fromString("Name"), Excel.fromInt(91)])
book = book.withSheet(sheet)
book.save("report.xlsx")

pdf := PDF.fromExcel(book)
pdf.save("report.pdf")

files: Pair<String, String> := {
    "report.xlsx": "report.xlsx"
    "report.pdf": "report.pdf"
}

Archive.zip("report.zip", files)
```

**Latex kaynak yan dosyası** — derlenmiş PDF'i, tam LaTeX kaynağıyla birlikte
yayınlayın:

```ahd
bring Latex as L

source: String := L.document(body: L.section("Findings"), title: "Report")
L.pdf(source, "article.pdf", "tex")
```

Bu, tek bir çağrıdan hem `article.pdf` hem de `article.tex` üretir; üçüncü
argüman yalnızca `""` (varsayılan, yalnızca PDF) veya `"tex"` kabul eder.
İkisini `Archive.zip("article-bundle.zip", {"article.pdf": "article.pdf", "article.tex": "article.tex"})`
ile aynı şekilde paketleyebilirsiniz.

## 31. JSON modülü

JSON, tipli `JSONValue` okur ve kurar. `Any` yok, dinamik tipleme yok: bir JSON değeri tam olarak Null, Bool, Int, Real, String, Array veya Object'tir; derleyici hangi erişimcinin yasal olduğunu `kind()` veya tipli alıcıdan sonra bilir.

```ahd
bring JSON
from JSON bring JSONValue
from JSON bring JSONError

student: JSONValue := JSON.object({
    "ad": JSON.fromString("Ali")
    "puan": JSON.fromInt(91)
    "gecti": JSON.fromBool(true)
})

text: String := JSON.stringify(student, true)
parsed: JSONValue := JSON.parse(text)
write(parsed.kind())

name: JSONValue? := parsed.get("ad")
if name != null {
    write(name.string())
}

missing: JSONValue? := parsed.get("takma")
if missing == null {
    write("takma ad yok")
}
```

Beklenen çıktı (`stringify` boşluk/satır ekleyebilir; `kind` ve iki yazı kararlıdır):

```text
Object
Ali
takma ad yok
```

Kullanacağınız kurucular: `fromString`, `fromInt`, `fromReal`, `fromBool`, `array`, `object` ve `nullValue` (`JSON.null()` değil). `parse` bir String okur; `{` interpolasyon olmasın diye literal JSON'u **raw** String ile yazın: `JSON.parse(r'{"a":1}')`.

`get(key)` `JSONValue?` döndürür çünkü anahtar yok olabilir. Non-null bir `JSONValue` üzerinde `string()`, `int()`, `array()` `kind()` uyuşmazsa `JSONError` fırlatır. `"Ali"` JSON String'dir; `.int()` başarısız olur.

```ahd
bring JSON
from JSON bring JSONError

attempt {
    write(JSON.fromString("Ali").int())
}
except JSONError as error {
    write("yanlis tur")
}
```

**Siz deneyin:** Küçük bir nesneyi `parse` edin, olmayan bir anahtar isteyin, sonuç `null` ise varsayılan bir ad yazdırın.

[JSON modül referansına](JSON_TR.md) bakın.

## 32. XML modülü

XML küçük, kapalı bir düğüm modelidir: her düğüm ya **Element** (adlı etiket, isteğe bağlı öznitelikler, çocuklar) ya **Text** (karakter verisi). `Any` yok, tam DOM yok. Düğüm kurar, bir Element'i belge olarak sarar, yazıya dökersiniz veya ayrıştırırsınız.

```ahd
bring XML
from XML bring XMLNode
from XML bring XMLDocument

student: XMLNode := XML.element(
    "ogrenci"
    {"id": "42"}
    [
        XML.element("ad", {}, [XML.text("Ali")])
        XML.element("puan", {}, [XML.text("91")])
    ]
)
document: XMLDocument := XML.document(student)
write(XML.stringify(document, true))
write(student.kind())
idAttr: String? := student.attribute("id")
if idAttr != null {
    write(idAttr)
}
```

`XML.document(root)` Element kök ister — Text orada `XMLError` fırlatır. `kind()` her düğümde çalışır. `name`, `attribute`, `children`, `elements` Element içindir. Text düğümünde bunlar `XMLError` fırlatır; karakter için `text()` kullanın.

`XML.parse` String'i `XMLDocument`'e okur (tam bir kök element). Kurduğunuz öznitelikler niteliksizdir (ad alanı yok).

**Siz deneyin:** Bir `sehir` çocuk elementi ekleyip `stringify`'ı tekrar yazdırın.

[XML modül referansına](XML_TR.md) bakın.

## 33. Env modülü

**Ortam değişkeni**, işletim sisteminin (veya bir `.env` dosyasının) programınıza verdiği adlı bir String'dir: port, veri klasörü, bir bayrak. `Env` her zaman `String` döndürür. `"8080"`ün `Int` olduğuna karar vermez.

```ahd
bring Env

found: String? := Env.get("PORT")
if found == null {
    write("PORT ayarli degil")
}

port: Int := int(Env.getOr("PORT", "8080"))
write(port)
```

- `get` → `String?`. `null` adın yokluğu demektir.
- `getOr(name, fallback)` yedek değeri yalnızca ad yoksa kullanır. Açıkça boş `""` boş kalır, yedekle değiştirilmez.
- `exists(name)` Bool testidir (`has` ayrılmış sözcüktür; metot `has` değildir).

`Env.load(".env")` `NAME=value` satırlarını süreç ortamına okur. Varsayılan olarak zaten tanımlı adların üzerine yazmaz (`override` `false`). Bozuk `.env` `EnvError` fırlatır. `Env.read(path)` çiftleri süreç ortamını değiştirmeden döndürür.

Sırları kaynak dosyaya gömmeyin; commit etmediğiniz bir `.env` yerel yapılandırma için olağan yerdir.

**Siz deneyin:** Ayarlamadığınız bir adla `getOr` çağırıp yedeği gördüğünüzü doğrulayın, sonra o String'i `int(...)` edin.

[Env modül referansına](ENV_TR.md) bakın.

## 34. Lists ve KeyValue modülleri

Bu iki **modül**, zaten kullandığınız List ve Pair *türleriyle* aynı şey değildir. `list.add(...)` mevcut List'i değiştirir. `Lists.chunk(...)` ve `KeyValue.with(...)` **yeni** değer döndürür, orijinali bırakır.

```ahd
bring Lists
bring KeyValue

numbers: List<Int> := [1, 2, 3, 4, 5]
write(Lists.chunk(numbers, 2))
write(Lists.flatten([[1, 2], [3]]))
write(Lists.unique([1, 1, 2, 2, 3]))
write(Lists.valueCounts(["Matematik", "Fizik", "Matematik"]))
write(numbers)
```

Beklenen çıktı:

```text
[[1, 2], [3, 4], [5]]
[1, 2, 3]
[1, 2, 3]
{"Matematik": 2, "Fizik": 1}
[1, 2, 3, 4, 5]
```

`numbers` değişmedi. `transpose` satırları sütun yapar (düzensiz girdi `ListsError`). `groupBy` elemanları bir geri çağırım anahtarına göre gruplar.

Pair için `KeyValue` yapısal araçtır:

```ahd
bring KeyValue

record: Pair<String, String> := KeyValue.combine(["ad", "puan"], ["Ali", "91"])
updated: Pair<String, String> := KeyValue.with(record, "puan", "95")
slim: Pair<String, String> := KeyValue.select(updated, ["ad"])
write(KeyValue.keys(record))
write(updated)
write(record)
```

Beklenen çıktı:

```text
["ad", "puan"]
{"ad": "Ali", "puan": "95"}
{"ad": "Ali", "puan": "91"}
```

`with` / `without` anahtar ekler veya siler. `select` / `drop` anahtar tutar veya gizler. `rename` ve `mapValues` ad veya değer yazar. `merge` ayrık anahtar ister; `overlay`'de ikinci Pair çakışmada kazanır.

**Siz deneyin:** Listeyi `3` boyutunda `chunk` edin, sonra orijinal listenin değişmediğini yazdırın.

Her imza için [Lists](LISTS_TR.md) ve [KeyValue](KEYVALUE_TR.md) referanslarına bakın.

## 35. SQLite: hatırlayan bir veritabanı

Şimdiye kadar programdaki değerler, program bitince kayboluyordu. Bir **veritabanı**, program kapandıktan sonra da satırları duran bir dosyadır. **SQLite**, bilgisayarınızda tek bir dosyada (veya denemek için bellekte) yaşayan küçük bir veritabanı motorudur. Siz sıradan SQL yazarsınız; AhdCode güvenli ve tipli bir köprüdür: parametreleri bağlar ve değerleri dönüştürür. ORM değildir, sorgu oluşturucu değildir, migration aracı değildir.

```ahd
bring SQLite
from SQLite bring Database
from SQLite bring SQLiteValue
from SQLite bring SQLiteError

db: Database := SQLite.open("notes.db")
```

`SQLite.open("notes.db")` o dosyayı çalışma dizininde açar; yoksa oluşturur. `SQLite.open(":memory:")` kapattığınızda kaybolan özel bir bellek veritabanıdır. Üst klasörler **sizin için oluşturulmaz**: `data/` yoksa `SQLite.open("data/app.db")` `SQLiteError` fırlatır.

Basit bir not defteri tablosu:

```ahd
db.execute("""
    CREATE TABLE IF NOT EXISTS notes (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT NOT NULL,
        body TEXT NOT NULL
    )
    """)
```

`execute`, değişen satır sayısını döndürür. `CREATE TABLE` hiçbir satırı değiştirmez, bu yüzden `0` döner. Bu normaldir.

### Parametreyle not eklemek

SQL'e değer koymanın güvenli yolu **parametre bağlamadır**. Her `?`, SQLite'ın bir `SQLiteValue` ile doldurduğu bir deliktir. SQL metni asla yeniden yazılmaz:

```ahd
changed: Int := db.execute(
    "INSERT INTO notes (title, body) VALUES (?, ?)",
    [
        SQLite.fromString("Alışveriş")
        SQLite.fromString("süt, ekmek, çay")
    ]
)
write(changed)
write(db.lastInsertId())
```

Beklenen çıktı (yeni bir dosyadaki ilk not):

```text
1
1
```

`lastInsertId()`, SQLite'ın bağlantıya özel son eklenen satır kimliğidir. Önemli olan `INSERT`'ten hemen sonra çağırın.

Kullanıcı metnini SQL cümlesine interpolasyonla yerleştirmeyin. Aşağıdaki başlık SQL gibi durur; `?` ile sıradan metin olarak saklanır ve tablo durur:

```ahd
db.execute(
    "INSERT INTO notes (title, body) VALUES (?, ?)",
    [
        SQLite.fromString("Robert'); DROP TABLE notes;--")
        SQLite.fromString("bu veri olarak kalır")
    ]
)
```

Tırnak, noktalı virgül, satır sonu, ters eğik çizgi, Türkçe karakterler ve emoji, parametre olarak gittiklerinde yalnızca veridir.

### Satır okumak

```ahd
rows: List<Pair<String, SQLiteValue>> := db.query(
    "SELECT id, title, body FROM notes ORDER BY id"
)
for row in rows {
    write("{row["id"].int()} {row["title"].string()}")
}
```

Bir satır bir `Pair`'dir: anahtarlar, SELECT'in listelediği **sütun etiketleridir**. SQL'den `SQLiteValue` ile çıkarsınız, sonra bir AhdCode tipi istersiniz:

| `kind()`   | Okuma         | AhdCode tipi |
| ---------- | ------------- | ------------ |
| `"Null"`   | `isNull()`    | —            |
| `"Int"`    | `int()`       | `Int`        |
| `"Real"`   | `real()`      | `Real`       |
| `"String"` | `string()`    | `String`     |

SQL `NULL`, AhdCode `null`'ı değil, `Null` türünde bir `SQLiteValue`'dur. Satır yapısal olarak `Pair<String, SQLiteValue>` kalır. Yanlış türde erişim `SQLiteError` fırlatır: bir String asla sayı olarak ayrıştırılmaz. `real()` ayrıca `Int` türünü de kabul eder (`x: Real := 3` genişlemesiyle aynı). v0.3.0'da `BLOB` desteklenmez: sorgulamak `SQLiteError` fırlatır.

İki sütunun aynı etiketi varsa (`SELECT a.id, b.id`) AhdCode `SQLiteError` fırlatır. `AS` yazın:

```sql
SELECT a.id AS a_id, b.id AS b_id
```

### Sıra, güncelleme, silme, kapatma

SQLite, `ORDER BY` olmadan satır sırası **vaat etmez**. AhdCode `List`'i SQLite'ın döndürdüğü sırayı korur; kendisi bir sıra uydurmaz. Sıra önemliyse `ORDER BY` yazın.

```ahd
db.execute(
    "UPDATE notes SET body = ? WHERE id = ?",
    [SQLite.fromString("süt, ekmek, çay, bal"), SQLite.fromInt(1)]
)
db.execute("DELETE FROM notes WHERE id = ?", [SQLite.fromInt(2)])
db.close()
```

`close()` sonrasında o `Database` üzerindeki (ve onun takma adlarındaki) her işlem `SQLiteError` fırlatır. İkinci kapatma başarılıdır. İşlem (transaction) açıkken kapatmak `SQLiteError` fırlatır: önce `commit()` veya `rollback()` çağırın. Hiçbir şey sessizce kaydedilmez.

Aynı dosyayı yeni bir programda yeniden açın: notlar durur. v0.3.0'ın noktası budur.

### İşlemler (transactions)

Bir transaction, birkaç ifadeyi hepsi başarılı ya da hiçbiri olmasın diye gruplar:

```ahd
db.begin()
attempt {
    db.execute(
        "UPDATE accounts SET balance = balance - ? WHERE id = ?",
        [SQLite.fromReal(10.0), SQLite.fromInt(1)]
    )
    db.execute(
        "UPDATE accounts SET balance = balance + ? WHERE id = ?",
        [SQLite.fromReal(10.0), SQLite.fromInt(2)]
    )
    db.commit()
}
except SQLiteError as error {
    db.rollback()
    write(error.message)
}
```

Bir `Database`'de aynı anda yalnızca bir transaction vardır. İç içe `begin()` `SQLiteError` fırlatır. v0.3.0'da savepoint yoktur.

### SQLite Not Defteri

Tam yürüyüş [`examples/v0.3/01_sqlite_notes.ahd`](../examples/v0.3/01_sqlite_notes.ahd) dosyasındadır. Onu bir **geçici dizine** kopyalayın, çalıştırın, sonra bir daha çalıştırın: ilk çalıştırmanın notları hâlâ `notes.db` içindedir. Bu dosya sıradan bir SQLite veritabanıdır; AhdCode programının parçası değildir.

v0.3.0 editörleri mevcut dil sunucusu üzerinden `SQLite`'ı keşfeder: `bring SQL` yazın, completion `SQLite` önerir. Ek bir eklenti gerekmez.

**Siz deneyin:** `":memory:"` açın, boş bırakılabilir `nickname TEXT` sütunlu bir `people` tablosu oluşturun, `SQLite.nullValue()` ile bir satır ekleyin, sorgulayın ve `kind()` ile `isNull()` yazdırın.

[SQLite modül referansına](SQLITE_TR.md) bakın.

## 36. Kod biçimlendirici (Formatter)

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

## 37. Komut satırı (CLI)

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
ahdcode lsp
```

Editörlerin kullandığı dil sunucusunu başlatır. v0.2.2 editör desteği; tanılamalar, hover, completion ve otomatik import, tanıma gitme ve referans bulma, rename, signature help, semantik renklendirme, inlay hints, quick fix'ler ve biçimlendirme içerir. Ayrıntılar için [dil sunucusu rehberine](LSP_TR.md) bakın.

```text
ahdcode --help
ahdcode --version
```

Yardım ve sürüm bilgisini gösterir.

Yeni başlıyorsanız çoğu zaman kullanacağınız komut `ahdcode run ...` olacaktır.

## 38. Etkileşimli kabuk (REPL)

Küçük bir şeyi denemek için her seferinde dosya oluşturmak zorunda değilsiniz. Terminalde yalnızca:

```bash
ahdcode
```

çalıştırın. Başlangıçta `ahdcode --version` ile eşleşen bir sürüm başlığı
yazdırılır, ardından REPL açılır ve AhdCode komutlarını `ahd>` isteminde tek
tek deneyebilirsiniz:

```text
ahd> x: Int := 5
ahd> x = x + 1
ahd> x
6
```

REPL bir **oturum** gibi davranır. Önceki başarılı komutlarda oluşturduğunuz değerleri hatırlar:

```text
ahd> name: String := "Ali"
ahd> write(name)
Ali
```

Bir komutta hata yapmanız önceki çalışan durumu silmez:

```text
ahd> x: Int := 5
ahd> x: Int := 7
error: duplicate declaration
ahd> x
5
```

Önceki komutların yan etkileri yeniden çalıştırılmaz. Örneğin:

```text
ahd> write("bir")
bir
ahd> write("iki")
iki
```

ikinci komutta `bir` tekrar yazılmaz.

`take()` REPL içinde de gerçek kullanıcı girdisini bekler:

```text
ahd> name: String := take("İsim: ")
İsim: Ali
ahd> write(name)
Ali
```

Function ve Class tanımları, modüller, List/Pair nesneleri ve Math rastgelelik durumu oturum boyunca korunur. Yerel modüller ve göreli File yolları, `ahdcode` komutunu başlattığınız klasöre göre çözülür.

REPL öğrenirken çok kullanışlıdır: bir fikri hızlıca deneyip sonucu görebilirsiniz. Daha uzun programlarda `.ahd` dosyası kullanmak daha düzenlidir.

Bir `PDFDocument` veya Latex kaynak String'i oluşturmak REPL'de sorunsuz
çalışır, ama gerçekten bir `.pdf`'e derlemek çalışmaz: `Latex.pdf(...)`,
`Latex.pdfFile(...)` ve `PDFDocument.save(...)` etkileşimli olarak
çağrıldığında hata fırlatır, çünkü derleme kalıcı evaluator'ın desteklemediği
harici bir render motorunu çağırır. Bu çağrıları bir `.ahd` dosyasından
çalıştırın. `Archive`'ın böyle bir sınırlaması yoktur — REPL'de tamamen
çalışır. Ayrıntılar için [REPL referansına](REPL_TR.md) bakın.

## 39. Sık yapılan başlangıç hataları

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

**21. Kullanıcı metnini SQL'e String interpolasyonuyla koymak**
- Yanlış: `db.execute("INSERT INTO notes (title) VALUES ('{title}')")`
- Neden: Bu, başlığı SQL'e yapıştırır. `Robert'); DROP TABLE notes;--` gibi bir başlık artık veri değildir.
- Doğru: `?` yer tutucusu ve `SQLite.fromString(title)` kullanın. Parametre bağlama metni veri olarak tutar.

## 40. Küçük Projeler

Bu küçük projeler rehberde öğretilenleri bir araya getirir. Onları tek başınıza kurmayı deneyin!

1. **Not Ortalaması Hesaplayıcı**: Kullanıcıdan 5 not isteyin. Onları bir `List<Int>` içine koyun. Geçersiz notları (0'dan küçük veya 100'den büyük) listeden çıkarın (filter). Kalan notların ortalamasını, minimum ve maksimum değerini, son olarak da öğrencinin geçip (ortalama >= 50) geçmediğini yazdırın.
2. **Basit Hesap Makinesi**: İki sayı ve bir operatör (`+`, `-`, `*`, `/`) almak için `take()` kullanın. İşlemi seçmek için operatör üzerinde `state` kullanın ve sonucu yazdırın. Sıfıra bölünme ihtimalini `attempt`/`except` ile yönetin.
3. **Sayı İstatistikleri**: `Math.randomInt(1, 100)` ile 100 adet rastgele sayı üretin. Bunlardan kaç tanesinin tek, kaç tanesinin çift olduğunu sayın (count) ve listeyi sıralayın. Bir sayının asal olup olmadığını kontrol eden bir fonksiyon yazın ve listeyi sadece asalları gösterecek şekilde filtreleyin (filter).
4. **Kelime Analizi**: Kullanıcıdan bir cümle girmesini isteyin. Kelimeleri ayırmak için `split(" ")` kullanın. Kelime sayısını bulun, en uzun kelimeyi bulun ve her kelimenin kendi uzunluğuyla eşleştiği bir `Pair<String, Int>` oluşturun.
5. **Menülü Program**: Bir `until` döngüsü kullanarak küçük bir banka simülasyonu yapın. Bir menü gösterin: 1. Para Yatır, 2. Para Çek, 3. Bakiye, 0. Çıkış. Bakiyeyi bir `Int` içinde saklayın ve kullanıcı 0 girene kadar programı döndürün.
6. **Sınıflarla (Class) Öğrenci Kaydı**: Bir `Student` sınıfı ve bir `Course` (Kurs) sınıfı oluşturun. Course içinde bir `List<Student>` bulunsun. Kursa yeni bir öğrenci eklemek için bir metot, kursun genel not ortalamasını hesaplamak için başka bir metot yazın.
7. **Tohumlu (Seeded) Rastgele Oyun**: `Math.seed(42)` kullanarak 1 ile 100 arasında "gizli bir sayı" üretin. Kullanıcıdan sayıyı tahmin etmesini isteyin. Doğru tahmin edene kadar "daha yüksek" veya "daha düşük" diye yönlendirin. Tohum kullanıldığı için, gizli sayı programı her çalıştırdığınızda aynı olacaktır—test yapmak için mükemmel!
8. **SQLite Not Defteri**: `notes.db` açın, yoksa bir `notes` tablosu oluşturun ve kullanıcının not eklemesine, listelemesine, başlığa göre aramasına, güncellemesine ve silmesine izin verin. Her değer için `?` parametreleri kullanın. Programı kapatıp yeniden çalıştırın: eski notlar durmalıdır.

## 41. Egzersizler

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
21. Küçük bir SQLite not defteri kurun: parametrelerle iki not ekleyin, `ORDER BY id` ile listeleyin, bir gövdeyi güncelleyin, bir satırı silin, veritabanını kapatın, aynı dosyayı yeniden açın ve kalan başlıkları yazdırın.

## 42. Çözüm İpuçları

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
21. `SQLite.open("notes.db")`, `CREATE TABLE IF NOT EXISTS`, `SQLite.fromString` ile `INSERT ... VALUES (?, ?)`, sonra `ORDER BY id` ile `query`. `close()` sonrası aynı yolu yeniden açın.

## 43. Sonraki adımlar ve teknik belgeler

Bu rehberi tamamladıktan sonra dilin ayrıntılarını şu belgelerden
derinleştirebilirsiniz:

- [Başlangıç](GETTING_STARTED_TR.md)
- [Dil turu](LANGUAGE_TOUR_TR.md)
- [Türler ve null](TYPES_AND_NULL_TR.md)
- [Kontrol Akışı](CONTROL_FLOW_TR.md)
- [Fonksiyonlar](FUNCTIONS_TR.md)
- [Sınıflar](CLASSES_TR.md)
- [Class Protocol Methods](PROTOCOLS_TR.md)
- [Koleksiyonlar](COLLECTIONS_TR.md)
- [Modüller](MODULES_TR.md)
- [Hatalar](ERRORS_TR.md)
- [Temel İşlevler](FUNDAMENTALS_TR.md)
- [String API](STRING_API_TR.md)
- [List API](LIST_API_TR.md)
- [Math](MATH_TR.md)
- [Time](TIME_TR.md)
- [Statistics](STATISTICS_TR.md)
- [Plot](PLOT_TR.md)
- [Numeric](NUMERIC_TR.md)
- [Latex](LATEX_TR.md)
- [Word](WORD_TR.md)
- [Excel](EXCEL_TR.md)
- [PDF](PDF_TR.md)
- [Archive](ARCHIVE_TR.md)
- [JSON](JSON_TR.md)
- [SQLite](SQLITE_TR.md)
- [XML](XML_TR.md)
- [Env](ENV_TR.md)
- [Lists](LISTS_TR.md)
- [KeyValue](KEYVALUE_TR.md)
- [File ve Path](FILESYSTEM_TR.md)
- [Regex](REGEX_TR.md)
- [CSV](CSV_TR.md)
- [Data](DATA_TR.md)
- [Tanılamalar](DIAGNOSTICS_TR.md)
- [CLI](CLI_TR.md)
- [Formatter](FORMATTER_TR.md)
- [REPL](REPL_TR.md)
- [Dil sunucusu](LSP_TR.md)
- [Tam v0.1 spesifikasyonu](../AHDCODE_LANGUAGE_SPEC_v0.1_TR.md)

Çalışan daha fazla örnek için [derlenmiş v0.1 örnekleri](../examples/v0.1/README_TR.md)
klasörüne ve [v0.3 SQLite Not Defteri](../examples/v0.3/README_TR.md) örneğine bakın.
