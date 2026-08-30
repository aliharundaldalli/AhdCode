# Başlangıç

[English](GETTING_STARTED.md) · [Türkçe] · [Dil turu](LANGUAGE_TOUR_TR.md) · [CLI](CLI_TR.md)

## Derleyiciyi kurun

AhdCode şu anda Go 1.25 veya daha yeni bir sürümle derlenir.

```bash
cd AhdCode
go test ./...
go install ./cmd/ahdcode
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

Sırada: [dil turunu](LANGUAGE_TOUR_TR.md) okuyun veya
[derlenmiş örnekleri](../examples/v0.1/README_TR.md) çalıştırın.
