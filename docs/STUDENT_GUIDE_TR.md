# AhdCode v1.4.0 Türkçe Öğrenci Rehberi

Bu rehber, **daha önce hiç programlama yapmamış birinin de takip edebilmesi** için hazırlanmıştır. Baştan sona sırayla okuyabilirsiniz; her bölümde önce ne yapmak istediğimizi görecek, sonra çalışan bir örnek yazacak, en son gerekli kuralları öğreneceksiniz.

Kodların yanında zaman zaman İngilizce teknik terimler de göreceksiniz. Bunları ilk okumada ezberlemeniz gerekmez. Bir kutu **Teknik not** diye başlıyorsa, programı çalıştırabilmek için o ayrıntıyı hemen bilmek zorunda değilsiniz.

En iyi öğrenme yolu, örnekleri yalnızca okumak değil çalıştırmaktır. Bir örneği kopyalayın, içindeki sayıyı veya metni değiştirin ve sonucun nasıl değiştiğine bakın.

Bu rehber dilin temel öğrenme yoludur. Özellikle CSV, Data, Plot, Excel, Word,
Latex, HTTP(S) ve HTML ile gerçek bir çıktı üretmek istiyorsanız, ilgili kısa
tanıtımdan sonra [Uygulamalı Modül Atölyeleri](PRACTICAL_MODULES_TR.md)
belgesine geçin. Orada modüller `CSV → Data → Statistics/Plot → Excel/Word/Latex` ve
`HTTPS → HTML` iş akışları içinde, kontrol noktaları ve görevlerle anlatılır.
Tek tek bütün imzalar ve sınır koşulları ise her modülün kendi referans
belgesindedir.

