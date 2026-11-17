# Tanılamaları anlamak

[English](DIAGNOSTICS.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Başlangıç](GETTING_STARTED_TR.md) · [Hatalar](ERRORS_TR.md)

Bir tanılama kod, önem derecesi, kaynak aralığı, açıklama ve çoğunlukla
uygulanabilir bir ipucu bildirir. Kodlar kuralı tanımaya yardım eder, ancak
kalıcı harici ABI olarak vaat edilmez. Bozuk bir yapıda sonraki mesajlara
geçmeden ilk tanılamayı düzeltin; v0.1.11 yaygın parser zincirlerini bastırırken
dosyanın ilerisindeki bağımsız hataları bildirmeyi sürdürür.

## Yaygın düzeltmeler

1. Atanan ifade operatörün yanında başlamalıdır.

```ahd
# geçersiz: PAR010
value :=
    5

# düzeltilmiş
value := 5
```

2. Metot zinciri yeni satırda `.` ile başlayamaz.

```ahd
# geçersiz: PAR013
important := entries
    .filter(lambda (x: String) -> x != "")

# düzeltilmiş
filtered := entries.filter(lambda (x: String) -> x != "")
important := filtered
```

3. Çalıştırılabilir blok içindeki bildirim `Local` ister.

```ahd
# geçersiz
if ready { count: Int := 1 }

# düzeltilmiş
if ready { count: Local Int := 1 }
```

4. Null olabilen değeri null olmayan işlemden önce daraltın.

```ahd
# geçersiz: user null olabilir
write(user.name)

# düzeltilmiş
if user is not null { write(user.name) }
```

5. Lambda gövdesi blok değil, tek ifadedir.

```ahd
# geçersiz
double := lambda (x: Int) -> { return x * 2 }

# düzeltilmiş
double := lambda (x: Int) -> x * 2
```

6. Bir lambda, kendi parametrelerinin dışındaki bir bağlamayı yalnızca açık
bağımlılık listesiyle okur: çevreleyen bir `Local` veya Function parametresi
için `#name`/`Local name`, bir modül bağlaması için `@name`/`Global name`.
Yalın bir isim (`lambda [minimum] (...)`) reddedilir -- türünü belirtin,
`lambda [#minimum] (...)` gibi. Bkz.
[Functions](FUNCTIONS_TR.md#açık-lambda-bağımlılıkları).

7. Function tanılamaları yanlış argüman sayısını, türünü veya adını belirtir.
Çağrıyı bildirilen parametre adları ve türleriyle eşleştirin; ilgisiz değerler
örtük dönüştürülmez.

8. Protokol tanılamaları sözleşmeyi adlandırır. Örneğin `CCompare` dönüşünü
`Int`, `CStr` dönüşünü `String` yapın; yeni protokol adı eklemeyin.

9. Geçersiz regex, çalışma zamanında yakalanabilir `RegexError` fırlatır:

```ahd
attempt { Regex.compile("(unfinished") }
except RegexError as error { write(error.message) }
```

10. Geçersiz tarihler, -840..840 dışı sabit ofsetler ve DateTime yıl 1..9999
dışına çıkan timestamp'ler `ValueError` fırlatır; Go panic/stack trace dil
davranışı değildir.

11. Bozuk tırnaklama, geçersiz ayraç, yinelenen/boş başlık ve kayıt şekli
uyuşmazlıkları `CSVError` fırlatır:

```ahd
attempt { CSV.parse("a,\"unfinished") }
except CSVError as error { write(error.message) }
```

Eksik initializer, atanan ifade, ikili sağ operand, index, çağrı, List, Pair ve
lambda gövdeleri; parser eksik parçayı bildiğinde yapıya özgü mesaj kullanır.
Kapanış tanılamaları beklenen `)`, `]` veya `}` işaretini adlandırır.

## Bir hata, bir tanılama

v2.8.0'dan itibaren derleyici bir hata için bir tanılama bildirir; ekrandaki
ilk mesaj, düzeltilecek olandır. Tanılama metinleri İngilizcedir.

- Bir satır yalnızca ilk sözdizimi hatasını bildirir.
- Kapatılmamış bir string gibi lexer'ın bildirdiği bir hatadan sonra, o hata
  düzeltilene kadar dosyanın devamından hiçbir şey bildirilmez.
- Tür denetimi yalnızca ayrıştırılabilen bir dosyada çalışır; bu yüzden bir
  sözdizimi hatasını, tanımsız kalan adlarla ilgili hatalar izlemez.
- Zaten hata bildirmiş bir deyim ayrıca satır sonunun eksikliğini, `{` işareti
  eksik olan bir blok da ayrıca `}` eksikliğini bildirmez.

12. Ayrılmış bir sözcük ad olamaz (`PAR015`). Ad olarak ilk kullanıldığı yerde,
bir öneriyle birlikte bir kez bildirilir:

```text
error [PAR015] main.ahd:1:1
'not' is a reserved keyword in AhdCode
  1 | not: Int := 70
    | ^^^
  hint: choose a different identifier, such as 'score'
```

13. Bir bildirim adını `:=` ile tanıtır. Tek başına `=` yalnızca zaten var olan
bir adı değiştirir.

```ahd
# geçersiz: PAR002
count: Int = 5

# düzeltilmiş
count: Int := 5
```

14. Bir blok, onu başlatan satırda açılır.

```ahd
# geçersiz: PAR002
if ready
{
    write("go")
}

# düzeltilmiş
if ready {
    write("go")
}
```

Tek istisna Function'dır: `uses [...]` listesi ve `{` işareti yeni birer
satırda başlayabilir; `ahdcode format` bunları tek bir başlıkta birleştirir.

15. Başka bir dilin yazdığı gibi yazılmış bir deyim (`function`, `def`, `var`,
`let`, `print`, `elif` ve benzerleri), ipucunda AhdCode yazımıyla yanıtlanır.

16. Niteleyicisi olup türü olmayan bir bildirim, değerinin türünü alır:
`limit: Constant := 20.0` bir `Constant Real`'dir. v2.8.0'dan önce bu yazım
dahili bir `IR002` tanılamasıyla başarısız oluyordu.

`FMT001` bir programdaki hata değildir. `ahdcode format`, ürettiği düzenin
kaynaktaki belirteç ve yorumların tam olarak aynısını taşıdığını denetler;
taşımıyorsa `FMT001` bildirir ve bozuk bir dosya yazmak yerine dosyayı
değiştirmeden bırakır. Lütfen dosyayı bildirin.
