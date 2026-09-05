# Class Protocol Methods

[English](PROTOCOLS.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Sınıflar](CLASSES_TR.md) · [Temel İşlevler](FUNDAMENTALS_TR.md)

AhdCode, bir Class'ın küçük, kapalı bir kesin metot ismi kümesi aracılığıyla
operatör davranışı tanımlamasına izin verir: **Class Protocol Methods**. Bu,
kasıtlı olarak dar bir derleyici uzantı yüzeyidir (compiler extension
surface), genel bir reflection veya metaprogramlama mekanizması değildir ve
Python'un magic-method sistemi değildir. Çift alt çizgi (double-underscore)
adlandırma kuralı, `__eq__`/`__repr__`/`__radd__` tarzı bir aile yoktur ve
her dil mekanizmasına (oluşturma, yineleme, indeksleme, öznitelik erişimi,
çağırma) kendi protokol ismini verme girişimi yoktur.

Tam olarak on tane Class Protocol Method vardır:

```text
CEqual CCompare
CAdd CSubtract CMultiply CDivide CRemainder CPower
CNegate CStr
```

**Yalnızca bu on kesin isim özeldir, ve yalnızca bir Class metot konumunda
olduklarında.** `C` harfinin kendisi ayrılmış (reserved) değildir.
`Calculate`, `Create`, `CWhatever` ve `CustomMethod`, Class içinde bile,
özel bir anlamı olmayan sıradan isimlerdir. Modül kapsamında, yukarıdaki
kesin isimler bile (örneğin modül-kökü bir `CAdd: Function := ...`) sıradan
Function olarak kalır -- protokol anlamı yalnızca on isimden biri bir Class
metot konumunu işgal ettiğinde var olur.

İsim orada ayrılmış olduğundan, bunlardan birini yeniden kullanan bir
Function olmayan üye, sessizce sıradan bir alana (field) dönüşmek yerine
derleme zamanında reddedilir:

```ahd
Bad: Class<> := {
    structure: Attributes := (x: Int)
    CAdd: Int := 5   // hata: CAdd ayrılmış bir Class Protocol Method konumudur
}
```

## Bildirim sözdizimi değişmez

Bir Class Protocol Method, diğer herhangi bir metotla aynı Function/metot
sözdizimiyle yazılır. Yeni bir bildirim biçimi yoktur:

```ahd
Vector2: Class<> := {
    structure: Attributes := (
        x: Real
        y: Real
    )

    CEqual: Function := (
        other: Vector2
    ) -> Bool {
        return attribute.x == other.x and attribute.y == other.y
    }

    CAdd: Function := (
        other: Vector2
    ) -> Vector2 {
        return Vector2(x: attribute.x + other.x, y: attribute.y + other.y)
    }

    CNegate: Function := (
    ) -> Vector2 {
        return Vector2(x: -attribute.x, y: -attribute.y)
    }

    CStr: Function := (
    ) -> String {
        return "Vector2({attribute.x}, {attribute.y})"
    }
}
```

Class sözdizimi (`Person: Class<> := { ... }`) ve Function bildirim
sözdizimi (`name: Function := (...) -> T { ... }`) değişmez.

## Gerekli imzalar

