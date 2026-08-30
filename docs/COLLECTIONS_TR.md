# Koleksiyonlar

[English](COLLECTIONS.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [List API](LIST_API_TR.md)

## List'ler

```ahd
numbers: List<Int> := [1, 2, 3]
write(numbers[0])
write(numbers[-1])
write(numbers[1:])
```

List'ler homojen, sıralı referans nesneleridir. Negatif indeksler
desteklenir; geçersiz indeksler `IndexError` fırlatır. Yalın (bare) boş bir
List, açık bir eleman türü gerektirir.

Eleman ve alıcı (receiver) null olabilirliği (nullability) birbirinden
ayrıdır: `List<User?>` null içerebilir, `List<User>?` kendisi null olabilir
ve `List<User?>?` ikisini de birleştirir.

Elemanlar arasındaki virgüller, aralarında zaten bir satır sonu (newline)
varsa isteğe bağlıdır ve sondaki virgüle her zaman izin verilir:

```ahd
numbers: List<Int> := [
    1
    2
    3
]
```

bu, `[1, 2, 3]` ile aynı List'tir. `ahdcode format` ([Formatter](FORMATTER_TR.md)'a
bakın) her iki yazımı da aynı şekilde işler: sığıyorsa tek satırda, aksi
halde sondaki virgül olmadan her eleman kendi satırında.

```ahd
alias: List<Int> := numbers
alias[0] = 9
write(numbers[0]) // 9
```

`==` koleksiyon içeriklerini derinlemesine (deeply) karşılaştırır. `same`
nesne kimliğini (object identity) karşılaştırır.

## Pair'ler

```ahd
scores: Pair<String, Int> := {
    "Ali": 85
    "Ayşe": 92
}

scores["Ali"] = 90
write(scores["Ali"])
```

Pair'ler ekleme sırasını (insertion order) korur. Bir anahtarı güncellemek
konumunu korur; silip yeniden eklemek onu sona ekler. Eksik anahtarlar
`KeyError` fırlatır. Geçerli anahtar türleri `String`, `Int` ve `Bool`'dur;
Real, Class ve null anahtarlar desteklenmez. Pair değerleri, açıkça
yazıldığında null olabilir, örneğin `Pair<String, User?>`.

Pair girdileri, List'lerle aynı esnek virgül kuralını izler: `{"Ali": 85,
"Ayşe": 92}` ile yukarıdaki çok satırlı biçim aynı değerdir.

`value in scores` Pair anahtarlarını kontrol eder. `has` yalnızca Class üye
varlığı (member existence) içindir.

## Constant ve derin dondurma (deep freeze)

```ahd
values: Constant List<Int> := [1, 2, 3]
```

Nesne ve ulaşılabilir referans grafiği dondurulur. Herhangi bir takma ad
(alias) üzerinden değişiklik `ConstantError` fırlatır (ve doğrudan bilinen
bir değişiklik kontrol sırasında reddedilir). Bağımsız, değiştirilebilir bir
kopya gelecekteki bir kütüphane işlevselliğini gerektirir; v0.1 henüz `copy`
veya `deepCopy` yayınlamaz.

List elemanları ve Pair anahtarları üzerinde yineleme, sığ bir anlık görüntü
(shallow snapshot) kullanır. Yapısal değişiklik, aktif döngünün ziyaret
ettiği şeyi değiştirmez.
