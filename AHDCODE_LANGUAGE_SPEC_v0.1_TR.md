# AhdCode Dil Spesifikasyonu v0.1

[English](AHDCODE_LANGUAGE_SPEC_v0.1.md) · [Türkçe]

> Bu belge, İngilizce **AhdCode Language Specification**'ın Türkçe karşılığıdır.
> Uygulama açısından iki belge aynı kuralları açıklar; metinsel bir çeviri
> uyuşmazlığı oluşursa İngilizce ana spesifikasyon esas alınır.

**Durum:** İlk uygulama için dondurulmuş (frozen) taslak çekirdek spesifikasyon
**Açıklama revizyonu:** 2026-08-28; açıklanan kurallar v0.1 için normatiftir
**Birincil uygulama hedefi:** Go
**Dosya uzantısı:** `.ahd`
**Başlangıç kapsamı:** yalnızca terminal/CLI dili. Web, HTTP, MySQL, SMTP,
HTML düzenleri (layouts), JSON'a özgü web kolaylıkları ve AhdWeb, çekirdek
dil güvenilir bir şekilde çalışana kadar açıkça ertelenmiştir.

---

## 1. Tasarım Felsefesi

AhdCode, birkaç güçlü kural etrafında tasarlanmıştır:

1. **Minimum satır sayısı yerine okunabilirlik.**
2. **Kısa teknik bir kısaltmanın hiçbir değer katmadığı yerde sade İngilizce
   kelimeler kullanın.**
3. **İlgisiz türleri sessizce zorlama (coerce) yapmayın.**
4. **Bildirimi ve değişikliği (mutation) görsel olarak farklı yapın.**
5. **Belirsiz olmadığında özlü (concise) sözdizimine izin verin, ancak tek
   bir kanonik formatter stili sağlayın.**
6. **Sıradan bir fonksiyon sorunu temiz bir şekilde çözebiliyorsa özel bir
   dil özelliği eklemeyin.**
7. **Yalnızca C, Python, Java, JavaScript veya başka bir dilde olduğu için
   özellik eklemeyin.**
8. **Derleyici, yalnızca güvenli ve benzersiz bir şekilde yapabildiğinde
   eksik tür ayrıntılarını çıkarabilir (infer edebilir). Kodun derlenmesini
   sağlamak için asla `Any`/dinamik davranışa geri dönmemelidir.**
9. **Ayrıştırıcı (parser) birkaç zararsız sunum biçimini kabul edebilir;
   kanonik AhdCode'un neye benzediğine formatter karar verir.**
10. **Önce çekirdek dil. Web, dilbilgisinin temeli olarak değil, daha sonra
    bir çalışma zamanı/kütüphane katmanı olarak gelir.**

AhdCode, Python gibi yaklaşılabilir, C ailesi diller gibi görsel olarak
yapılandırılmış ve aşırı törensellik olmadan statik olarak kontrollü
hissettirmelidir.

---

## 2. Büyük/Küçük Harf Duyarlılığı ve Tanımlayıcılar

AhdCode büyük/küçük harfe duyarlıdır (case-sensitive).

```ahd
student
Student
STUDENT
```

üç farklı tanımlayıcıdır (identifier).

AhdCode kaynak dosyaları UTF-8'dir. Geçersiz UTF-8, bir sözcüksel (lexical)
hatadır.

Tanımlayıcı kuralları, Python'un Unicode tanımlayıcı modelini izler:

- ilk karakter `_` veya bir Unicode `XID_Start` kod noktasıdır;
- sonraki her karakter `_` veya bir Unicode `XID_Continue` kod noktasıdır;
- tanımlayıcı karşılaştırmasından ve sembol aramasından önce, tanımlayıcılar
  Unicode NFKC ile normalleştirilir;
- normalleştirme, harf büyütme/küçültme katlaması (case folding) yapmaz;
  AhdCode büyük/küçük harfe duyarlı kalır;
- ayrılmış (reserved) AhdCode anahtar kelimeleri tanımlayıcı olarak
  kullanılamaz.

Örnekler:

```ahd
öğrenci2: String := "Ali"
_private: Int := 5

Student: Class<> := {
}
```

Geçersiz:

```ahd
2student: Int := 5
```

### 2.1 Ayrılmış ve bağlamsal (contextual) anahtar kelimeler

AhdCode v0.1, aşağıdaki kelimeleri her sözdizimsel bağlamda ayırır. Aynı
yazım ve büyük/küçük harfle tanımlayıcı olarak kullanılamazlar:

```text
and or not same is in has
if else while until for break continue
state condition default
attempt except ultimately toss return
bring from all as
true false null
Int Real String Bool Nothing
List Pair Function lambda Overload Override
Class Attributes Constant Local Global Confidential
Object Error
```

Aşağıdakiler bağlamsal anahtar kelimelerdir:

| Kelime | Anahtar kelime bağlamı |
|---|---|
| `structure` | bir Class içindeki `structure: Attributes := ...` bildirimi |
| `attribute` | Class yapı (structure) gövdeleri ve metotları içindeki örtük örnek alıcısı (implicit instance receiver) |
| `SuperClass` | kalıtsal öznitelik genişletmesi ve türetilmiş bir Class içindeki doğrudan-üst erişimi |

Bağlamsal bir anahtar kelime, yalnızca yukarıda listelenen bağlamda özel
anlamına sahiptir. O bağlamın dışında sıradan bir tanımlayıcı olarak
kullanılabilir.

`write`, `take`, `str`, `int`, `IndexError` gibi yerleşik (built-in) ve içe
aktarılmış isimler ile modül isimleri, ayrılmış anahtar kelimeler değil,
önceden bildirilmiş (predeclared) veya içe aktarılmış tanımlayıcılardır.

---

## 3. İfadeler, Satır Sonları, Virgüller ve Biçimlendirme

### 3.1 İfade sınırları

Satır sonları önemlidir ve ayrıştırıcı için kullanılabilir kalmalıdır.

Bir satır sonu, ayrıştırıcı bunun gerçekten devam eden bir ifade veya ifade
seviyesinde sınırlandırılmış bir yapı içinde olduğunu belirleyemedikçe bir
ifadeyi sonlandırır. Çok satırlı bir çağrı, koleksiyon literali, parametre
listesi, genel (generic) argüman listesi, indeks/dilim (slice) veya
gruplanmış bir ifade içinde, bir satır sonu bunun yerine öğeleri ayırabilir
veya o yapının dilbilgisine göre açık yapıyı devam ettirebilir.

Blok süslü parantezleri, ifade satır sonlarını bastırmaz (suppress etmez).
Çalıştırılabilir bir `{ ... }` bloğu içinde, bir satır sonu ifadeleri
ayırmaya devam eder. Bir bloğun kapanış parantezinin henüz görünmemiş
olması, bloktaki tüm satırları tek bir ifade yapmaz.

Bu yüzden bir satır sonu bir **ifadeyi (statement)** sonlandırır, mutlaka
bir **anlatımı (expression)** değil. Sözcüksel çözümleyicinin (lexer) satır
sonlarını genel olarak atması yerine, bu bağlamsal kararı ayrıştırıcı verir.

Bir sonek (infix) operatör veya açık bir anlatım sınırlandırıcısı,
anlatımı sözdizimsel olarak eksik yaptığında bir satır sonu bir anlatımı
devam ettirebilir. Bir anlatım tamamlandığında, onu izleyen satır sonu
ifadeyi sonlandırır. Sonraki satırın başındaki ikili (binary) bir operatör,
önceki tamamlanmış ifadeyi devam ettirmek için asla geriye uzanmaz.

Örneğin, bu `x = 5 + 2` olarak yorumlanmaz:

```ahd
x = 5
+ 2
```

Bu geçerli ve tercih edilendir:

```ahd
area: Local Real := PI * square(radius)
```

AhdCode `;` gerektirmez.

Birden fazla bağımsız ifade tek bir satıra sıkıştırılamaz.

Geçersiz:

```ahd
x: Int := 5 y: Int := 8 write(x + y)
```

### 3.2 Argüman ve eleman ayırıcılar

Fonksiyon argümanları ve koleksiyon elemanları için:

- bir virgül, aynı satırdaki öğeleri ayırabilir;
- bir satır sonu, öğeleri ayırabilir;
- yalnız başına düz boşluk, aynı satırdaki birden fazla argümanı
  **ayıramaz**.

Yokluk (absence), kaynak türlerinde `?` ile açıkça yazılır. Yalın `T`
kesinlikle null olamaz; `T?`, aynı temel türün null olabilen biçimidir.

Geçerli:

```ahd
swap(a, b)
```

```ahd
swap(
    a
    b
)
```

```ahd
swap(a
    b)
```

Geçersiz:

```ahd
swap(a b)
```

Aynı kavram, kısa ile çok satırlı koleksiyon literalleri için de geçerlidir.

Geçerli:

```ahd
numbers: List<Int> := [1, 2, 3]
```

```ahd
numbers: List<Int> := [
    1
    2
    3
]
```

### 3.3 Kanonik formatter politikası

Ayrıştırıcı, kasıtlı olarak formatter'dan daha izin vericidir.

Formatter şunları tercih etmelidir:

- virgüllerle tek satırda kısa çağrılar/listeler;
- her öğe kendi satırında ve zorunlu sondaki virgül olmadan çok satırlı
  çağrılar/listeler;
- çok satırlı fonksiyon bildirimleri;
- tutarlı girinti (indentation);
- satır başına bir ifade.

Biçimlendirme, program anlambilimini (semantics) değiştirmemelidir.

---

## 4. Yorumlar

Tek satırlık yorum:

```ahd
// comment
```

Çok satırlı yorum:

```ahd
/*
multiline
comment
*/
```

Çok satırlı yorumlar iç içe geçmez (nest). Bir açılış `/*`'ten sonraki ilk
`*/` yorumu kapatır. Yorumun içindeki herhangi bir ek `/*`, sıradan yorum
metnidir.

Yorumlar, formatter tarafından korunan önemsiz (trivia) öğelerdir. Lexer/token
modeli, ilk uygulama aşamasından itibaren kaynak aralıklarını (spans),
metnini ve yerleşimini korumalıdır; yorumlar biçimlendirmeden önce geri
dönüşü olmayan (irreversibly) şekilde atılmamalıdır.

---

## 5. Çekirdek Skaler Türler

AhdCode v0.1 şunları sağlar:

```text
Int
Real
String
Bool
```

### 5.1 Sayısal literal dilbilgisi

v0.1 sayısal literalleri yalnızca ASCII ondalık rakamları kullanır:

```text
digit       := "0" ... "9"
digits      := digit+
exponent    := ("e" | "E") ("+" | "-")? digits
IntLiteral  := digits
RealLiteral := digits "." digits exponent?
             | digits exponent
```

Örnekler:

```ahd
12
0012
1.25
1e6
1.2e-3
```

Baştaki sıfırlara izin verilir ve ondalık kalır. `_` ayırıcılar, `.5`, `5.`,
ikili/sekizli/onaltılık önekler, sayısal sonekler, `NaN` ve sonsuzluk
literalleri v0.1'in bir parçası değildir.

Baştaki bir `+` veya `-`, sayısal token'ın bir parçası değil, tekli (unary)
bir operatördür. Lexer, literal metni korur ve bir Int token'ını yalnızca
işaretsiz (unsigned) büyüklüğü pozitif `Int` maksimumundan büyük olduğu için
reddetmez.

İşaretli (signed) sabit ifadeler, semantik analiz sırasında değerlendirilir.
Nihai Int değerleri işaretli 64-bit aralığa sığmalıdır. Özellikle:

```ahd
minimum: Int := -9223372036854775808
```

geçerlidir, `9223372036854775807`'nin üzerindeki pozitif bir Int değeri ise
bir derleme zamanı semantik hatasıdır. Sonlu bir `float64` değeri
üretemeyen bir Real literali de aynı şekilde bir derleme zamanı semantik
hatasıdır.

### 5.2 Int

`Int`, işaretli 64-bit tam sayı (`int64` anlambilimi) olarak uygulanır.

```ahd
count: Int := 10
```

Tam sayı taşması (integer overflow) sessizce sarmalanmamalıdır (wrap).
Bir taşma çalışma zamanı hatası fırlatır.

### 5.3 Real

`Real`, 64-bit kayan noktalı gösterimle (`float64` anlambilimi) uygulanır.

```ahd
pi: Real := 3.14159
x: Real := 5
```

`Int -> Real`, izin verilen güvenli bir genişletme (widening) dönüşümüdür.

`Real` ismi, dil seviyesinde bir soyutlamadır. Matematiksel reel sayıların
tam gösterimini iddia **etmez**.

### 5.4 Bölme

`/` her zaman gerçek bölme yapar ve `Real` döndürür.

```ahd
5 / 2
```

şunu üretir:

```text
2.5
```

### 5.5 Açık sayısal dönüşümler

Önceden bildirilmiş Fundamentals dönüşümü `int`, tam olarak bir `Real` veya
`String` kabul eder ve `Int` döndürür. Bir `Real` girdisi sıfıra doğru
keser (truncate).

```ahd
int(3.7)   // 3
int(-3.7)  // -3
```

Bu, en yakın tam sayıya yuvarlama değil, bir dönüşümdür.

`String` girdisi için `int`, çevreleyen Unicode boşluklarını kırpar (trim)
ve ardından yalnızca şu ASCII-ondalık dilbilgisini kabul eder:

```text
IntText := ("+" | "-")? digits
digits  := ASCII digit+
```

Ondalık nokta, üs, alt çizgi veya onaltılık/ikili/sekizli biçim yoktur.
Geçersiz metin `DomainError` fırlatır; işaretli 64-bit aralığın dışında
matematiksel olarak geçerli bir ondalık ise `OverflowError` fırlatır.

Önceden bildirilmiş Fundamentals dönüşümü `real`, tam olarak bir `Int` veya
`String` kabul eder ve `Real` döndürür. Bir `Int` girdisi, dilin güvenli
`Int -> Real` genişletme dönüşümünün açık yazımıdır.

```ahd
real(2)         // 2.0
real("3")       // 3.0
real("3.14")    // 3.14
real("1e3")     // 1000.0
real("-2.5e-4") // -0.00025
```

`String` girdisi için `real`, çevreleyen Unicode boşluklarını kırpar ve
sıradan ASCII-ondalık sayısal-metin dilbilgisini kabul eder: isteğe bağlı
baştaki bir işaret, bir veya daha fazla rakam, `.` işaretinin her iki
tarafında rakamlar bulunan isteğe bağlı bir kesir ve kendi isteğe bağlı
işareti ve bir veya daha fazla rakamı olan isteğe bağlı bir `e` veya `E`
üssü. Ondalık tam sayı metni kabul edilir. Alt çizgiler, onaltılık/ikili/
sekizli biçimler, `NaN` ve sonsuzluk yazımları kabul edilmez. Geçersiz metin
`DomainError` fırlatır; sonlu olmayan veya `float64` aralığının dışında
ayrıştırılmış bir sonuç `OverflowError` fırlatır.

Bu dönüşüm isimleri genel bir zorlama (coercion) getirmez. Özellikle,
sayısal operatörler hâlâ String'leri reddeder ve `bool(...)` dönüşümü ile
truthiness v0.1 sözleşmesinin dışında kalır.

### 5.6 Bool

Yalnızca `Bool` değerleri koşul olarak kullanılabilir.

Python/JavaScript/PHP tarzı truthiness yoktur.

Geçersiz:

```ahd
if 5 {
}
```

Geçersiz:

```ahd
if "Ali" {
}
```

Geçerli:

```ahd
if age > 18 {
}
```

### 5.7 String

String'ler değiştirilemezdir (immutable).

```ahd
name: String := "Ali"
name += " Harun"
```

geçerlidir çünkü yeni bir String değeri üretilir ve yeniden bağlanır.

Geçersiz:

```ahd
name[0] = "V"
```

Bir String indeksi tek karakterli bir `String` döndürür. v0.1'de bir `Char`
türü yoktur.

---

## 6. String Literalleri, Kaçış Karakterleri, Unicode ve Interpolasyon

Tek ve çift tırnaklı string'ler eşdeğerdir:

```ahd
a: String := "hello"
b: String := 'hello'
```

Normal bir string'i açan sınırlayıcı (delimiter), onu kapatmalıdır da.

Geçersiz:

```ahd
"hello'
'hello"
```

Tam v0.1 kaçış karakteri kümesi şudur:

```text
\n
\r
\t
\\
\"
\'
\{
\}
```

Aynı kaçış karakteri kümesi normal ve çok satırlı string'lere uygulanır.
Başka herhangi bir kaçış dizisi bir sözcüksel hatadır. v0.1'de `\0`, `\b`,
`\f`, `\x...` veya `\u...` kaçış biçimi yoktur; Unicode karakterleri
doğrudan kaynak metninde yazılır.

Normal, tek veya çift tırnaklı bir string, fiziksel bir satır sonu
içeremez. Eşleşen üçlü tırnaklı bir string fiziksel satır sonları
içerebilir.

### 6.1 Çok satırlı string'ler

Hem üçlü-çift-tırnak hem de üçlü-tek-tırnak string'ler desteklenir.

```ahd
query: String := """
SELECT *
FROM students
WHERE name = '{name}'
"""
```

```ahd
text: String := '''
"double quotes" are allowed here.
'''
```

`""" ... """` içinde, sıradan `'` ve `"` karakterlerine izin verilir. Yalnızca
eşleşen üçlü sınırlayıcı literali kapatır.

Üçlü-string içeriği, açılış ve kapanış sınırlayıcıları arasında göründüğü
gibi tam olarak korunur. AhdCode otomatik girinti kaldırma (dedent),
kırpma (trim) veya ilk/son satır sonunu kaldırma yapmaz.

Yalnızca üç ardışık, eşleşen, kaçışsız tırnak karakteri bir üçlü string'i
kapatır. Kaçışlı bir tırnak, kapanış sınırlayıcısına katılmaz.

### 6.2 Interpolasyon

Tüm normal ve çok satırlı string'ler `{...}` interpolasyonunu destekler.

```ahd
name: String := "Ali"
write("Hello {name}")
```

Yalın süslü parantezlere kaçış işlemi uygulanabilir:

```ahd
text: String := "\{literal\}"
```

String-metin modunda, kaçışsız bir `{` interpolasyonu açar. İnterpolasyon,
eşleşen sıfır-derinlikteki (depth-zero) `}`'de sona erer. İçeriği, bir tane
boş olmayan AhdCode anlatımı olmalıdır; ifadelere (statements) ve
bildirimlere izin verilmez.

İç içe geçmiş `()`, `[]` ve `{}` sınırlayıcıları anlatım içinde
desteklenir. Anlatım içindeki string'ler ve yorumlar kendi sıradan sözcüksel
kurallarını izler ve içlerindeki süslü parantezler çevreleyen interpolasyon
derinliğini değiştirmez. Bir interpolasyon içindeki bir string literali,
kendi içinde interpolasyon içerebilir.

`\{` ve `\}`, string metninde yalın süslü parantezler üretir. String
metninde kaçışsız, eşleşmemiş bir `}` ve kapanış `}`'si olmayan bir
interpolasyon, ilgili süslü parantez aralığında bildirilen sözcüksel
hatalardır.

Sıradan tırnaklı bir string, interpolasyonu içinde dahil, fiziksel bir
satır sonu içeremez. Üçlü tırnaklı bir string içindeki bir interpolasyon
satırlara yayılabilir ve sıradan anlatım/satır sonu kurallarını izler.

İnterpolasyon anlatımı, `str` tarafından kabul edilen herhangi bir değer
üretebilir, ancak `Nothing` üretemez. Metinsel dönüşümü, `str(expression)`
ile aynı anlambilime sahiptir ve `+` gibi operatörler için genel
String-sayı zorlaması getirmez.

### 6.3 String işlemleri

`String` değiştirilemezdir. Aşağıdaki her işlem yeni bir değer döndürür ve
alıcısını (receiver) asla değiştirmez, bu yüzden birinin alıcısı diğerinin
sonucu olabilir. Alıcı ve her argüman `NonNull` olmalıdır.

```text
trim()                            -> String
lower()                           -> String
upper()                           -> String
capitalize()                      -> String
split(separator: String)          -> List<String>
replace(old: String, new: String) -> String
contains(text: String)            -> Bool
startsWith(prefix: String)        -> Bool
endsWith(suffix: String)          -> Bool
count(text: String)               -> Int
index(text: String)               -> Int
```

Bunlar, alıcının statik türü tarafından seçilen, `String` türü üzerindeki
tiplenmiş işlemlerdir. Fundamentals fonksiyonları değildir, hiçbir
parametre adı yayınlamazlar ve isimlendirilmiş bir argüman reddedilir. Bir
`Class`, alıcı türü kararı verdiği için, bu isimlerle kendi metotlarını
yine de bildirebilir.

`trim`, Unicode boşluğunu yalnızca baştan ve sondan kaldırır; iç kısımdaki
boşluklar korunur.

```ahd
write("[{"  Ali  ".trim()}]")
write("[{"\t Ali Harun \n".trim()}]")
```

=>

```text
[Ali]
[Ali Harun]
```

`lower` ve `upper`, deterministik, yerelden bağımsız Unicode basit
büyük/küçük harf eşlemeleridir. v0.1'de yerel (locale) yapılandırması ve
Türkçe-yerel özel büyük/küçük harf dönüşümü yoktur.

```ahd
write("AhdCode".lower())
write("AhdCode".upper())
```

=>

```text
ahdcode
AHDCODE
```

`capitalize`, ilk karakteri büyütür ve geri kalanı tam olarak yazıldığı gibi
bırakır. Python'un capitalize'ı değildir: metnin geri kalanı asla
küçültülmez. Boş bir String boş kalır.

```ahd
write("ali HARUN".capitalize())
write("aHD".capitalize())
```

=>

```text
Ali HARUN
AHD
```

Normalleştirme açıkça yazılır:

```ahd
write("ali HARUN".lower().capitalize())
```

=>

```text
Ali harun
```

`split`, ayırıcının her çakışmayan (non-overlapping) geçişinde böler ve
boş alanları korur. Ayırıcı boş olmamalıdır; boş bir ayırıcı yakalanabilir
`DomainError` fırlatır. v0.1'de parametresiz bir boşluk-bölme biçimi yoktur.

