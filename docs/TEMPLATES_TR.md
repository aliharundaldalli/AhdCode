# Templates standart modülü

[English](TEMPLATES.md) · [Türkçe]

[README'ye dön](../README.md) · [Modüller](MODULES_TR.md) · [HTML](HTML_TR.md) · [HTTP](HTTP_TR.md) · [Web](WEB_TR.md)

`Templates`, derleyiciye kayıtlı `builtin:Templates` modülüdür. Açıkça
içeri alınır; yanındaki bir `Templates.ahd` onu gölgeleyemez:

```ahd
bring Templates
from Templates bring TemplateSet
from Templates bring TemplateData
from Templates bring TemplateError
```

`Templates`, HTML dosyalarını Go'nun `html/template` motoruyla işler. İkinci
bir şablon dili, tarayıcı veya [`HTML`](HTML_TR.md) oluşturucusunun yerine
geçen bir şey değildir. Ağacı AhdCode içinde kuruyorsanız `HTML` kullanın.
Sayfa programın yanındaki `.html` dosyalarındaysa `Templates` kullanın.

Sıradan değerler kaçışlanır. Ham işaretleme için tek yol
`Templates.trusted`'dır. Şablon eylemleri AhdCode fonksiyonu çağırmaz, dosya
açmaz ve ağa ulaşmaz.

## Genel yüzey

```text
Templates.nullValue()                         -> TemplateData
Templates.fromBool(value: Bool)               -> TemplateData
Templates.fromInt(value: Int)                 -> TemplateData
Templates.fromReal(value: Real)               -> TemplateData
Templates.fromString(value: String)           -> TemplateData
Templates.list(values: List<TemplateData>)    -> TemplateData
Templates.object(values: Pair<String, TemplateData>) -> TemplateData
Templates.trusted(markup: String)             -> TemplateData

Templates.parse(name: String, source: String) -> TemplateSet
Templates.sources(names: List<String>, bodies: List<String>) -> TemplateSet
Templates.files(root: String)                 -> TemplateSet

TemplateSet.render(name: String, data: TemplateData) -> String
TemplateSet.layout(layout: String, page: String, data: TemplateData) -> String

TemplateError  (Error'dan türer)
```

`TemplateData` ve `TemplateSet` opaktır. `TemplateData()` veya
`TemplateSet()` ile kurulamazlar. `TemplateData`'nın üyesi yoktur.

## Veri

Şablona ulaşan her değer açık bir kurucuyla oluşturulur. `Any` yoktur;
sıradan bir AhdCode değerinden otomatik dönüşüm yoktur:

```ahd
data: TemplateData := Templates.object({
    "title": Templates.fromString("Ana sayfa")
    "count": Templates.fromInt(3)
    "topics": Templates.list([
        Templates.fromString("Şablonlar")
        Templates.fromString("Yerleşim")
    ])
})
```

`Templates.object`, Pair argümanını tam olarak bir kez değerlendirir (yerel
ve değerlendirici). `fromReal` sonlu olmayan bir sayıyı reddeder. İç içe
listeler ve nesneler bir JSON ağacıdır: çocuk değerler doğrudan gömülür,
böylece kodlama boyutu veri ile büyür, üstel olarak değil. Liste ve nesne
kurucuları kökü derinlik 0 kabul eder ve derinliği 32 ile sınırlar. Yeni
kapsayıcı dahil en fazla 10000 düğüme izin verir. Yinelenen bir nesne
anahtarı `TemplateError` yükseltir.

## Kaçışlama ve güvenilir işaretleme

`fromString` sıradan bir dizgi saklar. Go'nun `html/template`'i, eylem HTML
bağlamına yazdığında bunu kaçışlar; böylece `<script>` görünür metin olur.
Kesme işareti `&#39;` olur. Güvenilmeyen içerik için varsayılan budur.

`Templates.trusted`, zaten güvenilen işaretlemeyi saklar ve o kaçışlamadan
geçmeden yazar. Yalnızca uygulamanın kendisinin ürettiği içerik için
kullanın. İstek gövdesi, sorgu değeri, çerez veya veritabanı alanını, ayrı ve
açık bir adımdan geçirmeden `trusted`'a vermeyin.

Başka bir ham ekleme yardımcısı yoktur.

## Paylaşılan şablonlar ve sayfa yalıtımı

`pages/` altındaki adlar **sayfaya özeldir**. Diğer her şey (`layouts/`,
`components/` ve diğer yollar) **paylaşılandır**.

Her işleme çağrısı yalnızca seçilen sayfanın tanımlarını paylaşılan prototipin
bir kopyasına ekler. Sayfalar `content` ve `sidebar` gibi paylaşılan yerleşim
bloklarına tanım sağlayabilir. Her dosya kendi şablon adına sahiptir.
`pages/` altındaki adları yalnızca sayfalar tanımlayabilir. Başka bir dosyanın
adıyla çakışan tanımlar ve birden çok paylaşılan dosyanın aynı adı tanımlaması
yükleme sırasında `TemplateError` yükseltir.

Eşzamanlı `render` ve `layout` çağrılarının her biri kendi kopyasını kullanır.
`layout(layout, page, data)` çağrısında `layout` yüklenmiş bir paylaşılan
şablonun adı, `page` ise `pages/` altında yüklenmiş bir şablonun adı olmalıdır.
Her iki şablon türü de doğrudan işlenebilir.

## Şablonları yükleme

Adlar eğik çizgiyle ayrılmış göreli yollardır. `/` ile başlamazlar; `\`, `:`,
NUL veya satır sonu içermezler; `C:/a.html` gibi bir Windows birim yolu
olamazlar; `.`, `..` veya nokta ile başlayan parça kullanamazlar. En fazla 16
yol parçası ve 128 karakter.

```ahd
pages: TemplateSet := Templates.files("templates")
```

`files(root)` o dizini yükleme anında bir kez gezer. Yalnızca düzenli `.html`
dosyaları yüklenir. Simgesel bağlantı veya birleşim (junction) kökü
reddedilir. Ağaçtaki herhangi bir reparse noktası reddedilir. Kökün ataları
simgesel bağlantı ve Windows birleşimleri için denetlenir. Yaprak okumaları
boyut sınırlıdır ve son bileşendeki reparse noktasını reddeder. Bu kontrolden
sonra bir üst dizinin bağlantı ile değiştirilmesi kalan yarıştır; her
platformda kapatılmaz.

`parse` ve `sources`, `files` ile aynı şablon boyutu (1&nbsp;MiB), UTF-8, ad
ve küme boyutu (256) sınırlarını uygular. Sıradan bir AhdCode String `{...}`
ifadesini interpolation saydığı için `{{...}}` içeren şablon kaynağını ham
String olarak yazın.

## Render, yerleşim ve çıktı sınırları

```ahd
home: Function := (request: Request) -> Response {
    pages: Global TemplateSet
    return HTTP.html(pages.layout(
        "layouts/site.html"
        "pages/home.html"
        data
    ))
}
```

Modül kökündeki `pages` bağını okuyan bir Function `pages: Global TemplateSet`
bildirmelidir.

Çalıştırma çıktısı 8&nbsp;MiB ile sınırlıdır. Aşım `TemplateError` yükseltir;
başarılı HTML sessizce kesilmez.

Sonucu [`HTTP.html`](HTTP_TR.md) veya [`Web.markup`](WEB_TR.md) ile sunun.
Bu fonksiyonlar `text/html; charset=utf-8` ayarlar ve **yeniden kaçışlamaz**.

## Hata sayfaları

[`server.notFound`](HTTP_TR.md) ve [`server.internalError`](HTTP_TR.md),
`Function(Request) -> Response` alır. Yazılan durum her zaman 404 veya
500'dür. İşleyicinin `Location` başlığı düşürülür; böylece zorlanmış bir hata
sayfası yönlendirme olamaz. İşleyici başarısız olursa istemci düz metin
`Not Found` veya `Internal Server Error` alır. Hata metni, yığın ve yollar
gönderilmez. Diğer hata sayfası çağrılmaz.

## Canlı yenileme

`ahdcode dev`, giriş dosyasının yanındaki `templates/` altında düzenli
`.html` dosyalarını izler. Oluşturma, değiştirme, silme, geri getirme ve
yeniden adlandırma fark edilir. Birini kaydetmek süreci yeniden derler ve
yeniden başlatır; debounce penceresi bir patlamayı tek yeniden derlemeye
toplar. İzleme simgesel bağlantıları izlemez. `ahdcode run` onları izlemez.
`ahdcode dev` olmadan başlatılan bir program yenileme betiği enjekte etmez.

Şablonlar `Templates.files` çalıştığında bir kez yüklenir. Diskteki bir
dosyayı düzenlemek, süreç yeniden başlatılana kadar çalışan süreci
değiştirmez.

## Sahiplik ve sınırlar

- Küme başına en fazla 256 şablon, şablon kaynağı başına 1&nbsp;MiB.
- Süreç başına en fazla 1024 yüklü `TemplateSet` tutamacı. Kümeler süreç ömrü
  boyunca tutulur; canlı bir kümenin sessizce atılması yoktur. Sınır aşımı
  `TemplateError` yükseltir.
- Veri iç içeliği en fazla 32; en fazla 10000 veri düğümü.
- Üretilen çıktı en fazla 8&nbsp;MiB.
- [`ahdcode package`](PACKAGING_TR.md) masaüstü paketleri üretir.
  `templates/` veya `public/` dizinini pakete kopyalamaz. Bu paketleme
  sınırlaması ertelenmiştir; şablon tabanlı bir web uygulamasının bu
  dizinlere çalışan sürecin çalışma dizininin yanında ihtiyacı vardır; ya da
  şablonları `parse` / `sources` ile yüklemelidir.

## Türkçe başlangıç

`ahdcode init web basic`, bu modülü kullanan küçük bir Türkçe site yazar:
`templates/layouts/site.html`, `templates/components/nav.html`, ana sayfa,
hakkında, 404 ve 500 sayfaları, `/`, `/hakkinda` ve `/ornek-hata` yolları
ve `Templates.object({...})` verisini `pages.layout`'a veren işleyiciler.
[Web başlangıçları](WEB_TR.md#16-v018-web-starters) bölümüne bakın.
