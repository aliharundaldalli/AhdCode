# Türler, çıkarım ve null güvenliği

[English](TYPES_AND_NULL.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Koleksiyonlar](COLLECTIONS_TR.md)

## Statik türler ve çıkarım (inference)

| Tür | Anlamı |
|---|---|
| `Int` | denetimli (checked) işaretli 64-bit tam sayı |
| `Real` | sonlu 64-bit kayan noktalı sayı |
| `String` | değiştirilemez (immutable) Unicode metin |
| `Bool` | `true` veya `false` |
| `Complex` | karmaşık sayı |
| `List<T>` | sıralı, homojen, değiştirilebilir koleksiyon |
| `Pair<K, V>` | ekleme sırasını (insertion-order) koruyan homojen anahtar/değer koleksiyonu |
| `Class` | sınıf bildirimi ve örnek kimliği (instance identity) |
| `Function` | statik olarak çözülmüş, isimlendirilmiş çağrılabilir değer |
| `Nothing` | çalışma zamanı değeri olmayan Function dönüş türü |

Başlangıç değerinin (initializer) belirsiz olmayan, tam bir statik türü
olduğunda bir tür belirtimi (annotation) isteğe bağlıdır:

```ahd
age := 25
name := "Ali"
```

Bunlar dinamik değişkenler değildir. `age = "Ali"` bir derleme zamanı
hatasıdır. Açık tür belirtimleri hâlâ faydalıdır ve bazen gereklidir:

```ahd
age: Int := 25
```

İç içe geçmiş (nested) bir çalıştırılabilir kapsam (scope) içinde, kapsam
niyeti (scope intent) hâlâ açıkça belirtilmelidir:

```ahd
name: Local := "Ali"       // Local String olarak çıkarılır
user: Local User? := null  // null'dan User çıkarılamadığı için açık tür gerekir
```

Yalın, iç içe geçmiş bir `name := "Ali"` geçersizdir. Modül deposuna
(module storage) erişen bir Function, mevcut açık `Global` kurallarına tabi
kalmaya devam eder. Function parametreleri ve dönüş değerleri açıkça
tiplenmiş kalır ve Class sözdizimi `Person: Class<> := { ... }` olarak
kalır.

## Null olabilen türler

Yalın `T` null olamaz. `T?`, bir değerin bulunmayabileceğini (absent)
belirtir:

```ahd
user: User? := null
```

Null olabilirlik (nullability), her yapısal seviyede ayrı ayrı birleşir:

```text
User           null olmayan User
User?          null olabilen User
List<User?>    null olmayan, elemanları null olabilen List
List<User>?    kendisi null olabilen, elemanları null olmayan List
List<User?>?   kendisi ve elemanları null olabilen List
```

Çıkarım (inference), bu tam türü korur. `fetchUser()`, `User?` döndürüyorsa,
`user := fetchUser()` `User?` türünü çıkarır. Yalın `value := null`
geçersizdir çünkü `null`, temel bir tür belirleyemez; `value: User? := null`
geçerlidir.

`T`, güvenle `T?`'ye sığar. Ters yön, kontrol akışı (control flow) ifadenin
null olmadığını kanıtlamadıkça reddedilir:

```ahd
if user != null {
    write(user.name)
}
```

Kısa devre (short-circuiting) sağ tarafı daraltır (refine eder):

```ahd
if user != null and user.age >= 18 {
    write(user.name)
}
```

Erken bir null `return` de sonraki kodu daraltır. Null olabilen bir
bağlamayı (binding) yeniden atamak, önceki kanıtı geçersiz kılar. Null
olabilen parametre/dönüş türleri ve aşırı yüklemeler (overloads) aynı
kuralları kullanır: `(Int)` ve `(Int?)` farklı imzalardır ve çözülmemiş bir
`Int?`, sessizce null olmayan aşırı yüklemeyi çağıramaz.

Null olabilen bir koleksiyon alıcısı (receiver), üye/indeks erişiminden önce
daraltılmalıdır. `List<T?>` için her eleman bağımsız olarak null olabilir ve
aynı şekilde kontrol edilmelidir. `Constant` derin dondurma davranışı, null
olabilirlikten etkilenmez. AhdCode'da v0.1'de truthiness, optional chaining,
null-coalescing veya force-unwrap sözdizimi yoktur.

## `Nothing` farklıdır

`Nothing`, bir eylemin dönüş türüdür, null olabilen bir değer değildir:

```ahd
report: Function := (
    message: String
) -> Nothing {
    write(message)
}
```

Saklanamaz, yazdırılamaz (printed), interpolasyona alınamaz veya bir değer
olarak döndürülemez.