```ahd
write("a,b,c".split(","))
write("a,,b,".split(","))
write("".split(","))
```

=>

```text
["a", "b", "c"]
["a", "", "b", ""]
[""]
```

`replace`, soldan sağa, her çakışmayan geçişi yeniden yazar. Aranan metin
boş olmamalıdır ve öyleyse `DomainError` fırlatır; değiştirme metni boş
olabilir.

```ahd
write("banana".replace("na", "X"))
write("abc".replace("b", ""))
```

=>

```text
baXX
ac
```

`contains`, `startsWith` ve `endsWith`, sıradan String matematiğini izler,
bu yüzden boş bir arama metni eşleşir:

```ahd
write("abc".contains(""))
write("abc".startsWith(""))
write("abc".endsWith(""))
```

=>

```text
true
true
true
```

`count`, çakışmayan geçişleri sayar. AhdCode, boş bir arama metni için
uzunluk-artı-bir kuralını benimsemez: boş bir metin `DomainError` fırlatır.

```ahd
write("banana".count("a"))
write("banana".count("na"))
write("banana".count("x"))
```

=>

```text
3
2
0
```

`index`, ilk geçişi, asla bir UTF-8 byte ofseti değil, bir AhdCode
**karakter** indeksi olarak döndürür.

```ahd
write("banana".index("na"))
write("a✓b✓".index("✓"))
```

=>

```text
2
1
```

`index`'in gösterge (sentinel) bir sonucu yoktur. Alıcının içermediği bir
arama metni yakalanabilir `DomainError` fırlatır, boş bir arama metni de
öyle.

### 6.4 Ham (raw) String literalleri (v0.1.14)

Bir ham (raw) String literali, aralarında boşluk veya trivia olmadan
doğrudan bir String sınırlayıcısının önüne gelen, tam olarak küçük harfli
`r` öneki ile yazılır:

```ahd
r"raw text"
r'raw text'

r"""
multiline raw text
"""

r'''
multiline raw text
'''
```

Ham bir String hâlâ sıradan bir `String` değeridir. Ayrı bir `RawString`
türü yoktur ve bir ham String, bir kez oluşturulduktan sonra normal bir
String'den ayırt edilemez -- yalnızca onu üreten kaynak yazımı farklıdır.

Ham bir String'in içinde:

- kaçış işleme yoktur: `\n`, `\t`, `\\` ve diğer her ters eğik çizgi dizisi,
  çözümlenmeden olduğu gibi kopyalanır;
- `{...}` interpolasyonu yoktur: süslü parantezler sıradan karakterlerdir;
- literalin değeri, açılış ve kapanış sınırlayıcıları arasındaki tam kaynak
  metindir.

```ahd
name: String := "Ali"

write(r"{name}")
```

=>

```text
{name}
```

```ahd
write(r"\n")
```

=>

```text
\n
```

(bir satır sonu değil, ters eğik çizgi ve ardından `n` olmak üzere iki
karakter.)

Ham bir String'in kaçış mekanizması olmadığından, kendi sınırlayıcısını
içeremez: `r"..."` içindeki bir `"` veya `r'...'` içindeki bir `'` hiçbir
şeyle korunmaz, bu yüzden sınırlayıcının ilk geçişi her zaman literali
kapatır. Tam olarak o tırnak karakterine ihtiyaç duyan bir ham String, diğer
tırnak ailesini kullanmalıdır (`"` içeren metin için `r'...'`, `'` içeren
metin için `r"..."`), ya da kaçış gerektiğinde normal tırnaklı/üçlü biçimi
kullanmalıdır.

Ham üçlü string'ler, normal üçlü string'lerle aynı fiziksel satır sonu,
sınırlayıcı ve içerik koruma kurallarını izler (§6.1): literali yalnızca üç
ardışık eşleşen tırnak karakteri kapatır ve sınırlayıcılar arasındaki içerik,
girinti kaldırma veya kırpma olmadan tam olarak korunur.

Önek algılama, salt sözcüksel bir bitişiklik kontrolüdür: başka bir yerde
kullanılan `r` adlı bir tanımlayıcı -- `r := 5`, `r + 1`, `read("x")` --
etkilenmez, çünkü yalnızca hemen ardından `"` veya `'` gelen tek bir `r`
karakteri ham kipi etkinleştirir. `r` ayrılmış bir sözcük değildir. Yalnızca
tam olarak küçük harfli `r` ham kipi etkinleştirir; `R"..."`, ham bir String
değil, ayrı bir String literal belirteci tarafından izlenen sıradan bir `R`
tanımlayıcısıdır.

Ham String literalleri, aksi takdirde ağır kaçışlama gerektirecek metinleri
yazmak için vardır. İki temel kullanım:

Her `{n}` niceleyicisinin aksi takdirde kaçışlanması gereken düzenli ifade
(regular-expression) desenleri:

```ahd
bring Regex
from Regex bring Pattern

pattern: Pattern := Regex.compile(
    r"^MATH-[0-9]{3}$"
)
```

Ters eğik çizgilerin yaygın olduğu LaTeX kaynağı:

```ahd
formula: String := r"\frac{x^2 + 1}{\sqrt{x}}"
```

Özetle:

```text
normal String = kaçışlar + interpolasyon
ham String    = ne kaçış ne de interpolasyon
```

---

## 7. Nothing ve null

`Nothing` ve `null` kasıtlı olarak farklı kavramlardır.

### 7.1 Nothing

`Nothing`, AhdCode'un `void`'e eşdeğeridir.

Bir dönüş türüdür, bir çalışma zamanı değeri değildir.

```ahd
show: Function := (
    message: String
) -> Nothing {
    write(message)
    return
}
```

Sondaki yalın `return`, bir `Nothing` fonksiyonunda isteğe bağlıdır.

Geçersiz kavramlar:

```ahd
x: Nothing := ...
return Nothing
```

### 7.2 null

`null` şu anlama gelir:

> bildirilen tür bilinmektedir, ancak şu anda bir değer yoktur (absent).

Geçerli:

```ahd
name: String? := null
age: Int? := null
student: Student? := null
```

Bildirilen tür değişmeden kalır.

```ahd
age = 25       // geçerli
age = "Ali"    // tür hatası
```

`null` şunlar **değildir**:

- sıfır,
- boş String,
- false,
- `Nothing`,
- örtük bir dinamik tür.

AhdCode, semantik denetim sırasında **akış-duyarlı (flow-sensitive) null
durumu analizi** gerçekleştirir.

Dahili olarak, bir bağlama (binding) şu durumlardan biri olarak izlenebilir:

```text
Null
MaybeNull
NonNull
```

Bu durumlar derleyici analiz durumlarıdır. Herkese açık null olabilen tür
sözdizimini daraltırlar (refine); onun yerini almazlar.

Derleyici bir değerin null olduğunu veya null olabileceğini kanıtlayabiliyorsa,
güvensiz işlemler mümkün olduğunda derleme zamanında reddedilmelidir.

Örnek:

```ahd
age: Int? := null
write(age + 5)
```

bir derleme zamanı null-durumu hatasıdır.

Kesin, null olmayan bir atamadan sonra:

```ahd
age = 25
write(age + 5)
```

değer `NonNull` olarak kabul edilir ve işlem geçerlidir.

Dal (branch) koşulları null durumunu daraltır:

```ahd
student: Student? := findStudent()

if student != null {
    write(student.name)
}
```

Doğru dalın içinde, `student` null olmayan olarak ele alınır.

Kısa devre (short-circuit) Boolean mantığı da null-durumu daraltmasına
katılır:

```ahd
if student != null and student.age >= 18 {
    write(student.name)
}
```

`and`'in sağ tarafı, `student != null` bilgisi altında kontrol edilir.

Bir fonksiyon `null` döndürebiliyorsa, akış kontrolü analizi aksini
kanıtlamadıkça çağrı sonucu `MaybeNull` olarak izlenir.

Çalışma zamanı null kontrolleri, statik olarak kanıtlanamayan durumlar için
son bir güvenlik katmanı olarak kalır, ancak AhdCode kanıtlanabilir şekilde
güvensiz null kullanımını semantik analiz sırasında reddetmelidir.

Null-durumu bilgisi modül sınırlarını aşmalıdır. Dışa aktarılan bir bağlama
ve her somut dışa aktarılan çağrılabilir imza için derleyici tarafından
görülebilir sözleşme, çağrılabilir dönüş durumu dahil, ilgili `Null`,
`MaybeNull` veya `NonNull` durumunu korumalıdır. İçe aktarılan bir anlatım,
sessizce null olmayan veya dinamik olarak ele alınmak yerine, semantik
denetime bu korunmuş durumla başlar.

Bu diller-arası bilgi, derleyici meta verisidir, herkese açık AhdCode tür
sözdizimi değildir.

### 7.3 Constant null

Bir `Constant`, `null`'a başlatılamaz.

Geçersiz:

```ahd
id: Constant Int := null
```

### 7.4 Tiplenmiş koleksiyonlarda null

Null olabilirlik yapısal olarak birleşir:

```text
List<User?>   elemanları null olabilen, null olmayan List
List<User>?   elemanları null olmayan, null olabilen List
List<User?>?  elemanları null olabilen, null olabilen List
Pair<String, User?>
```

`null`, yalnızca karşılık gelen `?` çevreleyen türü açık yaptığında
görünebilir.

```ahd
names: List<String?> := [
    "Ali"
    null
    "Ayşe"
]
```

Bir Pair'de `null` anahtara izin verilmez. `T`, `T?`'ye atanabilir; `T?`,
akış analizi anlatımın null olmadığını kanıtlamadıkça `T`'ye atanamaz.

---

## 8. Bildirim ve Atama

### 8.1 Bildirim

Yeni, isimlendirilmiş bir değer şunu kullanır:

```text
name: Type := value
```

Örnek:

```ahd
age: Int := 28
```

`:=` her zaman ilk bildirimi/bağlamayı belirtir.

Bir başlangıç değerinin (initializer) belirsiz olmayan, tam bir statik türü
olduğunda, tür belirtimi (annotation) atlanabilir:

```ahd
age := 28              // Int
user := findStudent()  // O Function'ın dönüş türü olduğunda Student?
```

Bu, dinamik bir değişken değil, statik çıkarımdır (inference). Sonraki
atama hâlâ çıkarılan türle eşleşmelidir. Yalın `value := null` geçersizdir
çünkü hiçbir temel tür sağlamaz; `value: User? := null` yazın. Kapsam
niyeti (scope intent) açık kalır: iç içe geçmiş, çıkarımlı bir bildirim
`name: Local := "Ali"`'dir, yalın, iç içe geçmiş bir `name := "Ali"` ise
§9 altında geçersiz kalır.

### 8.2 Yeniden atama

Mevcut bir bağlama şunu kullanır:

```ahd
age = 29
```

`=` yeni bir değişken bildirmez.

### 8.3 Zincirleme atama yok

Geçersiz:

```ahd
a = b = c = 5
```

Geçersiz:

```ahd
a: Int := b: Int := 5
```

Ayrı ifadeler yazın.

### 8.4 Bileşik atama (Compound assignment)

Desteklenir:

```text
+=
-=
*=
/=
%=
^=
```

`^` üs alma anlamına geldiğinden, `^=` üs alma ataması anlamına gelir.

Bileşik atama, yalnızca operatör sonucu hedefin bildirilen türüne geri
atanabilir olduğunda geçerlidir.

`/` her zaman `Real` ürettiğinden:

- `Real /= Int` ve `Real /= Real` geçerlidir;
- `Int /= Int` ve `Int /= Real` derleme zamanı tür hatalarıdır.

`%` yalnızca `Int % Int` için var olduğundan, `%=` hem bir `Int` hedefi
hem de bir `Int` sağ işleneni gerektirir.

### 8.5 Artırma ve azaltma

Desteklenir:

```ahd
i++
++i
i--
--i
```

`++` ve `--` yalnızca `Int` bağlamaları için tanımlanmıştır ve yalnızca
bağımsız ifadeler olarak görünebilir.

Geçersiz:

```ahd
x: Int := ++i - j++
write(i++)
if ++i > 5 {
}
```

AhdCode'da önek (prefix) ve sonek (postfix) biçimlerinin aynı etkisi
vardır çünkü daha büyük anlatımlar içinde kullanımları yasaktır.

Bir `Real` veya `Int` olmayan herhangi bir değerle her iki biçimi de
kullanmak bir derleme zamanı tür hatasıdır.

---

## 9. Kapsam (Scope): Local, Global ve Gölgeleme (Shadowing)

### 9.1 Parametreler

Function ve structure parametreleri otomatik olarak sözcüksel olarak
yereldir (local). Function parametreleri, bağlama noktasında `Local`
değiştiricisini kullanmaz. Bir structure parametresinde, açık `Local`
ayrı bir Class-düzeni anlamına sahiptir: constructor girdisi sözcüksel
olarak yerel kalır ancak bir örnek özniteliğine dönüşmez. Bir `for`
yineleme değişkeni ve bir `except ... as error` bağlaması da örtük olarak
Local'dir.

### 9.2 Local bildirimleri

Yalnızca doğrudan modül kökü kapsamındaki bir bildirim, bir kapsam
değiştiricisini atlar:

```ahd
count: Int := 0
```

Modül kökünün altında iç içe geçmiş herhangi bir çalıştırılabilir sözcüksel
kapsam içindeki yeni, açık bir bildirim `Local` kullanır. Bu, fonksiyon ve
metot gövdelerini, structure gövdelerini ve modül seviyesindeki veya
çağrılabilir seviyesindeki `if`, döngü, `state`, `attempt`, `except` ve
`ultimately` bloklarını içerir.

```ahd
result: Local Real := 0
```

`for` tarafından tanıtılan örtük bağlama, döngü gövdesine kapsamlıdır.
`except ... as error` tarafından tanıtılan örtük bağlama, o except
gövdesine kapsamlıdır.

İç içe geçmiş bir blok, aynı çağrılabilir içindeki çevreleyen bir sözcüksel
kapsamdan bir bağlamayı `Global` olarak bildirmeden okuyabilir ve
değiştirebilir.

### 9.3 Global erişim

Bir fonksiyon, okuma erişimi dahil, bir global bağlamayı kullanmadan önce
açıkça bildirmelidir.

Örnek:

```ahd
counter: Int := 0

increase: Function := (
) -> Nothing {
    counter: Global Int
    counter++
}
```

Hiçbir gizli global yakalamaya (capture) izin verilmez.

`Global`, özellikle bir fonksiyon, metot veya structure gövdesi içinden
bir modül-kökü bağlamasına atıfta bulunur. Aynı çağrılabilir içindeki
çevreleyen bir Local bağlama için kullanılmaz. Modül seviyesindeki bir
kontrol-akışı bloğunda çalışan kod, modül-kökü bağlamalarına doğrudan
erişebilir ve onları `Global` olarak bildirmez.

### 9.4 Gölgeleme (Shadowing)

İç içe geçmiş kapsamlar arasında gölgelemeye izin verilir.

```ahd
x: Int := 5

if true {
    x: Local Int := 20
    write(x)
}

write(x)
```

Aynı ismi aynı kapsamda `:=` ile yeniden bildirmek bir hatadır.

---

## 10. Constant

`Constant`, ayrı bir tür değil, bir değiştiricidir (modifier).

```ahd
PI: Constant Real := 3.14159
```

Skaler değerler için, yeniden atama yasaktır.

Referans değerleri için, `Constant`, referans verilen nesne grafiğini
derin dondurur (deep-freezes).

```ahd
numbers: Constant List<Int> := [1, 2, 3]
```

Geçersiz:

```ahd
numbers[0] = 50
numbers.add(4)
```

Başka bir alias aynı nesneye atıfta bulunuyorsa, o da dondurulmuş nesneyi
değiştiremez.

Bağımsız bir kopyayı değiştirmek için, `copy` veya `deepCopy` gibi bir
kütüphane işlevselliği aracılığıyla yeni bir kopya açıkça oluşturulmalıdır.

Bir klon/kopya, Constant olarak bildirilmedikçe otomatik olarak Constant
değildir.

### 10.1 Derleme zamanı sabit anlatımları

Bir sabit anlatım (constant expression), derleme zamanında tamamen
değerlendirilebilen skaler bir anlatımdır.

Yalnızca şunları içerebilir:

- skaler literaller;
- parantezler;
- tekli `+`, `-` ve `not`;
- AhdCode'un saf yerleşik skaler operatörleri;
- başlangıç değerleri kendileri de sabit anlatımlar olan skaler `Constant`
  bağlamalarına referanslar.

Fonksiyon çağrıları, değiştirilebilir bağlama referansları, üye veya
indeks erişimi, List/Pair/Class oluşturma ve interpolasyon sabit anlatımlar
değildir.

Constant başlangıç değerlerinin bağımlılık grafiğindeki bir döngü, bir
derleme zamanı hatasıdır.

Bu tanım, dilin derleme zamanı bilgisi gerektirdiği her yerde normatiftir.
Üs alma (power) tiplemesi, sabit-anlatım değerlendirmesine bağlı değildir.

---

## 11. Referans ve Değer Anlambilimi (Semantics)

Değer-benzeri skaler türler:

```text
Int
Real
Bool
String
```

Referans türleri:

```text
List
Pair
Class örnekleri (instances)
```

Örnek:

```ahd
a: List<Int> := [1, 2, 3]
b: List<Int> := a

b[0] = 50
write(a[0])   // 50
```

### 11.1 Bir fonksiyon parametresini yeniden bağlamak

Referans nesneleri paylaşılır, ancak parametre bağlamaları yereldir.

```ahd
change: Function := (
    values: List<Int>
) -> Nothing {
    values = [9, 9, 9]
}
```

Bu, yalnızca yerel bağlama `values`'ı değiştirir.

Ama:

```ahd
values[0] = 99
```

paylaşılan nesneyi değiştirir.

---

## 12. List

`List<T>`, homojen, dinamik olarak boyutlandırılmış, sıralı bir
koleksiyondur.

```ahd
numbers: List<Int> := [1, 2, 3]
```

Tüm null olmayan elemanlar aynı eleman türüne sahip olmalıdır.

Geçersiz:

```ahd
values: List := [1, "Ali"]
```

### 12.1 Tür çıkarımı

Geçerli:

```ahd
numbers: List := [1, 2, 3]
```

Derleyici şunu çıkarır:

```text
List<Int>
```

Eleman türü çıkarılamadığından geçersiz:

```ahd
numbers: List := []
```

Kullanın:

```ahd
numbers: List<Int> := []
```

Yalnızca null değerler içeren bir liste de açık bir eleman türü gerektirir.

### 12.2 İndeksleme

```ahd
numbers[0]
numbers[-1]
```

Negatif indeksleme desteklenir.

Geçersiz bir indeks `IndexError` fırlatır.

### 12.3 Dilimleme (Slicing)

Desteklenir:

```ahd
numbers[1:4]
numbers[:4]
numbers[2:]
numbers[-3:]
```

Dilim-adım (slice-step) sözdizimi v0.1'in bir parçası değildir.

### 12.4 Değişiklik (Mutation)

`List<T>`, tam olarak şu iki eleman ekleme/kaldırma işlemine sahiptir:

```text
add(value: T)      -> Nothing
eject(index: Int)  -> Nothing
```

`add`, sona bir eleman ekler.

```ahd
values: List<Int> := [
    10
    20
]

values.add(30)
```

şunu üretir:

```text
[10, 20, 30]
```

`eject`, bir indeksteki elemanı kaldırır.

```ahd
values: List<Int> := [
    10
    20
    30
]

values.eject(1)
```

şunu üretir:

```text
[10, 30]
```

`eject`, sıradan List indekslemesiyle aynı negatif indekslemeyi kabul
eder, bu yüzden `values.eject(-1)` son elemanı kaldırır. Aralık dışı bir
indeks `IndexError` fırlatır. `eject`, v0.1'de kaldırılan elemanı
döndürmez.

Her iki işlem de yeni bir tane üretmek yerine mevcut List nesnesini
değiştirir, bu yüzden her alias değişikliği görür. İkisi de `Nothing`
döndürür ve bu yüzden değer olarak kullanılamaz. Alıcı `NonNull` olmalıdır,
argüman sıradan eleman atanabilirliğini izler ve bir `Constant` veya başka
şekilde dondurulmuş bir List ikisini de reddeder.

### 12.5 Sıralama

`List<T>`'nin üç sıralama işlemi vardır ve hepsi alıcıyı yerinde yeniden
yazar:

```text
sort()                     -> Nothing
sort(key: Function(T) -> K) -> Nothing
reverse()                  -> Nothing
shuffle()                  -> Nothing
```

`add` ve `eject` gibi, mevcut nesneyi değiştirirler, bu yüzden her alias
yeni sırayı görür, `Nothing` döndürürler ve bir `Constant` veya başka
şekilde dondurulmuş bir List onları reddeder. Hiçbir parametre adı
yayınlamazlar.

`reverse`, mevcut sırayı ters çevirir.

```ahd
values: List<Int> := [1, 2, 3]
alias: List<Int> := values

values.reverse()

write(alias)
```

=>

```text
[3, 2, 1]
```

`shuffle`, önyargısız (unbiased) bir yerinde Fisher–Yates permütasyonu
gerçekleştirir. Son elemandan ikinci elemana doğru yürür ve her `i`
indeksi için o elemanı, kapsayıcı `0..i` aralığından tek biçimli
(uniformly) seçilen bir indeksle değiştirir. §35.3'te açıklanan tam,
paylaşılan, deterministik Math rastgele dizisini kullanır; bu yüzden
`shuffle`, `Math.random` ve `Math.randomInt`, çağrı sırasında tek bir ortak
durumu ilerletir. `shuffle`, `bring Math` gerektirmez, ancak
`Math.seed(...)`, ondan önce paylaşılan diziyi sıfırlamak için
kullanılabilir.

```ahd
bring Math

Math.seed(42)
values: List<Int> := [1, 2, 3, 4, 5]
values.shuffle()

write(values)
```

=>

```text
[2, 3, 1, 5, 4]
```

