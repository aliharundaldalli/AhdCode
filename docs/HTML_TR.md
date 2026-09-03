# HTML standart modülü

[English](HTML.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [HTTP](HTTP_TR.md) · [Öğrenci Rehberi](STUDENT_GUIDE_TR.md#36-küçük-bir-web-sayfası)

`HTML`, derleyici tarafından kayıtlı `builtin:HTML` modülüdür. Oluşturucu
yarısı (`HTMLNode`) AhdCode v0.4.0 ile geldi; ayrıştırma yarısı
(`HTML.parse`, `HTMLDocument`, `HTMLElement`) v0.7.0'da eklendi. Açıkça
getirilir ve yanındaki bir `HTML.ahd` onu gölgeleyemez:

```ahd
bring HTML
from HTML bring HTMLNode
from HTML bring HTMLDocument
from HTML bring HTMLElement
from HTML bring HTMLError
```

`HTML`, tek bir modülü paylaşan iki bağımsız şeydir:

- güvenilir HTML metni üreten küçük, güvenli, yapılandırılmış bir
  **oluşturucu** (`HTML.text`, `HTML.element`, `HTML.render`,
  `HTML.document`), ve
- HTML metnini okuyup küçük bir CSS benzeri seçici diliyle içindeki öğeleri
  bulmanızı sağlayan bir **ayrıştırıcı ve sorgu yüzeyi** (`HTML.parse`,
  `HTMLDocument`, `HTMLElement`).

Şablon motoru, tarayıcı veya tam bir CSS motoru değildir. `HTML.raw` ve
`HTML.div` kısayolu yoktur. Oluşturucuya verilen dinamik metin ve öznitelik
değerleri çizim anında Go `html.EscapeString` ile kaçırılır.

## Genel yüzey

```text
// Oluşturucu (v0.4.0)
HTML.text(value: String) -> HTMLNode
HTML.element(name: String, attributes: Pair<String, String>, children: List<HTMLNode>) -> HTMLNode
HTML.render(node: HTMLNode) -> String
HTML.document(title: String, body: List<HTMLNode>) -> String

// Ayrıştırıcı (v0.7.0)
HTML.parse(source: String) -> HTMLDocument

HTMLDocument.select(selector: String) -> List<HTMLElement>
HTMLDocument.first(selector: String) -> HTMLElement?

HTMLElement.tag() -> String
HTMLElement.text() -> String
HTMLElement.attr(name: String) -> String?
HTMLElement.hasAttr(name: String) -> Bool
HTMLElement.select(selector: String) -> List<HTMLElement>
HTMLElement.first(selector: String) -> HTMLElement?

HTMLError  (Error'dan türer)
```

`HTMLNode`, `HTMLDocument` ve `HTMLElement` opak yerleşik Sınıflardır:
hiçbiri `HTMLNode()` / `HTMLDocument()` / `HTMLElement()` ile oluşturulamaz.
`HTMLNode` yalnızca `text`/`element`'ten gelir. `HTMLDocument` yalnızca
`HTML.parse`'tan gelir. `HTMLElement` yalnızca `HTMLDocument.select`,
`HTMLDocument.first`, `HTMLElement.select` veya `HTMLElement.first`'ten
gelir. `HTMLDocument` ve `HTMLElement` salt okunurdur: `setAttr`, `append`,
`remove` ya da başka hiçbir değiştirme işlemi yoktur.

## Oluşturucu ve ayrıştırıcı: iki ayrı kavram

Oluşturduğunuz bir `HTMLNode` (çizmek için `HTML.text`/`HTML.element` ile
kurduğunuz) ile ayrıştırılmış bir `HTMLElement` (`HTML.parse`'ın sizin için
bulduğu) birbiriyle ilişkisiz tiplerdir. Aralarında örtük ya da açık bir
dönüşüm yoktur: bir `HTMLNode`, bir `HTMLElement` beklenen yerde asla kabul
edilmez ve tersi de geçerlidir. Bir sayfayı ayrıştırıp öğelerinden birini
oluşturucuyla yeniden çizmek isterseniz, onu `HTML.element(...)` ile açıkça
yeniden kurun.

## Kaçırma (oluşturucu)

```ahd
write(HTML.render(HTML.text("<script>alert(1)</script>")))
```

`&lt;script&gt;alert(1)&lt;/script&gt;` yazdırır. Metin ve öznitelik
değerlerindeki `&`, `<`, `>` ve tırnaklar kaçırılır. Türkçe karakterler ve
emoji olduğu gibi geçer. Kaynak String/List/Pair değiştirilmez.

Kullanıcıdan veya veritabanından gelen her metin için `HTML.text` (veya
`HTML.element` öznitelik değerleri) kullanın. Bu değerleri ham bir HTML
String'ine birleştirmeyin.

## Öğeler (oluşturucu)

Etiket adları bir ASCII harfle başlamalı, sonra harf, rakam veya tire
gelebilir. Öznitelik adları ayrıca `_` kabul eder. Boş adlar, boşluklar,
tırnaklar, `<`, `>`, `=` ve `/` `HTMLError` ile reddedilir. Yinelenen
öznitelik adları `HTMLError` fırlatır.

Boş öğelerin (void) çocuğu olamaz: `area`, `base`, `br`, `col`, `embed`,
`hr`, `img`, `input`, `link`, `meta`, `param`, `source`, `track`, `wbr`.
Boş öğeye çocuk vermek `HTMLError` fırlatır. Kapanış etiketi olmadan
çizilirler.

```ahd
node: HTMLNode := HTML.element("h1", {}, [HTML.text("Hello")])
input: HTMLNode := HTML.element("input", {"name": "title", "value": userTitle}, [])
```

`HTML.document(title, body)` tam bir sayfa String'i döndürür:

```text
<!doctype html><html><head><meta charset="utf-8"><title>…</title></head><body>…</body></html>
```

Başlık kaçırılır. Bu String'i `HTTP.html(...)` ile gönderin ki yanıt
`text/html; charset=utf-8` olsun.

## Güvenilir statik HTML ve oluşturucu

`HTTP.html(r"""...""")` kaynakta yazdığınız bir String gönderir.
**Temizlemez**. Sabit bir merhaba sayfası için uygundur.

`HTML.text(kullanıcıDeğeri)` dinamik içerik içindir. Web Not Defteri başlık
ve gövdeleri SQLite String olarak saklar, sonra `HTML.text` ile çizer; böylece
`<script>` veritabanında veri kalır ve sayfada kaçırılmış görünür.

## Ayrıştırma: `HTML.parse` asla getirmez

```ahd
bring HTTP
from HTTP bring Client
from HTTP bring ClientResponse
bring HTML
from HTML bring HTMLDocument

client: Client := HTTP.client()
response: ClientResponse := client.get("https://example.com/notes")
document: HTMLDocument := HTML.parse(response.body())
```

`HTML.parse(source: String) -> HTMLDocument` tam olarak bir String alır ve
ondan kurulmuş ayrıştırılmış bir belge döndürür. URL argümanı yoktur.
Sayfayı almak ile onu ayrıştırmak iki bağımsız, açık adımdır --
`HTML.parse`'ın kendisi ağ erişimi bakımından saftır:

- asla bir HTTP isteği yapmaz,
- asla bir URL çözümlemez,
- işaretlemede adı geçen bir görseli, betiği, stil sayfasını veya iframe'i
  asla yüklemez, ve
- hiçbir şeyi asla çalıştırmaz.

`<img src="...">`, `<script src="...">`, `<link href="...">` veya
`<iframe src="...">` ayrıştırmak, bu URL'leri düz öznitelik metni olarak
adlandıran sıradan, erişilemez `HTMLElement` değerleri üretir -- hiçbir şey
asla çevrilmez (dial edilmez).

## JavaScript yok

```ahd
document: HTMLDocument := HTML.parse("<script>fetch('/x')</script>")
```

Bir `<script>` öğesi sıradan işaretleme olarak ayrıştırılır: içeriği, tıpkı
başka herhangi bir öğenin içeriği gibi, `.text()` ile istenirse erişilebilen
öğenin düz bir metin çocuğu olur. Asla kod olarak yorumlanmaz. JavaScript
motoru, DOM, olay döngüsü yoktur ve `onclick`/`onload`/`onerror`
öznitelikleri sıradan öznitelik metnidir -- `.attr("onclick")` çağırmak
kaynak dizesini döndürür ve hiçbir şey onu asla çalıştırmaz.

## Ayrıştırma modeli

`HTML.parse`, düzenli ifade değil, gerçek, elle yazılmış bir belirteçleyici
(tokenizer) ve ağaç kurucu kullanır. Sıradan bozuk HTML için tarayıcı benzeri
kurtarma sağlar -- kapanmamış etiketler, eksik `<html>`/`<head>`/`<body>`,
tırnaksız öznitelikler, karışık büyük/küçük harfli etiketler:

```ahd
document: HTMLDocument := HTML.parse("<div><p>Hello")
```

yine de kullanılabilir bir ağaç üretir (`Hello` metnini içeren bir `p`'yi
içeren bir `div`), hatasız. `HTML.parse` bir **doğrulayıcı değildir**:
sözdizimsel olarak taranabilir HTML asla reddedilmez ve eşleşmeyen veya eksik
bir kapanış etiketinden kurtarma, bir tanı değil, girdiden kurtarılabilecek
en makul ağacı üretir. Ayrıştırma yalnızca dahili bir boyut sınırını aşan
girdi, sıradan sayfaların çok ötesinde bir iç içe geçme derinliği, veya
kaynak geçerli UTF-8 olmadığında başarısız olur (`HTMLError`).

Yorumlar ve doctype tanınır ve atlanır; hiçbiri ayrıştırılmış ağacın parçası
olmaz, ve genel bir `Comment` ya da `Doctype` tipi yoktur. `<script>` ve
`<style>` içeriği, bir tarayıcının bu iki öğeye davranışıyla eşleşecek
şekilde, olduğu gibi (varlık kod çözümü yok, içinde etiket taraması yok)
yakalanır.

## Seçiciler: küçük, dondurulmuş bir alt küme

`select`/`first`, küçük, açık bir CSS benzeri seçici dilini kabul eder. Bu
listenin dışındaki her şey yaklaşık olarak eşleştirilmek yerine `HTMLError`
ile reddedilir -- kısmi eşleşme ve sessiz geri düşüş yoktur.

Desteklenenler:

| Söz dizimi | Anlamı | Örnek |
| --- | --- | --- |
| `*` | herhangi bir öğe | `*` |
| `tag` | etiket adı | `article`, `a`, `h2` |
| `#id` | id özniteliği (tam) | `#main` |
| `.class` | bir class token'ı (tam) | `.card` |
| `tag.class` vb. | bileşik (tümü eşleşmeli) | `article.card.featured` |
| `[attr]` | öznitelik mevcut | `[href]` |
| `[attr="değer"]` | öznitelik tam değeri (tırnaklı) | `[rel="next"]` |
| `A B` | B, A'nın soyundan gelir | `article a` |
| `A > B` | B, A'nın doğrudan çocuğudur | `article > h2` |
| `A, B` | seçici listesi (biri eşleşirse yeter) | `h1, h2` |

`>` ve `,` etrafındaki boşluklara izin verilir. Desteklen**meyen** ve
`HTMLError` ile reddedilenler: sözde sınıflar (pseudo-classes)
(`:first-child`, `:nth-child(...)`, `:not(...)`, ...), sözde öğeler
(pseudo-elements) (`::before`), kardeş birleştiriciler (`+`, `~`), diğer
öznitelik operatörleri (`^=`, `$=`, `*=`, `~=`, `|=`), CSS kaçış söz dizimi
ve XPath. Geçersiz bir seçici her zaman `HTMLError` fırlatır; asla sessizce
hiçbir şeyle eşleşmez ya da daha gevşek bir yoruma geri düşmez.

Eşleştirme kuralları:

- etiket adları ve öznitelik adları **büyük/küçük harfe duyarsız** eşleşir
  (HTML'in kendi kuralı -- kaynaktaki `DIV` ile bir seçicideki `div` eşleşir;
  `attr` ve `hasAttr` da öznitelik adlarını büyük/küçük harfe duyarsız
  arar);
- `id` değerleri, class token'ları ve `[attr="değer"]` değerleri **tam ve
  büyük/küçük harfe duyarlı** eşleşir;
- `.class`, `class` özniteliğinin boşlukla ayrılmış bir token'ıyla eşleşir;
- sonuçlar her zaman **belge sırasındadır**;
- bir seçici listesinin sonuçları **tekilleştirilir**: `"a, b"`'nin birden
  fazla dalıyla eşleşen bir öğe, ilk belge-sırası konumunda bir kez bildirilir.

## Öğe kapsamı

`HTMLDocument.select`/`.first` tüm belgeyi arar. `HTMLElement.select`/
`.first` yalnızca **o öğenin soyundan gelenleri** arar -- öğenin kendisi
asla dahil edilmez; bu, tanıdık `querySelectorAll` tarzı kapsamlamayla
eşleşir:

```ahd
articles: List<HTMLElement> := document.select("article.card")
firstArticle: HTMLElement := articles[0]
title: HTMLElement? := firstArticle.first("h2")
```

`title` yalnızca `firstArticle` içinde bulunur, belgede başka bir yerdeki
farklı bir makalede asla bulunmaz.

## Metin ve öznitelikler

`tag()`, **normalleştirilmiş (küçük harfli)** etiket adını döndürür.
Kaynaktaki `<DIV>` ile `<div>` ikisi de `tag() == "div"` bildirir; kaynağın
orijinal büyük/küçük harf kullanımını geri almanın bir yolu yoktur, çünkü
HTML'de bir anlam taşımaz.

`text()`, her soyundan gelen **metin düğümünün** içeriğini belge sırasında
birleştirir. CSS çizimini veya "görünür metni" simüle etmez: boşluk
sıkıştırılmaz ve öğeler arasına boşluk icat edilmez. HTML karakter
referansları (`&amp;`, `&#65;`, ...) ayrıştırma sırasında zaten çözülür.
Yorumlar asla metne katkıda bulunmaz. Baştaki/sondaki boşlukların
kaldırılmasını istiyorsanız sonuç üzerinde kendiniz `.trim()` çağırın.

`attr(name)`, ayrıştırılmış öznitelik değerini tam olarak yazıldığı gibi,
ya da öznitelik yoksa `null` döndürür. `hasAttr(name)`, değer boş olsa bile
var olmayı yok olmaktan ayırt eder (`<input disabled>`'ın `disabled`
değeri `""`'dir ve `hasAttr("disabled")` `true`'dur).

## Otomatik URL çözümlemesi yok

```ahd
link: HTMLElement? := article.first("a")
href: String? := link.attr("href")  // tam olarak "/notes/1", asla çözümlenmez
```

`HTML.parse` bir "sayfa URL'si" bilmez -- yalnızca kendisine verdiğiniz
String'i görür -- bu yüzden göreli bir `href`/`src` değeri tam olarak
ayrıştırıldığı gibi döndürülür, asla mutlak bir URL'ye dönüştürülmez.
v0.7.0'da `baseURL` ya da `resolveURL` yoktur. Mutlak bir URL gerektiğinde
sitenin bilinen tabanını döndürülen String ile kendiniz birleştirin.

## Bu modülün yapmadıkları

Şablon dili, `HTML.raw`, tarayıcı, headless çizim motoru, JavaScript
çalıştırma, CSS düzeni ya da hesaplanmış stiller, tam bir CSS seçici motoru
(yalnızca yukarıdaki dondurulmuş alt küme), otomatik kaynak getirme, URL
çözümlemesi ve DOM değiştirme API'si yoktur. Ayrıştırılmış
`HTMLDocument`/`HTMLElement` yüzeyi tasarım gereği salt okunurdur.
