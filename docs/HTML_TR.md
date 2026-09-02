# HTML standart modülü

[English](HTML.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [HTTP](HTTP_TR.md) · [Öğrenci Rehberi](STUDENT_GUIDE_TR.md#36-küçük-bir-web-sayfası)

`HTML`, AhdCode v0.4.0 ile gelen, derleyici tarafından kayıtlı
`builtin:HTML` modülüdür. Açıkça getirilir ve yanındaki bir `HTML.ahd`
onu gölgeleyemez:

```ahd
bring HTML
from HTML bring HTMLNode
from HTML bring HTMLError
```

`HTML` küçük, güvenli, yapılandırılmış bir HTML oluşturucudur. Şablon motoru,
DOM, ayrıştırıcı veya CSS motoru değildir. `HTML.raw` ve `HTML.div` kısayolu
yoktur. Dinamik metin ve öznitelik değerleri çizim anında Go
`html.EscapeString` ile kaçırılır. Düğümler orijinal kaçırılmamış String'i
saklar.

## Genel yüzey

```text
HTML.text(value: String) -> HTMLNode
HTML.element(name: String, attributes: Pair<String, String>, children: List<HTMLNode>) -> HTMLNode
HTML.render(node: HTMLNode) -> String
HTML.document(title: String, body: List<HTMLNode>) -> String

HTMLError  (Error'dan türer)
```

`HTMLNode` opak bir yerleşik Sınıftır: `HTMLNode()` ile oluşturulamaz, örnek
üyesi yoktur ve yalnızca `text` ile `element`'ten elde edilir. Tüm argümanlar
konumsaldır. Boş öznitelikler `{}`'dır. Boş çocuklar `[]`'dır. Öznitelik ve
çocuk sırası korunur.

## Kaçırma

```ahd
write(HTML.render(HTML.text("<script>alert(1)</script>")))
```

`&lt;script&gt;alert(1)&lt;/script&gt;` yazdırır. Metin ve öznitelik
değerlerindeki `&`, `<`, `>` ve tırnaklar kaçırılır. Türkçe karakterler ve
emoji olduğu gibi geçer. Kaynak String/List/Pair değiştirilmez.

Kullanıcıdan veya veritabanından gelen her metin için `HTML.text` (veya
`HTML.element` öznitelik değerleri) kullanın. Bu değerleri ham bir HTML
String'ine birleştirmeyin.

## Öğeler

Etiket adları bir ASCII harfle başlamalı, sonra harf, rakam veya tire
gelebilir. Öznitelik adları ayrıca `_` kabul eder. Boş adlar, boşluklar,
tırnaklar, `<`, `>`, `=` ve `/` `HTMLError` ile reddedilir. Yinelenen
öznitelik adları `HTMLError` fırlatır.

Boş öğelerin (void) çocuğu olamaz: `area`, `base`, `br`, `col`, `embed`,
`hr`, `img`, `input`, `link`, `meta`, `param`, `source`, `track`, `wbr`.
Boş öğeye çocuk vermek `HTMLError` fırlatır. Kapanış etiketi olmadan
çizilirler.

`HTML.document(title, body)` tam bir sayfa String'i döndürür; başlık
kaçırılır. Bu String'i `HTTP.html(...)` ile gönderin ki yanıt
`text/html; charset=utf-8` olsun.

## Güvenilir statik HTML ve oluşturucu

`HTTP.html(r"""...""")` kaynakta yazdığınız bir String gönderir. **Temizlemez**.
Sabit bir merhaba sayfası için uygundur.

`HTML.text(kullanıcıDeğeri)` dinamik içerik içindir. Web Not Defteri başlık
ve gövdeleri SQLite String olarak saklar, sonra `HTML.text` ile çizer; böylece
`<script>` veritabanında veri kalır ve sayfada kaçırılmış görünür.

## Bu modülün yapmadıkları

Şablon dili, `HTML.raw`, HTML ayrıştırıcı, CSS, JavaScript modülü, DOM ve
bileşen çerçevesi yoktur. Oluşturucu bilinçli olarak küçüktür.