Boş veya tek elemanlı bir List değişmeden kalır ve hiçbir rastgele
üretici çıktısı tüketmez. `shuffle`, elemanları incelemeden veya
kopyalamadan yeniden düzenler, bu yüzden null olabilen elemanlar sıradan
List anlambilimini korur. Alıcı hâlâ `NonNull` olmalıdır ve bir `Constant`
veya başka şekilde derin dondurulmuş bir List, rastgele durum tüketilmeden
önce değişikliği reddeder.

`sort`'un doğal biçimi artan sırada sıralar ve kararlıdır (stable). Eleman
türü `Int`, `Real` veya `String` olmalıdır; `Bool`, bir `Class`, bir
`Pair` veya iç içe geçmiş bir `List` dahil başka herhangi bir eleman türü,
metne sessiz bir dönüşüm yerine bir derleme zamanı reddidir. Bir `null`
elemanın doğal bir sırası yoktur ve List'i değiştirmeden bırakarak
yakalanabilir `NullError` fırlatır.

```ahd
values: List<Int> := [8, 3, 12, 5]

values.sort()

write(values)
```

=>

```text
[3, 5, 8, 12]
```

Anahtar (key) biçimi, bir elemanın Function'ının sonucuna göre sıralar.
Anahtar türü `K`, tam olarak `Int`, `Real` veya `String` olmalıdır; bir
`Bool` anahtarı reddedilir, `null` döndürebilecek bir anahtar Function da
öyle. v0.1'de bir karşılaştırıcı (comparator) biçimi ve azalan bir
parametre yoktur, çünkü azalan bir sıra olumsuzlanmış veya ters çevrilmiş
bir anahtarla yazılır.

```ahd
gradeOf: Function := (
    student: Student
) -> Int {
    return student.grade
}

students.sort(gradeOf)
```

Anahtar biçimi kararlı ve atomiktir. Her anahtar, alıcı yeniden
yazılmadan önce soldan sağa eleman başına tam olarak bir kez hesaplanır,
bu yüzden fırlatan bir anahtar Function hatasını yayar ve orijinal sırayı
değişmeden bırakır. Alıcı anlatımı tam olarak bir kez değerlendirilir.
Çalışma zamanında `null` döndüren bir anahtar Function `NullError`
fırlatır.

### 12.6 Arama

```text
count(value: T) -> Int
index(value: T) -> Int
```

Her ikisi de saf okumadır (pure reads): asla alıcıyı değiştirmez,
yeniden sıralamaz veya kopyalamaz, bu yüzden `NonNull` bir `Constant`
List geçerli bir alıcıdır. Her ikisi de `same` nesne kimliği yerine
sıradan derin `==` anlambilimiyle karşılaştırır ve her ikisi de `NonNull`
bir argüman gerektirir.

```ahd
values: List<Int> := [5, 7, 5, 9]

write(values.count(5))
write(values.index(5))
```

=>

```text
2
0
```

`index`, ilk eşleşmeyi bildirir ve gösterge bir sonucu yoktur: List'in
içermediği bir değer, `-1` döndürmek yerine yakalanabilir `DomainError`
fırlatır.

### 12.7 map ve filter

```text
map(transform: Function(T) -> U)   -> List<U>
filter(keep: Function(T) -> Bool)  -> List<T>
```

İkisi de yeni, değiştirilebilir bir List oluşturur ve alıcıyı asla
değiştirmez, bu yüzden bir `Constant` List geçerli bir alıcıdır. İkisi
de, işlem başladığında alınan sığ bir anlık görüntüyü (shallow snapshot)
yineler -- `for`'un izlediği aynı kural -- ve callback'i anlık görüntü
elemanı başına soldan sağa tam olarak bir kez çağırır. Bir callback
hatası normal şekilde yayılır.

Bir callback uyumlu bir Function değeridir: sıradan, bildirilmiş bir Function
veya ifade lambda'sı olabilir. Parametre türü, `List` değişmez (invariant)
olduğu için tam olarak eleman türü olmalıdır.

```ahd
double: Function := (
    x: Int
) -> Int {
    return x * 2
}

numbers: List<Int> := [1, 2, 3]
doubled: List<Int> := numbers.map(double)
squared: List<Int> := numbers.map(lambda (x: Int) -> x^2)
```

=>

```text
[2, 4, 6]
```

`map`, eleman türünü değiştirebilir:

```ahd
describe: Function := (
    x: Int
) -> String {
    return "Sayi: {x}"
}

texts: List<String> := numbers.map(describe)
```

Eşlenmiş bir Function bir değer döndürmelidir; bir `Nothing` sonucu
reddedilir.

`filter`, gerçek bir `Bool` yüklem (predicate) gerektirir. AhdCode'da
truthiness olmadığından, `Bool` olmayan bir sonuç bir derleme zamanı
reddi ve çalışma zamanında bir `null` sonuç `NullError` fırlatır.

```ahd
isEven: Function := (
    x: Int
) -> Bool {
    return x % 2 == 0
}

values: List<Int> := [1, 2, 3, 4]
evens: List<Int> := values.filter(isEven)
```

=>

```text
[2, 4]
```

Null olabilen bir List elemanı (`List<T?>`), sessizce atlanmak yerine
yazıldığı gibi callback'e geçirilir. Callback parametresi null olmadığında,
o eleman sıradan null-güvenliği kurallarıyla reddedilir.

---

## 13. Pair

`Pair<K, V>`, AhdCode'un sıralı, homojen anahtar/değer koleksiyonudur.

Aynı satırdaki genel (generic) tür argümanları virgüllerle ayrılmalıdır.

Geçerli:

```ahd
Pair<String, Int>
```

Geçersiz:

```ahd
Pair<String Int>
```

Bu, AhdCode'un genel ayırıcı kuralını izler: bir virgül aynı satırdaki
öğeleri ayırabilir, bir satır sonu ise çok satırlı yapılardaki öğeleri
ayırabilir. Yalnız başına düz boşluk bir ayırıcı değildir.

Tüm anahtarlar tek bir `K` anahtar türüne sahiptir.
Tüm değerler tek bir `V` değer türüne sahiptir.

```ahd
scores: Pair<String, Int> := {
    "Ali": 85
    "Ayşe": 92
}
```

Pair, heterojen bir nesne/kayıt (record) kabı **değildir**.

### 13.1 İç içe geçmiş Pair

İzin verilir:

```ahd
grades: Pair<String, Pair<String, Real>> := {
    "Ali": {
        "Analysis": 85.5
        "Algebra": 90
    }
}
```

Güvenli olduğunda kısmi çıkarıma izin verilir:

```ahd
data: Pair<String, Pair> := {
    ...
}
```

### 13.2 Boş Pair

Geçersiz:

```ahd
scores: Pair := {}
```

Geçerli:

```ahd
scores: Pair<String, Int> := {}
```

### 13.3 Anahtar türleri

v0.1 Pair anahtarları, şunlar gibi kararlı, basit skaler türleri
kullanabilir:

```text
String
Int
Bool
```

`Real`, sınıf örnekleri ve `null`, v0.1'de Pair anahtarları değildir.

### 13.4 Eksik anahtarlar

Eksik bir anahtara erişmek `KeyError` fırlatır.

### 13.5 Sıralama

Pair, ekleme sırasını (insertion order) korur.

Yeni bir anahtar eklemek, onu sona ekler.

Mevcut bir anahtarı güncellemek onu taşımaz.

Bir anahtarı kaldırıp daha sonra yeniden eklemek, onu yeni bir son girdi
olarak ekler.

### 13.6 Literallerde yinelenen anahtarlar

Tek bir Pair literali içindeki yinelenen anahtarlar derleme zamanı
hatalarıdır.

### 13.7 Değişiklik (Mutation)

`Pair<K, V>`, indeks ataması yoluyla eklenir ve güncellenir ve bir yerleşik
kaldırma işlemine sahiptir:

```text
pair[key] = value
eject(key: K)      -> Nothing
```

```ahd
scores: Pair<String, Int> := {}

scores["Ali"] = 85
scores["Ayşe"] = 92
```

Var olmayan bir anahtarı atamak, sona yeni bir girdi ekler. Zaten var olan
bir anahtarı atamak, onu taşımadan değerini günceller.

`eject`, bir anahtarı ve değerini kaldırır.

```ahd
scores.eject("Ali")
```

şunu üretir:

```text
{"Ayşe": 92}
```

Mevcut olmayan bir anahtarı kaldırmak `KeyError` fırlatır. §13.5 ile
birleştiğinde, yeniden eklenen kaldırılmış bir anahtar yeni bir son girdi
olur:

```ahd
scores["Ali"] = 85
scores["Ayşe"] = 92
scores.eject("Ali")
scores["Ali"] = 100
```

sırayı şöyle bırakır:

```text
Ayşe
Ali
```

`pair.add` yoktur. `eject`, mevcut Pair nesnesini değiştirir, bu yüzden
her alias değişikliği görür; `Nothing` döndürür ve bir değer olarak
kullanılamaz. Alıcı `NonNull` olmalıdır, anahtar §13.3'ün sıradan Pair
anahtarı kurallarını izler ve bir `Constant` veya başka şekilde
dondurulmuş bir Pair, ekleme, güncelleme ve `eject`'i reddeder.

---

## 14. Genel Değişmezlik (Generic Invariance)

Güvenli skaler genişletme:

```text
Int -> Real
```

izin verilir.

Değiştirilebilir genel koleksiyonlar değişmezdir (invariant).

Geçersiz:

```ahd
integers: List<Int> := [1, 2, 3]
reals: List<Real> := integers
```

Aynı şekilde:

```text
Pair<String, Int> -> Pair<String, Real>
```

örtük bir dönüşüm değildir.

Class kalıtım ataması izin verilir.

---

## 15. Fonksiyonlar

Fonksiyon bildirim sözdizimi:

```ahd
square: Function := (
    x: Real
) -> Real {
    return x^2
}
```

Sıradan bir Function bildirimi yalnızca modül kökünde görünebilir. Bir
metot Function bildirimi Class üye kapsamında görünebilir. Çalıştırılabilir
bloklar yeni Function bildirimleri içeremez.

Mevcut, isimlendirilmiş bir Function değeri hâlâ bir Local Function
bağlamasında saklanabilir:

```ahd
operation: Local Function := add
```

v0.1'de iç içe geçmiş bir Function bildirimi yoktur. İfade lambda'ları §50'de
belirtilir ve bu bildirim sözdizimini değiştirmez.

### 15.1 Dönüş davranışı

Bir fonksiyon en fazla bir değer döndürür.

Çoklu dönüş sözdizimi yoktur.

Bir `Nothing` fonksiyonu yalın bir `return` kullanabilir veya basitçe
sonuna ulaşabilir.

`Nothing` olmayan bir fonksiyon, her ulaşılabilir yolda uyumlu bir değer
veya tiplenmiş `null` döndürmelidir.

### 15.2 Varsayılan parametreler

İzin verilir:

```ahd
greet: Function := (
    name: String
    title: String := "Student"
) -> String {
    return "Hello {title} {name}"
}
```

Zorunlu, sıralı bir parametre, varsayılan bir parametreden sonra gelemez.

### 15.3 Sıralı ve isimlendirilmiş argümanlar

Bir çağrı ya tamamen sıralı ya da tamamen isimlendirilmiştir.

Karıştırılamazlar.

İsimlendirilmiş argüman sırası önemsizdir.

İsimlendirilmiş bir argüman, `name: expression` sözdizimini kullanır:

```ahd
createUser(
    name: "Ali"
    age: 25
)
```

Aynı satırdaki isimlendirilmiş argümanlar, genel ayırıcı kuralını izleyerek
virgül kullanır:

```ahd
createUser(name: "Ali", age: 25)
```

Sıralı ve isimlendirilmiş argümanlar karıştırıldığından geçersiz:

```ahd
createUser("Ali", age: 25)
```

### 15.4 Birinci sınıf isimlendirilmiş fonksiyonlar

İsimlendirilmiş fonksiyonlar değişkenlerde saklanabilir ve başka
fonksiyonlara geçirilebilir.

Bir parametre veya bağlama basitçe `Function` olarak tiplenebilir;
programcı `Function<Int -> Int>` gibi herkese açık bir fonksiyon-imza türü
yazmaz.

Ancak `Function`, dinamik bir çağrılabilir kaçış kapısı (escape hatch)
**değildir**.

Her `Function` bağlaması, derleme zamanında semantik denetleyici
tarafından bilinen tam olarak bir somut çağrılabilir imzaya çözülmelidir.

Örnek:

```ahd
calculate: Function := (
    operation: Function
    a: Int
    b: Int
) -> Int {
    return operation(a, b)
}
```

`operation(a, b)`'nin kullanımından ve çevreleyen dönüş türünden,
derleyici şununla uyumlu bir imza çıkarmalıdır:

```text
(Int, Int) -> Int
```

Bu çıkarılan imza, dahili bir derleyici türüdür. Kullanıcının onu yazması
gerekmez.

Birden fazla çağrılabilir imza eşit derecede geçerli kalırsa, derleme bir
belirsizlik (ambiguity) hatasıyla başarısız olur.

Bir güvenli imzayı çıkarmak için yeterli bilgi yoksa, derleme bir
fonksiyon-çıkarımı hatasıyla başarısız olur.

Derleyici, `Function`'ı asla sessizce dinamik olarak çağrılabilir olarak
ele almamalı veya imza doğruluğunu çalışma zamanına ertelememelidir.

### 15.5 İfade lambda'ları

`lambda (<tipli parametreler>) -> <ifade>`, mevcut `Function` türünde isimsiz
bir değer oluşturur. Statik türü ve null durumu, çağrılabilir dönüş
sözleşmesini oluşturan tek bir ifade gövdesi vardır. Yeni bir tür değildir ve
normal isimli Function sözdizimi değişmez. Lexical yakalama sınırlaması dahil
tam v0.1.10 kuralları §50'dedir.

---

## 16. Fonksiyon Aşırı Yükleme (Overloading)

Temel:

```ahd
calculate: Function := (
    x: Int
) -> Int {
    return x^2
}
```

Aşırı yükleme:

```ahd
calculate: Overload Function := (
    x: Real
) -> Real {
    return x^2
}
```

Kurallar:

1. tam tür eşleşmesi tercih edilir;
2. `Int -> Real` gibi güvenli genişletme kullanılabilir;
3. eşit en iyi adaylar => belirsiz aşırı yükleme derleme hatası;
4. dönüş türü tek başına aşırı yüklemeleri ayırt edemez.

---

## 17. Sınıflar (Classes)

Bir Class bildirimi yalnızca modül kökünde görünebilir. Çalıştırılabilir
bloklar ve Class üye kapsamları, iç içe geçmiş Class bildirimleri
içeremez.

Kök sınıf:

```ahd
Person: Class := {
}
```

Eşdeğer:

```ahd
Person: Class<> := {
}
```

İkisi de yerleşik `Object`'ten türer.

Kalıtım:

```ahd
Student: Class<Person> := {
}
```

v0.1'de tek bir doğrudan üst sınıf (superclass) vardır.

### 17.1 structure / Attributes

```ahd
structure: Attributes := (
    name: String
    age: Int
)
```

`Local` olmayan tüm girdiler otomatik olarak örnek öznitelikleri
(instance attributes) haline gelir.

`Local` olmayan bir structure girdisindeki `Constant` ve `Confidential`,
üretilen örnek özniteliğine uygulanır; sözcüksel kapsam değiştiricileri
değildirler. Bir Constant referans özniteliği, constructor onu
başlattığında ulaşılabilir nesne grafiğini derin dondurur. `Global`, bir
structure parametresinde geçerli değildir.

Class metotları şunu kullanır:

```ahd
attribute.name
```

### 17.2 Oluşturma (Construction)

Bir sınıf, fonksiyonlar için kullanılan aynı sıradan çağrılabilir
sözdizimi aracılığıyla oluşturulur. Sınıf adı çağrılan (callee) taraftır
ve argümanlar onun `structure: Attributes` parametrelerine göre kontrol
edilir.

```ahd
student: Student := Student(
    name: "Ali"
    age: 20
)
```

Constructor çağrıları tamamen sıralı veya tamamen isimlendirilmiş
olabilir. Diğer tüm çağrılarla aynı ayırıcı ve karıştırmama kurallarını
izlerler.

### 17.3 Local structure parametreleri

```ahd
User: Class<> := {
    structure: Attributes := (
        username: String
        password: Local String
    ) {
        attribute.passwordHash: Confidential String := hash(password)
    }
}
```

### 17.4 structure return

`structure` içinde `return`'e izin verilmez.

### 17.5 Kalıtsal öznitelikler

```ahd
Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Int
    )
}
```

### 17.6 Üst sınıf metotları

```ahd
SuperClass.describe()
```

doğrudan üst sınıf uygulamasını çağırır.

---

## 18. Override

Kalıtsal bir fonksiyonun kasıtlı olarak değiştirilmesi şunu gerektirir:

```ahd
describe: Override Function := (
) -> String {
    return "Student"
}
```

Yalın, kalıtsal-imza çakışması bir hatadır.

Uyumlu bir kalıtsal hedefi olmayan `Override Function` bir hatadır.

---

## 19. Confidential

v0.1'de yalnızca bir özel görünürlük değiştiricisi vardır:

```text
Confidential
```

Varsayılan herkese açıktır (public).

Confidential bir sınıf üyesi:

- tanımlandığı sınıf içinde erişilebilirdir;
- alt sınıflar için erişilebilirdir;
- sıradan dış nesne erişimi yoluyla erişilebilir değildir.

Modül seviyesinde bir Confidential sınıf/fonksiyon, herkese açık modül
API'sı değildir.

Ayrı `private`, `protected` veya `internal` anahtar kelimeleri yoktur.

---

## 20. Getter/Setter Sözdizimi Yok

AhdCode v0.1'de özel bir Getter/Setter yapısı yoktur.

Sıradan fonksiyonları artı öznitelikleri, `Constant`'ı ve
`Confidential`'ı kullanın.

---

## 21. Koşullar

```ahd
if condition {
}
else if anotherCondition {
}
else {
}
```

Koşullar Bool olmalıdır.

---

## 22. while

Önceden kontrol eden (pre-check) döngü.

```ahd
while count < 10 {
    count++
}
```

---

## 23. until

Sonradan kontrol eden (post-check) döngü.

```ahd
until count == 10 {
    count++
}
```

Anlambilim:

1. gövdeyi çalıştır;
2. koşulu test et;
3. doğruysa dur;
4. aksi halde tekrarla.

Gövde bu yüzden en az bir kez çalışır.

---

## 24. for ve Yineleme

Python benzeri bir yineleme modeli.

List:

```ahd
for number in numbers {
    write(number)
}
```

String:

```ahd
for character in "AhdCode" {
    write(character)
}
```

Pair, anahtarları ekleme sırasında (insertion order) verir.

`between`, `Int` değerleri verir; bkz. §32.

### 24.1 Yineleme bağlaması türü

Yineleme bağlaması açık bir tür taşıyabilir:

```ahd
for value: Int in values {
    write(value)
}
```

Tür belirtimi (annotation) isteğe bağlıdır. Her iki kanonik biçim de
şudur:

```text
for name in iterable
for name: Type in iterable
```

Bağlama her zaman örtük olarak `Local`'dir, bu yüzden bir kapsam
değiştiricisi yazılmaz; `for value: Local Int in ...` geçersizdir.

Açık bir tür, bir derleme zamanı kısıtlamasıdır: yinelenebilirin verdiği
tür olmalıdır. Bir `List<String>`'i `for value: Int` olarak yinelemek bir
derleme zamanı hatasıdır ve ilgisiz hiçbir eleman türü sessizce
dönüştürülmez.

### 24.2 Anlık görüntü yinelemesi (Snapshot iteration)

Döngü başında, AhdCode, yinelenebilirin yineleme görünümünün sığ bir
anlık görüntüsünü (shallow snapshot) alır.

Döngü sırasındaki yapısal değişiklik, aktif döngünün ziyaret ettiği şeyi
değiştirmez.

Referans verilen sınıf nesneleri, derin kopyalar değil, paylaşılan
referanslar olarak kalır.

---

## 25. break ve continue

`for`, `while` ve `until`'da desteklenir.

Yalnızca en yakın çevreleyen döngüyü etkilerler.

Etiket (label) yoktur.

---

## 26. state / condition

```ahd
state status {
    condition "active" {
        write("Active")
    }

    condition "blocked" {
        write("Blocked")
    }

    condition default {
        write("Unknown")
    }
}
```

Sonraki koşula düşme (fall-through) yoktur.

`break` gerekmez.

---

## 27. Hata Yönetimi

Anahtar kelimeler:

```text
attempt
except
ultimately
toss
```

Örnek:

```ahd
attempt {
    result: Local Real := divide(10, 0)
}
except DivisionByZeroError as error {
    write(error.message)
}
ultimately {
    write("Completed")
}
```

`toss` bir Error fırlatır.

AhdCode yerleşik `Error`'ı sağlar.

Özel hatalar normal kalıtımı kullanır:

```ahd
InvalidAgeError: Class<Error> := {
    structure: Attributes := (
        message: String
    )
}
```

`ultimately`'nin başarı, işlenmiş hata, yayılan hata durumlarında ve
bekleyen bir return tamamlanmadan önce çalışacağı garanti edilir.

`ultimately` yeni bir hata fırlatırsa, yeni hata aktif hale gelir.

---

## 28. Operatörler

Aritmetik:

```text
+ - * / % ^
```

`^`, sağa-birleşimli (right-associative) üs almadır.

Sayısal sonuç kuralları şunları içerir:

| Sol | Operatör | Sağ | Sonuç |
|---|---|---|---|
| `Int` | `/` | `Int` veya `Real` | `Real` |
| `Real` | `/` | `Int` veya `Real` | `Real` |
| `Int` | `%` | `Int` | `Int` |
| `Int` | `^` | `Int` | `Int` |
| `Real` | `^` | `Int` | `Real` |
| `Int` | `^` | `Real` | `Real` |
| `Real` | `^` | `Real` | `Real` |

Başka hiçbir `%` işlenen kombinasyonu geçerli değildir. `/`, `Int / Int`
dahil, her zaman `Real` üretir.