| Protokol | Açık parametreler | Dönüş türü |
|---|---|---|
| `CEqual` | tam olarak 1 | `Bool` |
| `CCompare` | tam olarak 1 | `Int` |
| `CAdd`, `CSubtract`, `CMultiply`, `CDivide`, `CRemainder`, `CPower` | tam olarak 1 | operatörün sonuç türü (Class'ın kendisi olması gerekmez) |
| `CNegate` | 0 | operatörün sonuç türü |
| `CStr` | 0 | `String` |

Hatalı biçimlendirilmiş (malformed) bir protokol bildirimi, asla bir çalışma
zamanı panic'i değil, normal bir semantik tanılamadır (diagnostic): yanlış
sayıda argüman (arity), yanlış dönüş türü veya ayrılmış bir isimle Function
olmayan bir üye, hepsi derleme sırasında yakalanır.

Aritmetik protokoller ve `CNegate`, kapsayan Class'ı döndürmek zorunda
değildir. Gelecekteki bir `Matrix * Vector -> Vector` meşrudur; operatör
ifadesinin statik türü basitçe seçilen metodun bildirilen dönüş türüdür.

## Operatör eşlemesi

| Operatör | Protokol |
|---|---|
| `==` | `CEqual` |
| `!=` | aynı `CEqual` çağrısının mantıksal olumsuzlaması -- `CNotEqual` yoktur |
| `<`, `<=`, `>`, `>=` | dördü de tek bir `CCompare` çağrısından türer |
| `+` | `CAdd` |
| `-` (ikili) | `CSubtract` |
| `*` | `CMultiply` |
| `/` | `CDivide` |
| `%` | `CRemainder` |
| `^` | `CPower` |
| `-` (tekli) | `CNegate` |

### Eşitlik: `==` ve `!=`

`a`'nın Class'ı sağladığında `a == b`, `a.CEqual(b)`'yi çağırır. `a != b`
her zaman `!(a.CEqual(b))`'dir -- aynı çağrının olumsuzu -- bu yüzden `==`
ve `!=`, iki bağımsız olarak uygulanmış metot aracılığıyla asla
çelişemez. `CEqual`, hiçbir zaman `CCompare`'den türetilmez ve `CCompare`
hiçbir zaman `CEqual`'den türetilmez: bir tür, doğal bir sıralaması olmadan
anlamlı bir şekilde eşitlik-karşılaştırılabilir olabilir, ve tam tersi de
geçerlidir.

Bir Class hiçbir `CEqual` sağlamıyorsa, `==`/`!=` v0.1.8-öncesi davranışını
(referans eşitliği, `same` ile aynı) değişmeden korur.

### Sıralama: `<`, `<=`, `>`, `>=`

`CLess`, `CGreater`, `CLessEqual` veya `CGreaterEqual` yoktur. Dört
karşılaştırma operatörünün tümü, ifade başına **tam olarak bir kez**
değerlendirilen tek bir `CCompare` çağrısı cinsinden tanımlanır:

```text
a <  b   =>  a.CCompare(b) <  0
a <= b   =>  a.CCompare(b) <= 0
a >  b   =>  a.CCompare(b) >  0
a >= b   =>  a.CCompare(b) >= 0
```

Herhangi bir negatif, sıfır veya pozitif `Int`, geçerli bir `CCompare`
sonucudur -- tam olarak `-1`, `0`, `1` ile sınırlı değildir:

```ahd
Score: Class<> := {
    structure: Attributes := (value: Int)
    CCompare: Function := (
        other: Score
    ) -> Int {
        return attribute.value - other.value   // herhangi bir işaret uygundur
    }
}
```

### Aritmetik ve tekli olumsuzlama

`+`, `-`, `*`, `/`, `%` ve `^`, `CAdd`, `CSubtract`, `CMultiply`,
`CDivide`, `CRemainder` ve `CPower`'a eşlenir. Tekli `-`, `CNegate`'e
eşlenir.

Dağıtım (dispatch) **sol işlenene dayalıdır**. Ters operatör protokolleri
(`CReverseAdd`/`CRAdd`/vb.) yoktur:

```ahd
value + 3   // value'nun Class'ı CAdd(Int) sahipse çalışır
3 + value   // value'nun CAdd'ini DENEMEZ -- bu sıradan bir tür hatasıdır
            // ilkel (primitive) kurallar altında bağımsız olarak geçerli olmadıkça
```

### Aşırı yükleme (Overloading)

Class Protocol Methods, sıradan metot aşırı yükleme mekanizmasını yeniden
kullanır -- ikinci bir aşırı yükleme sistemi yoktur. Bir Class, mevcut
aşırı yükleme kuralları izin verdiği sürece birden fazla `CAdd`
aşırı yüklemesi bildirebilir:

```ahd
CAdd: Function := (
    other: Vector2
) -> Vector2 { ... }

CAdd: Overload Function := (
    scalar: Real
) -> Vector2 { ... }
```

Operatör çözümlemesi, sıradan bir çağrıyla aynı statik aşırı yükleme
çözümleme kurallarını kullanır. Belirsizlik (ambiguity), zaten aşırı
yüklenmiş metot çağrıları için olduğu gibi, bir derleme zamanı hatasıdır.

### Kalıtım ve override etme

Bir protokol metodu, tıpkı diğer herhangi bir metot gibi kalıtılır ve
override edilir. Kalıtsal bir protokol metodunu değiştirmek `Override`
gerektirir ve operatör dağıtımı, sıradan bir metot çağrısıyla aynı dinamik
dağıtımı kullanır, bu yüzden çalışma zamanı nesnesi bir alt sınıf olduğunda
daha türetilmiş bir override çalışır. Kavramsal olarak:

    Dog örneği tutan Animal referansı

Statik olarak geçerli protokol metodu Dog tarafından override edildiyse,
çalışma zamanı dağıtımı sıradan metot dağıtımıyla tutarlı davranmalıdır.
Operatörler için ikinci bir nesne dağıtım modeli icat edilmez.

### Bileşik atama (Compound assignment)

Bir Class hedefi üzerinde `+=`, `-=`, `*=`, `/=`, `%=` ve `^=`, eşleşen
aritmetik protokolü yeniden kullanır -- kavramsal olarak `a += b`, normal
atama-uyumluluğu kurallarına tabi olarak `a = a + b` gibi davranır. Alıcı
(receiver), bir üye veya indekslenmiş hedef dahil, tam olarak bir kez
değerlendirilir. Ayrı bir yerinde (in-place) protokol yoktur (`CIAdd` tarzı
bir metot yoktur). `++` ve `--` bununla ilgisizdir ve Class değerlerine
genişletilmez.

## Null olabilirlik

Protokol dağıtımı, null güvenliğini asla zayıflatmaz. Sol işlenen
null olabiliyorsa, bir protokol metodu onun üzerinde çağrılmadan önce
sıradan akış analizi (flow analysis) ile `NonNull`'a daraltılmalıdır --
diğer herhangi bir metot çağrısıyla aynı gereklilik:

```ahd
user: User?

user + other   // user, null olmayana daraltılmadıkça hâlâ geçersiz
```

Sağ taraftaki argüman, kendi bildirilen parametre türünü ve null
olabilirliğini normal şekilde kullanır; bir protokol açıkça null olabilen
bir argüman kabul edebilir (`CEqual(other: User?) -> Bool`), ancak bu kabul
hakkında hiçbir şey otomatik olarak çıkarılmaz.

## Bu ne değildir

- Python magic method'ları değil: `__eq__`, `__lt__`, `__repr__`,
  `__radd__`/`__iadd__` yok, ve hiç çift alt çizgi kuralı yok.
- Genel bir reflection veya metaprogramlama sistemi değil: `CStructure`,
  `CConstructor`, `CGetAttribute`, `CSetAttribute`, `CIterator`, `CLength`,
  `CCall`, `CEnter` veya `CExit` yoktur. Yalnızca yukarıdaki on isim protokol
  anlamı taşır.
- Ters operatör yok, yerinde (in-place) protokol yok, `CRepr`/`CAbs` yok.
- `str(value)`, `CStr`'nin katıldığı tek dönüşümdür; ayrı bir
  geliştirici/hata ayıklama string protokolü yoktur.

`C` ile -- veya başka herhangi bir harfle -- başlayan her şey, tamamen
sıradan bir tanımlayıcı (identifier) olarak kalır.
