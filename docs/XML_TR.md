# XML standart modülü

[English](XML.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [JSON](JSON_TR.md)

`XML`, derleyici tarafından kayıtlı `builtin:XML` modülüdür. Açıktır ve
kardeş bir `XML.ahd` dosyası onu gölgeleyemez:

```ahd
bring XML
from XML bring XMLNode
from XML bring XMLDocument
from XML bring XMLError
```

XML, bir XML standartları paketi değil, yapılandırılmış veri desteğidir:
tam bir DOM değil, küçük ve sınırlandırılmış bir node modelidir; içinde
hiçbir yerde `Any` veya dinamik kaçış yolu yoktur.

## Node modeli

`XMLNode`, tam olarak iki türü temsil eder:

```text
Element
Text
```

Bu bilinçli bir tercihtir: küçük, kapalı bir küme, büyük bir DOM Class
hiyerarşisi olmadan sıralı, karışık text/element içeriğini kapsar. Comment,
CDATA, ProcessingInstruction, Doctype veya Entity için ayrı bir herkese açık
tür yoktur — ayrıştırma comment'leri ve processing instruction'ları yok
sayar, ve bir `CDATA` bölümü diğer her metin gibi sıradan bir `Text` node'u
olarak kurtarılır.

`XMLDocument`, tam olarak bir kök `XMLNode`'u sarar; bu kök bir `Element`
olmalıdır. `XMLNode` ve `XMLDocument` her ikisi de değiştirilemezdir: hiçbir
erişimci alıcıya değiştirilebilir bir takma ad açığa çıkarmaz ve node verisi
döndüren her erişimci taze, bağımsız bir değer döndürür.

## Oluşturma

```text
XML.text(value: String)                                        -> XMLNode
XML.element(name: String, attributes: Pair<String,String>, children: List<XMLNode>) -> XMLNode
XML.document(root: XMLNode)                                     -> XMLDocument
```

`XML.document(root)`, bir `Element` kökü gerektirir; bir `Text` node'u
geçirmek `XMLError` fırlatır. Sıradan oluşturma için String birleştirmeye
gerek yoktur:

```ahd
student: XMLNode := XML.element(
    "student"
    {"id": "42"}
    [
        XML.element("name", {}, [XML.text("Ali")])
        XML.element("score", {}, [XML.text("91")])
    ]
)
document: XMLDocument := XML.document(student)
```

`XML.element`, yalnızca niteliksiz (namespace'siz) element'ler oluşturur —
namespace-nitelikli bir element üretmek, bu sürümde yalnızca mevcut
namespace-nitelikli XML'i ayrıştırarak mümkündür, oluşturarak değil (bkz.
[Namespace'ler](#namespaceler)).

## Ayrıştırma ve okuma

```text
XML.parse(source: String) -> XMLDocument
XML.read(path: String)    -> XMLDocument
```

Ayrıştırma, Go standart kitaplığının `encoding/xml` decoder'ını kullanır —
`Word.read`'in DOCX için kullandığı aynı token-yürüme yaklaşımı — elle
yazılmış bir gramer değil. Bir belge tam olarak bir kök element'e sahip
olmalıdır: kök yok, birden fazla üst düzey element, veya kökten önce/sonra
boşluk-olmayan içerik — hepsi `XMLError` fırlatır. Bir element üzerinde
yinelenen bir öznitelik (attribute) geçersiz XML'dir ve `XMLError` fırlatır.
Çocuk sırası ve karışık text/element sırası her zaman ayrıştırıldığı gibi
tam olarak korunur.

AhdCode'un sıradan tırnaklı String'leri `{` ve `}`'yi interpolation
sınırlayıcısı olarak yorumladığından ve XML metni `<`, `>` ve `"` ile
doluysa, literal XML kaynağını bir raw String olarak yazın:

```ahd
document: XMLDocument := XML.parse(r'<a id="1"><b>text</b></a>')
```

Ayrıştırma sınırlandırılmıştır: 8&nbsp;MiB'den büyük girdi ve 256 seviyeden
derin element iç içeliği, tamamlanmadan önce `XMLError` fırlatır.

## XMLNode erişimcileri

```text
kind()      -> String
name()      -> String
namespace() -> String
text()      -> String

attribute(name: String) -> String?
attributes()             -> Pair<String, String>

children() -> List<XMLNode>
elements() -> List<XMLNode>

XMLDocument.root() -> XMLNode
```

`kind()`, tam olarak `"Element"` veya `"Text"` döndürür ve asla fırlatmaz.

`name()`, `namespace()`, `attribute()`, `attributes()`, `children()` ve
`elements()` yalnızca `Element` içindir: bunlardan herhangi birini bir
`Text` node'unda çağırmak `XMLError` fırlatır — JSON modülünün
`JSONValue`'sunun kullandığı aynı tek biçimli yanlış-tür kuralı.

`text()`, her iki türde de geçerli olan tek üyedir, türe bağlı anlamla: bir
`Text` node'u için kendi içeriğidir; bir `Element` için, o element'in
*doğrudan* `Text` çocuklarının belge sırasındaki birleşimidir — iç içe
geçmiş torun metni düzleştirilmez.

```ahd
p: XMLNode := XML.parse(r'<p>one<b>two</b>three</p>').root()
write(p.text())     -- "onethree" (yalnızca doğrudan Text çocukları)
```

`attribute(name)`, `String?` döndürür: `null`, özniteliğin yok olduğu
anlamına gelir. `attributes()`, her özniteliği ekleme sıralı bir
`Pair<String, String>` olarak döndürür.

`children()`, belge sırasındaki her doğrudan `Element`/`Text` çocuğunu
döndürür; `elements()`, yine sırayla yalnızca `Element` çocuklarını
döndürür.

`XMLDocument.root()`, belgenin kök `XMLNode`'unu döndürür — ayrıştırılmış/
okunmuş bir `XMLDocument`'tan node ağacına geri dönüş yoludur.

## Serileştirme

```text
XML.stringify(document: XMLDocument, pretty: Bool = false) -> String
XML.write(document: XMLDocument, path: String, pretty: Bool = false) -> Nothing
```

Her iki mod da metinde `&`, `<`, `>`'yi ve ayrıca öznitelik değerlerinde
`"`'yi ve üç boşluk kontrol karakterini kaçışlar; her ikisi de geçerli,
iyi biçimli XML üretir.

Kompakt çıktı (`pretty = false`, varsayılan) her zaman tam olarak round-trip
yapar: `XML.parse(XML.stringify(document))`, aynı ağacı tanımlar.

Pretty çıktı sabit iki boşluklu bir girinti kullanır, ancak bu girintiyi
yalnızca saf `Element` çocuklarından oluşan bir dizi arasına ekler. İçeriği
yalnızca metin veya karışık text/element olan bir element, her iki modda da
her zaman çocukları satır içinde işlenmiş olarak render edilir; çünkü bir
`Text` çocuğunun yanına boşluk eklemek, orijinal ağaçta olmayan içerik
eklemek anlamına gelir. Bu, her XML pretty-printer'ının yaptığı iyi bilinen
aynı ödünleşimdir — pretty çıktı, insan tarafından okunabilirlik için bir
kolaylıktır ve kayıpsız olması garanti edilen kompakt çıktıdır.

`XML.write`, çıktısını Word ve JSON'un kullandığı aynı
temp-dosya-sonra-yeniden-adlandır kuralıyla aşamalı olarak hazırlar ve
atomik olarak yayımlar: başarısız bir yazma, hedefte zaten bulunan bir
dosyayı asla bozmaz.

## Öznitelikler

v0.1.17 yüzeyi için, sıradan niteliksiz öznitelikler bir `Pair<String,
String>`dir, ekleme sırası korunur. Bir `Pair` kendi başına yinelenen bir
anahtar taşıyamaz, bu yüzden `XML.element`'in `attributes` argümanı asla
yinelenme üretemez — ancak ham ayrıştırılmış XML üretebilir ve kaynak
metindeki yinelenen bir öznitelik `XMLError` fırlatır. Sayısal veya diğer
String-olmayan öznitelik değerleri asla otomatik olarak zorlanmaz; önce
açıkça `str(...)` ile dönüştürün.

## Namespace'ler

Bu sürümdeki XML namespace desteği bilinçli olarak sınırlandırılmıştır:

- **Ayrıştırma namespace-farkındadır.** `XML.parse`/`XML.read`, her
  element'in namespace önekini (Go'nun `encoding/xml` decoder'ı aracılığıyla)
  tam URI'sine çözer ve bunu `namespace()` ile açığa çıkarır; namespace'siz
  bir element `""` bildirir.
- **Orijinal önek yazımı tam olarak korunmaz.** Yalnızca çözülmüş URI
  tutulur — bu sürüm byte-byte önek round-trip'i vaat etmez.
- **Oluşturma yalnızca niteliksizdir.** `XML.element`'in namespace parametresi
  yoktur; bu sürümde AhdCode kaynağından namespace-nitelikli bir element
  *oluşturmanın* bir yolu yoktur, yalnızca mevcut birini ayrıştırmanın.
- **Öznitelikler niteliksiz kalır.** Küçük `Pair<String, String>` öznitelik
  yüzeyi, öznitelik başına namespace niteliğini taşımaz.

Bu bilinçli bir sınırdır, bir gözden kaçırma değil: tam bir namespace-yazma
API'si (QName türleri, önek bağlamaları, öznitelik başına namespace'ler)
yapılandırılmış veri desteği kapsamındaki bir sürüm için önemli bir yüzey
ekler.

## Güvenlik

`XML.parse`/`XML.read` asla ağa erişmez ve hiçbir şey çalıştırmaz. Go'nun
`encoding/xml` decoder'ı harici bir DTD alt kümesini getirmez veya işlemez
ve varsayılan olarak özel genel varlıkları (entity) genişletmez — tanımsız
bir varlık referansı bir ikame değil, bir ayrıştırma hatasıdır; bu yüzden
klasik XXE ve billion-laughs saldırıları, bu modülün eklemediği ek kod
olmadan erişilebilir değildir. Comment'ler, processing instruction'lar ve
DOCTYPE bildirimleri bayt olarak tanınır ama asla bir çalıştırma mekanizması
olarak yorumlanmaz.

## Hatalar

`XMLError` doğrudan `Error`'dan türer ve her XML'e özgü hatayı kapsar:
bozuk girdi, kök element yok, birden fazla kök element, geçersiz (`Text`)
bir `XMLDocument` kökü, yinelenen bir öznitelik, yanlış türde bir node
yöntemi, derinlik/boyut sınırları ve eksik/okunamayan/yazılamayan bir
dosya.

```ahd
attempt {
    XML.parse("<a><b></a></b>")
} except XMLError as error {
    write(error.message)
}
```

## Kapsam dışı

XML, bir XML standartları paketi değil, yapılandırılmış veri desteğidir. Bu
sürümde XPath, XSLT, XML Schema/XSD doğrulama, DTD işleme, harici varlık
çözümlemesi, tam bir DOM mutasyon API'si, özel bir Comment/
ProcessingInstruction API'si, bir namespace-önek yönetim çerçevesi, kanonik
XML (C14N), dijital imzalar, bir SOAP çerçevesi veya HTML ayrıştırma yoktur.