`Int % Int`, kesilmiş-bölme (truncated-division), bölünenin işaretini
taşıyan kalan anlambilimini kullanır. Bölüm (quotient), taban (floor)
yerine sıfıra doğru kesilir. Sonuç olarak `-5 % 2` `-1`, `5 % -2` `1` ve
`-5 % -2` `-1`'dir. Sıfır olmayan bir bölen için `a = trunc(a / b) * b +
(a % b)` ile tutarlıdır. Bu, Python'un taban-mod (floor-mod) anlambilimi
değildir. Sıfır bir bölen, yakalanabilir `DivisionByZeroError` fırlatır.

Üs alma sonuç türleri yalnızca işlenen türlerine bağlıdır; Constant
durumu, derleme-zamanı değerlendirmesi, üs işareti ve isteğe bağlı
optimize edici/aralık analizi sonuç türünü etkilemez. `Int ^ Int`,
denetimli (checked) Int aritmetiği kullanır. Negatif bir Int üssü,
değerlendirme sırasında `DomainError` fırlatır ve işaretli 64-bit Int
aralığının dışında bir sonuç `OverflowError` fırlatır. `Real`'e
dönüştürülmez.

`Int ^= Int`, her Int sağ işleneni için geçerlidir. Negatif bir üs
`DomainError` fırlatır ve taşma, değerlendirme sırasında `OverflowError`
fırlatır. `Int ^= Real` geçersizdir çünkü işlem, bir Int hedefe örtük
olarak atanamayan `Real` üretir. `Real ^= Int` ve `Real ^= Real`
geçerlidir.

Diğer yerleşik anlamlar şunları içerir:

- operatörün izin verdiği yerde güvenli `Int -> Real` ile sayısal
  aritmetik;
- `String + String`;
- uyumlu, aynı eleman türü için `List<T> + List<T>`;
- `String * Int`.

Örtük String-sayı zorlaması (coercion) yoktur.

v0.1.8, bir Class'ın kendi örnekleri için bu operatörleri tanımlamasına
izin veren on tane Class Protocol Method'dan (`CAdd`, `CSubtract`,
`CMultiply`, `CDivide`, `CRemainder`, `CPower` ve `CEqual`/`CCompare`/
`CNegate`/`CStr` karşılıkları) oluşan küçük, kapalı bir küme ekler. Bu,
genel/sınırsız operatör aşırı yüklemesi değildir: yalnızca bu on kesin,
ayrılmış isim protokol anlamı taşır, ve yalnızca bir Class metot konumunu
işgal ettiklerinde. Tam spesifikasyon için §47'ye bakın.

---

## 29. Eşitlik ve Tür Operatörleri

### == / !=

- skaler: değer eşitliği;
- `5 == 5.0`, sayısal uyumluluk yoluyla true'dur;
- List/Pair: derin değer eşitliği;
- Class: nesne/referans kimliği, **Class**, `CEqual` Class Protocol
  Method'unu (§47) sağlamadıkça, sağladığı durumda `a == b`,
  `a.CEqual(b)`'yi çağırır ve `a != b` her zaman onun mantıksal
  olumsuzlamasıdır. `CEqual`'i olmayan bir Class, yukarıdaki yalın
  referans-kimliği kuralını korur.

### same

Katı tür + değer/durum. `same`, `CEqual`'den hiçbir zaman etkilenmez:
her zaman ham (raw) kimlik testidir, herhangi bir Class Protocol
Method'dan bağımsızdır.

```ahd
5 same 5       // true
5 same 5.0     // false
```

Sınıf için: tam çalışma zamanı türü + aynı örnek.

`List` ve `Pair` için, `==`/`!=` derin değer karşılaştırması yaparken,
`same` nesne/referans kimlik karşılaştırması yapar. Eşit içeriğe sahip
iki farklı koleksiyon `==` ile eşit karşılaştırılır ancak `same` ile
false verir; aynı koleksiyonun alias'ları `same` ile true karşılaştırılır.

### is / is not

Kalıtım dahil tür üyeliği.

### in / not in

- List => değer üyeliği;
- String => alt metin (substring) üyeliği;
- Pair => anahtar üyeliği.

### has / has not

Yalnızca Class/nesne üye varlığı (member existence).

Pair için kullanılmaz.

Sol işlenen, standart null-güvenliği kurallarına göre null olmayan bir
Class nesnesi olmalıdır.

`has`, yazıldığı anlatımın statik türünü değil, nesnenin **gerçek
çalışma zamanı Class'ını ve kalıtım zincirini** inceler. Bir üst tür
değişkeninde tutulan bir örnek, bu yüzden hâlâ gerçek Class'ının
bildirdiği üyeleri bildirir.

```ahd
person: Person := Student(name: "Ali", number: 42)

write(person has name)     // kalıtsal öznitelik
write(person has describe) // kalıtsal metot
write(person has number)   // çalışma zamanı Class'ının özniteliği
write(person has study)    // çalışma zamanı Class'ının metodu
write(person has nickname) // böyle bir üye yok
```

=>

```text
true
true
true
true
false
```

Öznitelikler ve metotlar her ikisi de üyelerdir, override edilmiş bir
metot her iki Class'ın da üyesidir ve sıradan bir üst sınıf örneği asla
bir alt sınıf üyesi kazanmaz. Sağ işlenen, tırnaksız bir üye
belirleyicisidir: bir üyeyi isimlendirir, asla bir bağlama olarak
değerlendirilmez ve hiçbir şey çalıştırmaz. Sol anlatım tam olarak bir
kez değerlendirilir. `has not`, aynı aramanın tam mantıksal
olumsuzlamasıdır.

`has`, erişim izni değil **üye varlığını** kontrol eder. Confidential bir
üye var sayılır, bu yüzden `object has secret`, üye varsa `true`
döndürür. Bu, erişim kurallarını **atlamaz**; `object.secret`, normal
`Confidential` sözleşmesine göre kısıtlı kalır.

---

## 30. Boolean Operatörler ve Kısa Devre

```text
and
or
not
```

Yalnızca Bool.

`and` ve `or` kısa devre yapar.

`not x == 5` şu anlama gelir:

```ahd
not (x == 5)
```

---

## 31. Operatör Önceliği

Kavramsal olarak yüksekten alçağa:

1. çağrı/grup/indeks/üye
2. `^` sağa-birleşimli
3. tekli sayısal işaretler
4. `* / %`
5. `+ -`
6. `< <= > >=`
7. `== != same is/is not in/not in has/has not`
8. ortaya çıkan Boolean karşılaştırma/anlatım üzerinde `not`
9. `and`
10. `or`

`++`/`--` bağımsız ifadelerdir.

---

## 32. between

`between`, v0.1'de kullanılabilir. Önceden bildirilmiştir ve `bring`
gerektirmez.

Python benzeri tam sayı aralık anlambilimine sahiptir ve bir ila üç
`Int` argümanı kabul eder:

```text
between(stop)
between(start, stop)
between(start, stop, step)
```

`start` varsayılan olarak `0`'dır ve `step` varsayılan olarak `1`'dir.
Stop her zaman dışlanır.

```ahd
for value in between(5) {
    write(value)
}
```

=> `0 1 2 3 4`

```ahd
for value in between(1, 5) {
    write(value)
}
```

=> `1 2 3 4`

```ahd
for value in between(0, 10, 2) {
    write(value)
}
```

=> `0 2 4 6 8`

Negatif bir adım geriye doğru sayar:

```ahd
for value in between(5, 0, -1) {
    write(value)
}
```

=> `5 4 3 2 1`

Adım stop'a ulaşamadığında, yineleme boştur. Hem `between(0, 5, -1)` hem
de `between(5, 0, 1)` hiçbir şey vermez.

Sıfır bir adım ilerleme kaydetmez ve yakalanabilir bir `DomainError`
fırlatır. Asla sessizce `1` adımı olarak ele alınmaz.

### 32.1 Tembel (Lazy) yineleme

`between` bir `List` üretmez. Tüm durumu mevcut değeri, stop'u ve adımı
olan tembel bir yinelemedir, bu yüzden herhangi bir aralığı yinelemek,
kaç değer verdiğinden bağımsız olarak sabit bellek kullanır:

```ahd
for value in between(1, 10000000) {
    write(value)
}
```

hiçbir koleksiyon tahsis etmez. Her değer talep üzerine (on demand)
hesaplanır, bu yüzden `break` hemen durur ve `continue`, kalanı
somutlaştırmadan (materializing) bir sonraki değere ilerler. Destekleyen
bir koleksiyon olmadığından, §24.2'nin sığ anlık görüntü kuralı ona
uygulanmaz.

Yineleme, işaretli 64-bit `Int` aralığından çıkacak herhangi bir adımdan
önce durur, bu yüzden `Int` sınırlarına yakın bir aralık sarmalamak
yerine sona erer.

`between` değerinin tek herkese açık sözleşmesi, onu yinelemenin `Int`
verdiğidir. v0.1, onun için hiçbir tür sözdizimi ve başka hiçbir işlem
tanımlamaz: indekslenemez, dilimlenemez, değiştirilemez, temizlenemez,
bir `List`'e dönüştürülemez veya `str` ile işlenemez.

---

## 33. bring

AhdCode, `import` değil `bring` kullanır.

### 33.1 v0.1 modül-adı çözümlemesi

Bir v0.1 modül referansı, `bring` dilbilgisi tarafından zaten kabul
edilen, büyük/küçük harfe duyarlı tek bir tanımlayıcıdır. Yerel bir modül
için, `ModuleName`, içe aktaran kaynak dosyasını içeren dizindeki
`ModuleName.ahd` dosyasına çözülür. Tanımlayıcı normalleştirmesi,
çözümlemeden önce sıradan AhdCode NFKC tanımlayıcı kuralını izler.

v0.1'de noktalı modül yolları, göreli-yol sözdizimi, paket-kökü araması,
yapılandırılabilir kaynak arama yolu veya örtük dizin-modülü kuralı
yoktur. Bu özellikler, uygulamaya özgü bir arama sezgisi (heuristic)
yerine daha sonraki, açık bir dil revizyonu gerektirir.

Derleyici tarafından kayıtlı yerleşik modül isimleri, yerleşik modül
kayıt defteri (registry) aracılığıyla çözülür. Yerel bir dosya, kayıtlı
bir yerleşik modül adının yerini alamaz (shadow edemez). Modül-adı ve
dosya adı eşleştirmesi büyük/küçük harfe duyarlı kalır.

```ahd
bring Mathematics
```

Bu, modülü bir isim uzayı (namespace) olarak içe aktarır. Herkese açık
üyelerine bu isim uzayı aracılığıyla erişilir:

```ahd
result: Real := Mathematics.sqrt(25)
```

```ahd
from Utilities bring all
```

```ahd
from Mathematics bring (
    sqrt
    sin
    cos
)
```

`from Mathematics bring sqrt`, seçilen herkese açık sembolü doğrudan içe
aktarır, bu yüzden `Mathematics.sqrt(...)` yerine `sqrt(...)` olarak
çağrılır. Yukarıdaki çok satırlı biçim, aynı kuralı listelenen her
sembole uygular. `bring all`, `Confidential` olmayan her herkese açık
sembolü doğrudan içe aktarır.

Döngüsel bring'ler derleme zamanı hatalarıdır.

bring tarafından tanıtılan isim çakışmaları derleme zamanı hatalarıdır.

Confidential modül-seviyesi semboller dışarıdan bring edilemez.

### 33.2 Modül takma adları (Aliases)

Bir isim uzayı içe aktarımı, modülü farklı bir isim altında
bağlayabilir:

```ahd
bring Time as T
bring Math as M
bring Engine as E
```

Takma ad geçerli bir tanımlayıcı olmalıdır ve büyük/küçük harfe
duyarlıdır. `bring Time as T`, `T`'yi bağlar ve `Time`'ı **bağlamaz**;
her iki ismi de içe aktarmak iki `bring` ifadesi gerektirir. Bir takma
ad, sıradan bağlama kurallarına katılır, bu yüzden zaten kullanımda olan
bir isim, olağan içe aktarım çakışması tanılamasıdır.

Takma adlar, derleyici tarafından sağlanan standart modüllere ve kaynak
modüllere tekdüze (uniformly) olarak uygulanır. Diğer biçimler
değişmez:

```ahd
bring Module
from Module bring name
from Module bring (
    first
    second
)
from Module bring all
```

v0.1'de sembol takma adı (`from Time bring DateTime as DT`) ve isim
uzayı nitelikli (namespace-qualified) tür sözdizimi yoktur. Bir tür,
isimlendirilmeden önce hâlâ içe aktarılır:

```ahd
bring Time as T
from Time bring DateTime

current: DateTime := T.now()
```

Kanonik biçimlendirme `bring Time as T`'dir.


---

## 34. Fundamentals

Aşağıda mevcut olarak listelenen fonksiyonlar, her modülde önceden
bildirilmiştir ve `bring` gerektirmez.

Çekirdek terminal G/Ç:

```text
write
take
```

Mevcut Fundamentals fonksiyonları:

```text
str
int
real
len
clear
between
abs
sum
min
max
```

Planlanan erken Fundamentals fonksiyonları şunları içerir:

```text
bool
swap
combine
merge
jump
copy
deepCopy
```

### 34.1 Kanonik str anlambilimi

`str`, yerelden bağımsız (locale-independent) ve deterministiktir.
v0.1'de kullanıcı-tanımlı veya özel bir `str` geçersiz kılma (override)
mekanizması yoktur.

Skaler ve null dönüşümü:

| Girdi | Sonuç |
|---|---|
| `String` | String değerinin kendisi |
| `Int` | taban-10 ondalık metin |
| `Real` | en kısa, tur-güvenli (round-trip) ondalık metin; tam sayı bir Real `.0`'ı korur |
| `Bool` | `"true"` veya `"false"` |
| `null` | `"null"` |

Real ondalık ayırıcısı her zaman `.`'dır. Kanonik bilimsel gösterim (scientific
notation), gerektiğinde kullanılabilir ve küçük harf `e` kullanır. Negatif
sıfır korunur.

Örnekler:

```text
str(5)     -> "5"
str(5.0)   -> "5.0"
str(-0.0)  -> "-0.0"
str(true)  -> "true"
str(null)  -> "null"
```

Koleksiyonlar, kanonik, literal-benzeri bir gösterim kullanır:

```text
str([1, 2, 3])
-> "[1, 2, 3]"

str(["Ali", "Ayşe"])
-> "[\"Ali\", \"Ayşe\"]"

str({"Ali": 90, "Ayşe": 95})
-> "{\"Ali\": 90, \"Ayşe\": 95}"
```

List sırası ve Pair ekleme sırası korunur. İç içe geçmiş değerler, aynı
gösterimi özyinelemeli (recursively) olarak kullanır. Bir koleksiyonun
içine yerleştirilmiş bir String, çift tırnaklarla çevrilir ve tam AhdCode
string kaçış kurallarıyla kaçışa (escape) tabi tutulur.

Varsayılan Class örnek gösterimi `<ClassName>`'dır. Örneğin, bir Student
örneği şuna dönüşür:

```text
<Student>
```

Öznitelikler otomatik olarak yazdırılmaz. Bu, Confidential üye ifşasını
(disclosure) önler ve özyinelemeli nesne-grafiği gezinmesinden (traversal)
kaçınır.

İsimlendirilmiş bir Function değeri şöyle gösterilir:

```text
<Function functionName>
```

`Nothing`, `str` tarafından kabul edilmez.

### 34.2 len

`len`, boyutlandırılabilir bir değerin boyutunu bildirir.

```text
len(String)     -> Int
len(List<T>)    -> Int
len(Pair<K,V>)  -> Int
```

`len(String)`, byte değil, karakter sayar. `len(List<T>)`, elemanları
sayar ve `len(Pair<K,V>)`, girdileri sayar.

`len`, skaler sayısal türleri, `Bool`'u, `Class` örneklerini, `Function`
değerlerini veya `Nothing`'i kabul etmez. Null olabilen bir değer, `len`
uygulanmadan önce `NonNull` olmalıdır.

```ahd
write(len("añb"))
```

=> `3`

### 34.3 clear

`clear`, bir koleksiyonu yerinde boşaltır.

```text
clear(List<T>)    -> Nothing
clear(Pair<K,V>)  -> Nothing
```

`clear`, yeni bir nesne oluşturmaz. Nesne kimliği değişmez, bu yüzden
koleksiyonun her alias'ı boşaltılmış durumu görür.

```ahd
a: List<Int> := [1, 2, 3]
b: List<Int> := a

clear(a)

write(len(b))
```

=> `0`

Aynı referans anlambilimi `Pair` için de geçerlidir:

```ahd
scores: Pair<String, Int> := {
    "Ali": 85
}

alias: Pair<String, Int> := scores

clear(scores)

write(len(alias))
```

=> `0`

`clear`, `String`'i, skaler türleri, `Class` örneklerini, `Function`
değerlerini, `null`'ı veya `Nothing`'i kabul etmez. String'ler
değiştirilemez olduğundan, boş bir String, `clear` yerine yeniden
bağlama (rebinding) ile üretilir.

Null olabilen bir `List` veya `Pair`, `clear` uygulanmadan önce
`NonNull` olmalıdır.

Doğrudan bilinen bir `Constant` hedefi temizlemek, bir derleme zamanı
değişiklik hatasıdır, çünkü `clear`, koleksiyonu yeniden bağlamak yerine
değiştirir:

```ahd
values: Constant List<Int> := [1, 2, 3]
clear(values)
```

geçersizdir.

`clear`, `Nothing` döndürür, bu yüzden sonucu bağlanamaz veya bir değer
olarak kullanılamaz.

### 34.4 abs

`abs`, sayısal büyüklüktür (magnitude).

```text
abs(Int)   -> Int
abs(Real)  -> Real
```

Sonuç türü tam olarak argüman türüdür, bu yüzden `abs` kendi başına
hiçbir sayısal genişletme (widening) getirmez.

```ahd
write(abs(5))
write(abs(-5))
write(abs(2.5))
write(abs(-2.5))
```

=>

```text
5
5
2.5
2.5
```

`abs`, `String`'i, `Bool`'u, `List`'i, `Pair`'i, `Class` örneklerini,
`Function` değerlerini, `null`'ı veya `Nothing`'i kabul etmez. Örtük bir
`String` dönüşümü yoktur: `abs(int("-5"))` açıkça yazılır. Null olabilen
bir değer, `abs` uygulanmadan önce `NonNull` olmalıdır.

Minimum `Int`'in bir `Int` büyüklüğü yoktur, bu yüzden sarmalamak
yerine sıradan denetimli-aritmetik (checked-arithmetic) sözleşmesini
izler:

```ahd
write(abs(-9223372036854775808))
```

yakalanabilir `OverflowError` fırlatır.

`abs(Real)`, sonlu-`Real` kurallarını korur ve `-0.0` için `0.0` üretir.

### 34.5 sum, min ve max

`sum`, `min` ve `max`, sayısal bir `List`'i indirger (reduces).

```text
sum(List<Int>)   -> Int
sum(List<Real>)  -> Real

min(List<Int>)   -> Int
min(List<Real>)  -> Real

max(List<Int>)   -> Int
max(List<Real>)  -> Real
```

v0.1'de bir vararg biçimi yoktur: her biri tam olarak bir `List` alır.

```ahd
values: List<Int> := [8, 3, 12, 5]

write(sum(values))
write(min(values))
write(max(values))
```

=>

```text
28
3
12
```

```ahd
values: List<Real> := [3.5, -2.0, 8.25]

write(sum(values))
write(min(values))
write(max(values))
```

=>

```text
9.75
-2.0
8.25
```

`List` genel değişmezliği (generic invariance) değişmez: `List<Int>`,
bir `List<Real>` değildir ve hiçbir eleman dönüştürülmez. `List<Bool>`,
`List<String>`, `Pair`, `String` ve skaler değerler reddedilir. `List`
argümanı `NonNull` olmalıdır.

Boş bir `List`'in `sum`'ı, eleman türünün toplamsal özdeşliğidir
(additive identity):

```ahd
ints: List<Int> := []
reals: List<Real> := []

write(sum(ints))
write(sum(reals))
```

=>

```text
0
0.0
```

`min` ve `max`'ın böyle bir özdeşliği yoktur, bu yüzden boş bir `List`
yakalanabilir `DomainError` fırlatır:

```text
DomainError: min requires a non-empty List
DomainError: max requires a non-empty List
```

Bir indirgeme sırasında karşılaşılan bir `null` eleman, sıfır olarak ele
alınmak veya atlanmak yerine yakalanabilir `NullError` fırlatır.

`Int` toplama, denetimli `Int` aritmetiği kullanır, bu yüzden taşan bir
toplam `OverflowError` fırlatır. `Real` toplama, sonlu-`Real`
kurallarını izler ve asla sessizce sonlu olmayan bir toplam üretmez.

Üç indirgeme de saf okumadır. Argümanı değiştirmez, yeniden sıralamaz
veya kopyalamazlar, nesne kimliği değişmez ve bu yüzden `NonNull` bir
`Constant` `List` geçerli bir argümandır:

```ahd
values: Constant List<Int> := [4, 1, 9]

write(sum(values))
write(min(values))
write(max(values))
```

=>

```text
14
1
9
```

Planlanan sonraki Fundamentals/veri-yapısı özellikleri şunları
içerebilir:

```text
FSet
FLinkedList
FStack
FQueue
FDeque
Matrix
Complex
```

### swap

Tuple ataması yoktur.

Kullanın:

```ahd
swap(a, b)
```

### combine

Homojen anahtar List + homojen değer List => Pair.

Uzunluk uyuşmazlığı bir hatadır.

### jump

Çekirdek dilimleme, adım sözdizimine sahip değildir. Adımlı seçim,
`jump` gibi bir kütüphane fonksiyonuna aittir.

---

## 35. Math Standart Modülü

`Math`, derleyici tarafından kayıtlı bir standart modüldür. Fundamentals
gibi önceden bildirilmemiştir ve sıradan modül sözdizimi aracılığıyla içe
aktarılmalıdır:

```ahd
bring Math

