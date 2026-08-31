# JSON standart modülü

[English](JSON.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [XML](XML_TR.md)

`JSON`, derleyici tarafından kayıtlı `builtin:JSON` modülüdür. Açıktır ve
kardeş bir `JSON.ahd` dosyası onu gölgeleyemez:

```ahd
bring JSON
from JSON bring JSONValue
from JSON bring JSONError
```

JSON, AhdCode'a `Any`, dinamik tipleme veya reflection getirmez. `JSONValue`,
kapalı, statik tipli, değiştirilemez, özyinelemeli bir değer modelidir; her
işlemin sabit, bildirilmiş bir tipi vardır ve bir değer asla sessizce
ilgisiz bir tipe dönüşmez.

## JSONValue modeli

`JSONValue`, JSON'un kendisinin tanımladığı yedi türü tam olarak temsil eder:

```text
Null
Bool
Int
Real
String
Array   (bir List<JSONValue>)
Object  (bir Pair<String, JSONValue>, ekleme sırası korunur)
```

Sekizinci bir tür yoktur ve açık bir genişletme noktası yoktur. Bir
`JSONValue` değiştirilemezdir: bir `JSONValue` (veya onların
`List<JSONValue>`/`Pair<String, JSONValue>` biçimini) döndüren her erişimci
taze, bağımsız bir değer döndürür; alıcıya bir takma ad asla değil.

## Ayrıştırma ve okuma

```text
JSON.parse(source: String) -> JSONValue
JSON.read(path: String)    -> JSONValue
```

Bir JSON belgesi tam olarak bir üst düzey değer içermelidir; sondaki
boşluk-olmayan içerik reddedilir. Object anahtarları yinelenme kontrolünden
geçer: `{"a":1,"a":2}`, son değeri sessizce tutmak yerine `JSONError`
fırlatır. Object ekleme sırası ve Array sırası her zaman korunur. Standart
JSON kaçış dizileri (`\"`, `\\`, `\/`, `\b`, `\f`, `\n`, `\r`, `\t`, ve
surrogate çiftleri dahil `\uXXXX`) desteklenir. `NaN` ve `Infinity` JSON
literalleri değildir ve diğer bozuk girdiler gibi reddedilir.

Sayı literalleri tam olarak iki türde okunur:

```text
91      -> Int
-12     -> Int
3.14    -> Real
1e3     -> Real
1.0     -> Real
```

Bir lekseme, bir kesir (`.`) veya üs (`e`/`E`) içeriyorsa tam olarak
`Real`dir; aksi halde `Int`tir. AhdCode'un `Int` aralığına sığmayan bir tam
sayı lekseme'si `JSONError` fırlatır — asla sessizce bir `Real`e veya başka
bir tipe dönüşmez. Ayrıştırıldıktan sonra sonlu olmayan bir real literal de
(etkin biçimde sonsuz) aynı şekilde reddedilir.

Ayrıştırma sınırlandırılmıştır: 8&nbsp;MiB'den büyük girdi ve 256 seviyeden
derin Array/Object iç içeliği, tamamlanmadan önce `JSONError` fırlatır.

AhdCode'un sıradan tırnaklı String'leri `{` ve `}`'yi interpolation
sınırlayıcısı olarak yorumladığından, literal JSON metnini bir raw String
olarak yazın; böylece süslü parantezler sıradan karakter olur:

```ahd
document: JSONValue := JSON.parse(r'{"a":1}')
```

(Bir raw String'in kaçış mekanizması yoktur, bu yüzden kendi
sınırlayıcısını içeremez — her zaman `"` içeren JSON metni için `r'...'`
kullanın.)

## Oluşturma

`JSONValue`, sıradan bir `String`, `Int`, `Real`, `Bool`, `List` veya
`Pair`'in örtük dönüşümüyle asla oluşturulmaz. Her `JSONValue`, açık bir JSON
Function'ıyla inşa edilir:

```text
JSON.nullValue()                              -> JSONValue
JSON.fromBool(value: Bool)                    -> JSONValue
JSON.fromInt(value: Int)                      -> JSONValue
JSON.fromReal(value: Real)                    -> JSONValue
JSON.fromString(value: String)                -> JSONValue
JSON.array(values: List<JSONValue>)           -> JSONValue
JSON.object(values: Pair<String, JSONValue>)  -> JSONValue
```

(`JSON.null()` değil `JSON.nullValue()`: `null` her sözdizimsel bağlamda
saklı bir anahtar kelimedir, bu yüzden `.` sonrası bir üye adı olarak asla
yazılamaz.)

`JSON.fromReal`, sonlu olmayan bir `Real`i (NaN veya sonsuz) `JSONError` ile
reddeder — bunu üretebilecek AhdCode aritmetiği, JSON'a vermeden önce açıkça
dönüştürülmelidir, asla örtük olarak değil.

```ahd
student: JSONValue := JSON.object({
    "name": JSON.fromString("Ali")
    "score": JSON.fromInt(91)
    "active": JSON.fromBool(true)
})
```

## Erişimciler

```text
kind()   -> String
isNull() -> Bool

bool()   -> Bool
int()    -> Int
real()   -> Real
string() -> String

array()  -> List<JSONValue>
object() -> Pair<String, JSONValue>

get(key: String)   -> JSONValue?
at(index: Int)     -> JSONValue
```

`kind()`, tam olarak `"Null"`, `"Bool"`, `"Int"`, `"Real"`, `"String"`,
`"Array"` veya `"Object"`'ten birini döndürür.

`get` dışındaki her erişimci, alıcının türü uyuşmadığında `JSONError`
fırlatır:

```ahd
attempt {
    JSON.fromString("x").int()
} except JSONError as error {
    write(error.message)
}
```

`real()`, "yanlış tür `JSONError` fırlatır" kuralının bilinçli tek
istisnasıdır: bir `Int` alıcısını da kabul eder ve onu `Real`e genişletilmiş
olarak döndürür — AhdCode'un başka yerlerde zaten uyguladığı aynı güvenli
`Int -> Real` genişlemesi. `int()` asla tersini yapmaz — tam sayı gibi
görünen bir değere sahip bir `Real` (`5.0`) yine de `int()`'ten `JSONError`
fırlatır.

`get(key)` yalnızca `Object` içindir (bir `Object`-olmayan alıcı, diğer her
yanlış-tür erişimi gibi `JSONError` fırlatır) ve `JSONValue?` döndürür:
`null`, anahtarın yok olduğu anlamına gelir, anahtarın değerinin JSON'un
kendi `Null`'u olduğu anlamına asla gelmez — `JSON.nullValue()` tutan mevcut
bir anahtar, `kind()`'ı `"Null"` olan null-olmayan bir `JSONValue` döndürmeye
devam eder.

`at(index)` yalnızca `Array` içindir ve sıradan List indeks kurallarını
izler (negatif bir indeks sondan geriye sayar); aralık dışı bir indeks
`JSONError` fırlatır.

```ahd
name := parsed.get("name")
if name != null {
    write(name.string())
}
```

## Serileştirme

```text
JSON.stringify(value: JSONValue, pretty: Bool = false) -> String
JSON.write(value: JSONValue, path: String, pretty: Bool = false) -> Nothing
```

Kompakt çıktının (`pretty = false`, varsayılan) önemsiz boşluğu yoktur.
Pretty çıktı sabit iki boşluklu bir girinti kullanır. Her iki mod da:

- Array sırasını ve Object ekleme sırasını korur;
- String içeriğini doğru şekilde kaçışlar;
- asla `NaN` veya `Infinity` yaymaz (pratikte erişilemez, çünkü
  `JSON.fromReal` bunları zaten oluşturma anında reddeder);
- geçerli JSON üretir, deterministik biçimde — aynı `JSONValue`'yu iki kez
  serileştirmek her zaman aynı metni üretir ve `parse(stringify(value))`,
  `value`'ya anlamsal olarak eşittir.

`JSON.write`, çıktısını Word'ün `.docx` için kullandığı aynı
temp-dosya-sonra-yeniden-adlandır kuralıyla aşamalı olarak hazırlar ve
atomik olarak yayımlar: başarısız bir yazma, hedefte zaten bulunan bir
dosyayı asla bozmaz.

```ahd
text: String := JSON.stringify(student, true)
JSON.write(student, "student.json", true)
```

## Hatalar

`JSONError` doğrudan `Error`'dan türer ve her JSON'a özgü hatayı kapsar:
bozuk girdi, yinelenen anahtarlar, sondaki içerik, derinlik/boyut sınırları,
tam sayı taşması, sonlu olmayan bir `Real`, yanlış türde bir erişimci, aralık
dışı bir `Array` indeksi ve eksik/okunamayan/yazılamayan bir dosya.
`JSON.read`/`JSON.write`, `FileError` yerine doğrudan `JSONError` fırlatır;
böylece tüm modülü uçtan uca tek bir hata türü kapsar.

```ahd
attempt {
    JSON.read("missing.json")
} except JSONError as error {
    write(error.message)
}
```

## Güvenlik ve sınırlar

`JSON.parse`/`JSON.read` asla ağa erişmez ve hiçbir şey çalıştırmaz; yalnızca
JSON grameri üzerinde yürürler. Girdi 8&nbsp;MiB'den sonra, Array/Object iç
içeliği 256 seviyeden sonra reddedilir — her iki sınır da bu sürümün
uygulama ayrıntılarıdır ve gözden geçirilebilir, ancak bozuk veya kötü niyetli
bir belgenin patolojik özyinelemeye veya sınırsız bellek kullanımına neden
olmasını önlemek için her zaman uygulanır.

## Kapsam dışı

JSON, bir şema veya sorgu dili değil, tipli bir veri değişim modülüdür: JSON
Schema/JSON Pointer/JSONPath desteği, akış (streaming) ayrıştırıcı veya
`Any`/dinamik kaçış yolu yoktur. Çözülmüş veriler üzerinde filtreleme,
sıralama veya türetme gerektiğinde, önce açıkça `Data`'nın `Table`'ına veya
sıradan AhdCode değerlerine dönüştürün.
