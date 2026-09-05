# Başlangıç

[English](GETTING_STARTED.md) · [Türkçe] · [Dil turu](LANGUAGE_TOUR_TR.md) · [CLI](CLI_TR.md)

## Derleyiciyi kurun

AhdCode şu anda Go 1.25 veya daha yeni bir sürümle derlenir.

```bash
cd AhdCode
go test ./...
go install ./cmd/ahdcode ./cmd/ahdnumeric ./cmd/ahdplot ./cmd/ahdsqlite
```

Eğer `Latex` modülünü veya `PDF` modülünün `.save()` metodunu kullanmayı
planlıyorsanız (ikisi de aynı çevrimdışı render motorunu paylaşır),
çevrimdışı (offline) Latex/Tectonic çalışma zamanını da hazırlamanız (stage)
gerekir. `Archive` böyle bir hazırlığa ihtiyaç duymaz. `SQLite`, yukarıda
kurulan paketli `ahdsqlite` yardımcısını kullanır; sistem `sqlite3` gerekmez.
Bu adım, sabitlenmiş
kaynakları indirmek için bir defaya mahsus ağ bağlantısı kullanır:

```bash
go run ./tooling/latex/cmd/package-latex --output "$(go env GOPATH)"
```

Hazırlık (staging) aşamasından sonra, AhdCode'un normal Latex işlemleri tamamen çevrimdışı çalışmaya devam eder.

Go'nun ikili dosya (binary) dizininin `PATH`'te olduğundan emin olun:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

Kurulumu doğrulayın:

```bash
ahdcode --version
```

## İlk programınız

`hello.ahd` dosyasını oluşturun:

```ahd
name := "AhdCode"
write("Hello {name}")
```

Derleyici `String` türünü çıkarır (infer eder); bağlama yine de statik olarak
tiplenmiştir. Niyetinizi açıkça ifade etmek istediğinizde veya çıkarım
(inference) yetersiz kaldığında açık bir tür belirtimi (`name: String := ...`)
yazın.

Kısa, yeniden kullanılabilir bir işlem için yalnızca ifade içeren lambda,
mevcut `Function` türünde bir değer oluşturur:

```ahd
square := lambda (value: Int) -> value^2
write(square(5))
```

Lambda parametreleri açık tür gerektirir; dönüş türü tek ifadeden çıkarılır.
Blok veya birden çok adım için normal bir Function kullanın.

Çalıştırın:

```bash
ahdcode run hello.ahd
```

Yerel (native) bir çalıştırılabilir dosya oluşturun:

```bash
ahdcode build hello.ahd -o hello
./hello
```

## Bir Web uygulaması başlatın

Bulunulan dizini kurar. Proje adı, paket yöneticisi veya ağ indirmesi yoktur:

```bash
mkdir MyPortal
cd MyPortal
ahdcode init web
ahdcode dev app.ahd
```

`app.ahd`, `.env`, `.env.example`, `.gitignore`, bir Config / Page / Layout /
Component, `public/app.css` ve boş başlangıç dosyaları `public/css/style.css`
ile `public/js/main.js` yazılır. `.env` yalnızca güvenli geliştirme
varsayılanlarını tutar (`APP_HOST=localhost`, loopback HTTP) ve gitignore’dadır.
`.env.example` güvenle commitlenir. Süreç ortamı hâlâ `.env`’den üstündür.
Var olan dosyaların üzerine yazılmaz; `--force` yoktur.

`http://127.0.0.1:8080` adresini açın. Başlangıç sayfası JavaScript veya CDN
yüklemez. Form, CSRF ve oturum burada yoktur; bkz.
[v0.16 form örneği](../examples/v0.16/forms_validation/README_TR.md).

## Girdi

`take` bir satır okur. Metin döndürür, bu yüzden sayısal girdi için açık bir
dönüşüm gerekir:

```ahd
name := take("Name: ")
age := int(take("Age: "))

write("{name} is {age}")
```

## Kaynak kodu biçimlendirmek

```bash
ahdcode format hello.ahd
ahdcode format --check hello.ahd
```

İlk komut dosyayı atomik olarak günceller. İkincisi ise dosyanın zaten
kanonik (standart) biçimde olup olmadığını yalnızca kontrol eder.

## Bir web uygulaması kurun

```ahd
bring Web

home: Function := (request: Request) -> Response {
    return Web.html(Web.UI.h1("Merhaba"))
}
```

`bring Web` birinci taraf web çatısıdır: yönlendirme, yanıtlar ve anlamsal bir
HTML bileşen katmanı tek bir içe aktarmada, paket yöneticisi olmadan
çevrimdışı çözülür. Bkz. [Web rehberi](WEB_TR.md) ve çalıştırılabilir
[Ahd Akademi örneği](../examples/v0.15/ahd_academi).

Sırada: [dil turunu](LANGUAGE_TOUR_TR.md) ve
[tanılama rehberini](DIAGNOSTICS_TR.md) okuyun, bir
[web uygulaması](WEB_TR.md) kurun veya UTC Time ve CSV dahil
[derlenmiş örnekleri](../examples/v0.1/README_TR.md),
[Data tablolarını](DATA_TR.md), [PDF](PDF_TR.md) üretimini ve
[Archive](ARCHIVE_TR.md) paketlemesini çalıştırın.

## v0.16: tam form iş akışı

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

`Page`, `Layout` ve `Component` bir uygulamayı düzenleme biçimleridir, dil
yapısı değil; işleyici adları da sıradan tanımlayıcılardır: örnek `Page` soneki
olmadan `register`, `registerSubmit` ve `profile` işleyicilerine yönlendirir,
`registerPage` kullanan uygulamalar ise değişmeden çalışmayı sürdürür. Bkz.
[10.1 Adlandırma](WEB_TR.md#101-adlandırma).