write(Math.sqrt(25.0))
write(Math.PI)
```

`from Math bring sqrt` ve mevcut seçici veya `bring all` biçimleri,
kaynak modüllerle aynı modül-arayüzü kurallarıyla çalışır. Kanonik modül
kimliği `builtin:Math`'tır; kardeş bir `Math.ahd` onun yerini alamaz. Her
Math argümanı `NonNull` olmalıdır. `Real` olarak yazılan bir parametre
yalnızca `Real`'i ve mevcut örtük `Int -> Real` genişletmesini kabul
eder -- asla `String`, `Bool` veya yeni bir zorlama (coercion) değil.

Tam v0.1 herkese açık yüzeyi şudur:

```text
PI: Constant Real
E:  Constant Real

round(Real)      -> Real
round(Real, Int) -> Real
floor(Real)      -> Int
ceil(Real)       -> Int

sqrt(Real)  -> Real
sin(Real)   -> Real
cos(Real)   -> Real
tan(Real)   -> Real
log(Real)   -> Real
log10(Real) -> Real
exp(Real)   -> Real

seed(Int)           -> Nothing
random()            -> Real
randomInt(Int, Int) -> Int
```

`PI` ve `E`, değiştirilemez float64 matematiksel sabitlerdir. Sıradan,
okunabilir isim uzayı üyeleridir ancak atanamaz veya güncellenemez.
`abs`, `sum`, `min` ve `max`, Fundamentals'ta kalır; `Math` onları
takma adlandırmaz (alias etmez). `Math.pow` yoktur; üs alma `^` kullanır.

### 35.1 Yuvarlama ve tam sayı sınırları

`round(value)`, tam sayı bir `Real` döndürür. Tam buçuklar sıfırdan
uzağa doğru yuvarlanır:

```text
Math.round(3.4)   -> 3.0
Math.round(3.5)   -> 4.0
Math.round(-3.5)  -> -4.0
```

İki argümanlı biçim, `digits` ondalık basamağa yuvarlar. `digits`,
`0..15` içinde olmalıdır; aksi halde yakalanabilir `DomainError`
fırlatır. Deterministik float64 prosedürü, `10^digits` ile çarpar,
sıfırdan-uzağa-buçuk yuvarlamasını uygular, ardından aynı faktöre böler.
Sonlu bir değeri ölçeklendirmek taşmaya neden olursa, değer değişmeden
döndürülür çünkü float64 aralığı zaten istenen ondalık hassasiyeti
aşmaktadır.

```text
Math.round(3.14159, 2) -> 3.14
Math.round(2.675, 2)   -> 2.68
```

`floor` ve `ceil`, `Int` döndürür ve pozitif ve negatif değerler için
matematiksel tanımlarını izler. İşaretli Int64'ün dışında bir sonuç,
yakalanabilir `OverflowError` fırlatır; asla sarmalanmaz.

### 35.2 Klasik matematik

`sqrt`, asal kareköktür. Negatif bir girdi `DomainError` fırlatır. `sin`,
`cos` ve `tan`, radyan kullanır; v0.1'de derece modu yoktur. `log`, doğal
logaritmadır ve `log10` on tabanındadır. İkisi de sıfırdan kesinlikle
büyük bir değer gerektirir ve aksi halde `DomainError` fırlatır. `exp`,
sonucu sonlu `Real` aralığını aşarsa `OverflowError` fırlatır.

Her Math sonucu, sonlu-`Real` sözleşmesine uyar. Bir çalışma zamanı
işlemi asla `NaN` veya sonsuzluk göstermez: tanımsız sonuçlar
`DomainError` olur ve aralık dışı sonlu matematik `OverflowError` olur.

### 35.3 Sözde rastgele (Pseudo-random) dizi ve açık tohumlama

Yerel program yürütmesi başına bir paylaşılan Math sözde rastgele dizisi
vardır. Her taze yürütmede AhdCode, işletim sisteminin kriptografik
olarak güvenli entropi kaynağından tam olarak sekiz byte okur ve bunları
tek bir küçük-endian (little-endian), işaretsiz 64-bit başlangıç durumu
olarak yorumlar. Bu byte'ları elde edememek bir başlangıç
başarısızlığıdır; çalışma zamanı asla sessizce sabit bir tohum, saat,
süreç kimliği veya saat değeri ile değiştirme yapmaz. İki tohumlanmamış
yürütmenin eşleşmesi gerekmez ve farklı olması da gerekmez, çünkü bir
entropi çakışması teorik olarak hâlâ mümkündür. Başlangıç tohumu, herkese
açık bir API aracılığıyla gösterilmez.

`Math.seed(value)`, paylaşılan durumu açıkça sıfırlar. Her işaretli
Int64 tohumu geçerlidir ve ikiye tümleyen (two's-complement) bit
kalıbıyla dahili `uint64` durumuna eşlenir. Aynı değerle yeniden
tohumlamak, tam olarak aynı diziyi yeniden üretir, bu yüzden açık
tohumlama, tekrarlanabilir testler ve simülasyonlar için desteklenen
mekanizmadır. Farklı modülleri başlatırken ve giriş modülünde yapılan
çağrılar, gerçek çalışma zamanı sırasında aynı diziyi ilerletir.

v0.1, SplitMix64'ü sabitler. Durum `s` için, bir çıktı, tam olarak şu
işaretsiz 64-bit sarmalama (wrapping) işlemlerini gerçekleştirir:

```text
s = s + 0x9e3779b97f4a7c15
z = s
z = (z xor (z >> 30)) * 0xbf58476d1ce4e5b9
z = (z xor (z >> 27)) * 0x94d049bb133111eb
z = z xor (z >> 31)
```

Bu algoritma ve her açıkça tohumlanmış dizi, Go sürümleri, işletim
sistemleri ve desteklenen mimariler genelinde v0.1 tekrarlanabilirlik
sözleşmesinin bir parçasıdır. OS entropisi yalnızca durumu başlatır;
`Math.random`, `Math.randomInt` ve `List.shuffle`, sözde rastgele kalır
ve kriptografik olarak güvenli değildir.

`random()` bir kez ilerler ve yüksek 53 çıktı bitinden `0.0 <= result <
1.0` içinde bir `Real` oluşturur:

```text
float64(z >> 11) * 2^-53
```

Tohum `557` için ilk değerler şu şekilde sabitlenmiştir:

```text
0.4121990632081577
0.4686510900868295
0.5840201876345011
```

`randomInt(min, max)`, kapsayıcı (inclusive) sınırlar kullanır. `min >
max`, `DomainError` fırlatır. `min == max` olduğunda, üretici durumunu
tüketmeden o değeri döndürür. Diğer her aralık için, işaretsiz aralık
genişliğine göre mod almadan önce reddetme örneklemesi (rejection
sampling) kullanır, bu yüzden sonucun modulo yanlılığı (bias) yoktur.
Genişlik aritmetiği işaretsizdir ve sıfırı ve her iki Int64 sınırını
geçen aralıkları destekler. Sıfır genişlik, tam `2^64`-değerli Int64
alanını belirtir ve bir ham çıktıyı o tam sıralı aralığa eşler.

---

## 36. Time Standart Modülü

`Time`, tam olarak `Math` gibi sıradan modül sözdizimi aracılığıyla içe
aktarılan, derleyici tarafından kayıtlı bir standart modüldür:

```ahd
bring Time
from Time bring DateTime
from Time bring Duration
```

Kanonik modül kimliği `builtin:Time`'dır; kardeş bir `Time.ahd` onun
yerini alamaz. AhdCode'da isim uzayı nitelikli tür sözdizimi olmadığından,
bir Time türü, onu içe aktardıktan sonra `DateTime` olarak yazılır, asla
`Time.DateTime` olarak değil. Her Time argümanı `NonNull` olmalıdır.

Tam herkese açık yüzey şudur:

```text
Time.now()                        -> DateTime
Time.utc()                        -> DateTime
Time.timestamp()                  -> Int
Time.fromTimestamp(milliseconds: Int) -> DateTime
Time.monotonic()                  -> Real
Time.sleep(milliseconds: Int)     -> Nothing
Time.duration(milliseconds: Int)  -> Duration
Time.between(first: DateTime, second: DateTime) -> Duration
Time.dateTime(
    year: Int,
    month: Int,
    day: Int,
    hour: Int = 0,
    minute: Int = 0,
    second: Int = 0,
    millisecond: Int = 0
) -> DateTime
Time.dateTimeUTC(year, month, day, hour = 0, minute = 0, second = 0, millisecond = 0) -> DateTime
Time.dateTimeOffset(year, month, day, offsetMinutes, hour = 0, minute = 0, second = 0, millisecond = 0) -> DateTime
```

### 36.1 Yerel, UTC, sabit ofset ve timestamp

`Time.now()`, ana bilgisayarın (host) **yerel** tarih ve saatini bildirir
ve `Time.dateTime`, yerel bir sivil an oluşturur. `Time.utc()` ve
`Time.dateTimeUTC` UTC kullanır. `Time.dateTimeOffset`, -840 ile 840 arasında
tam dakikalık sabit ofset kullanır. Unix timestamp,
`1970-01-01 00:00:00 UTC`'den itibaren işaretli milisaniyedir;
`Time.timestamp()` güncel değeri okur, `Time.fromTimestamp` UTC görünümünü
döndürür. DateTime yılı 1..9999 içinde temsil edilebildiğinde negatif
timestamp geçerlidir. v0.1.11'de adlandırılmış/IANA saat dilimi veritabanı yoktur.
Yayınlanan ofset gösterimi dakika hassasiyetindedir: `offsetMinutes` her zaman
tam dakikadır ve AhdCode kaynağının adlandırabildiği her ofset tam dakikadır.
Birkaç tarihsel host-yerel bölge, saniye içeren bir ofsette bulunur
(`Europe/Istanbul` 1880 öncesinde `+01:55:52`'dir). Böyle bir an yine de tam
olarak temsil edilir: `offsetMinutes` tam dakika kısmını bildirir, artan
saniyeler ise çalışma zamanı gösterimi olarak korunur, bu yüzden an ne kırpılır
ne de kayar. Bu artık yayınlanan bir öznitelik değildir -- okunamaz ve `has`
onu bildirmez -- bu yüzden DateTime öznitelik yüzeyi tam olarak §36.2'deki
dokuz isimden ibaret kalır.

### 36.2 DateTime

`DateTime`, dokuz salt okunur `Int` özniteliği gösterir:

```text
year  month  day  hour  minute  second  millisecond  weekday  offsetMinutes
```

`weekday`, günleri Pazartesi'den başlayarak numaralandırır:

| gün | değer |
|---|---|
| Pazartesi | 1 |
| Salı | 2 |
| Çarşamba | 3 |
| Perşembe | 4 |
| Cuma | 5 |
| Cumartesi | 6 |
| Pazar | 7 |

Her öznitelik `Constant`'tır, bu yüzden birini atamak sıradan Constant
tanılamasıdır:

```ahd
current: DateTime := Time.now()

current.year = 2030
```

geçersizdir.

`offsetMinutes`, UTC'nin doğusundaki sabit ofsettir. `DateTime`, sekiz üye yayınlar:

```text
before(other: DateTime)     -> Bool
after(other: DateTime)      -> Bool
sameMoment(other: DateTime) -> Bool
timestamp()                 -> Int
toUTC()                     -> DateTime
toLocal()                   -> DateTime
toOffset(offsetMinutes: Int) -> DateTime
toString()                  -> String
```

`toString`, deterministiktir, yerelden bağımsızdır ve asla bir saat
dilimini isimlendirmez:

```text
YYYY-MM-DD HH:MM:SS
```

Milisaniyeler, metin yerine `millisecond` özniteliği üzerinden okunur.
`str(value)`, bir `DateTime`'ı `<DateTime>` olarak işler, çünkü §34.1
kasıtlı olarak Class özniteliklerini yazdırmaz.

`DateTime`, `CCompare`'i (§47) uygulamaz, bu yüzden `<` ve `>` ona
uygulanmaz. Sıralama `before` ve `after` ile yazılır. `DateTime`, `CEqual`'i
de uygulamaz, bu yüzden `==` ve `same`, §29'un sıradan Class kuralını --
nesne kimliğini -- izler ve ayrı ayrı oluşturulmuş eşit anlar `==`
**değildir**. Değer karşılaştırması `sameMoment`'tır ve iki `Duration`
değeri `milliseconds` üzerinden karşılaştırılır.

`timestamp`, `toUTC`, `toLocal` ve `toOffset` temsil edilen anı korur.
`before`, `after`, `sameMoment` ve `Time.between`, ofsetler farklı olsa bile
görüntülenen sivil alanları değil anları karşılaştırır.

### 36.3 Bir DateTime oluşturmak

`Time.dateTime`, `Time.dateTimeUTC` ve `Time.dateTimeOffset`, her bileşeni Gregoryen takvime göre doğrular ve
imkânsız bir anı sarmalamak yerine `ValueError` fırlatır:

```text
year         1..9999
month        1..12
day          1..(o yılın o ayının uzunluğu)
hour         0..23
minute       0..59
second       0..59
millisecond  0..999
offsetMinutes -840..840 (sabit-ofset kurucusu ve dönüşümü)
```

```ahd
Time.dateTime(year: 2028, month: 2, day: 29)
```

geçerlidir, `2026-02-29`, `2026-02-30`, ay `13` ve saat `25` ise hepsi
`ValueError`'dır. `DateTime` ve `Duration` hiçbir zaman doğrudan
oluşturulmaz; yalnızca önce doğrulama yapan Time fonksiyonlarından gelir.

### 36.4 Duration

`Duration`, bir takvim tarihi değil, geçen zamandır. İki salt okunur
öznitelik gösterir:

```text
milliseconds Int
seconds      Real
```

```ahd
wait: Duration := Time.duration(milliseconds: 1500)

write(wait.milliseconds)
write(wait.seconds)
```

=>

```text
1500
1.5
```

Bir `Duration` negatif olabilir, çünkü işaretli bir fark kullanışlıdır.
Negatif bir değer, büyüklüğüne dönüştürülmek yerine korunur.

### 36.5 İki an arasındaki fark

`Time.between(first, second)`, `second - first`'tir:

```ahd
a: DateTime := Time.dateTime(year: 2026, month: 1, day: 1)
b: DateTime := Time.dateTime(year: 2026, month: 1, day: 2)

write(Time.between(a, b).milliseconds)
write(Time.between(b, a).milliseconds)
```

=>

```text
86400000
-86400000
```

Her iki tarafta da aynı an, sıfır bir `Duration` üretir.

### 36.6 Calendar

`Time.Calendar`, bir `DateTime`'a ihtiyaç duymadan Gregoryen takvimin
kendisi hakkındaki soruları yanıtlar:

```text
Time.Calendar.isLeapYear(year: Int)                     -> Bool
Time.Calendar.daysInMonth(year: Int, month: Int)        -> Int
Time.Calendar.weekday(year: Int, month: Int, day: Int)  -> Int
```

Bir artık yıl 4'e bölünür, 400'e bölünmesi gereken bir yüzyıl yılı
hariç: `2028` ve `2000` artık yıllardır, `2100` ve `1900` değildir.
`weekday`, `weekday` özniteliğiyle aynı Pazartesi=1..Pazar=7
numaralandırmasını kullanır. Geçersiz bir yıl, ay veya tarih
`ValueError` fırlatır.

`Calendar` salt okunurdur ve oluşturulmaz. v0.1'de ay veya gün isimleri,
yerelleştirme veya takvim işleme yoktur.

### 36.7 Geçen zaman ve bekleme

```ahd
start: Real := Time.monotonic()

Time.sleep(100)

elapsed: Real := Time.monotonic() - start
```

`Time.monotonic()`, asla geriye doğru hareket etmeyen bir saat üzerinde
bir **saniye** okumasıdır. Yalnızca farklar anlamlıdır; mutlak değerin
hiçbir takvim anlamı yoktur.

`Time.sleep`, **milisaniye** alır. Sıfır hemen döner ve negatif bir
istek, sıfıra kırpılmak yerine `ValueError` fırlatır.

### 36.8 Bu sürümde olmayanlar

`Time`; kasıtlı olarak adlandırılmış/IANA bölgelere, DST yapılandırma
nesnelerine, biçim dizelerine (format strings), `parse`'a, ISO-8601 veya RFC
3339 okuyucusuna, ay/gün isimlerine ve doğal dil tarihlerine sahip değildir.
Mevcut `toString()` çıktısı `YYYY-MM-DD HH:MM:SS` olarak kalır ve ofset eklemez.

---

## 37. Latex Standart Modülü

`Latex`, tam olarak `Math` ve `Time` gibi, bir takma adla (alias) birlikte
içe aktarılan, derleyici tarafından kayıtlı bir standart modüldür:

```ahd
bring Latex as L
from Latex bring LatexError
```

Kanonik kimlik `builtin:Latex`'tır. Her argüman `NonNull` olmalıdır.

Tam herkese açık yüzey şudur:

```text
Latex.pdf(source: String, output: String)     -> Nothing
Latex.pdfFile(input: String, output: String)  -> Nothing
Latex.escape(text: String)                    -> String
Latex.section(title: String)                  -> String
Latex.subsection(title: String)               -> String
Latex.equation(source: String)                -> String
Latex.document(body: String, title: String = "", author: String = "") -> String
Latex.table(headers: List<String>, rows: List<List<String>>) -> String

LatexError
```

### 37.1 Metin yardımcıları

`escape`, TeX'e özel karakterler `\ { } $ & # % _ ^ ~` için metin-bağlamı
kaçış işlemidir (escaping). Ham matematiği temizlediğini (sanitize) iddia
etmez.

`section` ve `subsection`, başlıklarını kaçış işlemine tabi tutar.
`equation`, kasıtlı olarak kaçış işlemi yapmaz, çünkü ham LaTeX matematik
kaynağı kabul eder. `table`, `booktabs` kaynağı üretir ve her hücreyi
kaçış işlemine tabi tutar; sütun sayısı başlıklardan farklı olan bir
satır `ValueError` fırlatır.

`document`, önsözü (preamble) paketlenmiş Latin Modern yazı tipi
dosyalarını açıkça isimlendiren tam bir belge döndürür, bu yüzden
render işlemi asla bir ana bilgisayar sistem yazı tipine bağlı değildir.

### 37.2 Derleme

`pdf`, bir kaynak String'i derler ve `pdfFile`, mevcut bir `.tex`
dosyasını derler; `\includegraphics` gibi belgeye-göreli varlıkları girdi
dosyasının dizinine göre çözer.

Derleme, bir AhdCode kurulumuyla birlikte gelen bir Tectonic motoru ve
yerel bir kaynak paketi kullanır. AhdCode, `PATH`'te bulunan bir
`tectonic`'i asla çalıştırmaz, asla bir sistem TeX kurulumuna geri
dönmez ve çalışma zamanında asla bir kaynak indirmez. Bu yüzden
desteklenen bir belge, boş önbelleğe ve ağ bağlantısı olmayan taze bir
makinede derlenir. Eksik bir paketlenmiş motor veya paket, bir
`LatexError`'dır.

### 37.3 Güvenlik ve sınırlar

Motor güvenilmeyen (untrusted) modda çalışır, bu yüzden kabuk kaçışı
(shell escape) kullanılamaz ve hiçbir AhdCode yapısı onu etkinleştiremez.
Motor, bir kabuk komut dizesi yerine bir argüman vektörüyle başlatılır,
bu yüzden boşluk, Unicode, tırnak, `$`, `;`, `&` veya parantez içeren
yollar güvenli kalır. Derleme, 30 saniyelik bir zaman aşımıyla
sınırlıdır; zaman aşımında süreç sonlandırılır, geçici dosyalar
kaldırılır ve `LatexError` fırlatılır.

### 37.4 Çıktı güvenliği

Kaynak, hem başarıda hem başarısızlıkta kaldırılan benzersiz, güvenli
bir geçici dizinde derlenir. PDF, geçici bir konumda üretilir ve istenen
hedefin yerini almadan önce varlık, normal-dosya durumu, sıfır olmayan
boyut ve `%PDF-` imzası kontrol edilir, bu yüzden başarısız bir derleme
asla zaten geçerli olan bir hedef PDF'yi yok etmez.

### 37.5 LatexError

`LatexError`, derleme başarısızlığını, eksik bir paketlenmiş motoru veya
paketi, zaman aşımını, motor süreç başarısızlığını ve üretilmemiş bir
PDF'yi kapsar. Motor tanılamaları sınırlıdır, bu yüzden bozuk bir belge
terminali dolduramaz, ancak ilk yararlı TeX hatası korunur.

### 37.6 Bu sürümde olmayanlar

BibTeX yönetimi, paket yöneticisi, TikZ veya Beamer soyutlaması, PDF
düzenleyici veya ayrıştırıcı ve Markdown veya HTML dönüşümü yoktur.

---

### 37.7 Path ve File Standart Modülleri

`Path` ve `File`, sıradan modül sistemi aracılığıyla içe aktarılan,
derleyici tarafından kayıtlı standart modüllerdir. Önceden bildirilmiş
Fundamentals değildirler.

```ahd
bring Path
bring File
from File bring FileError
```

Tam v0.1 herkese açık yüzeyleri şudur:

```text
Path.join(parts: List<String>) -> String
Path.ext(path: String)         -> String
Path.base(path: String)        -> String
Path.dir(path: String)         -> String

File.exists(path: String)                  -> Bool
File.readText(path: String)                -> String
File.writeText(path: String, content: String) -> Nothing
File.append(path: String, content: String) -> Nothing
File.delete(path: String)                  -> Nothing
File.createDir(path: String)               -> Nothing
File.list(path: String)                    -> List<String>

FileError
```

Path işlemleri, ana bilgisayar işletim sisteminin yol kurallarını
kullanır ve hiçbir dosya sistemi erişimi gerçekleştirmez. Dosya metni
UTF-8'dir; geçersiz UTF-8 okumak `FileError` fırlatır. `File.list`,
yalnızca en yakın giriş isimlerini kararlı artan sözlüksel sırada
döndürür ve asla özyinelemez (recurse). Göreli yollar, bir REPL
oturumunun başlatıldığı dizin dahil, yürütülen sürecin geçerli çalışma
dizinini kullanır.