Bu rehber öğretir; bir referans kılavuzu ya da değişiklik günlüğü değildir. Bir
modülün, yeni başlayan birinin ilk okumada ihtiyaç duyduğundan fazlası varsa,
ilgili bölüm bunu söyler ve bütün imzaları listeleyen referans sayfasına bağlantı
verir.

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
  - [Uygulamalı modül çalışma rotası](#uygulamalı-modül-çalışma-rotası)
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
- [36. Küçük bir web sayfası](#36-küçük-bir-web-sayfası)
- [37. Çerezler ve oturumlar](#37-çerezler-ve-oturumlar)
- [38. HTTP Client](#38-http-client)
- [39. HTML ayrıştırma ve web kazıma (scraping)](#39-html-ayrıştırma-ve-web-kazıma-scraping)
- [40. Dosya yüklemeleri](#40-dosya-yüklemeleri)
- [41. E-posta gönderme (SMTP)](#41-e-posta-gönderme-smtp)
- [42. Kod biçimlendirici (Formatter)](#42-kod-biçimlendirici-formatter)
- [43. Komut satırı (CLI)](#43-komut-satırı-cli)
- [44. Etkileşimli kabuk (REPL)](#44-etkileşimli-kabuk-repl)
- [45. Sık yapılan başlangıç hataları](#45-sık-yapılan-başlangıç-hataları)
- [46. Küçük Projeler](#46-küçük-projeler)
- [47. Egzersizler](#47-egzersizler)
- [48. Çözüm İpuçları](#48-çözüm-i̇puçları)
- [49. Sonraki adımlar ve teknik belgeler](#49-sonraki-adımlar-ve-teknik-belgeler)
- [50. Güvenlik: parola hashleme ve güvenli belirteçler](#50-güvenlik-parola-hashleme-ve-güvenli-belirteçler)
- [51. MySQL: ağ veritabanı sunucusu](#51-mysql-ağ-veritabanı-sunucusu)
- [52. Web: bir uygulama kurmak](#52-web-bir-uygulama-kurmak)
- [53. Tam bir form iş akışı](#53-tam-bir-form-iş-akışı)
- [54. Gruplar, bekçiler ve starter'lar](#54-gruplar-bekçiler-ve-starterlar)
- [55. Görebildiğiniz veritabanları: `ahdcode databases`](#55-görebildiğiniz-veritabanları-ahdcode-databases)
- [56. Uygulamanızın yerel adresi](#56-uygulamanızın-yerel-adresi)
- [57. UUID: benzersiz kimlikler](#57-uuid-benzersiz-kimlikler)
- [58. Gizli değerleri dosyadan okumak](#58-gizli-değerleri-dosyadan-okumak)
- [59. PostgreSQL: güçlü bir ağ veritabanı](#59-postgresql-güçlü-bir-ağ-veritabanı)
- [60. WebSocket: canlı bağlantılar](#60-websocket-canlı-bağlantılar)
- [61. Hepsi bir arada: gerçek zamanlı yoklama uygulaması](#61-hepsi-bir-arada-gerçek-zamanlı-yoklama-uygulaması)
- [62. Terminal: çıktı üzerinde daha fazla denetim](#62-terminal-çıktı-üzerinde-daha-fazla-denetim)

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

AhdCode v1.0.0 ilk kararlı sürümdür. Güncel sürüm olan v1.2.0;
[Cron](CRON_TR.md) ile zamanlama, [Characters](CHARACTERS_TR.md) ile Unicode
karakter araçları ve [Latex](LATEX_TR.md) için TikZ vektör grafikleri ekler; bu
rehber onlara ihtiyaç duymaz, başvuru sayfaları onları anlatır.

Onunla küçük komut satırı programları yazabilir veya bunları yerel executable uygulamalara derleyebilirsiniz; veriyi yerel bir SQLite veritabanında ya da bir MySQL sunucusunda tutabilirsiniz; birinci taraf `Web` çatısıyla eksiksiz bir web uygulaması kurabilirsiniz — sayfalar, yerleşimler, formlar, doğrulama, CSRF, flash mesajları, oturumlar ve dosya yüklemeleri; dış HTTP ve HTTPS API'leri çağırabilir, HTML ayrıştırabilir, SMTP ile e-posta gönderebilir, `Security` ile parola hashleyip güvenli belirteç üretebilir ve dil sunucusunu (`ahdcode lsp`) VS Code gibi bir editörden kullanabilirsiniz.

Araç zinciri de dille birlikte büyüdü. `ahdcode dev` siz kaydettikçe dosyalarınızı izler, yeniden derler ve yeniden başlatır. `ahdcode init web` çalışan bir starter proje yazar. `ahdcode databases` yerel veritabanı çalışma alanı AhdDataStudio'yu açar. Ve v0.19'dan beri yerel çalıştırdığınız bir web uygulaması kendine ait okunabilir bir adres alır — bir port numarası yerine `http://ahdakademi.test/` — bunu 56. bölüm anlatır.

Bazı standart modüller, örneğin SQLite, AhdCode'un sağladığı yardımcı çalışma zamanı bileşenlerini kullanır. HTTP, HTML, çerez, oturum ve HTTP Client, çalışma zamanının içindeki Go standart kütüphanesini kullanır; üçüncü taraf bir HTTP bağımlılığı eklemezler.

Dili kullanmak için hangi sürümün neyi eklediğini bilmeniz gerekmez. Merak ediyorsanız bu geçmişi [README](../README_TR.md) tutar.

> **Teknik not:** Program çalışmadan önce türlerin kontrol edilmesine *static checking* denir.

## 2. Kurulum ve ilk programınız

Platformunuza uygun sürüm paketini [Kurulum](INSTALLATION_TR.md) adımlarına göre kurun.
Paket; özel Go araç zincirini, AhdDataStudio'yu, yardımcı programları ve çevrimdışı
LaTeX kaynaklarını içerir. Git, kaynak deposu, sistem Go kurulumu, `AHDCODE_ROOT`,
npm veya ayrı TeX kurulumu gerekmez.

Yeni bir terminal açıp doğrulayın:

```bash
ahdcode --version
```

Kaynaktan derleme, geliştirici iş akışıdır; kök README belgesine bakın.

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

### Uygulamalı modül çalışma rotası

Modül öğrenirken üç belge türünün görevi farklıdır:

| Belge | Ne için kullanılır? |
|---|---|
| Bu öğrenci rehberi | Kavramı ilk kez öğrenmek ve küçük örneği görmek |
| [Uygulamalı Modül Atölyeleri](PRACTICAL_MODULES_TR.md) | Baştan sona bir iş akışı kurmak ve alıştırma yapmak |
| Modül referansı | Bütün fonksiyonları, parametreleri, hataları ve sınırları doğrulamak |

Önerilen pratik sıra şöyledir:

1. CSV ile dışarıdan gelen metin tablosunu okuyun.
2. Data ile filtreleyin, sıralayın, dönüştürün ve gruplayın.
3. Sayı gereken yerde açıkça `int(...)` / `real(...)` kullanın.
4. Statistics ile özet hesaplayıp Plot ile grafiği görünür kılın.
5. Excel ile tipli `.xlsx`, Word ile `.docx` veya Latex ile `.pdf` üretin.
6. HTTP Client ile HTTPS içeriği alın; HTML ile güvenli sayfa kurun veya
   alınmış işaretlemeyi ayrıştırın.

Bu sırada hiçbir modül diğerinin görevini gizlice üstlenmez. CSV ve Data
sayısal tür tahmin etmez; Excel formül hesaplamaz; HTML URL indirmez; HTTP
aldığı HTML'i kendi başına ayrıştırmaz.

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

### Bir uygulamayı dosyalara bölmek: `require(...)`

`bring` bir *modül* içe aktarır — `Time` gibi ya da kendi `Greeting.ahd`
dosyanız gibi, kendi ad alanı olan bir şey. Bir uygulama kurarken genellikle
bunun tersini istersiniz: birlikte tek bir program oluşturan, tek bir ad
kümesini paylaşan birkaç dosya.

`require(...)` tam olarak bunun içindir:

```ahd
require("Config/App.ahd")
require("Pages/Home.ahd")

bring Web
from Web bring App

akademi: App := Web.app(application)
akademi.get("/", homePage)
akademi.start()
```

Her `require(...)` satırı, kendisini yazan dosyaya göreli bir dosya adı verir.
Derleyici o dosyayı *bu* programın parçası olarak okur; böylece
`Pages/Home.ahd` içinde tanımlanan `homePage` burada hiçbir önek olmadan
sıradan bir isimdir.

Aklınızda tutmaya değer üç kural:

- yol bir **String sabitidir**, değişken değil. Derleyicinin, herhangi bir
  şeyi çalıştırmadan önce programın tamamını bilmesi gerekir;
- aynı dosyayı iki kez `require` etmek sorun değildir. Bir kez okunur;
- döngü, `bring`'de olduğu gibi derleme zamanı hatasıdır.

Bu rehberdeki her ciddi uygulama böyle düzenlenmiştir ve `ahdcode init web`
sizin için bunu üretir. `ahdcode dev`, `require(...)` çizgesindeki her dosyayı
izler; bu yüzden bir sayfayı düzenlemek de tıpkı giriş dosyasını düzenlemek
gibi yeniden derleyip yeniden başlatır. Kuralların tamamı:
[`require(...)`](REQUIRE_TR.md).

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

Dosya okuma, bozuk değer, özel ayraç ve kayıt yazma adımlarını birlikte
çalışmak için [CSV atölyesine](PRACTICAL_MODULES_TR.md#1-csv-metin-tablosunu-güvenle-taşımak)
geçin.

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

Temizleme, sayısal sıralama, gruplama, Statistics'e geçiş ve veri kalite
kontrolü için [Data atölyesini](PRACTICAL_MODULES_TR.md#2-data-string-tablosunu-şekillendirmek)
tamamlayın.

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

Data sütunundan sayısal liste üretme, doğru grafik türünü seçme ve aynı PNG
grafiği Word ile Latex raporlarına gömme akışı için
[Plot atölyesini](PRACTICAL_MODULES_TR.md#3-plot-veriyi-okunabilir-bir-grafiğe-dönüştürmek)
tamamlayın.

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

Article, Report ve Beamer üretimini; denklem, tablo, görsel, kaynakça ve
`.tex` hata ayıklama çıktısıyla birlikte görmek için
[Latex atölyesine](PRACTICAL_MODULES_TR.md#6-latex-akademik-pdf-ve-sunum-üretmek)
geçin.

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

Data tablosu ve Plot grafiğinden doğrulanabilir bir DOCX raporu kurmak için
[Word atölyesini](PRACTICAL_MODULES_TR.md#5-word-tablo-ve-görselli-docx-raporu)
tamamlayın.

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

Tipli hücre, formül, Range, stil ve kaydedilen kitabı yeniden okuma akışı için
[Excel atölyesine](PRACTICAL_MODULES_TR.md#4-excel-tipli-hücrelerle-gerçek-xlsx-üretmek)
geçin.

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

**Sunucudaki parolalar.** v1.4.0 ile gelen `Env.secret("DB_PASSWORD")`, değeri ya `DB_PASSWORD` değişkeninden ya da `DB_PASSWORD_FILE` değişkeninin gösterdiği dosyadan okur; ikisi de yoksa `null` döndürür. Docker ve benzeri platformlar parolaları bu şekilde, dosya olarak verir. Kurallar ve örnekler [58. bölümde](#58-gizli-değerleri-dosyadan-okumak).

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

### Tablolar, satırlar ve sütunlar

Bir tabloyu, bir Excel sayfasının tek bir sayfası gibi düşünün. Bir adı
(`notes`), her satırın taşıyacağı bilginin *türünü* belirten sabit bir
**sütun** kümesi vardır -- `id`, `title`, `body` gibi -- ve kaydettiğiniz her
not için bir **satır** eklenir. Tablodaki her satır aynı sütunlara, aynı
sırada sahiptir; satırdan satıra yalnızca değerler değişir.

**SQL** (Structured Query Language / Yapılandırılmış Sorgu Dili), bir
tabloyla konuşmak için kullandığınız dildir: oluşturmak, satır eklemek,
satırları geri okumak, değiştirmek, silmek. SQL'in tamamını öğrenmenize
gerek yok -- bu bölümde ihtiyacınız olan birkaç kelime aşağıda, gerçek bir
AhdCode örneğinde karşınıza çıkacak:

| SQL kelimesi | Sade dille anlamı |
|---|---|
| `CREATE TABLE` | bu sütunlarla yeni bir tablo oluştur |
| `INSERT INTO ... VALUES (...)` | yeni bir satır ekle |
| `SELECT ... FROM ...` | satırları geri oku |
| `WHERE` | yalnızca bu koşula uyan satırlar |
| `ORDER BY` | bu sırayla |
| `UPDATE ... SET ...` | var olan satırları değiştir |
| `DELETE FROM ...` | satırları sil |
| `PRIMARY KEY` | bir satırı benzersiz şekilde tanımlayan sütun, bir öğrenci numarasının tek bir öğrenciyi tanımlaması gibi |

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

## 36. Küçük bir web sayfası

v0.3.0, program kapansa da hatırlayan bir veritabanı öğretti. v0.4.0, bu
veriyi bu makinenin tarayıcısında açmanın yolunu ekler: yerel sunucu için
`HTTP`, kullanıcı metninin etiket olmaması için `HTML`.

### Bir tarayıcı sayfa açtığında ne olur?

Bir adresi tarayıcıya yazıp Enter'a bastığınızda, tarayıcı bir yere
**istek** (request) adında küçük bir mesaj gönderir ve bekler. O yerdeki
program bir **sunucudur** (server): isteği okur, ne yapacağına karar verir
ve geri bir **yanıt** (response) gönderir -- genellikle tarayıcının ekrana
çizdiği bir HTML sayfası. Bu bölümde, kendi AhdCode programınız kendi
bilgisayarınızda çalışan o sunucu olacak.

İsteğin iki parçası bir yeni başlayan için en önemlisi:

- **Yöntem** (method), isteğin ne tür olduğunu söyler. `GET`, "bana
  bakacağım bir şey ver" demektir -- bir sayfa açmak, bir bağlantıya
  tıklamak. `POST`, "işte veri, bununla bir şey yap" demektir -- bir form
  göndermek.
- **Yol** (path), adresteki host'tan sonraki kısımdır, örneğin `/` veya
  `/notes`. Her yolun ne yapacağına programınız karar verir; o isimde bir
  dosya var diye kendiliğinden bir şey olmaz.

**Port**, bilgisayardaki *hangi* çalışan programın isteği alacağını söyleyen
bir sayıdır (`8080` gibi) -- aynı bilgisayarda birçok program aynı anda
dinleyebilir, her biri kendi portunda, birden fazla dairenin aynı cadde
adresini paylaşıp her birinin kendi kapı numarasına sahip olması gibi.

**HTML**, bir web sayfasının yazıldığı biçimlendirme dilidir. `<h1>...</h1>`
gibi bir **etiket** (tag), bir içerik parçasını büyük başlık olarak
işaretler; `<form>` ise tarayıcının toplayıp sunucuya geri göndereceği bir
girdi grubunu işaretler. HTML'i önceden bilmenize gerek yok -- bu bölümün
kullandığı her etiket aşağıdaki örneklerde gösteriliyor.

`127.0.0.1` **yalnızca bu bilgisayar** demektir. Açacağınız adres
`http://127.0.0.1:8080/` şeklindedir. `0.0.0.0` bağlamayı gelişigüzel
yapmayın. `Server.start()` programı durdurana kadar terminali meşgul tutar.

### İlk GET yolu

```ahd
bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response

home: Function := (request: Request) -> Response {
    return HTTP.html(
        r"""
        <!doctype html>
        <html>
        <body>
            <h1>Hello from AhdCode</h1>
        </body>
        </html>
        """
    )
}

app: Server := HTTP.server("127.0.0.1", 8080)
app.get("/", home)
app.start()
```

Çalıştırın, sonra tarayıcıda `http://127.0.0.1:8080/` açın. **Hello from
AhdCode** görmelisiniz. `HTTP.html`, **sizin** yazdığınız işaretleme içindir.
Kaçırmaz. Bu yüzden merhaba sayfası ham String `r"""..."""` kullanır.

Bir işleyici her zaman `(request: Request) -> Response` şeklindedir. Derleyici
bunu program çalışmadan önce denetler.

### İstek yöntemi, yol ve sorgu

```ahd
hello: Function := (request: Request) -> Response {
    write(request.method())
    write(request.path())
    name: Local String? := request.query("name")
    if name != null {
        return HTTP.text("Hello {name}")
    }
    return HTTP.text("Hello")
}

app.get("/hello", hello)
```

`GET /hello?name=Ayşe` için `request.query("name")` `Ayşe` String'idir; ad yoksa
`null`'dır. `queryAll` yinelenenlerin hepsini döndürür. Yol, `?` sorgusunu
içermez.

### Dinamik metin için güvenli HTML

Metin sorgudan, formdan veya SQLite'tan geliyorsa `HTML.text` kullanın.
Oluşturucu `<`, `>`, `&` ve tırnakları kaçırır:

```ahd
bring HTML

page: String := HTML.document(
    "Notes"
    [HTML.element("p", {}, [HTML.text(userText)])]
)
return HTTP.html(page)
```

`HTML.document` tam bir sayfa kurar ve başlığı kaçırır. `HTTP.html` bu zaten
güvenli String'i HTML içerik türüyle gönderir.

Kullanıcı metnini **asla** `"<p>" + userText + "</p>"` diye birleştirip HTML
olarak göndermeyin.

### POST form ve yönlendirme

```ahd
bring HTTP
from HTTP bring Server
from HTTP bring Request
from HTTP bring Response

home: Function := (request: Request) -> Response {
    return HTTP.html(
        r"""
        <!doctype html>
        <html>
        <body>
            <form method="post" action="/notes">
                <input name="title">
                <button type="submit">Save</button>
            </form>
        </body>
        </html>
        """
    )
}

save: Function := (request: Request) -> Response {
    title: Local String? := request.form("title")
    if title == null {
        return HTTP.text("title is required", 400)
    }
    return HTTP.redirect("/")
}

app: Server := HTTP.server("127.0.0.1", 8080)
app.get("/", home)
app.post("/notes", save)
app.start()
```

Çalıştırın, sonra `http://127.0.0.1:8080/` açın. Formu gönderin. Formlar
`application/x-www-form-urlencoded` kullanır. Başarılı bir POST'tan sonra
`HTTP.redirect("/")` (durum 303) tarayıcının formu yeniden göndermek yerine
sayfayı GET etmesini sağlar.

### SQLite ile birleştirmek: Web Not Defteri

Her işleyicide `notes.db` açın, `?` parametreleriyle küçük bir SQL deyimi
çalıştırın, dönmeden önce `close()` çağırın. Saklanan her başlık ve gövdeyi
`HTML.text` ile çizin. Silme formundaki `id`'yi `int(...)` ile dönüştürün.

Tam yürüyüş [`examples/v0.4/03_web_notes.ahd`](../examples/v0.4/03_web_notes.ahd)
dosyasındadır. `notes.db` depo içinde oluşmasın diye onu bir **geçici dizine**
kopyalayın. `<script>alert(1)</script>` ve `Robert'); DROP TABLE notes;--`
gibi başlıklar sıradan veri kalır: bağlı SQL parametreleri ve kaçırılmış HTML.

v0.4.0 editörleri mevcut dil sunucusu üzerinden `HTTP` ve `HTML`'i keşfeder:
`bring HT` yazın, completion ikisini de önerir. Ek bir eklenti gerekmez.

**Siz deneyin:** `GET /ok` üzerinde `HTTP.text("ok")` sunun, sonra tarayıcıda
`http://127.0.0.1:8080/ok` açın.

[HTTP modül referansına](HTTP_TR.md) ve [HTML modül referansına](HTML_TR.md)
bakın.

## 37. Çerezler ve oturumlar

v0.4.0 bir sayfa sundu. v0.5.0, **bu tarayıcının** değerini **o tarayıcıyla**
paylaşmadan tutmayı ekler.

### Çerez nedir?

**Çerez** (cookie), sunucunun tarayıcıdan hatırlamasını istediği küçük bir
metin parçasıdır. Bir kez ayarlandıktan sonra, tarayıcı aynı metni aynı
siteye gönderdiği her sonraki isteğe kendiliğinden ekler -- çerezin tek özel
yaptığı budur: tarayıcı onu sizin yerinize tekrar gönderir, sayfanızda veya
AhdCode programınızda fazladan bir kod olmadan. Bir sitenin, HTTP'nin kendisi
bir istekten diğerine hiçbir hafızaya sahip olmamasına rağmen, aynı
tarayıcıyı birkaç sayfa yüklemesi boyunca "tanıyabilmesinin" sebebi çerezdir.

AhdCode'un ayarladığı çerez yalnızca rastgele bir kimlik taşır -- bir isim
değil, bir parola değil. Gerçek değerler sunucuda, bellekte, o kimlikle
aranarak durur. Bu bir giriş çerçevesi değildir. Bir formdan sonra adı
kendiniz saklayabilirsiniz.

Modül düzeyindeki `SessionStore`, her işleyicide diğer modül bağlamaları gibi
`Global` ile bildirilir.

### İstek çerezi

```ahd
theme: Local String? := request.cookie("theme")
values: Local List<String> := request.cookieAll("theme")
```

`cookie` ilk değerdir veya `null`. `cookieAll` tüm değerlerdir veya `[]`.

### Yanıt çerezi

```ahd
return HTTP.redirect("/").withCookie(HTTP.cookie("theme", "dark"))
```

`HTTP.deleteCookie("theme")` tarayıcıya unutmasını söyler. `withCookie` ile
gönderin. Çerezler değişmezdir: `withHttpOnly(true)` yeni bir Cookie döndürür.

### SessionStore, String değerler, açık commit

```ahd
sessions: SessionStore := HTTP.sessions()

count: Function := (request: Request) -> Response {
    sessions: Global SessionStore
    session: Local Session := sessions.open(request)
    raw: Local String? := session.get("count")
    value: Local Int := 0
    if raw != null {
        value = int(raw)
    }
    value = value + 1
    session.set("count", str(value))
    return sessions.commit(session, HTTP.text("count = {str(value)}"))
}
```

`set` başlık yazmaz. `commit` yeni bir Response döndürür. Değerler yalnızca
String'dir; `int(...)` sıradan dil dönüşümüdür.

`set`/`rotate` olmadan `open` kalıcı oturum veya `Set-Cookie` üretmez.

### Giriş tarzı rotate ve çıkış destroy

Bir adı kabul ettikten sonra (veya daha sonra parolayı kendiniz doğruladıktan
sonra):

```ahd
session.rotate()
session.set("name", name)
return sessions.commit(session, HTTP.redirect("/panel"))
```

`rotate()` eski çerez kimliğini işe yaramaz kılar. Çıkışta:

```ahd
session.destroy()
return sessions.commit(session, HTTP.redirect("/"))
```

`remove` bir anahtarı siler. `clear` değerleri boşaltır ama oturumu tutar.
`destroy` sunucu oturumunu ve tarayıcı çerezini siler.

### İki tarayıcı, sonra yeniden başlatma

[Oturum giriş örneğini](../examples/v0.5/03_session_login.ahd) iki tarayıcıda
(veya bir pencere ve bir gizli pencerede) açın. Birinde Ali, diğerinde Mehmet
olarak devam edin. İkisi de oturumda kalır. Ali çıkış yapsın: Mehmet hâlâ
Mehmet'tir.

Programı durdurup yeniden başlatın. Aynı tarayıcı çerezi adı geri getirmez.
Bellek oturumları tasarım gereği kaybolur. Kendinizin sakladığı SQLite
satırları duruyor olabilir.

**Siz deneyin:** `examples/v0.5/02_session_counter.ahd` çalıştırın, iki
tarayıcıda yenileyin, sayıların bağımsız kaldığını görün.

Ayrıntılar için [HTTP modül referansına](HTTP_TR.md) bakın.

## 38. HTTP Client

v0.5.0 bir tarayıcının değerini sakladı. v0.6.0 bir AhdCode programının dış
bir HTTP veya HTTPS hizmetini çağırmasını sağlar. Sunucu türleri (`Request`,
`Response`) ile istemci türleri (`ClientRequest`, `ClientResponse`) ayrıdır.
Yapay zeka satıcı modülü yoktur. JSON'u mevcut JSON modülüyle üretir, String'i
kendiniz gönderirsiniz.

### Basit HTTPS GET

```ahd
bring HTTP
from HTTP bring Client
from HTTP bring ClientResponse

client: Client := HTTP.client()
response: ClientResponse := client.get("https://example.com/")
write(str(response.status()))
write(response.body())
```

HTTPS, sistemin güvenilen kökleriyle sertifikayı doğrular. Doğrulamayı kapatan
bir seçenek yoktur.

### Özel istek, başlıklar, POST

```ahd
bring HTTP
from HTTP bring Client
from HTTP bring ClientRequest
from HTTP bring ClientResponse

client: Client := HTTP.client()
request: ClientRequest := HTTP.clientRequest("POST", "https://example.com/")
request = request.withHeader("Content-Type", "text/plain; charset=utf-8")
request = request.withBody("hello")
response: ClientResponse := client.send(request)
```

`withHeader` o adı değiştirir. `addHeader` başka bir değer ekler. Orijinal
istek değişmez.

### JSON API ve Env jetonu

```ahd
bring HTTP
from HTTP bring Client
from HTTP bring ClientRequest
from HTTP bring ClientResponse
bring JSON
from JSON bring JSONValue
bring Env

client: Client := HTTP.client()
token: String := Env.getOr("API_TOKEN", "")
payload: JSONValue := JSON.object({"question": JSON.fromString("2+2")})
request: ClientRequest := HTTP.clientRequest("POST", "https://api.example.com/v1/chat")
request = request.withHeader("Authorization", "Bearer {token}")
request = request.withHeader("Content-Type", "application/json")
request = request.withBody(JSON.stringify(payload))
response: ClientResponse := client.send(request)
parsed: JSONValue := JSON.parse(response.body())
```

Bu genel bir API şeklidir. AhdCode bir OpenAI, Anthropic veya Gemini modülü
göndermez.

### Hatalar, zaman aşımı, 4xx/5xx

`401`, `429` ve `500` yine `ClientResponse` döner. `status()` ve `body()`'yi
kendiniz okursunuz. Taşıma sorunları — bozuk URL, DNS, TLS, zaman aşımı, çok
büyük veya UTF-8 olmayan gövde — `HTTPError` fırlatır.

```ahd
bring HTTP
from HTTP bring Client
from HTTP bring ClientResponse
from HTTP bring HTTPError

client: Client := HTTP.client(1)
attempt {
    response: Local ClientResponse := client.get("https://example.com/")
    if response.status() >= 400 {
        write("API durumu {str(response.status())}")
    } else {
        write(response.body())
    }
} except HTTPError as error {
    write(error.message)
}
```

`HTTP.client(1)` tüm istek için en fazla bir saniye bekler.

Ayrıntılar için [HTTP modül referansına](HTTP_TR.md) ve
[`examples/v0.6`](../examples/v0.6/README_TR.md) bakın.

URL parçaları, 4xx/5xx ile taşıma hatası farkı, JSON POST, Env jetonu ve
istemci güvenlik kontrol listesi için
[HTTP/HTTPS atölyesini](PRACTICAL_MODULES_TR.md#7-http-ve-https-istek-yanıt-ve-hata-sınırı)
tamamlayın.

## 39. HTML ayrıştırma ve web kazıma (scraping)

36. bölüm `HTML`'i sayfa *kurmak* için kullandı. v0.7.0 diğer yönü ekliyor:
bir yerden aldığınız HTML metnini arayabileceğiniz bir ağaca dönüştürmek.
`HTML.parse` kendi başına **hiçbir şey indirmez** -- yalnızca elinizde
zaten olan String'i okur.

### Sabit bir String'i ayrıştırmak

```ahd
bring HTML

document := HTML.parse("<h1>Hello &amp; AhdCode</h1>")
heading := document.first("h1")
if heading != null {
    write(heading.text())
}
```

`document.first(selector)`, eşleşen ilk öğeyi ya da hiçbir şey eşleşmezse
`null` döndürür -- `heading`'in `if ... != null` koruması gerektirmesinin
sebebi budur, tıpkı başka herhangi bir null olabilir değer gibi. `text()`
size öğenin metin içeriğini verir; `&amp;` zaten `&`'e çözülmüş olarak.

### Öznitelikler ve `select`

```ahd
bring HTML

document := HTML.parse("<a href=\"/notes/1\" class=\"link\">Read</a>")
link := document.first("a")
if link != null {
    write(link.tag())        // "a" -- tag() her zaman küçük harflidir
    write(link.hasAttr("href"))  // true
    href := link.attr("href")
    if href != null {
        write(href)           // "/notes/1" -- asla mutlak bir URL'ye dönüştürülmez
    }
}

cards := document.select(".card")
write(str(len(cards)))
```

`select(selector)`, her eşleşmeyi belgedeki sırasıyla bir
`List<HTMLElement>` olarak döndürür. `selector`, küçük bir CSS benzeri
örüntü kümesini anlar: bir etiket adı, `#id`, `.class`, `[attr]`,
`[attr="değer"]`, bunların birleşimleri, "içinde bir yerde" için boşluk
(`article a`), "doğrudan çocuk" için `>` (`article > h2`) ve "bunlardan
biri" için virgül (`h1, h2`). Bunların dışındaki her şey --
`:nth-child(...)`, `+`, `~` vb. -- ne demek istediğinizi tahmin etmek
yerine `HTMLError` fırlatır.

### İç içe seçim

Bir `HTMLElement` üzerinde `.select`/`.first` çağırmak yalnızca *o öğenin
içini* arar, tüm belgeyi değil:

```ahd
articles := document.select("article.card")
firstArticle := articles[0]
title := firstArticle.first("h2")
```

`title` yalnızca `firstArticle`'ın içinden gelebilir -- farklı bir
makaledeki eşleşen bir `h2` asla döndürülmez.

### HTTP Client + HTML.parse: gerçek bir sayfayı ALMAK VE ayrıştırmak

`HTML.parse` ile 38. bölümdeki HTTP Client, doğal olarak birleşen iki ayrı
araçtır. Sayfayı almak bir çağrı, onu ayrıştırmak başka bir çağrıdır:

```ahd
bring HTTP
from HTTP bring Client
from HTTP bring ClientResponse
bring HTML
from HTML bring HTMLDocument
from HTML bring HTMLElement

client: Client := HTTP.client()
response: ClientResponse := client.get("https://example.com/notes")
document: HTMLDocument := HTML.parse(response.body())

articles: List<HTMLElement> := document.select("article.card")
for article in articles {
    heading: Local := article.first("h2")
    if heading != null {
        write(heading.text())
    }
}
```

**Teknik not.** `HTML.parse` kendi başına asla bir ağ isteği yapmaz -- hiç
URL argümanı bile yoktur. Ayrıca hiçbir şeyi asla çalıştırmaz: sayfadaki bir
`<script>` etiketi, `HTML.parse` için tıpkı bir `<p>` etiketi gibi yalnızca
metindir. AhdCode içinde tarayıcı, JavaScript motoru veya DOM yoktur --
yalnızca `select`/`first` ile arayabileceğiniz bir ağaç vardır.

### Kazıdığınızı kaydetmek

35. bölüm `SQLite`'ı tanıttı. İkisi doğrudan birleşir: `HTML` ile
çıkarın, `SQLite` ile saklayın, tıpkı öncekiyle aynı şekilde bağlı
parametreler kullanarak -- asla String birleştirmesiyle değil:

```ahd
db: Database := SQLite.open("scraped.db")
db.execute("""
    CREATE TABLE IF NOT EXISTS notes (
        id TEXT PRIMARY KEY,
        title TEXT NOT NULL,
        href TEXT NOT NULL
    )
    """)

for article in articles {
    heading: Local := article.first("h2")
    link: Local := article.first("a")
    if heading != null {
        if link != null {
            href: Local String? := link.attr("href")
            if href != null {
                db.execute(
                    "INSERT OR REPLACE INTO notes (id, title, href) VALUES (?, ?, ?)"
                    [SQLite.fromString(href), SQLite.fromString(heading.text()), SQLite.fromString(href)]
                )
            }
        }
    }
}
```

Ayrıntılar için [HTML modül referansına](HTML_TR.md) ve
[`examples/v0.7`](../examples/v0.7/README_TR.md) bakın.

Güvenli dinamik sayfa, seçici kapsamı, null kontrolü ve HTTPS ile alınan
sayfayı ayrıştırma akışı için
[HTML atölyesine](PRACTICAL_MODULES_TR.md#8-html-güvenli-sayfa-kurmak-ve-belge-ayrıştırmak)
geçin.

## 40. Dosya yüklemeleri

Dosya taşıyan bir `<form>` bunu `enctype="multipart/form-data"` ile
belirtmelidir. Metin alanları yine `request.form` ile, dosyalar
`request.file` ile gelir.

### Form

```ahd
HTML.element("form", {"method": "post", "action": "/upload", "enctype": "multipart/form-data"}, [
    HTML.element("input", {"type": "text", "name": "title"}, [])
    HTML.element("input", {"type": "file", "name": "paper", "accept": "application/pdf"}, [])
    HTML.element("button", {"type": "submit"}, [HTML.text("Yükle")])
])
```

### Yüklemeyi okumak

```ahd
upload: Function := (request: Request) -> Response {
    title: Local String? := request.form("title")
    paper: Local UploadedFile? := request.file("paper")
    if paper == null {
        return HTTP.text("lütfen bir dosya seçin", 400)
    }
    write(paper.originalName())          // "paper.pdf" - yalnızca görüntüleme
    write(str(paper.size()))             // tam bayt sayısı
    return HTTP.text("alındı")
}
```

`request.file(name)`, o alan dosya taşımadığında `null`'dır; bu yüzden her
zamanki `!= null` koruması gerekir. `<input type="file" name="papers" multiple>`
için `request.files("papers")` tüm dosyaları sırasıyla verir.

### İki içerik türü, yalnızca birine güvenebilirsiniz

```ahd
paper.declaredContentType()   // tarayıcının iddiası
paper.detectedContentType()   // baytların gerçekte neye benzediği
```

Herkes `virus.exe` dosyasının adını `paper.pdf` yapıp `application/pdf`
iddia edebilir. Uzantı ve bildirim, istekle birlikte gelen sadece birer
metindir. Yalnızca `detectedContentType()` dosyanın kendi baytlarına bakar:

```ahd
if paper.detectedContentType() != "application/pdf" {
    return HTTP.text("bu bir PDF değil", 415)
}
```

**Teknik not.** Algılama yaygın biçimlerin *şeklini* tanır. Virüs taraması
yapmaz ve bir PDF'in açılmasının güvenli olduğunu kanıtlamaz.

### Güvenli kaydetmek

Bu bölümün en önemli kuralı:

```ahd
// ASLA böyle yapmayın
storedPath := "uploads/" + paper.originalName()
```

O adı yükleyen kişi seçer. `../../etc/passwd` gibi bir ad klasörünüzün
dışına yazardı ve `paper.pdf` yükleyen iki kişi birbirinin dosyasını ezerdi.
Bunun yerine şunu yapın:

```ahd
storedPath := paper.save("uploads/papers")
```

`save`, *sizin* belirttiğiniz dizinin içinde rastgele bir dosya adı seçer,
mevcut bir dosyayı asla ezmez ve gerçekten kullandığı yolu döndürür; örneğin
`uploads/papers/8e8f30c65c4d4d23...`. Her yükleme bir kez kaydedilebilir.

### Dosya diskte, bilgiler veritabanında

```ahd
db.execute(
    "INSERT INTO papers (title, original_name, stored_path, detected_content_type, size) VALUES (?, ?, ?, ?, ?)"
    [
        SQLite.fromString(title)
        SQLite.fromString(paper.originalName())
        SQLite.fromString(storedPath)
        SQLite.fromString(paper.detectedContentType())
        SQLite.fromInt(paper.size())
    ]
)
```

PDF'in kendisi `uploads/papers/` içinde kalır; veritabanı yalnızca nerede
olduğunu ve ne olduğunu tutar. Bir yüklemeyi hiç kaydetmezseniz, istek
bittiğinde otomatik olarak silinir.

### Dosyayı geri sunmak

Kaydetmek hikayenin yalnızca yarısı -- seminer öğrencisinin PDF'i daha sonra
tekrar açması da gerekir. Depolama ve sunum iki ayrı meseledir: saklanan
yolun kasıtlı olarak uzantısı yoktur (`save` asla bir uzantı uydurmaz), ama
tarayıcının dosyayı nasıl göstereceğini bilmesi için yine de bir
`Content-Type` gerekir.

```ahd
paper: Function := (request: Request) -> Response {
    id: Local String? := request.query("id")
    if id == null {
        return HTTP.text("id gerekli", 400)
    }
    row: Local Pair<String, SQLiteValue>? := lookupPaper(int(id))
    if row == null {
        return HTTP.text("bulunamadı", 404)
    }
    found: Local Pair<String, SQLiteValue> := row
    return HTTP.file(found["stored_path"].string(), "application/pdf")
}
```

`HTTP.file`, baytları doğrudan diskten tarayıcıya akıtır; onları hiçbir
zaman bir AhdCode `String`'ine dönüştürmez, bu yüzden bir PDF'in ikili
içeriği bozulmadan kalır. `contentType` her zaman sizin kararınızdır --
AhdCode bunu asla yoldan veya dosyanın kendi baytlarından tahmin etmez.

Tarayıcının dosyayı göstermek yerine indirmesini sağlamak, ve ona saklanan
opak addan daha dostane bir ad vermek için `HTTP.download` kullanın:

```ahd
return HTTP.download(
    found["stored_path"].string()
    "application/pdf"
    found["original_name"].string()
)
```

Sunum adı, saklanan yoldan tamamen bağımsızdır: yükleyenin kullandığı
orijinal ad dahil herhangi bir şey olabilir ve dosya sistemine asla
dokunmaz. AhdCode bunu yanıt başlığı için güvenle kodlar; boşluk içeren
veya `Özet Çalışması.pdf` gibi Türkçe karakterli bir ad için bile.

**Teknik not.** İstekten gelen bir yolu doğrudan `HTTP.file`/`HTTP.download`'a
asla geçirmeyin. Yukarıdaki `lookupPaper` gibi önce kendi veritabanınızdan
arayın; böylece bir istek yalnızca uygulamanızın zaten bildiği bir dosyayı
adlandırabilir -- diskteki rastgele bir yolu değil.

### Sunucunuzu durdurmak

`ahdcode run app.ahd` çalışırken kaynağınızın yanında bir `app.run` dosyası
tutar. Uygulamayı durdurmak için portu veya süreç kimliğini bulmanız gerekmez:

```bash
ahdcode kill app.run
```

Ayrıntılar için [HTTP modül referansına](HTTP_TR.md) ve yükleme rehberi
için [`examples/v0.8`](../examples/v0.8/README_TR.md) bakın; dosyayı
yükledikten sonra `HTTP.file`/`HTTP.download` ile geri sunmak için
[`examples/v0.9.1`](../examples/v0.9.1/README_TR.md) bakın.

## 41. E-posta gönderme (SMTP)

AhdCode, SMTP üzerinden gerçek e-posta gönderebilir. Bu yalnızca gönderimdir:
gelen kutusu, IMAP ve dosya eki henüz yoktur.

Gerçek bir SMTP parolasını kaynağa yazmayın. `Env` ile okuyun.

### 1. İstemci oluşturmak

```ahd
bring SMTP
from SMTP bring SMTPClient
from SMTP bring SMTPMessage
from SMTP bring SMTPError

client: SMTPClient := SMTP.client("127.0.0.1", 2525, "none")
```

`SMTP.client` bağlanmaz. Ağ yalnızca `send` çağrıldığında kullanılır.

### 2. Güvenlik kipleri

Tam küçük harf değeri kullanın:

- `"starttls"` — bağlan, STARTTLS iste, TLS doğrula (varsayılan)
- `"tls"` — hemen TLS, örtük TLS
- `"none"` — açık düz metin, yerel test sunucusu için

STARTTLS isterseniz ve sunucu onu ilan etmezse `SMTPError` alırsınız. AhdCode
sessizce düz metne devam etmez.

### 3. İleti

```ahd
message: SMTPMessage := SMTP.message(
    "sender@example.com"
    ["student@example.com"]
    "AhdCode Semineri"
)
```

Her liste öğesi bir postadır. Tek bir String olarak
`"a@example.com, b@example.com"` yazmayın.

### 4. Metin postası

```ahd
message = message.withText("Merhaba AhdCode")
```

### 5. HTML postası

```ahd
message = message.withHtml("<p>Merhaba <strong>Hatay</strong></p>")
```

İkisi de ayarlanırsa posta `multipart/alternative` olur (önce metin, sonra
HTML). SMTP HTML'i **temizlemez**. İçerik kullanıcıdan geldiyse uygulamanız
önce onu temizlemelidir.

### 6. To, Cc ve Bcc

```ahd
message = message.withCc(["bob@example.com"])
message = message.withBcc(["secret@example.com"])
```

Bcc alıcıları SMTP sunucusuna zarf alıcısı olarak gönderilir, ancak posta
DATA'sına **Bcc başlığı yazılmaz**. Diğer alıcılar Bcc adresini görmez.

### 7. Reply-To

```ahd
message = message.withReplyTo("reply@example.com")
```

`withReplyTo` tekrar çağrılırsa önceki değeri değiştirir.

### 8. Env kimlik bilgileri

```ahd
bring Env

host := Env.getOr("SMTP_HOST", "127.0.0.1")
port := int(Env.getOr("SMTP_PORT", "2525"))
security := Env.getOr("SMTP_SECURITY", "starttls")
username := Env.getOr("SMTP_USERNAME", "")
password := Env.getOr("SMTP_PASSWORD", "")

client := SMTP.client(host, port, security)
if username != "" {
    client = client.withPlainAuth(username, password)
}
```

v0.9'daki tek kimlik doğrulama AUTH PLAIN'dir. Düz metinde (`"none"`) parola
gönderilmeden reddedilir.

### 9. STARTTLS ve TLS

Gerçek bir posta sunucusu için 587'de `"starttls"` veya 465'te `"tls"` tercih
edin. Çalışma zamanı sertifika zincirini ve makine adını sistem güvenine göre
doğrular. "Doğrulamayı atla" anahtarı yoktur.

### 10. SMTPError

```ahd
attempt {
    client.send(message)
    write("sent")
} except SMTPError as error {
    write(error.message)
}
```

Reddedilen bir alıcı, TLS hatası, zaman aşımı veya eksik gövde `SMTPError`
yükseltir. Bir `send` bir işlemdir: AhdCode yeniden denemez, çünkü yeniden
deneme aynı postayı iki kez teslim edebilir.

Ayrıntılar için [SMTP modül referansına](SMTP_TR.md) ve
[`examples/v0.9`](../examples/v0.9/README_TR.md) bakın.

## 42. Kod biçimlendirici (Formatter)

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

## 43. Komut satırı (CLI)

AhdCode'u terminalden birkaç temel komutla kullanabilirsiniz. Yeni
başlıyorsanız en çok kullanacağınız ikisi `ahdcode run` ve — daha büyük bir şey
kurmaya başladığınızda — `ahdcode dev` olacaktır.

### Çalıştırmak ve derlemek

```bash
ahdcode run file.ahd
```

Programı derleyip çalıştırır.

```bash
ahdcode build file.ahd
```

Programı kendi başına çalışan yerel bir executable'a dönüştürür ve yolunu
yazar. Adı seçmek için `-o ad` ekleyin.

```bash
ahdcode format file.ahd
```

Dosyayı ortak stile göre biçimlendirir (42. bölüm).

### Sürekli çalışan bir program üzerinde çalışmak

Bir web uygulaması bitmez — istek bekler. Her düzenlemeden sonra elle yeniden
başlatmak çabuk sıkıcı olur; bu yüzden:

```bash
ahdcode dev app.ahd
```

`dev` programı derler, başlatır ve `require(...)` çizgesindeki her dosyayı
izler. Her kayıtta yeniden derler:

- derleme **başarılıysa**, çalışan program yenisiyle değiştirilir;
- derleme **başarısızsa**, hatalar yazılır ve *son çalışan sürüm çalışmaya
  devam eder*. Bozuk bir kayıt sitenizi asla düşürmez;
- program kendiliğinden çökerse `dev` bunu söyler ve zaten bozuk olduğunu
  bildiği bir ikili dosyayı yeniden denemek yerine bir sonraki kaydınızı
  bekler.

Durdurmak için Ctrl+C.

### Başka yerde başlattığınız bir şeyi durdurmak

`run` veya `dev` çalışırken AhdCode, programınızın yanına ona nasıl
ulaşılacağını söyleyen küçük bir dosya bırakır: `app.run` veya `app.dev`. Başka
bir terminalden:

```bash
ahdcode stop app.dev     # düzgün kapanmasını iste ve bunu bekle
ahdcode kill app.run     # hemen durdur
```

`stop` kibar olanıdır: ister, sonra sürecin gerçekten çıktığını doğrulamak için
bekler. `kill` beklemez. İkisi de bir süreç kimliği aramaz — bu bilinçlidir ve
nedenini [CLI rehberi](CLI_TR.md) anlatır.

Kaynak adını verip hangi oturumun çalıştığını AhdCode'un bulmasını da
sağlayabilirsiniz:

```bash
ahdcode stop app.ahd
```

### Proje başlatmak ve verinize bakmak

```bash
ahdcode init web        # bu klasöre çalışan bir web starter yaz
ahdcode databases       # yerel veritabanı çalışma alanı AhdDataStudio'yu aç
ahdcode local status    # bu makinenin sunduğu yerel adresler
```

54, 55 ve 56. bölümler bunları kullanır.

### Editör desteği ve yardım

```bash
ahdcode lsp
```

Editörlerin kullandığı dil sunucusunu başlatır: tanılamalar, hover, otomatik
import'lu completion, tanıma gitme ve referans bulma, rename, signature help,
semantik renklendirme, inlay hints, quick fix'ler ve biçimlendirme. Normalde
bunu kendiniz yazmazsınız — editörünüz başlatır. Bkz.
[dil sunucusu rehberi](LSP_TR.md).

```bash
ahdcode --help
ahdcode --version
```

Yardım ve sürüm bilgisini gösterir. `--help` komut yüzeyinin tamamını listeler;
[CLI rehberi](CLI_TR.md) her birini ayrıntılı anlatır.

## 44. Etkileşimli kabuk (REPL)

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

## 45. Sık yapılan başlangıç hataları

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

**22. UUID'leri `==` ile karşılaştırmak**
- Yanlış: `if kimlik == digerKimlik { ... }`
- Neden: `==` iki değişkenin aynı nesneyi gösterip göstermediğini sorar. Aynı metinden ayrı ayrı ayrıştırılmış iki UUID bile `false` verir.
- Doğru: `if kimlik.equals(digerKimlik) { ... }`

**23. Bir UUID'yi `str(...)` ile metne çevirmek**
- Yanlış: `write("kimlik: " + str(kimlik))`
- Neden: `str`, yerleşik sınıflar için `<UUIDValue>` yazar; UUID'nin kendisini değil.
- Doğru: `write("kimlik: " + kimlik.string())`

**24. PostgreSQL'de `?` yer tutucusu kullanmak**
- Yanlış: `db.query("SELECT ad FROM ogrenciler WHERE eposta = ?", [PostgreSQL.fromString(eposta)])`
- Neden: PostgreSQL'in yer tutucuları numaralıdır. `?` onun için bir yer tutucu değildir; sorgu `(42601) syntax error` gibi bir hatayla reddedilir.
- Doğru: `WHERE eposta = $1`

**25. PostgreSQL işleminde bir hatadan sonra devam etmek**
- Yanlış: Bir `islem.execute(...)` hatasını yakalayıp aynı işlemle devam etmek ve sonunda `commit()`'in kaydetmesini beklemek.
- Neden: PostgreSQL ilk hatadan sonra işlemi iptal eder. Sonraki her ifade `(25P02)` hatası verir ve `commit()` hiçbir şey kaydetmez.
- Doğru: Hatayı yakalayınca `rollback()` çağırın; gerekiyorsa yeni bir `db.begin()` ile baştan başlayın.

**26. WebSocket bağlantısını kayıttan silmeyi unutmak**
- Yanlış: `withOpen` içinde bağlantıyı bir `Pair`'e eklemek ama `withClose` yazmamak.
- Neden: Kapanan bağlantılar kayıtta kalır, `send` onlar için `false` döner ve kayıt sürekli büyür.
- Doğru: `withClose` içinde `kayit = KeyValue.without(kayit, socket.id())`.

**27. Bir WebSocket geri çağrısında uzun süren iş yapmak**
- Yanlış: Mesaj geri çağrısının içinde saniyelerce süren bir iş yapmak, örneğin yavaş bir dış API'yi beklemek.
- Neden: Aynı sunucudaki geri çağrılar ve HTTP işleyicileri birer birer çalışır. Biri beklerse bütün istekler ve bağlantılar bekler.
- Doğru: Geri çağrıları kısa tutun. Veritabanına kısa bir sorgu sorun değildir; uzun beklemek sorundur.

## 46. Küçük Projeler

Bu küçük projeler rehberde öğretilenleri bir araya getirir. Onları tek başınıza kurmayı deneyin!

1. **Not Ortalaması Hesaplayıcı**: Kullanıcıdan 5 not isteyin. Onları bir `List<Int>` içine koyun. Geçersiz notları (0'dan küçük veya 100'den büyük) listeden çıkarın (filter). Kalan notların ortalamasını, minimum ve maksimum değerini, son olarak da öğrencinin geçip (ortalama >= 50) geçmediğini yazdırın.
2. **Basit Hesap Makinesi**: İki sayı ve bir operatör (`+`, `-`, `*`, `/`) almak için `take()` kullanın. İşlemi seçmek için operatör üzerinde `state` kullanın ve sonucu yazdırın. Sıfıra bölünme ihtimalini `attempt`/`except` ile yönetin.
3. **Sayı İstatistikleri**: `Math.randomInt(1, 100)` ile 100 adet rastgele sayı üretin. Bunlardan kaç tanesinin tek, kaç tanesinin çift olduğunu sayın (count) ve listeyi sıralayın. Bir sayının asal olup olmadığını kontrol eden bir fonksiyon yazın ve listeyi sadece asalları gösterecek şekilde filtreleyin (filter).
4. **Kelime Analizi**: Kullanıcıdan bir cümle girmesini isteyin. Kelimeleri ayırmak için `split(" ")` kullanın. Kelime sayısını bulun, en uzun kelimeyi bulun ve her kelimenin kendi uzunluğuyla eşleştiği bir `Pair<String, Int>` oluşturun.
5. **Menülü Program**: Bir `until` döngüsü kullanarak küçük bir banka simülasyonu yapın. Bir menü gösterin: 1. Para Yatır, 2. Para Çek, 3. Bakiye, 0. Çıkış. Bakiyeyi bir `Int` içinde saklayın ve kullanıcı 0 girene kadar programı döndürün.
6. **Sınıflarla (Class) Öğrenci Kaydı**: Bir `Student` sınıfı ve bir `Course` (Kurs) sınıfı oluşturun. Course içinde bir `List<Student>` bulunsun. Kursa yeni bir öğrenci eklemek için bir metot, kursun genel not ortalamasını hesaplamak için başka bir metot yazın.
7. **Tohumlu (Seeded) Rastgele Oyun**: `Math.seed(42)` kullanarak 1 ile 100 arasında "gizli bir sayı" üretin. Kullanıcıdan sayıyı tahmin etmesini isteyin. Doğru tahmin edene kadar "daha yüksek" veya "daha düşük" diye yönlendirin. Tohum kullanıldığı için, gizli sayı programı her çalıştırdığınızda aynı olacaktır—test yapmak için mükemmel!
8. **SQLite Not Defteri**: `notes.db` açın, yoksa bir `notes` tablosu oluşturun ve kullanıcının not eklemesine, listelemesine, başlığa göre aramasına, güncellemesine ve silmesine izin verin. Her değer için `?` parametreleri kullanın. Programı kapatıp yeniden çalıştırın: eski notlar durmalıdır.
9. **Web Not Defteri**: Notları `127.0.0.1` üzerinde bir tarayıcıda sunun. Notları `HTML.text` ile listeleyin, POST `/notes` ve bağlı SQLite parametreleriyle not ekleyin, sonra `/` adresine yönlendirin. Dinamik metin ham HTML'e birleştirilmemelidir.
10. **Canlı Sohbet Odası**: 60.4'teki sohbet odasını geliştirin. Mesajları JSON olarak gönderin (`tur`, `ad`, `metin`), tarayıcıda katılma/ayrılma bildirimlerini farklı renkte gösterin, boş ya da 300 karakterden uzun mesajları reddedin ve `/kim` komutuyla odadakileri listeleyin.
11. **PostgreSQL Not Defteri**: 59. bölümdeki `ogrenciler` tablosuna bir `notlar` tablosu ekleyin (`id uuid`, `ogrenci_id uuid REFERENCES ogrenciler(id)`, `ders text`, `puan numeric(5, 2)`). Öğrenci ve not ekleyen, her öğrencinin ortalamasını SQL'de `avg` ile hesaplayan ve bir sınıfın bütün notlarını tek bir işlemde (transaction) kaydeden bir program yazın.
12. **Canlı Anket**: Seçenekleri ve oyları PostgreSQL'de tutan bir anket sayfası kurun. Bir oy `POST /oy` ile kaydedildikten sonra güncel sayıları JSON olarak bütün açık sayfalara WebSocket ile gönderin; sayfa yenilenmeden sayılar değişsin.

## 47. Egzersizler

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
22. `127.0.0.1` üzerinde `GET /ok` için `HTTP.text("ok")` sunun ve tarayıcıda açın.
23. Genel bir HTTPS sayfasında `HTTP.client().get` kullanın, `status()` yazın ve `HTTPError` ile 404 `ClientResponse` ayrımını yapın.

### v1.4.0: UUID, gizli değerler, PostgreSQL ve WebSocket
24. Beş `UUID.v4()` üretin. Her birinin `version()` değerinin 4 olduğunu ve hiçbir ikisinin `equals` ile eşit olmadığını doğrulayın.
25. Kullanıcıdan `take()` ile bir kimlik isteyin. Geçerliyse küçük harfli biçimini, değilse "geçersiz kimlik" yazdırın. Hiçbir girdide program hatayla durmasın.
26. `Env.secret("SMTP_PASSWORD")` okuyan bir program yazın: değer yoksa açıklayıcı bir mesaj, varsa yalnızca uzunluğunu yazdırsın. Önce değişkenle, sonra `SMTP_PASSWORD_FILE` ile deneyin.
27. PostgreSQL'de bir `kitaplar` tablosu oluşturun (`id uuid`, `ad text`, `fiyat numeric(8, 2)`), üç kitap ekleyin ve en pahalı kitabı `ORDER BY fiyat DESC LIMIT 1` ile bulun.
28. 27'deki tabloda bütün fiyatlara %10 indirim uygulayın, `affectedRows()` değerini yazdırın, sonra `round(avg(fiyat), 2)` ile yeni ortalamayı okuyun.
29. Bir işlemde (transaction) iki kitap ekleyin; ikincisinin `id` değeri için bilerek `"kimlik-degil"` gönderin. `rollback()` sonrasında tabloda kaç kitap olduğunu yazdırın.
30. `/buyuk` yolunda, bağlantı açılır açılmaz "merhaba" gönderen ve gelen her mesajı büyük harfe çevirip geri yollayan bir WebSocket sunucusu yazın.
31. 60.4'teki sohbet odasına, `POST /yonetici` ile gelen metni herkese `[yönetici]` önekiyle gönderen bir HTTP işleyicisi ekleyin. İsteği 60.5'teki gibi bir belirteçle koruyun.

## 48. Çözüm İpuçları

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
22. `HTTP.server("127.0.0.1", 8080)`, `app.get("/ok", handler)`, `HTTP.text("ok")`, sonra `app.start()`.
23. `HTTP.client()`, `client.get("https://example.com/")`, `response.status()`. 404 hâlâ `ClientResponse`; TLS veya zaman aşımı `HTTPError`.
24. Kimlikleri bir `List<UUIDValue>` içine koyun; iç içe iki döngüyle her çifti `equals` ile karşılaştırın (bir kimliği kendisiyle karşılaştırmayın).
25. `UUID.isValid(metin)` hiçbir zaman hata vermez. Önce onu sorun, sonra `UUID.parse(metin).string()` yazdırın.
26. `Env.secret` sonucu `String?`'dir; `null` kontrolünden sonra `len(...)` kullanın. Dosyayla denerken `Env.unset("SMTP_PASSWORD")` ile değişkeni kaldırmayı unutmayın.
27. `$1::uuid` ve `$3::numeric` dönüşümlerini yazın. `fiyat` sütunu `string()` ile okunur.
28. `UPDATE kitaplar SET fiyat = fiyat * $1::numeric` ve `PostgreSQL.fromString("0.90")`.
29. 59.7'deki kalıbı kullanın: `attempt` içinde iki `execute` ve `commit()`, `except PostgreSQLError` içinde `rollback()`. Sayı, işlemden önceki sayıyla aynı çıkmalıdır.
30. `withOpen` içinde `socket.send("merhaba")`, mesaj geri çağrısında `socket.send(metin.upper())`.
31. HTTP işleyicileri ve WebSocket geri çağrıları aynı kilidi paylaşır; işleyiciden `herkeseGonder` çağırabilirsiniz. Belirteci `Env.secret` ile okuyup `Security.secureEqual` ile karşılaştırın.

## 49. Sonraki adımlar ve teknik belgeler

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
- [MySQL](MYSQL_TR.md)
- [PostgreSQL](POSTGRESQL_TR.md)
- [HTTP](HTTP_TR.md)
- [WebSocket](WEBSOCKET_TR.md)
- [Security](SECURITY_TR.md)
- [HTML](HTML_TR.md)
- [XML](XML_TR.md)
- [Env](ENV_TR.md)
- [Lists](LISTS_TR.md)
- [KeyValue](KEYVALUE_TR.md)
- [UUID](UUID_TR.md)
- [File ve Path](FILESYSTEM_TR.md)
- [Regex](REGEX_TR.md)
- [CSV](CSV_TR.md)
- [Data](DATA_TR.md)
- [Web](WEB_TR.md)
- [require(...)](REQUIRE_TR.md)
- [Tanılamalar](DIAGNOSTICS_TR.md)
- [CLI](CLI_TR.md)
- [Env](ENV_TR.md)
- [AhdDataStudio](../tools/AhdDataStudio/README_TR.md)
- [Formatter](FORMATTER_TR.md)
- [REPL](REPL_TR.md)
- [Dil sunucusu](LSP_TR.md)
- [Uygulamalı Modül Atölyeleri](PRACTICAL_MODULES_TR.md)
- [Tam v0.1 spesifikasyonu](../AHDCODE_LANGUAGE_SPEC_v0.1_TR.md)

Çalışan daha fazla örnek için [derlenmiş v0.1 örnekleri](../examples/v0.1/README_TR.md)
klasörüne, [v0.3 SQLite Not Defteri](../examples/v0.3/README_TR.md),
[v0.4 Web Not Defteri](../examples/v0.4/README_TR.md),
[v0.5 çerezler ve oturumlar](../examples/v0.5/README_TR.md),
[v0.6 HTTP Client](../examples/v0.6/README_TR.md),
[v0.7 HTML ayrıştırma ve web kazıma](../examples/v0.7/README_TR.md),
[v0.8 dosya yükleme](../examples/v0.8/README_TR.md),
[v0.9 SMTP e-posta](../examples/v0.9/README_TR.md),
[v0.9.1 ikili-güvenli HTTP dosya yanıtları](../examples/v0.9.1/README_TR.md),
[v0.10 Security örnekleri](../examples/v0.10/README.md),
[v0.11 MySQL örnekleri](../examples/v0.11/README_TR.md),
[v0.12 çekiliş uygulaması](../examples/v0.12/README_TR.md),
[v0.14 çok dosyalı Web uygulaması](../examples/v0.14/multi_file_web),
[v0.15 Web uygulamaları](../examples/v0.15/ahd_academi),
[v0.16 form ve doğrulama örneği](../examples/v0.16/forms_validation/README_TR.md),
[v0.17 rota ve bekçi örneği](../examples/v0.17/routes_guards),
[v0.18 Web starter'ları](../examples/v0.18/README_TR.md) ve
[v1.4 gerçek zamanlı yoklama uygulaması](../examples/v1.4/realtime_attendance/README_TR.md)
sayfalarına bakın.

## 50. Güvenlik: parola hashleme ve güvenli belirteçler

`Security` modülü (v0.10.0) üç odaklı araç sunar: Argon2id parola hashleme,
opak rastgele belirteçler ve sabit zamanlı dizi karşılaştırma.

### 50.1 Parola hashleme

```ahd
bring Security
from Security bring SecurityError

hash: String := Security.passwordHash("kullanici-parolasi")
write(hash)  // $argon2id$v=19$m=65536,t=3,p=1$...$...
```

`passwordHash`, rastgele bir tuz üretir, Argon2id çalıştırır ve tek bir
kendi kendini tanımlayan PHC dizisi döndürür. Bu diziyi veritabanınıza kaydedin.
Düz metin parolayı asla saklamayın.

### 50.2 Parola doğrulama

```ahd
ok: Bool := Security.passwordVerify("kullanici-parolasi", sakliHash)
if ok {
    write("giriş kabul edildi")
} else {
    write("giriş reddedildi")
}
```

Yanlış parola `false` döndürür. Hatalı biçimlendirilmiş bir hash `SecurityError`
yükseltir — bozulmuş depolama durumunu yanlış paroladan ayırt etmek için ayrıca
yakalayın.

### 50.3 Güvenli belirteç üretme

```ahd
tok: String := Security.token()
write(len(tok))  // her zaman 43
```

`Security.token`, işletim sisteminden 32 rastgele bayt okur ve bunları 43
URL güvenli base64 karakteri olarak kodlar (256 bit entropi). CSRF gizli
alanları, parola sıfırlama bağlantıları ve API anahtarları için kullanın.

### 50.4 Sabit zamanlı karşılaştırma

```ahd
ayni: Bool := Security.secureEqual(sakliToken, alinanToken)
```

Güvenilmeyen bir kaynaktan gelen bir değeri bilinen bir sırla karşılaştırırken
`==` yerine `secureEqual` kullanın.

### 50.5 CSRF kalıbı

```ahd
tok := Security.token()
session.set("csrf", tok)

saklanan: String? := session.get("csrf")
gonderilen: String? := req.field("csrf")

if saklanan == null or gonderilen == null {
    return HTTP.text("reddedildi", 403)
}
if Security.secureEqual(saklanan, gonderilen) {
    session.set("csrf", Security.token())
    return HTTP.text("kabul edildi")
}
return HTTP.text("reddedildi", 403)
```

Tam belge için [SECURITY_TR.md](SECURITY_TR.md) ve
[v0.10 örnekleri](../examples/v0.10/README.md) sayfalarına bakın.

## 51. MySQL: ağ veritabanı sunucusu

Bölüm 35, kendi bilgisayarınızda tek bir dosyada yaşayan bir veritabanı olan
SQLite'ı kullandı. **MySQL** önemli bir yönden farklıdır — o bir
*sunucudur*. Bir dosya açmak yerine, ağ üzerinden kendi makinenizde veya
tamamen başka bir yerdeki bir sunucuda çalışıyor olabilecek bir `mysqld`
sürecine bağlanırsınız; bu sürece aynı anda birçok program konuşabilir.
Yazdığınız SQL neredeyse aynıdır; değişen şey ona nasıl ulaştığınızdır.

### 1. Bağlan

```ahd
bring Env
bring MySQL
from MySQL bring MySQLDatabase
from MySQL bring MySQLError

host := Env.getOr("MYSQL_HOST", "127.0.0.1")
username := Env.getOr("MYSQL_USERNAME", "app")
password := Env.getOr("MYSQL_PASSWORD", "")

attempt {
    db: Local MySQLDatabase := MySQL.connect(host, username, password, 3306, "okul")
    write("bağlandı")
    db.close()
} except MySQLError as error {
    write("bağlanılamadı: " + error.message)
}
```

Gerçek bir parolayı asla doğrudan kaynak kodunuza yazmayın — bölüm 41'deki
`SMTP` kimlik bilgileri gibi, `Env` ile okuyun. `MySQL.connect` yalnızca bu
bilgileri hatırlamakla kalmaz: size bir `MySQLDatabase` vermeden önce
gerçekten sunucuyu arar ve yanıt verdiğini kontrol eder, böylece yanlış bir
ana bilgisayar veya yanlış bir parola tam orada `MySQLError` yükseltir, ilk
sorgunuzda değil.

### 2. Parametreli ekleme

```ahd
db.execute(
    "INSERT INTO students (name, grade) VALUES (?, ?)"
    [MySQL.fromString("Ada"), MySQL.fromString("95.5")]
)
```

SQLite ile aynı kural: her `?`, gerçek bir bağlı parametredir, bu yüzden
içinde başıboş bir tırnak veya noktalı virgül bulunan bir isim, asla SQL
olarak çalıştırılmadan sıradan metin olarak saklanır. `MySQL.fromString`/
`fromInt`/`fromReal`/`nullValue`, bağladığınız değerleri oluşturur —
`SQLite.fromString` ve benzerlerinin kullandığı aynı şekiller.

### 3. Sorgula

```ahd
rows := db.query("SELECT id, name, grade FROM students ORDER BY id")
for row in rows {
    write(row["name"].string() + ": " + row["grade"].string())
}
```

Satırlar, tanıdık `Pair` şeklinde geri gelir. SQLite'tan farklı bir şey var:
yukarıdaki `grade` gibi bir `DECIMAL` sütunu bir `String` olarak kalır
(`"95.5"`), asla bir `Real` olmaz. İkili kayan nokta her ondalık kesri tam
olarak temsil edemez, bu yüzden sessizce dönüştürmek bir notu veya fiyatı
sessizce bozabilir. Üzerinde gerçekten aritmetiğe ihtiyacınız varsa
`real(...)` ile açıkça dönüştürün.

### 4. İşlem (Transaction)

```ahd
tx := db.begin()
attempt {
    tx.execute("UPDATE accounts SET balance = balance - ? WHERE id = ?", [...])
    tx.execute("UPDATE accounts SET balance = balance + ? WHERE id = ?", [...])
    tx.commit()
} except MySQLError as error {
    tx.rollback()
    write(error.message)
}
```

`db.begin()`, bağımsız bir işlem açar — açıkken aynı `MySQLDatabase`
üzerindeki diğer sorguları engellemez. `commit()` ile ya iki güncelleme de
gerçekleşir, ya da `rollback()` ile hiçbiri gerçekleşmez.

### 5. Kapat

```ahd
db.close()
```

Bağlantıyı havuza geri bırakır. Sonrasında kapalı bir `MySQLDatabase`
kullanmak, sessizce hiçbir şey yapmak yerine `MySQLError` yükseltir.

**Teknik not.** `MySQL` ve `SQLite`, iki ayrı tip kümesine sahip iki ayrı
modüldür (`MySQLDatabase` ile `Database`, `MySQLValue` ile `SQLiteValue`).
Bir program ikisini de aynı anda `bring` edebilir — biri paylaşılan bir ağ
veritabanı için, biri küçük yerel bir önbellek için — aralarında hiçbir
karışıklık olmaz.

[MySQL modül referansına](MYSQL_TR.md) ve
[`examples/v0.11`](../examples/v0.11/README_TR.md) klasörüne bakın.

## 52. Web: bir uygulama kurmak

36. bölüm, her AhdCode web sayfasının altındaki iki ilkeli -- `HTTP` ve
`HTML` -- gösterdi. `Web`, bunları bileştiren birinci taraf çatıdır; böylece
sıradan bir uygulama birkaç yerine tek bir içe aktarmayla yetinir.

```ahd
bring Web
```

`Web`, `HTTP` ve `HTML`'in yerini almaz. Onların türlerini değiştirmeden
yeniden dışa aktarır -- `Web` üzerinden ulaştığınız bir `Request`, 36.
bölümdeki `Request`'in aynısıdır -- ve `bring HTTP` / `bring HTML` tam olarak
eskisi gibi çalışmayı sürdürür.

### Web.UI ile sayfa kurmak

36. bölümde bir paragrafı şöyle yazdınız:

```ahd
HTML.element("p", {}, [HTML.text("Hoş geldiniz")])
```

Bu açıktır ve hâlâ doğrudur. `Web.UI` aynı şeyi daha doğrudan söyler:

```ahd
Web.UI.p("Hoş geldiniz")
```

İkisi de aynı sayfayı kurar. Tüm kütüphanenin kuralı kısadır:

- **metin** tutan öğe bir String alır: `h1`, `h2`, `p`, `span`, `li`, `td`,
  `button`, `label`, `option`
- **başka öğeler** tutan öğe bir liste alır: `section`, `article`, `nav`,
  `header`, `footer`, `main`, `ul`, `ol`, `table`, `form`

```ahd
Web.UI.section(
    [
        Web.UI.h1("Ahd Akademi")
        Web.UI.p("Hoş geldiniz")
        Web.UI.a("/hakkinda", "Akademi hakkında")
        Web.UI.img("/assets/logo.svg", "Ahd Akademi")
    ]
)
```

Öznitelikler en sonda gelir ve isteğe bağlıdır; kısa çağrılar kısa kalır:

```ahd
Web.UI.p("Kayıt açıldı", {"class": "notice"})
Web.UI.section([...], {"class": "hero", "id": "top"})
```

`img` her zaman `alt` metnini ister. Bu bilinçlidir: ekran okuyucunun ona
ihtiyacı vardır, bu yüzden imza unutmanıza izin vermez. Yalnızca dekoratif
bir görsel bilinçli olarak `""` geçer.

Bir metin öğesinin içinde öğeler gerektiğinde -- kalın bir sözcük içeren
paragraf, bağlantı içeren liste öğesi -- `Nodes` eşini kullanın:

```ahd
Web.UI.pNodes([Web.UI.text("Merhaba "), Web.UI.strong("Ali")])
Web.UI.liNodes([Web.UI.a("/hakkinda", "Hakkında")])
```

`Web.UI`'nin adlandırmadığı her etiket için `HTML.element` hâlâ oradadır.

### Metin her zaman kaçışlanır

Bu, 36. bölümün anlattığı korumanın aynısıdır ve `Web.UI`'nin her yerinde
geçerlidir:

```ahd
Web.UI.p("<script>alert(1)</script>")
```

bu karakterleri sayfada metin olarak gösterir. Asla çalışan bir betiğe
dönüşmez. Bundan çıkış yolu yoktur -- `raw` yok, `unsafeHTML` yok -- ve bu
bilinçlidir.

### Sayfalar, Yerleşimler, Bileşenler

Bir uygulamanın nasıl düzenlendiğini üç sözcük anlatır:

```
Page       bir sayfanın içeriğini üretir
Layout     o içeriği ortak kabukla sarar
Component  yeniden kullanılabilir bir parça üretir
```

Bunların hiçbiri özel sözdizimi değildir. **Her biri `HTMLNode` döndüren
sıradan bir Function'dır.** Kalıtılacak bir şey yoktur, kaydedilecek bir şey
yoktur.

Bir Bileşen:

```ahd
notice: Function := (title: String, message: String) -> HTMLNode {
    return Web.UI.section([Web.UI.h2(title), Web.UI.p(message)], {"class": "notice"})
}
```

Bir Yerleşim:

```ahd
mainLayout: Function := (title: String, content: List<HTMLNode>) -> Response {
    body: Local List<HTMLNode> := [Web.UI.main(content)]
    return Web.page(title, body, [Web.UI.stylesheet("/assets/app.css")])
}
```

Bir Sayfa:

```ahd
homePage: Function := (request: Request) -> Response {
    return mainLayout("Ana Sayfa", [Web.UI.h1("Ahd Akademi")])
}
```

### Yapılandırma

Bir uygulama ayarlarını ortamdan, tek bir yerde okur:

```
APP_NAME       Ahd Akademi
APP_ENV        development, test veya production
APP_HOST       ahdakademi.com        (bir konak -- https:// yok, /yol yok, :port yok)
APP_PROTOCOL   http veya https
SERVER_HOST    127.0.0.1
SERVER_PORT    8080
```

```ahd
bring Web
from Web bring AppConfig

application: AppConfig := Web.configure()
```

Hiçbir şey tahmin edilmez. `APP_ENV` eksikse veya yanlış yazılmışsa program
başlangıçta durur ve hangi anahtarın hatalı olduğunu söyler; amaçlamadığınız
bir şey olarak çalışmaz.

Yerel çalışırken bunları `.env` adlı bir dosyaya koyun ve **onu asla
işlemeyin (commit etmeyin)**. Bunun yerine adları ve yer tutucu değerleri
içeren `.env.example`'ı işleyin. Gerçek parolalar ve anahtarlar sürüm
denetiminin dışında kalır -- ve her başlayışında onları yeniden okuyan
üretilmiş programın da dışında.

Birbirine benzeyen ama aynı olmayan iki çift:

```
APP_PROTOCOL + APP_HOST    bir insanın yazdığı:      https://ahdakademi.com
SERVER_HOST  + SERVER_PORT programınızın bağlandığı: 127.0.0.1:8080
```

Gerçek bir sunucuda bunlar farklıdır, çünkü önde duran bir şey (Cloudflare,
nginx) genel adresi karşılar ve programınıza iletir.

### Hepsini bir araya getirmek

```ahd
require("Config/App.ahd")
require("Pages/Home.ahd")
bring Web
from Web bring App

academy: App := Web.app(application)

academy.assets("/assets", "public")
academy.get("/", homePage)
academy.start()
```

Çalışırken şöyle çalıştırın:

```bash
ahdcode dev app.ahd
```

Herhangi bir `.ahd` dosyasını düzenlemek yeniden derler ve yeniden başlatır.
`public/app.css`'i düzenlemek bunu yapmaz -- tarayıcı bir sonraki yenilemede
yeni dosyayı alır.

Eksiksiz bir örnek `examples/v0.15/ahd_academi` içindedir; tam referans
[docs/WEB_TR.md](WEB_TR.md).

## 53. Tam bir form iş akışı

[Form örneği](../examples/v0.16/forms_validation/README_TR.md),
`ahdcode run examples/v0.16/forms_validation/app.ahd` ile veritabanı olmadan
çalışır. `http://127.0.0.1:8160/register` adresini açın. GET bir istek bağlamı
oluşturup gizli CSRF alanını gösterir. POST açıkça CSRF doğrular ve doğrulama
hatalarını toplar. Geçersiz girdide yalnızca seçilmiş ad/e-posta değerleri Web.UI
kaçırmasıyla yeniden gösterilir; parola ve onayı boş kalır. Geçerli girdide flash
mesajı yazılıp `/profile` adresine yönlendirilir; bu işleyici mesajı tüketip silmeyi
kaydeder, böylece yenilemede mesaj görünmez.

İstek başına bir `Web.context(request, store)` kullanın ve her yanıt yolunda
`context.respond(response)` döndürün. İkinci sonlandırma `WebContextError`
üretir. `context.session` olağan Session değeridir; uygulamanın giriş ve koruma
kodları açık kalır. `Web.form(request)` oturumsuz da kullanılabilir.
`Form.integer` eksik null ile geçersiz `FormValueError` durumlarını;
`Form.optional` eksik ile boş girdiyi ayırır. `Web.errors()` zorunlu alan,
uzunluk, eşleşme, e-posta biçimi, izin verilen değerler, kesin hex renk ve özel
alan hatalarını belirli sırayla destekler.

`form.old(["name", "email"])` seçimini açık yapın; parola, sıfırlama doğrulayıcısı
ve diğer sırları seçmeyin. Otomatik eski girdi saklama, flash gösterimi,
middleware veya auth çerçevesi yoktur. [Web rehberi](WEB_TR.md) tam akışı ve kesin
API'yi öğretir; v0.15 API'leri kaynak uyumluluğunu korur.

## 54. Gruplar, bekçiler ve starter'lar

### Starter'lar

`ahdcode init web` hangi starter'ın yazılacağını sorar; ya da doğrudan
adlandırabilirsiniz (`ahdcode init web empty`, `basic`, `admin`):

- **Empty** — bir karşılama sayfası. Veritabanı yok, giriş yok.
- **Basic** — aynı kabuk artı `.env` içinde ortak uygulama ve posta
  yapılandırması.
- **Admin** — Home, Login ve bir Dashboard; SQLite ya da MySQL üzerinde tek bir
  yönetici hesabıyla.

Yazdığı her şey okuyup değiştirebileceğiniz sıradan AhdCode'dur: sayfalar,
yerleşimler, bileşenler ve bir `Config/` klasörü; hepsi 19. bölümdeki gibi
`require(...)` ile birbirine bağlanmıştır. Şablonlar ve Bootstrap CLI'ın
içindedir; bu yüzden `init` ağ gerektirmez ve hiçbir paket yöneticisi kurmaz.
Var olan bir dosyanın ya da veritabanının üzerine asla yazmaz.

Admin starter'ı bir veritabanı adı ve bir yönetici sorar, veritabanını
oluşturur ve parolayı `Security.passwordHash` ile hashler (50. bölüm). SQLite
seçtiyseniz 55. bölüm oradan devam eder.

### Rotalar, gruplar ve bekçiler

Bağlam duyarlı rotalar, gruplar ve sıralı bekçiler ayrı,
açık katman olarak kalır:
[`examples/v0.17/routes_guards`](../examples/v0.17/routes_guards).
`Function(Request) -> Response` ile `App.get` çalışmayı sürdürür. Bekçi
devam için `null`, durmak için sonlandırılmış `Response` döner. Web bir
yöneticinin kim olduğunu bilmez. Genel ara katman zinciri yoktur.

`Page`, `Layout` ve `Component` bir uygulamayı düzenleme biçimleridir, dil
yapısı değil; işleyici adları da sıradan tanımlayıcılardır: örnek `Page` soneki
olmadan `register`, `registerSubmit` ve `profile` işleyicilerine yönlendirir,
`registerPage` kullanan uygulamalar ise değişmeden çalışmayı sürdürür. Bkz.
[10.1 Adlandırma](WEB_TR.md#101-adlandırma).

## 55. Görebildiğiniz veritabanları: `ahdcode databases`

35. bölüm koddan bir SQLite veritabanı açtı. Er ya da geç bir veritabanına
*bakmak* da istersiniz: içinde hangi tablolar var, gerçekten ne saklanıyor, o
INSERT çalıştı mı.

```bash
ahdcode databases
```

Bu, kendi makinenizde çalışan küçük bir veritabanı çalışma alanı olan
**AhdDataStudio**'yu başlatır. Kendisi de bir AhdCode programıdır —
kaynağını `tools/AhdDataStudio/` altında okuyabilirsiniz — ve dilin parçası
değildir. Şu adreste açılır:

```text
http://ahddatabasestudio.test/
```

ve bu ad makinenizde çözülsün ya da çözülmesin, her zaman şurada:

```text
http://127.0.0.1:8081/AhdDataStudio
```

Yalnızca `127.0.0.1` dinler. Yerel bir geliştirme aracıdır ve asla açık bir
adrese konmamalıdır.

### Hangi veritabanları görünür

Studio bilgisayarınızda veritabanı dosyası **aramaz**. Bu bilinçlidir:
diskinizde veritabanına benzeyen şeyleri dolaşan bir araç, hakkında akıl
yürütemeyeceğiniz bir araçtır.

Bunun yerine bir SQLite veritabanı *kaydedilmişse* görünür:

```bash
ahdcode databases add ./notes.db      # bu dosyayı kaydet
ahdcode databases list                # neler kayıtlı
ahdcode databases remove ./notes.db   # unut
```

`remove` hakkında iki şeyi açıkça söylemek gerekir. *Kaydı* siler, dosyayı
değil — veritabanınıza dokunulmaz. Ve yalnızca adını verdiğiniz kaydı siler.

Kayıtlı bir dosya şu anda yoksa — takılı olmayan bir harici disk, taşıdığınız
bir proje klasörü — `list` onu sessizce unutmak yerine `unavailable` olarak
gösterir. Kayıt defterinizden bir şeyi yalnızca siz silersiniz.

### Genellikle bunu elle yapmanız gerekmez

Admin starter'ıyla bir proje oluşturup SQLite seçtiğinizde:

```bash
ahdcode init web admin
ahdcode databases
```

`init`'in az önce oluşturduğu veritabanı zaten oradadır. Kendini kaydeder;
yapılandırılacak bir şey ve yazılacak bir şey yoktur.

MySQL farklı çalışır ve bu bilinçlidir: bir MySQL bağlantısı kullanıcı adı ve
parola ister; bunlar da herhangi bir kayıt defterinde değil, AhdDataStudio'nun
kendi `.env` dosyasında yaşar. AhdCode veritabanı parolalarınızı bir listede
tutmaz.

Bkz. [AhdDataStudio rehberi](../tools/AhdDataStudio/README_TR.md) ve
[CLI](CLI_TR.md#ahdcode-databases).

## 56. Uygulamanızın yerel adresi

36. bölümde bir web sayfası çalıştırdığınızda `http://127.0.0.1:8080` gibi bir
şey açtınız. Bu çalışır ama bir web sitesine benzemez; ayrıca iki projeniz
açıksa hangi portun hangisi olduğunu hatırlamanız gerekir.

Uygulamanız gerçek adını zaten biliyor. `.env` dosyasında:

```text
APP_NAME=Ahd Akademi
APP_ENV=development
APP_HOST=ahdakademi.com
APP_PROTOCOL=http
SERVER_HOST=127.0.0.1
SERVER_PORT=8080
```

`APP_HOST`, insanların bir gün yazacağı adrestir. Bu yüzden şunu
çalıştırdığınızda:

```bash
ahdcode dev app.ahd
```

AhdCode ondan *yerel* bir ad türetir ve uygulamanızı orada sunar:

```text
AhdCode Web
  Ahd Akademi (development)

  Open:
  http://127.0.0.1:8080

  Local identity:
  http://ahdakademi.test/
  Bind: 127.0.0.1:8080
```

`ahdakademi.com`, `ahdakademi.test` olur. Sonek eklenmez, değiştirilir; böylece
yerel ad aynı proje gibi okunur. `.test`, tam olarak bunun için sonsuza dek
ayrılmış bir adres sonudur — gerçek bir web sitesine asla ait olamaz — bu
yüzden yerel çalışma kazara gerçek sunucuya ulaşamaz.

Bir değil iki şeyin yazıldığına dikkat edin:

- **Open**, programınızın gerçekten bağlandığı sokettir: `SERVER_HOST` ve
  `SERVER_PORT`. Bu her zaman çalışır.
- **Local identity**, bu makinenin ona yönlendirdiği addır.

Bunlar farklı olgulardır; bu yüzden ayrı gösterilirler. Bağlanma adresi soketin
nerede olduğunu, yerel ad ise bu bilgisayarın ona neyi işaret ettiğini söyler.

### İki proje, aynı ad

Yine `ahdakademi.com` için yapılandırılmış ikinci bir projeyi başlatın;
`ahdakademi1.test`, sonra `ahdakademi2.test` alır. İlkini durdurun ve adı
sıradaki için yeniden serbest kalsın.

### Adın çözülmesini sağlamak

`ahdcode dev` bir uçbirimde ilk kez bir `.test` adına ihtiyaç duyduğunda
sorar:

```text
Enable local .test names? [Y/n]
```

Bu soru herhangi bir yönetici parolasından önce gelir. Evet derseniz sonraki
proje adları otomatik eklenir. Her proje için ayrı bir komut çalıştırmanız
gerekmez.

`ahdcode local hosts apply` ve `remove` AhdCode'un bloğunu elle onarmak veya
geri almak için durur. Bloğun dışındaki her şey olduğu gibi kalır.

Hayır derseniz veya bir uçbirimde değilseniz hiçbir şey bozulmaz: bağ adresi
(`http://127.0.0.1:<port>`) çalışmaya devam eder.

### Neyin çalıştığını görmek

```bash
ahdcode local status
```

bu makinenin şu anda sunduğu her yerel adresi ve her birinin neyi işaret
ettiğini listeler. "`ahdakademi1.test` hangi projeydi?" diye merak ederseniz,
yanıtı budur.

### Bunun ne olmadığı

İki dürüst sınır:

- **HTTPS değil, HTTP'dir.** Yerelde `https://`, bilgisayarınızın güven
  deposuna bir sertifika otoritesi kurmak demektir; bu, bir adı yönlendirmekten
  çok daha büyük bir karardır. AhdCode bunu yapmaz ve düz HTTP sunup ona
  güvenli demektense `APP_PROTOCOL=https` ile başlamayı reddeder.
- **Yalnızca sizin makineniz.** Yönlendirici geri döngüyü dinler ve yalnızca
  geri döngüye iletir. Bilgisayarınızın dışından hiçbir şey ona erişemez ve bu,
  bir şey yayınlamanın yolu değildir.

Dağıtım için `APP_HOST` ve `APP_PROTOCOL` *herkese açık* adresi tanımlar ve
uygulamanızın önündeki gerçek bir sunucu HTTPS'i sonlandırır. Bu ayrımı
[Web rehberi](WEB_TR.md#15-production) anlatır.

## 57. UUID: benzersiz kimlikler

v1.4.0 ile gelen dört yeniliğin ilki `UUID` modülüdür. Sonraki bölümler
sırasıyla gizli değerleri (58), PostgreSQL'i (59) ve WebSocket'i (60) anlatır;
61. bölüm dördünü tek bir uygulamada birleştirir.

### 57.1 Neden bir kimliğe ihtiyacımız var?

35. bölümde her nota `INTEGER PRIMARY KEY AUTOINCREMENT` ile 1, 2, 3 diye
numara verdik. Tek bir dosya ve tek bir program varken bu harikadır. Ama şu
durumları düşünün:

- İki ayrı sunucu aynı anda yeni kayıt ekliyor. İkisi de sıradaki numaranın 5
  olduğunu düşünürse iki farklı kayıt aynı numarayı alır.
- Adres çubuğunda `/fatura/41` gören biri `/fatura/42`'yi denemeye başlar.
- Kaydı veritabanına yazmadan *önce* ona bir kimlik vermek istiyorsunuz;
  örneğin tarayıcıya göndereceğiniz bir mesajda.

**UUID** (Universally Unique Identifier), merkezî bir sayaç olmadan her yerde
üretilebilen 128 bitlik bir kimliktir. Metin olarak her zaman 36 karakterdir:

```text
01a0a0c9-8fa6-733f-9f09-81191abcce97
^^^^^^^^ ^^^^ ^
8        4    sürüm rakamı (burada 7)
```

Aynı UUID'nin iki kez üretilmesi pratikte imkânsızdır; bu yüzden iki sunucu
birbirine sormadan kimlik üretebilir.

### 57.2 UUID üretmek: v4 ve v7

```ahd
bring UUID
from UUID bring UUIDValue

rastgele: UUIDValue := UUID.v4()
sirali: UUIDValue := UUID.v7()

write(rastgele.string())
write(sirali.string())
write("sürümler: " + str(rastgele.version()) + " ve " + str(sirali.version()))
```

Çıktı her çalıştırmada farklıdır, ama biçimi hep aynıdır:

```text
3f2b8c1e-9a4d-4f6b-8e21-5c7d9a0b1e44
01a0a0c9-8fa6-733f-9f09-81191abcce97
sürümler: 4 ve 7
```

- `UUID.v4()` tamamen rastgeledir.
- `UUID.v7()` o anki zamanla başlar, sonuna rastgele bitler ekler. Bu yüzden
  **sonra üretilen v7 her zaman sonra sıralanır.** Veritabanında birincil
  anahtar olarak kullanacaksanız çoğu zaman v7 daha iyi seçimdir.

Bir `UUIDValue` bir `String` değildir. Metne ihtiyacınız olduğunda
`.string()` çağırırsınız.

### 57.3 v7 değerleri üretim sırasıyla dizilir

```ahd
bring UUID
from UUID bring UUIDValue

kimlikler: List<UUIDValue> := []
for i in between(0, 5) {
    kimlikler.add(UUID.v7())
}

sirali: Bool := true
for i in between(1, len(kimlikler)) {
    if kimlikler[i - 1].compare(kimlikler[i]) >= 0 {
        sirali = false
    }
}

for kimlik in kimlikler {
    write(kimlik.string())
}
write("üretim sırasıyla dizili: " + str(sirali))
```

`a.compare(b)`, `a` önce geliyorsa `-1`, eşitse `0`, sonra geliyorsa `1`
döndürür. Tek bir program içinde her `UUID.v7()` bir öncekinden büyüktür;
bilgisayarın saati geriye alınsa bile. Aynı sıra metinde de geçerlidir: v7
metinlerini alfabetik sıralamak onları üretim zamanına göre sıralar.

Aynı döngüyü `UUID.v4()` ile deneyin: sonuç çoğu zaman `false` olur, çünkü v4
bir sıra vaat etmez.

### 57.4 Metinden UUID'ye: `parse` ve `isValid`

Kullanıcıdan, bir adresten ya da bir dosyadan gelen kimlik bir `String`'dir.
Onu `UUID.parse` ile `UUIDValue`'ya çevirirsiniz. Kurallar katıdır: tam 36
karakter, `-` ile ayrılmış 8-4-4-4-12 gruplar. Büyük harf kabul edilir, çıktı
her zaman küçük harftir.

```ahd
bring UUID

girdiler: List<String> := [
    "017F22E2-79B0-7CC3-98C4-DC0C0C07398F"
    "017f22e2-79b0-7cc3-98c4-dc0c0c07398f"
    r"{017f22e2-79b0-7cc3-98c4-dc0c0c07398f}"
    "urn:uuid:017f22e2-79b0-7cc3-98c4-dc0c0c07398f"
    "017f22e279b07cc398c4dc0c0c07398f"
    " 017f22e2-79b0-7cc3-98c4-dc0c0c07398f"
    "merhaba"
]

for metin in girdiler {
    if UUID.isValid(metin) {
        write("geçerli  -> " + UUID.parse(metin).string())
    }
    else {
        write("geçersiz -> " + metin)
    }
}
```

Beklenen çıktı:

```text
geçerli  -> 017f22e2-79b0-7cc3-98c4-dc0c0c07398f
geçerli  -> 017f22e2-79b0-7cc3-98c4-dc0c0c07398f
geçersiz -> {017f22e2-79b0-7cc3-98c4-dc0c0c07398f}
geçersiz -> urn:uuid:017f22e2-79b0-7cc3-98c4-dc0c0c07398f
geçersiz -> 017f22e279b07cc398c4dc0c0c07398f
geçersiz ->  017f22e2-79b0-7cc3-98c4-dc0c0c07398f
geçersiz -> merhaba
```

Süslü parantezli satırı neden `r"..."` ile yazdık? Sıradan bir String içindeki
`{` araya değer eklemeyi (interpolasyon) başlatır (6. bölüm). Ham String'de
`{` sıradan bir karakterdir.

`isValid` hiçbir zaman hata vermez. `parse` ise geçersiz metinde `UUIDError`
fırlatır; bunu 18. bölümdeki gibi yakalayabilirsiniz:

```ahd
bring UUID
from UUID bring (UUIDValue, UUIDError)

kimlikOku: Function := (metin: String) -> UUIDValue? {
    attempt {
        return UUID.parse(metin)
    }
    except UUIDError as error {
        write("reddedildi: " + error.message)
        return null
    }
}

bulunan: UUIDValue? := kimlikOku("017f22e2-79b0-7cc3-98c4-dc0c0c07398f")
if bulunan != null {
    write("bulundu: " + bulunan.string())
}

hatali: UUIDValue? := kimlikOku("42")
write("hatalı girdi null döndü: " + str(hatali == null))
```

Hata mesajı reddedilen metni hiçbir zaman tekrar etmez; kullanıcıdan gelen
bir değeri günlüğe taşımaz.

### 57.5 İki UUID'yi karşılaştırmak: `equals`, `compare` ve `==` tuzağı

```ahd
bring UUID
from UUID bring UUIDValue

a: UUIDValue := UUID.parse("017f22e2-79b0-7cc3-98c4-dc0c0c07398f")
b: UUIDValue := UUID.parse("017F22E2-79B0-7CC3-98C4-DC0C0C07398F")
c: UUIDValue := a

write(a.equals(b))
write(a == b)
write(a == c)
write(a.compare(b))
write(str(a))
write(a.string())
```

Beklenen çıktı:

```text
true
false
true
0
<UUIDValue>
017f22e2-79b0-7cc3-98c4-dc0c0c07398f
```

- `equals` iki UUID'nin **aynı 128 biti** taşıyıp taşımadığını sorar. İki
  kimliği karşılaştırırken bunu kullanın.
- `==`, 13. bölümdeki referans davranışını izler: iki değişken **aynı nesneyi**
  mi gösteriyor? `a` ve `b` ayrı ayrı ayrıştırıldığı için `false`, `c := a`
  aynı nesneyi paylaştığı için `true`.
- `str(a)` metni değil `<UUIDValue>` yazar. Metin için `a.string()`.

### 57.6 Sıfır UUID

```ahd
bring UUID
from UUID bring UUIDValue

sahip: UUIDValue := UUID.zero()
write(sahip.string())
write("henüz atanmadı: " + str(sahip.isZero()))

sahip = UUID.v7()
write("henüz atanmadı: " + str(sahip.isZero()))
```

`UUID.zero()`, `00000000-0000-0000-0000-000000000000` değeridir. Bazı sistemler
"boş kimlik" için bunu kullanır. Kendi kodunuzda "henüz yok" demek için çoğu
zaman `UUIDValue?` ve `null` (16. bölüm) daha açıktır; sıfır UUID'yi ancak
karşınızdaki sistem onu beklediğinde kullanın.

### 57.7 Hangi kimliği ne zaman kullanmalı?

| | `Identity.id()` | `Security.token()` | `UUID.v4()` | `UUID.v7()` |
| --- | --- | --- | --- | --- |
| Amaç | AhdCode içi genel kimlik | **gizli** belirteç | başka sistemlerle uyumlu kimlik | zamana göre sıralı kimlik |
| Metin | 22 karakter | 43 karakter | 36 karakter | 36 karakter |
| Gizli mi? | hayır | **evet** | hayır | hayır; ne zaman üretildiğini gösterir |

En önemli kural: **UUID bir sır değildir.** Oturum anahtarı, parola sıfırlama
bağlantısı, API anahtarı gibi tahmin edilmemesi gereken her şey için
`Security.token()` kullanın (50. bölüm). Bir v7 UUID'ye bakan herkes aşağı
yukarı ne zaman üretildiğini okuyabilir.

### 57.8 UUID'leri bir veritabanında saklamak

UUID'nin 36 karakterlik metnini saklarsınız. SQLite'ta (35. bölüm) `TEXT`
sütunu yeterlidir:

```ahd
bring SQLite
bring UUID
from SQLite bring Database
from UUID bring UUIDValue

db: Database := SQLite.open(":memory:")
db.execute("CREATE TABLE dersler (id TEXT PRIMARY KEY, baslik TEXT NOT NULL)")

for baslik in ["Değişkenler", "Döngüler", "Fonksiyonlar"] {
    db.execute(
        "INSERT INTO dersler (id, baslik) VALUES (?, ?)"
        [SQLite.fromString(UUID.v7().string()), SQLite.fromString(baslik)]
    )
}

satirlar := db.query("SELECT id, baslik FROM dersler ORDER BY id")
for satir in satirlar {
    kimlik: Local UUIDValue := UUID.parse(satir["id"].string())
    write(satir["baslik"].string() + " -> sürüm " + str(kimlik.version()))
}
db.close()
```

Beklenen çıktı:

```text
Değişkenler -> sürüm 7
Döngüler -> sürüm 7
Fonksiyonlar -> sürüm 7
```

`ORDER BY id` satırları ekleme sırasıyla getirdi, çünkü v7 metinleri üretim
zamanına göre sıralanır. MySQL'de `CHAR(36)` kullanın; PostgreSQL'in ise kendi
`uuid` türü vardır (59. bölüm).

**Siz deneyin:** 57.8'deki programda `UUID.v7()` yerine `UUID.v4()` yazın ve
`ORDER BY id` sırasının artık ekleme sırası olmadığını görün.

Tam başvuru: [UUID](UUID_TR.md) ·
[`examples/v0.1/69_uuid.ahd`](../examples/v0.1/69_uuid.ahd).

## 58. Gizli değerleri dosyadan okumak

### 58.1 Parolayı nereye koymalı?

Bir veritabanı parolasını kaynak koda yazmak en kötü seçenektir: kod git
geçmişine girer ve oradan kolay kolay silinmez. 33. bölümde `.env` dosyasını
ve ortam değişkenlerini gördünüz; yerelde çalışırken doğru yer orasıdır.

Sunucularda ise çoğu zaman bir adım daha ileri gidilir: Docker, Kubernetes ya
da systemd gibi platformlar parolayı programın okuyabildiği **bir dosya**
olarak yerleştirir ve dosyanın yolunu `_FILE` ile biten bir değişkende verir:

```text
DB_PASSWORD_FILE=/run/secrets/db_password
```

`Env.secret("DB_PASSWORD")` iki yolu da tek çağrıda karşılar.

### 58.2 Kurallar

| `DB_PASSWORD` | `DB_PASSWORD_FILE` | `Env.secret("DB_PASSWORD")` |
| --- | --- | --- |
| yok | yok | `null` |
| var (`""` bile olsa) | yok | `DB_PASSWORD` değeri |
| yok | bir dosya yolu | dosyanın içeriği |
| yok | `""` | `EnvError` |
| var | var | `EnvError`: yalnızca birini ayarlayın |

- Dosyanın sonundaki **tek** satır sonu (`\n` ya da `\r\n`) kaldırılır; başka
  hiçbir şey kırpılmaz.
- Dosya en fazla 1 MiB, geçerli UTF-8 ve NUL baytı içermeyen bir dosya
  olmalıdır.
- Dosya yoksa ya da okunamıyorsa `EnvError` fırlatılır; `Env.secret` asla
  sessizce başka bir değere geri dönmez.
- İkisi birden ayarlıysa hata verir. Böylece unutulmuş eski bir değişken
  hiçbir zaman gizlice kazanmaz.

### 58.3 Dosyadan okumak

Bu program kendi gizli dosyasını oluşturur, sonra onu okur:

```ahd
bring Env
bring File
bring Path

klasor := "gizli-deneme"
if File.exists(klasor) == false {
    File.createDir(klasor)
}
dosya := Path.join([klasor, "db_password.txt"])
File.writeText(dosya, "cok-gizli-parola\n")

Env.unset("DB_PASSWORD")
Env.unset("DB_PASSWORD_FILE")
bos: String? := Env.secret("DB_PASSWORD")
write("hiçbiri ayarlı değil, sonuç null: " + str(bos == null))

Env.set("DB_PASSWORD_FILE", dosya)
parola: String? := Env.secret("DB_PASSWORD")
if parola != null {
    write("dosyadan okundu, uzunluk: " + str(len(parola)))
    write(
        "sondaki satır sonu kaldırıldı: " + str(parola.endsWith("\n") == false)
    )
}
```

Beklenen çıktı:

```text
hiçbiri ayarlı değil, sonuç null: true
dosyadan okundu, uzunluk: 16
sondaki satır sonu kaldırıldı: true
```

Program parolanın kendisini değil, yalnızca uzunluğunu yazıyor. Bu bilinçli
bir alışkanlıktır: gizli bir değer ekrana, günlüğe ya da bir HTTP yanıtına
yazılmamalıdır.

### 58.4 Hataları yakalamak

```ahd
bring Env
from Env bring EnvError

dene: Function := (durum: String) -> Nothing {
    attempt {
        deger: Local String? := Env.secret("API_KEY")
        if deger == null {
            write(durum + ": ayarlı değil")
        }
        else {
            write(durum + ": okundu")
        }
    }
    except EnvError as error {
        write(durum + ": " + error.message)
    }
}

Env.unset("API_KEY")
Env.unset("API_KEY_FILE")
dene("hiçbiri")

Env.set("API_KEY", "abc123")
dene("yalnızca API_KEY")

Env.set("API_KEY_FILE", "api_key.txt")
dene("ikisi birden")

Env.unset("API_KEY")
dene("olmayan dosya")

Env.set("API_KEY_FILE", "")
dene("boş yol")
```

Beklenen çıktı (çalışma klasöründe `api_key.txt` yokken):

```text
hiçbiri: ayarlı değil
yalnızca API_KEY: okundu
ikisi birden: API_KEY and API_KEY_FILE are both set; set only one
olmayan dosya: the secret file named by API_KEY_FILE could not be read
boş yol: API_KEY_FILE is set but empty
```

Mesajlar değişkenin **adını** söyler; değeri, dosyanın içeriğini ya da yolunu
asla içermez.

### 58.5 Yerelde `.env`, sunucuda dosya

Yerelde commit etmediğiniz bir `.env` dosyası yeterlidir:

```text
DB_PASSWORD=yerel-gelistirme-parolasi
```

`Web.start()` ve `Web.configure()` `.env` dosyasını kendileri yükler. Sıradan
bir betikte `Env.secret` çağırmadan önce `Env.load(".env")` çağırın.

Sunucuda aynı program hiç değişmeden dosyadan okur. Docker Compose ile:

```yaml
services:
  uygulama:
    image: benim-uygulamam
    environment:
      DB_PASSWORD_FILE: /run/secrets/db_password
    secrets:
      - db_password

secrets:
  db_password:
    file: ./db_password.txt
```

systemd ile (`%d`, systemd'nin kimlik bilgileri klasörüdür):

```ini
[Service]
LoadCredential=db_password:/etc/uygulamam/db_password
Environment=DB_PASSWORD_FILE=%d/db_password
```

### 58.6 Unutmayın

- `Env.secret` değeri *okur*, sonra onu gizlemez. Elinizdeki sıradan bir
  `String`'dir. `write(parola)` yazarsanız ekrana çıkar.
- `DB_PASSWORD` ile `DB_PASSWORD_FILE`'ı birlikte ayarlamayın.
- Gizli dosyayı da `.env` gibi git'e eklemeyin.

**Siz deneyin:** 58.3'teki programda dosyaya `"parola\n\n"` yazın. Uzunluk kaç
çıkıyor? (İpucu: yalnızca *bir* satır sonu kaldırılır.)

Tam başvuru: [Env](ENV_TR.md#secret) ·
[`examples/v0.1/70_env_secret.ahd`](../examples/v0.1/70_env_secret.ahd).

## 59. PostgreSQL: güçlü bir ağ veritabanı

51. bölümde MySQL ile ağ üzerinden bir veritabanı sunucusuna bağlandınız.
**PostgreSQL** de bir sunucudur ve dünyada en çok kullanılan veritabanlarından
biridir. AhdCode v1.4.0 onun için ayrı bir `PostgreSQL` modülü getirir.

PostgreSQL'i öğrenmeye değer kılan birkaç özellik:

- `uuid`, `boolean`, `numeric`, `timestamptz`, `jsonb` gibi zengin türler.
- Katı işlemler (transaction): bir adım başarısız olursa işlemin geri kalanı
  da reddedilir; yarım kayıt oluşmaz.
- Açık ve tutarlı hata kodları (SQLSTATE).

Bu bölümdeki programlar birbirini izler: önce bir tablo oluşturacak, sonra
öğrenci ekleyip okuyacak, güncelleyecek ve işlemlerle çalışacaksınız.

### 59.1 Yerelde bir PostgreSQL sunucusu

Denemek için kendi bilgisayarınızda bir sunucu yeterlidir. Docker varsa tek
komut:

```bash
docker run --name ahd-postgres -e POSTGRES_USER=app -e POSTGRES_PASSWORD=yerel-parola -e POSTGRES_DB=okul -p 5432:5432 -d postgres:18
```

macOS'ta Homebrew ile:

```bash
brew install postgresql@18
brew services start postgresql@18
"$(brew --prefix postgresql@18)/bin/createuser" --pwprompt app
"$(brew --prefix postgresql@18)/bin/createdb" --owner app okul
```

Ubuntu/Debian'da:

```bash
sudo apt install postgresql
sudo -u postgres createuser --pwprompt app
sudo -u postgres createdb --owner app okul
```

Windows'ta [postgresql.org](https://www.postgresql.org/download/) adresindeki
kurucuyu kullanın ve kurulumdan sonra pgAdmin ile `app` kullanıcısını ve `okul`
veritabanını oluşturun.

Ardından çalışma klasörünüzde commit etmeyeceğiniz bir `.env` dosyası açın:

```text
DB_HOST=127.0.0.1
DB_PORT=5432
DB_USERNAME=app
DB_PASSWORD=yerel-parola
DB_DATABASE=okul
DB_SECURITY=none
```

`DB_SECURITY=none` şifresiz bağlantı demektir ve **yalnızca kendi
bilgisayarınızdaki** bir sunucu için uygundur. Ağ üzerindeki gerçek bir
sunucuda `tls` kullanın (varsayılan budur).

### 59.2 Paylaşılan bağlantı dosyası

Bu bölümdeki her program aynı şekilde bağlanır. Tekrar etmemek için bağlantıyı
bir kez `Baglanti.ahd` dosyasına yazalım ve 19. bölümdeki `require(...)` ile
kullanalım:

```ahd
// Baglanti.ahd -- bu bölümdeki programların paylaştığı bağlantı.
bring Env
bring File
bring PostgreSQL
from PostgreSQL bring PostgreSQLDatabase

okulBaglan: Function := () -> PostgreSQLDatabase {
    if File.exists(".env") {
        Env.load(".env")
    }
    parola: Local String? := Env.secret("DB_PASSWORD")
    if parola == null {
        toss Error("DB_PASSWORD ya da DB_PASSWORD_FILE ayarlayın.")
    }
    return PostgreSQL.connect(
        Env.getOr("DB_HOST", "127.0.0.1")
        Env.getOr("DB_USERNAME", "app")
        parola
        int(Env.getOr("DB_PORT", "5432"))
        Env.getOr("DB_DATABASE", "okul")
        Env.getOr("DB_SECURITY", "tls")
    )
}
```

`PostgreSQL.connect` argümanları sırasıyla: sunucu, kullanıcı adı, parola,
port, veritabanı ve güvenlik kipidir. `connect` yalnızca bilgileri saklamaz:
sunucuya gerçekten bağlanır, kimliğinizi doğrular ve yanıt verdiğini denetler.
Yanlış bir parola ya da kapalı bir sunucu hemen burada `PostgreSQLError`
fırlatır, ilk sorgunuzda değil.

`PGHOST`, `PGPASSWORD` gibi PostgreSQL'e özgü ortam değişkenleri ve `~/.pgpass`
dosyası hiçbir şeyi değiştirmez. Nereye bağlanılacağına yalnızca `connect`
argümanları karar verir.

İlk deneme:

```ahd
require("Baglanti.ahd")
bring PostgreSQL
from PostgreSQL bring PostgreSQLError

attempt {
    db: Local := okulBaglan()
    sonuc: Local := db.query("SELECT current_database() AS ad")
    write("bağlandı: " + sonuc[0]["ad"].string())
    db.close()
}
except PostgreSQLError as error {
    write("bağlanılamadı: " + error.message)
}
```

Beklenen çıktı:

```text
bağlandı: okul
```

### 59.3 Tablo oluşturmak

```ahd
require("Baglanti.ahd")

db := okulBaglan()
db.execute("""
    CREATE TABLE IF NOT EXISTS ogrenciler (
        id uuid PRIMARY KEY,
        ad text NOT NULL,
        eposta text NOT NULL UNIQUE,
        aktif boolean NOT NULL DEFAULT true,
        ortalama numeric(5, 2),
        kayit_zamani timestamptz NOT NULL DEFAULT now()
    )
    """)
write("ogrenciler tablosu hazır")
db.close()
```

| Sütun | Türü | Anlamı |
| --- | --- | --- |
| `id` | `uuid` | 57. bölümdeki UUID; uygulama `UUID.v7()` ile üretir |
| `ad` | `text` | uzunluğu sınırsız metin |
| `eposta` | `text ... UNIQUE` | aynı e-posta iki kez kaydedilemez |
| `aktif` | `boolean` | gerçek bir `true`/`false` |
| `ortalama` | `numeric(5, 2)` | kuruş hatası olmayan ondalık sayı, örneğin `91.50`; boş (`NULL`) olabilir |
| `kayit_zamani` | `timestamptz` | saat dilimi bilen zaman; varsayılanı "şimdi" |

`IF NOT EXISTS` sayesinde programı ikinci kez çalıştırmak hata vermez.

### 59.4 Satır eklemek: `$1`, `$2` ve `RETURNING`

MySQL ve SQLite `?` kullanıyordu. PostgreSQL'in yer tutucuları **numaralıdır**:
`$1`, `$2`, `$3`... Değerler yine her zaman ayrı gönderilir ve asla SQL
metninin içine yapıştırılmaz.

```ahd
require("Baglanti.ahd")
bring PostgreSQL
bring UUID
from PostgreSQL bring PostgreSQLDatabase

ogrenciEkle: Function := (
    db: PostgreSQLDatabase
    ad: String
    eposta: String
    ortalama: String
) -> Nothing {
    eklenen: Local := db.query(
        "INSERT INTO ogrenciler (id, ad, eposta, ortalama) VALUES ($1::uuid, $2, $3, $4::numeric) RETURNING id, kayit_zamani"
        [
            PostgreSQL.fromString(UUID.v7().string())
            PostgreSQL.fromString(ad)
            PostgreSQL.fromString(eposta)
            PostgreSQL.fromString(ortalama)
        ]
    )
    write(ad + " eklendi, kimlik: " + eklenen[0]["id"].string())
}

db := okulBaglan()
db.execute("DELETE FROM ogrenciler")
ogrenciEkle(db, "Ayşe Yılmaz", "ayse@example.com", "91.50")
ogrenciEkle(db, "Mehmet Kaya", "mehmet@example.com", "78.25")
ogrenciEkle(db, "Zeynep Demir", "zeynep@example.com", "85.00")
db.close()
```

Beklenen çıktı (kimlikler sizde farklı olur):

```text
Ayşe Yılmaz eklendi, kimlik: 01a0a0c9-8fa6-733f-9f09-81191abcce97
Mehmet Kaya eklendi, kimlik: 01a0a0c9-8fa7-7a41-b3c2-0d5e6f718293
Zeynep Demir eklendi, kimlik: 01a0a0c9-8fa7-7b10-8c77-2e9d4a5b6c01
```

Dört şeye dikkat edin:

1. **`$1::uuid` ve `$4::numeric`**: `PostgreSQL.fromString` metin gönderir.
   `::uuid` "bu metni uuid olarak yorumla" demektir. Türün belli olmadığı
   yerlerde bu dönüşümleri SQL'de açıkça yazın.
2. **`RETURNING id, kayit_zamani`**: PostgreSQL'de `lastInsertId()` yoktur.
   Sunucunun ürettiği değerleri (burada varsayılan `kayit_zamani`) `RETURNING`
   ile isteyip `query` ile okursunuz.
3. **`DELETE FROM ogrenciler`**: programı yeniden çalıştırdığınızda `UNIQUE`
   e-posta kuralına takılmamak için başta tabloyu boşalttık.
4. **Bir çağrı, bir ifade**: `"UPDATE ...; DROP TABLE ..."` gibi noktalı
   virgülle iki ifade göndermek hata verir ve hiçbiri çalışmaz.

Değer göndermenin beş yolu vardır: `PostgreSQL.fromString`, `fromInt`,
`fromReal`, `fromBool` ve `nullValue()`.

### 59.5 Satırları okumak ve türler

```ahd
require("Baglanti.ahd")

db := okulBaglan()
satirlar := db.query(
    "SELECT ad, aktif, ortalama, kayit_zamani FROM ogrenciler ORDER BY ad"
)
for satir in satirlar {
    ad: Local String := satir["ad"].string()
    aktif: Local Bool := satir["aktif"].bool()
    ortalama: Local String := satir["ortalama"].string()
    write(ad + " | aktif: " + str(aktif) + " | ortalama: " + ortalama)
}

ilk := satirlar[0]
write("ad: " + ilk["ad"].kind())
write("aktif: " + ilk["aktif"].kind())
write("ortalama: " + ilk["ortalama"].kind())
write("kayit_zamani: " + ilk["kayit_zamani"].kind())
write("örnek zaman: " + ilk["kayit_zamani"].string())
db.close()
```

Beklenen çıktı (zaman sizde farklı olur):

```text
Ayşe Yılmaz | aktif: true | ortalama: 91.50
Mehmet Kaya | aktif: true | ortalama: 78.25
Zeynep Demir | aktif: true | ortalama: 85.00
ad: String
aktif: Bool
ortalama: String
kayit_zamani: String
örnek zaman: 2026-09-15 08:30:12.482913+00
```

Her satır, 35. bölümden tanıdığınız `Pair` biçimindedir. Değerleri şöyle
okursunuz:

| PostgreSQL türü | `kind()` | Okuma |
| --- | --- | --- |
| `NULL` | `"Null"` | `isNull()` |
| `boolean` | `"Bool"` | `bool()` |
| `smallint`, `integer`, `bigint` | `"Int"` | `int()` |
| `real`, `double precision` | `"Real"` | `real()` |
| `numeric` | `"String"` | `string()`, örneğin `"91.50"` |
| `uuid`, `text`, `json`, `jsonb` | `"String"` | `string()` |
| `date`, `timestamp`, `timestamptz` | `"String"` | `string()`; `timestamptz` her zaman UTC (`+00`) |
| `bytea` | `"Binary"` | `binarySize()`, `binaryBase64()` |

İki önemli ayrıntı:

- **`numeric` bir `String` olarak gelir.** `91.50` gibi bir değeri `Real`'e
  çevirmek kuruş hataları doğurabilir; bu yüzden AhdCode bunu sessizce yapmaz.
  Aritmetik gerekiyorsa açıkça `real(ortalama)` yazın ya da hesabı SQL'de
  yaptırın.
- **Zamanlar UTC gösterilir.** `timestamptz` bir andır; AhdCode onu her zaman
  `+00` ile yazar. Yanlış erişimciyi çağırmak (örneğin `ortalama` için
  `int()`) açıklayıcı bir `PostgreSQLError` verir.

### 59.6 Güncellemek, NULL ve özet sorgular

```ahd
require("Baglanti.ahd")
bring PostgreSQL
from PostgreSQL bring PostgreSQLResult

db := okulBaglan()

yukseltilen: PostgreSQLResult := db.execute(
    "UPDATE ogrenciler SET ortalama = ortalama + $1::numeric WHERE ortalama < $2::numeric"
    [PostgreSQL.fromString("5"), PostgreSQL.fromString("80")]
)
write("notu yükseltilen: " + str(yukseltilen.affectedRows()))

pasif: PostgreSQLResult := db.execute(
    "UPDATE ogrenciler SET aktif = $1, ortalama = NULL WHERE eposta = $2"
    [PostgreSQL.fromBool(false), PostgreSQL.fromString("zeynep@example.com")]
)
write("pasif yapılan: " + str(pasif.affectedRows()))

satirlar := db.query("SELECT ad, aktif, ortalama FROM ogrenciler ORDER BY ad")
for satir in satirlar {
    ortalama: Local String := "yok"
    if satir["ortalama"].isNull() == false {
        ortalama = satir["ortalama"].string()
    }
    aktif: Local String := str(satir["aktif"].bool())
    write(
        satir["ad"].string() + " | aktif: " + aktif + " | ortalama: " + ortalama
    )
}

ozet := db.query(
    "SELECT count(*) AS toplam, round(avg(ortalama), 2) AS genel FROM ogrenciler WHERE aktif"
)
toplam: Int := ozet[0]["toplam"].int()
genel: String := ozet[0]["genel"].string()
write("aktif öğrenci: " + str(toplam) + ", genel ortalama: " + genel)
db.close()
```

Beklenen çıktı:

```text
notu yükseltilen: 1
pasif yapılan: 1
Ayşe Yılmaz | aktif: true | ortalama: 91.50
Mehmet Kaya | aktif: true | ortalama: 83.25
Zeynep Demir | aktif: false | ortalama: yok
aktif öğrenci: 2, genel ortalama: 87.38
```

- `execute`, bir `PostgreSQLResult` döndürür; `affectedRows()` o ifadenin kaç
  satırı değiştirdiğini söyler.
- SQL `NULL`, AhdCode `null`'ı değildir: `kind()` değeri `"Null"` olan bir
  `PostgreSQLValue`'dur. Okumadan önce `isNull()` ile sorun.
- `count(*)` bir `bigint` döndürür ve `int()` ile okunur; `avg(...)` ise
  `numeric` olduğu için `String` gelir.

Silmek de aynı kalıptır:

```ahd
require("Baglanti.ahd")
bring PostgreSQL

db := okulBaglan()
silinen := db.execute(
    "DELETE FROM ogrenciler WHERE aktif = $1"
    [PostgreSQL.fromBool(false)]
)
write("silinen öğrenci: " + str(silinen.affectedRows()))
db.close()
```

### 59.7 Transaction: hep ya da hiç

Bir işlem (transaction), birkaç ifadeyi "ya hepsi kaydedilsin ya hiçbiri" diye
gruplar. `db.begin()` bir `PostgreSQLTransaction` verir; ifadeleri onun
üzerinden çalıştırır, sonunda `commit()` ya da `rollback()` çağırırsınız.

```ahd
require("Baglanti.ahd")
bring PostgreSQL
bring UUID
from PostgreSQL bring (PostgreSQLTransaction, PostgreSQLError)

db := okulBaglan()
kayit := "INSERT INTO ogrenciler (id, ad, eposta) VALUES ($1::uuid, $2, $3)"

islem: PostgreSQLTransaction := db.begin()
attempt {
    islem.execute(
        kayit
        [
            PostgreSQL.fromString(UUID.v7().string())
            PostgreSQL.fromString("Can Öz")
            PostgreSQL.fromString("can@example.com")
        ]
    )
    // Aynı e-posta ikinci kez: UNIQUE kuralı bu ifadeyi reddeder.
    islem.execute(
        kayit
        [
            PostgreSQL.fromString(UUID.v7().string())
            PostgreSQL.fromString("Can Öz (kopya)")
            PostgreSQL.fromString("can@example.com")
        ]
    )
    islem.commit()
    write("iki kayıt da kaydedildi")
}
except PostgreSQLError as error {
    islem.rollback()
    write("geri alındı: " + error.message)
}

sayim := db.query(
    "SELECT count(*) AS n FROM ogrenciler WHERE eposta = $1"
    [PostgreSQL.fromString("can@example.com")]
)
write("can@example.com kayıt sayısı: " + str(sayim[0]["n"].int()))
db.close()
```

Beklenen çıktı:

```text
geri alındı: PostgreSQL execution failed: (23505) duplicate key value violates unique constraint "ogrenciler_eposta_key"
can@example.com kayıt sayısı: 0
```

İlk `INSERT` başarılı olduğu hâlde tabloya **hiçbir şey** yazılmadı. İşlem geri
alındığı için yarım kayıt kalmadı.

**PostgreSQL'in katı kuralı.** MySQL'de işlem içindeki bir ifade başarısız
olursa işleme devam edebilirsiniz. PostgreSQL'de edemezsiniz: bir ifade
başarısız olduktan sonra işlem **iptal edilmiş** sayılır. Aşağıdaki program bu
kuralı bilerek çiğner:

```ahd
require("Baglanti.ahd")
bring PostgreSQL
from PostgreSQL bring (PostgreSQLTransaction, PostgreSQLError)

db := okulBaglan()
islem: PostgreSQLTransaction := db.begin()

attempt {
    islem.execute(
        "UPDATE ogrenciler SET ortalama = $1::numeric WHERE eposta = $2"
        [
            PostgreSQL.fromString("çok iyi")
            PostgreSQL.fromString("ayse@example.com")
        ]
    )
}
except PostgreSQLError as error {
    write("1. ifade: " + error.message)
}

attempt {
    islem.execute(
        "UPDATE ogrenciler SET aktif = true WHERE eposta = $1"
        [PostgreSQL.fromString("mehmet@example.com")]
    )
}
except PostgreSQLError as error {
    write("2. ifade: " + error.message)
}

attempt {
    islem.commit()
}
except PostgreSQLError as error {
    write("commit: " + error.message)
}
islem.rollback()
db.close()
```

Beklenen çıktı:

```text
1. ifade: PostgreSQL execution failed: (22P02) invalid input syntax for type numeric: "çok iyi"
2. ifade: PostgreSQL execution failed: (25P02) current transaction is aborted, commands ignored until end of transaction block
commit: PostgreSQL transaction was rolled back because an earlier statement failed
```

Kurala göre:

- İlk hatadan sonra aynı işlemdeki **her** ifade reddedilir.
- İptal edilmiş bir işlemde `commit()` hiçbir şeyi kaydetmez ve hata verir.
- Başarısız bir `commit()` sonrasında `rollback()` sessizce hiçbir şey yapmaz.
  Bu yüzden 59.7'nin ilk programındaki `attempt` / `except` / `rollback()`
  kalıbı her durumda güvenlidir.

### 59.8 Hata kodlarını anlamak

Sunucudan gelen her hata parantez içinde beş karakterlik bir **SQLSTATE**
kodu taşır. En sık göreceklerinizden bazıları:

| Kod | Anlamı |
| --- | --- |
| `23505` | `UNIQUE` kuralı ihlal edildi (aynı değer zaten var) |
| `23502` | `NOT NULL` bir sütun boş bırakıldı |
| `23503` | yabancı anahtar (foreign key) kuralı ihlal edildi |
| `22P02` | değer o türe çevrilemedi (`"çok iyi"` bir sayı değil) |
| `42P01` | böyle bir tablo yok |
| `42703` | böyle bir sütun yok |
| `42601` | SQL yazım hatası |
| `25P02` | işlem daha önceki bir hata yüzünden iptal edildi |

Kullanıcıya teknik mesajı değil, anlaşılır bir cümle gösterin:

```ahd
require("Baglanti.ahd")
bring PostgreSQL
bring UUID
from PostgreSQL bring PostgreSQLError

kullaniciMesaji: Function := (error: PostgreSQLError) -> String {
    if error.message.contains("(23505)") {
        return "Bu e-posta adresiyle zaten bir kayıt var."
    }
    if error.message.contains("(23502)") {
        return "Zorunlu bir alan boş bırakıldı."
    }
    return "Kayıt yapılamadı. Lütfen daha sonra tekrar deneyin."
}

db := okulBaglan()
attempt {
    db.execute(
        "INSERT INTO ogrenciler (id, ad, eposta) VALUES ($1::uuid, $2, $3)"
        [
            PostgreSQL.fromString(UUID.v7().string())
            PostgreSQL.fromString("Ayşe Yılmaz")
            PostgreSQL.fromString("ayse@example.com")
        ]
    )
}
except PostgreSQLError as error {
    write(kullaniciMesaji(error))
}
db.close()
```

Beklenen çıktı:

```text
Bu e-posta adresiyle zaten bir kayıt var.
```

Hata mesajları parolanızı ve satır değerlerini tekrar eden `DETAIL` / `HINT`
alanlarını hiçbir zaman içermez; yine de ayrıntılı mesajı kullanıcıya değil,
sunucu günlüğüne yazın.

### 59.9 MySQL'den farkları

| | MySQL (51. bölüm) | PostgreSQL |
| --- | --- | --- |
| Yer tutucular | `?` | `$1`, `$2`, … |
| Varsayılan port | `3306` | `5432` |
| Üretilen kimlik | `lastInsertId()` | `RETURNING` ile `query` |
| Mantıksal değerler | tamsayı | gerçek `boolean`; `fromBool` ve `bool()` |
| Tarih ve saat | sunucunun metni | sabit biçimler; `timestamptz` UTC |
| İşlem içinde hata | işlem kullanılabilir kalır | işlem iptal olur; `commit()` hata verir |
| Tek çağrıda birden fazla ifade | desteklenmez | hata verir, hiçbiri çalışmaz |

Bir program `SQLite`, `MySQL` ve `PostgreSQL`'i birlikte `bring` edebilir;
türleri ayrıdır ve karışmaz.

### 59.10 Kapatmak ve temizlemek

`db.close()` bağlantı havuzunu bırakır; kapatılmış bir veritabanını kullanmak
`this PostgreSQLDatabase is closed` hatası verir. Bir web uygulamasında
genellikle tek bir bağlantı açık tutulur ve bütün istekler onu paylaşır;
`PostgreSQLDatabase` aynı anda kullanıma uygun bir havuzdur. 61. bölümdeki
uygulama bunu gösterir.

Bu bölümün tablosunu silmek için:

```ahd
require("Baglanti.ahd")

db := okulBaglan()
db.execute("DROP TABLE IF EXISTS ogrenciler")
write("tablo silindi")
db.close()
```

**Siz deneyin:** Bir `dersler` tablosu (`id uuid`, `ad text`, `kredi integer`)
oluşturun, üç ders ekleyin ve `SELECT sum(kredi) AS toplam FROM dersler`
sonucunu `int()` ile okuyun.

Tam başvuru: [PostgreSQL](POSTGRESQL_TR.md) ·
[`examples/v0.1/71_postgresql.ahd`](../examples/v0.1/71_postgresql.ahd).

## 60. WebSocket: canlı bağlantılar

### 60.1 HTTP ile WebSocket arasındaki fark

36. bölümdeki web sayfaları **istek-yanıt** ile çalışır: tarayıcı sorar, sunucu
yanıtlar, bağlantı biter. Sunucunun "yeni bir mesaj geldi" diyebilmesi için
tarayıcının tekrar sorması gerekir.

**WebSocket** açık kalan bir bağlantıdır. Bir kez kurulur; sonra iki taraf da
istediği an mesaj gönderebilir:

```text
HTTP:       tarayıcı --istek--> sunucu --yanıt--> (bağlantı biter)

WebSocket:  tarayıcı ==açılış==> sunucu
            tarayıcı <--mesaj--- sunucu    (sunucu istediği an gönderir)
            tarayıcı ---mesaj--> sunucu
            ...                            (biri kapatana kadar açık)
```

Sohbet uygulamaları, canlı skor tabloları, yoklama panoları ve bildirimler
böyle çalışır. AhdCode v1.4.0'da WebSocket **sunucu** tarafıdır: uç noktalar
`HTTP` modülündeki `Server` üzerinde, rotalarınızın yanında yaşar. Karşı
taraftaki istemci çoğu zaman tarayıcıdaki JavaScript'tir.

### 60.2 Yankı sunucusu

Aşağıdaki program hem bir sayfa hem de bir WebSocket uç noktası sunar. Sayfa
bağlanır; yazdığınız her mesaj sunucudan geri gelir.

```ahd
bring HTTP
from HTTP bring (Server, Request, Response, WebSocket)

sayfa: Function := (request: Request) -> Response {
    return HTTP.html(r"""<!doctype html>
<html lang="tr">
<meta charset="utf-8">
<title>Yankı</title>
<input id="mesaj" placeholder="Bir şey yazın">
<button id="gonder">Gönder</button>
<ul id="gelenler"></ul>
<script>
const socket = new WebSocket(`ws://${location.host}/yanki`);
const liste = document.getElementById("gelenler");
socket.addEventListener("message", (olay) => {
  const satir = document.createElement("li");
  satir.textContent = olay.data;
  liste.append(satir);
});
document.getElementById("gonder").addEventListener("click", () => {
  const kutu = document.getElementById("mesaj");
  socket.send(kutu.value);
  kutu.value = "";
});
</script>
""")
}

yanitla: Function := (socket: WebSocket, metin: String) -> Nothing {
    socket.send("sunucu aldı: " + metin)
}

server: Server := HTTP.server("127.0.0.1", 8080)
server.get("/", sayfa)
server.websocket("/yanki", HTTP.websocket(yanitla))
server.start()
```

`ahdcode run yanki.ahd` ile çalıştırın ve `http://127.0.0.1:8080/` adresini
açın. Bir şey yazıp **Gönder**'e bastığınızda altında `sunucu aldı: ...`
belirir.

Parça parça:

- `HTTP.websocket(yanitla)` bir **uç nokta** oluşturur. `yanitla`, her mesaj
  geldiğinde `(socket, metin)` ile çağrılır.
- `server.websocket("/yanki", ...)` uç noktayı `/yanki` yoluna bağlar. Bunu
  `server.start()` çağrısından **önce** yapın. Aynı yol hem `get` rotası hem
  de WebSocket olamaz.
- `socket.send(metin)` o bağlantıya bir mesaj gönderir.
- HTML içindeki `${location.host}` JavaScript'e aittir. Onu `r"""..."""` ham
  String'i içine yazdığımız için AhdCode `{...}` kısmını interpolasyon sanmaz.
- Tarayıcı gelen mesajı `textContent` ile yazar, `innerHTML` ile değil. Böylece
  biri `<script>` gönderse bile bu yalnızca metin olarak görünür.

### 60.3 Yaşam döngüsü: açılış, mesaj, kapanış

Bir bağlantının üç anı vardır ve her biri için bir geri çağrı verebilirsiniz.
Bu program her anı bir günlüğe yazar ve günlüğü `/gunluk` adresinde gösterir:

```ahd
bring HTTP
from HTTP bring (Server, Request, Response, WebSocket, WebSocketEndpoint)

gunluk: List<String> := []

kaydet: Function := (satir: String) -> Nothing {
    gunluk: Global List<String>
    gunluk.add(satir)
}

acildi: Function := (socket: WebSocket, request: Request) -> Nothing {
    kaydet("açıldı: " + socket.id())
    socket.send("hoş geldiniz")
}

geldi: Function := (socket: WebSocket, metin: String) -> Nothing {
    kaydet(socket.id() + " yazdı: " + metin)
    if metin == "güle güle" {
        socket.close(1000, "görüşürüz")
    }
    else {
        socket.send("aldım: " + metin)
    }
}

kapandi: Function := (socket: WebSocket, kod: Int, neden: String) -> Nothing {
    kaydet("kapandı: " + socket.id() + " kod " + str(kod) + " neden: " + neden)
}

gunlukSayfasi: Function := (request: Request) -> Response {
    gunluk: Global List<String>
    metin: Local String := ""
    for satir in gunluk {
        metin = metin + satir + "\n"
    }
    return HTTP.text(metin)
}

ucNokta: WebSocketEndpoint := HTTP.websocket(geldi)
ucNokta = ucNokta.withOpen(acildi)
ucNokta = ucNokta.withClose(kapandi)

server: Server := HTTP.server("127.0.0.1", 8080)
server.get("/gunluk", gunlukSayfasi)
server.websocket("/canli", ucNokta)
server.start()
```

Sunucu çalışırken tarayıcıda `http://127.0.0.1:8080/gunluk` adresini açın
(şimdilik boş), geliştirici konsolunu açıp şunu yazın:

```js
socket = new WebSocket("ws://127.0.0.1:8080/canli")
socket.onmessage = (olay) => console.log(olay.data)
socket.send("merhaba")
socket.send("güle güle")
```

Konsolda `hoş geldiniz` ve `aldım: merhaba` görünür. Sonra sayfayı yenileyin;
günlük şuna benzer:

```text
açıldı: 3kQ9mZ0bT1u2w8xY4vLc7A
3kQ9mZ0bT1u2w8xY4vLc7A yazdı: merhaba
3kQ9mZ0bT1u2w8xY4vLc7A yazdı: güle güle
kapandı: 3kQ9mZ0bT1u2w8xY4vLc7A kod 1000 neden: görüşürüz
```

Günlük sıradan bir `List<String>`'dir; geri çağrılar ve `/gunluk` işleyicisi onu
`Global` ile paylaşır. WebSocket geri çağrılarının HTTP işleyicileriyle aynı
veriyi paylaşabilmesi, 60.4 ve 60.5'te kullanacağımız temel fikirdir.

AhdCode şu sırayı garanti eder:

1. `withOpen` her bağlantı için **tam bir kez** ve ilk olarak çalışır. Tarayıcı
   bağlantıyı açık görmeden önce biter; `acildi` içinde gönderilen mesaj ilk
   mesaj olarak ulaşır.
2. Mesaj geri çağrısı her mesaj için geliş sırasıyla bir kez çalışır.
3. `withClose` her bağlantı için **tam bir kez** ve en son çalışır; bağlantı
   ister düzgün kapansın ister kopsun.

`socket.id()`, bağlantıya özgü ve hiç tekrar kullanılmayan bir metindir.
Kapanış kodları:

| Kod | Anlamı |
| --- | --- |
| `1000` | normal kapanış |
| `1001` | taraf ayrılıyor (örneğin sayfa kapandı) |
| `1003` | ikili (binary) mesaj gönderildi; yalnızca metin desteklenir |
| `1006` | bağlantı koptu; ağ üzerinden hiç gönderilmez, yalnızca bildirilir |
| `1007` | mesaj geçerli UTF-8 değil |
| `1008` | politika ihlali (örneğin gönderim kuyruğu doldu) |
| `1009` | mesaj izin verilen boyuttan büyük |
| `1011` | sunucudaki geri çağrı hata verdi |

`close` için `1000`, `1001`, `1008`, `1011` ve `3000`–`4999` arası kodları
kullanabilirsiniz.

**Geri çağrılar birer birer çalışır.** Aynı sunucudaki HTTP işleyicileri ve
WebSocket geri çağrıları asla aynı anda çalışmaz. Bu, paylaşılan değişkenleri
kilitsiz kullanabilmenizi sağlar; ama yavaş bir geri çağrı herkesi bekletir.
Geri çağrıları kısa tutun.

### 60.4 Sohbet odası: kayıt tutmak ve herkese göndermek

Bir mesajı herkese göndermek için açık bağlantıları bir yerde tutmanız gerekir.
AhdCode bunu sizin yerinize yapmaz; sıradan bir `Pair` yeterlidir:

```ahd
bring HTTP
bring KeyValue
from HTTP bring (Server, Request, Response, WebSocket, WebSocketEndpoint)

baglantilar: Pair<String, WebSocket> := {}
isimler: Pair<String, String> := {}

herkeseGonder: Function := (metin: String) -> Int {
    baglantilar: Global Pair<String, WebSocket>
    ulasan: Local Int := 0
    for socket in KeyValue.values(baglantilar) {
        if socket.send(metin) {
            ulasan += 1
        }
    }
    return ulasan
}

katildi: Function := (socket: WebSocket, request: Request) -> Nothing {
    baglantilar: Global Pair<String, WebSocket>
    isimler: Global Pair<String, String>
    ad: Local String := "misafir"
    istenen: Local String? := request.query("isim")
    if istenen != null {
        if istenen.trim() != "" {
            ad = istenen.trim()
        }
    }
    herkeseGonder(ad + " sohbete katıldı")
    baglantilar[socket.id()] = socket
    isimler[socket.id()] = ad
    socket.send(
        "hoş geldin " + ad + ", odada " + str(len(baglantilar)) + " kişi var"
    )
}

mesajGeldi: Function := (socket: WebSocket, metin: String) -> Nothing {
    isimler: Global Pair<String, String>
    if metin.trim() == "" {
        return
    }
    herkeseGonder(isimler[socket.id()] + ": " + metin)
}

ayrildi: Function := (socket: WebSocket, kod: Int, neden: String) -> Nothing {
    baglantilar: Global Pair<String, WebSocket>
    isimler: Global Pair<String, String>
    ad: Local String := isimler[socket.id()]
    baglantilar = KeyValue.without(baglantilar, socket.id())
    isimler = KeyValue.without(isimler, socket.id())
    herkeseGonder(ad + " ayrıldı")
}

sayfa: Function := (request: Request) -> Response {
    return HTTP.html(r"""<!doctype html>
<html lang="tr">
<meta charset="utf-8">
<title>Sohbet</title>
<ul id="akis"></ul>
<form id="form"><input id="metin" autocomplete="off"><button>Gönder</button></form>
<script>
const ad = prompt("Adınız?") || "misafir";
const socket = new WebSocket(`ws://${location.host}/sohbet?isim=${encodeURIComponent(ad)}`);
socket.addEventListener("message", (olay) => {
  const satir = document.createElement("li");
  satir.textContent = olay.data;
  document.getElementById("akis").append(satir);
});
document.getElementById("form").addEventListener("submit", (olay) => {
  olay.preventDefault();
  const kutu = document.getElementById("metin");
  socket.send(kutu.value);
  kutu.value = "";
});
</script>
""")
}

ucNokta: WebSocketEndpoint := HTTP.websocket(mesajGeldi)
ucNokta = ucNokta.withOpen(katildi)
ucNokta = ucNokta.withClose(ayrildi)
ucNokta = ucNokta.withMaxMessageBytes(2000)
ucNokta = ucNokta.withMaxConnections(50)

server: Server := HTTP.server("127.0.0.1", 8080)
server.get("/", sayfa)
server.websocket("/sohbet", ucNokta)
server.start()
```

İki farklı tarayıcı sekmesinde `http://127.0.0.1:8080/` adresini açın, iki ayrı
ad girin ve yazışın. Bir sekmeyi kapattığınızda diğerinde "... ayrıldı"
görünür.

Önemli noktalar:

- **Kayıt `withOpen` içinde eklenir, `withClose` içinde silinir.**
  `KeyValue.without` (34. bölüm) anahtarı çıkarılmış yeni bir `Pair` döndürür.
- **`send` beklemez.** Mesajı o bağlantının kuyruğuna koyar ve `true` döndürür;
  bağlantı kapalıysa `false`. Mesajları okuyamayacak kadar yavaş bir istemcinin
  kuyruğu (varsayılan 64 mesaj) dolarsa bağlantısı `1008` ile kapatılır;
  böylece tek bir yavaş istemci sunucunun belleğini dolduramaz.
- **Sınırlar.** `withMaxMessageBytes(2000)` 2000 bayttan büyük mesajı `1009` ile
  reddeder; `withMaxConnections(50)` 51. bağlantıyı daha açılışta `503` ile
  geri çevirir. Varsayılanlar 65536 bayt ve 1024 bağlantıdır.
- Katılma duyurusunu, yeni kişiyi kayda eklemeden **önce** gönderdik; bu yüzden
  kişi kendi katılma mesajını almaz.

### 60.5 JSON mesajları ve bir HTTP işleyicisinden yayın

Gerçek uygulamalarda mesajlar çoğu zaman düz metin değil JSON'dur (31. bölüm).
Böylece tarayıcı gelen mesajın *türüne* bakarak ne yapacağına karar verir.

Ayrıca yayın her zaman bir WebSocket mesajından başlamaz: bir form gönderimi,
başka bir program ya da zamanlanmış bir iş de "herkese duyur" diyebilir. Geri
çağrılar ve HTTP işleyicileri aynı kilidi paylaştığı için bir HTTP işleyicisi
kayıt üzerinden doğrudan yayın yapabilir.

```ahd
bring HTTP
bring JSON
bring KeyValue
bring Env
bring Security
from HTTP bring (Server, Request, Response, WebSocket, WebSocketEndpoint)
from JSON bring (JSONValue, JSONError)

panolar: Pair<String, WebSocket> := {}

olay: Function := (tur: String, metin: String) -> String {
    return JSON.stringify(
        JSON.object(
            {"tur": JSON.fromString(tur), "metin": JSON.fromString(metin)}
        )
    )
}

yayinla: Function := (satir: String) -> Int {
    panolar: Global Pair<String, WebSocket>
    ulasan: Local Int := 0
    for socket in KeyValue.values(panolar) {
        if socket.send(satir) {
            ulasan += 1
        }
    }
    return ulasan
}

acildi: Function := (socket: WebSocket, request: Request) -> Nothing {
    panolar: Global Pair<String, WebSocket>
    panolar[socket.id()] = socket
    socket.send(olay("bilgi", "bağlandınız"))
}

geldi: Function := (socket: WebSocket, metin: String) -> Nothing {
    attempt {
        belge: Local JSONValue := JSON.parse(metin)
        tur: Local JSONValue? := belge.get("tur")
        if tur != null {
            if tur.string() == "ping" {
                socket.send(olay("pong", "buradayım"))
                return
            }
        }
        socket.send(olay("hata", "bilinmeyen mesaj türü"))
    }
    except JSONError as error {
        socket.send(olay("hata", "JSON bekleniyordu"))
    }
}

kapandi: Function := (socket: WebSocket, kod: Int, neden: String) -> Nothing {
    panolar: Global Pair<String, WebSocket>
    panolar = KeyValue.without(panolar, socket.id())
}

duyuru: Function := (request: Request) -> Response {
    beklenen: Local String? := Env.secret("DUYURU_TOKEN")
    gelen: Local String? := request.header("Authorization")
    if beklenen == null or gelen == null {
        return HTTP.text("yetkisiz", 401)
    }
    if Security.secureEqual("Bearer " + beklenen, gelen) == false {
        return HTTP.text("yetkisiz", 401)
    }
    ulasan: Local Int := yayinla(olay("duyuru", request.body()))
    return HTTP.text("{ulasan} panoya gönderildi")
}

ucNokta: WebSocketEndpoint := HTTP.websocket(geldi)
ucNokta = ucNokta.withOpen(acildi)
ucNokta = ucNokta.withClose(kapandi)

server: Server := HTTP.server("127.0.0.1", 8080)
server.websocket("/pano", ucNokta)
server.post("/duyuru", duyuru)
server.start()
```

Çalıştırın:

```bash
DUYURU_TOKEN=uzun-rastgele-bir-deger ahdcode run pano.ahd
```

Başka bir terminalden bir duyuru gönderin:

```bash
curl -H "Authorization: Bearer uzun-rastgele-bir-deger" --data "Yarın ders yok" http://127.0.0.1:8080/duyuru
```

Açık her panoya şu mesaj ulaşır ve `curl` kaç panoya gittiğini yazar:

```text
{"tur":"duyuru","metin":"Yarın ders yok"}
```

Bu örnekte:

- Mesajlar `JSON.object` ile kurulur, metin birleştirerek değil. Böylece
  `"` ya da `\` içeren bir duyuru JSON'u bozamaz.
- Gelen metin `JSON.parse` ile ayrıştırılır; bozuk JSON `JSONError` fırlatır
  ve istemciye kibar bir hata mesajı gider.
- `/duyuru`, 58. bölümdeki `Env.secret` ile okunan bir belirteç ister ve onu
  50. bölümdeki `Security.secureEqual` ile karşılaştırır.

Tarayıcı tarafında JSON'u şöyle okursunuz:

```js
socket.addEventListener("message", (olay) => {
  const mesaj = JSON.parse(olay.data);
  if (mesaj.tur === "duyuru") {
    alert(mesaj.metin);
  }
});
socket.send(JSON.stringify({ tur: "ping" }));
```

### 60.6 Güvenlik: kim bağlanabilir?

**Köken (origin) denetimi.** Tarayıcılar her WebSocket açılışında sayfanın
adresini `Origin` başlığıyla gönderir. AhdCode varsayılan olarak yalnızca
**aynı kökene** izin verir: `http://127.0.0.1:8080` üzerindeki bir sayfa
bağlanabilir, başka bir sitedeki sayfa `403` alır. Uygulamanız ayrı bir alan
adında çalışan bir ön yüze hizmet veriyorsa izin verilen kökenleri tam olarak
yazın:

```ahd
ucNokta = ucNokta.withAllowedOrigins(["https://okul.example.com"])
```

Joker karakter (`"*"`) kabul edilmez.

**Giriş yapmış kullanıcılar.** Bağlantıyı açılmadan önce reddetmek için
`withAccept` kullanın. `null` döndürmek kabul etmek, bir `Response` döndürmek
reddetmek demektir. 37. bölümdeki oturum deposuyla:

```ahd
kabul: Function := (request: Request) -> Response? {
    oturumlar: Global SessionStore
    oturum: Local Session := oturumlar.open(request)
    if oturum.get("user_id") == null {
        return HTTP.text("önce giriş yapın", 401)
    }
    return null
}
ucNokta = ucNokta.withAccept(kabul)
```

Tarayıcılar WebSocket açarken kendi başlığınızı eklemenize izin vermez, ama
çerezleri gönderir; bu yüzden tarayıcı için oturum çerezi doğru yoldur.
Başka bir programın bağlandığı durumlarda 60.5'teki gibi bir
`Authorization: Bearer ...` başlığı kullanılabilir.

**Bilmeniz gereken sınırlar:**

- Yalnızca metin mesajları desteklenir; ikili mesaj bağlantıyı `1003` ile
  kapatır.
- Mesaj içerikleri hiçbir zaman günlüğe yazılmaz.
- v1.4.0'da AhdCode'dan başka bir sunucuya bağlanan bir WebSocket **istemcisi**
  yoktur.

### 60.7 Tarayıcıda yeniden bağlanmak

Sunucu yeniden başlatıldığında (`ahdcode dev` her kaydetmede bunu yapar) ya da
ağ koptuğunda bağlantı kapanır. **Hiçbir taraf kendiliğinden yeniden
bağlanmaz**; bunu sayfanın JavaScript'i yapmalıdır. Bekleme süresini her
denemede ikiye katlayan basit bir kalıp:

```js
let bekleme = 1000;

function baglan() {
  const socket = new WebSocket(`ws://${location.host}/pano`);
  socket.addEventListener("open", () => {
    bekleme = 1000;
  });
  socket.addEventListener("message", (olay) => {
    console.log(olay.data);
  });
  socket.addEventListener("close", () => {
    setTimeout(baglan, bekleme);
    bekleme = Math.min(bekleme * 2, 30000);
  });
}

baglan();
```

Sunucu her bağlantıya 30 saniyede bir `ping` gönderir; yanıt vermeyen istemci
`1006` ile kapatılır. Bu sayede kopmuş bağlantılar kayıtta sonsuza dek kalmaz.

### 60.8 `Web` çatısıyla ve yayında

52. bölümdeki `Web` çatısında aynı uç nokta `Web.websocket` ile oluşturulur ve
`App.websocket` ile kaydedilir:

```ahd
bring Web
from Web bring (App, WebSocket)

yanki: Function := (socket: WebSocket, metin: String) -> Nothing {
    socket.send(metin)
}

site: App := Web.start()
site.websocket("/canli", Web.websocket(yanki))
site.start()
```

`Web.start()`, 52. bölümdeki `APP_NAME`, `APP_ENV` gibi ayarları `.env`
dosyasından okur. WebSocket uç noktaları bir rota grubuna (`RouteGroup`)
eklenemez; bekçi kontrolünüzü `withAccept` içine yazın.

Uygulamayı yayına aldığınızda önündeki ters vekil (reverse proxy) yükseltmeyi
geçirmelidir. Caddy'nin `reverse_proxy` yönergesi bunu ek ayar olmadan yapar;
HTTPS kullanan sitelerde tarayıcı `ws://` yerine `wss://` ile bağlanır. nginx
ayarları için [WebSocket başvurusuna](WEBSOCKET_TR.md#ters-vekil-arkasında)
bakın.

**Siz deneyin:** 60.4'teki sohbet odasına bir `/kim` komutu ekleyin: bir
kullanıcı `/kim` yazdığında yalnızca ona, odadaki kişilerin adlarını gönderin.

Tam başvuru: [WebSocket](WEBSOCKET_TR.md) ·
[`examples/v0.1/72_websocket_echo.ahd`](../examples/v0.1/72_websocket_echo.ahd).

## 61. Hepsi bir arada: gerçek zamanlı yoklama uygulaması

[`examples/v1.4/realtime_attendance`](../examples/v1.4/realtime_attendance/README_TR.md),
57–60. bölümlerde öğrendiklerinizi tek bir Web uygulamasında birleştirir.
Öğretmen giriş yapar, bir öğrencinin adını yazıp **Check in**'e basar;
açık olan her pano yeni yoklamayı sayfa yenilenmeden hemen gösterir.

### 61.1 Çalıştırmak

59.1'deki gibi bir PostgreSQL sunucusu açın; ardından `attendance` adlı bir
kullanıcı ve veritabanı oluşturun:

```bash
createuser --pwprompt attendance
createdb --owner attendance attendance
```

Sonra:

```bash
cd examples/v1.4/realtime_attendance
cp .env.example .env
psql "host=127.0.0.1 dbname=attendance user=attendance" -f schema.sql
ahdcode dev app.ahd
```

`.env` içinde `DB_PASSWORD` ile uzun ve rastgele bir `NOTIFY_TOKEN` ayarlayın.
Uygulama `http://127.0.0.1:8140/` adresinde açılır. İki tarayıcı penceresinde
açıp birinden yoklama alın; diğeri anında güncellenir. İsterseniz ikinci bir
terminalde `ahdcode run jobs.ahd` çalıştırın: dakikada bir, son bir saatin
özetini panolara duyurur.

### 61.2 Bir yoklamanın yolculuğu

```text
tarayıcı: form POST /check-in
   │
   ▼
Pages/CheckIn.ahd
   ├─ UUID.v7() ile yeni kimlik                        (57. bölüm)
   ├─ INSERT ... VALUES ($1::uuid, $2) RETURNING created_at   (59. bölüm)
   ├─ JSON satırı kur ve liveBroadcast(...)            (60. bölüm)
   └─ Web.redirect("/")
   │
   ▼
Live.ahd: kayıttaki her WebSocket'e send
   │
   ▼
public/js/live.js: JSON.parse, listeye textContent ile ekle
```

Veritabanı bağlantısı `Config/Database.ahd` içinde bir kez açılır ve parolası
58. bölümdeki gibi okunur:

```text
password: Local String? := Env.secret("DB_PASSWORD")
if password == null {
    toss Error("Set DB_PASSWORD, or DB_PASSWORD_FILE to a file holding it.")
}
```

`Live.ahd`, 60.6'daki gibi yalnızca giriş yapmış kullanıcıları kabul eder ve
panodan mesaj beklemez:

```text
liveMessage: Function := (socket: WebSocket, text: String) -> Nothing {
    socket.close(1008, "this socket is read-only")
}
```

`Pages/CheckIn.ahd` kaydı yazar ve herkese duyurur:

```text
checkInID: Local UUIDValue := UUID.v7()
stored: Local := database.query(
    "INSERT INTO check_ins (id, student) VALUES ($1::uuid, $2) RETURNING created_at"
    [
        PostgreSQL.fromString(checkInID.string())
        PostgreSQL.fromString(student)
    ]
)
```

### 61.3 Dikkat edilecek tasarım kararları

- **Sayfa JavaScript olmadan da tamdır.** Pano son 20 yoklamayı sunucuda
  çizer; `live.js` yalnızca sonradan gelenleri ekler.
- **Önce kaydet, sonra duyur.** Yayın, `INSERT` başarılı olduktan sonra
  yapılır; veritabanına yazılmamış bir yoklama asla panoda görünmez.
- **Yeniden bağlanma görünürdür.** `live.js` bağlantı koptuğunda bir geri
  sayım gösterir ve bekleme süresini 1 saniyeden 30 saniyeye kadar artırır.
- **İkinci program.** Bir Web sunucusu da bir Cron zamanlayıcısı da çalıştığı
  programı meşgul eder. Bu yüzden `jobs.ahd` ayrı çalışır ve özetini bearer
  belirteçli `POST /internal/notify` ile gönderir; 60.5'teki `/duyuru` ile aynı
  kalıp.

**Siz deneyin:** Uygulamaya "bugün kaç yoklama alındı?" sayısını gösteren bir
satır ekleyin. İpucu:
`SELECT count(*) AS n FROM check_ins WHERE created_at >= date_trunc('day', now())`.

## 62. Terminal: çıktı üzerinde daha fazla denetim

7. bölümdeki `write` ve `take` çoğu programa yeter. v1.5.0 ile gelen
`Terminal` modülü, bir programın daha fazlasına ihtiyaç duyduğu anlar içindir:
bir hatayı başka bir yere göndermek, çıktının hemen görünmesini sağlamak,
çıktıya birinin bakıp bakmadığını anlamak ya da bir sözcüğü renklendirmek.
`write`, `take` ve `str` değişmez.

### 62.1 String'leri emit ile birleştirmek

```ahd
bring Terminal

ad := "Ali"
puan := 95
Terminal.emit([ad, str(puan), "Geçti"])
Terminal.emit([ad, str(puan), "Geçti"], " | ")
Terminal.emit(parts: ["Yükleniyor"], ending: "")
Terminal.emit(parts: ["..."], separator: "", ending: "\n")
```

Beklenen çıktı:

```text
Ali 95 Geçti
Ali | 95 | Geçti
Yükleniyor...
```

`emit` bir `List<String>` alır, ayırıcıyı (varsayılanı bir boşluk) yalnızca iki
parçanın arasına koyar ve sonu (varsayılanı bir satır sonu) en sona ekler. Hiçbir
şeyi dönüştürmez; örneğin `str(puan)` yazmasının nedeni budur.

10. bölümdeki kuralı hatırlayın: bir çağrı ya tamamen sıralı ya da tamamen
isimli argümanlarla yapılır. `Terminal.emit([ad], " | ")` ve
`Terminal.emit(parts: [ad], separator: " | ")` ikisi de doğrudur;
aynı çağrıda sıralı bir List'in ardından isimli bir `separator` yazmak ise
derleme hatasıdır.

### 62.2 Hatalar standart hataya gider

```ahd
bring Terminal

write("çalışıyor...")
Terminal.error("Yapılandırma bulunamadı")
```

Bir programın iki çıktı akışı vardır. `write` ve `emit` **standart çıktıyı**,
`Terminal.error` ise **standart hatayı** kullanır. Programı
`ahdcode run program.ahd > cikti.txt` ile çalıştırın: `çalışıyor...` dosyaya
gider, ama hata yine ekranda, programı çalıştıran kişinin göreceği yerde görünür.

### 62.3 Çıktının hemen görünmesini sağlamak

Derlenmiş bir program standart çıktısını biriktirir ve daha büyük parçalar hâlinde
yazar; bu daha hızlıdır. Biriken metin program bittiğinde, `take` bir satır
okumadan önce ve `Terminal.error` yazmadan önce yazılır. 36. bölümdeki bir web
sunucusu ya da bekleyen bir döngü gibi çalışmaya devam eden bir program, metni
hemen yazmak için `Terminal.flush()` çağırabilir:

```ahd
bring Terminal

Terminal.emit(["Sunucu 8080 portunda başlıyor"])
Terminal.flush()
```

### 62.4 Çıktıya biri bakıyor mu?

```ahd
bring Terminal

genislik: Int? := Terminal.width()
if Terminal.isInteractive() and genislik != null {
    Terminal.emit(["Bu terminal {genislik} sütun genişliğinde."])
}
else {
    Terminal.emit(["Çıktı bir dosyaya ya da başka bir programa gidiyor."])
}
```

`Terminal.isInteractive()`, standart çıktı bir terminal penceresiyse `true`, bir
dosyaya yönlendirildiyse ya da başka bir programa aktarıldıysa `false` olur.
`Terminal.width()` ve `Terminal.height()` `Int?` döndürür: ölçülecek bir terminal
yoksa `null` olurlar ve AhdCode asla boyut tahmin etmez.

### 62.5 Rahatsız edecekleri yerde kaybolan renkler

```ahd
bring Terminal

gecti := Terminal.style(text: "Geçti", foreground: "green", bold: true)
kaldi := Terminal.style(text: "Kaldı", foreground: "red")
Terminal.emit(["Ali:", gecti])
Terminal.emit(["Mehmet:", kaldi])
```

Terminalde `Geçti` kalın yeşil, `Kaldı` kırmızı görünür. Çıktıyı bir dosyaya
yönlendirin; dosyada düz `Geçti` ve `Kaldı` olur: `style` yalnızca
`Terminal.supportsColor()` `true` iken renk ekler; bunun için bir terminal,
ayarlı olmayan ya da boş bir `NO_COLOR` ve `dumb` olmayan bir `TERM` gerekir.
Renkler `default`, `black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan` ve
`white`'tır; `"Red"` bile olsa başka her ad `TerminalError` verir.

### 62.6 Bir koleksiyonun içine bakmak

```ahd
bring Terminal

notlar: Pair<String, List<Int>> := {"Ali": [90, 95, 100], "Ayşe": [85, 91, 97]}
write(notlar)
Terminal.pretty(notlar)
```

Beklenen çıktı:

```text
{"Ali": [90, 95, 100], "Ayşe": [85, 91, 97]}
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

`write` her şeyi tek satıra koyar; `Terminal.pretty` ise bir List ya da Pair'i
satırlara yayar, böylece büyük bir koleksiyon kolayca okunur. Başka her değer
`write`'ın yazdığı gibi yazılır.

**Siz deneyin:** Üç öğrencinin tablosunu `Terminal.emit` ve `" | "` ile yazan,
50'nin altındaki her puanı kırmızıya boyayan ve bir puan eksikse
`Terminal.error` ile uyarı yazan bir program yazın. Programı önce normal, sonra
`> cikti.txt` ile çalıştırıp karşılaştırın.

[Terminal modül referansına](TERMINAL_TR.md) ve
[`examples/v1.5/terminal_demo`](../examples/v1.5/terminal_demo/README_TR.md)
klasörüne bakın.
