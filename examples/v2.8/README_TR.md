# Metin, şekiller, işaretçi olayları ve zamanlayıcılar (v2.8)

[English](README.md) · [Türkçe]

[Graphics](../../docs/GRAPHICS_TR.md) Canvas'ına v2.8 ile eklenenler için üç
küçük program.

## shapes_and_text

```bash
cd shapes_and_text
ahdcode run main.ahd
```

Üç hizalamada `Canvas.text`, `Canvas.polygon` ile çizilmiş bir üçgen ve
köşeleri hesaplanan beş köşeli bir yıldız, bir elips ve bir yay içeren tek bir
sayfa. Sayfa `shapes.png` ve `shapes.svg` olarak kaydedilir, sonra
`Canvas.image` ile kendi içine yeniden çizilir ve `shapes-with-image.png`
olarak kaydedilir.

## bouncing_ball

```bash
cd bouncing_ball
ahdcode run main.ahd
```

Bir top dört duvardan seker. `Canvas.every(16, frame)` Canvas'ı saniyede
yaklaşık altmış kez yeniden çizer ve her çağrı tek bir tam kare olarak
gösterilir; bu yüzden temizleyip yeniden çizmek titremez. Boşluk tuşu
`Canvas.cancel` ile duraklatır ve yeni bir zamanlayıcıyla sürdürür, S
`bouncing-ball.png` dosyasını kaydeder, Escape bitirir.

## free_drawing

```bash
cd free_drawing
ahdcode run main.ahd
```

Çizmek için fare düğmesini basılı tutup hareket ettirin. `Canvas.onClick` bir
çizgiyi başlatır, `Canvas.onDrag` bildirilen her konum için bir parça ekler,
`Canvas.onMove` çizgiyi bitirir. C sayfayı temizler, S `drawing.png` dosyasını
kaydeder, Escape bitirir.

Bu programlardaki Function'lar, kullandıkları modül bağlamalarını başlık
satırında ya da ayrı bir satırda `uses [@canvas, ...]` ile adlandırır.