Sıradan işletim sistemi başarısızlıkları, yakalanabilir AhdCode
hatalarıdır. `FileError`, `IOError`'dan miras alır ve o da `Error`'dan
miras alır; `File.exists`'e geçirilen eksik bir yol `false` verir,
diğer işlemlerin başarısızlıkları ise `FileError` fırlatır. Hiçbir ham
ana bilgisayar hata değeri veya Go paniği ifşa edilmez.

---

## 38. Çekirdek Terminal Girdi/Çıktı

### 38.1 take

`take`, terminal girdi fonksiyonudur. Tam olarak iki biçimi vardır:

```text
take()               -> String
take(prompt: String) -> String
```

Her ikisi de standart girdiden tam olarak bir satır okur ve onu bir
`String` olarak döndürür.

```ahd
name: String := take()
```

```ahd
name: String := take("Name: ")
```

Prompt biçimi, önce prompt'u kendi yeni satırını eklemeden yazar ve
prompt, program girdi için bloke olmadan önce görünürdür. Prompt asla
döndürülen metnin bir parçası değildir. Prompt argümanı `NonNull` bir
`String` olmalıdır.

Döndürülen `String`, hem `LF` hem `CRLF` biçimlerinde, satır sonlandırıcı
karakteri hariç tutar. Girilen metin içindeki sıradan boşluklar
korunur:

| stdin | sonuç |
|---|---|
| `Ali\n` | `"Ali"` |
| `  Ali  \n` | `"  Ali  "` |
| `\n` | `""` |
| girdi sonu | `""` |

`take`, okuduğunu asla ayrıştırmaz veya dönüştürmez. AhdCode kesin
olarak statik tipli kalır, bu yüzden sayısal girdi sıradan
dönüşümlerden geçer:

```ahd
age: Int := int(take("Age: "))
value: Real := real(take())
```

Bu geçersizdir, çünkü bir `String` örtük olarak bir `Int` değildir:

```ahd
age: Int := take()
```

`take`, v0.1'deki tek terminal girdi fonksiyonudur. `takeInt`,
`takeReal`, `input` veya `readLine` yoktur.

### 38.2 write

```ahd
write("Hello {name}")
```

---

## 39. Çalışma Zamanı Sayısal Güvenliği

AhdCode, sürpriz düşük seviyeli sayısal davranışlar yerine açık hataları
tercih eder.

- sıfıra bölme => hata;
- Int64 taşması => hata, sessiz sarmalama (wrap) yok;
- Real taşması => sıradan AhdCode işlemlerinde hata;
- gerçek-alan (real-domain) geçersiz işlemler, mümkün olduğunda NaN'ı
  sessizce ifşa etmek yerine bir alan hatasını (domain error) tercih
  etmelidir.

Karmaşık matematik, daha sonraki bir Complex olanağına aittir.

---

## 40. Desteklenmeyen v0.1 Özellikleri

Kasıtlı olarak hariç tutulanlar:

- web çalışma zamanı / AhdWeb
- HTTP yönlendirme
- MySQL
- SMTP
- HTML düzenleri
- statik Class üyeleri
- Getter/Setter sözdizimi
- blok/deyim lambda'ları ve lexical closure'lar
- genel/sınırsız kullanıcı-tanımlı operatör aşırı yüklemesi (yalnızca
  §47'nin on sabit Class Protocol Methods'ı vardır; keyfi operatör
  tanımı, ters operatörler veya yerinde (in-place) protokoller yoktur)
- çoklu dönüş değerleri
- tuple/çoklu atama
- zincirleme atama
- goto/etiketler
- etiketli break/continue
- dilim-adım (slice-step) sözdizimi
- Char türü
- JS tarzı örtük zorlama (coercion)
- `===`
- trait'ler/interface'ler/mixin'ler
- dekoratörler/anotasyonlar
- çoklu kalıtım
- `reduce` ve diğer katlama (fold) işlemleri
- `sort`'un karşılaştırıcı (comparator) veya azalan (descending) biçimleri
- `append`, `push`, `remove`, `findIndex`, `foreach`, `select`, `where`
  ve `transform` gibi List/String işlem takma adları (aliases)

---

## 41. Planlanan Derleyici Boru Hattı

```text
AhdCode kaynağı (.ahd)
        ↓
Lexer
        ↓
Parser
        ↓
AST
        ↓
Semantik/tür analizi
        ↓
Tipli/indirgenmiş (lowered) IR
        ↓
Go kod üretimi
        ↓
Go derleyicisi
        ↓
yerel (native) çalıştırılabilir dosya
```

IR, herkese açık AhdCode sözdizimi değil, dahili bir derleyici
katmanıdır. Go üretiminden önce denetimli aritmetik, çalışma zamanı
null güvenliği, derin dondurma (deep freeze), anlık görüntü (snapshot)
yinelemesi, Class kimliği ve diğer AhdCode anlambilimini açık hale
getirebilir.

Uygulamanın kendisi Go ile yazılabilir.

AhdCode, ince bir Python/JavaScript eval sarmalayıcısı veya yalnızca
regex tabanlı bir çevirici olmamalıdır.

---

## 42. CLI ve etkileşimli araç zinciri

```bash
ahdcode
```

REPL.

REPL, dosya derlemesiyle aynı lexer, parser, semantik denetleyici ve
tipli/indirgenmiş IR'yi kullanır. Doğrulanmış yeni IR, tek bir kalıcı
değerlendiricide (evaluator) çalışır; önceki ifadeler yeniden
çalıştırılmaz. Değerler, takma adlar (aliases), Function'lar, Class'lar,
içe aktarmalar, modül başlatma, Math RNG durumu, çalışma dizini ve
terminal akışları kalıcıdır. Ayrı bir mini-dil değildir. Oturum
bildirimleri kalıcıdır ve sıradan aynı-kapsam bildirim kuralları
yürürlükte kalır: `x := 5` girmek ve sonra `x := 7` girmek, tekrarlanan
bir bildirim hatasıdır. Yeniden atama `x = 7` olarak yazılır. Başarısız
bir semantik denetim veya yakalanabilir bir çalışma zamanı Error'u, son
başarıyla işlenmiş oturum durumunu atmaz veya REPL'i sonlandırmaz.

`take()`, REPL ile aynı gerçek terminal girdi akışını okur. Prompt'u
bloke olmadan önce flush edilir ve tükettiği tek yanıt satırı, sonraki
bir REPL komutu olarak ayrıştırılmaz. Türü `Nothing` olmayan üst düzey
bir ifade, kanonik biçimde yazdırılır.

```bash
ahdcode run hello.ahd
```

Dosyayı çalıştır.

```bash
ahdcode build hello.ahd
```

Go arka ucu (backend) aracılığıyla yerel çalıştırılabilir dosya
oluştur.

```bash
ahdcode format hello.ahd
```

Kanonik yerinde (in-place) formatter. `ahdcode format --check hello.ahd`,
dosyayı değiştirmeden aynı doğrulama ve kanonikleştirme karşılaştırmasını
gerçekleştirir; yalnızca kaynak zaten kanonik olduğunda başarılı olur.

`ahdcode --help`, desteklenen komutları açıklar ve `ahdcode --version`,
kanonik v0.1 sürüm dizesini yazdırır. Bilinmeyen komutlar ve bayraklar
(flags), bir kabuk çağırmadan veya argümanları kaynak metin olarak ele
almadan başarısız olur.

---

## 43. Örnek Program

```ahd
PI: Constant Real := 3.14159

square: Function := (
    x: Real
) -> Real {
    return x^2
}

radiusInput: Int := 5
radius: Real := real(radiusInput)

if radius > 0 {
    area: Local Real := PI * square(radius)
    write("Area: {area}")
}
else {
    write("Radius must be positive")
}
```

---

## 44. Örnek Class

```ahd
Person: Class<> := {
    structure: Attributes := (
        name: String
        age: Int
    )

    describe: Function := (
    ) -> String {
        return "{attribute.name} - {attribute.age}"
    }
}

Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Constant Int
        password: Local String
    ) {
        attribute.passwordHash: Confidential String := hash(password)
    }

    describe: Override Function := (
    ) -> String {
        base: Local String := SuperClass.describe()
        return "{base} - {attribute.number}"
    }
}
```

---

## 45. Uygulama Kuralı: Anlambilimi Sessizce İcat Etme

Spesifikasyon belirsiz olduğunda:

1. Python, Go, C, Java veya JavaScript'i sessizce kopyalama;
2. yalnızca uygulaması kolay olduğu için herkese açık sözdizimi ekleme;
3. belirsizliği izole et;
4. belgelendir;
5. dondurmadan önce bir dil-tasarım kararı iste.

Özellikle, belirsizliği gizli bir `Any`, dinamik Function dispatch'i
veya denetimsiz null olabilen davranış ekleyerek çözme.

---

## 46. v0.1 Tamamlanma Tanımı (Definition of Done)

Çekirdek v0.1, şu durumda anlamlı olarak canlıdır:

1. Lexer, Unicode tanımlayıcıları, sayıları, string'leri, üçlü
   string'leri, kaçış karakterlerini, yorumları, anahtar kelimeleri,
   operatörleri, yeni satır/virgül ayrımını ele alır.
2. Parser, çekirdek bildirimler, ifadeler, Function'lar, Class'lar,
   kontrol akışı, koleksiyonlar, bring ve hata yönetimi için gerçek bir
   AST oluşturur.
3. Semantik denetleyici; türleri, akış-duyarlı null-durum kurallarını,
   Function imza çıkarımını, kapsamı, aşırı yüklemeyi, kalıtımı,
   Constant'ı ve görünürlüğü zorunlu kılar.
4. `write` ve `take` çalışır.
5. Function'lar, class'lar, List'ler, Pair'ler, döngüler ve hata
   yönetimi doğru şekilde çalıştırılır.
6. Go kod üretimi, temsili `.ahd` programlarını derler.
7. REPL, sıradan çekirdek kod için çalışır.
8. Formatter deterministiktir/idempotenttir.
9. Testler, bozuk sözdizimini ve düşmanca (adversarial) semantik
   durumları kapsar.
10. Çekirdek tamamlanma için hiçbir web işlevselliği gerekli değildir.

---

## 47. Class Protocol Methods (v0.1.8)

Bir Class, kendi örnekleri için operatör davranışını, **Class Protocol
Methods** adı verilen, tam olarak on ayrılmış (reserved) metot isminden
oluşan küçük, kapalı bir küme aracılığıyla tanımlayabilir:

```text
CEqual CCompare
CAdd CSubtract CMultiply CDivide CRemainder CPower
CNegate CStr
```

Başka hiçbiri yoktur. Bu, kasıtlı olarak dar bir derleyici genişletmesidir,
genel/sınırsız operatör aşırı yüklemesi değildir ve Python'un magic-method
sistemi değildir: `__eq__`/`__lt__`/`__repr__`/`__radd__` tarzı çift alt
çizgi kuralı yoktur ve her dil mekanizmasına (oluşturma, yineleme,
indeksleme, öznitelik erişimi, çağırma) kendi protokol ismini verme
girişimi yoktur.

### 47.1 İsimlerin ayrıldığı yer

On isim, yalnızca bir Class metot yuvasını (slot) işgal ettiklerinde
derleyici tarafından özeldir. Modül kapsamında, `CAdd: Function := ...`,
sıradan bir Function olarak kalır. Bir Class içinde, on isimden biri
olmayan bir isim (`Calculate`, `Create`, `CWhatever`, `CustomMethod` vb.)
sıradan bir üye olarak kalır; `C` harfinin kendisi hiçbir anlam taşımaz.
On isimden birini yeniden kullanan ancak bir Function olmayan bir Class
gövdesi üyesi -- örneğin `CAdd: Int := 5` -- sessizce kabul edilen bir
alan değil, bir derleme zamanı hatasıdır (ayrılmış Class Protocol Method
yuvası).

### 47.2 Bildirim sözdizimi değişmemiştir

Bir Class Protocol Method, sıradan metot sözdizimiyle yazılır; yeni bir
bildirim biçimi yoktur ve Function/Class bildirim sözdizimi değişmez:

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
```

### 47.3 Gerekli imzalar

| Protokol | Açık parametreler | Dönüş türü |
|---|---|---|
| `CEqual` | tam olarak 1 | `Bool` |
| `CCompare` | tam olarak 1 | `Int` |
| `CAdd`, `CSubtract`, `CMultiply`, `CDivide`, `CRemainder`, `CPower` | tam olarak 1 | operatörün sonuç türü; içeren Class'a eşit olması gerekmez |
| `CNegate` | 0 | operatörün sonuç türü |
| `CStr` | 0 | `String` |

Bozuk bir bildirim (yanlış arite, yanlış dönüş türü veya ayrılmış bir
ismin Function olmayan bir yuvayı işgal etmesi), asla bir çalışma zamanı
paniği değil, sıradan bir semantik tanılamadır. Aritmetik protokoller ve
`CNegate`, meşru olarak içeren Class'tan farklı bir tür döndürebilir --
örneğin gelecekteki bir `Matrix * Vector -> Vector` -- ve operatör
ifadesinin statik türü basitçe seçilen metodun bildirilen dönüş türüdür.

### 47.4 Operatör eşlemesi

| Operatör | Protokol |
|---|---|
| `==` | `CEqual` |
| `!=` | aynı `CEqual` çağrısının mantıksal olumsuzlaması; `CNotEqual` yoktur |
| `<`, `<=`, `>`, `>=` | dördü de, ifade başına tam olarak bir kez değerlendirilen tek bir `CCompare` çağrısından türetilir |
| `+` | `CAdd` |
| `-` (ikili) | `CSubtract` |
| `*` | `CMultiply` |
| `/` | `CDivide` |
| `%` | `CRemainder` |
| `^` | `CPower` |
| `-` (tekli) | `CNegate` |

`CCompare`'ın sonucu geleneksel işaret yorumunu kullanır ve
`-1`/`0`/`1` ile **sınırlı değildir**:

```text
a <  b   =>  a.CCompare(b) <  0
a <= b   =>  a.CCompare(b) <= 0
a >  b   =>  a.CCompare(b) >  0
a >= b   =>  a.CCompare(b) >= 0
```

`CEqual`, asla `CCompare`'dan türetilmez ve `CCompare`, asla `CEqual`'dan
türetilmez: bir tür, doğal bir sıralaması olmadan anlamlı bir şekilde
eşitlik-karşılaştırılabilir olabilir ve tersi de geçerlidir. Bir Class
hiçbir `CEqual` sağlamıyorsa, `==`/`!=`, §29'un v0.1.8 öncesi referans-
eşitliği kuralını değiştirmeden korur; hiçbir `CCompare` sağlamıyorsa,
`<`/`<=`/`>`/`>=`, sıradan statik tür hatası olarak kalır.

### 47.5 Dispatch sol-operanda dayalıdır

Ters-operatör protokolü yoktur (`CReverseAdd`/`CRAdd` vb.):

```ahd
value + 3   // value'nun Class'ı CAdd(Int)'i bildiriyorsa çalışır
3 + value   // value'nun CAdd'ini DENEMEZ; sıradan ilkel-operatör kuralları
            // uygulanır ve bağımsız olarak geçerli olmadıkça bu bir statik tür hatasıdır
```

### 47.6 Aşırı yükleme, kalıtım ve dinamik dispatch

Class Protocol Methods, sıradan metot aşırı-yükleme-çözümlemesini,
kalıtımı ve dinamik-dispatch makinesini yeniden kullanır; ikinci bir
sistem yoktur. Bir Class, mevcut `Overload Function` kuralları (§26)
altında birden fazla `CAdd` aşırı yüklemesi bildirebilir, operatör
çözümlemesi açık bir çağrıyla aynı statik aşırı yükleme çözümleme
kurallarını kullanır ve belirsizlik sıradan derleme zamanı hatasıdır.
Bir protokol metodu, tam olarak diğer herhangi bir metot gibi miras
alınır ve geçersiz kılınır (`Override`, birini değiştirmek için
gereklidir) ve operatör dispatch'i, sıradan bir metot çağrısıyla aynı
dinamik dispatch'i kullanır: statik olarak geçerli bir protokol metodu
bir alt sınıf (subclass) tarafından geçersiz kılınmışsa, çalışma zamanı
nesnesi o alt sınıf olduğunda, tam olarak §26'daki gibi geçersiz kılma
(override) çalışır.

### 47.7 Bileşik atama

Bir Class hedefi üzerinde `+=`, `-=`, `*=`, `/=`, `%=` ve `^=`, eşleşen
aritmetik protokolü yeniden kullanır: `a += b`, normal atama-uyumluluğu
kurallarına tabi olarak, alıcı (receiver) tam olarak bir kez
değerlendirilerek (bir üye veya indekslenmiş hedef dahil) `a = a + b`
gibi davranır. Ayrı bir yerinde (in-place) protokol yoktur (`CIAdd`
tarzı bir metot yoktur). `++` ve `--` ilgisizdir ve Class değerlerine
genişletilmemiştir.

### 47.8 Null olabilirlik

Protokol dispatch'i, §§17-24'ün null-güvenliği kurallarını asla
zayıflatmaz. Sol operand null olabiliyorsa, üzerinde bir protokol
metodu çağrılabilmesi için önce sıradan akış analiziyle `NonNull`'a
daraltılmalıdır -- diğer herhangi bir metot çağrısıyla aynı gereklilik:

```ahd
user: User?

user + other   // user, null olmayana daraltılmadıkça geçersiz
```

Sağ taraf argümanı, kendi bildirilen parametre türünü ve null
olabilirliğini normal şekilde kullanır; bir protokol, null olabilen bir
argümanı açıkça kabul edebilir (`CEqual(other: User?) -> Bool`), ancak
bu kabulle ilgili hiçbir şey otomatik olarak çıkarılmaz.

### 47.9 CStr ve str()

Statik olarak bildirilen türü `CStr`'ı çözen bir Class örneği için,
Fundamental `str(value)` (§34) ona dispatch eder ve `write`, aynı
paylaşılan dönüşüm yolundan yararlanır. `repr()`/`CRepr` yoktur ve
ikinci bir geliştirici/hata ayıklama string protokolü yoktur. `CStr`'ı
olmayan bir Class, §34.1'in mevcut `<ClassName>` render'ını korur.

---

## 48. Çalışma Zamanı İç Gözlem (Introspection) Fundamentals: `type()` ve `id()` (v0.1.8)

### 48.1 `type(value) -> String`

`value`'nun kanonik AhdCode tür ismini bir `String` olarak döndürür.
Küçük bir çalışma zamanı/iç gözlem yardımcısıdır, bir reflection
çerçevesi değildir: asla birinci sınıf bir tür nesnesi döndürmez ve
metaclass veya reflection API'si yoktur.

```text
type(5)      -> "Int"
type(5.0)    -> "Real"
type("Ali")  -> "String"
type(true)   -> "Bool"
type(null)   -> "Null"
```

Koleksiyonlar, dilin yeterli statik tür bilgisine sahip olduğu her
yerde kanonik AhdCode generic gösterimini kullanır: `List<Int>`,
`List<Int?>`, `Pair<String, Int>`. `type`, asla bir arka uç/Go
uygulama türünü ifşa etmez.

Bir Class örneği için, `type`, ifadenin statik/bildirilen türünü değil,
**en-türetilmiş (most-derived) çalışma zamanı Class'ını** bildirir:

```ahd
animal: Animal := Dog(name: "Rex")
write(type(animal))   // "Animal" değil, "Dog"
```

Şu anda null olmayan bir değer tutan null olabilen bir değer için,
`type`, bildirilen türün sondaki `?`'sini değil, içerilen değerin kendi
türünü bildirir. `null`'ın kendisi için, `type`, literal `"Null"`
String'ini bildirir; bu, içsel bir Fundamental özel durumdur ve `Null`'ı
sıradan bir kaynak-seviyesi bildirim türü olarak **tanıtmaz** ve
§17'nin `x := null` çıkarım reddini zayıflatmaz.

Bir Function değerinin `type()` metni, başka yerlerde tanılamalarda
zaten kullanılan aynı kanonik imza biçimlendirmesini yeniden kullanır
(`Function(ParamType, ...) -> ReturnType`), asla bir Go-seviyesi
gösterim değil.

### 48.2 `id(reference) -> Int`

Bir **List, Pair veya Class örneği** için opak, çalışma zamanı tarafından
yönetilen bir kimlik numarası döndürür. `Int`, `Real`, `String` ve `Bool`
kabul edilmez -- `id(5)` ve `id("x")` derleme zamanı hatalarıdır ve
hiçbir ilkel (primitive), yalnızca bir kimlik üretmek için kutulanmaz
(boxed).

Döndürülen sayı:

- opaktır ve yalnızca geçerli süreç veya REPL oturumu içinde anlamlıdır;
- bir bellek adresi **değildir** ve asla birinden türetilmez;
- ayrı program yürütmeleri arasında kararlı olması garanti edilmez;
- serileştirme verisi veya kalıcı/veritabanı tanımlayıcısı değildir;
- dahili olarak artan bir tahsis ediciden (allocator) üretilmiş olabilir,
  ancak bir program asla tahsis sırasına değil, yalnızca iki kimlik
  arasındaki eşitlik/eşitsizliğe bağlı olmalıdır.

Bir süreç veya REPL oturumu içinde, bir takma ad (alias) kimliğini
orijinaliyle paylaşır (`b := a` olduğunda `id(a) == id(b)`) ve aynı anda
var olan iki farklı nesne farklı kimliklere sahiptir. Kimlik, bir
nesnenin tüm yaşam süresi boyunca kararlıdır ve mutasyondan etkilenmez
(bir List, Pair veya Class örneğini mutasyona uğratmak asla `id()`'sini
değiştirmez). `id()`, §§17-24'ün sıradan null olabilen-kullanım
kurallarına tabi olarak null olmayan bir kimlik-taşıyan referans
gerektirir.

`id()`, `same`'ın (§29) yerini almaz: `same`, sıradan programatik kimlik
testidir; `id()`, hata ayıklama, günlükleme (logging) ve iç gözlem için
vardır. Desteklenen canlı referans değerleri için,
`(a same b) == (id(a) == id(b))`.

---

## 49. Regex Standart Modülü (v0.1.9)

`Regex`, tam olarak `Math` ve `Time` gibi açıktır (§33, §35, §36):
kullanılmadan önce `bring Regex` ile içe aktarılmalıdır ve kanonik
kimliği `builtin:Regex`'tır, bu yüzden kardeş bir `Regex.ahd` dosyası
onu gölgeleyemez (shadow).

```ahd
bring Regex
from Regex bring Pattern
from Regex bring RegexError
```

### 49.1 Bir desen (pattern) derlemek

```text
Regex.compile(pattern: String) -> Pattern
```

`Regex.compile`, Go `regexp` (RE2) sözdizimini kullanarak bir deseni
derler ve bir `Pattern` örneği döndürür. Geçersiz bir desen,
`Error`'dan doğrudan türeyen (IOError'dan değil) yakalanabilir
`RegexError`'ı fırlatır. `Pattern`, derleyici tarafından sağlanan bir
Class'tır: tam olarak §36'daki `Time.dateTime` ve `DateTime` gibi, asla
doğrudan oluşturulmaz, yalnızca `Regex.compile` tarafından üretilir.

Class, `Regex` değil `Pattern` olarak adlandırılmıştır, özellikle
modülün kendi isim uzayından bağımsız olarak adlandırılabilmesi için
(`bring Regex` zaten `Regex` ismini bağlar; türü adlandırmak için
`from Regex bring Pattern` gereklidir).

### 49.2 Pattern üyeleri

```text
matches(text: String)                    -> Bool
find(text: String)                       -> String?
findAll(text: String)                    -> List<String>
groups(text: String)                     -> List<String>?
replace(text: String, replacement: String) -> String
split(text: String)                      -> List<String>
```

- `matches`, desenin `text` içinde herhangi bir yerde bulunup
  bulunmadığını bildirir (örtük bir tam-string sabitleyici değildir;
  bunun için desende `^...$` yazın).
- `find`, ilk eşleşmeyi veya desen `text` içinde geçmiyorsa `null`
  döndürür. Sonuç `String?`'tir; kullanımdan önce sıradan null-güvenliği
  daraltması (§§17-24) uygulanır.
- `findAll`, sırayla her çakışmayan (non-overlapping) eşleşmeyi
  döndürür; eşleşme yoksa boş bir `List<String>`.
- `groups`, ilk eşleşmenin tam eşleşme metnini ve ardından yakalama
  gruplarını döndürür (indeks `0` tüm eşleşmedir) veya desen
  geçmiyorsa `null`. Eşleşmemiş isteğe bağlı bir grup, temeldeki RE2
  alt-eşleşme (submatch) kuralıyla eşleşerek boş bir `String` olarak
  bildirilir.
- `replace`, her eşleşmeyi, `$1`, `$2` vb. olarak yakalama gruplarına
  başvurabilen `replacement` ile yeniden yazar.
- `split`, `text`'i desenin her eşleşmesinde böler.

Her argüman `String` ve `NonNull`'dır. `has`/`has not` (§29), bu altı
ismi bir `Pattern` örneğinin var olan üyeleri olarak bildirir.

### 49.3 Önbelleğe alma ve determinizm

Bir `Pattern`'ın tek gözlemlenebilir durumu, kaynak desen metnidir
(sıradan Class anlambilimi aracılığıyla okunabilir); derlenmiş eşleyici
(matcher) kendisi, desen metnine göre dahili olarak önbelleğe alınan bir
uygulama detayıdır, bu yüzden aynı `Pattern` değerinin tekrarlanan
kullanımı -- veya aynı desen dizesiyle tekrarlanan `Regex.compile`
çağrıları -- derleme maliyetini tekrar tekrar ödemez. Eşleştirme, yer
değiştirme (replacement) ve bölme (splitting), belirli bir desen ve
girdi için deterministiktir.

### 49.4 RegexError

```ahd
attempt {
    Regex.compile("(unterminated")
}
except RegexError as error {
    write(error.message)
}
```

`RegexError`, yalnızca `Regex.compile` tarafından geçersiz desen
sözdiziminde fırlatılır. Başka hiçbir `Pattern` işlemi onu fırlatmaz:
eşleştirme, bulma, yer değiştirme ve bölme, bir `Pattern` var
olduktan sonra asla başarısız olmaz.

---

## 50. İfade Lambda'ları (v0.1.10)

Lambda, AhdCode'un mevcut `Function` türünde bir değer oluşturmak için kısa
bir ifade sözdizimidir. Bir `Lambda` türü, ikinci bir çağrılabilir aile veya
isimli Function bildirimlerinin yerine geçen bir sistem değildir.

### 50.1 Gramer

```text
lambda-expression  ::= "lambda" "(" [ lambda-parameter-list ] ")"
                       "->" expression
