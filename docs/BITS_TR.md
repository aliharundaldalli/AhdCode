# Bits standart modülü

[English](BITS.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Modüller](MODULES_TR.md) · [Math](MATH_TR.md) · [Numeric](NUMERIC_TR.md) · [Security](SECURITY_TR.md)

`Bits`, AhdCode v1.1.0 ile gelen, açık ve derleyiciye kayıtlı `builtin:Bits`
modülüdür. Kardeş bir `Bits.ahd` onun yerini alamaz:

```ahd
bring Bits
from Bits bring BitsError
```

AhdCode dilbilgisinde bit operatörü yoktur — `and`, `or` ve `not` dilin
**mantıksal** operatörleridir, `^` ise üs almadır — bu yüzden bit işlemleri
bunun yerine adlandırılmış çağrılar olarak sunulur. Her işlem, dilin tek tam
sayı türü olan işaretli 64 bit `Int` üzerinde, onun ikiye tümleyen (two's
complement) bit deseni üzerinden tanımlıdır.

## Genel yüzey

```text
Bits.bitAnd(left: Int, right: Int)            -> Int
Bits.bitOr(left: Int, right: Int)             -> Int
Bits.bitXor(left: Int, right: Int)            -> Int
Bits.bitNot(value: Int)                       -> Int

Bits.shiftLeft(value: Int, distance: Int)          -> Int
Bits.shiftRight(value: Int, distance: Int)         -> Int
Bits.shiftRightUnsigned(value: Int, distance: Int) -> Int
Bits.rotateLeft(value: Int, distance: Int)         -> Int
Bits.rotateRight(value: Int, distance: Int)        -> Int

Bits.count(value: Int)                        -> Int
Bits.leadingZeros(value: Int)                 -> Int
Bits.trailingZeros(value: Int)                -> Int
```

Dört mantıksal işlem `and`, `or`, `xor` ve `not` yerine `bitAnd`, `bitOr`,
`bitXor` ve `bitNot` olarak yazılır; çünkü bu sözcüklerden üçü dilin kendi
ayrılmış (reserved) sözcükleridir.

## Kaydırmalar ve döndürmeler

`shiftRight` **aritmetiktir**: işaret biti çoğaltılır, bu yüzden `-8` bir bit
sağa kaydırıldığında `-4` olur. `shiftRightUnsigned` ise değeri işaretsiz
kabul eder ve boşalan bitleri sıfırla doldurur — SHA-256 ve benzeri
algoritmaların öngördüğü kaydırma budur. Döndürmeler bitleri 64 bitlik sözcük
(word) içinde hareket ettirir; hiçbir bit kaybolmaz.

```ahd
write(str(Bits.bitAnd(12, 10)))              // 8
write(str(Bits.shiftRight(-8, 1)))           // -4   (sign preserved)
write(str(Bits.shiftRightUnsigned(-1, 60)))  // 15   (zero filled)
write(str(Bits.rotateLeft(1, 63)))           // -9223372036854775808
```

`0..63` dışındaki bir kaydırma veya döndürme mesafesi, sessizce sıfır üretmek
yerine `BitsError` fırlatır; çünkü "64 bit kaydır" neredeyse her zaman sıfır
isteği değil, çağıran taraftaki bir aritmetik hatasıdır:

```ahd
attempt {
    write(str(Bits.shiftLeft(1, 64)))
} except BitsError as error {
    write(error.message)   // Bits shift distance must be between 0 and 63
}
```

## Bit sayma

`count`, popülasyon sayımıdır (population count): kaç bitin 1 olduğunu verir.
`leadingZeros` ve `trailingZeros`, `0` değeri için `64` döndürür; çünkü tüm
bitler sıfırdır.

```ahd
write(str(Bits.count(255)))          // 8
write(str(Bits.count(-1)))           // 64  (every bit set)
write(str(Bits.leadingZeros(1)))     // 63
write(str(Bits.trailingZeros(8)))    // 3
write(str(Bits.trailingZeros(0)))    // 64
```

### Hata türü

```ahd
from Bits bring BitsError
```

`BitsError`, `Error`'dan türer. Onu yalnızca kaydırma ve döndürme fonksiyonları
fırlatabilir; mantıksal işlemler ve sayma fonksiyonları her girdi için
tanımlıdır (total).
