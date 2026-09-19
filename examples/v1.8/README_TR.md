# GUI temelleri ve pencere olayları (v1.8)

[English](README.md) · [Türkçe]

v1.8 [GUI](../../docs/GUI_TR.md) modülü ve [Graphics](../../docs/GRAPHICS_TR.md)
Canvas'ının yeni olayları için iki küçük program.

## simple_ledger

```bash
cd simple_ledger
ahdcode run main.ahd
```

Müşteri adı, tutar, "Paid" onay kutusu ve Save düğmesi olan bir form. Save
callback'i girdiyi sıradan AhdCode ile (`trim`, `real`) doğrular, kaydı
mevcut [SQLite](../../docs/SQLITE_TR.md) modülüyle geçerli dizindeki
`ledger.db` dosyasına yazar ve sonucu bir Label'da gösterir. GUI veritabanını
bilmez: callback yalnızca SQLite'ı çağırır. Escape pencereyi kapatır.
`ledger.db` ilk çalıştırmada oluşur ve depoya dahil değildir.

Bu bir muhasebe programı değildir: tablo görünümü, arama veya düzenleme yoktur.

## turtle_events

```bash
cd turtle_events
ahdcode run main.ahd
```

Ok tuşları Turtle'ı döndürür ve her basışta 20 birimlik bir çizgi çizer; bir
tıklama Turtle'ı çizmeden o noktaya taşır (Kartezyen koordinatlar: orijin
merkezde, +y yukarı); S tuşu `turtle-events.png` ve `turtle-events.svg`
dosyalarını kaydeder. Terminalden girdi okumaz: tuşları ve tıklamaları Canvas
penceresi `Canvas.onKey` ve `Canvas.onClick` ile bildirir.
