# Latex standart modülü

[English](LATEX.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Time modülü](TIME_TR.md)

Latex, AhdCode String'lerini PDF belgelerine dönüştürür. Math ve Time gibi
açıktır (explicit) ve alias'lar dahil sıradan modül biçimleriyle çalışır:

```ahd
bring Latex
bring Latex as L
from Latex bring LatexError
```

Kanonik kimlik `builtin:Latex`'tir; kardeş bir `Latex.ahd` onun yerini
alamaz. Her argüman `NonNull` olmalıdır.

## Yüzey (Surface)

```text
pdf(source: String, output: String)      -> Nothing
pdfFile(input: String, output: String)   -> Nothing
escape(text: String)                     -> String
section(title: String)                   -> String
subsection(title: String)                -> String
equation(source: String)                 -> String
document(body: String, title: String = "", author: String = "") -> String
table(headers: List<String>, rows: List<List<String>>, mathColumns: List<Int> = []) -> String

LatexError
```

## Metin yardımcıları (Text helpers)

`escape`, **metin bağlamı (text-context)** kaçış (escaping) işlemidir.
TeX'e özel `\ { } $ & # % _ ^ ~` karakterlerini işler ve başka hiçbir şeyi
işlemez — ham matematiği (raw mathematics) sterilize ettiğini iddia etmez.

`section` ve `subsection`, başlıklarını (title) kaçış işlemine tabi tutar.
`equation` kasıtlı olarak kaçış işlemi **yapmaz**: amaç bu olduğu için ham
LaTeX matematik kaynağını olduğu gibi alır.

`document`, tam bir belge döndürür. Önsözü (preamble), paketlenmiş
(bundled) Latin Modern yazı tipi (font) dosyalarını açıkça adlandırır, bu
yüzden bir belge, hiç yazı tipi kurulu olmayan bir makinede bile aynı
şekilde işlenir.

`table`, deterministik `booktabs` kaynağı üretir ve her hücreye kaçış
işlemi uygular. Sütun sayısı başlıklardan farklı bir satır `ValueError`'dır.

`mathColumns`, belirli sütunları satır içi (inline) matematiğe dahil eder.
Listelenmiş bir sütundaki bir hücreye ham LaTeX olarak güvenilir ve kaçış
işlemi yerine `\( ... \)` içine sarılır, bu yüzden `^`, `_`, süslü
parantezler ve `\ln` gibi komutlar hayatta kalır. Başlıklar her zaman
sıradan kaçışlı metindir ve listelenmeyen her sütun mevcut kaçış işlemini
korur — genel bir ham-LaTeX baypası (bypass) yoktur.

```ahd
body += L.table(
    ["Fonksiyon", "Bigeometrik türev", "Yorum"],
    [
        ["g(x)=x^a", "e^a", "İlk türev sabittir"],
        ["g(x)=e^\{a(\\ln x)^m\}", "e^\{am(\\ln x)^\{m-1\}\}", "Logaritmik aile"]
    ],
    [0, 1]
)
```

AhdCode, tek bir çağrıda sıralı (positional) ve isimlendirilmiş (named)
argümanlara izin vermez, bu yüzden listeyi yukarıdaki gibi sıralı geçirin
veya her argümanı isimlendirin:

```ahd
body += L.table(headers: titles, rows: values, mathColumns: [0, 1])
```

Sütun indeksleri sıfır tabanlıdır (zero-based). Negatif veya aralık dışı
bir indeks `ValueError`'dır ve tekrarlanan bir indeks, sınırlayıcıları
(delimiters) iç içe geçirmek yerine o sütunu bir kez seçer. `mathColumns`'u
atlamak, her hücreyi tam olarak öncekiyle aynı şekilde kaçışlı bırakır.

## Derleme

`pdf`, bir kaynak String'i derler; `pdfFile`, var olan bir `.tex` dosyasını
derler ve `\includegraphics` gibi belgeye göreli varlıkları (assets) girdi
dosyasının dizinine göre çözer.

Derleme, bir AhdCode kurulumuyla birlikte gelen **paketlenmiş** bir
Tectonic motoru ve **paketlenmiş** bir yerel kaynak paketi tarafından
yapılır:

```text
libexec/ahdcode/latex/tectonic
libexec/ahdcode/latex/ahdcode-latex.ttb
libexec/ahdcode/latex/THIRD_PARTY_NOTICES.txt
```

AhdCode, `PATH`'te bulunan bir `tectonic`'i asla çalıştırmaz, hiçbir zaman
bir sistem TeX kurulumuna geri dönmez (fall back) ve çalışma zamanında
hiçbir şey indirmez. Paketlenmiş motor veya paket eksikse, bu bir
`LatexError`'dır.

## Yapısı gereği çevrimdışı

Motor, izole bir çağrı-başına (per-invocation) önbellek (cache) ve yalnızca
yerel-paket (local-bundle-only) politikasıyla çağrılır, bu yüzden
desteklenen bir belge, boş bir önbelleğe ve ağa sahip olmayan taze bir
makinede derlenir. Ayrıca kurulu bir TeX dağıtımı ve çalışma zamanı kaynak
indirmesi yoktur.

## Güvenlik

Motor güvenilmeyen (untrusted) modda çalışır, bu yüzden `\write18` kabuk
kaçışı (shell escape) kullanılamaz ve hiçbir AhdCode kaynak yapısı bunu
etkinleştiremez. Motor, bir kabuk komut dizesi değil bir argüman vektörüyle
başlatılır — bu yüzden boşluk, Unicode, tırnak, `$`, `;`, `&` veya parantez
içeren yollar güvende kalır.

Derleme, 30 saniyelik bir zaman aşımıyla (timeout) sınırlıdır. Zaman
aşımında motor süreci sonlandırılır, geçici dosyalar kaldırılır ve bir
`LatexError` fırlatılır.

## Çıktı güvenliği

Kaynak, başarı ve başarısızlıkta kaldırılan benzersiz, güvenli bir geçici
dizinde derlenir. PDF, geçici bir konuma üretilir, varlığı, sıradan-dosya
(regular-file) durumu, sıfır olmayan boyutu ve `%PDF-` imzası kontrol edilir
ve ancak o zaman istenen hedefe taşınır. Başarısız bir derleme bu yüzden
hiçbir zaman zaten geçerli bir hedef PDF'i yok etmez.

## LatexError

Tek bir hata, Latex'e özgü başarısızlıkları kapsar: derleme başarısızlığı,
eksik bir paketlenmiş motor veya paket, zaman aşımı, motor süreç
başarısızlığı ve üretilmemiş bir PDF. Motor tanılamaları, hatalı
biçimlendirilmiş bir belgenin terminali doldurmasını önlemek için
sınırlıdır; bu sırada ilk faydalı TeX hatası korunur.

```ahd
bring Latex as L
from Latex bring LatexError

attempt {
    L.pdf(source: source, output: "report.pdf")
} except LatexError as error {
    write(error.message)
}
```

## Desteklenen temel çizgi (baseline)

`article`, `amsmath`/`amssymb`/`mathtools`, `graphicx`, `booktabs`,
`array`, `geometry`, `xcolor`, `hyperref`, `fontspec`, Latin Modern yazı
tipleri, Computer Modern matematik ve heceleme (hyphenation) verisi.
Türkçe dahil Unicode metin, kutudan çıktığı gibi çalışır.

Bu sürümde olmayanlar: BibTeX, bir paket yöneticisi, TikZ veya Beamer
soyutlamaları, bir PDF düzenleyici veya ayrıştırıcı ve Markdown veya HTML
dönüşümü.
