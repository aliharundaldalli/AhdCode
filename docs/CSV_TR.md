# CSV standart modülü

[English](CSV.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [File ve Path](FILESYSTEM_TR.md)

İlk kez öğreniyorsanız önce kayıt/satır seçimi, tırnaklama, tür dönüşümü ve
hata yakalamayı birlikte ele alan [CSV atölyesini](PRACTICAL_MODULES_TR.md#1-csv-metin-tablosunu-güvenle-taşımak)
çalışın; bu sayfayı eksiksiz API referansı olarak kullanın.

`CSV`, derleyiciye kayıtlı `builtin:CSV` modülüdür. Açıkça içe aktarılır ve
kardeş bir `CSV.ahd` onun yerini alamaz:

```ahd
bring CSV
from CSV bring CSVError
```

CSV bir metin taşıma modülüdür. Her hücre `String` olarak kalır; sayı/tarih
çıkarımı yapmaz ve tablo ya da DataFrame soyutlaması sunmaz.

## Ham satırlar

```text
parse(text: String, delimiter: String = ",") -> List<List<String>>
stringify(rows: List<List<String>>, delimiter: String = ",") -> String
read(path: String, delimiter: String = ",") -> List<List<String>>
write(path: String, rows: List<List<String>>, delimiter: String = ",") -> Nothing
```

Ham ayrıştırma değişken genişlikli satırları kabul eder. Standart tırnaklama
desteklenir: ayraçlar ve yeni satırlar tırnaklı alanlarda bulunabilir; bir
tırnak `""` ile kaçırılır. LF/CRLF girdi, Unicode ve boş alanlar kabul edilir.
Go CSV yazıcısı deterministik çıktıyı tanımlar; `stringify([])` sonucu `""`dır.

```ahd
rows: List<List<String>> := CSV.parse("name,note\nAli,\"hello, world\"\n")
write(rows[1][1])
```

## Kayıtlar

```text
parseRecords(text, delimiter = ",") -> List<Pair<String, String>>
readRecords(path, delimiter = ",") -> List<Pair<String, String>>
stringifyRecords(records, delimiter = ",") -> String
writeRecords(path, records, delimiter = ",") -> Nothing
```

İlk satır başlıkları verir. Boş girdi ve yalnız başlık içeren belge boş List
üretir. Başlıklar boş olamaz, benzersiz olmalıdır ve her veri satırı tam başlık
genişliğinde olmalıdır.

Kayıt yazarken ilk Pair sütun sırasını belirler. Sonraki Pair'ler farklı ekleme
sırası kullanabilir, ancak tam olarak aynı anahtar kümesine sahip olmalıdır.
Eksik veya fazla anahtar `CSVError` fırlatır. Boş kayıt List'i `""`a çevrilir.

## Ayraçlar ve hatalar

Ayraç tam bir geçerli Unicode scalar içermeli; tırnak, CR veya LF olmamalıdır.
Boş, çok scalar içeren, geçersiz UTF-8 ya da desteklenmeyen ayraçlar
`CSVError` fırlatır. Bozuk tırnaklama ve kayıt/başlık şekli hataları da doğrudan
`Error`'dan türeyen yakalanabilir `CSVError` olur.

`read`/`write` dosya erişim hataları `FileError`/`IOError` anlamını korur.
Göreli yollar işlem çalışma dizisini; kalıcı REPL'de REPL'in başlatıldığı
dizini kullanır.

```ahd
attempt {
    CSV.parse("a,\"unfinished")
} except CSVError as error {
    write(error.message)
}
```

## CSV taşımadır, Data ise tablo katmanıdır

CSV kasıtlı olarak metin taşımada durur: String satırlarını ve başlık
anahtarlı String kayıtlarını ayrıştırır ve serileştirir, asla tür çıkarımı
yapmaz. Bu veriler üzerinde filtreleme, sıralama, gruplama veya sütun türetme
istediğinizde, onu [Data modülüne](DATA_TR.md) verin; `Table`'ı aynı String
hücreler üzerine kuruludur ve bu modülün okuyucusunu ve yazıcısını yeniden
kullanır.
