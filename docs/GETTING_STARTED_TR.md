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