lambda-parameter   ::= identifier ":" type-reference
```

Parametre listesi sıradan virgül/satır sonu ayırıcı kurallarını kullanır. Her
parametrenin açık bir türü vardır; sıfır parametre geçerlidir. Mevcut Function
parametre türleri uygulanır. Parametre bildirim değiştiricileri ve varsayılan
değerler v0.1.10'da kabul edilmez; varsayılan parametre gerektiğinde isimli bir
Function bildirimi kullanılır.

```ahd
positive := lambda (x: Int) -> x > 0
difference := lambda (x: Int, y: Int) -> x^2 - y^2
now := lambda () -> Time.now()
```

Yazılan bir lambda dönüş belirtimi yoktur. Statik dönüş türü ve dönüş null
durumu, tek gövde ifadesinden çıkarılır. Geçersiz veya çözümlenemeyen gövde
türü semantik hatadır; asla dinamik tiplemeye geri dönülmez. Gizli
String/sayısal/truthiness zorlamalarının bulunmaması dahil mevcut atanabilirlik,
null olabilirlik, argüman sayısı ve dönüşüm kuralları değişmeden uygulanır.

### 50.2 Yalnızca ifade sınırı

Bir lambda gövdesi tam olarak bir ifadedir. `{ ... }` bloğu, `return`, `if`,
döngü, bildirim, `attempt` veya başka bir deyim gövdesi geçersizdir. Deyim
gerektiren mantık, değişmeyen isimli Function biçimini kullanır:

```ahd
positive: Function := (x: Int) -> Bool {
    if x <= 0 {
        return false
    }
    return true
}
```

### 50.3 Function uyumluluğu ve callback'ler

Herkese açık kaynak türü `Function` olarak kalırken derleyici somut
çağrılabilir imzayı dahili olarak korur:

```ahd
positive: Function := lambda (x: Int) -> x > 0
inferred := lambda (x: Int) -> x > 0
values.filter(lambda (x: Int) -> x > 0)
values.map(lambda (x: Int) -> x^2)
values.sort(lambda (x: Int) -> -x)
```

Lambda, bu kesin Function imzasının kabul edildiği her yerde çalışır. `map`,
`filter` ve anahtarlı `sort` mevcut sözleşmelerini korur; koleksiyon API'si
eklenmez veya yeniden tasarlanmaz.

### 50.4 Kapsam ve lexical yakalama

Lambda bir ifadedir, bildirim değildir. Atanması, sıradan modül-kökü ve açık
`Local` bildirim kurallarını izler. Lambda parametreleri lambda içinde örtük
olarak yereldir.

v0.1.10 hiçbir closure ortamı tanıtmadı: bir lambda, çevreleyen bir
çağrılabilirin lexical kapsamındaki bir bağlamayı veya bir modül bağlamasını
hiç okuyamıyordu. v0.1.13 bu kısıtlamayı açık bir bağımlılık listesiyle (§54)
değiştirir: çevreleyen bir Function parametresi veya `Local`, lambda onu
`#name`/`Local name` olarak listelediğinde okunabilir; bir modül bağlaması ise
lambda onu `@name`/`Global name` olarak listelediğinde okunabilir -- bu,
sıradan bir Function'ın zaten ihtiyaç duyduğu açık `Global` bildirimini
yansıtır. Her iki türden bir bağlamayı listelemeden okumak semantik hata olarak
kalır. Function'lar, Class'lar, ad alanları ve içe aktarımlar mevcut görünürlük
kurallarını korur ve hiçbir bağımlılık-listesi girdisine ihtiyaç duymaz;
modül bağlamaları için mevcut açık `Global` kuralı yalnızca ifade lambda
olduğu için zayıflatılmaz -- bu aynı kural, sadece kısaca yazılmıştır.

### 50.5 Uygulama ve araçlar

`lambda` ayrılmış bir anahtar kelimedir ve gerçek bir `LambdaExpr` AST düğümüne
ayrıştırılır. Semantik denetleyici, her Function değeri için kullanılan aynı
somut çağrılabilir imzayı üretir. Lowering, sıradan tiplenmiş bir Function IR
çağrılabiliri ve `FunctionValueExpr` üretir; böylece yerel Go backend'i ve
kalıcı değerlendirici mevcut Function adaptörlerini ve çağrı yollarını yeniden
kullanır. Kaynak yeniden yazımı veya çalışma zamanı Lambda kimliği yoktur ve
`id()` Function'lara genişletilmez.

Kalıcı REPL, lambda Function değerlerini komutlar arasında korur. Formatter,
sıradan Function parametre-listesi düzenini kullanır ve idempotenttir:

```ahd
lambda (x: Int) -> x > 0
```

Uzun parametre listeleri mevcut 80-sütun politikasına göre bölünür; tek ifade
`->` sonrasında kalır.

---

## 51. CSV Standart Modülü (v0.1.11)

`CSV`, açık ve derleyiciye kayıtlı `builtin:CSV` modülüdür; kardeş `CSV.ahd`
onun yerini alamaz. Yalnız String taşır; tür çıkarımı veya DataFrame/tablo
modellemesi yapmaz.

```text
parse(text: String, delimiter: String = ",") -> List<List<String>>
stringify(rows: List<List<String>>, delimiter: String = ",") -> String
read(path: String, delimiter: String = ",") -> List<List<String>>
write(path: String, rows: List<List<String>>, delimiter: String = ",") -> Nothing
parseRecords(text: String, delimiter: String = ",") -> List<Pair<String, String>>
readRecords(path: String, delimiter: String = ",") -> List<Pair<String, String>>
stringifyRecords(records: List<Pair<String, String>>, delimiter: String = ",") -> String
writeRecords(path: String, records: List<Pair<String, String>>, delimiter: String = ",") -> Nothing
```

Ham ayrıştırma standart tırnaklama, kaçırılmış tırnak, gömülü ayraç/yeni satır,
LF/CRLF, Unicode, boş alan ve değişken genişlikli satırları destekler. Boş ham
satırlar `""`a çevrilir; kodlama deterministik Go `encoding/csv` çıktısıdır.

Kayıt ayrıştırma ilk satırı boş olmayan benzersiz başlıklar olarak kullanır.
Boş ve yalnız başlıklı girdi boş List döndürür. Her veri satırı tam başlık
genişliğinde olmalıdır. Yazmada ilk Pair sütun sırasını belirler; sonraki her
Pair, ekleme sırası farklı olsa da tam aynı anahtar kümesine sahip olmalıdır.

Ayraç tam bir geçerli Unicode scalar olmalı; tırnak, CR veya LF olmamalıdır.
Geçersiz ayraç, bozuk CSV/UTF-8 ve başlık/kayıt şekli doğrudan `Error`'dan
türeyen `CSVError` fırlatır. Dosya erişim hataları `FileError`/`IOError`
anlamını korur. Göreli yollar REPL başlatma dizini dahil işlem çalışma dizinidir.

## 52. Tanılama kalitesi ve recovery (v0.1.11)

Tanılamalar kod, önem, mesaj, ipucu ve kesin kaynak aralığı taşır. `PAR010`,
atanan/varsayılan ifadenin operatörün fiziksel satırından sonra başlamasını;
`SEM022`, lexical yakalama yasağını belirtmeye devam eder. `PAR013`,
desteklenmeyen yeni-satır başı nokta devamını belirtir ve türev tanılama
zincirlerini bastırırken ilerideki bağımsız hataları gizlemeden deyim sınırında
toparlanır.

Yapı bilindiğinde eksik initializer, atama, ikili operand, index, lambda
gövdesi, çağrı/List/Pair/grup ve kapatıcı mesajları eksik parçayı adlandırır.
Lambda blokları reddedilmeye devam eder. Çalışma zamanı alan hataları AhdCode
Error'larıdır (`RegexError`, `ValueError`, `CSVError`, `FileError`); Go panic
veya stack trace göstermemelidir.

---

## 53. Data Standart Modülü (v0.1.12)

`Data`, `Math`, `Time`, `Regex` ve `CSV` gibi açıktır (§33, §51): kullanılmadan
önce içe aktarılmalıdır ve kanonik kimliği `builtin:Data`'dır, bu yüzden
kardeş bir `Data.ahd` onu gölgeleyemez.

```ahd
bring Data
from Data bring Table
from Data bring DataError
```

`Data`, mevcut String, List, Pair, Function ve CSV olanakları üzerine kurulu
bir tablo yapısı ve dönüşüm katmanıdır. Kasıtlı olarak heterojen bir DataFrame
çalışma zamanı değildir: `Any`, dinamik veya birleşim (union) türü, varyant
hücre veya yeni bir dil seviyesi generic eklemez.

### 53.1 String hücre modeli

Her Table hücresi bir `String`'tir. Kanonik satır `Pair<String, String>`,
satır koleksiyonu ise `List<Pair<String, String>>`'tir.

`Data`, **hiçbir** tür çıkarımı ve **hiçbir** örtük dönüşüm yapmaz: `"95"`,
`"3.14"` ve `"true"` `String` kalır; boş bir hücre `null` değil, boş bir
`String`'tir. v0.1.12'de Data'ya özgü eksik-değer modeli yoktur. Sayısal bir
değer, sıradan dönüşümlerle (§5.5) elde edilir:

```ahd
total: Int := int(row["score"])
```

### 53.2 Bir Table oluşturmak

```text
Data.fromRows(columns: List<String>, rows: List<List<String>>) -> Table
Data.fromRecords(records: List<Pair<String, String>>)          -> Table
Data.fromCSV(text: String, delimiter: String = ",")            -> Table
Data.readCSV(path: String, delimiter: String = ",")            -> Table
```

`Table` derleyici tarafından sağlanır ve tam olarak `DateTime` (§36) ve
`Pattern` (§49) gibi asla doğrudan oluşturulmaz; bir değer yalnızca bu
fonksiyonlardan veya başka bir Table işleminden gelir.

Sütun isimleri boş olmamalı ve benzersiz olmalıdır. `fromRows`, her satırın tam
olarak `len(columns)` hücreye sahip olmasını gerektirir ve asla doldurmaz,
kırpmaz veya isim uydurmaz. `fromRecords`, ilk kaydı kanonik sütun sırası
olarak alır ve sonraki her kaydın, herhangi bir ekleme sırasında, tam olarak
aynı anahtar kümesini taşımasını gerektirir; değerler kanonik sıraya kopyalanır
ve çağıranın Pair'leri değiştirilmez. Boş kayıtlar, hiçbir şema
çıkarılamayacağı için boş, sıfır sütunlu bir Table verir. Sıfır satır
geçerlidir ve şemayı korur.

`fromCSV` ve `readCSV`, CSV modülünün okuyucusunu yeniden kullanır; bu yüzden
Data ikinci bir CSV grameri tanımlamaz. İlk satır başlıktır ve
`CSV.parseRecords`'un aksine yalnızca başlıktan oluşan bir belge şemasını
korur: veri satırı olmayan `name,score`, `columns = ["name", "score"]` ve
`rowCount() == 0` verir. Boş girdi sıfır sütun ve sıfır satır verir.

### 53.3 Değiştirilemezlik

Her Table işlemi saftır. `head`, `tail`, `select`, `drop`, `rename`, `reverse`,
`filter`, `sort`, `transform` ve `derive`, her biri yeni bir Table döndürür ve
alıcıyı (receiver) değiştirmez. v0.1.12 hiçbir değiştiren üye yayınlamaz:
`setCell`, `appendRow`, `deleteRow` veya yerinde mod yoktur.

Table'ın depolaması yayınlanan bir öznitelik değildir. Okunamaz ve `has` (§29)
onu bildirmez; bu yüzden bir program, değiştirmek için arkadaki koleksiyonlara
ulaşamaz. Bir Table'ın döndürdüğü her değer taze bir anlık görüntüdür:
`columns()`, `rows()`, `row()` veya `column()` sonucunu değiştirmek Table'ı
asla etkilemez.

### 53.4 Table üyeleri

```text
rowCount()            -> Int
columnCount()         -> Int
columns()             -> List<String>
rows()                -> List<Pair<String, String>>
row(index: Int)       -> Pair<String, String>
column(name: String)  -> List<String>

head(count: Int = 5)  -> Table
tail(count: Int = 5)  -> Table
select(columns: List<String>) -> Table
drop(columns: List<String>)   -> Table
rename(oldName: String, newName: String) -> Table
reverse()             -> Table

filter(function: Function)                    -> Table
sort(column: String)                          -> Table
sort(function: Function)                      -> Table
transform(column: String, function: Function) -> Table
derive(name: String, function: Function)      -> Table

unique(column: String)      -> List<String>
valueCounts(column: String) -> Pair<String, Int>
groupBy(column: String)     -> Pair<String, Table>

toCSV(delimiter: String = ",")                  -> String
writeCSV(path: String, delimiter: String = ",") -> Nothing
```

`row`, §12.2'nin List indeks kurallarını izler: negatif indeks sondan sayar ve
geçersiz bir indeks `IndexError` fırlatır. `head`/`tail`, `rowCount()`'tan
büyük bir sayıyı kırpar, sıfır için aynı sütunlara sahip ve satırsız bir Table
döndürür ve negatif bir sayıyı reddeder. `select`, istenen sırayı çıktı sırası
olarak kullanır; `drop`, kalan sütunların özgün sırasını korur; ikisi de
bilinmeyen bir sütunu ve tekrarlanan bir isteği reddeder. `rename`, sütunun
konumunu korur.

`unique`, `valueCounts` ve `groupBy`, sütunun String hücresine göre anahtarlanır
ve ilk-görülme sırasını kullanır; bir grup içindeki satırlar kaynak sırasını
korur ve gruplanan her Table kaynak şemasına sahiptir. v0.1.12 hiçbir toplama
(aggregation) sözdizimi eklemez.

`toCSV` ve `writeCSV`, CSV modülünün yazıcısını kullanır ve sütun sırasında
başlığı ve ardından veri satırlarını yayar. Sıfır sütunlu, sıfır satırlı bir
Table `""` olarak serileşir.

### 53.5 Geri çağırma (callback) sözleşmeleri

Geri çağırma alan üyeler, §15 ve §50'nin sıradan Function/Lambda makinesini
yeniden kullanır; Data hiçbir çağrılabilir tür ve dinamik dispatch eklemez. Her
sözleşme sabittir ve statik olarak denetlenir:

| üye | sözleşme |
|---|---|
| `filter` | `(Pair<String, String>) -> Bool` |
| `sort` | `(Pair<String, String>) -> Int` \| `Real` \| `String` |
| `transform` | `(String) -> String` |
| `derive` | `(Pair<String, String>) -> String` |

Bir geri çağırma bir satır anlık görüntüsü alır, bu yüzden onu değiştirmek
Table'a ulaşamaz ve kaynak sırasında satır başına tam olarak bir kez çalışır.
`sort`, her iki biçimde de kararlı ve artandır ve anahtarını satır başına tam
olarak bir kez değerlendirir; bu, §12.5'in List anahtarlı sıralamasıyla
eşleşir. Karşılaştırıcı geri çağırma ve azalan biçim yoktur; azalan sıra,
negatiflenmiş sayısal bir anahtarla veya `reverse()` ile yazılır. Bir geri
çağırma başarısızlığı sıradan bir AhdCode Error'u olarak yayılır ve kaynak
Table'ı değiştirmeden bırakır. §50'nin lambda kısıtlamaları, lexical `Local`
yakalamanın yokluğu dahil, yürürlükte kalır.

### 53.6 DataError

`DataError` doğrudan `Error`'dan türer. Data'ya özgü yapısal başarısızlıkları
kapsar: tekrarlanan veya boş sütun ismi, satır genişliği uyuşmazlığı, kayıt
anahtar kümesi uyuşmazlığı, bilinmeyen sütun, tekrarlanan `select`/`drop`
isteği, negatif `head`/`tail` sayısı ve zaten var olan bir `derive` hedefi.

Diğer alanlar kendi hata kimliğini korur; böylece bir tanılama başarısız olan
katmanı isimlendirir: CSV sözdizimi ve geçersiz ayraç `CSVError` kalır,
`readCSV`/`writeCSV` dosya sistemi erişimi `FileError`/`IOError` kalır ve
geçersiz bir `row` indeksi `IndexError` kalır. Hiçbir Go paniği, Go tür ismi
veya stack trace ifşa edilmez.

### 53.7 Bir değer olarak Table

`Table` sıradan bir Class referansıdır. `type(table)`, `"Table"` bildirir
(§48.1); `id()` ve `same`, diğer herhangi bir referanstaki gibi davranır.
Table hiçbir Class Protocol Method (§47) uygulamaz; bu yüzden `==` ve `same`
referans kimliğini korur; Data, tablolar için değer eşitliği tanımlamaz.

### 53.8 Bu sürümde olmayanlar

`Data`, kasıtlı olarak join, merge, concat, pivot, melt, MultiIndex, indeks
etiketleri, sorgu dizeleri, SQL, pencere fonksiyonları, rolling, resample,
kategorik dtype, tembel yürütme veya ifade ağaçlarına sahip değildir. Şema
çıkarımı, otomatik sayısal veya tarih ayrıştırma ve null çıkarımı yoktur.

İstatistik de yoktur: `sum`, `mean`, `median`, `variance`, `stdev`, `quantile`,
`correlation` ve `describe`, daha sonraki bir Statistics olanağına aittir; o
olanağın, açık bir dönüşümle üretilen `List<Int>` ve `List<Real>` tüketmesi
amaçlanmaktadır, böylece Data asla dinamik tipli olmaya zorlanmaz.

---

## 54. Açık Lambda Bağımlılıkları (v0.1.13)

