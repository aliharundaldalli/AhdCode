# Env standart modülü

[English](ENV.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [File ve Path](FILESYSTEM_TR.md)

`Env`, derleyici tarafından kayıtlı `builtin:Env` modülüdür. Açıktır ve
kardeş bir `Env.ahd` dosyası onu gölgeleyemez:

```ahd
bring Env
from Env bring EnvError
```

Env bilinçli olarak küçük kalır: işlem ortam değişkenleri ve sınırlandırılmış
bir `.env` dosya biçimi, her ikisi de düz `String` olarak, hiçbir sayısal/
boolean çıkarım, shell interpolation veya komut çalıştırma olmadan.

## Yüzey

```text
Env.get(name: String)    -> String?
Env.getOr(name: String, fallback: String) -> String
Env.exists(name: String) -> Bool

Env.set(name: String, value: String) -> Nothing
Env.unset(name: String)  -> Nothing

Env.read(path: String)   -> Pair<String, String>
Env.load(path: String, override: Bool = false) -> Nothing
```

(`Env.has` değil `Env.exists`: `has` saklı bir anahtar kelimedir — `x has y`
protokol operatörü — ve `.` sonrası bir üye adı olarak görünemez; `exists`,
mevcut `File.exists` adlandırmasıyla eşleşir.)

## get, getOr, exists

`Env.get(name)`, `String?` döndürür: `null`, değişkenin yok olduğu anlamına
gelir. `Env.exists(name)`, yokluğu açıkça mevcut ama boş bir değerden ayırt
eder — ikisi de gerçek, farklı durumlardır:

```ahd
found: String? := Env.get("PORT")
if found != null {
    write(found)
}
write(Env.exists("PORT"))
```

`Env.getOr(name, fallback)`, yalnızca değişken yokken `fallback` döndürür;
açıkça mevcut boş bir `String`, fallback ile değiştirilmeden `""` olarak
döndürülür. Otomatik sayısal veya boolean dönüşüm yoktur — açıkça
dönüştürün:

```ahd
port: Int := int(Env.getOr("PORT", "8080"))
```

## set ve unset

`Env.set`/`Env.unset`, mevcut AhdCode işleminin kendi ortamını değiştirir;
sonraki `Env.get`/`Env.exists` çağrıları ve sonrasında başlatılan alt
süreçler, OS semantiğinin izin verdiği yerde değişikliği görür. Bir ad,
herhangi bir şey değiştirilmeden önce doğrulanır: boş olmamalı ve bir NUL
baytı veya `=` içermemelidir. Değerler hata mesajlarında asla loglanmaz.

## `.env` biçimi

```text
KEY=value
KEY="value"
KEY='value'

# tam satır comment

EMPTY=
```

- Bir anahtar `[A-Za-z_][A-Za-z0-9_]*` ile eşleşir.
- Bir satırın başındaki boşluk yok sayılır; ilk boşluk-olmayan karakteri `#`
  olan bir satır tam satır comment'tir; boş bir satır yok sayılır.
- Tırnaksız bir değer, `=`'den satır sonuna kadar olan her şeydir, baştaki ve
  sondaki boşluklar kırpılır.
- Çift tırnaklı bir değer tam olarak `\\`, `\"`, `\n`, `\r` ve `\t`'yi
  destekler; başka herhangi bir kaçış reddedilir. Tek tırnaklı bir değer
  literaldir — açılış tırnağından sonra eşleşen kapanış tırnağına kadar
  hiçbir şey özel olarak işlenmez.
- Tırnaklı bir değerin kapanış tırnağından sonra, sondaki boşluk dışında
  hiçbir şey gelemez.

Bilinçli olarak `$(...)`, `` `...` ``, `${...}`, `$NAME` veya başka herhangi
bir shell-tarzı genişletme yoktur: bir `.env` değeri literal metin olarak
okunur, asla değerlendirilmez ve asla bir süreç başlatmaz.

```ahd
Env.load(".env")
databasePath: String := Env.getOr("DATABASE_PATH", "app.db")
```

## read ve load

`Env.read(path)`, bir dosyayı ayrıştırır ve onu ekleme sıralı bir
`Pair<String, String>` olarak döndürür, işlem ortamına hiç dokunmadan.

`Env.load(path, override = false)`, dosyanın tamamını ayrıştırır, tamamen
doğrular ve yalnızca ondan sonra uygular — bozuk bir sonraki satır, aynı
çağrıdan işlemi asla yarı güncellenmiş bırakamaz. `override = false` ile
(varsayılan), zaten mevcut bir değişken — `exists()`'in kullandığı aynı
yokluk-karşı-boş-farkında şekilde kontrol edilir — dokunulmadan bırakılır:
mevcut işlem/OS ortamı dosyaya karşı kazanır. `override = true` ile, `.env`
değeri orada ne olursa olsun her zaman onun yerini alır.

```ahd
Env.load(".env")            -- mevcut ortam kazanır
Env.load(".env", true)      -- .env her zaman kazanır
```

Bir `.env` dosyası içindeki yinelenen anahtarlar, sonuncunun sessizce
kazanmasına izin vermek yerine `EnvError` ile reddedilir.

## Hatalar

`EnvError` doğrudan `Error`'dan türer ve şunları kapsar: eksik/okunamayan
bir `.env` dosyası, bozuk bir atama, geçersiz bir anahtar, sonlandırılmamış
tırnaklı bir değer, yinelenen bir anahtar, geçersiz bir kaçış dizisi ve bir
OS düzeyinde `set`/`unset` hatası. Hata mesajları asla değişkenin değerini
içermez.

```ahd
attempt {
    Env.load("missing.env")
} except EnvError as error {
    write(error.message)
}
```
