# Sınıflar

[English](CLASSES.md) · [Türkçe]

[README'ye dön](../README_TR.md) · [Fonksiyonlar](FUNCTIONS_TR.md) · [Class Protocol Methods](PROTOCOLS_TR.md)

## Yapı (Structure) ve oluşturma

```ahd
Person: Class<> := {
    structure: Attributes := (
        name: String
        age: Int
    )

    describe: Function := (
    ) -> String {
        return "{attribute.name}, {attribute.age}"
    }
}

person: Person := Person(name: "Ali", age: 28)
write(person.describe())
```

Nesne oluşturma (construction), sıradan çağrı sözdizimini kullanır.
Ayrıştırıcının (parser) ayrı bir constructor-çağrısı biçimine ihtiyacı
yoktur.

`Local` olmayan yapı (structure) girdileri özelliklere (attributes)
dönüşür. `Constant`, bir özelliği değiştirilemez yapar ve bir referans
değerini derin dondurur (deep-freezes). `Local`, bir constructor girdisini
bir özellik oluşturmadan yapı gövdesine (structure body) sunar.

```ahd
Account: Class<> := {
    structure: Attributes := (
        id: Constant Int
        password: Local String
    ) {
        attribute.passwordHint: Confidential String := password[0]
    }
}
```

## Kalıtım (Inheritance) ve override'lar

```ahd
Student: Class<Person> := {
    structure: Attributes := (
        SuperClass.attributes
        number: Constant Int
    )

    describe: Override Function := (
    ) -> String {
        base: Local String := SuperClass.describe()
        return "{base}, #{attribute.number}"
    }
}
```

v0.1'de tek bir doğrudan üst sınıf (superclass) vardır. Kalıtsal bir metodu
değiştirmek `Override` gerektirir. `SuperClass.describe()`, doğrudan üst
sınıfın uygulamasını çağırır.

## Confidential üyeler

`Confidential` üyeler, tanımlandıkları sınıf ve alt sınıflarda erişilebilir,
ancak sıradan dış erişim yoluyla erişilemez. Modül kökündeki Confidential
bildirimler, herkese açık modül dışa aktarımları (public module exports)
değildir.

## Operatör davranışı

Bir Class, `==`, sıralama (`<`/`<=`/`>`/`>=`), aritmetik, tekli (unary) `-`
ve `str()` davranışını, on tam sayıda, derleyici tarafından tanınan
(compiler-recognized) metot ismi aracılığıyla tanımlar -- `CEqual`,
`CCompare`, `CAdd`, `CSubtract`, `CMultiply`, `CDivide`, `CRemainder`,
`CPower`, `CNegate`, `CStr`. Gerekli imzalar ve operatör anlambilimi
(semantics) için [Class Protocol Methods](PROTOCOLS_TR.md)'a bakın. Sadece
`C` ile başlayan dahil, başka hiçbir isim özel bir anlam taşımaz.