Bir lambda hâlâ tek bir ifadedir (§50). v0.1.13; lambda içine blok gövde,
deyim, `return` veya yerel bildirim eklemez; deyimli gövde isimli Function'ın
işi olarak kalır. Eklediği şey, o tek ifadenin kendi parametrelerinin dışından
seçili bağlamaları okuyabilmesidir: çevreleyen bir lexical bağlama, veya bir
modül/global bağlaması.

### 54.1 Sözdizimi

`lambda` ile parametre listesi arasına isteğe bağlı bir bağımlılık listesi
yazılır. Her girdi kendi türünü, ister kısaca ister tam olarak, belirtir:

```text
lambda [ bağımlılık, bağımlılık, ... ] ( <tipli parametreler> ) -> <ifade>

bağımlılık := "#" isim | "Local" isim
            | "@" isim | "Global" isim
```

```ahd
lambda [#minimum] (score: Int) -> score >= minimum
lambda [#low, #high] (value: Int) -> value >= low and value <= high
lambda [@Maximum] (score: Int) -> score <= Maximum
lambda [#minimum, @Maximum] (score: Int) -> score >= minimum and score <= Maximum
lambda [Local minimum, Global Maximum] (score: Int) -> score >= minimum and score <= Maximum
```

`#name` ve `Local name` aynı bağımlılığın iki yazımıdır; `@name` ve
`Global name` de öyledir. Bir bağımlılık listesi kısa ve uzun yazımları
serbestçe karıştırabilir; formatter, kaynağın hangi yazımı kullandığını
korur, birini diğerine dönüştürmez. Yinelenen bağımlılık denetimi `#x` ve
`Local x`'i tek bir girdi sayar; bu yüzden ikisini birden listelemek iki
bağımlılık değil, bir yinelenen-bağımlılık hatasıdır.

Başka bir yazım yoktur: yalın bir isim (`lambda [minimum] (...)`) reddedilir.
Her girdi, çevreleyen bir lexical bağlamaya mı yoksa bir modül/global
bağlamasına mı bağımlı olduğunu -- ve nasıl -- belirtmelidir; böylece bir
lambda'nın neye bağımlı olduğu lambda'nın yazıldığı yerde görünür olur.

Listeyi tamamen atlamak veya `lambda [] (...)` yazmak, lambda'nın kendi
parametrelerinin dışında hiçbir şey okumadığı anlamına gelir; her iki biçim de
v0.1.10'dan değişmemiştir, bu yüzden v0.1.13'ten önce yazılmış her lambda
anlamını korur.

### 54.2 Yerel yakalama (`#name` / `Local name`)

`#name` (veya `Local name`) çevreleyen bir lexical bağlamayı okur: bir Function
parametresi, bir `Local` veya bir `for`/`except` bağlaması. Yakalama asla
çıkarılmaz. Böyle bir bağlamayı listelemeden okumak, bağlamayı isimlendiren bir
derleme zamanı hatasıdır; böylece bir lambda'nın lexical bağımlılıkları
lambda'nın yazıldığı yerde görünür olur:

```ahd
run: Function := (
) -> Bool {
    minimum: Local Int := 70
    check: Local Function := lambda (score: Int) -> score >= minimum
    return check(80)
}
```

reddedilir; aynı lambda `lambda [#minimum] (...)` (veya
`lambda [Local minimum] (...)`) olarak yazıldığında kabul edilir.

Yakalama **değere göredir** ve lambda değerinin oluşturulduğu yerde bir kez
değerlendirilir. Çevreleyen bağlamadaki sonraki bir değişiklik lambda içinde
görünmez:

```ahd
step: Local Int := 1
first: Local Function := lambda [#step] (x: Int) -> x + step
step = step + 100
second: Local Function := lambda [#step] (x: Int) -> x + step
// first(0) 1'dir; second(0) 101'dir
```

Yakalanan değer dilin sıradan değer ve referans kurallarına uyar (§11): bir
`List`, `Pair` veya Class örneğini yakalamak referansı kopyalar; bu yüzden
referans verilen nesne, tam olarak onu parametre olarak geçirmekteki gibi
paylaşılır. `Constant` derin dondurma anlambilimi etkilenmez.

Yakalanan bir isim lambda içinde salt okunurdur: `#`/`Local` çevreleyen değeri
verir, çevreleyen değişkenin sahipliğini değil. v0.1.13 değiştirilebilir bir
closure hücresi, referans kutusu, `Ref` veya `Cell` tanıtmaz.

Yakalanan bir değer, çevreleyen çağrı döndükten sonra da geçerli kalır; bu
yüzden bir lambda değeri onu oluşturan çerçeveden daha uzun yaşayabilir.

### 54.3 Global bağımlılığı (`@name` / `Global name`)

`@name` (veya `Global name`) bir yakalama değildir: lambda'nın bir
modül/global bağlamasını kasıtlı olarak okuduğuna dair açık bir bildirimdir;
bu, sıradan bir Function'ın modül durumuna dokunmak için zaten ihtiyaç duyduğu
`Global` bildirimini yansıtır (§30). Bir lambda'nın bu bildirimi yazabileceği
bir deyim gövdesi yoktur, bu yüzden aynı niyeti belirttiği yer bağımlılık
listesidir:

```ahd
Maximum: Int := 100

check: Function := lambda [@Maximum] (score: Int) -> score <= Maximum
```

`@name`, `Maximum`'u closure depolamasına kopyalamaz, modül bağlamasını
anlık görüntülemez ve onu bir `Local`'e dönüştürmez. Gerçek modül bağlamasını,
dilin mevcut global-mutasyon kuralları altında, tam olarak sıradan bir
Function'ın `x: Global Type` bildirimi gibi okur:

```ahd
Maximum: Int := 100

check: Function := lambda [@Maximum] (score: Int) -> score <= Maximum

check(50)  // true, Maximum 100'dür
Maximum = 40
check(50)  // false, @Maximum canlı bağlamayı gözlemler, anlık görüntü değil
```

Bir lambda içinde bir modül bağlamasını, bağımlılığı bildirmeden okumak,
listelenmemiş bir `#`/`Local` yakalaması gibi bir derleme zamanı hatasıdır.
`Confidential` görünürlüğü ve dilin diğer modül-erişim kuralları etkilenmez:
`@` bunları ne atlar ne de ikinci bir global-durum modeli tanıtır.

### 54.4 Neler listelenebilir

`#`/`Local` yalnızca çevreleyen bir lexical bağlamayı isimlendirir;
`@`/`Global` yalnızca modül kökündeki bir değer bağlamasını isimlendirir.
Bir isim için yanlış türü kullanmak -- çevreleyen bir Local için `@`, ya da
bir modül bağlaması için `#` -- sessizce yeniden yorumlanmak yerine ayrı bir
tanılamadır.

Modül kökündeki bir Class, isim uzayı veya Function bildirimi sıradan aramayla
erişilir ve her iki türde de listelenmemelidir; birini listelemek hatadır. Bu,
sıradan bir Function için geçerli olan kuralı yansıtır -- o da bir modül
Function'ını çağırmak veya bir Class'a başvurmak için `Global` bildirimine
ihtiyaç duymaz -- bir lambda'nın bağımlılık listesi bir Function'ınkinden daha
geniş bir kural değildir, yalnızca aynı kuralın daha kısa bir yazımıdır.

Bir bağımlılık her iki yazım altında da tekrarlanmamalı ve bir lambda
parametresi ismiyle çakışmamalıdır; her biri ayrı bir tanılamadır.

### 54.5 Tipleme ve uygulama

Her `#`/`Local` yakalaması statik olarak çözülür ve çevreleyen bağlamanın tam
türünü korur; bu yüzden closure'lar dinamik bir ortam ve `Any` tanıtmaz.
Yakalanan bağlamalar yükseltilmiş (lifted) çağrılabilirin baştaki
parametreleri olur; bu, closure depolamasını ikinci bir mekanizma yerine
sıradan tipli parametre geçişi yapar. Çağrılabilirin yayınlanan imzası hâlâ
yalnızca bildirilen parametrelerini tanımlar, bu yüzden çağıranlar ve callback
adaptörleri etkilenmez.

Her `@`/`Global` bağımlılığı, sıradan bir Function'ın `Global` bildiriminin
kullandığı aynı takma ad (alias) mekanizmasını kullanarak gerçek modül
bağlamasına bir takma ad kurar; bu yüzden ayrı bir IR veya çalışma zamanı
temsiline ihtiyaç duymaz: lambda, modül bağlamasını tam olarak bir Function'ın
okuyacağı gibi okur.

Yerel arka uç ve kalıcı REPL değerlendiricisi aynı modeli uygular ve aynı
sonuçları üretir. Bağımlılıklar mevcut her callback ile birlikte çalışır; buna
`List.map`/`filter`/`sort` ve §53.5'teki Data callback'leri dahildir.

Formatter bağımlılık listesini, her girdinin seçtiği yazımı koruyarak render
eder ve idempotenttir.

---

## 55. Statistics Standart Modülü (v0.1.13)

`Statistics`, `Math`, `Time`, `Regex`, `CSV` ve `Data` gibi açıktır (§33):
kullanılmadan önce içe aktarılmalıdır ve kanonik kimliği
`builtin:Statistics`'tir, bu yüzden kardeş bir `Statistics.ahd` onu
gölgeleyemez.

```ahd
bring Statistics
from Statistics bring StatisticsError
```

Tipli sayısal List'ler üzerinde betimleyici istatistiktir ve kasıtlı olarak
`Data`'ya bağımlı değildir: bir Table hücresi `String`'tir (§53.1), bu yüzden
bir program bir istatistik istemeden önce açıkça dönüştürür. Bu, her iki modülü
de dinamik bir sayısal değer tanıtmak yerine katı tutar.

### 55.1 Yüzey ve tipleme

Her fonksiyon açık bir `Int`/`Real` aşırı yükleme çifti olarak yayınlanır ve
sıradan aşırı yükleme makinesiyle (§16) çözülür; bu yüzden bir sonucun statik
türü her zaman bilinir ve zayıf tipli bir giriş noktası yoktur.

```text
sum(values: List<Int>)   -> Int      sum(values: List<Real>)   -> Real
min(values: List<Int>)   -> Int      min(values: List<Real>)   -> Real
max(values: List<Int>)   -> Int      max(values: List<Real>)   -> Real
range(values: List<Int>) -> Int      range(values: List<Real>) -> Real
mode(values: List<Int>)  -> Int      mode(values: List<Real>)  -> Real

mean(values)           -> Real
median(values)         -> Real
variance(values)       -> Real
sampleVariance(values) -> Real
stdDev(values)         -> Real
sampleStdDev(values)   -> Real
quantile(values, probability: Real) -> Real
```

Cevabı girdinin kendi değerlerinden biri olan bir istatistik eleman türünü
korur; ortalama alan veya dağılım ölçen bir istatistik her zaman `Real`'dir.
`Int` sonuçlar dilin denetimli aritmetiğini kullanır; bu yüzden aralık dışı bir
`Int` toplamı veya aralığı sarmalamak yerine `OverflowError` fırlatır.

Örtük String dönüşümü yoktur: `Statistics.mean(["10", "20"])` derlenmez.

### 55.2 Tanımlar

`median`, sıralanmış verinin ortadaki değeridir; çift sayıda değer için
ortadaki ikisinin ortalamasını alır ve her zaman `Real`'dir.

`variance` ve `stdDev` **popülasyon** biçimleridir (`n`'e böler);
`sampleVariance` ve `sampleStdDev` **örneklem** biçimleridir (`n - 1`'e böler).
Tanımın asla örtük kalmaması için iki isim de vardır.

`mode`, en sık görülen değerdir; eşitlik girdideki ilk geçişe göre çözülür, bu
yüzden sonuç asla map yineleme sırasına bağlı değildir.

`quantile`, sıra istatistikleri arasında doğrusal interpolasyon yapar: veri
artan sırada iken konum `probability * (n - 1)`'dir ve kesirli bir konum
komşuları arasında interpolasyon yapar. `probability`, `0.0..1.0` aralığında
olmalıdır; başka bir şey kırpma değil hatadır. `0.0` minimum, `1.0`
maksimumdur ve tek değerli bir List kendi quantile'ıdır.

### 55.3 Boş ve tanımsız girdi

Boş bir List'in `sum`'ı toplamsal birim öğedir (`0`/`0.0`); toplamları
birleştirilebilir tutan tek toplam budur. Diğer her istatistik boş bir List
için tanımsızdır ve `StatisticsError` fırlatır. `sampleVariance` ve
`sampleStdDev` ek olarak en az iki değer gerektirir.

`StatisticsError` doğrudan `Error`'dan türer ve yalnızca girdisi için tanımsız
olan bir istatistik için kullanılır; Data, CSV veya dosya sistemi
başarısızlıkları için asla yeniden kullanılmaz.

Bir istatistik asla `NaN` veya sonsuzluk vermez: §39'un sonlu-`Real`
sözleşmesi korunur ve bunun yerine `StatisticsError` bildirilir.

### 55.4 Değiştirilemezlik

Bir istatistik girdisini asla değiştirmez. `median` veya `quantile` için
sıralama bir anlık görüntü üzerinde çalışır, bu yüzden çağıranın List'i sırasını
korur. Yerel arka uç ve kalıcı REPL değerlendiricisi her sonuçta ve her hatada
uyuşur.

### 55.5 Bu sürümde olmayanlar

v0.1.13 yalnızca betimleyici istatistiktir: çıkarımsal test, regresyon,
dağılım, rastgele örnekleme veya grafik çizimi yoktur. `frequency` fonksiyonu
yoktur, çünkü bir frekans tablosu `Pair<K, Int>` olurdu ve bir Pair anahtarı
`String`, `Int` veya `Bool` olmak zorundadır (§13.3); bu yüzden `List<Real>`
girdisinin ifade edilebilir bir sonucu yoktur. `mode` ve `Table.valueCounts`
(§53.4) yaygın ihtiyaçları karşılar.

---

## 56. Plot Standart Modülü (v0.1.14)

`Plot`, `Math`, `Time`, `Regex`, `CSV`, `Data` ve `Statistics` gibi açıktır
(§33, §55): kullanılmadan önce içe aktarılmalıdır ve kanonik kimliği
`builtin:Plot`'tur, bu yüzden kardeş bir `Plot.ahd` dosyası onun yerini
alamaz (shadow edemez).

```ahd
bring Plot
from Plot bring Chart
from Plot bring Figure
from Plot bring PlotError
```

Statistics gibi, Plot da kasıtlı olarak Data'ya bağımlı değildir: bir Table
hücresi bir `String`'dir (§53.1), bu yüzden bir program bir sütunu
çizmeden önce açıkça dönüştürür.

### 56.1 Genel tipler

Plot, yüzeyinin ihtiyaç duyduğu en küçük nesne modelini tanıtır: `Chart`,
`Figure` ve `PlotError`. Tek bir grafik -- line, scatter, bar, histogram,
box veya error bar -- bir `Chart`'tır. Çoklu-grafik kompozisyonu (§56.7)
bir `Figure`'dır. Hiçbiri doğrudan oluşturulamaz; ikisi de yalnızca Plot'un
modül fonksiyonları ve `Chart` için kendi metotları tarafından üretilir:

```ahd
Plot.line(x, y)                                -> Chart
Plot.scatter(x, y)                             -> Chart
Plot.bar(labels: List<String>, values)         -> Chart
Plot.histogram(values, bins: Int)              -> Chart
Plot.box(values)                               -> Chart
Plot.errorBar(x, y, lowerErrors, upperErrors)  -> Chart
Plot.new()                                     -> Chart
Plot.subplots(rows: Int, columns: Int, charts: List<Chart>) -> Figure
```

### 56.2 Katı sayısal girdi

Her sayısal argüman, bağımsız olarak `List<Int>` veya `List<Real>` kabul
eder; bu, sıradan overload çözümlemesiyle (§16) çözülür ve bir `Int` List
dahili olarak güvenle `Real`'a genişletilir -- Statistics'in genişletmesiyle
aynıdır (§55.1). Bir `List<String>` -- rakam metni tutsa bile -- asla kabul
edilmez; Plot hiçbir String-to-number zorlaması (coercion) tanıtmaz ve Data
entegrasyonu, tıpkı Statistics için olduğu gibi açık kalır (§55, §53.1):

```ahd
scores: List<Int> := table.column("score").map(
    lambda (value: String) -> int(value)
)

chart := Plot.histogram(scores, 10)
```

Her grafik oluşturucusu ve `Chart.line`/`Chart.scatter`, boş sayısal girdi
için `PlotError` fırlatır -- `Statistics.mean([])`'in aldığı aynı alan
(domain) hatası muamelesi (§55.3).

### 56.3 Grafik meta verisi ve değiştirilemezlik

```text
chart.title(text: String)   -> Chart
chart.xLabel(text: String)  -> Chart
chart.yLabel(text: String)  -> Chart
chart.legend(enabled: Bool) -> Chart
chart.size(width: Int, height: Int) -> Chart
```

Her Chart metodu saftır (pure): yeni bir Chart döndürür ve alıcısını
(receiver) asla değiştirmez -- her Table işleminin zaten kullandığı kural
(§53.5). Yapılandırma, yerinde değişiklik yerine yeniden atama yoluyla
zincirlenir:

```ahd
chart := Plot.line(x, y)
chart = chart.title("Experiment")
chart = chart.xLabel("Time")
```

`size`, pozitif çıktı boyutlarını ayarlar; bir Chart'ın varsayılan boyutu
800x600'dür. Her Plot fonksiyonu ve Chart metodu, List argümanlarının bir
anlık görüntüsünü (snapshot) okur ve çağıranın List'ini asla yeniden
sıralamaz veya başka bir şekilde değiştirmez.

### 56.4 Birden çok seri

`chart.line(x, y, label)` ve `chart.scatter(x, y, label)`, bir Chart'a bir
seri daha ekler; böylece bir line ve bir scatter serisi -- veya ikisinden
birkaçı -- bir legend ile tek bir Chart'ı paylaşabilir. `Plot.line(x, y)`/
`Plot.scatter(x, y)`, etiketsiz tek bir seriyle bir Chart başlatmanın
kısayoludur; `chart.line`/`chart.scatter` onu genişletir veya bu şekilde
zaten oluşturulmuş bir Chart'ı genişletir. Bir `bar`, `histogram`, `box`
veya `errorBar` Chart'ına bir seri eklemek `PlotError` fırlatır: bu grafik
türleri kendi kendine yeterlidir.

### 56.5 Save (kaydetme)

```text
chart.save(path: String) -> Nothing
figure.save(path: String) -> Nothing
```

Çıktı biçimi dosya uzantısından çıkarılır. Desteklenen biçimler PNG
(`.png`), SVG (`.svg`) ve PDF'dir (`.pdf`); başka herhangi bir uzantı
`PlotError` fırlatır. Göreli bir yol, programın çalışma dizinine göre
çözülür -- File'ın kullandığı aynı kural. Bir render veya dosya sistemi
hatası, ham bir arka uç hatası değil, her zaman `PlotError` fırlatır.

### 56.6 Show (gösterme)

```text
chart.show() -> Nothing
figure.show() -> Nothing
```

`show()`, AhdCode'a özgü bir geçici alanda benzersiz bir geçici PNG'ye
render eder ve onu platformun standart görüntü açma mekanizmasıyla açar
(macOS'ta `open`, Linux'ta `xdg-open`, Windows'ta kabuğun `start` komutu);
böylece bir grafiği incelemek asla elle kaydedip dosyayı bulmayı
gerektirmez. Geçici görüntü otomatik olarak silinmez, çünkü harici
görüntüleyici `show()` döndükten sonra da onu okumaya devam eder. `show()`
bir masaüstü oturumu gerektirir; başsız (headless) bir ortam, askıda kalmak
yerine temiz bir şekilde `PlotError` ile başarısız olur.

### 56.7 Subplot'lar

```ahd
figure := Plot.subplots(
    2, 2,
    [
        Plot.line(x1, y1),
        Plot.scatter(x2, y2),
        Plot.histogram(values, 10),
        Plot.box(values)
    ]
)

figure.show()
figure.save("summary.pdf")
```

`charts` satır-öncelikli (row-major) sıradadır. `rows` ve `columns`'ın her
ikisi de pozitif olmalıdır ve grafik sayısı `rows * columns`'ı aşamaz; tam
bir sayı gerektirmek yerine, hücrelerden daha az grafiğe izin verilir ve
kalan hücreler boş bırakılır. Bir `Figure`, `Plot.subplots` tarafından
üretilen açık, immutable bir değerdir -- v0.1.14'te mutable global bir
"geçerli subplot" durumu yoktur. Bir Figure'ın save/show boyutu, grid
boyutlarından belirlenimci (deterministic) şekilde türetilir; v0.1.14 bir
`Figure.size` yayımlamaz.

### 56.8 PlotError

`PlotError`, doğrudan `Error`'dan türer (§7, §55.3'ün örüntüsü). Plot, her
plot'a özgü çalışma zamanı hatası için onu fırlatır: eşleşmeyen `x`/`y`
uzunlukları, boş grafik verisi, geçersiz bir bin sayısı, eşleşmeyen bar
etiketleri/değerleri, eşleşmeyen error-bar verisi, negatif hata
büyüklükleri, desteklenmeyen bir çıktı biçimi, geçersiz subplot boyutları,
subplot hücrelerinden daha fazla grafik, bir render hatası, bir geçici
dosya hatası ve bir görüntüleyici-açma hatası. Statik bir tip uyuşmazlığı,
sıradan bir derleme-zamanı tanılaması olarak kalır; `PlotError` yalnızca
tip denetleyicisinin önceden eleyemediği şeyleri kapsar.

### 56.9 Bu sürümde olmayanlar

v0.1.14 tam olarak altı grafik ailesini destekler: line, scatter, bar,
histogram, box ve error bar. Pie, heatmap, contour, violin, stem, polar,
3D, candlestick, area veya surface grafiği yoktur ve keyfi özel plotter
enjeksiyonu yoktur. Bir `Numeric` tipi yoktur -- sayısal esneklik yalnızca
`Int`/`Real` genişletmesidir, Statistics ile eşleşir -- genel bir GUI
çerçevesi yoktur ve ikincil eksenler yoktur.

---

# AhdCode v0.1 Çekirdek Spesifikasyonu Sonu
